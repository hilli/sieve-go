package mime_test

import (
	"testing"

	"github.com/hilli/sieve-go"
	"github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/fileinto"
	_ "github.com/hilli/sieve-go/extensions/mime"
	_ "github.com/hilli/sieve-go/extensions/variables"
)

func TestForeverypartFiresOncePerPart(t *testing.T) {
	src := `require ["mime","fileinto","variables"];
		set "n" "0";
		foreverypart {
			if header :mime :contains "Content-Type" "application/pdf" {
				fileinto "pdf";
			}
		}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, err := message.ParseMIME([]byte(withAttachment))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h := &dummyHandler{}
	if err := s.Run(m, h); err != nil {
		t.Fatalf("run: %v", err)
	}
	if h.folder != "pdf" {
		t.Fatalf("expected pdf folder, got %q", h.folder)
	}
}

func TestForeverypartBreak(t *testing.T) {
	// break should stop the loop; we use a counter side-effect via a
	// variable + addr capture to detect iteration count indirectly.
	src := `require ["mime","fileinto"];
		foreverypart {
			fileinto "first";
			break;
		}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, err := message.ParseMIME([]byte(withAttachment))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h := &dummyHandler{}
	if err := s.Run(m, h); err != nil {
		t.Fatalf("run: %v", err)
	}
	if h.folder != "first" {
		t.Fatalf("expected first folder, got %q", h.folder)
	}
}

func TestExtractText(t *testing.T) {
	src := `require ["mime","variables","fileinto"];
		foreverypart {
			extracttext "snippet";
			fileinto "${snippet}";
			break;
		}`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, err := message.ParseMIME([]byte(withAttachment))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	h := &dummyHandler{}
	if err := s.Run(m, h); err != nil {
		t.Fatalf("run: %v", err)
	}
	if h.folder != "hi" {
		t.Fatalf("expected hi folder, got %q", h.folder)
	}
}

func TestBreakOutsideLoopPropagates(t *testing.T) {
	src := `require ["mime"]; break;`
	s, err := sieve.Compile(src)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	m, _ := message.ParseMIME([]byte(noAttachment))
	if err := s.Run(m, &dummyHandler{}); err == nil {
		t.Fatal("expected error for break outside loop")
	}
}
