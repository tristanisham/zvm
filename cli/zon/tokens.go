package zon

type token int

const (
	tkLeftBrace token = iota
	tkRightBrace
	tkPeriod
	tkEqual
	tkComma
	tkQuote
	tkLexeme
	tkBackslash
	tkUnderscore
	tkColon
	tkForwardSlash
)

// tc is a Token Container that represents state for parsed tokens.
type tc struct {
	lexeme string
	token
	depth int
}
