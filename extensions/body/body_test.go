package body

import (
	"strings"
	"testing"

	"github.com/hilli/sieve-go/interpreter"
	"github.com/hilli/sieve-go/message"
	"github.com/hilli/sieve-go/parser"
)

type h struct{ actions []string }

func (x *h) Keep() error              { x.actions = append(x.actions, "keep"); return nil }
func (x *h) Discard() error           { x.actions = append(x.actions, "discard"); return nil }
func (x *h) Redirect(a string) error  { x.actions = append(x.actions, "redirect:"+a); return nil }

func run(t *testing.T, src string, body string) (*h, error) {
	t.Helper()
	i := interpreter.New()
	Register(i)
	a, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	s, err := i.Compile(a)
	if err != nil {
		return nil, err
	}
	msg := message.NewBuilder().SetBody([]byte(body)).Build()
	rec := &h{}
	return rec, s.Run(msg, rec)
}

func TestCapability(t *testing.T) {
	if Capability != "body" {
		t.Fatalf("Capability: %q", Capability)
	}
}

func TestBodyContains(t *testing.T) {
	r, err := run(t, `require ["body"]; if body :contains "hello" { discard; }`, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyMatches(t *testing.T) {
	r, err := run(t, `require ["body"]; if body :matches "*world*" { discard; }`, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyStringList(t *testing.T) {
	r, err := run(t, `require ["body"]; if body :is ["alpha", "bravo"] { discard; }`, "bravo")
	if err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyNoMatch(t *testing.T) {
	r, err := run(t, `require ["body"]; if body :contains "xxx" { discard; }`, "hello")
	if err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "keep" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyContentErrors(t *testing.T) {
	_, err := run(t, `require ["body"]; if body :content "text/plain" "x" { keep; }`, "")
	if err == nil || !strings.Contains(err.Error(), ":content") {
		t.Fatalf("expected :content error, got %v", err)
	}
}

func TestBodyArgErrors(t *testing.T) {
	for _, src := range []string{
		`require ["body"]; if body { keep; }`,
		`require ["body"]; if body 1 { keep; }`,
	} {
		_, err := run(t, src, "")
		if err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}
