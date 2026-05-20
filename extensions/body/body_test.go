package body

import (
	"testing"

	"github.com/hilli/sieve-go"
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

func TestBodyContentBarePrefix(t *testing.T) {
	const raw = "Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/plain\r\n\r\nhello plain\r\n" +
		"--B--\r\n"
	msg, _ := message.ParseMIME([]byte(raw))
	src := `require ["body"]; if body :content "text" :contains "hello" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &h{}
	if err := s.Run(msg, r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) == 0 || r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

// TestBodyContentEmptyPrefix matches all parts.
func TestBodyContentEmptyPrefix(t *testing.T) {
	const raw = "Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/plain\r\n\r\nfindme\r\n" +
		"--B--\r\n"
	msg, _ := message.ParseMIME([]byte(raw))
	src := `require ["body"]; if body :content "" :contains "findme" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &h{}
	if err := s.Run(msg, r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) == 0 || r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

// TestBodyContentNoMIMEProvider — Builder-built messages return no parts.
func TestBodyContentNoMIMEProvider(t *testing.T) {
	r, err := run(t, `require ["body"]; if body :content "text" :contains "x" { discard; }`, "x")
	if err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "keep" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyContentArgError(t *testing.T) {
	const raw = "Content-Type: text/plain\r\n\r\nhi\r\n"
	msg, _ := message.ParseMIME([]byte(raw))
	src := `require ["body"]; if body :content :contains "x" { keep; }` // :content has no string after it
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Run(msg, &h{}); err == nil {
		t.Fatal("expected error for :content without value")
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
