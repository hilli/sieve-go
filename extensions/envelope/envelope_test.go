package envelope

import (
	"testing"

	"sieve/interpreter"
	"sieve/message"
	"sieve/parser"
)

type h struct{ actions []string }

func (x *h) Keep() error              { x.actions = append(x.actions, "keep"); return nil }
func (x *h) Discard() error           { x.actions = append(x.actions, "discard"); return nil }
func (x *h) Redirect(a string) error  { x.actions = append(x.actions, "redirect:"+a); return nil }

func TestCapability(t *testing.T) {
	if Capability != "envelope" {
		t.Fatalf("Capability: %q", Capability)
	}
}

func TestEnvelopeMatch(t *testing.T) {
	i := interpreter.New()
	Register(i)
	a, _ := parser.Parse(`require ["envelope"]; if envelope :is :domain "from" "example.com" { discard; }`)
	s, err := i.Compile(a)
	if err != nil {
		t.Fatal(err)
	}
	msg := message.NewBuilder().SetEnvelope("from", "alice@example.com").Build()
	rec := &h{}
	if err := s.Run(msg, rec); err != nil {
		t.Fatal(err)
	}
	if rec.actions[0] != "discard" {
		t.Fatalf("actions: %v", rec.actions)
	}
}

func TestEnvelopeNoMatch(t *testing.T) {
	i := interpreter.New()
	Register(i)
	a, _ := parser.Parse(`require ["envelope"]; if envelope "from" "x@y" { discard; }`)
	s, _ := i.Compile(a)
	rec := &h{}
	if err := s.Run(message.NewBuilder().Build(), rec); err != nil {
		t.Fatal(err)
	}
	if rec.actions[0] != "keep" { // implicit
		t.Fatalf("actions: %v", rec.actions)
	}
}
