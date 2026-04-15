package glerp

import (
	"fmt"
	"strconv"
	"strings"
)

// Expr is any scheme value or expression that can be evaluated.
type Expr interface {
	Eval(env *Environment) (Expr, error)
	Token() Token
	String() string
}

// joinExprs formats a slice of expressions into a single string with the
// given separator, used by ListExpr, VectorExpr, and ValuesExpr.
func joinExprs(exprs []Expr, sep string) string {
	parts := make([]string, len(exprs))

	for i, e := range exprs {
		parts[i] = e.String()
	}

	return strings.Join(parts, sep)
}

// isFalse reports whether e is the Scheme false value (#f).
// In Scheme, only #f is falsy; everything else (including 0, "", and '()) is truthy.
func isFalse(e Expr) bool {
	b, ok := e.(*BoolExpr)
	return ok && !b.val
}

// NumberExpr is a numeric literal.
type NumberExpr struct {
	tok Token
	val float64
}

func (e *NumberExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *NumberExpr) Token() Token { return e.tok }

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
	tok Token
	val string
}

func (e *StringExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *StringExpr) Token() Token { return e.tok }

// Value returns the raw string contents, without surrounding quotes.
func (e *StringExpr) Value() string { return e.val }

// String returns the quoted representation, e.g. "hello".
func (e *StringExpr) String() string { return fmt.Sprintf("%q", e.val) }

// BoolExpr is a boolean value (#t or #f).
type BoolExpr struct {
	tok Token
	val bool
}

func (e *BoolExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *BoolExpr) Token() Token { return e.tok }

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
	tok Token
	val string
}

func (e *SymbolExpr) Eval(env *Environment) (Expr, error) {
	return env.Find(e.val)
}

// Token returns the source token for this expression.
func (e *SymbolExpr) Token() Token { return e.tok }

// String returns the symbol name.
func (e *SymbolExpr) String() string { return e.val }

// Pair is a traditional Scheme cons cell with a car and a cdr.
// Lists in glerp are represented as a chain of Pair objects, ending in a
// null ListExpr (the empty list).
type Pair struct {
	tok Token
	car Expr
	cdr Expr
}

func (p *Pair) Eval(env *Environment) (Expr, error) {
	proc, err := EvalFull(p.car, env)
	if err != nil {
		return nil, fmt.Errorf("in procedure position: %w", err)
	}

	if f, ok := proc.(*FormExpr); ok {
		var slice []Expr
		if pair, ok := p.cdr.(*Pair); ok {
			var err error
			slice, err = pair.ToSlice()
			if err != nil {
				return nil, err
			}
		}
		return f.fn(slice, env)
	}

	if transformer, ok := proc.(*TransformerExpr); ok {
		result, err := trampoline(apply(transformer.proc, []Expr{p}))
		if err != nil {
			return nil, err
		}
		return EvalFull(result, env)
	}

	if transformer, ok := proc.(*SyntaxRulesExpr); ok {
		expanded, err := transformer.expand(p)
		if err != nil {
			return nil, err
		}
		return EvalFull(expanded, env)
	}

	var args []Expr
	curr := p.cdr
	for {
		switch l := curr.(type) {
		case *Pair:
			val, err := EvalFull(l.car, env)
			if err != nil {
				return nil, err
			}
			args = append(args, val)
			curr = l.cdr
		case *ListExpr:
			if len(l.elements) == 0 {
				goto apply
			}
			return nil, fmt.Errorf("dotted list in procedure application")
		default:
			return nil, fmt.Errorf("dotted list in procedure application")
		}
	}

apply:
	return apply(proc, args)
}

func (p *Pair) Token() Token { return p.tok }

func (p *Pair) String() string {
	var sb strings.Builder
	sb.WriteString("(")
	curr := p
	for {
		sb.WriteString(curr.car.String())
		switch next := curr.cdr.(type) {
		case *Pair:
			sb.WriteString(" ")
			curr = next
		case *ListExpr:
			if len(next.elements) == 0 {
				sb.WriteString(")")
				return sb.String()
			}
			// Should not happen with proper list structure but handle anyway.
			sb.WriteString(" . ")
			sb.WriteString(next.String())
			sb.WriteString(")")
			return sb.String()
		default:
			sb.WriteString(" . ")
			sb.WriteString(next.String())
			sb.WriteString(")")
			return sb.String()
		}
	}
}

