package subaddress_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/fileinto"
	_ "github.com/hilli/sieve-go/extensions/subaddress"
)

type rec struct{ box string }

func (r *rec) Keep() error              { r.box = "INBOX"; return nil }
func (r *rec) Discard() error           { r.box = "DISCARD"; return nil }
func (r *rec) Redirect(string) error    { return nil }
func (r *rec) FileInto(mb string) error { r.box = mb; return nil }

func run(t *testing.T, src string) *rec {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, _ := mail.ReadMessage(strings.NewReader(
		"To: alice+work@example.com\r\nSubject: x\r\n\r\n"))
	h := &rec{}
	if err := s.Run(sievemail.FromNetMail(m), h); err != nil {
		t.Fatalf("run: %v", err)
	}
	return h
}

func TestUser(t *testing.T) {
	got := run(t, `require ["subaddress","fileinto"];
		if address :user "To" "alice" { fileinto "yes"; }`)
	if got.box != "yes" {
		t.Fatalf("got %q", got.box)
	}
}

func TestDetail(t *testing.T) {
	got := run(t, `require ["subaddress","fileinto"];
		if address :detail "To" "work" { fileinto "yes"; }`)
	if got.box != "yes" {
		t.Fatalf("got %q", got.box)
	}
}

func TestDetailEmpty(t *testing.T) {
	got := run(t, `require ["subaddress","fileinto"];
		if address :detail :is "To" "" { fileinto "no"; }`)
	if got.box == "no" {
		t.Fatalf("expected detail to be non-empty for alice+work")
	}
}
