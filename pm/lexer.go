//go:generate goyacc -l -o parser.go parser.y
package pm

import (
	"bytes"
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/charmbracelet/log"
)

// Parse parses the input and returs the result.
func Parse(input []byte) (map[string]interface{}, error) {
	l := newLex(input)
	_ = yyParse(l)
	return l.result, l.err
}

type lex struct {
	input  []byte
	pos    int
	result map[string]interface{}
	err    error
}

func newLex(input []byte) *lex {
	return &lex{
		input: input,
	}
}

// Lex satisfies yyLexer.
func (l *lex) Lex(lval *yySymType) int {
	token := l.scanNormal(lval)
	log.Debugf("Lex: returning token %s (%d)\n", yyTokname(token), token)
	return token
}

func (l *lex) scanNormal(lval *yySymType) int {
	for b := l.next(); b != 0; b = l.next() {
		log.Debugf("scanNormal: processing character %s (%d) @ (%d)\n", string(b), b, l.pos)
		switch {
		case unicode.IsSpace(rune(b)) || b == '\n' || b == '\r':
			continue
		case b == '/' && l.peek() == '/': // Check for the beginning of a comment
			for b := l.next(); b != '\n' && b != 0; b = l.next() {
				// Skip until end of line or end of input
			}
			continue // Continue scanning after skipping the comment
		case b == '"':
			return l.scanString(lval)
		case unicode.IsDigit(rune(b)) || b == '+' || b == '-':
			l.backup()
			return l.scanNum(lval)
		case unicode.IsLetter(rune(b)):
			l.backup()
			return l.scanLiteral(lval)
		case b == '.' && unicode.IsLetter(rune(b)) || rune(b) == '_':
			l.backup()
			return l.scanKey(lval)
		default:
			return int(b)
		}
	}
	return 0
}

var escape = map[byte]byte{
	'"':  '"',
	'\\': '\\',
	'/':  '/',
	'b':  '\b',
	'f':  '\f',
	'n':  '\n',
	'r':  '\r',
	't':  '\t',
}

func (l *lex) scanString(lval *yySymType) int {
	buf := bytes.NewBuffer(nil)
	for b := l.next(); b != 0; b = l.next() {
		switch b {
		case '\\':
			// TODO(sougou): handle \uxxxx construct.
			b2 := escape[l.next()]
			if b2 == 0 {
				return LexError
			}
			buf.WriteByte(b2)
		case '"':
			lval.val = buf.String()
			return String
		default:
			buf.WriteByte(b)
		}
	}
	return LexError
}

func (l *lex) scanNum(lval *yySymType) int {
	buf := bytes.NewBuffer(nil)
	for {
		b := l.next()
		switch {
		case unicode.IsDigit(rune(b)):
			buf.WriteByte(b)
		case strings.IndexByte(".+-eE", b) != -1:
			buf.WriteByte(b)
		default:
			l.backup()
			val, err := strconv.ParseFloat(buf.String(), 64)
			if err != nil {
				return LexError
			}
			lval.val = val
			return Number
		}
	}
}

var literal = map[string]interface{}{
	"true":  true,
	"false": false,
	"null":  nil,
}

func (l *lex) scanLiteral(lval *yySymType) int {
	buf := bytes.NewBuffer(nil)
	for {
		b := l.next()
		switch {
		case unicode.IsLetter(rune(b)):
			buf.WriteByte(b)
		default:
			l.backup()
			val, ok := literal[buf.String()]
			if !ok {
				return LexError
			}
			lval.val = val
			return Literal
		}
	}
}

func (l *lex) backup() {
	if l.pos == -1 {
		return
	}
	l.pos--
}

// peek advances the iterator without consuming the present value
func (l *lex) peek() byte {
	if l.pos >= len(l.input) || l.pos == -1 {
		l.pos = -1
		return 0
	}

	return l.input[l.pos]
}

// next advances the iterator and consumes the present value
func (l *lex) next() byte {
	if l.pos >= len(l.input) || l.pos == -1 {
		l.pos = -1
		return 0
	}
	l.pos++
	return l.input[l.pos-1]
}

// Error satisfies yyLexer.
func (l *lex) Error(s string) {
	l.err = errors.New(s)
}

func (l *lex) scanKey(lval *yySymType) int {
	buf := bytes.NewBuffer(nil)
	for {
		b := l.next()
		switch {
		case unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b)) || b == '_':
			buf.WriteByte(b)
		default:
			l.backup()
			str := buf.String()
			lval.val = str
			return UnquotedKey
		}
	}
}
