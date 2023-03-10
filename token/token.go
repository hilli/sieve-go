package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// Identifiers + literals
	IDENT  = "IDENT"  // add, foobar, x, y, ...
	STRING = "STRING" //
	INT   = "INT"   // 12345

	// Operators
	ASSIGN = "="
	PLUS   = "+"

	// Delimiters
	COMMA     = ","
	SEMICOLON = ";"

	LPAREN = "("
	RPAREN = ")"
	LBRACE = "{"
	RBRACE = "}"
	LBRACK = "["
	RBRACK = "]"

	// Keywords
	IF       = "IF"
	ADDRESS  = "ADDRESS"
	MATCHES  = ":MATCHES"
	IS       = ":IS"
	CONTAINS = ":CONTAINS"
	REQUIRE = "REQUIRE"
)


var keywords = map[string]TokenType{
	"require":  REQUIRE,
	"if":       IF,
	"address":  ADDRESS,
	"matches":  MATCHES,
	"is":       IS,
	"contains": CONTAINS,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
