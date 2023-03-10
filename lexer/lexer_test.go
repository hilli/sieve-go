package lexer

import (
	"sieve/token"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `require [ "fileinto" ];
	if address :matches "From" "test@example.com" {
		fileinto "INBOX.test";
	}`

	tests := []struct {
		expectedType    token.TokenType
		expectedLiteral string
	}{
		{token.REQUIRE, "require"},
		{token.LBRACK, "["},
		{token.STRING, "fileinto"},
		{token.RBRACK, "]"},
		{token.SEMICOLON, ";"},
		{token.IF, "if"},
		{token.IDENT, "address"},
		{token.MATCHES, ":matches"},
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
	for i, tt := range tests {
		tok := l.NextToken()
		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong, expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong, expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}
