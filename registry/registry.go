// Package registry holds the set of Sieve commands, tests, and tagged
// arguments that an interpreter knows about. Core (RFC 5228) builtins
// are registered automatically by package interpreter on construction;
// extension packages register additional functionality through their
// init() and can opt in to a capability that scripts must `require`.
//
// The registry is the single source of truth for what a script may
// contain: validation walks the AST and checks every command/test name
// against the registry, and the interpreter dispatches through it.
package registry

import (
	"fmt"
	"sync"

	"sieve/ast"
	"sieve/message"
)

// ActionFunc executes a registered action against the host.
//
// Implementations may inspect args (positional and tagged) and call into
// the supplied Handler. They must not retain refs to args after returning.
type ActionFunc func(ctx Context, args *ast.Arguments) error

// TestFunc evaluates a registered test against the current message.
type TestFunc func(ctx Context, args *ast.Arguments, children []*ast.Test) (bool, error)

// Context is what action/test implementations are given at run time.
// It is intentionally minimal; richer extensions can grow it through
// well-known keys on State if needed.
type Context interface {
	Message() message.Message
	Handler() Handler
	// EvalTest dispatches a child test through the registry. allof/anyof/not
	// use this to evaluate their children.
	EvalTest(t *ast.Test) (bool, error)
	// MarkExplicitAction is called by actions that satisfy the implicit-keep
	// rule (RFC 5228 §2.10.6) so the interpreter knows not to keep at the
	// end of the script.
	MarkExplicitAction()
	// Stop signals "stop;" — the interpreter halts after the current
	// command returns.
	Stop()
}

// Handler is what the host application implements to receive Sieve
// actions. Extensions that add new actions can either type-assert this
// interface to a richer one defined in their own package, or accept a
// generic ExtHandler from State (see Context).
type Handler interface {
	Keep() error
	Discard() error
	Redirect(addr string) error
}

// Registry holds registered actions, tests, and capabilities.
type Registry struct {
	mu      sync.RWMutex
	actions map[string]actionEntry
	tests   map[string]testEntry
	// caps is the set of capabilities considered "available" by this
	// registry. Core capabilities (none for RFC 5228 base) plus anything
	// registered with a non-empty Requires.
	caps map[string]bool
}

type actionEntry struct {
	fn       ActionFunc
	requires string // capability name, or "" for always-available
}

type testEntry struct {
	fn       TestFunc
	requires string
}

func New() *Registry {
	return &Registry{
		actions: map[string]actionEntry{},
		tests:   map[string]testEntry{},
		caps:    map[string]bool{},
	}
}

// RegisterAction adds an action. If requires is non-empty, scripts must
// `require "<cap>";` to use this action.
func (r *Registry) RegisterAction(name string, fn ActionFunc, requires string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.actions[name] = actionEntry{fn: fn, requires: requires}
	if requires != "" {
		r.caps[requires] = true
	}
}

// RegisterTest adds a test under the given name.
func (r *Registry) RegisterTest(name string, fn TestFunc, requires string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tests[name] = testEntry{fn: fn, requires: requires}
	if requires != "" {
		r.caps[requires] = true
	}
}

// LookupAction returns the action and its required capability.
func (r *Registry) LookupAction(name string) (ActionFunc, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.actions[name]
	if !ok {
		return nil, "", false
	}
	return e.fn, e.requires, true
}

// LookupTest returns the test and its required capability.
func (r *Registry) LookupTest(name string) (TestFunc, string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.tests[name]
	if !ok {
		return nil, "", false
	}
	return e.fn, e.requires, true
}

// HasCapability reports whether a capability string is known.
func (r *Registry) HasCapability(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.caps[name]
}

// ErrUnknown is returned by validation when a command or test is not in
// the registry.
type ErrUnknown struct {
	Kind string // "action" or "test"
	Name string
}

func (e *ErrUnknown) Error() string { return fmt.Sprintf("unknown %s %q", e.Kind, e.Name) }
