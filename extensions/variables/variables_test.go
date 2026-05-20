package variables_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/fileinto"
	_ "github.com/hilli/sieve-go/extensions/variables"
)

type rec struct{ box string }

func (r *rec) Keep() error              { r.box = "INBOX"; return nil }
func (r *rec) FileInto(mb string) error { r.box = mb; return nil }
func (r *rec) Redirect(string) error    { return nil }
func (r *rec) Discard() error           { r.box = ""; return nil }

func run(t *testing.T, src string) *rec {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, _ := mail.ReadMessage(strings.NewReader("Subject: Hello World\r\nFrom: a@b\r\n\r\nbody\r\n"))
	h := &rec{}
	if err := s.Run(sievemail.FromNetMail(m), h); err != nil {
		t.Fatalf("run: %v", err)
	}
	return h
}

func TestSetAndExpand(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set "box" "Sales";
		fileinto "${box}";
	`)
	if got.box != "Sales" {
		t.Fatalf("got %q", got.box)
	}
}

func TestModifierUpperLower(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set :upper "x" "hello";
		set :lower "y" "WORLD";
		fileinto "${x}-${y}";
	`)
	if got.box != "HELLO-world" {
		t.Fatalf("got %q", got.box)
	}
}

func TestModifierLength(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set :length "n" "abcd";
		fileinto "len-${n}";
	`)
	if got.box != "len-4" {
		t.Fatalf("got %q", got.box)
	}
}

func TestModifierUpperFirst(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set :upperfirst "x" "hello";
		fileinto "${x}";
	`)
	if got.box != "Hello" {
		t.Fatalf("got %q", got.box)
	}
}

func TestModifierQuoteWildcard(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set :quotewildcard "x" "a*b?c";
		fileinto "${x}";
	`)
	if got.box != `a\*b\?c` {
		t.Fatalf("got %q", got.box)
	}
}

func TestStringTestIs(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set "x" "abc";
		if string :is "${x}" "abc" { fileinto "yes"; } else { fileinto "no"; }
	`)
	if got.box != "yes" {
		t.Fatalf("got %q", got.box)
	}
}

func TestStringTestContains(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set "x" "the quick brown fox";
		if string :contains "${x}" "quick" { fileinto "yes"; }
	`)
	if got.box != "yes" {
		t.Fatalf("got %q", got.box)
	}
}

func TestMatchesCaptures(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		if header :matches "Subject" "Hello *" {
			fileinto "got-${1}";
		}
	`)
	if got.box != "got-World" {
		t.Fatalf("got %q", got.box)
	}
}

func TestMatchesCaptureWhole(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		if header :matches "Subject" "*World" {
			fileinto "[${0}]";
		}
	`)
	if got.box != "[Hello World]" {
		t.Fatalf("got %q", got.box)
	}
}

func TestSetInvalidName(t *testing.T) {
	_, err := sieve.Compile(`require ["variables"]; set "1bad" "x";`)
	if err != nil {
		t.Fatalf("compile (should be runtime): %v", err)
	}
	// Runtime error path: invalid name only fails when executed.
	m, _ := mail.ReadMessage(strings.NewReader("Subject: x\r\n\r\n"))
	s, _ := sieve.Compile(`require ["variables"]; set "1bad" "x";`)
	if err := s.Run(sievemail.FromNetMail(m), &rec{}); err == nil {
		t.Fatal("expected runtime error for invalid name")
	}
}

func TestExpansionEscape(t *testing.T) {
	got := run(t, `
		require ["variables", "fileinto"];
		set "x" "Y";
		fileinto "\\${x}";
	`)
	if got.box != "${x}" {
		t.Fatalf("got %q", got.box)
	}
}
