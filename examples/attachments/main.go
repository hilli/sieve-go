// Demonstrates RFC 5703 "mime" extension: filing any message with an
// attachment into a dedicated folder.
package main

import (
	"fmt"
	"log"

	"github.com/hilli/sieve-go"
	"github.com/hilli/sieve-go/message"

	_ "github.com/hilli/sieve-go/extensions/fileinto"
	_ "github.com/hilli/sieve-go/extensions/mime"
)

const script = `require ["mime","fileinto"];

if header :mime :anychild :contains "Content-Disposition" "attachment" {
    fileinto "Attachments";
} else {
    keep;
}
`

const raw = "From: a@example.com\r\n" +
	"To: b@example.com\r\n" +
	"Subject: report\r\n" +
	"MIME-Version: 1.0\r\n" +
	"Content-Type: multipart/mixed; boundary=\"B\"\r\n" +
	"\r\n" +
	"--B\r\nContent-Type: text/plain\r\n\r\nSee attached.\r\n" +
	"--B\r\nContent-Type: application/pdf\r\nContent-Disposition: attachment; filename=\"q1.pdf\"\r\n\r\nPDF...\r\n" +
	"--B--\r\n"

type handler struct{}

func (handler) Keep() error              { fmt.Println("kept"); return nil }
func (handler) Discard() error           { fmt.Println("discarded"); return nil }
func (handler) Redirect(a string) error  { fmt.Println("redirect", a); return nil }
func (handler) FileInto(f string) error  { fmt.Println("fileinto", f); return nil }

func main() {
	s, err := sieve.Compile(script)
	if err != nil {
		log.Fatal(err)
	}
	msg, err := message.ParseMIME([]byte(raw))
	if err != nil {
		log.Fatal(err)
	}
	if err := s.Run(msg, handler{}); err != nil {
		log.Fatal(err)
	}
}
