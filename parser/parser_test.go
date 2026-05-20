package parser

import (
	"testing"

	"sieve/ast"
)

func TestParse_Simple(t *testing.T) {
	src := `require ["fileinto"];
if address :is :domain "From" "example.com" {
    fileinto "INBOX.example";
    stop;
}
keep;`
	s, err := Parse(src)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got := len(s.Commands); got != 3 {
		t.Fatalf("want 3 top-level commands, got %d", got)
	}
	req := s.Commands[0]
	if req.Name != "require" {
		t.Fatalf("first cmd: want require, got %s", req.Name)
	}
	sl, ok := req.Args.Positional[0].(ast.StringListValue)
	if !ok {
		t.Fatalf("require arg: want StringListValue, got %T", req.Args.Positional[0])
	}
	if len(sl.Values) != 1 || sl.Values[0] != "fileinto" {
		t.Fatalf("require list: got %v", sl.Values)
	}

	ifCmd := s.Commands[1]
	if ifCmd.Name != "if" || ifCmd.Test == nil || !ifCmd.HasBlock {
		t.Fatalf("if command not parsed: %+v", ifCmd)
	}
	if ifCmd.Test.Name != "address" || len(ifCmd.Test.Args.Tags) != 2 {
		t.Fatalf("address test: %+v", ifCmd.Test)
	}
	if len(ifCmd.Block) != 2 {
		t.Fatalf("if block: want 2 commands, got %d", len(ifCmd.Block))
	}
}

func TestParse_AllofAnyofNot(t *testing.T) {
	src := `if allof(not exists "X-Spam", anyof(header :contains "Subject" "hi", size :over 1K)) { discard; }`
	s, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(s.Commands) != 1 {
		t.Fatalf("want 1 cmd, got %d", len(s.Commands))
	}
	test := s.Commands[0].Test
	if test.Name != "allof" || len(test.Children) != 2 {
		t.Fatalf("allof: %+v", test)
	}
	if test.Children[0].Name != "not" || test.Children[0].Children[0].Name != "exists" {
		t.Fatalf("not/exists: %+v", test.Children[0])
	}
	any := test.Children[1]
	if any.Name != "anyof" || len(any.Children) != 2 {
		t.Fatalf("anyof: %+v", any)
	}
}

func TestParse_ElsifElse(t *testing.T) {
	src := `if header :is "X" "1" { keep; } elsif header :is "X" "2" { discard; } else { stop; }`
	s, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(s.Commands) != 3 {
		t.Fatalf("want 3 cmds, got %d", len(s.Commands))
	}
	if s.Commands[1].Name != "elsif" || s.Commands[1].Test == nil {
		t.Fatalf("elsif: %+v", s.Commands[1])
	}
	if s.Commands[2].Name != "else" || s.Commands[2].Test != nil || !s.Commands[2].HasBlock {
		t.Fatalf("else: %+v", s.Commands[2])
	}
}

func TestParse_Errors(t *testing.T) {
	cases := []string{
		`if {}`,        // missing test
		`keep`,         // missing ;
		`keep ;;`,      // stray ;
		`["a" "b"];`,   // starts with [
		`require [1];`, // non-string in list
	}
	for _, src := range cases {
		if _, err := Parse(src); err == nil {
			t.Errorf("expected error for %q", src)
		}
	}
}
