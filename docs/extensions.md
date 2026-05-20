# Writing a Sieve extension

This library is designed around a small core (RFC 5228) plus a registry
that anything else plugs into. The
[IANA Sieve Extensions registry](https://www.iana.org/assignments/sieve-extensions/sieve-extensions.xhtml)
lists 100+ extensions; this guide shows how to add one.

## Anatomy of an extension

An extension is a Go package under `extensions/<name>/`. It:

1. Declares a `Capability` constant — the string scripts use in
   `require ["…"];`.
2. Provides a `Register(*sieve.Interpreter)` function that calls one or
   more of:
   * `i.Registry().RegisterAction(name, fn, Capability)`
   * `i.Registry().RegisterTest(name, fn, Capability)`
   * `i.Registry().RegisterMatchType(":tag", fn, Capability)`
3. Has an `init()` that calls `Register(sieve.Default())` so that
   simply importing the package (even via `import _`) makes the
   capability available to `sieve.Compile`.
4. Optionally defines a `Handler` interface that extends
   `sieve.Handler` for any new host-facing actions.

That's all. Validation, capability gating, and `require` integration
are handled by the registry.

## Picking the right hook

| Sieve feature                       | Register as       |
| ----------------------------------- | ----------------- |
| New action (`vacation`, `notify`)   | `RegisterAction`  |
| New test (`body`, `hasflag`)        | `RegisterTest`    |
| New match-type tag (`:regex`)       | `RegisterMatchType` |

Other kinds of tags (address parts, body transforms, comparators) are
inspected by the test that uses them — read the tags off
`ast.Arguments.Tags` inside your `TestFunc`.

## Skeleton

```go
// Package myext implements the Sieve "myext" extension (RFC NNNN).
package myext

import (
    "fmt"

    "sieve"
    "sieve/ast"
    "sieve/registry"
)

const Capability = "myext"

// Handler is the host interface. Omit if the extension only adds tests.
type Handler interface {
    sieve.Handler
    DoTheThing(arg string) error
}

func Register(i *sieve.Interpreter) {
    i.Registry().RegisterAction("dothething", action, Capability)
    i.Registry().RegisterTest("isthething", test, Capability)
}

func init() { Register(sieve.Default()) }

func action(ctx registry.Context, args *ast.Arguments) error {
    if len(args.Positional) != 1 {
        return fmt.Errorf("dothething: expected 1 string argument")
    }
    s, ok := args.Positional[0].(ast.StringValue)
    if !ok {
        return fmt.Errorf("dothething: argument must be a string")
    }
    h, ok := ctx.Handler().(Handler)
    if !ok {
        return fmt.Errorf("dothething: handler does not implement myext.Handler")
    }
    ctx.MarkExplicitAction() // suppress implicit keep, if the action delivers
    return h.DoTheThing(s.Value)
}

func test(ctx registry.Context, args *ast.Arguments, _ []*ast.Test) (bool, error) {
    // Inspect args.Positional and args.Tags, consult ctx.Message().
    return false, nil
}
```

## Arguments and tags

`ast.Arguments` holds two lists in source order:

* `Positional` — `[]ast.Value`, where each value is one of
  `StringValue`, `NumberValue`, or `StringListValue`.
* `Tags` — `[]ast.TaggedArg{Name string, Pos ast.Position}` containing
  every `:tag` token. Tags that carry a follow-on argument leave that
  argument in `Positional` — the registry does not pair them
  automatically, because the pairing rule is per-extension.

Helpers in the core (`interpreter.LookupMatcher`) handle match-type
tags. For your own tags, walk `args.Tags` and use
`strings.EqualFold(tg.Name, ":mytag")`.

## Match-type extensions

A match type is a function `func(value, key string) bool`. Register it
as `:tagname` (include the leading colon). Existing tests
(`header`, `address`, `envelope`, `body`, …) will pick it up
automatically, because they consult the registry via
`interpreter.LookupMatcher`.

```go
func Register(i *sieve.Interpreter) {
    i.Registry().RegisterMatchType(":regex", matchRegex, Capability)
}
```

Validation enforces that scripts which use your match type also
`require` the capability — see `extensions/regex/`.

## Host actions that need richer interfaces

If your action needs the host to do something not in `sieve.Handler`,
define an interface in your extension package that embeds
`sieve.Handler` and add the new methods. Inside the action body:

```go
h, ok := ctx.Handler().(myext.Handler)
if !ok {
    return fmt.Errorf("dothething: handler does not implement myext.Handler")
}
```

This keeps `sieve.Handler` minimal and lets the host opt into only the
extensions it actually supports.

## Implicit keep

If your action delivers the message somewhere (the way `fileinto`,
`redirect`, and `discard` do), call `ctx.MarkExplicitAction()` so the
interpreter does not run the implicit-keep at the end of the script
(RFC 5228 §2.10.6).

## Run-time `stop`

If your action ever needs to terminate script execution, call
`ctx.Stop()`. The interpreter checks this between commands.

## Testing

Add an integration test to `extensions_test.go` that compiles and runs
a small script exercising your extension. Cover at least:

* Success path (extension does what it should).
* Validation failure when `require` is missing.
* Argument errors (wrong type, wrong count).

Then run `go vet ./... && go test ./...`.

## Wiring it in

1. Add a one-line mention to `readme.md` under the status list.
2. Don't forget the blank import in `examples/` if you want the
   examples to exercise it.

Hosts that build a custom interpreter (`sieve.NewInterpreter()`) call
`myext.Register(i)` explicitly instead of relying on the default
registry; both styles work side by side.

## Examples in this repo

Pick the one closest to what you're building:

* `extensions/fileinto/` — simplest action with a host sub-interface.
* `extensions/envelope/` — test that reuses a core implementation under
  a different capability.
* `extensions/body/` — test that reads the message body, with body
  transforms.
* `extensions/imap4flags/` — multiple actions and a test that share a
  Handler interface.
* `extensions/regex/` — match-type extension; no actions or tests of
  its own.
