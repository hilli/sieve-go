package vacation_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	"github.com/hilli/sieve-go/extensions/vacation"
)

type plain struct{}

func (plain) Keep() error           { return nil }
func (plain) Discard() error        { return nil }
func (plain) Redirect(string) error { return nil }

type vh struct {
	plain
	p vacation.Params
}

func (h *vh) Vacation(p vacation.Params) error { h.p = p; return nil }

func run(t *testing.T, src string, h sieve.Handler) error {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		return err
	}
	m, _ := mail.ReadMessage(strings.NewReader("Subject: x\r\n\r\n"))
	return s.Run(sievemail.FromNetMail(m), h)
}

func TestVacationSimple(t *testing.T) {
	h := &vh{}
	if err := run(t, `require ["vacation"]; vacation "On holiday";`, h); err != nil {
		t.Fatal(err)
	}
	if h.p.Reason != "On holiday" {
		t.Fatalf("reason: %q", h.p.Reason)
	}
}

func TestVacationFull(t *testing.T) {
	h := &vh{}
	src := `require ["vacation"];
		vacation :days 7 :subject "Auto" :from "a@b" :handle "h1"
		         :addresses ["x@y", "z@w"] :mime "Hi there";`
	if err := run(t, src, h); err != nil {
		t.Fatal(err)
	}
	if h.p.Days != 7 || h.p.Subject != "Auto" || h.p.From != "a@b" ||
		h.p.Handle != "h1" || !h.p.Mime || h.p.Reason != "Hi there" ||
		len(h.p.Addresses) != 2 {
		t.Fatalf("got %+v", h.p)
	}
}

func TestVacationMissingReason(t *testing.T) {
	if err := run(t, `require ["vacation"]; vacation :days 1;`, &vh{}); err == nil {
		t.Fatal("expected error")
	}
}

func TestVacationWrongHandler(t *testing.T) {
	err := run(t, `require ["vacation"]; vacation "x";`, plain{})
	if err == nil || !strings.Contains(err.Error(), "vacation.Handler") {
		t.Fatalf("want handler error, got %v", err)
	}
}
