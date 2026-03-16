package glerp

import (
	"fmt"
	"strconv"
	"strings"

	"go.e64ec.com/glerp/token"
)

// Expr is any scheme value or expression that can be evaluated.
type Expr interface {
	Eval(env *Environment) (Expr, error)
	Token() token.Token
	String() string
}

// NumberExpr is a numeric literal.
type NumberExpr struct {
	tok token.Token
	val float64
}

func (e *NumberExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *NumberExpr) Token() token.Token { return e.tok }

// Value returns the underlying float64 value.
func (e *NumberExpr) Value() float64 { return e.val }

func (e *NumberExpr) String() string {
	if e.val == float64(int64(e.val)) {
		return strconv.FormatInt(int64(e.val), 10)
	}
	return strconv.FormatFloat(e.val, 'f', -1, 64)
}

// StringExpr is a string literal.
type StringExpr struct {
	tok token.Token
	val string
}

func (e *StringExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *StringExpr) Token() token.Token { return e.tok }

// Value returns the raw string contents, without surrounding quotes.
func (e *StringExpr) Value() string { return e.val }

// String returns the quoted representation, e.g. "hello".
func (e *StringExpr) String() string { return fmt.Sprintf("%q", e.val) }

// BoolExpr is a boolean value (#t or #f).
type BoolExpr struct {
	tok token.Token
	val bool
}

func (e *BoolExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *BoolExpr) Token() token.Token { return e.tok }

// Value returns the underlying boolean. Only #f is false; everything else is truthy.
func (e *BoolExpr) Value() bool { return e.val }

func (e *BoolExpr) String() string {
	if e.val {
		return "#t"
	}
	return "#f"
}

// SymbolExpr is a symbol that resolves to a value via environment lookup.
type SymbolExpr struct {
	tok token.Token
	val string
}

func (e *SymbolExpr) Eval(env *Environment) (Expr, error) {
	return env.Find(e.val)
}

// Token returns the source token for this expression.
func (e *SymbolExpr) Token() token.Token { return e.tok }

// String returns the symbol name.
func (e *SymbolExpr) String() string { return e.val }

// ListExpr is a parenthesized s-expression. Evaluation dispatches on the head:
// special forms are handled directly; otherwise it is a procedure application.
type ListExpr struct {
	tok      token.Token
	elements []Expr
}

// Token returns the source token for this expression.
func (e *ListExpr) Token() token.Token { return e.tok }

// Elements returns the expressions contained in this list.
func (e *ListExpr) Elements() []Expr { return e.elements }

func (e *ListExpr) String() string {
	if len(e.elements) == 0 {
		return "()"
	}
	parts := make([]string, len(e.elements))
	for i, el := range e.elements {
		parts[i] = el.String()
	}
	return "(" + strings.Join(parts, " ") + ")"
}

func (e *ListExpr) Eval(env *Environment) (Expr, error) {
	if len(e.elements) == 0 {
		return e, nil
	}

	head := e.elements[0]
	tail := e.elements[1:]

	// Procedure application: evaluate head then args, then apply.
	proc, err := head.Eval(env)
	if err != nil {
		return nil, fmt.Errorf("in procedure position: %w", err)
	}

	// User-registered forms receive unevaluated arguments — they control
	// their own evaluation, just like define, lambda, or if.
	if f, ok := proc.(*FormExpr); ok {
		return f.fn(tail, env)
	}

	// Macro application: expand the transformer against the unevaluated call
	// form, then evaluate the result in the same environment.
	if transformer, ok := proc.(*SyntaxRulesExpr); ok {
		expanded, err := transformer.expand(e)
		if err != nil {
			return nil, err
		}
		return expanded.Eval(env)
	}

	args := make([]Expr, len(tail))
	for i, arg := range tail {
		args[i], err = arg.Eval(env)
		if err != nil {
			return nil, err
		}
	}

	return apply(proc, args)
}

// LambdaExpr is a user-defined procedure (closure).
type LambdaExpr struct {
	tok    token.Token
	params []string
	body   []Expr
	env    *Environment
}

func (e *LambdaExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *LambdaExpr) Token() token.Token { return e.tok }

// String returns a summary representation showing the parameter list.
func (e *LambdaExpr) String() string {
	return "(lambda (" + strings.Join(e.params, " ") + ") ...)"
}

