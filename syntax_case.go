package glerp

import (
	"fmt"
	"maps"
)

// syntaxBindingsKey is the environment key used to thread macro bindings
// from syntax-case to the syntax template form.
const syntaxBindingsKey = "%syntax-bindings%"

// syntaxEnvExpr stores macro pattern bindings and literals for use by
// the syntax template form. Bound to syntaxBindingsKey in the environment
// by syntax-case and with-syntax.
type syntaxEnvExpr struct {
	bindings *macroBindings
	literals map[string]bool
}

func (e *syntaxEnvExpr) Eval(_ *Environment) (Expr, error) { return e, nil }
func (e *syntaxEnvExpr) Token() Token                      { return Token{} }
func (e *syntaxEnvExpr) String() string                    { return "#<syntax-env>" }

// TransformerExpr wraps a procedure for use as a syntax-case macro
// transformer. When the transformer appears in the head position of a
// form, the entire unevaluated call form is passed to the wrapped
// procedure as a single argument.
type TransformerExpr struct {
	proc Expr // LambdaExpr or BuiltinExpr
}

func (e *TransformerExpr) Eval(_ *Environment) (Expr, error) { return e, nil }
func (e *TransformerExpr) Token() Token                      { return Token{} }
func (e *TransformerExpr) String() string                    { return "#<transformer>" }

// evalSyntaxCase implements:
//
//	(syntax-case <input> (<literal> ...)
//	  [<pattern> <body>]
//	  [<pattern> <fender> <body>]
//	  ...)
//
// The input is evaluated, then matched against each clause's pattern in
// order. If a fender is present, it is evaluated after a successful match
// and must return a truthy value for the clause to be selected. Pattern
// variables are bound in the clause body's environment and also stored
// for use by the syntax template form.
func evalSyntaxCase(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 3 {
		return nil, fmt.Errorf("syntax-case: expected input, literals, and at least one clause")
	}

	input, err := args[0].Eval(env)
	if err != nil {
		return nil, err
	}

	litList, ok := args[1].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("syntax-case: literals must be a list")
	}

	literals := make(map[string]bool, len(litList.elements))
	for _, el := range litList.elements {
		sym, ok := el.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("syntax-case: literal must be a symbol, got %s", el.String())
		}
		literals[sym.val] = true
	}

	for _, arg := range args[2:] {
		clause, ok := arg.(*ListExpr)
		if !ok || len(clause.elements) < 2 || len(clause.elements) > 3 {
			return nil, fmt.Errorf("syntax-case: clause must be [pattern body] or [pattern fender body], got %s", arg.String())
		}

		pattern := clause.elements[0]

		var fender, body Expr
		if len(clause.elements) == 3 {
			fender = clause.elements[1]
			body = clause.elements[2]
		} else {
			body = clause.elements[1]
		}

		b := newMacroBindings()
		if !matchPattern(pattern, input, literals, b) {
			continue
		}

		child := env.Extend()
		bindSyntaxVars(child, b)
		storeSyntaxBindings(child, b, literals, env)

		if fender != nil {
			fv, err := fender.Eval(child)
			if err != nil {
				return nil, err
			}
			if fb, ok := fv.(*BoolExpr); ok && !fb.val {
				continue
			}
		}

		return body.Eval(child)
	}

	return nil, fmt.Errorf("syntax-case: no matching clause for %s", input.String())
}

// evalSyntax implements (syntax <template>). It expands the template
// using pattern variable bindings from the enclosing syntax-case or
// with-syntax. Outside of a syntax-case context it acts like quote.
func evalSyntax(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("syntax: expected 1 argument, got %d", len(args))
	}

	se := lookupSyntaxEnv(env)
	if se == nil {
		return args[0], nil
	}

	return expandTemplate(args[0], se.bindings, se.literals)
}

// evalWithSyntax implements:
//
//	(with-syntax ([<pattern> <expr>] ...) <body> ...)
//
// Each expr is evaluated and matched against its pattern. The resulting
// bindings are merged with any existing syntax-case bindings and made
// available to the body (both as environment bindings and for use by
// the syntax template form).
func evalWithSyntax(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("with-syntax: expected bindings and body")
	}

	bindingList, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("with-syntax: first argument must be a list of bindings")
	}

	existing := lookupSyntaxEnv(env)
	merged := newMacroBindings()
	mergedLiterals := make(map[string]bool)

	if existing != nil {
		maps.Copy(merged.vars, existing.bindings.vars)
		maps.Copy(merged.ellipsis, existing.bindings.ellipsis)
		maps.Copy(mergedLiterals, existing.literals)
	}

	for _, binding := range bindingList.elements {
		pair, ok := binding.(*ListExpr)
		if !ok || len(pair.elements) != 2 {
			return nil, fmt.Errorf("with-syntax: each binding must be [pattern expr], got %s", binding.String())
		}

		val, err := pair.elements[1].Eval(env)
		if err != nil {
			return nil, err
		}

		b := newMacroBindings()
		if !matchPattern(pair.elements[0], val, mergedLiterals, b) {
			return nil, fmt.Errorf("with-syntax: pattern %s does not match value %s",
				pair.elements[0].String(), val.String())
		}

		maps.Copy(merged.vars, b.vars)
		maps.Copy(merged.ellipsis, b.ellipsis)
	}

	child := env.Extend()
	bindSyntaxVars(child, merged)
	child.Bind(syntaxBindingsKey, &syntaxEnvExpr{bindings: merged, literals: mergedLiterals})

	return evalBody(args[1:], child)
}

