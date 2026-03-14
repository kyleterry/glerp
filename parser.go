package glerp

import (
	"fmt"
	"strconv"

	"go.e64ec.com/glerp/token"
)

// Parser builds an expression tree from a token stream.
type Parser struct {
	lexer *token.Lexer
}

// Run parses all top-level expressions from the token stream.
func (p *Parser) Run() ([]Expr, error) {
	var exprs []Expr
	for p.lexer.PeekToken().Kind != token.EOF {
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
	case token.LParen:
		return p.parseList()

	case token.Quote:
		// Desugar 'expr → (quote expr)
		p.lexer.NextToken()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		quoteSym := &SymbolExpr{
			tok: token.Token{Kind: token.QuoteLong, Value: "quote"},
			val: "quote",
		}
		return &ListExpr{tok: tok, elements: []Expr{quoteSym, inner}}, nil

	case token.Number:
		p.lexer.NextToken()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", tok.Value, err)
		}
		return &NumberExpr{tok: tok, val: v}, nil

	case token.String:
		p.lexer.NextToken()
		return &StringExpr{tok: tok, val: tok.Value}, nil

	case token.BTrue:
		p.lexer.NextToken()
		return &BoolExpr{tok: tok, val: true}, nil

	case token.BFalse:
		p.lexer.NextToken()
		return &BoolExpr{tok: tok, val: false}, nil

	case token.Comment:
		// Skip comment tokens and parse the next expression.
		p.lexer.NextToken()
		return p.parseExpr()

	case token.EOF:
		return nil, fmt.Errorf("unexpected EOF")

	default:
		// Keywords (define, lambda, if, car, …) and plain symbols are all
		// represented as SymbolExpr; special-form dispatch happens at eval time.
		if tok.Kind.IsKeyword() || tok.Kind == token.Symbol {
			p.lexer.NextToken()
			return &SymbolExpr{tok: tok, val: tok.Value}, nil
		}
		return nil, fmt.Errorf("unexpected token: %s (%q)", tok.Kind, tok.Value)
	}
}

func (p *Parser) parseList() (Expr, error) {
	open := p.lexer.NextToken() // consume '('
	var elements []Expr
	for {
		peek := p.lexer.PeekToken()
		if peek.Kind == token.RParen {
			p.lexer.NextToken()
			break
		}
		if peek.Kind == token.EOF {
			return nil, fmt.Errorf("unexpected EOF: unclosed '('")
		}
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elements = append(elements, expr)
	}
	return &ListExpr{tok: open, elements: elements}, nil
}

// NewParser creates a parser that reads from the given lexer.
func NewParser(lexer *token.Lexer) *Parser {
	return &Parser{lexer: lexer}
}
