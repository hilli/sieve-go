package mime_test

import (
	"strings"
	"testing"

	"sieve"
	"sieve/message"

	_ "sieve/extensions/fileinto"
	mimeext "sieve/extensions/mime"
)

// dummyHandler captures the actions taken by a script run.
type dummyHandler struct {
	kept     bool
	folder   string
	redirect string
}

func (h *dummyHandler) Keep() error            { h.kept = true; return nil }
func (h *dummyHandler) Discard() error         { return nil }
func (h *dummyHandler) Redirect(a string) error { h.redirect = a; return nil }
func (h *dummyHandler) FileInto(f string) error { h.folder = f; return nil }

const withAttachment = "From: a@b\r\nSubject: s\r\nMIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
	"--B\r\nContent-Type: text/plain\r\n\r\nhi\r\n" +
	"--B\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"r.pdf\"\r\n\r\nPDF\r\n" +
	"--B--\r\n"

const noAttachment = "From: a@b\r\nSubject: s\r\nMIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
	"--B\r\nContent-Type: text/plain\r\n\r\nhi\r\n" +
	"--B\r\nContent-Type: text/html\r\n\r\n<p>x</p>\r\n" +
	"--B--\r\n"

func runScript(t *testing.T, src, raw string) *dummyHandler {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg, err := message.ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h := &dummyHandler{}
	if err := s.Run(msg, h); err != nil {
		t.Fatalf("run: %v", err)
	}
	return h
}

func TestCapabilityRegistered(t *testing.T) {
	if mimeext.Capability != "mime" {
		t.Errorf("Capability = %q", mimeext.Capability)
	}
}

func TestHeaderMIMEAnyChildAttachment(t *testing.T) {
	src := `require ["mime","fileinto"];
if header :mime :anychild :contains "Content-Disposition" "attachment" {
    fileinto "Attachments";
}`
	h := runScript(t, src, withAttachment)
	if h.folder != "Attachments" {
		t.Errorf("folder = %q", h.folder)
	}
}

func TestHeaderMIMEAnyChildNoAttachment(t *testing.T) {
	src := `require ["mime","fileinto"];
if header :mime :anychild :contains "Content-Disposition" "attachment" {
    fileinto "Attachments";
}`
	h := runScript(t, src, noAttachment)
	if h.folder != "" {
		t.Errorf("should not file, got %q", h.folder)
	}
	if !h.kept {
		t.Errorf("implicit keep expected")
	}
}

func TestExistsMIMEAnyChild(t *testing.T) {
	src := `require ["mime","fileinto"];
if exists :mime :anychild "Content-Disposition" {
    fileinto "Attachments";
}`
	h := runScript(t, src, withAttachment)
	if h.folder != "Attachments" {
		t.Errorf("folder = %q", h.folder)
	}
	h2 := runScript(t, src, noAttachment)
	if h2.folder != "" {
		t.Errorf("no-attachment folder = %q", h2.folder)
	}
}

func TestAddressMIMEAnyChild(t *testing.T) {
	// Address test against a child part header is unusual but supported.
	raw := "Content-Type: multipart/mixed; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: message/rfc822\r\nFrom: nested@inner.example\r\n\r\nbody\r\n" +
		"--B--\r\n"
	src := `require ["mime","fileinto"];
if address :mime :anychild :domain :is "From" "inner.example" {
    fileinto "Nested";
}`
	h := runScript(t, src, raw)
	if h.folder != "Nested" {
		t.Errorf("folder = %q", h.folder)
	}
}

func TestNoMIMETagsDelegatesToCore(t *testing.T) {
	src := `require ["mime","fileinto"];
if header :contains "Subject" "s" {
    fileinto "Hit";
}`
	h := runScript(t, src, withAttachment)
	if h.folder != "Hit" {
		t.Errorf("folder = %q", h.folder)
	}
}

func TestMIMEAnyChildWithoutMIMEErrors(t *testing.T) {
	src := `require ["mime","fileinto"];
if header :anychild :contains "Content-Disposition" "attachment" {
    fileinto "X";
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg, _ := message.ParseMIME([]byte(withAttachment))
	err = s.Run(msg, &dummyHandler{})
	if err == nil || !strings.Contains(err.Error(), ":anychild requires :mime") {
		t.Errorf("expected :anychild requires :mime error, got %v", err)
	}
}

func TestMIMETopLevelEqualsMessageHeaders(t *testing.T) {
	// :mime without :anychild looks at top-level part = the message itself.
	src := `require ["mime","fileinto"];
if header :mime :is "Subject" "s" {
    fileinto "Top";
}`
	h := runScript(t, src, withAttachment)
	if h.folder != "Top" {
		t.Errorf("folder = %q", h.folder)
	}
}

func TestRequireMissing(t *testing.T) {
	src := `if header :mime :anychild :contains "Content-Disposition" "attachment" { discard; }`
	if err := sieve.Validate(src); err == nil {
		t.Errorf("expected validation error without require mime")
	}
}

func TestMIMEHeaderArgErrors(t *testing.T) {
	src := `require ["mime","fileinto"];
if header :mime :anychild "Content-Disposition" {
    fileinto "X";
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg, _ := message.ParseMIME([]byte(withAttachment))
	err = s.Run(msg, &dummyHandler{})
	if err == nil || !strings.Contains(err.Error(), "expected 2 positional") {
		t.Errorf("want arity error, got %v", err)
	}
}

func TestMIMEExistsArgErrors(t *testing.T) {
	src := `require ["mime","fileinto"];
if exists :mime :anychild "A" "B" {
    fileinto "X";
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg, _ := message.ParseMIME([]byte(withAttachment))
	err = s.Run(msg, &dummyHandler{})
	if err == nil || !strings.Contains(err.Error(), "expected 1 argument") {
		t.Errorf("want arity error, got %v", err)
	}
}

func TestMIMEAddressArgErrors(t *testing.T) {
	src := `require ["mime","fileinto"];
if address :mime :anychild "From" {
    fileinto "X";
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg, _ := message.ParseMIME([]byte(withAttachment))
	err = s.Run(msg, &dummyHandler{})
	if err == nil || !strings.Contains(err.Error(), "expected 2 positional") {
		t.Errorf("want arity error, got %v", err)
	}
}

func TestMIMEAnyChildWithoutMIMEProvider(t *testing.T) {
	// Using Builder (no MIMEProvider parts) — :anychild finds nothing.
	src := `require ["mime","fileinto"];
if header :mime :anychild :contains "Content-Disposition" "attachment" {
    fileinto "X";
} else {
    fileinto "Y";
}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	msg := message.NewBuilder().AddHeader("Subject", "s").Build()
	h := &dummyHandler{}
	if err := s.Run(msg, h); err != nil {
		t.Fatalf("run: %v", err)
	}
	if h.folder != "Y" {
		t.Errorf("folder = %q (want Y — :anychild found no parts)", h.folder)
	}
}