// FormExpr is a Go-implemented special form. Unlike BuiltinExpr, its arguments
// are passed unevaluated, giving the implementation full control over
// evaluation semantics — identical to built-in forms like define and if.
// Register one via Environment.RegisterForm.
type FormExpr struct {
	name string
	fn   func(args []Expr, env *Environment) (Expr, error)
}

func (e *FormExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; FormExpr values have no source position.
func (e *FormExpr) Token() token.Token { return token.Token{} }

// String returns a display name identifying this as a special form.
func (e *FormExpr) String() string { return fmt.Sprintf("#<form:%s>", e.name) }

// ValuesExpr holds multiple return values produced by (values ...).
// It may only appear where multiple values are explicitly consumed, such as
// define-values. Using a ValuesExpr in a single-value position is an error.
type ValuesExpr struct {
	vals []Expr
}

func (e *ValuesExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; ValuesExpr values have no source position.
func (e *ValuesExpr) Token() token.Token { return token.Token{} }

// String returns a readable representation of all contained values.
func (e *ValuesExpr) String() string {
	parts := make([]string, len(e.vals))
	for i, v := range e.vals {
		parts[i] = v.String()
	}
	return "(values " + strings.Join(parts, " ") + ")"
}

// Values returns the individual expressions wrapped by this object.
func (e *ValuesExpr) Values() []Expr { return e.vals }

// VoidExpr is the unspecified return value produced by side-effecting forms
// such as display, newline, define, set!, and import. It is distinct from #f
// so the REPL and callers can suppress printing it.
type VoidExpr struct{}

func (e *VoidExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; VoidExpr has no source position.
func (e *VoidExpr) Token() token.Token { return token.Token{} }

// String returns an empty string; void is intentionally invisible.
func (e *VoidExpr) String() string { return "" }

// BuiltinExpr is a Go-implemented procedure.
type BuiltinExpr struct {
	name string
	fn   func(args []Expr) (Expr, error)
}

func (e *BuiltinExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; BuiltinExpr values have no source position.
func (e *BuiltinExpr) Token() token.Token { return token.Token{} }

// String returns a display name identifying this as a built-in procedure.
func (e *BuiltinExpr) String() string { return fmt.Sprintf("#<builtin:%s>", e.name) }

// apply calls a procedure (lambda or builtin) with already-evaluated arguments.
func apply(proc Expr, args []Expr) (Expr, error) {
	switch p := proc.(type) {
	case *BuiltinExpr:
		return p.fn(args)
	case *LambdaExpr:
		if len(args) != len(p.params) {
			return nil, fmt.Errorf("%s: expected %d args, got %d", p.String(), len(p.params), len(args))
		}
		child := p.env.Extend()
		for i, param := range p.params {
			child.Bind(param, args[i])
		}
		return evalBody(p.body, child)
	default:
		return nil, fmt.Errorf("%s is not a procedure", proc.String())
	}
}

func evalDefine(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("define: too few arguments")
	}
	switch target := args[0].(type) {
	case *SymbolExpr:
		// (define name value)
		if len(args) != 2 {
			return nil, fmt.Errorf("define: variable form expects exactly 1 value")
		}
		val, err := args[1].Eval(env)
		if err != nil {
			return nil, err
		}
		env.Bind(target.val, val)
		return Void(), nil
	case *ListExpr:
		// (define (name params...) body...) — sugar for (define name (lambda ...))
		if len(target.elements) == 0 {
			return nil, fmt.Errorf("define: missing function name")
		}
		nameSym, ok := target.elements[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("define: function name must be a symbol")
		}
		lambda, err := makeLambda(target.elements[1:], args[1:], env)
		if err != nil {
			return nil, err
		}
		env.Bind(nameSym.val, lambda)
		return Void(), nil
	default:
		return nil, fmt.Errorf("define: target must be a symbol or list, got %T", args[0])
	}
}

func evalLambda(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("lambda: requires parameter list and body")
	}
	paramList, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("lambda: parameters must be a list")
	}
	return makeLambda(paramList.elements, args[1:], env)
}

func makeLambda(paramExprs []Expr, body []Expr, env *Environment) (*LambdaExpr, error) {
	params := make([]string, len(paramExprs))
	for i, p := range paramExprs {
		sym, ok := p.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("lambda: parameter must be a symbol, got %T", p)
		}
		params[i] = sym.val
	}
	return &LambdaExpr{params: params, body: body, env: env}, nil
}

