// Package imap4flags implements a subset of the Sieve "imap4flags"
// extension (RFC 5232): the actions `setflag`, `addflag`, `removeflag`
// and the test `hasflag`. The `:flags` tag argument to fileinto/keep is
// not yet supported — host applications wanting that should track the
// pending flag set themselves via the Handler callbacks below.
package imap4flags

import (
	"fmt"
	"strings"

	"sieve"
	"sieve/ast"
	"sieve/registry"
)

const Capability = "imap4flags"

// Handler is the host interface. It extends sieve.Handler with three flag
// mutation methods. Each receives the full list (possibly empty) of flag
// strings parsed from the action arguments.
type Handler interface {
	sieve.Handler
	SetFlags(flags []string) error
	AddFlags(flags []string) error
	RemoveFlags(flags []string) error
	// CurrentFlags returns the set of flags currently associated with the
	// message; used by the hasflag test.
	CurrentFlags() []string
}

func Register(i *sieve.Interpreter) {
	r := i.Registry()
	r.RegisterAction("setflag", actionSet, Capability)
	r.RegisterAction("addflag", actionAdd, Capability)
	r.RegisterAction("removeflag", actionRemove, Capability)
	r.RegisterTest("hasflag", testHasflag, Capability)
}

func init() { Register(sieve.Default()) }

func actionSet(ctx registry.Context, args *ast.Arguments) error {
	flags, err := flagsArg(args, "setflag")
	if err != nil {
		return err
	}
	h, ok := ctx.Handler().(Handler)
	if !ok {
		return fmt.Errorf("setflag: handler does not implement imap4flags.Handler")
	}
	return h.SetFlags(flags)
}

func actionAdd(ctx registry.Context, args *ast.Arguments) error {
	flags, err := flagsArg(args, "addflag")
	if err != nil {
		return err
	}
	h, ok := ctx.Handler().(Handler)
	if !ok {
		return fmt.Errorf("addflag: handler does not implement imap4flags.Handler")
	}
	return h.AddFlags(flags)
}

func actionRemove(ctx registry.Context, args *ast.Arguments) error {
	flags, err := flagsArg(args, "removeflag")
	if err != nil {
		return err
	}
	h, ok := ctx.Handler().(Handler)
	if !ok {
		return fmt.Errorf("removeflag: handler does not implement imap4flags.Handler")
	}
	return h.RemoveFlags(flags)
}

func testHasflag(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	// RFC 5232 §3.2: hasflag with one arg uses the message flag list.
	// With two args (variable-list, key-list) it tests the named variable;
	// we don't yet support Sieve variables, so accept only the one-arg form.
	if len(args.Positional) != 1 {
		return false, fmt.Errorf("hasflag: only the one-argument form is supported (variables not implemented)")
	}
	keys, ok := stringsOf(args.Positional[0])
	if !ok {
		return false, fmt.Errorf("hasflag: argument must be a string or string list")
	}
	h, ok := ctx.Handler().(Handler)
	if !ok {
		return false, fmt.Errorf("hasflag: handler does not implement imap4flags.Handler")
	}
	cur := normalize(h.CurrentFlags())
	for _, k := range splitAll(keys) {
		if cur[strings.ToLower(k)] {
			return true, nil
		}
	}
	return false, nil
}

// flagsArg parses the flag-list argument of setflag/addflag/removeflag.
// Per RFC 5232 §3.1, flags may appear as a string list or as a single
// space-separated string.
func flagsArg(args *ast.Arguments, name string) ([]string, error) {
	if len(args.Positional) != 1 {
		return nil, fmt.Errorf("%s: expected 1 argument, got %d", name, len(args.Positional))
	}
	raw, ok := stringsOf(args.Positional[0])
	if !ok {
		return nil, fmt.Errorf("%s: argument must be a string or string list", name)
	}
	return splitAll(raw), nil
}

func splitAll(values []string) []string {
	var out []string
	for _, v := range values {
		out = append(out, strings.Fields(v)...)
	}
	return out
}

func normalize(flags []string) map[string]bool {
	out := make(map[string]bool, len(flags))
	for _, f := range flags {
		out[strings.ToLower(f)] = true
	}
	return out
}

func stringsOf(v ast.Value) ([]string, bool) {
	switch x := v.(type) {
	case ast.StringValue:
		return []string{x.Value}, true
	case ast.StringListValue:
		return x.Values, true
	}
	return nil, false
}
