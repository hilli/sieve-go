# sieve-go

A [Sieve mail filtering language](https://www.rfc-editor.org/rfc/rfc5228)
implementation in Go, designed to be embedded in mail-handling
applications. Parse, validate, and execute Sieve scripts; plug in your
own action handlers; extend the language with new commands and tests.

## Status

RFC 5228 core implemented:

* lexer with quoted/multi-line strings, numbers with K/M/G quantifiers,
  tagged arguments, hash + bracket comments
* recursive-descent parser → AST
* extension registry for commands, tests, and capabilities
* interpreter with `if`/`elsif`/`else`, `stop`, implicit keep
* core actions: `keep`, `discard`, `redirect`
* core tests: `address`, `header`, `exists`, `size`, `envelope`
* match types: `:is`, `:contains`, `:matches` (with `*` / `?` globs)
* address parts: `:all`, `:localpart`, `:domain`
* combinators: `allof`, `anyof`, `not`, `true`, `false`
* `validate` separate from `run`
* example extensions:
  * `fileinto` (RFC 5232)
  * `envelope` (RFC 5228 §5.4 capability)
  * `body` (RFC 5173) — `:raw` / `:text`
  * `imap4flags` (RFC 5232) — `setflag`/`addflag`/`removeflag`/`hasflag`
  * `regex` (draft-ietf-sieve-regex) — `:regex` match type

## Install

```
go get sieve
```

## Usage

```go
import (
    "sieve"
    _ "sieve/extensions/fileinto" // self-registers fileinto
    "sieve/message"
)

type Handler struct{}
func (Handler) Keep() error                { /* deliver to INBOX */ return nil }
func (Handler) Discard() error             { return nil }
func (Handler) Redirect(addr string) error { /* forward */ return nil }
func (Handler) FileInto(mb string) error   { /* deliver to mb */ return nil }

s, err := sieve.Compile(`
    require ["fileinto"];
    if header :contains "Subject" "[oncall]" {
        fileinto "Oncall";
        stop;
    }
`)
if err != nil { panic(err) }

msg := message.NewBuilder().
    AddHeader("From", "alice@example.com").
    AddHeader("Subject", "[oncall] page").
    Build()

_ = s.Run(msg, Handler{})
```

### Validate only

```go
if err := sieve.Validate(userSubmittedScript); err != nil {
    return fmt.Errorf("invalid: %w", err)
}
```

### Examples

```
go run ./examples/simple
echo 'keep;' | go run ./examples/validate
```

## Adding extensions

Each Sieve extension lives in its own package under `extensions/`.
A registration looks like:

```go
package myext

import "sieve"

const Capability = "myext"

func Register(i *sieve.Interpreter) {
    i.Registry().RegisterAction("myaction", action, Capability)
    i.Registry().RegisterTest("mytest", test, Capability)
}

func init() { Register(sieve.Default()) }
```

Scripts then opt in with `require ["myext"];`. Validation fails if the
capability is missing from the registry, so importing the package is the
gate.

See `extensions/fileinto/fileinto.go` for a complete worked example.
There are 100+ extensions in the
[IANA registry](https://www.iana.org/assignments/sieve-extensions/sieve-extensions.xhtml);
the registry-based design lets each be added as a self-contained package.

## Package layout

```
.
├── sieve.go        top-level façade (Compile, Validate, NewInterpreter)
├── ast/            AST node types
├── token/          token type constants
├── lexer/          lexer
├── parser/         parser → ast.Script
├── registry/       extensible action/test/capability registry
├── interpreter/    AST walker + RFC 5228 builtins
├── message/        Message interface + net/mail adapter + Builder
├── extensions/
│   ├── fileinto/   RFC 5232
│   ├── envelope/   RFC 5228 §5.4
│   ├── body/       RFC 5173
│   ├── imap4flags/ RFC 5232 (subset)
│   └── regex/      draft-ietf-sieve-regex
└── examples/
    ├── simple/     embed library, run script
    └── validate/   validate-only CLI
```

## References

* [RFC 5228](https://www.rfc-editor.org/rfc/rfc5228) — Sieve core
* [RFC 5232](https://www.rfc-editor.org/rfc/rfc5232) — fileinto
* [IANA Sieve Extensions](https://www.iana.org/assignments/sieve-extensions/sieve-extensions.xhtml)
* [Writing An Interpreter In Go](https://interpreterbook.com) — overall design inspiration
