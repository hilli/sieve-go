// Command validate checks a Sieve script for syntax and semantic errors
// without running it. Useful for accepting user-submitted scripts in a
// web UI or similar.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hilli/sieve-go"
	_ "github.com/hilli/sieve-go/extensions/fileinto"
)

func main() {
	flag.Parse()

	var src []byte
	var err error
	if flag.NArg() == 1 {
		src, err = os.ReadFile(flag.Arg(0))
	} else {
		src, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	if err := sieve.Validate(string(src)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("ok")
}
