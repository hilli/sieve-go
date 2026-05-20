// Package fileinto implements the Sieve "fileinto" extension (RFC 5232).
//
// The host application must implement the Handler interface (which
// extends sieve.Handler with a FileInto method). Importing this package
// for its side effects registers the action on the default interpreter.
// To install the extension into a custom interpreter, call Register.
package fileinto

import (
	"fmt"

	"sieve"
	"sieve/ast"
	"sieve/registry"
)

// Capability is the string scripts must `require` to use fileinto.
const Capability = "fileinto"

// Handler is the action interface the host must implement. It extends
// the core sieve.Handler with the FileInto method.
type Handler interface {
	sieve.Handler
	FileInto(mailbox string) error
}

// Register installs the fileinto action on the given interpreter.
func Register(i *sieve.Interpreter) {
	i.Registry().RegisterAction("fileinto", action, Capability)
}

func action(ctx registry.Context, args *ast.Arguments) error {
	if len(args.Positional) != 1 {
		return fmt.Errorf("fileinto: expected 1 string argument")
	}
	s, ok := args.Positional[0].(ast.StringValue)
	if !ok {
		return fmt.Errorf("fileinto: argument must be a string")
	}
	h, ok := ctx.Handler().(Handler)
	if !ok {
		return fmt.Errorf("fileinto: handler does not implement fileinto.Handler")
	}
	ctx.MarkExplicitAction()
	return h.FileInto(s.Value)
}

func init() {
	// Self-register on the default interpreter so that simply importing
	// this package enables fileinto for the package-level sieve.Compile.
	// Hosts that build a custom interpreter should call Register(i) on
	// it directly.
	Register(defaultI)
}

// defaultI is a reference to the same interpreter used by sieve.Compile.
// We obtain it through a small unexported accessor in the sieve package.
var defaultI = sieve.Default()
