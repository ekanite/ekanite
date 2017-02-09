package query

import "strings"

// Token represents a lexical token.
type Token int

const (
	ILLEGAL Token = iota // Illegal tokens
	EOF                  // End-of-file
	WS                   // Whitespace
	COLON                // ;

	// STRING represents search terms
	STRING // search fields terms

	keywordBeg

	AND // AND boolean
	OR  // OR boolean
	NOT // NOT boolean

	keywordEnd

	LPAREN // (
	RPAREN // )

)

var tokens = [...]string{
	ILLEGAL: "ILLEGAL",
	EOF:     "EOF",
	WS:      "WS",
	COLON:   ":",

	AND: "AND",
	OR:  "OR",
	NOT: "NOT",

	LPAREN: "(",
	RPAREN: ")",
}

var keywords map[string]Token

func init() {
	keywords = make(map[string]Token)
	for tok := keywordBeg + 1; tok < keywordEnd; tok++ {
		keywords[strings.ToLower(tokens[tok])] = tok
	}
	for _, tok := range []Token{AND, OR} {
		keywords[strings.ToLower(tokens[tok])] = tok
	}
}

func (t Token) isOperator() bool {
	return t == AND || t == OR || t == NOT
}

// String returns the string representation of the token.
func (t Token) String() string {
	if t >= 0 && t < Token(len(tokens)) {
		return tokens[t]
	}
	return ""
}

// Precedence returns the operator precedence of the binary operator token.
func (t Token) Precedence() int {
	switch t {
	case OR:
		return 1
	case AND:
		return 2
	case NOT:
		return 3
	}
	return 0
}

// Lookup returns the token associated with a given string.
func Lookup(ident string) (Token, bool) {
	if tok, ok := keywords[strings.ToLower(ident)]; ok {
		return tok, true
	}
	return ILLEGAL, false
}

// tokstr returns a literal if provided, otherwise returns the token string.
func tokstr(t Token, lit string) string {
	if lit != "" {
		return lit
	}
	return t.String()
}
