package editheader_test

import (
	"net/mail"
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	sievemail "github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/editheader"
)

type plain struct{}

func (plain) Keep() error           { return nil }
func (plain) Discard() error        { return nil }
func (plain) Redirect(string) error { return nil }

type rec struct {
	plain
	added   []string // "name=value@top|bottom"
	deleted []string // "name=pat1,pat2@idx,last"
}

func (r *rec) AddHeader(field, value string, atTop bool) error {
	loc := "bottom"
	if atTop {
		loc = "top"
	}
	r.added = append(r.added, field+"="+value+"@"+loc)
	return nil
}

func (r *rec) DeleteHeader(field string, patterns []string, _ func(string, string) bool, index int, fromLast bool) error {
	tag := ""
	if index != 0 {
		tag += "idx"
	}
	if fromLast {
		tag += "+last"
	}
	r.deleted = append(r.deleted, field+"="+strings.Join(patterns, ",")+"@"+tag)
	return nil
}

func run(t *testing.T, src string, h sieve.Handler) error {
	t.Helper()
	s, err := sieve.Compile(src)
	if err != nil {
		return err
	}
	m, _ := mail.ReadMessage(strings.NewReader("Subject: x\r\n\r\n"))
	return s.Run(sievemail.FromNetMail(m), h)
}

func TestAddHeaderDefaultPrepends(t *testing.T) {
	h := &rec{}
	if err := run(t, `require ["editheader"]; addheader "X-Foo" "bar";`, h); err != nil {
		t.Fatal(err)
	}
	if len(h.added) != 1 || h.added[0] != "X-Foo=bar@top" {
		t.Fatalf("got %v", h.added)
	}
}

func TestAddHeaderLastAppends(t *testing.T) {
	h := &rec{}
	if err := run(t, `require ["editheader"]; addheader :last "X-Foo" "bar";`, h); err != nil {
		t.Fatal(err)
	}
	if h.added[0] != "X-Foo=bar@bottom" {
		t.Fatalf("got %v", h.added)
	}
}

func TestDeleteHeader(t *testing.T) {
	h := &rec{}
	if err := run(t, `require ["editheader"]; deleteheader "X-Spam";`, h); err != nil {
		t.Fatal(err)
	}
	if h.deleted[0] != "X-Spam=@" {
		t.Fatalf("got %v", h.deleted)
	}
}

func TestDeleteHeaderWithPattern(t *testing.T) {
	h := &rec{}
	src := `require ["editheader"]; deleteheader :index 2 :last :contains "X-Spam" ["yes","probable"];`
	if err := run(t, src, h); err != nil {
		t.Fatal(err)
	}
	if h.deleted[0] != "X-Spam=yes,probable@idx+last" {
		t.Fatalf("got %v", h.deleted)
	}
}

func TestEditheaderWrongHandler(t *testing.T) {
	err := run(t, `require ["editheader"]; addheader "X" "y";`, plain{})
	if err == nil || !strings.Contains(err.Error(), "editheader.Handler") {
		t.Fatalf("want handler error, got %v", err)
	}
}
