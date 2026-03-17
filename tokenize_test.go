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
			description: "define boolean",
			content:     `(define cool #f)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Define, Value: "define"},
				{Kind: Symbol, Value: "cool"},
				{Kind: BFalse, Value: "#f"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "true boolean",
			content:     `#t`,
			expectedTokens: []Token{
				{Kind: BTrue, Value: "#t"},
				{Kind: EOF, Value: ""},
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
			description: "define string",
			content:     `(define some-str "hello, world")`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Define, Value: "define"},
				{Kind: Symbol, Value: "some-str"},
				{Kind: String, Value: "hello, world"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "integer literal",
			content:     `42`,
			expectedTokens: []Token{
				{Kind: Number, Value: "42"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "float literal",
			content:     `3.14`,
			expectedTokens: []Token{
				{Kind: Number, Value: "3.14"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "negative integer",
			content:     `(+ -5 3)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Add, Value: "+"},
				{Kind: Number, Value: "-5"},
				{Kind: Number, Value: "3"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "negative float",
			content:     `-3.14`,
			expectedTokens: []Token{
				{Kind: Number, Value: "-3.14"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "arithmetic operators",
			content:     `(+ 1 (* 2 (/ 6 (- 4 1))))`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Add, Value: "+"},
				{Kind: Number, Value: "1"},
				{Kind: LParen, Value: "("},
				{Kind: Multiply, Value: "*"},
				{Kind: Number, Value: "2"},
				{Kind: LParen, Value: "("},
				{Kind: Divide, Value: "/"},
				{Kind: Number, Value: "6"},
				{Kind: LParen, Value: "("},
				{Kind: Subtract, Value: "-"},
				{Kind: Number, Value: "4"},
				{Kind: Number, Value: "1"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "comparison operators",
			content:     `(< 1 2)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Less, Value: "<"},
				{Kind: Number, Value: "1"},
				{Kind: Number, Value: "2"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "square brackets",
			content:     `(let [(x 1) (y 2)] (+ x y))`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Let, Value: "let"},
				{Kind: LBrack, Value: "["},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "x"},
				{Kind: Number, Value: "1"},
				{Kind: RParen, Value: ")"},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "y"},
				{Kind: Number, Value: "2"},
				{Kind: RParen, Value: ")"},
				{Kind: RBrack, Value: "]"},
				{Kind: LParen, Value: "("},
				{Kind: Add, Value: "+"},
				{Kind: Symbol, Value: "x"},
				{Kind: Symbol, Value: "y"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "quote shorthand",
			content:     `'(1 2 3)`,
			expectedTokens: []Token{
				{Kind: Quote, Value: "'"},
				{Kind: LParen, Value: "("},
				{Kind: Number, Value: "1"},
				{Kind: Number, Value: "2"},
				{Kind: Number, Value: "3"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "backtick quasiquote",
			content:     "`(a ,x ,@xs)",
			expectedTokens: []Token{
				{Kind: Backtick, Value: "`"},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "a"},
				{Kind: Comma, Value: ","},
				{Kind: Symbol, Value: "x"},
				{Kind: CommaAt, Value: ",@"},
				{Kind: Symbol, Value: "xs"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "comma at end of atom",
			content:     `(a,b)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "a"},
				{Kind: Comma, Value: ","},
				{Kind: Symbol, Value: "b"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "line comment",
			content:     `(+ 1 2) ; add numbers`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Add, Value: "+"},
				{Kind: Number, Value: "1"},
				{Kind: Number, Value: "2"},
				{Kind: RParen, Value: ")"},
				{Kind: Comment, Value: "; add numbers"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "comment discards rest of line",
			content:     "(+ 1 ; ignore\n  2)",
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Add, Value: "+"},
				{Kind: Number, Value: "1"},
				{Kind: Comment, Value: "; ignore"},
				{Kind: Number, Value: "2"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "keywords",
			content:     `(define (f x) (if (empty? x) (car x) (cdr x)))`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Define, Value: "define"},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "f"},
				{Kind: Symbol, Value: "x"},
				{Kind: RParen, Value: ")"},
				{Kind: LParen, Value: "("},
				{Kind: If, Value: "if"},
				{Kind: LParen, Value: "("},
				{Kind: Empty, Value: "empty?"},
				{Kind: Symbol, Value: "x"},
				{Kind: RParen, Value: ")"},
				{Kind: LParen, Value: "("},
				{Kind: Car, Value: "car"},
				{Kind: Symbol, Value: "x"},
				{Kind: RParen, Value: ")"},
				{Kind: LParen, Value: "("},
				{Kind: Cdr, Value: "cdr"},
				{Kind: Symbol, Value: "x"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "lambda and set!",
			content:     `(lambda (x) (set! x 1))`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Lambda, Value: "lambda"},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "x"},
				{Kind: RParen, Value: ")"},
				{Kind: LParen, Value: "("},
				{Kind: SetBang, Value: "set!"},
				{Kind: Symbol, Value: "x"},
				{Kind: Number, Value: "1"},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "let*",
			content:     `(let* [(a 1)] a)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: LetStar, Value: "let*"},
				{Kind: LBrack, Value: "["},
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: "a"},
				{Kind: Number, Value: "1"},
				{Kind: RParen, Value: ")"},
				{Kind: RBrack, Value: "]"},
				{Kind: Symbol, Value: "a"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "interpolated string simple",
			content:     `$"hello {name}!"`,
			expectedTokens: []Token{
				{Kind: InterpString, Value: "hello {name}!"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "interpolated string with expression",
			content:     `$"result: {(+ 1 2)}"`,
			expectedTokens: []Token{
				{Kind: InterpString, Value: "result: {(+ 1 2)}"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "interpolated string with nested string",
			content:     `$"say {"hi"}"`,
			expectedTokens: []Token{
				{Kind: InterpString, Value: `say {"hi"}`},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "interpolated string no interpolation",
			content:     `$"plain text"`,
			expectedTokens: []Token{
				{Kind: InterpString, Value: "plain text"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "cons keyword",
			content:     `(cons 1 '())`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Cons, Value: "cons"},
				{Kind: Number, Value: "1"},
				{Kind: Quote, Value: "'"},
				{Kind: LParen, Value: "("},
				{Kind: RParen, Value: ")"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "greater than and equal symbols",
			content:     `(>= x 0)`,
			expectedTokens: []Token{
				{Kind: LParen, Value: "("},
				{Kind: Symbol, Value: ">="},
				{Kind: Symbol, Value: "x"},
				{Kind: Number, Value: "0"},
				{Kind: RParen, Value: ")"},
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "empty input",
			content:     ``,
			expectedTokens: []Token{
				{Kind: EOF, Value: ""},
			},
		},
		{
			description: "whitespace only",
			content:     `   `,
			expectedTokens: []Token{
				{Kind: EOF, Value: ""},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			is := is.New(t)
			tz := NewTokenizer()
			toks, err := tz.Run(strings.NewReader(test.content))
			is.NoErr(err)
			is.Equal(test.expectedTokens, toks)
		})
	}
}
