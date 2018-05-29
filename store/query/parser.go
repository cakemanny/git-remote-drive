// This is practically copied verbatim from:
//   https://blog.gopheracademy.com/advent-2014/parsers-lexers/
// found after having read:
//   http://blog.leahhanson.us/post/recursecenter2016/recipeparser.html
package query

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type Token int

const (
	ILLEGAL Token = iota
	EOF
	WS

	// data
	beginDataTokens
	IDENT
	STRING
	TRUE
	FALSE
	endDataTokens

	// Operators
	beginOperatorTokens
	EQUALS
	IN       // list contains equal item
	CONTAINS // text contains substring
	endOperatorTokens
	// The actual language has a lot more operators, but we don't need them
	// for our tests, so they are omitted

	// Keywords
	AND
	OR
	NOT
)

var eof = rune(0)

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch rune) bool {
	return (ch >= '0' && ch <= '9')
}

type Scanner struct {
	r *bufio.Reader
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: bufio.NewReader(r)}
}

func (s *Scanner) read() rune {
	ch, _, err := s.r.ReadRune()
	if err != nil {
		return eof
	}
	return ch
}

func (s *Scanner) unread() { _ = s.r.UnreadRune() }

func (s *Scanner) Scan() (tok Token, lit string) {
	ch := s.read()

	if isWhitespace(ch) {
		s.unread()
		return s.scanWhitespace()
	} else if isLetter(ch) {
		s.unread()
		return s.scanIdent()
	} else if ch == '\'' {
		s.unread()
		return s.scanString()
	}

	switch ch {
	case eof:
		return EOF, ""
	case '=':
		return EQUALS, string(ch)
	}

	return ILLEGAL, string(ch)
}

func (s *Scanner) scanWhitespace() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

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

func (s *Scanner) scanIdent() (tok Token, lit string) {
	var buf bytes.Buffer
	buf.WriteRune(s.read())

	for {
		if ch := s.read(); ch == eof {
			break
		} else if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			s.unread()
			break
		} else {
			buf.WriteRune(ch)
		}
	}

	switch strings.ToLower(buf.String()) {
	case "in":
		return IN, buf.String()
	case "and":
		return AND, buf.String()
	case "or":
		return OR, buf.String()
	case "not":
		return NOT, buf.String()
	case "true":
		return TRUE, buf.String()
	case "false":
		return FALSE, buf.String()
	}
	return IDENT, buf.String()
}

func (s *Scanner) scanString() (tok Token, lit string) {
	var buf bytes.Buffer
	// skip '
	strStart := s.read()

	for {
		if ch := s.read(); ch == eof {
			// should be an error
			break
		} else if ch == '\\' {
			ch = s.read()
			switch ch {
			case 'n':
				buf.WriteRune('\n')
			case 't':
				buf.WriteRune('\t')
			case '\'':
				buf.WriteRune('\'')
			case '"':
				buf.WriteRune('"')
			case '\\':
				buf.WriteRune('\\')
			default:
				buf.WriteRune('\\')
				buf.WriteRune(ch)
			}
		} else if ch == strStart {
			break
		} else {
			buf.WriteRune(ch)
		}
	}
	return STRING, buf.String()
}

type Parser struct {
	s   *Scanner
	buf struct {
		tok Token  // last read token
		lit string // last read literal
		n   int    // buffer size (max=1)
	}
}

func NewParser(r io.Reader) *Parser {
	return &Parser{s: NewScanner(r)}
}

func (p *Parser) scan() (tok Token, lit string) {
	if p.buf.n != 0 {
		p.buf.n = 0
		return p.buf.tok, p.buf.lit
	}

	tok, lit = p.s.Scan()

	p.buf.tok, p.buf.lit = tok, lit
	return
}

func (p *Parser) unscan() { p.buf.n = 1 }

// scanIgnoreWhitespace
func (p *Parser) nextTok() (tok Token, lit string) {
	tok, lit = p.scan()
	if tok == WS {
		tok, lit = p.scan()
	}
	return
}

type Expr struct {
	Ands []And
}
type And struct {
	Tests []Test
}
type Test struct {
	Lhs Datum
	Op  Token // IN, EQUALS
	Rhs Datum
}
type Datum struct {
	Type Token  // STRING, IDENT, TRUE, FALSE, NUMBER
	Lit  string // true, false
}

func ExprEqual(e1, e2 Expr) bool {
	if len(e1.Ands) != len(e2.Ands) {
		return false
	}
	for i, and := range e1.Ands {
		if !AndEqual(and, e2.Ands[i]) {
			return false
		}
	}
	return true
}
func AndEqual(a1, a2 And) bool {
	if len(a1.Tests) != len(a2.Tests) {
		return false
	}
	for i, test := range a1.Tests {
		if test != a2.Tests[i] {
			return false
		}
	}
	return true
}

/*

expr ::= conj (OR conj)*
conj ::= test (AND test)*
test ::= datum operator datum
datum ::= ident | string | bool
operator ::= 'in' | '='

*/

func (p *Parser) Parse() (*Expr, error) {

	conjunctions := []And{}
	conj, err := p.ParseConjuction()
	if err != nil {
		return nil, err
	}
	conjunctions = append(conjunctions, conj)

	for {
		tok, lit := p.nextTok()
		if tok == ILLEGAL || tok == EOF {
			break
		}
		if tok != OR {
			return nil, fmt.Errorf(
				"Error parsing expression\n  Expected: or\n  Found: %v\n", lit)
		}

		conj, err := p.ParseConjuction()
		if err != nil {
			return nil, err
		}
		conjunctions = append(conjunctions, conj)
	}
	return &Expr{conjunctions}, nil
}

func (p *Parser) ParseConjuction() (And, error) {

	tests := []Test{}
	test, err := p.ParseTest()
	if err != nil {
		return And{}, err
	}
	tests = append(tests, test)

	for {
		tok, lit := p.nextTok()
		if tok == ILLEGAL || tok == EOF {
			break
		}
		if tok != AND {
			return And{}, fmt.Errorf(
				"Error parsing conjuction\n  Expected: and\n  Found: %v\n", lit)
		}

		test, err := p.ParseTest()
		if err != nil {
			return And{}, err
		}
		tests = append(tests, test)
	}

	return And{tests}, nil
}

func (p *Parser) ParseTest() (Test, error) {

	lhs, err := p.ParseDatum()
	if err != nil {
		return Test{}, err
	}
	op, err := p.ParseOperator()
	if err != nil {
		return Test{}, err
	}
	rhs, err := p.ParseDatum()
	if err != nil {
		return Test{}, err
	}

	return Test{
		Lhs: lhs,
		Op:  op,
		Rhs: rhs,
	}, nil
}

func (p *Parser) ParseDatum() (Datum, error) {
	tok, lit := p.nextTok()

	if tok < beginDataTokens || tok > endDataTokens {
		return Datum{}, fmt.Errorf(
			"Error parsing datum\n"+
				"  Expected: <string> or <ident> or 'true' or 'false'\n"+
				"  Found: %v\n", lit)
	}

	return Datum{tok, lit}, nil
}
func (p *Parser) ParseOperator() (tok Token, err error) {
	tok, lit := p.nextTok()
	if tok < beginOperatorTokens || tok > endOperatorTokens {
		err = fmt.Errorf(
			"Error parsing operator\n"+
				"  Expected: 'in', 'contains' or '='\n"+
				"  Found: %v\n",
			lit,
		)
	}
	return
}
