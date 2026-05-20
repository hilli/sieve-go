// Package body implements the Sieve "body" extension (RFC 5173).
//
// `body [BODY-TRANSFORM] [MATCH-TYPE] <key-list>` tests whether the
// transformed message body matches any of the keys.
//
// Supported body transforms:
//
//   :raw   — match the body bytes as-is (default if no transform given).
//   :text  — same as :raw for plain text messages. A real MIME-aware
//            implementation would decode and concatenate text/* parts;
//            documented as a known limitation.
//   :content "type/subtype" — not yet supported; returns an error so
//            scripts that need it fail loudly rather than silently
//            mismatching.
package body

import (
	"fmt"
	"io"
	"strings"

	"sieve"
	"sieve/ast"
	"sieve/interpreter"
	"sieve/registry"
)

const Capability = "body"

func Register(i *sieve.Interpreter) {
	i.Registry().RegisterTest("body", test, Capability)
}

func init() { Register(sieve.Default()) }

func test(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
	for _, tg := range args.Tags {
		if strings.EqualFold(tg.Name, ":content") {
			return false, fmt.Errorf("body :content is not implemented")
		}
	}
	if len(args.Positional) != 1 {
		return false, fmt.Errorf("body: expected 1 string-list argument, got %d", len(args.Positional))
	}
	keys, ok := stringsOf(args.Positional[0])
	if !ok {
		return false, fmt.Errorf("body: argument must be a string or string list")
	}
	bodyBytes, err := io.ReadAll(ctx.Message().Body())
	if err != nil {
		return false, fmt.Errorf("body: read: %w", err)
	}
	value := string(bodyBytes)
	matcher := interpreter.LookupMatcher(ctx, args)
	for _, k := range keys {
		if matcher(value, k) {
			return true, nil
		}
	}
	return false, nil
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
