package ast

import (
	"testing"

	"github.com/hilli/sieve-go/token"
)

func TestPosFrom(t *testing.T) {
	tk := token.Token{Line: 7, Col: 3}
	p := PosFrom(tk)
	if p.Line != 7 || p.Col != 3 {
		t.Fatalf("PosFrom: got %+v", p)
	}
}

// TestValueMarkers ensures each Value implementation satisfies the
// interface (compile-time + run-time check via a switch).
func TestValueMarkers(t *testing.T) {
	values := []Value{
		StringValue{Value: "x"},
		NumberValue{Value: 1},
		StringListValue{Values: []string{"a"}},
	}
	for _, v := range values {
		switch v.(type) {
		case StringValue, NumberValue, StringListValue:
			// ok
		default:
			t.Fatalf("unexpected value type %T", v)
		}
	}
}