// bindSyntaxVars binds pattern variables from macro bindings into the
// environment so the syntax-case body can reference them as regular
// variables. Simple vars are bound directly; ellipsis vars are bound
// as ListExpr values.
func bindSyntaxVars(env *Environment, b *macroBindings) {
	for name, val := range b.vars {
		env.Bind(name, val)
	}

	for name, vals := range b.ellipsis {
		env.Bind(name, &ListExpr{elements: vals})
	}
}

// storeSyntaxBindings saves macro bindings into the environment for use
// by the syntax template form. Merges with any existing bindings from
// an enclosing syntax-case.
func storeSyntaxBindings(env *Environment, b *macroBindings, literals map[string]bool, outer *Environment) {
	existing := lookupSyntaxEnv(outer)

	merged := newMacroBindings()
	mergedLiterals := make(map[string]bool)

	if existing != nil {
		maps.Copy(merged.vars, existing.bindings.vars)
		maps.Copy(merged.ellipsis, existing.bindings.ellipsis)
		maps.Copy(mergedLiterals, existing.literals)
	}

	maps.Copy(merged.vars, b.vars)
	maps.Copy(merged.ellipsis, b.ellipsis)
	maps.Copy(mergedLiterals, literals)

	env.Bind(syntaxBindingsKey, &syntaxEnvExpr{bindings: merged, literals: mergedLiterals})
}

// evalQuasisyntax implements (quasisyntax <template>) / #`<template>.
// Like syntax, it expands pattern variables from the enclosing syntax-case,
// but allows unsyntax (#,) and unsyntax-splicing (#,@) escapes to inject
// computed values into the template.
func evalQuasisyntax(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("quasisyntax: expected 1 argument, got %d", len(args))
	}

	se := lookupSyntaxEnv(env)

	return expandQuasisyntax(args[0], se, env)
}

// expandQuasisyntax walks a template, expanding pattern variables via syntax
// bindings, evaluating (unsyntax ...) escapes, and splicing (unsyntax-splicing ...).
func expandQuasisyntax(tmpl Expr, se *syntaxEnvExpr, env *Environment) (Expr, error) {
	list, ok := tmpl.(*ListExpr)
	if !ok {
		// Atom: expand as syntax template if we have bindings, otherwise return as-is.
		if se != nil {
			return expandTemplate(tmpl, se.bindings, se.literals)
		}

		return tmpl, nil
	}

	if len(list.elements) == 0 {
		return list, nil
	}

	// Check for (unsyntax expr) form.
	if sym, ok := list.elements[0].(*SymbolExpr); ok && sym.val == "unsyntax" {
		if len(list.elements) != 2 {
			return nil, fmt.Errorf("unsyntax: expected 1 argument, got %d", len(list.elements)-1)
		}

		return list.elements[1].Eval(env)
	}

	// Walk children, handling unsyntax-splicing.
	var result []Expr

	for _, elem := range list.elements {
		inner, ok := elem.(*ListExpr)
		if ok && len(inner.elements) == 2 {
			if sym, ok := inner.elements[0].(*SymbolExpr); ok && sym.val == "unsyntax-splicing" {
				val, err := inner.elements[1].Eval(env)
				if err != nil {
					return nil, err
				}

				spliced, ok := val.(*ListExpr)
				if !ok {
					return nil, fmt.Errorf("unsyntax-splicing: expected list, got %s", val.String())
				}

				result = append(result, spliced.elements...)

				continue
			}
		}

		expanded, err := expandQuasisyntax(elem, se, env)
		if err != nil {
			return nil, err
		}

		result = append(result, expanded)
	}

	return &ListExpr{elements: result}, nil
}

// lookupSyntaxEnv retrieves the syntax environment from the given scope.
func lookupSyntaxEnv(env *Environment) *syntaxEnvExpr {
	if env == nil {
		return nil
	}

	val, err := env.Find(syntaxBindingsKey)
	if err != nil {
		return nil
	}

	se, _ := val.(*syntaxEnvExpr)

	return se
}
