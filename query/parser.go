package query

import (
	"fmt"
	"io"
)

// Expr represents an expression.
type Expr interface {
	node()
}

// FieldExpr represents a field expression.
type FieldExpr struct {
	Field string
	Term  string
}

func (f *FieldExpr) node() {}

func (f *FieldExpr) String() string {
	return fmt.Sprintf("%s:%s", f.Field, f.Term)
}

// BinaryExpr represents a binary expression.
type BinaryExpr struct {
	Op  Token
	LHS Expr
	RHS Expr
}

func (b *BinaryExpr) node() {}

func (b *BinaryExpr) String() string {
	return fmt.Sprintf("%s %s %s", b.LHS, tokens[b.Op], b.RHS)
}

// ParenExpr represents a parenthesized expression.
type ParenExpr struct {
	Expr Expr
}

func (*ParenExpr) node() {}

// Statement is an encapsulation of a set of term queries. AND is implicit.
type Statement struct {
	Expressions []*FieldExpr
}

// String returns the string representation of a statement
func (s *Statement) String() string {
	var b string
	for _, e := range s.Expressions {
		b = b + e.String()
	}
	return b
}

// Parser represents a command parser
type Parser struct {
	s   *Lexer
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}

	defaultField string // Search field if none specified.
}

// NewParser returns a new instance of Parser.
func NewParser(r io.Reader, defaultField string) *Parser {
	return &Parser{s: NewLexer(r), defaultField: defaultField}
}

// lex returns the next token from the underlying lexer.
// If a token has been unlexed then read that instead.
func (p *Parser) lex() (tok Token, lit string) {
	// If we have a token on the buffer, then return it.
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	// Otherwise read the next token from the lexer.
	tok, lit = p.s.Lex()

	// Save it to the buffer in case we unlex later.
	p.buf.tok, p.buf.lit = tok, lit

	return
}

// unlex puts the previously read token back onto the buffer.
func (p *Parser) unlex() { p.buf.n = 1 }

// lexIgnoreWhitespace lexes the next non-whitespace token.
func (p *Parser) lexIgnoreWhitespace() (tok Token, lit string) {
	tok, lit = p.lex()
	if tok == WS {
		tok, lit = p.lex()
	}
	return
}

// Parse parses an expression.
func (p *Parser) Parse() (Expr, error) {
	tok, _ := p.lexIgnoreWhitespace()
	if tok == EOF {
		return nil, nil
	}
	p.unlex()

	expr, err := p.parseFieldExpr()
	if err != nil {
		fmt.Println("return err from first call to parse:", err.Error())
		return nil, err
	}

	for {
		op, _ := p.lexIgnoreWhitespace()
		if op == EOF {
			return expr, nil
		} else if op == RPAREN {
			p.unlex()
			return expr, nil
		} else if !op.isOperator() {
			op = AND
			p.unlex()
		}

		rhs, err := p.parseFieldExpr()
		if err != nil {
			return nil, err
		}

		// Assign the new root based on the precedence of the LHS and RHS operators.
		if lhs, ok := expr.(*BinaryExpr); ok && lhs.Op.Precedence() < op.Precedence() {
			expr = &BinaryExpr{
				Op:  lhs.Op,
				LHS: lhs.LHS,
				RHS: &BinaryExpr{LHS: lhs.RHS, RHS: rhs, Op: op},
			}
		} else {
			expr = &BinaryExpr{LHS: expr, RHS: rhs, Op: op}
		}
	}
}

func (p *Parser) parseFieldExpr() (Expr, error) {
	// If the first token is a LPAREN then parse it as its own grouped expression.
	if tok, _ := p.lexIgnoreWhitespace(); tok == LPAREN {
		expr, err := p.Parse()
		if err != nil {
			return nil, err
		}

		// Expect an RPAREN at the end.
		if tok, lit := p.lexIgnoreWhitespace(); tok != RPAREN {
			return nil, fmt.Errorf("found '%s', expected )", tokstr(tok, lit))
		}

		return &ParenExpr{Expr: expr}, nil
	}
	p.unlex()

	tok, f1 := p.lexIgnoreWhitespace()
	if tok != STRING {
		return nil, fmt.Errorf("found '%s', expected FIELD or SEARCH TERM", tokstr(tok, f1))
	}

	tok, _ = p.lexIgnoreWhitespace()
	if tok == COLON {
		tok, f2 := p.lexIgnoreWhitespace()
		if tok != STRING {
			return nil, fmt.Errorf("found '%s', expected SEARCH TERM", tokstr(tok, f2))
		}
		return &FieldExpr{Field: f1, Term: f2}, nil
	}
	p.unlex()
	return &FieldExpr{Field: p.defaultField, Term: f1}, nil
}
