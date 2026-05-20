# sieve-go

A pluggable [Sieve mail filtering language](https://www.rfc-editor.org/rfc/rfc5228)
implementation in Go. Designed to be embedded in mail-handling
applications: parse a script, plug in your handler, route mail.

* **Compile / Validate / Run** — separate parsing and validation from
  execution, so you can accept user-submitted scripts safely.
* **Extensible** — actions, tests, and match-type tags live in a
  registry. Each [IANA Sieve extension](https://www.iana.org/assignments/sieve-extensions/sieve-extensions.xhtml)
  can be a small standalone package.
* **Mail-format agnostic** — your application supplies a `Message`
  interface; a `net/mail` adapter is included.
* **No external dependencies.**

## Status

RFC 5228 core: lexer, parser, AST, registry, interpreter with `if` /
`elsif` / `else`, `stop`, implicit keep, the standard actions
(`keep`, `discard`, `redirect`), tests (`address`, `header`, `exists`,
`size`), combinators (`allof`, `anyof`, `not`, `true`, `false`), match
types (`:is`, `:contains`, `:matches` with `*` / `?` globs), and
address parts (`:all`, `:localpart`, `:domain`).

Bundled extensions:

| Package                       | Capability   | Spec                          |
| ----------------------------- | ------------ | ----------------------------- |
| `extensions/fileinto`         | `fileinto`   | RFC 5232 (incl. `:flags`)     |
| `extensions/envelope`         | `envelope`   | RFC 5228 §5.4                 |
| `extensions/body`             | `body`       | RFC 5173 (`:raw`/`:text`/`:content`) |
| `extensions/imap4flags`       | `imap4flags` | RFC 5232 (full)               |
| `extensions/variables`        | `variables`  | RFC 5229                      |
| `extensions/regex`            | `regex`      | draft-ietf-sieve-regex        |
| `extensions/mime`             | `mime`       | RFC 5703 (`:mime` / `:anychild`) |

The core also honours the standard `:comparator` tag with the
`i;ascii-casemap` (default) and `i;octet` comparators from RFC 4790.

## Install

```sh
go get github.com/hilli/sieve-go
```

## Quick start

```go
package main

import (
    "fmt"
    "log"

    "github.com/hilli/sieve-go"
    _ "github.com/hilli/sieve-go/extensions/fileinto" // enables `require ["fileinto"];`
    "github.com/hilli/sieve-go/message"
)

type Handler struct{ delivered string }

func (h *Handler) Keep() error               { h.delivered = "INBOX"; return nil }
func (h *Handler) FileInto(mb string) error  { h.delivered = mb; return nil }
func (h *Handler) Discard() error            { h.delivered = "/dev/null"; return nil }
func (h *Handler) Redirect(addr string) error { return nil }

func main() {
    script, err := sieve.Compile(`
        require ["fileinto"];
        if header :contains "Subject" "[oncall]" {
            fileinto "Oncall";
            stop;
        }
    `)
    if err != nil { log.Fatal(err) }

    msg := message.NewBuilder().
        AddHeader("From", "alice@example.com").
        AddHeader("Subject", "[oncall] disk full").
        Build()

    h := &Handler{}
    if err := script.Run(msg, h); err != nil { log.Fatal(err) }
    fmt.Println("delivered to:", h.delivered) // -> Oncall
}
```

### Validate without running

For UIs that accept user-submitted scripts:

```go
if err := sieve.Validate(userScript); err != nil {
    return fmt.Errorf("invalid sieve script: %w", err)
}
```

Validation catches unknown commands, unknown tests, missing `require`,
malformed argument shapes, and similar problems before any message ever
touches the script.

### Run the examples

```sh
go run ./examples/simple
go run ./examples/attachments
echo 'keep;' | go run ./examples/validate
```

### Detecting attachments

Load the `mime` extension and parse the raw message with
`message.ParseMIME`, which surfaces MIME parts to `:mime :anychild`:

```go
import (
    _ "github.com/hilli/sieve-go/extensions/fileinto"
    _ "github.com/hilli/sieve-go/extensions/mime"
)

script, _ := sieve.Compile(`require ["mime","fileinto"];
if header :mime :anychild :contains "Content-Disposition" "attachment" {
    fileinto "Attachments";
}`)
msg, _ := message.ParseMIME(rawMessageBytes)
script.Run(msg, handler)
```

## Extending the language

Each Sieve extension is a separate Go package that self-registers on
import. To enable one in scripts processed by the package-level
`sieve.Compile`, just blank-import it:

```go
import (
    _ "github.com/hilli/sieve-go/extensions/body"
    _ "github.com/hilli/sieve-go/extensions/regex"
)
```

…and scripts use `require ["body", "regex"];`.

For hosts that need an isolated registry (e.g. multiple tenants with
different allowed extensions), build a fresh interpreter:

```go
i := sieve.NewInterpreter()
fileinto.Register(i)        // explicit, no global side effects
s, err := /* parse and compile through i */
```

To **write** a new extension, see [`docs/extensions.md`](docs/extensions.md).

## Package layout

```
.
├── sieve.go            top-level façade: Compile, Validate, NewInterpreter, Default
├── ast/                AST node types
├── token/              token types
├── lexer/              source → tokens
├── parser/             tokens → AST
├── registry/           pluggable actions/tests/match-types/capabilities
├── interpreter/        AST walker + RFC 5228 builtins
├── message/            Message interface + net/mail adapter + Builder
├── extensions/
│   ├── fileinto/       RFC 5232
│   ├── envelope/       RFC 5228 §5.4
│   ├── body/           RFC 5173
│   ├── imap4flags/     RFC 5232
│   ├── variables/      RFC 5229
│   ├── regex/          draft-ietf-sieve-regex
│   └── mime/           RFC 5703 (subset)
├── examples/
│   ├── simple/         embed the library
│   └── validate/       stdin → ok / error CLI
├── docs/
│   └── extensions.md   how to write an extension
└── AGENTS.md           guidance for AI coding agents
```

## Limitations

Known unimplemented pieces — each one fails loudly (validation error,
runtime error, or simply doesn't fire) rather than silently mismatching:

* **RFC 5703 (`mime`) control flow** — only the `:mime` / `:anychild`
  tags ship. `foreverypart`, `break`, `replace`, `enclose`, and
  `extracttext` are not registered, so scripts using them fail
  validation as unknown commands. (Adding them requires a new
  `RegisterCommand` mechanism in the registry.)
* **Other IANA extensions** — `include`, `reject`, `vacation`,
  `editheader`, `notify`, `relational`, `subaddress`, etc. are not
  built. The registry is the extension point — see
  `docs/extensions.md`.

These are good first contributions if you want to dig in.

## References

* [RFC 5228](https://www.rfc-editor.org/rfc/rfc5228) — Sieve core
* [RFC 5229](https://www.rfc-editor.org/rfc/rfc5229) — variables
* [RFC 5232](https://www.rfc-editor.org/rfc/rfc5232) — fileinto, imap4flags
* [RFC 5173](https://www.rfc-editor.org/rfc/rfc5173) — body
* [RFC 5703](https://www.rfc-editor.org/rfc/rfc5703) — MIME part tests
* [IANA Sieve Extensions](https://www.iana.org/assignments/sieve-extensions/sieve-extensions.xhtml)
* [Writing An Interpreter In Go](https://interpreterbook.com) — overall design inspiration
