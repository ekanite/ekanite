package query

import (
	"bufio"
	"bytes"
	"io"
)

var eof = rune(0)

// Lexer represents a lexer.
type Lexer struct {
	r *bufio.Reader
}

// NewLexer returns a new instance of a Lexer.
func NewLexer(r io.Reader) *Lexer {
	return &Lexer{r: bufio.NewReader(r)}
}

// read reads the next rune from the bufferred reader.
// Returns the query.eofif an error occurs.
func (s *Lexer) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

// unread puts the previously read rune on the buffer.
func (s *Lexer) unread() { _ = s.r.UnreadRune() }

// Lex returns the next token and associated literal value.
func (s *Lexer) Lex() (tok Token, lit string) {
	ch := s.read()

	// If whitespace, then consume it and all following whitespace.
	// A letter means an IDENT or reserved word.
	if isWhitespace(ch) {
		s.unread()
		return s.lexWhitespace()
	} else if ch == eof {
		return EOF, ""
	} else if ch == '(' {
		return LPAREN, "("
	} else if ch == ')' {
		return RPAREN, ")"
	} else if ch == ':' {
		return COLON, ":"
	} else {
		s.unread()
		tok, lit := s.lexString()

		// Check for keyword match.
		if kw, ok := Lookup(lit); ok {
			return kw, lit
		}
		return tok, lit
	}

	return ILLEGAL, string(ch)
}

// lexWhitespace consumes the current rune and all contiguous whitespace.
func (s *Lexer) lexWhitespace() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent whitespace character into the buffer.
	// Non-whitespace characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isWhitespace(ch) {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	return WS, buf.String()
}

// lexIdent consumes the current rune and all contiguous String runes.
func (s *Lexer) lexString() (tok Token, lit string) {
	// Create a buffer and read the current character into it.
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	// Read every subsequent string character into the buffer.
	// Non-string characters and EOF will cause the loop to exit.
	for {
		if ch := s.read(); ch == eof {
			break
		} else if ch == ':' || isWhitespace(ch) || isParen(ch) {
			// end of String lex
			s.unread()
			break
		} else {
			_, _ = buf.WriteRune(ch)
		}
	}

	// Return as regular string.
	return STRING, buf.String()
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isParen(ch rune) bool {
	return ch == '(' || ch == ')'
}