func (p *Pair) Car() Expr     { return p.car }
func (p *Pair) Cdr() Expr     { return p.cdr }
func (p *Pair) SetCar(e Expr) { p.car = e }
func (p *Pair) SetCdr(e Expr) { p.cdr = e }

// ToSlice converts a proper list (chain of pairs ending in '()) to a slice.
// Returns an error if the list is improper (dotted).
func (p *Pair) ToSlice() ([]Expr, error) {
	var res []Expr
	var curr Expr = p
	for {
		switch l := curr.(type) {
		case *Pair:
			res = append(res, l.car)
			curr = l.cdr
		case *ListExpr:
			if l == Null() {
				return res, nil
			}
			return nil, fmt.Errorf("dotted list cannot be converted to slice")
		default:
			return nil, fmt.Errorf("dotted list cannot be converted to slice")
		}
	}
}

var nullSingleton = &ListExpr{}

// Null returns the empty list singleton.
func Null() *ListExpr {
	return nullSingleton
}

// ListExpr is a parenthesized s-expression. Evaluation dispatches on the head:
// special forms are handled directly; otherwise it is a procedure application.
//
// In glerp, ListExpr now primarily serves as a bridge for special forms and
// as the representation for the empty list. Chains of pairs are converted to
// ListExpr during evaluation to reuse the slice-based dispatch logic.
type ListExpr struct {
	tok      Token
	elements []Expr
}

// Token returns the source token for this expression.
func (e *ListExpr) Token() Token { return e.tok }

// Elements returns the expressions contained in this list.
func (e *ListExpr) Elements() []Expr { return e.elements }

func (e *ListExpr) String() string {
	if len(e.elements) == 0 {
		return "()"
	}

	return "(" + joinExprs(e.elements, " ") + ")"
}

func (e *ListExpr) Eval(env *Environment) (Expr, error) {
	if len(e.elements) == 0 {
		return e, nil
	}

	head := e.elements[0]
	tail := e.elements[1:]

	proc, err := EvalFull(head, env)
	if err != nil {
		return nil, fmt.Errorf("in procedure position: %w", err)
	}

	if f, ok := proc.(*FormExpr); ok {
		return f.fn(tail, env)
	}

	if transformer, ok := proc.(*TransformerExpr); ok {
		result, err := trampoline(apply(transformer.proc, []Expr{e}))
		if err != nil {
			return nil, err
		}
		return EvalFull(result, env)
	}

	if transformer, ok := proc.(*SyntaxRulesExpr); ok {
		expanded, err := transformer.expand(e)
		if err != nil {
			return nil, err
		}
		return EvalFull(expanded, env)
	}

	args := make([]Expr, len(tail))

	for i, arg := range tail {
		args[i], err = EvalFull(arg, env)
		if err != nil {
			return nil, err
		}
	}

	return apply(proc, args)
}

// VectorExpr is a fixed-length, mutable array of Scheme values.
// Vectors are self-evaluating (like numbers and strings). They are written
// as #(elem ...) and support O(1) access by index via vector-ref.
type VectorExpr struct {
	tok      Token
	elements []Expr
}

func (e *VectorExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *VectorExpr) Token() Token { return e.tok }

// Elements returns the expressions contained in this vector.
func (e *VectorExpr) Elements() []Expr { return e.elements }

// Length returns the number of elements in this vector.
func (e *VectorExpr) Length() int { return len(e.elements) }

func (e *VectorExpr) String() string {
	return "#(" + joinExprs(e.elements, " ") + ")"
}

// HashTableExpr is a mutable hash table mapping Scheme values to Scheme values.
// Keys are compared by their String() representation. Written as {k1 v1 k2 v2 ...}.
type HashTableExpr struct {
	tok   Token
	pairs []Expr          // unevaluated AST pairs (nil after eval)
	data  map[string]Expr // String(key) -> value
	keys  map[string]Expr // String(key) -> evaluated key
	order []string        // insertion-order of String(key)
}

func (e *HashTableExpr) Eval(env *Environment) (Expr, error) {
	if e.data != nil {
		return e, nil
	}

	ht := newHashTable(e.tok)

	for i := 0; i < len(e.pairs); i += 2 {
		k, err := EvalFull(e.pairs[i], env)
		if err != nil {
			return nil, err
		}

		v, err := EvalFull(e.pairs[i+1], env)
		if err != nil {
			return nil, err
		}

		ht.Set(k, v)
	}

	return ht, nil
}

