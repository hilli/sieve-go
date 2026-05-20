package fileinto

import (
	"errors"
	"strings"
	"testing"

	"sieve"
	"sieve/interpreter"
	"sieve/message"
	"sieve/parser"
)

type plainHandler struct{}

func (plainHandler) Keep() error              { return nil }
func (plainHandler) Discard() error           { return nil }
func (plainHandler) Redirect(string) error    { return nil }

type fhandler struct {
	plainHandler
	got string
	err error
}

func (f *fhandler) FileInto(mb string) error { f.got = mb; return f.err }

func runWith(t *testing.T, src string, h sieve.Handler) error {
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

func TestFileIntoCapabilityConstant(t *testing.T) {
	if Capability != "fileinto" {
		t.Fatalf("Capability: got %q", Capability)
	}
}

func TestFileIntoSuccess(t *testing.T) {
	h := &fhandler{}
	if err := runWith(t, `require ["fileinto"]; fileinto "Box";`, h); err != nil {
		t.Fatal(err)
	}
	if h.got != "Box" {
		t.Fatalf("FileInto: got %q", h.got)
	}
}

func TestFileIntoArgErrors(t *testing.T) {
	for _, src := range []string{
		`require ["fileinto"]; fileinto;`,
		`require ["fileinto"]; fileinto "a" "b";`,
		`require ["fileinto"]; fileinto 1;`,
	} {
		err := runWith(t, src, &fhandler{})
		if err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}

func TestFileIntoWrongHandler(t *testing.T) {
	err := runWith(t, `require ["fileinto"]; fileinto "B";`, plainHandler{})
	if err == nil || !strings.Contains(err.Error(), "fileinto.Handler") {
		t.Fatalf("expected handler interface error, got %v", err)
	}
}

func TestFileIntoHandlerError(t *testing.T) {
	h := &fhandler{err: errors.New("disk full")}
	err := runWith(t, `require ["fileinto"]; fileinto "B";`, h)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected handler error to propagate, got %v", err)
	}
}

func TestFileIntoSuppressesImplicitKeep(t *testing.T) {
	// If implicit keep ran, Keep() would be called too. We can't see that
	// directly without a richer recorder; instead verify FileInto was the
	// only delivery action by using a Handler that fails Keep.
	h := &keepFailHandler{}
	if err := runWith(t, `require ["fileinto"]; fileinto "B";`, h); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type keepFailHandler struct {
	fhandler
}

func (k *keepFailHandler) Keep() error { return errors.New("implicit keep ran but should not have") }
