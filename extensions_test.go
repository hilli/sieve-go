package sieve_test

import (
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	_ "github.com/hilli/sieve-go/extensions/body"
	_ "github.com/hilli/sieve-go/extensions/envelope"
	_ "github.com/hilli/sieve-go/extensions/imap4flags"
	_ "github.com/hilli/sieve-go/extensions/regex"
)

func TestEnvelopeExtension(t *testing.T) {
	src := `require ["envelope"];
if envelope :is :domain "from" "example.com" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyExtensionRaw(t *testing.T) {
	src := `require ["body"]; if body :contains "hello" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestBodyContentNotImplemented(t *testing.T) {
	src := `require ["body"]; if body :content "text/plain" "x" { keep; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Run(sampleMsg(), &recorder{}); err == nil {
		t.Fatal("expected an error for body :content (not implemented)")
	}
}

func TestRegexExtension(t *testing.T) {
	src := `require ["regex"];
if header :regex "Subject" "^\\[oncall\\] .*" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) == 0 || r.actions[0] != "discard" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestRegexRequiresCapability(t *testing.T) {
	if err := sieve.Validate(`if header :regex "X" "y" { keep; }`); err == nil {
		t.Fatal("expected validation error: :regex without require")
	}
}

// flagRecorder implements imap4flags.Handler.
type flagRecorder struct {
	recorder
	flags []string
}

func (f *flagRecorder) SetFlags(fl []string) error    { f.flags = append([]string{}, fl...); f.actions = append(f.actions, "setflag:"+strings.Join(fl, ",")); return nil }
func (f *flagRecorder) AddFlags(fl []string) error    { f.flags = append(f.flags, fl...); f.actions = append(f.actions, "addflag:"+strings.Join(fl, ",")); return nil }
func (f *flagRecorder) RemoveFlags(fl []string) error { f.actions = append(f.actions, "removeflag:"+strings.Join(fl, ",")); return nil }
func (f *flagRecorder) CurrentFlags() []string        { return f.flags }

func TestImap4flagsExtension(t *testing.T) {
	src := `require ["imap4flags"];
addflag "\\Seen";
if hasflag "\\Seen" { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &flagRecorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	want := []string{"addflag:\\Seen", "discard"}
	for i, w := range want {
		if i >= len(r.actions) || r.actions[i] != w {
			t.Fatalf("actions: got %v want %v", r.actions, want)
		}
	}
}

func TestImap4flagsSpaceSeparated(t *testing.T) {
	src := `require ["imap4flags"]; setflag "\\Seen \\Flagged"; keep;`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &flagRecorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.flags) != 2 || r.flags[0] != "\\Seen" || r.flags[1] != "\\Flagged" {
		t.Fatalf("flags: %v", r.flags)
	}
}
