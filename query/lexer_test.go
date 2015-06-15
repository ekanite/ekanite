package query

import (
	"reflect"
	"strings"
	"testing"
)

// Test lexing of individual tokens
func TestLexer_Lex(t *testing.T) {
	var tests = []struct {
		s   string
		tok Token
		lit string
	}{
		// EOR and whitespace
		{s: ``, tok: EOF},
		{s: ` `, tok: WS, lit: " "},
		{s: "\t", tok: WS, lit: "\t"},
		{s: " ws", tok: WS, lit: " "},
		{s: ":", tok: COLON, lit: ":"},
		// Strings
		{s: `foo`, tok: STRING, lit: `foo`},
		{s: `_foo`, tok: STRING, lit: `_foo`},
		{s: `"qux.qaz`, tok: STRING, lit: `"qux.qaz`},
		{s: "apache.status", tok: STRING, lit: "apache.status"},
		{s: "time", tok: STRING, lit: "time"},
		{s: "_myfield:", tok: STRING, lit: "_myfield"},
		{s: "500)", tok: STRING, lit: "500"},
		// Keywords
		{s: "AND", tok: AND, lit: "AND"},
		{s: "OR", tok: OR, lit: "OR"},
		{s: "NOT", tok: NOT, lit: "NOT"},
		{s: "NoT", tok: NOT, lit: "NoT"},
		// Other tokens
		{s: `(foo`, tok: LPAREN, lit: "("},
		{s: `)`, tok: RPAREN, lit: ")"},
	}
	for i, tt := range tests {
		s := NewLexer(strings.NewReader(tt.s))
		tok, lit := s.Lex()
		if tt.tok != tok {
			t.Errorf("%d. %q token mismatch: exp=%q got=%q <%q>", i, tt.s, tt.tok, tok, lit)
		} else if tt.lit != lit {
			t.Errorf("%d. %q literal mismatch: exp=%q got=%q", i, tt.s, tt.lit, lit)
		}
	}
}

// Test lexing of a token stream
func TestLexer_Stream(t *testing.T) {
	type result struct {
		tok Token
		lit string
	}
	exp := []result{
		{tok: LPAREN, lit: "("},
		{tok: STRING, lit: "GET"},
		{tok: WS, lit: " "},
		{tok: OR, lit: "OR"},
		{tok: WS, lit: " "},
		{tok: STRING, lit: "POST"},
		{tok: RPAREN, lit: ")"},
		{tok: WS, lit: " "},
		{tok: STRING, lit: "apache.status"},
		{tok: COLON, lit: ":"},
		{tok: STRING, lit: "404"},
		{tok: WS, lit: " "},
		{tok: OR, lit: "OR"},
		{tok: WS, lit: " "},
		{tok: STRING, lit: "apache.status"},
		{tok: COLON, lit: ":"},
		{tok: STRING, lit: "500"},
		{tok: EOF, lit: ""},
	}

	// Instantiate a lexer.
	s := `(GET OR POST) apache.status:404 OR apache.status:500`
	l := NewLexer(strings.NewReader(s))

	// Continually scan until we reach the end.
	var act []result
	for {
		tok, lit := l.Lex()
		act = append(act, result{tok, lit})
		if tok == EOF {
			break
		}
	}

	// Check token counts.
	if len(exp) != len(act) {
		t.Fatalf("token count mismatch: exp=%d, got=%d", len(exp), len(act))
	}

	// Check token match.
	for i := range exp {
		if !reflect.DeepEqual(exp[i], act[i]) {
			t.Fatalf("%d. token mismatch:\n\nexp=%#v\n\ngot=%#v", i, exp[i], act[i])
		}
	}
}
