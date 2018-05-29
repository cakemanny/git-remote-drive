package query

import "strings"
import "testing"

func TestParse(t *testing.T) {

	r := strings.NewReader("name = 'dan' and 'root' in parents and trashed = false")
	parser := NewParser(r)

	expr, err := parser.Parse()
	if err != nil {
		t.Error(`Error parsing:"name = 'dan' and 'root' in parents and trashed = false"`)
	}

	expectedExpr := &Expr{
		Ands: []And{
			And{
				Tests: []Test{
					Test{
						Lhs: Datum{IDENT, "name"},
						Op:  EQUALS,
						Rhs: Datum{STRING, "dan"},
					},
					Test{
						Lhs: Datum{STRING, "root"},
						Op:  IN,
						Rhs: Datum{IDENT, "parents"},
					},
					Test{
						Lhs: Datum{IDENT, "trashed"},
						Op:  EQUALS,
						Rhs: Datum{FALSE, "false"},
					},
				},
			},
		},
	}

	if !ExprEqual(*expr, *expectedExpr) {
		t.Error(
			"\nExpected:",
			*expectedExpr,
			"\n  Actual:",
			*expr,
		)
	}
}
