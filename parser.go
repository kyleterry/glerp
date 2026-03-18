package glerp

import (
	"fmt"
	"strconv"
	"strings"
)

// Parser converts a token stream into a slice of Expr values ready for
// evaluation. It is a recursive-descent parser for s-expressions.
type Parser struct {
	lexer *Lexer
}

// Run parses all top-level expressions from the token stream.
func (p *Parser) Run() ([]Expr, error) {
	var exprs []Expr

	for p.lexer.PeekToken().Kind != EOF {
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}

	return exprs, nil
}

func (p *Parser) parseExpr() (Expr, error) {
	tok := p.lexer.PeekToken()
	switch tok.Kind {
	case LParen, LBrack:
		return p.parseList()

	case HashLParen:
		return p.parseVector()

	case Quote:
		// Desugar 'expr → (quote expr)
		p.lexer.NextToken()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		quoteSym := &SymbolExpr{
			tok: Token{Kind: QuoteLong, Value: "quote"},
			val: "quote",
		}
		return &ListExpr{tok: tok, elements: []Expr{quoteSym, inner}}, nil

	case Backtick:
		// Desugar `expr → (quasiquote expr)
		p.lexer.NextToken()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "quasiquote"}, val: "quasiquote"}
		return &ListExpr{tok: tok, elements: []Expr{sym, inner}}, nil

	case Comma:
		// Desugar ,expr → (unquote expr)
		p.lexer.NextToken()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "unquote"}, val: "unquote"}
		return &ListExpr{tok: tok, elements: []Expr{sym, inner}}, nil

	case CommaAt:
		// Desugar ,@expr → (unquote-splicing expr)
		p.lexer.NextToken()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "unquote-splicing"}, val: "unquote-splicing"}
		return &ListExpr{tok: tok, elements: []Expr{sym, inner}}, nil

	case Number:
		p.lexer.NextToken()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", tok.Value, err)
		}
		return &NumberExpr{tok: tok, val: v}, nil

	case String:
		p.lexer.NextToken()
		return &StringExpr{tok: tok, val: tok.Value}, nil

	case InterpString:
		p.lexer.NextToken()
		return parseInterpString(tok)

	case BTrue:
		p.lexer.NextToken()
		return &BoolExpr{tok: tok, val: true}, nil

	case BFalse:
		p.lexer.NextToken()
		return &BoolExpr{tok: tok, val: false}, nil

	case Comment:
		// Skip comment tokens and parse the next expression.
		p.lexer.NextToken()
		return p.parseExpr()

	case EOF:
		return nil, fmt.Errorf("unexpected EOF")

	default:
		// Keywords (define, lambda, if, car, …) and plain symbols are all
		// represented as SymbolExpr; special-form dispatch happens at eval time.
		if tok.Kind.IsKeyword() || tok.Kind == Symbol {
			p.lexer.NextToken()
			return &SymbolExpr{tok: tok, val: tok.Value}, nil
		}
		return nil, fmt.Errorf("unexpected token: %s (%q)", tok.Kind, tok.Value)
	}
}

func (p *Parser) parseList() (Expr, error) {
	open := p.lexer.NextToken() // consume '(' or '['
	close := RParen
	if open.Kind == LBrack {
		close = RBrack
	}

	var elements []Expr

	for {
		peek := p.lexer.PeekToken()
		if peek.Kind == close {
			p.lexer.NextToken()
			break
		}
		if peek.Kind == EOF {
			return nil, fmt.Errorf("unexpected EOF: unclosed '%s'", open.Value)
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, expr)
	}

	return &ListExpr{tok: open, elements: elements}, nil
}

func (p *Parser) parseVector() (Expr, error) {
	open := p.lexer.NextToken() // consume '#('

	var elements []Expr

	for {
		peek := p.lexer.PeekToken()
		if peek.Kind == RParen {
			p.lexer.NextToken()
			break
		}
		if peek.Kind == EOF {
			return nil, fmt.Errorf("unexpected EOF: unclosed '#('")
		}

		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}

		elements = append(elements, expr)
	}

	return &VectorExpr{tok: open, elements: elements}, nil
}

// NewParser creates a parser that reads from the given lexer.
func NewParser(lexer *Lexer) *Parser {
	return &Parser{lexer: lexer}
}

type interpSegment struct {
	text   string
	isExpr bool
}

// splitInterp splits an interpolated string value into alternating literal and
// expression segments. The input is the raw content between the outer quotes,
// with {…} markers still present, e.g. "Hello {name}, {(+ 1 2)}".
func splitInterp(s string) []interpSegment {
	var segs []interpSegment
	var buf strings.Builder
	depth := 0

	for _, r := range s {
		switch {
		case r == '{' && depth == 0:
			if buf.Len() > 0 {
				segs = append(segs, interpSegment{buf.String(), false})
				buf.Reset()
			}
			depth = 1
		case r == '{':
			depth++
			buf.WriteRune(r)
		case r == '}' && depth == 1:
			segs = append(segs, interpSegment{buf.String(), true})
			buf.Reset()
			depth = 0
		case r == '}':
			depth--
			buf.WriteRune(r)
		default:
			buf.WriteRune(r)
		}
	}

	if buf.Len() > 0 {
		segs = append(segs, interpSegment{buf.String(), false})
	}

	return segs
}

// parseInterpString desugars an InterpString token into a string-append call.
// $"Hello {name}!" becomes (string-append "Hello " (->string name) "!").
// If there are no interpolations the result is a plain StringExpr.
func parseInterpString(tok Token) (Expr, error) {
	segs := splitInterp(tok.Value)

	// No interpolations — plain string literal.
	hasExpr := false
	for _, s := range segs {
		if s.isExpr {
			hasExpr = true
			break
		}
	}
	if !hasExpr {
		return &StringExpr{tok: tok, val: tok.Value}, nil
	}

	appendSym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "string-append"}, val: "string-append"}
	toStrSym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "->string"}, val: "->string"}

	var parts []Expr

	for _, seg := range segs {
		if !seg.isExpr {
			if seg.text != "" {
				parts = append(parts, &StringExpr{tok: tok, val: seg.text})
			}
			continue
		}
		lexer, err := NewLexer(strings.NewReader(seg.text))
		if err != nil {
			return nil, fmt.Errorf("interpolation {%s}: %w", seg.text, err)
		}
		sub := &Parser{lexer: lexer}
		expr, err := sub.parseExpr()
		if err != nil {
			return nil, fmt.Errorf("interpolation {%s}: %w", seg.text, err)
		}
		parts = append(parts, &ListExpr{tok: tok, elements: []Expr{toStrSym, expr}})
	}

	if len(parts) == 1 {
		return parts[0], nil
	}

	return &ListExpr{tok: tok, elements: append([]Expr{appendSym}, parts...)}, nil
}
