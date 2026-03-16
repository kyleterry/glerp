// Package glerp is a small Scheme interpreter designed for embedding in Go
// programs. It provides a subset of Scheme sufficient for configuration and
// scripting use cases.
package glerp

import (
	"strings"

	"go.e64ec.com/glerp/token"
)

// voidSingleton is the single shared VoidExpr instance.
var voidSingleton Expr = &VoidExpr{}

// Void returns the canonical unspecified result for side-effecting forms.
// Form handlers that have side effects but no meaningful return value should
// return Void(). The REPL suppresses printing VoidExpr values.
func Void() Expr { return voidSingleton }

// Eval is a convenience function that parses and evaluates all top-level
// expressions in src within env, returning the result of each one.
func Eval(src string, env *Environment) ([]Expr, error) {
	lexer, err := token.NewLexer(strings.NewReader(src))
	if err != nil {
		return nil, err
	}

	p := NewParser(lexer)
	exprs, err := p.Run()
	if err != nil {
		return nil, err
	}

	results := make([]Expr, 0, len(exprs))

	for _, expr := range exprs {
		result, err := expr.Eval(env)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}
