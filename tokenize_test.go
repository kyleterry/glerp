package glerp

import (
	"strings"
	"testing"

	"github.com/matryer/is"
)

func TestTokenization(t *testing.T) {
	tests := []struct {
		description    string
		content        string
		expectedTokens []Token
	}{
		{
			description: "define bools",
			content:     `(define cool #f)`,
			expectedTokens: []Token{
				{
					Kind:  LParen,
					Value: "(",
				},
				{
					Kind:  Define,
					Value: "define",
				},
				{
					Kind:  Symbol,
					Value: "cool",
				},
				{
					Kind:  BFalse,
					Value: "#f",
				},
				{
					Kind:  RParen,
					Value: ")",
				},
				{
					Kind:  EOF,
					Value: "",
				},
			},
		},
		{
			// Regression: end-of-line flush must reset t.current so that a word
			// spanning the last column of a line is not emitted twice.
			description: "multi-line define",
			content:     "(define x\n  5)",
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Define, Value: "define"},
				{Kind: Symbol, Value: "x"},
				{Kind: Number, Value: "5"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "define strings",
			content:     `(define some-str "hello, world")`,
			expectedTokens: []Token{
				{
					Kind:  LParen,
					Value: "(",
				},
				{
					Kind:  Define,
					Value: "define",
				},
				{
					Kind:  Symbol,
					Value: "some-str",
				},
				{
					Kind:  String,
					Value: "hello, world",
				},
				{
					Kind:  RParen,
					Value: ")",
				},
				{
					Kind:  EOF,
					Value: "",
				},
			},
		},
	}

	is := is.New(t)
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			tz := NewTokenizer()
			toks, err := tz.Run(strings.NewReader(test.content))
			is.NoErr(err)

			is.Equal(test.expectedTokens, toks)
		})
	}
}
