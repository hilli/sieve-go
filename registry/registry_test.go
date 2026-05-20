package registry

import (
	"testing"

	"sieve/ast"
)

func TestRegisterAndLookupAction(t *testing.T) {
	r := New()
	r.RegisterAction("noop", func(Context, *ast.Arguments) error { return nil }, "myext")
	if _, _, ok := r.LookupAction("missing"); ok {
		t.Fatal("missing action should not be found")
	}
	got, req, ok := r.LookupAction("noop")
	if !ok || got == nil {
		t.Fatal("expected to find noop action")
	}
	if req != "myext" {
		t.Fatalf("requires: got %q", req)
	}
	if !r.HasCapability("myext") {
		t.Fatal("capability myext should be auto-registered")
	}
	if r.HasCapability("other") {
		t.Fatal("unknown capability should not be present")
	}
}

func TestRegisterTest(t *testing.T) {
	r := New()
	r.RegisterTest("t", func(Context, *ast.Arguments, []*ast.Test) (bool, error) { return true, nil }, "")
	fn, req, ok := r.LookupTest("t")
	if !ok || fn == nil || req != "" {
		t.Fatalf("LookupTest: %v %q %v", fn, req, ok)
	}
}

func TestRegisterMatchType(t *testing.T) {
	r := New()
	r.RegisterMatchType(":xyz", func(a, b string) bool { return a == b }, "xyzcap")
	fn, req, ok := r.LookupMatchType(":xyz")
	if !ok || fn == nil || req != "xyzcap" {
		t.Fatalf("LookupMatchType: %v %q %v", fn, req, ok)
	}
	if !fn("a", "a") || fn("a", "b") {
		t.Fatal("matcher returned wrong value")
	}
	if _, _, ok := r.LookupMatchType(":nope"); ok {
		t.Fatal("unknown match type should not be found")
	}
	if !r.HasCapability("xyzcap") {
		t.Fatal("matcher capability should be auto-registered")
	}
}

func TestEmptyRequiresNotACapability(t *testing.T) {
	r := New()
	r.RegisterAction("a", func(Context, *ast.Arguments) error { return nil }, "")
	r.RegisterTest("t", func(Context, *ast.Arguments, []*ast.Test) (bool, error) { return false, nil }, "")
	r.RegisterMatchType(":m", func(string, string) bool { return false }, "")
	if r.HasCapability("") {
		t.Fatal("empty capability should never be present")
	}
}

func TestErrUnknown(t *testing.T) {
	e := &ErrUnknown{Kind: "action", Name: "foo"}
	if got := e.Error(); got != `unknown action "foo"` {
		t.Fatalf("Error: %q", got)
	}
}
