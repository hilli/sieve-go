package regex

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
	if Capability != "regex" {
		t.Fatalf("Capability: %q", Capability)
	}
}

func TestRegexMatchAndNoMatch(t *testing.T) {
	cases := []struct {
		pat, val, want string
	}{
		{`^foo`, "foobar", "discard"},
		{`bar$`, "foobar", "discard"},
		{`^x+$`, "yyy", "keep"},
	}
	for _, c := range cases {
		i := interpreter.New()
		Register(i)
		src := `require ["regex"]; if header :regex "Subject" "` + c.pat + `" { discard; }`
		a, err := parser.Parse(src)
		if err != nil {
			t.Fatalf("parse %s: %v", src, err)
		}
		s, err := i.Compile(a)
		if err != nil {
			t.Fatalf("compile %s: %v", src, err)
		}
		msg := message.NewBuilder().AddHeader("Subject", c.val).Build()
		r := &h{}
		if err := s.Run(msg, r); err != nil {
			t.Fatal(err)
		}
		if r.actions[0] != c.want {
			t.Errorf("pat=%q val=%q: got %v want %s", c.pat, c.val, r.actions, c.want)
		}
	}
}

func TestInvalidRegexNeverMatches(t *testing.T) {
	if matchRegex("anything", "(") {
		t.Fatal("invalid regex should not match")
	}
	// Cache should hold the nil result.
	if matchRegex("anything", "(") {
		t.Fatal("invalid regex should not match (cached)")
	}
}

func TestCompileCache(t *testing.T) {
	// Same pattern returns the same compiled regex object.
	r1 := compile("abc")
	r2 := compile("abc")
	if r1 != r2 {
		t.Fatal("expected cached regex to be reused")
	}
}
