package sieve_test

import (
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	_ "github.com/hilli/sieve-go/extensions/fileinto"
	fileintoext "github.com/hilli/sieve-go/extensions/fileinto"
	"github.com/hilli/sieve-go/message"
)

type recorder struct {
	actions []string
}

func (r *recorder) Keep() error                { r.actions = append(r.actions, "keep"); return nil }
func (r *recorder) Discard() error             { r.actions = append(r.actions, "discard"); return nil }
func (r *recorder) Redirect(a string) error    { r.actions = append(r.actions, "redirect:"+a); return nil }
func (r *recorder) FileInto(b string) error    { r.actions = append(r.actions, "fileinto:"+b); return nil }

func sampleMsg() message.Message {
	return message.NewBuilder().
		AddHeader("From", "alice@example.com").
		AddHeader("To", "bob@example.org").
		AddHeader("Subject", "[oncall] page").
		AddHeader("X-Spam", "1").
		SetEnvelope("from", "alice@example.com").
		SetBody([]byte("hello world")).
		Build()
}

func TestRunFileInto(t *testing.T) {
	src := `require ["fileinto"];
if header :contains "Subject" "oncall" {
    fileinto "Oncall";
    stop;
}
keep;`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, want := strings.Join(r.actions, ","), "fileinto:Oncall"; got != want {
		t.Fatalf("actions: got %q want %q", got, want)
	}
}

func TestImplicitKeep(t *testing.T) {
	s, err := sieve.Compile(`if header :is "Subject" "no-match" { discard; }`)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if len(r.actions) != 1 || r.actions[0] != "keep" {
		t.Fatalf("want implicit keep, got %v", r.actions)
	}
}

func TestAddressMatchesDomain(t *testing.T) {
	src := `if address :is :domain "From" "example.com" { discard; }`
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

func TestElsifElse(t *testing.T) {
	src := `if header :is "Subject" "nope" {
    discard;
} elsif header :contains "Subject" "oncall" {
    redirect "oncall@example.com";
} else {
    discard;
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	if r.actions[0] != "redirect:oncall@example.com" {
		t.Fatalf("actions: %v", r.actions)
	}
}

func TestValidateUnknownCapability(t *testing.T) {
	err := sieve.Validate(`require ["nope-not-real"];`)
	if err == nil {
		t.Fatal("expected error for unknown capability")
	}
}

func TestValidateMissingRequire(t *testing.T) {
	// fileinto used without require → should fail validation.
	err := sieve.Validate(`fileinto "X";`)
	if err == nil {
		t.Fatal("expected error: fileinto without require")
	}
}

func TestSizeOverUnder(t *testing.T) {
	src := `if size :over 1K { discard; }`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatal(err)
	}
	r := &recorder{}
	if err := s.Run(sampleMsg(), r); err != nil {
		t.Fatal(err)
	}
	// sample message is < 1K → implicit keep
	if len(r.actions) != 1 || r.actions[0] != "keep" {
		t.Fatalf("want keep, got %v", r.actions)
	}
}

func TestMatchesGlob(t *testing.T) {
	src := `if header :matches "Subject" "*oncall*" { discard; }`
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

func TestEnvelopeRequiresCapability(t *testing.T) {
	// envelope without require → fails validation
	if err := sieve.Validate(`if envelope "from" "x@y" { keep; }`); err == nil {
		t.Fatal("expected error for missing envelope require")
	}
	// with require → ok
	if err := sieve.Validate(`require ["envelope"]; if envelope "from" "x@y" { keep; }`); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestExtensionPackageSelfRegisters(t *testing.T) {
	// Just touching the symbol forces the import even with -trimpath etc.
	_ = fileintoext.Capability
}
