// package token defines the tokens used in the lexer and parser. This file is
// influenced by the Go language's token package.
package token

var (
	keywords   map[string]TokenKind
	delimeters map[string]TokenKind
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
	literal_end

	delimeter_beg
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
	delimeter_end

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
var tokens = [...]string{
	Illegal:   "ILLEGAL",
	EOF:       "EOF",
	Comment:   ";",
	Atom:      "ATOM",
	Symbol:    "SYMBOL",
	Number:    "NUMBER",
	String:    "STRING",
	LParen:    "(",
	LBrack:    "[",
	LBrace:    "{",
	RParen:    ")",
	RBrack:    "]",
	RBrace:    "}",
	Quote:     "'",
	DQuote:    "\"",
	Backtick:  "`",
	Comma:     ",",
	CommaAt:   ",@",
	Car:       "car",
	Cdr:       "cdr",
	Cons:      "cons",
	Empty:     "empty?",
	QuoteLong: "quote",
	Define:    "define",
	If:        "if",
	Lambda:    "lambda",
	Let:       "let",
	LetStar:   "let*",
	SetBang:   "set!",
	BTrue:     "#t",
	BFalse:    "#f",
	Less:      "<",
	Greater:   ">",
	Add:       "+",
	Subtract:  "-",
	Multiply:  "*",
	Divide:    "/",
}

// String returns the value size of the tokens mapping. This value is the
// representation of the token. For non-form tokens, the string returned will
// be the constant name (e.g. String would be "STRING"). For form tokens, the
// string returned is the actual char sequence (e.g. Less would be "<" and Cdr
// would be "cdr").
func (t TokenKind) String() string {
	return tokens[t]
}

func (t TokenKind) IsDelimeter() bool {
	return delimeter_beg < t && t < delimeter_end
}

func (t TokenKind) IsKeyword() bool {
	return keyword_beg < t && t < keyword_end
}

func (t TokenKind) IsLiteral() bool {
	return literal_beg < t && t < literal_end
}

func Lookup(s string) TokenKind {
	if tok, ok := delimeters[s]; ok {
		return tok
	} else if tok, ok := keywords[s]; ok {
		return tok
	}

	if s == Comment.String() {
		return Comment
	}

	return Atom
}

func IsDelimeter(s string) bool {
	_, ok := delimeters[s]

	return ok
}

func IsKeyword(s string) bool {
	_, ok := keywords[s]

	return ok
}

func init() {
	keywords = make(map[string]TokenKind, keyword_end-keyword_beg)
	for i := keyword_beg + 1; i < keyword_end; i++ {
		keywords[tokens[i]] = i
	}

	delimeters = make(map[string]TokenKind, delimeter_end-delimeter_beg)
	for i := delimeter_beg + 1; i < delimeter_end; i++ {
		delimeters[tokens[i]] = i
	}
}
