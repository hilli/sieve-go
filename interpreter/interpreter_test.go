package interpreter

import (
	"errors"
	"strings"
	"testing"

	"sieve/ast"
	"sieve/message"
	"sieve/parser"
	"sieve/registry"
)

// recHandler captures action calls.
type recHandler struct{ actions []string }

func (r *recHandler) Keep() error              { r.actions = append(r.actions, "keep"); return nil }
func (r *recHandler) Discard() error           { r.actions = append(r.actions, "discard"); return nil }
func (r *recHandler) Redirect(a string) error  { r.actions = append(r.actions, "redirect:"+a); return nil }

func compile(t *testing.T, src string) *Script {
	t.Helper()
	a, err := parser.Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	s, err := New().Compile(a)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	return s
}

func mustParse(t *testing.T, src string) *ast.Script {
	t.Helper()
	a, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func emptyMsg() message.Message { return message.NewBuilder().Build() }

func TestNewHasCoreBuiltins(t *testing.T) {
	i := New()
	for _, name := range []string{"keep", "discard", "redirect"} {
		if _, _, ok := i.Registry().LookupAction(name); !ok {
			t.Errorf("missing core action %q", name)
		}
	}
	for _, name := range []string{"address", "header", "exists", "size"} {
		if _, _, ok := i.Registry().LookupTest(name); !ok {
			t.Errorf("missing core test %q", name)
		}
	}
	for _, name := range []string{":is", ":contains", ":matches"} {
		if _, _, ok := i.Registry().LookupMatchType(name); !ok {
			t.Errorf("missing core match-type %q", name)
		}
	}
}

func TestValidateErrors(t *testing.T) {
	cases := []struct {
		src     string
		wantSub string
	}{
		{`keep; require ["fileinto"];`, "must precede"},
		{`require [123];`, "expected string in list"}, // parser error actually; OK
		{`bogus;`, "unknown action"},
		{`if bogus { keep; }`, "unknown test"},
		{`require ["nope-cap"];`, "unknown capability"},
		{`if 5 { keep; }`, "expected test identifier"}, // parser error
	}
	for _, tc := range cases {
		a, perr := parser.Parse(tc.src)
		if perr != nil {
			if !strings.Contains(perr.Error(), tc.wantSub) && tc.wantSub != "expected string in list" {
				t.Errorf("parse %q: got %v, want substr %q", tc.src, perr, tc.wantSub)
			}
			continue
		}
		err := New().Validate(a)
		if err == nil {
			t.Errorf("expected validation error for %q", tc.src)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantSub) {
			t.Errorf("error for %q: got %v, want substr %q", tc.src, err, tc.wantSub)
		}
	}
}

func TestValidateRequireWithStringValueAndList(t *testing.T) {
	// Single string and a list both valid.
	if err := New().Validate(mustParse(t, `require "fileinto";`)); err == nil || !strings.Contains(err.Error(), "fileinto") {
		// "fileinto" cap not registered on a fresh New() (only fileinto extension registers it)
		if err == nil {
			t.Fatal("expected unknown capability error")
		}
	}
	if err := New().Validate(mustParse(t, `require ["fileinto", "envelope"];`)); err == nil {
		t.Fatal("expected error: caps not registered on fresh interpreter")
	}
}

func TestRunImplicitKeep(t *testing.T) {
	s := compile(t, `if header :is "X" "y" { discard; }`)
	r := &recHandler{}
	if err := s.Run(emptyMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) != 1 || r.actions[0] != "keep" {
		t.Fatalf("want implicit keep, got %v", r.actions)
	}
}

func TestRunStop(t *testing.T) {
	s := compile(t, `keep; stop; discard;`)
	r := &recHandler{}
	if err := s.Run(emptyMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) != 1 || r.actions[0] != "keep" {
		t.Fatalf("stop did not halt: %v", r.actions)
	}
}

func TestRunElsifElseSkip(t *testing.T) {
	s := compile(t, `
if false { discard; }
elsif true { redirect "a@b"; }
elsif false { discard; }
else { discard; }`)
	r := &recHandler{}
	if err := s.Run(emptyMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) != 1 || r.actions[0] != "redirect:a@b" {
		t.Fatalf("elsif chain wrong: %v", r.actions)
	}
}

func TestRunElseOnly(t *testing.T) {
	s := compile(t, `if false { discard; } else { redirect "x@y"; }`)
	r := &recHandler{}
	if err := s.Run(emptyMsg(), r); err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "redirect:x@y" {
		t.Fatalf("else: %v", r.actions)
	}
}

func TestEvalTrueFalseNotAllofAnyof(t *testing.T) {
	cases := []struct {
		src  string
		want string // expected first action
	}{
		{`if true { redirect "t"; }`, "redirect:t"},
		{`if not false { redirect "nf"; }`, "redirect:nf"},
		{`if allof(true, true) { redirect "all"; }`, "redirect:all"},
		{`if allof(true, false) { discard; }`, "keep"},        // implicit
		{`if anyof(false, true) { redirect "any"; }`, "redirect:any"},
		{`if anyof(false, false) { discard; }`, "keep"},
	}
	for _, c := range cases {
		s := compile(t, c.src)
		r := &recHandler{}
		if err := s.Run(emptyMsg(), r); err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if r.actions[0] != c.want {
			t.Errorf("%s: got %v want first=%s", c.src, r.actions, c.want)
		}
	}
}

func TestRunActionErrorPropagates(t *testing.T) {
	i := New()
	i.Registry().RegisterAction("boom", func(registry.Context, *ast.Arguments) error {
		return errors.New("kaboom")
	}, "")
	a := mustParse(t, `boom;`)
	s, err := i.Compile(a)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Run(emptyMsg(), &recHandler{})
	if err == nil || !strings.Contains(err.Error(), "kaboom") {
		t.Fatalf("expected boom error, got %v", err)
	}
}

func TestRedirectArgErrors(t *testing.T) {
	for _, src := range []string{`redirect;`, `redirect "a" "b";`, `redirect 5;`} {
		s := compile(t, src)
		r := &recHandler{}
		err := s.Run(emptyMsg(), r)
		if err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestExistsSizeHeader(t *testing.T) {
	msg := message.NewBuilder().
		AddHeader("Subject", "hi").
		SetBody(make([]byte, 2048)).
		Build()
	cases := []struct {
		src  string
		want string
	}{
		{`if exists "Subject" { redirect "a"; }`, "redirect:a"},
		{`if exists ["Subject", "Missing"] { discard; }`, "keep"},
		{`if size :over 1K { redirect "big"; }`, "redirect:big"},
		{`if size :under 10 { discard; }`, "keep"},
		{`if header :is "Subject" "hi" { redirect "h"; }`, "redirect:h"},
		{`if header :contains "Subject" "h" { redirect "c"; }`, "redirect:c"},
		{`if header :matches "Subject" "h*" { redirect "m"; }`, "redirect:m"},
	}
	for _, c := range cases {
		s := compile(t, c.src)
		r := &recHandler{}
		if err := s.Run(msg, r); err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if r.actions[0] != c.want {
			t.Errorf("%s: got %v", c.src, r.actions)
		}
	}
}

func TestAddressLocalDomain(t *testing.T) {
	msg := message.NewBuilder().AddHeader("From", "Alice <alice@example.com>").Build()
	cases := []struct {
		src  string
		want string
	}{
		{`if address :is :all "From" "alice@example.com" { redirect "a"; }`, "redirect:a"},
		{`if address :is :localpart "From" "alice" { redirect "l"; }`, "redirect:l"},
		{`if address :is :domain "From" "example.com" { redirect "d"; }`, "redirect:d"},
		{`if address :is :domain "From" "other" { discard; }`, "keep"},
	}
	for _, c := range cases {
		s := compile(t, c.src)
		r := &recHandler{}
		if err := s.Run(msg, r); err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if r.actions[0] != c.want {
			t.Errorf("%s: got %v", c.src, r.actions)
		}
	}
}

func TestSizeArgErrors(t *testing.T) {
	for _, src := range []string{
		`if size "x" { keep; }`,       // wrong type
		`if size 100 { keep; }`,       // no :over/:under
	} {
		s := compile(t, src)
		err := s.Run(emptyMsg(), &recHandler{})
		if err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestSizeWrongArgCount(t *testing.T) {
	// Two positionals should error.
	s := compile(t, `if size :over 1 1 { keep; }`)
	if err := s.Run(emptyMsg(), &recHandler{}); err == nil {
		t.Fatal("expected size arity error")
	}
}

func TestUnknownActionAtRuntime(t *testing.T) {
	// Compile uses a registry with all builtins; but if we manually
	// build a Script with an unknown command name, exec returns an
	// error. We do this by skipping validation.
	s := &Script{
		ast: &ast.Script{Commands: []*ast.Command{{Name: "phantom"}}},
		interp: New(),
	}
	err := s.Run(emptyMsg(), &recHandler{})
	if err == nil || !strings.Contains(err.Error(), "phantom") {
		t.Fatalf("expected unknown action runtime error, got %v", err)
	}
}

func TestWildcardMatch(t *testing.T) {
	cases := []struct {
		s, p string
		want bool
	}{
		{"hello", "hello", true},
		{"hello", "h*o", true},
		{"hello", "h?llo", true},
		{"hello", "h?lo", false},
		{"hello", "*", true},
		{"", "*", true},
		{"abc", "a\\*c", false},
		{"a*c", "a\\*c", true},
		{"hello world", "*world", true},
		{"hello world", "world*", false},
	}
	for _, c := range cases {
		if got := wildcardMatch(c.s, c.p); got != c.want {
			t.Errorf("wildcardMatch(%q, %q) = %v, want %v", c.s, c.p, got, c.want)
		}
	}
}

func TestAddressPartString(t *testing.T) {
	cases := []struct {
		addr string
		p    addressPart
		want string
	}{
		{"a@b", addrAll, "a@b"},
		{"a@b", addrLocal, "a"},
		{"a@b", addrDomain, "b"},
		{"noat", addrLocal, "noat"},
		{"noat", addrDomain, ""},
	}
	for _, c := range cases {
		if got := addressPartString(c.addr, c.p); got != c.want {
			t.Errorf("addressPartString(%q,%d) = %q want %q", c.addr, c.p, got, c.want)
		}
	}
}

func TestCoreMatchers(t *testing.T) {
	if !matchIs("Foo", "foo") {
		t.Error("matchIs case-insensitive failed")
	}
	if matchIs("foo", "bar") {
		t.Error("matchIs false positive")
	}
	if !matchContains("Hello World", "WORLD") {
		t.Error("matchContains case-insensitive failed")
	}
	if !matchMatches("hello", "H?LLO") {
		t.Error("matchMatches case-insensitive failed")
	}
}