func evalIf(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("if: expected 2 or 3 arguments, got %d", len(args))
	}
	cond, err := args[0].Eval(env)
	if err != nil {
		return nil, err
	}
	// In Scheme, only #f is falsy.
	if b, ok := cond.(*BoolExpr); ok && !b.val {
		if len(args) == 3 {
			return args[2].Eval(env)
		}
		return Void(), nil
	}
	return args[1].Eval(env)
}

func evalLet(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("let: requires bindings and body")
	}
	bindings, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("let: bindings must be a list")
	}
	child := env.Extend()
	for _, b := range bindings.elements {
		pair, ok := b.(*ListExpr)
		if !ok || len(pair.elements) != 2 {
			return nil, fmt.Errorf("let: each binding must be (name value)")
		}
		sym, ok := pair.elements[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("let: binding name must be a symbol")
		}
		// Evaluate binding values in the outer env (parallel binding).
		val, err := pair.elements[1].Eval(env)
		if err != nil {
			return nil, err
		}
		child.Bind(sym.val, val)
	}
	return evalBody(args[1:], child)
}

func evalLetStar(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("let*: requires bindings and body")
	}
	bindings, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("let*: bindings must be a list")
	}
	child := env.Extend()
	for _, b := range bindings.elements {
		pair, ok := b.(*ListExpr)
		if !ok || len(pair.elements) != 2 {
			return nil, fmt.Errorf("let*: each binding must be (name value)")
		}
		sym, ok := pair.elements[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("let*: binding name must be a symbol")
		}
		// Evaluate each binding in the growing child env (sequential binding).
		val, err := pair.elements[1].Eval(child)
		if err != nil {
			return nil, err
		}
		child.Bind(sym.val, val)
	}
	return evalBody(args[1:], child)
}

func evalQuote(args []Expr, _ *Environment) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("quote: expected 1 argument, got %d", len(args))
	}
	return args[0], nil
}

func evalQuasiquote(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("quasiquote: expected 1 argument, got %d", len(args))
	}
	return expandQQ(args[0], 0, env)
}

// isTagged reports whether expr is a list whose first element is a symbol with
// the given name, returning the remaining elements if so.
func isTagged(expr Expr, name string) ([]Expr, bool) {
	list, ok := expr.(*ListExpr)
	if !ok || len(list.elements) == 0 {
		return nil, false
	}
	sym, ok := list.elements[0].(*SymbolExpr)
	if !ok || sym.val != name {
		return nil, false
	}
	return list.elements[1:], true
}

// expandQQ recursively expands a quasiquote template at the given nesting
// depth. depth 0 means we are in the innermost quasiquote and unquote
// expressions are evaluated immediately.
func expandQQ(expr Expr, depth int, env *Environment) (Expr, error) {
	if inner, ok := isTagged(expr, "unquote"); ok {
		if len(inner) != 1 {
			return nil, fmt.Errorf("unquote: expected 1 argument, got %d", len(inner))
		}
		if depth == 0 {
			return inner[0].Eval(env)
		}
		expanded, err := expandQQ(inner[0], depth-1, env)
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: token.Token{Kind: token.Symbol, Value: "unquote"}, val: "unquote"}
		return &ListExpr{tok: expr.Token(), elements: []Expr{sym, expanded}}, nil
	}

	if inner, ok := isTagged(expr, "quasiquote"); ok {
		if len(inner) != 1 {
			return nil, fmt.Errorf("quasiquote: expected 1 argument, got %d", len(inner))
		}
		expanded, err := expandQQ(inner[0], depth+1, env)
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: token.Token{Kind: token.Symbol, Value: "quasiquote"}, val: "quasiquote"}
		return &ListExpr{tok: expr.Token(), elements: []Expr{sym, expanded}}, nil
	}

	list, ok := expr.(*ListExpr)
	if !ok {
		return expr, nil
	}

	var result []Expr
	for _, el := range list.elements {
		if spliceArgs, ok := isTagged(el, "unquote-splicing"); ok {
			if len(spliceArgs) != 1 {
				return nil, fmt.Errorf("unquote-splicing: expected 1 argument, got %d", len(spliceArgs))
			}
			if depth == 0 {
				val, err := spliceArgs[0].Eval(env)
				if err != nil {
					return nil, err
				}
				spliceList, ok := val.(*ListExpr)
				if !ok {
					return nil, fmt.Errorf("unquote-splicing: expected a list, got %s", val.String())
				}
				result = append(result, spliceList.elements...)
				continue
			}
			expanded, err := expandQQ(spliceArgs[0], depth-1, env)
			if err != nil {
				return nil, err
			}
			sym := &SymbolExpr{tok: token.Token{Kind: token.Symbol, Value: "unquote-splicing"}, val: "unquote-splicing"}
			result = append(result, &ListExpr{tok: el.Token(), elements: []Expr{sym, expanded}})
			continue
		}
		expanded, err := expandQQ(el, depth, env)
		if err != nil {
			return nil, err
		}
		result = append(result, expanded)
	}
	return &ListExpr{tok: list.tok, elements: result}, nil
}