func (e *HashTableExpr) Token() Token { return e.tok }

func (e *HashTableExpr) String() string {
	if e.data == nil {
		return "{" + joinExprs(e.pairs, " ") + "}"
	}

	if len(e.order) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(e.order)*2)

	for _, sk := range e.order {
		if k, ok := e.keys[sk]; ok {
			parts = append(parts, k.String(), e.data[sk].String())
		}
	}

	return "{" + strings.Join(parts, " ") + "}"
}

func newHashTable(tok Token) *HashTableExpr {
	return &HashTableExpr{
		tok:  tok,
		data: make(map[string]Expr),
		keys: make(map[string]Expr),
	}
}

func (e *HashTableExpr) Get(key Expr) (Expr, bool) {
	v, ok := e.data[e.key(key)]
	return v, ok
}

func (e *HashTableExpr) Set(key, val Expr) {
	sk := e.key(key)

	if _, exists := e.data[sk]; !exists {
		e.order = append(e.order, sk)
	}

	e.data[sk] = val
	e.keys[sk] = key
}

func (e *HashTableExpr) Delete(key Expr) {
	sk := e.key(key)

	delete(e.data, sk)
	delete(e.keys, sk)

	for i, k := range e.order {
		if k == sk {
			e.order = append(e.order[:i], e.order[i+1:]...)
			break
		}
	}
}

func (e *HashTableExpr) key(key Expr) string {
	return fmt.Sprintf("%T:%s", key, key.String())
}

func (e *HashTableExpr) Size() int { return len(e.data) }

// LambdaExpr is a user-defined procedure (closure).
type LambdaExpr struct {
	tok    Token
	params []string
	rest   string // non-empty when the lambda accepts variadic trailing args
	body   []Expr
	env    *Environment
}

