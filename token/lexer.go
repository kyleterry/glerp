package token

import (
	"fmt"
	"io"
)

type Lexer struct {
	tokens []Token
}

func (l *Lexer) PeekToken() Token {
	if len(l.tokens) > 0 {
		return l.tokens[0]
	}

	return Token{Kind: EOF, Value: ""}
}

func (l *Lexer) NextToken() Token {
	tok := l.PeekToken()

	if tok.Kind != EOF {
		l.tokens = l.tokens[1:]
	}

	return tok
}

func NewLexer(r io.Reader) (*Lexer, error) {
	tzr := NewTokenizer()
	toks, err := tzr.Run(r)
	if err != nil {
		return nil, fmt.Errorf("tokenizer error: %w", err)
	}

	return &Lexer{tokens: toks}, nil
}
