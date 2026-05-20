package reject_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/reject"
)

type plain struct{}

func (plain) Keep() error           { return nil }
func (plain) Discard() error        { return nil }
func (plain) Redirect(string) error { return nil }

type rejecter struct {
	plain
	rejected string
	ereject  string
}

func (r *rejecter) Reject(reason string) error  { r.rejected = reason; return nil }
func (r *rejecter) Ereject(reason string) error { r.ereject = reason; return nil }

type rejectOnly struct {
	plain
	rejected string
}

func (r *rejectOnly) Reject(reason string) error { r.rejected = reason; return nil }

func run(t *testing.T, src string, h sieve.Handler) error {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		return err
	}
	m, _ := mail.ReadMessage(strings.NewReader("Subject: x\r\n\r\n"))
	return s.Run(sievemail.FromNetMail(m), h)
}

func TestReject(t *testing.T) {
	h := &rejecter{}
	if err := run(t, `require ["reject"]; reject "nope";`, h); err != nil {
		t.Fatal(err)
	}
	if h.rejected != "nope" {
		t.Fatalf("got %q", h.rejected)
	}
}

func TestEreject(t *testing.T) {
	h := &rejecter{}
	if err := run(t, `require ["ereject"]; ereject "go away";`, h); err != nil {
		t.Fatal(err)
	}
	if h.ereject != "go away" {
		t.Fatalf("got %q", h.ereject)
	}
}

func TestErejectFallsBackToReject(t *testing.T) {
	h := &rejectOnly{}
	if err := run(t, `require ["ereject"]; ereject "fallback";`, h); err != nil {
		t.Fatal(err)
	}
	if h.rejected != "fallback" {
		t.Fatalf("got %q", h.rejected)
	}
}

func TestRejectWrongHandler(t *testing.T) {
	err := run(t, `require ["reject"]; reject "x";`, plain{})
	if err == nil || !strings.Contains(err.Error(), "reject.Handler") {
		t.Fatalf("want handler error, got %v", err)
	}
}

func TestRejectArgError(t *testing.T) {
	err := run(t, `require ["reject"]; reject;`, &rejecter{})
	if err == nil {
		t.Fatal("expected error")
	}
}
