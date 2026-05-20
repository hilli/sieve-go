package message

import (
	"io"
	"strings"
	"testing"
)

const mimeFixture = "From: sender@example.com\r\n" +
	"To: rcpt@example.com\r\n" +
	"Subject: hi\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=\"BOUND\"\r\n" +
	"\r\n" +
	"--BOUND\r\n" +
	"Content-Type: text/plain; charset=\"utf-8\"\r\n" +
	"\r\n" +
	"hello\r\n" +
	"--BOUND\r\n" +
	"Content-Type: application/pdf; name=\"report.pdf\"\r\n" +
	"Content-Disposition: attachment; filename=\"report.pdf\"\r\n" +
	"\r\n" +
	"%PDF-1.4 ...\r\n" +
	"--BOUND--\r\n"

func TestParseMIME(t *testing.T) {
	msg, err := ParseMIME([]byte(mimeFixture))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if got := msg.Header("Subject"); len(got) != 1 || got[0] != "hi" {
		t.Errorf("Subject = %v", got)
	}
	mp, ok := msg.(MIMEProvider)
	if !ok {
		t.Fatalf("ParseMIME result does not implement MIMEProvider")
	}
	parts := mp.MIMEParts()
	if len(parts) != 2 {
		t.Fatalf("want 2 parts, got %d", len(parts))
	}
	if parts[0].ContentType() != "text/plain" {
		t.Errorf("part[0].ContentType = %q", parts[0].ContentType())
	}
	if parts[0].IsAttachment() {
		t.Errorf("text part should not be attachment")
	}
	if !parts[1].IsAttachment() {
		t.Errorf("pdf part should be attachment")
	}
	if parts[1].ContentType() != "application/pdf" {
		t.Errorf("part[1].ContentType = %q", parts[1].ContentType())
	}
	body, _ := io.ReadAll(parts[0].Body())
	if !strings.Contains(string(body), "hello") {
		t.Errorf("part[0] body = %q", body)
	}
	if hs := parts[1].Header("Content-Disposition"); len(hs) != 1 {
		t.Errorf("Content-Disposition headers = %v", hs)
	}
	if all := parts[0].AllHeaders(); len(all) == 0 {
		t.Errorf("AllHeaders empty")
	}
}

func TestParseMIMENonMultipart(t *testing.T) {
	raw := "From: a@b\r\nSubject: plain\r\nContent-Type: text/plain\r\n\r\nhi\r\n"
	msg, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	if parts := msg.(MIMEProvider).MIMEParts(); len(parts) != 0 {
		t.Errorf("non-multipart should have 0 parts, got %d", len(parts))
	}
}

func TestParseMIMENested(t *testing.T) {
	raw := "Content-Type: multipart/mixed; boundary=\"A\"\r\n\r\n" +
		"--A\r\nContent-Type: multipart/alternative; boundary=\"B\"\r\n\r\n" +
		"--B\r\nContent-Type: text/plain\r\n\r\ntxt\r\n" +
		"--B\r\nContent-Type: text/html\r\n\r\n<p>h</p>\r\n" +
		"--B--\r\n" +
		"--A\r\nContent-Disposition: attachment\r\nContent-Type: application/zip\r\n\r\nZZ\r\n" +
		"--A--\r\n"
	msg, err := ParseMIME([]byte(raw))
	if err != nil {
		t.Fatalf("ParseMIME: %v", err)
	}
	parts := msg.(MIMEProvider).MIMEParts()
	if len(parts) != 3 {
		t.Fatalf("want 3 leaf parts, got %d", len(parts))
	}
	if !parts[2].IsAttachment() {
		t.Errorf("zip part should be attachment")
	}
}

func TestParseMIMEBadHeaders(t *testing.T) {
	if _, err := ParseMIME([]byte("not a message")); err == nil {
		t.Errorf("expected error for malformed message")
	}
}

func TestNewMIMEPart(t *testing.T) {
	p := NewMIMEPart([]Header{
		{Name: "Content-Type", Value: "application/pdf"},
		{Name: "Content-Disposition", Value: "attachment; filename=x.pdf"},
	}, []byte("data"))
	if p.ContentType() != "application/pdf" {
		t.Errorf("ContentType = %q", p.ContentType())
	}
	if !p.IsAttachment() {
		t.Errorf("IsAttachment should be true")
	}
	if got := p.Header("Content-Type"); len(got) != 1 {
		t.Errorf("Header() = %v", got)
	}
}

func TestBuilderMIMEParts(t *testing.T) {
	p := NewMIMEPart([]Header{{Name: "Content-Type", Value: "text/plain"}}, []byte("x"))
	msg := NewBuilder().AddMIMEPart(p).Build()
	mp, ok := msg.(MIMEProvider)
	if !ok || len(mp.MIMEParts()) != 1 {
		t.Fatalf("builder MIMEParts not exposed: %v", mp)
	}
}

func TestMIMEPartContentTypeFallback(t *testing.T) {
	p := NewMIMEPart([]Header{{Name: "Content-Type", Value: "garbage~~~"}}, nil)
	if p.ContentType() != "garbage~~~" {
		t.Errorf("ContentType fallback = %q", p.ContentType())
	}
}

func TestMIMEPartIsAttachmentBareWord(t *testing.T) {
	p := NewMIMEPart([]Header{{Name: "Content-Disposition", Value: "Attachment"}}, nil)
	if !p.IsAttachment() {
		t.Errorf("bare 'Attachment' should count")
	}
}