func (e *LambdaExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns the source token for this expression.
func (e *LambdaExpr) Token() Token { return e.tok }

// String returns a summary representation showing the parameter list.
func (e *LambdaExpr) String() string {
	return "#<procedure>"
}

// FormExpr is a Go-implemented special form. Unlike BuiltinExpr, its arguments
// are passed unevaluated, giving the implementation full control over
// evaluation semantics (identical to built-in forms like define and if).
// Register one via Environment.RegisterForm.
type FormExpr struct {
	name string
	fn   func(args []Expr, env *Environment) (Expr, error)
}

func (e *FormExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; FormExpr values have no source position.
func (e *FormExpr) Token() Token { return Token{} }

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
func (e *ValuesExpr) Token() Token { return Token{} }

// String returns a readable representation of all contained values.
func (e *ValuesExpr) String() string {
	return "(values " + joinExprs(e.vals, " ") + ")"
}

// Values returns the individual expressions wrapped by this object.
func (e *ValuesExpr) Values() []Expr { return e.vals }

// VoidExpr is the unspecified return value produced by side-effecting forms
// such as display, newline, define, set!, and import. It is distinct from #f
// so the REPL and callers can suppress printing it.
type VoidExpr struct{}

func (e *VoidExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; VoidExpr has no source position.
func (e *VoidExpr) Token() Token { return Token{} }

// String returns an empty string; void is intentionally invisible.
func (e *VoidExpr) String() string { return "" }

// BuiltinExpr is a Go-implemented procedure.
type BuiltinExpr struct {
	name string
	fn   func(args []Expr) (Expr, error)
}

func (e *BuiltinExpr) Eval(_ *Environment) (Expr, error) { return e, nil }

// Token returns an empty token; BuiltinExpr values have no source position.
func (e *BuiltinExpr) Token() Token { return Token{} }

// String returns a display name identifying this as a built-in procedure.
func (e *BuiltinExpr) String() string { return fmt.Sprintf("#<builtin:%s>", e.name) }

// apply calls a procedure (lambda or builtin) with already-evaluated arguments.
func apply(proc Expr, args []Expr) (Expr, error) {
	switch p := proc.(type) {
	case *BuiltinExpr:
		return p.fn(args)
	case *LambdaExpr:
		if p.rest == "" && len(args) != len(p.params) {
			return nil, fmt.Errorf("%s: expected %d args, got %d", p.String(), len(p.params), len(args))
		}
		if p.rest != "" && len(args) < len(p.params) {
			return nil, fmt.Errorf("%s: expected at least %d args, got %d", p.String(), len(p.params), len(args))
		}

		child := p.env.Extend()

		for i, param := range p.params {
			child.Bind(param, args[i])
		}

		if p.rest != "" {
			restList, _ := builtinList(args[len(p.params):])
			child.Bind(p.rest, restList)
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

		val, err := EvalFull(args[1], env)
		if err != nil {
			return nil, err
		}

		env.Bind(target.val, val)

		return Void(), nil

	case *Pair:
		// (define (name params...) body...) is sugar for (define name (lambda (params...) body...))
		nameSym, ok := target.car.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("define: function name must be a symbol")
		}

		lambda, err := makeLambda(target.cdr, args[1:], env)
		if err != nil {
			return nil, err
		}

		env.Bind(nameSym.val, lambda)

		return Void(), nil
	default:
		return nil, fmt.Errorf("define: target must be a symbol or pair, got %T", args[0])
	}
}

func evalLambda(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("lambda: requires parameter list and body")
	}

	return makeLambda(args[0], args[1:], env)
}

func makeLambda(paramsExpr Expr, body []Expr, env *Environment) (*LambdaExpr, error) {
	var params []string
	var rest string

	curr := paramsExpr
	for {
		switch p := curr.(type) {
		case *Pair:
			sym, ok := p.car.(*SymbolExpr)
			if !ok {
				return nil, fmt.Errorf("lambda: parameter must be a symbol, got %T", p.car)
			}
			params = append(params, sym.val)
			curr = p.cdr
		case *SymbolExpr:
			rest = p.val
			goto done
		case *ListExpr:
			if len(p.elements) == 0 {
				goto done
			}
			return nil, fmt.Errorf("lambda: invalid parameter list")
		default:
			return nil, fmt.Errorf("lambda: invalid parameter list")
		}
	}
done:
	return &LambdaExpr{params: params, rest: rest, body: body, env: env}, nil
}

func evalIf(args []Expr, env *Environment) (Expr, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("if: expected 2 or 3 arguments, got %d", len(args))
	}

	cond, err := EvalFull(args[0], env)
	if err != nil {
		return nil, err
	}

	if isFalse(cond) {
		if len(args) == 3 {
			return &TailCall{expr: args[2], env: env}, nil
		}
		return Void(), nil
	}

	return &TailCall{expr: args[1], env: env}, nil
}

// evalLetBindings is the shared core of let and let*. In let (sequential=false),
// all binding values are evaluated in the outer env before any are bound. In
// let* (sequential=true), each binding is evaluated in the growing child env.
func evalLetBindings(name string, args []Expr, env *Environment, sequential bool) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("%s: requires bindings and body", name)
	}

	bindingSlice, err := toSlice(name, args[0])
	if err != nil {
		return nil, err
	}

	child := env.Extend()

	for _, b := range bindingSlice {
		pairSlice, err := toSlice(name, b)
		if err != nil || len(pairSlice) != 2 {
			return nil, fmt.Errorf("%s: each binding must be (name value)", name)
		}

		sym, ok := pairSlice[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("%s: binding name must be a symbol", name)
		}

		evalEnv := env
		if sequential {
			evalEnv = child
		}

		val, err := EvalFull(pairSlice[1], evalEnv)
		if err != nil {
			return nil, err
		}

		child.Bind(sym.val, val)
	}

	return evalBody(args[1:], child)
}

func evalLet(args []Expr, env *Environment) (Expr, error) {
	return evalLetBindings("let", args, env, false)
}

