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

	// Dispatch special forms when the head is a keyword symbol.
	if sym, ok := head.(*SymbolExpr); ok {
		switch sym.tok.Kind {
		case token.Define:
			return evalDefine(tail, env)
		case token.Lambda:
			return evalLambda(tail, env)
		case token.If:
			return evalIf(tail, env)
		case token.Let:
			return evalLet(tail, env)
		case token.LetStar:
			return evalLetStar(tail, env)
		case token.SetBang:
			return evalSetBang(tail, env)
		case token.QuoteLong:
			if len(tail) != 1 {
				return nil, fmt.Errorf("quote: expected 1 argument, got %d", len(tail))
			}
			return tail[0], nil
		}
	}

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

// --- special form evaluators ---

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
		return val, nil
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
		return lambda, nil
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
		return &BoolExpr{val: false}, nil
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
	return val, env.Set(sym.val, val)
}

func evalBody(exprs []Expr, env *Environment) (Expr, error) {
	var result Expr = &BoolExpr{}
	for _, expr := range exprs {
		var err error
		result, err = expr.Eval(env)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
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
