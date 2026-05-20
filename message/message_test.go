package message

import (
	"io"
	"net/mail"
	"strings"
	"testing"
)

func TestBuilder(t *testing.T) {
	m := NewBuilder().
		AddHeader("From", "alice@example.com").
		AddHeader("from", "carol@example.com").
		AddHeader("To", "bob@example.com").
		SetEnvelope("From", "alice@example.com").
		SetBody([]byte("hi")).
		Build()

	if got := m.Header("from"); len(got) != 2 || got[0] != "alice@example.com" || got[1] != "carol@example.com" {
		t.Fatalf("Header lookup not case-insensitive: %v", got)
	}
	if got := m.Header("missing"); got != nil {
		t.Fatalf("missing header: got %v", got)
	}
	if got := m.AllHeaders(); len(got) != 3 {
		t.Fatalf("AllHeaders: got %d", len(got))
	}
	if env := m.Envelope("from"); len(env) != 1 || env[0] != "alice@example.com" {
		t.Fatalf("Envelope case-insensitive: %v", env)
	}
	if env := m.Envelope("unknown"); env != nil {
		t.Fatalf("unknown env: %v", env)
	}

	// Body should be re-readable.
	b1, _ := io.ReadAll(m.Body())
	b2, _ := io.ReadAll(m.Body())
	if string(b1) != "hi" || string(b2) != "hi" {
		t.Fatalf("Body not re-readable: %q %q", b1, b2)
	}
	if m.Size() <= 0 {
		t.Fatalf("Size: got %d", m.Size())
	}
}

func TestEnvelopeOnNetMailIsNil(t *testing.T) {
	raw := "From: a@x\r\nTo: b@y\r\n\r\nbody"
	nm, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	m := FromNetMail(nm)
	if env := m.Envelope("from"); env != nil {
		t.Fatalf("expected nil envelope, got %v", env)
	}
	if got := m.Header("From"); len(got) != 1 || got[0] != "a@x" {
		t.Fatalf("From header: %v", got)
	}
	body, _ := io.ReadAll(m.Body())
	if string(body) != "body" {
		t.Fatalf("body: %q", body)
	}
}
