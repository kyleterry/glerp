// Token types and lookup tables for the lexer and parser. This file is
// influenced by the Go language's token package.
package glerp

var (
	keywords   map[string]TokenKind
	delimiters map[string]TokenKind
)

// Token is a lexical entity containing a component of the scheme language.
type Token struct {
	// Kind contains the token's kind (detailed in the list below).
	Kind TokenKind
	// Value contains the raw value of the token as a string.
	Value string
}

// TokenKind is the set of lexical entities for scheme.
type TokenKind int

const (
	Illegal TokenKind = iota
	EOF
	Comment

	literal_beg
	Atom
	Symbol
	Number
	String
	InterpString
	literal_end

	delimiter_beg
	LParen
	LBrack
	LBrace
	RParen
	RBrack
	RBrace
	Quote
	DQuote
	Backtick
	Comma
	CommaAt
	delimiter_end

	keyword_beg
	Car
	Cdr
	Cons
	QuoteLong
	Empty
	Define
	If
	Lambda
	Let
	LetStar
	SetBang
	BTrue
	BFalse
	Less
	Greater
	Add
	Subtract
	Multiply
	Divide
	keyword_end
)

// This array's order must match the order of the TokenKind constants above.
var tokenNames = [...]string{
	Illegal:      "ILLEGAL",
	EOF:          "EOF",
	Comment:      ";",
	Atom:         "ATOM",
	Symbol:       "SYMBOL",
	Number:       "NUMBER",
	String:       "STRING",
	InterpString: "INTERP_STRING",
	LParen:       "(",
	LBrack:       "[",
	LBrace:       "{",
	RParen:       ")",
	RBrack:       "]",
	RBrace:       "}",
	Quote:        "'",
	DQuote:       "\"",
	Backtick:     "`",
	Comma:        ",",
	CommaAt:      ",@",
	Car:          "car",
	Cdr:          "cdr",
	Cons:         "cons",
	Empty:        "empty?",
	QuoteLong:    "quote",
	Define:       "define",
	If:           "if",
	Lambda:       "lambda",
	Let:          "let",
	LetStar:      "let*",
	SetBang:      "set!",
	BTrue:        "#t",
	BFalse:       "#f",
	Less:         "<",
	Greater:      ">",
	Add:          "+",
	Subtract:     "-",
	Multiply:     "*",
	Divide:       "/",
}

// String returns the string representation of the token kind. For non-form
// tokens, the string returned will be the constant name (e.g. String would be
// "STRING"). For form tokens, the string returned is the actual char sequence
// (e.g. Less would be "<" and Cdr would be "cdr").
func (t TokenKind) String() string {
	return tokenNames[t]
}

func (t TokenKind) IsDelimiter() bool {
	return delimiter_beg < t && t < delimiter_end
}

func (t TokenKind) IsKeyword() bool {
	return keyword_beg < t && t < keyword_end
}

func (t TokenKind) IsLiteral() bool {
	return literal_beg < t && t < literal_end
}

func lookupToken(s string) TokenKind {
	if tok, ok := delimiters[s]; ok {
		return tok
	}

	if tok, ok := keywords[s]; ok {
		return tok
	}

	if s == Comment.String() {
		return Comment
	}

	return Atom
}

func isDelimiter(s string) bool {
	_, ok := delimiters[s]

	return ok
}

func init() {
	keywords = make(map[string]TokenKind, keyword_end-keyword_beg)

	for i := keyword_beg + 1; i < keyword_end; i++ {
		keywords[tokenNames[i]] = i
	}

	delimiters = make(map[string]TokenKind, delimiter_end-delimiter_beg)

	for i := delimiter_beg + 1; i < delimiter_end; i++ {
		delimiters[tokenNames[i]] = i
	}
}
