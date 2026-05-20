// Command simple shows how a host application embeds the sieve library:
// parse a script, supply a message, and implement a Handler.
package main

import (
	"fmt"
	"log"

	"github.com/hilli/sieve-go"
	_ "github.com/hilli/sieve-go/extensions/fileinto" // self-registers fileinto
	"github.com/hilli/sieve-go/message"
)

const script = `
require ["fileinto", "envelope"];

# Route oncall pages to a dedicated mailbox.
if header :contains "Subject" "[oncall]" {
    fileinto "Oncall";
    stop;
}

# Drop mail from known-bad senders.
if address :is :domain "From" "spam.example" {
    discard;
}

# Forward anything tagged for review.
if header :matches "Subject" "*[review]*" {
    redirect "review@example.com";
}
`

// handler implements both sieve.Handler and fileinto.Handler.
type handler struct{ delivered string }

func (h *handler) Keep() error               { h.delivered = "INBOX"; return nil }
func (h *handler) FileInto(mb string) error  { h.delivered = mb; return nil }
func (h *handler) Discard() error            { h.delivered = "/dev/null"; return nil }
func (h *handler) Redirect(addr string) error {
	fmt.Printf("(would forward to %s)\n", addr)
	h.delivered = "forwarded:" + addr
	return nil
}

func main() {
	s, err := sieve.Compile(script)
	if err != nil {
		log.Fatalf("compile: %v", err)
	}

	msg := message.NewBuilder().
		AddHeader("From", "alice@example.com").
		AddHeader("To", "team@example.com").
		AddHeader("Subject", "[oncall] disk full on db-01").
		SetBody([]byte("please ack")).
		Build()

	h := &handler{}
	if err := s.Run(msg, h); err != nil {
		log.Fatalf("run: %v", err)
	}
	fmt.Printf("delivered to: %s\n", h.delivered)
}