func evalLetStar(args []Expr, env *Environment) (Expr, error) {
	return evalLetBindings("let*", args, env, true)
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
	elements, err := toSlice(name, expr)
	if err != nil || len(elements) == 0 {
		return nil, false
	}

	sym, ok := elements[0].(*SymbolExpr)
	if !ok || sym.val != name {
		return nil, false
	}

	return elements[1:], true
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
			return EvalFull(inner[0], env)
		}
		expanded, err := expandQQ(inner[0], depth-1, env)
		if err != nil {
			return nil, err
		}
		sym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "unquote"}, val: "unquote"}
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
		sym := &SymbolExpr{tok: Token{Kind: Symbol, Value: "quasiquote"}, val: "quasiquote"}
		return &ListExpr{tok: expr.Token(), elements: []Expr{sym, expanded}}, nil
	}

	// Hash tables: expand key/value elements.
	if ht, ok := expr.(*HashTableExpr); ok && ht.pairs != nil {
		var result []Expr

		for _, el := range ht.pairs {
			expanded, err := expandQQ(el, depth, env)
			if err != nil {
				return nil, err
			}

			result = append(result, expanded)
		}

		return &HashTableExpr{tok: ht.tok, pairs: result}, nil
	}

	// Vectors: expand elements the same way as lists, but produce a VectorExpr.
	if vec, ok := expr.(*VectorExpr); ok {
		var result []Expr

		for _, el := range vec.elements {
			expanded, err := expandQQ(el, depth, env)
			if err != nil {
				return nil, err
			}
			result = append(result, expanded)
		}

		return &VectorExpr{tok: vec.tok, elements: result}, nil
	}

	elements, err := toSlice("quasiquote", expr)
	if err != nil {
		// Improper list (dotted)
		if p, ok := expr.(*Pair); ok {
			car, err := expandQQ(p.car, depth, env)
			if err != nil {
				return nil, err
			}
			cdr, err := expandQQ(p.cdr, depth, env)
			if err != nil {
				return nil, err
			}
			return &Pair{tok: p.tok, car: car, cdr: cdr}, nil
		}
		return expr, nil
	}

	var result []Expr

	for _, el := range elements {
		if spliceArgs, ok := isTagged(el, "unquote-splicing"); ok {
			if len(spliceArgs) != 1 {
				return nil, fmt.Errorf(
					"unquote-splicing: expected 1 argument, got %d",
					len(spliceArgs),
				)
			}

			if depth == 0 {
				val, err := EvalFull(spliceArgs[0], env)
				if err != nil {
					return nil, err
				}

				spliceSlice, err := toSlice("unquote-splicing", val)
				if err != nil {
					return nil, fmt.Errorf(
						"unquote-splicing: expected a list, got %s",
						val.String(),
					)
				}

				result = append(result, spliceSlice...)
				continue
			}

			expanded, err := expandQQ(spliceArgs[0], depth-1, env)
			if err != nil {
				return nil, err
			}

			sym := &SymbolExpr{
				tok: Token{Kind: Symbol, Value: "unquote-splicing"},
				val: "unquote-splicing",
			}
			unquoteList, _ := builtinList([]Expr{sym, expanded})
			result = append(result, unquoteList)
			continue
		}

		expanded, err := expandQQ(el, depth, env)
		if err != nil {
			return nil, err
		}

		result = append(result, expanded)
	}

	return builtinList(result)
}

func evalSetBang(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("set!: expected 2 arguments, got %d", len(args))
	}

	sym, ok := args[0].(*SymbolExpr)
	if !ok {
		return nil, fmt.Errorf("set!: target must be a symbol")
	}

	val, err := EvalFull(args[1], env)
	if err != nil {
		return nil, err
	}

	if err := env.Set(sym.val, val); err != nil {
		return nil, err
	}

	return Void(), nil
}

func evalBody(exprs []Expr, env *Environment) (Expr, error) {
	if len(exprs) == 0 {
		return Void(), nil
	}

	for _, expr := range exprs[:len(exprs)-1] {
		if _, err := EvalFull(expr, env); err != nil {
			return nil, err
		}
	}

	return &TailCall{expr: exprs[len(exprs)-1], env: env}, nil
}

