package relational_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/fileinto"
	_ "github.com/hilli/sieve-go/extensions/relational"
)

type rec struct{ box string }

func (r *rec) Keep() error              { r.box = "INBOX"; return nil }
func (r *rec) Discard() error           { r.box = "DISCARD"; return nil }
func (r *rec) Redirect(string) error    { return nil }
func (r *rec) FileInto(mb string) error { r.box = mb; return nil }

func run(t *testing.T, src, raw string) *rec {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, _ := mail.ReadMessage(strings.NewReader(raw))
	h := &rec{}
	if err := s.Run(sievemail.FromNetMail(m), h); err != nil {
		t.Fatalf("run: %v", err)
	}
	return h
}

func TestCountGT(t *testing.T) {
	got := run(t,
		`require ["relational","fileinto"];
		 if address :count "gt" "To" "1" { fileinto "many"; }`,
		"To: a@x, b@x, c@x\r\nSubject: s\r\n\r\n")
	if got.box != "many" {
		t.Fatalf("got %q", got.box)
	}
}

func TestCountEQZero(t *testing.T) {
	got := run(t,
		`require ["relational","fileinto"];
		 if header :count "eq" "X-Spam" "0" { fileinto "clean"; }`,
		"Subject: s\r\n\r\n")
	if got.box != "clean" {
		t.Fatalf("got %q", got.box)
	}
}

func TestValueGT(t *testing.T) {
	got := run(t,
		`require ["relational","fileinto"];
		 if header :value "gt" "X-Priority" "3" { fileinto "low"; }`,
		"X-Priority: 5\r\nSubject: s\r\n\r\n")
	if got.box != "low" {
		t.Fatalf("got %q", got.box)
	}
}

func TestValueLE(t *testing.T) {
	got := run(t,
		`require ["relational","fileinto"];
		 if header :value "le" "X-Priority" "3" { fileinto "high"; }`,
		"X-Priority: 1\r\nSubject: s\r\n\r\n")
	if got.box != "high" {
		t.Fatalf("got %q", got.box)
	}
}
