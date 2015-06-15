package query

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Ensure the parser can parse an empty query.
func TestParser_ParseQuery_Empty(t *testing.T) {
	expr, err := NewParser(strings.NewReader(``), "defField").Parse()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if expr != nil {
		t.Fatalf("expected nil expr, got %v", expr)
	}
}

// Ensure the parser can parse strings into Query ASTs.
// XXX Still missing tests for parenthesized queries.
func TestParser_ParseStatement(t *testing.T) {
	defaultField := "defField"

	var tests = []struct {
		s    string
		expr Expr
		err  string
	}{
		{
			s:    `sshd`,
			expr: &FieldExpr{Field: defaultField, Term: "sshd"},
		},
		{
			s: `sshd pamd`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "sshd"},
				RHS: &FieldExpr{Field: defaultField, Term: "pamd"},
			},
		},
		{
			s: `sshd AND pamd`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "sshd"},
				RHS: &FieldExpr{Field: defaultField, Term: "pamd"},
			},
		},
		{
			s: `sshd and pamd`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "sshd"},
				RHS: &FieldExpr{Field: defaultField, Term: "pamd"},
			},
		},
		{
			s: `sshd OR pamd`,
			expr: &BinaryExpr{
				Op:  OR,
				LHS: &FieldExpr{Field: defaultField, Term: "sshd"},
				RHS: &FieldExpr{Field: defaultField, Term: "pamd"},
			},
		},
		{
			s: `GET apache.status:404`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "GET"},
				RHS: &FieldExpr{Field: "apache.status", Term: "404"},
			},
		},
		{
			s:    `sourceip:192.168.1.22`,
			expr: &FieldExpr{Field: "sourceip", Term: "192.168.1.22"},
		},
		{
			s: `GET AND apache.status:404 OR apache.status:500`,
			expr: &BinaryExpr{
				Op: OR,
				LHS: &BinaryExpr{
					Op:  AND,
					LHS: &FieldExpr{Field: defaultField, Term: "GET"},
					RHS: &FieldExpr{Field: "apache.status", Term: "404"},
				},
				RHS: &FieldExpr{Field: "apache.status", Term: "500"},
			},
		},
		{
			s: `GET AND (apache.status:404 OR apache.status:500)`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "GET"},
				RHS: &ParenExpr{
					Expr: &BinaryExpr{
						Op:  OR,
						LHS: &FieldExpr{Field: "apache.status", Term: "404"},
						RHS: &FieldExpr{Field: "apache.status", Term: "500"},
					},
				},
			},
		},
		{
			s: `GET (apache.status:404 OR apache.status:500)`,
			expr: &BinaryExpr{
				Op:  AND,
				LHS: &FieldExpr{Field: defaultField, Term: "GET"},
				RHS: &ParenExpr{
					Expr: &BinaryExpr{
						Op:  OR,
						LHS: &FieldExpr{Field: "apache.status", Term: "404"},
						RHS: &FieldExpr{Field: "apache.status", Term: "500"},
					},
				},
			},
		},

		// Errors
		{s: `apache.status:`, err: `found 'EOF', expected SEARCH TERM`},
		{s: `GET AND`, err: `found 'EOF', expected FIELD or SEARCH TERM`},
		{s: `GET AND NOT`, err: `found 'NOT', expected FIELD or SEARCH TERM`},
		{s: `:500`, err: `found ':', expected FIELD or SEARCH TERM`},
		{s: `GET (apache.status:404 OR apache.status:500`, err: `found 'EOF', expected )`},
		{s: `GET (apache.status:404 OR apache.status:`, err: `found 'EOF', expected SEARCH TERM`},
	}

	for i, tt := range tests {
		fmt.Println("testing:", tt.s)
		expr, err := NewParser(strings.NewReader(tt.s), defaultField).Parse()
		if !reflect.DeepEqual(tt.err, errstring(err)) {
			t.Errorf("%d. %q: error mismatch:\n  exp=%s\n  got=%s\n\n", i, tt.s, tt.err, err)
		} else if tt.err == "" && !reflect.DeepEqual(tt.expr, expr) {
			t.Errorf("%d. %q\n\nstmt mismatch:\n\nexp=%#v %s\n\ngot=%#v %s\n\n", i, tt.s, tt.expr, tt.expr, expr, expr)
		}
	}
}

// errstring returns the string representation of an error.
func errstring(err error) string {
	if err != nil {
		return err.Error()
	}
	return ""
}