func evalSetBang(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("set!: expected 2 arguments, got %d", len(args))
	}
	sym, ok := args[0].(*SymbolExpr)
	if !ok {
		return nil, fmt.Errorf("set!: target must be a symbol")
	}
	val, err := args[1].Eval(env)
	if err != nil {
		return nil, err
	}
	if err := env.Set(sym.val, val); err != nil {
		return nil, err
	}
	return Void(), nil
}

func evalBody(exprs []Expr, env *Environment) (Expr, error) {
	result := Void()
	for _, expr := range exprs {
		var err error
		result, err = expr.Eval(env)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// evalDefineValues implements (define-values (name ...) expr).
// expr must evaluate to a ValuesExpr whose arity matches the name list.
// As a special case, a single-name list accepts any non-values result.
func evalDefineValues(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("define-values: expected name list and expression, got %d args", len(args))
	}
	nameList, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("define-values: first argument must be a list of names")
	}
	syms := make([]string, len(nameList.elements))
	for i, el := range nameList.elements {
		sym, ok := el.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("define-values: names must be symbols, got %s", el.String())
		}
		syms[i] = sym.val
	}
	result, err := args[1].Eval(env)
	if err != nil {
		return nil, err
	}
	if mv, ok := result.(*ValuesExpr); ok {
		if len(mv.vals) != len(syms) {
			return nil, fmt.Errorf("define-values: expected %d values, got %d", len(syms), len(mv.vals))
		}
		for i, name := range syms {
			env.Bind(name, mv.vals[i])
		}
		return Void(), nil
	}
	// Single (non-values) result: only valid with exactly one name.
	if len(syms) != 1 {
		return nil, fmt.Errorf("define-values: expected %d values, got 1", len(syms))
	}
	env.Bind(syms[0], result)
	return Void(), nil
}

// eqv reports whether two expressions are equivalent in the sense of Scheme's
// eqv?: identical booleans, equal numbers, equal strings, or identical symbols.
func eqv(a, b Expr) bool {
	switch x := a.(type) {
	case *NumberExpr:
		y, ok := b.(*NumberExpr)
		return ok && x.val == y.val
	case *StringExpr:
		y, ok := b.(*StringExpr)
		return ok && x.val == y.val
	case *BoolExpr:
		y, ok := b.(*BoolExpr)
		return ok && x.val == y.val
	case *SymbolExpr:
		y, ok := b.(*SymbolExpr)
		return ok && x.val == y.val
	}
	return false
}

// evalCase implements (case <key> ((<datum> ...) <body> ...) ... [(else <body> ...)]).
// The key is evaluated once; each clause's datum list is compared against it
// using eqv. The body of the first matching clause is evaluated and returned.
// An else clause matches unconditionally. Returns void if no clause matches.
func evalCase(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("case: requires a key expression and at least one clause")
	}
	key, err := args[0].Eval(env)
	if err != nil {
		return nil, err
	}
	for _, arg := range args[1:] {
		clause, ok := arg.(*ListExpr)
		if !ok || len(clause.elements) < 2 {
			return nil, fmt.Errorf("case: invalid clause %s", arg.String())
		}
		head := clause.elements[0]
		body := clause.elements[1:]
		if sym, ok := head.(*SymbolExpr); ok && sym.val == "else" {
			return evalBody(body, env)
		}
		datums, ok := head.(*ListExpr)
		if !ok {
			return nil, fmt.Errorf("case: clause head must be a datum list or else, got %s", head.String())
		}
		for _, datum := range datums.elements {
			if eqv(key, datum) {
				return evalBody(body, env)
			}
		}
	}
	return Void(), nil
}

