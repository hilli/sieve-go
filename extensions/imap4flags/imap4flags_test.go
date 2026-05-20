package imap4flags

import (
	"strings"
	"testing"

	"github.com/hilli/sieve-go"
	"github.com/hilli/sieve-go/interpreter"
	"github.com/hilli/sieve-go/message"
	"github.com/hilli/sieve-go/parser"
)

type plain struct{}

func (plain) Keep() error           { return nil }
func (plain) Discard() error        { return nil }
func (plain) Redirect(string) error { return nil }

type fh struct {
	plain
	flags   []string
	actions []string
}

func (f *fh) SetFlags(fl []string) error    { f.flags = append([]string{}, fl...); f.actions = append(f.actions, "set:"+strings.Join(fl, ",")); return nil }
func (f *fh) AddFlags(fl []string) error    { f.flags = append(f.flags, fl...); f.actions = append(f.actions, "add:"+strings.Join(fl, ",")); return nil }
func (f *fh) RemoveFlags(fl []string) error { f.actions = append(f.actions, "rm:"+strings.Join(fl, ",")); return nil }
func (f *fh) CurrentFlags() []string        { return f.flags }

func run(t *testing.T, src string, h sieve.Handler) error {
	t.Helper()
	i := interpreter.New()
	Register(i)
	a, err := parser.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	s, err := i.Compile(a)
	if err != nil {
		return err
	}
	return s.Run(message.NewBuilder().Build(), h)
}

func TestCapability(t *testing.T) {
	if Capability != "imap4flags" {
		t.Fatalf("Capability: %q", Capability)
	}
}

func TestAllActions(t *testing.T) {
	h := &fh{}
	src := `require ["imap4flags"];
addflag "\\Seen";
addflag ["\\Flagged", "\\Draft"];
removeflag "\\Draft";
setflag "\\Replied";`
	if err := run(t, src, h); err != nil {
		t.Fatal(err)
	}
	want := []string{`add:\Seen`, `add:\Flagged,\Draft`, `rm:\Draft`, `set:\Replied`}
	if strings.Join(h.actions, "|") != strings.Join(want, "|") {
		t.Fatalf("actions: got %v want %v", h.actions, want)
	}
}

func TestSpaceSeparatedFlags(t *testing.T) {
	h := &fh{}
	if err := run(t, `require ["imap4flags"]; setflag "\\Seen \\Flagged";`, h); err != nil {
		t.Fatal(err)
	}
	if len(h.flags) != 2 || h.flags[0] != "\\Seen" || h.flags[1] != "\\Flagged" {
		t.Fatalf("flags: %v", h.flags)
	}
}

func TestHasflagTrueAndFalse(t *testing.T) {
	h := &fh{flags: []string{"\\Seen"}}
	src := `require ["imap4flags"]; if hasflag "\\Seen" { discard; }`
	if err := run(t, src, h); err != nil {
		t.Fatal(err)
	}

	// hasflag false → implicit keep.
	h2 := &fh{}
	src2 := `require ["imap4flags"]; if hasflag "\\Seen" { discard; }`
	if err := run(t, src2, h2); err != nil {
		t.Fatal(err)
	}
}

func TestHasflagWrongHandler(t *testing.T) {
	err := run(t, `require ["imap4flags"]; if hasflag "x" { keep; }`, plain{})
	if err == nil || !strings.Contains(err.Error(), "imap4flags.Handler") {
		t.Fatalf("expected handler error, got %v", err)
	}
}

func TestActionArgErrors(t *testing.T) {
	for _, src := range []string{
		`require ["imap4flags"]; setflag;`,
		`require ["imap4flags"]; setflag 1;`,
		`require ["imap4flags"]; setflag "a" "b";`,
		`require ["imap4flags"]; if hasflag { keep; }`,
		`require ["imap4flags"]; if hasflag 1 { keep; }`,
		`require ["imap4flags"]; if hasflag "a" "b" "c" { keep; }`,
	} {
		err := run(t, src, &fh{})
		if err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestSetflagWrongHandler(t *testing.T) {
	err := run(t, `require ["imap4flags"]; setflag "x";`, plain{})
	if err == nil || !strings.Contains(err.Error(), "imap4flags.Handler") {
		t.Fatalf("expected handler error, got %v", err)
	}
}
