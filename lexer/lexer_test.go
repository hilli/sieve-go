package lexer

import (
	"testing"

	"sieve/token"
)

func TestNextToken_Basic(t *testing.T) {
	input := `require ["fileinto"];
if address :matches "From" "test@example.com" {
	fileinto "INBOX.test";
}`

	want := []struct {
		typ token.TokenType
		lit string
	}{
		{token.IDENT, "require"},
		{token.LBRACK, "["},
		{token.STRING, "fileinto"},
		{token.RBRACK, "]"},
		{token.SEMICOLON, ";"},
		{token.IDENT, "if"},
		{token.IDENT, "address"},
		{token.TAG, ":matches"},
		{token.STRING, "From"},
		{token.STRING, "test@example.com"},
		{token.LBRACE, "{"},
		{token.IDENT, "fileinto"},
		{token.STRING, "INBOX.test"},
		{token.SEMICOLON, ";"},
		{token.RBRACE, "}"},
		{token.EOF, ""},
	}

	l := New(input)
	for i, tt := range want {
		tok := l.NextToken()
		if tok.Type != tt.typ {
			t.Fatalf("tests[%d] type: want %q got %q (lit=%q)", i, tt.typ, tok.Type, tok.Literal)
		}
		if tok.Literal != tt.lit {
			t.Fatalf("tests[%d] lit: want %q got %q", i, tt.lit, tok.Literal)
		}
	}
}

func TestNextToken_NumbersAndQuantifiers(t *testing.T) {
	l := New(`size :over 500K; size :under 2M; 3G 7`)
	want := []struct {
		typ token.TokenType
		lit string
	}{
		{token.IDENT, "size"}, {token.TAG, ":over"}, {token.NUMBER, "512000"}, {token.SEMICOLON, ";"},
		{token.IDENT, "size"}, {token.TAG, ":under"}, {token.NUMBER, "2097152"}, {token.SEMICOLON, ";"},
		{token.NUMBER, "3221225472"}, {token.NUMBER, "7"}, {token.EOF, ""},
	}
	for i, tt := range want {
		tok := l.NextToken()
		if tok.Type != tt.typ || tok.Literal != tt.lit {
			t.Fatalf("tests[%d]: want %q/%q got %q/%q", i, tt.typ, tt.lit, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_Comments(t *testing.T) {
	l := New(`# hash comment
/* block
   comment */ keep ; # trailing`)
	want := []struct {
		typ token.TokenType
		lit string
	}{
		{token.IDENT, "keep"},
		{token.SEMICOLON, ";"},
		{token.EOF, ""},
	}
	for i, tt := range want {
		tok := l.NextToken()
		if tok.Type != tt.typ || tok.Literal != tt.lit {
			t.Fatalf("tests[%d]: want %q/%q got %q/%q", i, tt.typ, tt.lit, tok.Type, tok.Literal)
		}
	}
}

func TestNextToken_QuotedStringEscapes(t *testing.T) {
	l := New(`"a\"b\\c\nd"`)
	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("type: want STRING got %q", tok.Type)
	}
	// \" -> ", \\ -> \, \n -> n (undefined escape: char itself, backslash dropped)
	if tok.Literal != `a"b\cnd` {
		t.Fatalf("lit: got %q", tok.Literal)
	}
}

func TestNextToken_MultilineText(t *testing.T) {
	input := "text:\nhello\n..dotted\n.\n"
	l := New(input)
	tok := l.NextToken()
	if tok.Type != token.STRING {
		t.Fatalf("type: want STRING got %q (lit=%q)", tok.Type, tok.Literal)
	}
	if tok.Literal != "hello\n.dotted\n" {
		t.Fatalf("lit: got %q", tok.Literal)
	}
	if eof := l.NextToken(); eof.Type != token.EOF {
		t.Fatalf("want EOF, got %q", eof.Type)
	}
}

func TestNextToken_Position(t *testing.T) {
	l := New("if\n  address")
	tok := l.NextToken()
	if tok.Line != 1 || tok.Col != 1 {
		t.Fatalf("if: want 1:1 got %d:%d", tok.Line, tok.Col)
	}
	tok = l.NextToken()
	if tok.Line != 2 || tok.Col != 3 {
		t.Fatalf("address: want 2:3 got %d:%d", tok.Line, tok.Col)
	}
}

func TestNextToken_UnterminatedString(t *testing.T) {
	l := New(`"oops`)
	tok := l.NextToken()
	if tok.Type != token.ILLEGAL {
		t.Fatalf("want ILLEGAL got %q", tok.Type)
	}
}