func evalBegin(args []Expr, env *Environment) (Expr, error) {
	if len(args) == 0 {
		return Void(), nil
	}
	return evalBody(args, env)
}

func evalCond(args []Expr, env *Environment) (Expr, error) {
	for _, arg := range args {
		clause, ok := arg.(*ListExpr)
		if !ok || len(clause.elements) < 2 {
			return nil, fmt.Errorf("cond: invalid clause %s", arg.String())
		}
		test := clause.elements[0]
		body := clause.elements[1:]
		if sym, ok := test.(*SymbolExpr); ok && sym.val == "else" {
			return evalBody(body, env)
		}
		result, err := test.Eval(env)
		if err != nil {
			return nil, err
		}
		if b, ok := result.(*BoolExpr); !ok || b.val {
			return evalBody(body, env)
		}
	}
	return Void(), nil
}

func evalAnd(args []Expr, env *Environment) (Expr, error) {
	var result Expr = boolean(true)
	for _, arg := range args {
		val, err := arg.Eval(env)
		if err != nil {
			return nil, err
		}
		if b, ok := val.(*BoolExpr); ok && !b.val {
			return boolean(false), nil
		}
		result = val
	}
	return result, nil
}

func evalOr(args []Expr, env *Environment) (Expr, error) {
	for _, arg := range args {
		val, err := arg.Eval(env)
		if err != nil {
			return nil, err
		}
		if b, ok := val.(*BoolExpr); !ok || b.val {
			return val, nil
		}
	}
	return boolean(false), nil
}

// evalDo implements the R7RS do iteration form:
//
//	(do ((<var> <init> [<step>]) ...)
//	    (<test> <result> ...)
//	  <command> ...)
//
// Variables are bound to their <init> values, then each iteration evaluates
// <test>: if truthy, the <result> expressions are evaluated and the last is
// returned (void if none). Otherwise the <command> body is run for side
// effects, then all <step> expressions are evaluated in parallel in the
// current environment and the bindings are updated simultaneously.
func evalDo(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("do: requires variable specs and a termination clause")
	}

	varList, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("do: first argument must be a list of variable specs")
	}

	type spec struct {
		name string
		step Expr // nil means no step — variable keeps its value
	}
	specs := make([]spec, len(varList.elements))

	loopEnv := env.Extend()
	for i, el := range varList.elements {
		clause, ok := el.(*ListExpr)
		if !ok || len(clause.elements) < 2 || len(clause.elements) > 3 {
			return nil, fmt.Errorf("do: variable spec must be (var init) or (var init step), got %s", el.String())
		}
		sym, ok := clause.elements[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("do: variable name must be a symbol, got %s", clause.elements[0].String())
		}
		init, err := clause.elements[1].Eval(env)
		if err != nil {
			return nil, err
		}
		loopEnv.Bind(sym.val, init)
		var step Expr
		if len(clause.elements) == 3 {
			step = clause.elements[2]
		}
		specs[i] = spec{name: sym.val, step: step}
	}

	term, ok := args[1].(*ListExpr)
	if !ok || len(term.elements) == 0 {
		return nil, fmt.Errorf("do: termination clause must be a non-empty list")
	}
	testExpr := term.elements[0]
	resultExprs := term.elements[1:]

	commands := args[2:]

	for {
		testVal, err := testExpr.Eval(loopEnv)
		if err != nil {
			return nil, err
		}
		// Truthy test: return result (void if no result expressions).
		if b, ok := testVal.(*BoolExpr); !ok || b.val {
			if len(resultExprs) == 0 {
				return Void(), nil
			}
			return evalBody(resultExprs, loopEnv)
		}

		// Execute body commands for side effects.
		for _, cmd := range commands {
			if _, err := cmd.Eval(loopEnv); err != nil {
				return nil, err
			}
		}

		// Evaluate all step expressions in the current environment (parallel).
		next := make([]Expr, len(specs))
		for i, s := range specs {
			if s.step != nil {
				val, err := s.step.Eval(loopEnv)
				if err != nil {
					return nil, err
				}
				next[i] = val
			} else {
				next[i], _ = loopEnv.Find(s.name)
			}
		}
		// Apply all updates simultaneously.
		for i, s := range specs {
			loopEnv.Bind(s.name, next[i])
		}
	}
}
