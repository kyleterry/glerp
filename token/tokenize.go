package token

import (
	"bufio"
	"io"
	"unicode"
)

type mode int

const (
	normal mode = iota
	quoted
)

// Tokenizer can read from a reader and split the content up into Tokens. It
// holds the mode state so that tokens that require more than 1 character can
// be tokenized. Currently is supports two modes: normal and quoted. Normal
// mode can read and tokenize all delimeters except string quotes, and all
// atoms except strings. Quoted mode is used to keep track of whether or not we
// are reading a string. Quoted mode is started when the tokenizer sees a " and
// is flipped back to normal mode when it sees another ".
type Tokenizer struct {
	toks    []Token
	current string
	mode    mode
}

// Run takes an io.Reader and returns a slice of Tokens. The content read from
// r must be a scheme expression, but this is not a parser, so incorrect scheme
// can still be successfully tokenized.
func (t *Tokenizer) Run(r io.Reader) ([]Token, error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		t.toks = append(t.toks, t.tokenizeLine(line)...)
	}

	for i, tok := range t.toks {
		if tok.Kind == Atom {
			t.toks[i] = t.tokenizeAtom(tok.Value)
		}
	}

	t.toks = append(t.toks, Token{Kind: EOF})

	return t.toks, scanner.Err()
}

func (t *Tokenizer) tokenizeLine(line string) []Token {
	var tokens []Token

	for i, r := range line {
		if t.mode == quoted {
			if r != '"' {
				t.current += string(r)
			}

			if r == '"' {
				tokens = append(tokens, Token{
					Kind:  String,
					Value: t.current,
				})

				t.current = ""
				t.mode = normal
			}

			continue
		}

		switch {
		case unicode.IsSpace(r):
			if t.current != "" {
				tokens = append(tokens, Token{
					Kind:  Lookup(t.current),
					Value: t.current,
				})
				t.current = ""
			}
		case r == '"':
			t.mode = quoted
		case IsDelimeter(string(r)):
			if t.current != "" {
				tokens = append(tokens, Token{
					Kind:  Lookup(t.current),
					Value: t.current,
				})
				t.current = ""
			}

			tokens = append(tokens, Token{
				Kind:  Lookup(string(r)),
				Value: string(r),
			})
		case r == ';':
			tokens = append(tokens, Token{
				Kind:  Comment,
				Value: line[i:],
			})

			return tokens
		default:
			t.current += string(r)

			if i == len(line)-1 {
				tokens = append(tokens, Token{
					Kind:  Lookup(t.current),
					Value: t.current,
				})
				t.current = ""
			}
		}
	}

	return tokens
}

func (t *Tokenizer) tokenizeAtom(atom string) Token {
	if unicode.IsDigit(rune(atom[0])) {
		return Token{Kind: Number, Value: atom}
	}
	// Negative numeric literal: '-' followed by a digit (e.g. -5, -3.14).
	if len(atom) > 1 && atom[0] == '-' && unicode.IsDigit(rune(atom[1])) {
		return Token{Kind: Number, Value: atom}
	}
	return Token{Kind: Symbol, Value: atom}
}

// NewTokenizer returns a new Tokenizer with its mode set to normal.
func NewTokenizer() *Tokenizer {
	return &Tokenizer{mode: normal}
}
