// Package ast defines the Sieve abstract syntax tree.
//
// A Script is a sequence of Commands. A Command has a name, positional
// and tagged arguments, an optional test (for control-flow commands), and
// an optional block of nested commands. Tests have the same shape as
// commands but produce a boolean and never have a block.
//
// The AST is intentionally minimal and uniform: command and test names are
// stored as strings rather than typed constants so that the parser does
// not need to know which extensions are loaded. Semantic validation
// happens later, against the registry.
package ast

import "sieve/token"

// Position is a 1-based line/column in the source.
type Position struct {
	Line int
	Col  int
}

func PosFrom(t token.Token) Position { return Position{Line: t.Line, Col: t.Col} }

// Script is a parsed Sieve script.
type Script struct {
	Commands []*Command
}

// Command is one statement: an identifier followed by arguments and either
// a ";" or a block. For control-flow commands (if/elsif/else) the Test
// field is populated.
type Command struct {
	Name     string
	Pos      Position
	Args     Arguments
	Test     *Test      // for if/elsif; nil otherwise
	Block    []*Command // for if/elsif/else and any future block command; nil otherwise
	HasBlock bool       // distinguishes empty block "{ }" from no block at all
}

// Test is a boolean expression used inside if/elsif and combinators
// (allof/anyof/not).
type Test struct {
	Name     string
	Pos      Position
	Args     Arguments
	Children []*Test // for allof/anyof/not
}

// Arguments holds positional and tagged arguments in source order.
// Positional values may be StringValue, NumberValue, or StringListValue.
type Arguments struct {
	Tags       []TaggedArg
	Positional []Value
}

// TaggedArg is a :tag argument. RFC 5228 tags can either be bare flags
// (e.g. :is) or carry a positional argument that follows (e.g.
// :comparator "i;ascii-casemap"). The follow-on argument is left in
// Positional in source order; the registry interprets the relationship.
type TaggedArg struct {
	Name string // includes leading ":"
	Pos  Position
}

// Value is any positional argument value.
type Value interface{ valueNode() }

type StringValue struct {
	Value string
	Pos   Position
}

type NumberValue struct {
	// Canonical byte count (K/M/G already applied by the lexer).
	Value uint64
	Pos   Position
}

type StringListValue struct {
	Values []string
	Pos    Position
}

func (StringValue) valueNode()     {}
func (NumberValue) valueNode()     {}
func (StringListValue) valueNode() {}
