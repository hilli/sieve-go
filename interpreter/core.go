package interpreter

import (
	"fmt"
	"net/mail"
	"strings"

	"sieve/ast"
	"sieve/registry"
)

// RegisterCore registers the RFC 5228 built-in actions and tests onto
// the given registry. None of these require a capability — they are
// always available.
func RegisterCore(r *registry.Registry) {
	// Actions.
	r.RegisterAction("keep", actionKeep, "")
	r.RegisterAction("discard", actionDiscard, "")
	r.RegisterAction("redirect", actionRedirect, "")

	// Tests.
	r.RegisterTest("address", testAddress, "")
	r.RegisterTest("header", testHeader, "")
	r.RegisterTest("exists", testExists, "")
	r.RegisterTest("size", testSize, "")
	// envelope is technically a separate extension (RFC 5228 §5.4 + an
	// envelope capability), but we provide it under the "envelope" cap.
	r.RegisterTest("envelope", testEnvelope, "envelope")
}

// ---------- actions ----------

func actionKeep(ctx registry.Context, args *ast.Arguments) error {
	ctx.MarkExplicitAction()
	return ctx.Handler().Keep()
}

func actionDiscard(ctx registry.Context, args *ast.Arguments) error {
	ctx.MarkExplicitAction()
	return ctx.Handler().Discard()
}

func actionRedirect(ctx registry.Context, args *ast.Arguments) error {
	if len(args.Positional) != 1 {
		return fmt.Errorf("redirect: expected 1 string argument")
	}
	addr, ok := args.Positional[0].(ast.StringValue)
	if !ok {
		return fmt.Errorf("redirect: expected string argument")
	}
	ctx.MarkExplicitAction()
	return ctx.Handler().Redirect(addr.Value)
}

// ---------- tests ----------

// testHeader implements `header [MATCH-TYPE] <header-names> <key-list>`.
func testHeader(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	names, keys, err := twoStringLists(args, "header")
	if err != nil {
		return false, err
	}
	mk := matchKindOf(args)
	msg := ctx.Message()
	for _, hn := range names {
		for _, v := range msg.Header(hn) {
			for _, k := range keys {
				if matchString(v, k, mk) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// testAddress implements `address [ADDRESS-PART] [MATCH-TYPE] <header-list> <key-list>`.
func testAddress(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	names, keys, err := twoStringLists(args, "address")
	if err != nil {
		return false, err
	}
	mk := matchKindOf(args)
	ap := addressPartOf(args)
	msg := ctx.Message()
	for _, hn := range names {
		for _, raw := range msg.Header(hn) {
			addrs, err := mail.ParseAddressList(raw)
			if err != nil {
				continue
			}
			for _, a := range addrs {
				part := addressPartString(a.Address, ap)
				for _, k := range keys {
					if matchString(part, k, mk) {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

// testEnvelope mirrors testAddress but reads from msg.Envelope.
func testEnvelope(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	names, keys, err := twoStringLists(args, "envelope")
	if err != nil {
		return false, err
	}
	mk := matchKindOf(args)
	ap := addressPartOf(args)
	msg := ctx.Message()
	for _, hn := range names {
		for _, raw := range msg.Envelope(hn) {
			part := addressPartString(raw, ap)
			for _, k := range keys {
				if matchString(part, k, mk) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// testExists is true iff every named header is present at least once.
func testExists(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	if len(args.Positional) != 1 {
		return false, fmt.Errorf("exists: expected 1 argument")
	}
	names, ok := stringsOf(args.Positional[0])
	if !ok {
		return false, fmt.Errorf("exists: expected string or string list")
	}
	msg := ctx.Message()
	for _, n := range names {
		if len(msg.Header(n)) == 0 {
			return false, nil
		}
	}
	return true, nil
}

// testSize implements `size :over N` / `size :under N`.
func testSize(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	if len(args.Positional) != 1 {
		return false, fmt.Errorf("size: expected 1 number argument")
	}
	n, ok := args.Positional[0].(ast.NumberValue)
	if !ok {
		return false, fmt.Errorf("size: expected number")
	}
	sz := uint64(ctx.Message().Size())
	for _, tg := range args.Tags {
		switch strings.ToLower(tg.Name) {
		case ":over":
			return sz > n.Value, nil
		case ":under":
			return sz < n.Value, nil
		}
	}
	return false, fmt.Errorf("size: requires :over or :under")
}

// ---------- helpers ----------

func twoStringLists(args *ast.Arguments, name string) ([]string, []string, error) {
	if len(args.Positional) != 2 {
		return nil, nil, fmt.Errorf("%s: expected 2 positional arguments, got %d", name, len(args.Positional))
	}
	a, ok := stringsOf(args.Positional[0])
	if !ok {
		return nil, nil, fmt.Errorf("%s: first argument must be string or string list", name)
	}
	b, ok := stringsOf(args.Positional[1])
	if !ok {
		return nil, nil, fmt.Errorf("%s: second argument must be string or string list", name)
	}
	return a, b, nil
}

func addressPartString(addr string, p addressPart) string {
	at := strings.LastIndexByte(addr, '@')
	switch p {
	case addrLocal:
		if at < 0 {
			return addr
		}
		return addr[:at]
	case addrDomain:
		if at < 0 {
			return ""
		}
		return addr[at+1:]
	default:
		return addr
	}
}

// matchString implements the three match types using ASCII case-insensitive
// comparison (the default :comparator "i;ascii-casemap" per RFC 5228 §2.7.3).
// :matches uses ? and * wildcards (no character classes).
func matchString(s, key string, mk matchKind) bool {
	switch mk {
	case matchIs:
		return strings.EqualFold(s, key)
	case matchContains:
		return strings.Contains(strings.ToLower(s), strings.ToLower(key))
	case matchMatches:
		return wildcardMatch(strings.ToLower(s), strings.ToLower(key))
	}
	return false
}

// wildcardMatch implements the RFC 5228 :matches glob: '?' matches any
// single character, '*' matches zero or more characters; '\' escapes.
func wildcardMatch(s, pat string) bool {
	// Iterative dynamic algorithm with backtracking.
	si, pi := 0, 0
	star, ss := -1, 0
	for si < len(s) {
		switch {
		case pi < len(pat) && pat[pi] == '\\' && pi+1 < len(pat):
			if s[si] == pat[pi+1] {
				si++
				pi += 2
				continue
			}
		case pi < len(pat) && (pat[pi] == '?' || pat[pi] == s[si]):
			si++
			pi++
			continue
		case pi < len(pat) && pat[pi] == '*':
			star = pi
			ss = si
			pi++
			continue
		}
		if star != -1 {
			pi = star + 1
			ss++
			si = ss
			continue
		}
		return false
	}
	for pi < len(pat) && pat[pi] == '*' {
		pi++
	}
	return pi == len(pat)
}
