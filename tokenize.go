package glerp

import (
	"bufio"
	"io"
	"unicode"
)

type tokenizerMode int

const (
	modeNormal tokenizerMode = iota
	modeQuoted
	modeInterpString // inside $"...", outside a {} block
	modeInterpExpr   // inside {} within an interpolated string
	modeInterpQuoted // inside a "" string literal within a {} block
)

// Tokenizer can read from a reader and split the content up into Tokens. It
// holds the mode state so that tokens that require more than 1 character can
// be tokenized. Currently is supports two modes: normal and quoted. Normal
// mode can read and tokenize all delimeters except string quotes, and all
// atoms except strings. Quoted mode is used to keep track of whether or not we
// are reading a string. Quoted mode is started when the tokenizer sees a " and
// is flipped back to normal mode when it sees another ".
type Tokenizer struct {
	toks       []Token
	current    string
	mode       tokenizerMode
	braceDepth int
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
	var toks []Token
	skip := false

	for i, r := range line {
		if skip {
			skip = false
			continue
		}

		if t.mode == modeQuoted {
			if r != '"' {
				t.current += string(r)
			}

			if r == '"' {
				toks = append(toks, Token{
					Kind:  String,
					Value: t.current,
				})

				t.current = ""
				t.mode = modeNormal
			}

			continue
		}

		if t.mode == modeInterpString {
			switch r {
			case '"':
				toks = append(toks, Token{Kind: InterpString, Value: t.current})
				t.current = ""
				t.mode = modeNormal
			case '{':
				t.current += "{"
				t.mode = modeInterpExpr
				t.braceDepth = 1
			default:
				t.current += string(r)
			}
			continue
		}

		if t.mode == modeInterpExpr {
			switch r {
			case '{':
				t.braceDepth++
				t.current += "{"
			case '}':
				t.braceDepth--
				t.current += "}"
				if t.braceDepth == 0 {
					t.mode = modeInterpString
				}
			case '"':
				t.current += "\""
				t.mode = modeInterpQuoted
			default:
				t.current += string(r)
			}
			continue
		}

		if t.mode == modeInterpQuoted {
			t.current += string(r)
			if r == '"' {
				t.mode = modeInterpExpr
			}
			continue
		}

		switch {
		case unicode.IsSpace(r):
			if t.current != "" {
				toks = append(toks, Token{
					Kind:  lookupToken(t.current),
					Value: t.current,
				})
				t.current = ""
			}
		case r == '"':
			if t.current == "$" {
				t.current = ""
				t.mode = modeInterpString
			} else {
				t.mode = modeQuoted
			}
		case r == ',':
			if t.current != "" {
				toks = append(toks, Token{
					Kind:  lookupToken(t.current),
					Value: t.current,
				})
				t.current = ""
			}
			if i+1 < len(line) && line[i+1] == '@' {
				toks = append(toks, Token{Kind: CommaAt, Value: ",@"})
				skip = true
			} else {
				toks = append(toks, Token{Kind: Comma, Value: ","})
			}
		case isDelimiter(string(r)):
			if t.current != "" {
				toks = append(toks, Token{
					Kind:  lookupToken(t.current),
					Value: t.current,
				})
				t.current = ""
			}

			toks = append(toks, Token{
				Kind:  lookupToken(string(r)),
				Value: string(r),
			})
		case r == ';':
			toks = append(toks, Token{
				Kind:  Comment,
				Value: line[i:],
			})

			return toks
		default:
			t.current += string(r)

			if i == len(line)-1 {
				toks = append(toks, Token{
					Kind:  lookupToken(t.current),
					Value: t.current,
				})
				t.current = ""
			}
		}
	}

	return toks
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
	return &Tokenizer{mode: modeNormal}
}