// evalDefineValues implements (define-values (name ...) expr).
// expr must evaluate to a ValuesExpr whose arity matches the name list.
// As a special case, a single-name list accepts any non-values result.
func evalDefineValues(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf(
			"define-values: expected name list and expression, got %d args",
			len(args),
		)
	}

	nameSlice, err := toSlice("define-values", args[0])
	if err != nil {
		return nil, err
	}

	syms := make([]string, len(nameSlice))

	for i, el := range nameSlice {
		sym, ok := el.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("define-values: names must be symbols, got %s", el.String())
		}
		syms[i] = sym.val
	}

	result, err := EvalFull(args[1], env)
	if err != nil {
		return nil, err
	}

	if mv, ok := result.(*ValuesExpr); ok {
		if len(mv.vals) != len(syms) {
			return nil, fmt.Errorf(
				"define-values: expected %d values, got %d",
				len(syms),
				len(mv.vals),
			)
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
// For vectors, eqv? tests identity (pointer equality), not structural equality.
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
	case *Pair:
		return a == b
	case *ListExpr:
		return a == b
	case *VectorExpr:
		return a == b
	case *HashTableExpr:
		return a == b
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

	key, err := EvalFull(args[0], env)
	if err != nil {
		return nil, err
	}

	for _, arg := range args[1:] {
		clauseSlice, err := toSlice("case", arg)
		if err != nil || len(clauseSlice) < 2 {
			return nil, fmt.Errorf("case: invalid clause %s", arg.String())
		}

		head := clauseSlice[0]
		body := clauseSlice[1:]

		if sym, ok := head.(*SymbolExpr); ok && sym.val == "else" {
			return evalBody(body, env)
		}

		datumSlice, err := toSlice("case", head)
		if err != nil {
			return nil, fmt.Errorf(
				"case: clause head must be a datum list or else, got %s",
				head.String(),
			)
		}

		for _, datum := range datumSlice {
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
		clauseSlice, err := toSlice("cond", arg)
		if err != nil || len(clauseSlice) < 2 {
			return nil, fmt.Errorf("cond: invalid clause %s", arg.String())
		}

		test := clauseSlice[0]
		body := clauseSlice[1:]

		if sym, ok := test.(*SymbolExpr); ok && sym.val == "else" {
			return evalBody(body, env)
		}

		result, err := EvalFull(test, env)
		if err != nil {
			return nil, err
		}

		if !isFalse(result) {
			return evalBody(body, env)
		}
	}

	return Void(), nil
}

func evalAnd(args []Expr, env *Environment) (Expr, error) {
	if len(args) == 0 {
		return boolean(true), nil
	}

	for _, arg := range args[:len(args)-1] {
		val, err := EvalFull(arg, env)
		if err != nil {
			return nil, err
		}

		if isFalse(val) {
			return boolean(false), nil
		}
	}

	return &TailCall{expr: args[len(args)-1], env: env}, nil
}

func evalOr(args []Expr, env *Environment) (Expr, error) {
	if len(args) == 0 {
		return boolean(false), nil
	}

	for _, arg := range args[:len(args)-1] {
		val, err := EvalFull(arg, env)
		if err != nil {
			return nil, err
		}

		if !isFalse(val) {
			return val, nil
		}
	}

	return &TailCall{expr: args[len(args)-1], env: env}, nil
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

	varSlice, err := toSlice("do", args[0])
	if err != nil {
		return nil, err
	}

	type spec struct {
		name string
		step Expr // nil means no step; variable keeps its value
	}
	specs := make([]spec, len(varSlice))
	loopEnv := env.Extend()

	for i, el := range varSlice {
		clauseSlice, err := toSlice("do", el)
		if err != nil || len(clauseSlice) < 2 || len(clauseSlice) > 3 {
			return nil, fmt.Errorf(
				"do: variable spec must be (var init) or (var init step), got %s",
				el.String(),
			)
		}

		sym, ok := clauseSlice[0].(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf(
				"do: variable name must be a symbol, got %s",
				clauseSlice[0].String(),
			)
		}

		init, err := EvalFull(clauseSlice[1], env)
		if err != nil {
			return nil, err
		}

		loopEnv.Bind(sym.val, init)

		var step Expr
		if len(clauseSlice) == 3 {
			step = clauseSlice[2]
		}

		specs[i] = spec{name: sym.val, step: step}
	}

	termSlice, err := toSlice("do", args[1])
	if err != nil || len(termSlice) == 0 {
		return nil, fmt.Errorf("do: termination clause must be a non-empty list")
	}
	testExpr := termSlice[0]
	resultExprs := termSlice[1:]

	commands := args[2:]

	for {
		testVal, err := EvalFull(testExpr, loopEnv)
		if err != nil {
			return nil, err
		}

		if !isFalse(testVal) {
			if len(resultExprs) == 0 {
				return Void(), nil
			}
			return evalBody(resultExprs, loopEnv)
		}

		for _, cmd := range commands {
			if _, err := EvalFull(cmd, loopEnv); err != nil {
				return nil, err
			}
		}

		next := make([]Expr, len(specs))
		for i, s := range specs {
			if s.step != nil {
				val, err := EvalFull(s.step, loopEnv)
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
