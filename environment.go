package glerp

import "fmt"

// Environment is a lexically-scoped map of variable bindings. Each environment
// optionally references an outer (enclosing) scope, forming the chain used for
// variable lookup and lambda closures.
type Environment struct {
	vals    map[string]Expr
	outer   *Environment
	exports []string
}

// Bind creates or overwrites a binding in the current scope.
func (e *Environment) Bind(name string, val Expr) {
	e.vals[name] = val
}

// Find looks up name in the current scope, then outer scopes.
func (e *Environment) Find(name string) (Expr, error) {
	if v, ok := e.vals[name]; ok {
		return v, nil
	}
	if e.outer != nil {
		return e.outer.Find(name)
	}
	return nil, fmt.Errorf("unbound variable: %s", name)
}

// Set mutates an existing binding, searching parent scopes.
func (e *Environment) Set(name string, val Expr) error {
	if _, ok := e.vals[name]; ok {
		e.vals[name] = val
		return nil
	}
	if e.outer != nil {
		return e.outer.Set(name, val)
	}
	return fmt.Errorf("unbound variable: %s", name)
}

// FormFn is the signature for a special form handler. Arguments are passed
// unevaluated; the handler controls its own evaluation semantics.
type FormFn func([]Expr, *Environment) (Expr, error)

// RegisterForm binds name to a custom special form in this environment.
// Unlike Bind with a BuiltinExpr, form handlers receive their arguments
// unevaluated and are free to control evaluation themselves.
func (e *Environment) RegisterForm(name string, fn FormFn) {
	e.Bind(name, &FormExpr{name: name, fn: fn})
}

// StandardForms returns the default set of special forms (define-values, case,
// do, begin, cond, and, or, import, export). The returned map is a fresh copy
// — callers may add, remove, or replace entries before passing it to
// NewEnvironment.
func StandardForms() map[string]FormFn {
	return map[string]FormFn{
		"define-values": evalDefineValues,
		"case":          evalCase,
		"do":            evalDo,
		"begin":         evalBegin,
		"cond":          evalCond,
		"and":           evalAnd,
		"or":            evalOr,
		"import":        evalImport,
		"export":        evalExport,
	}
}

// Extend creates a child environment with this environment as the outer scope.
func (e *Environment) Extend() *Environment {
	return &Environment{
		vals:  make(map[string]Expr),
		outer: e,
	}
}

// Names returns the names bound directly in this scope, not inherited from
// outer scopes. Used by the library loader to enumerate a library's definitions.
func (e *Environment) Names() []string {
	names := make([]string, 0, len(e.vals))
	for name := range e.vals {
		names = append(names, name)
	}
	return names
}

// DeclareExports records the set of names this scope explicitly exports.
// Called by the (export ...) form inside a library file.
func (e *Environment) DeclareExports(names []string) {
	e.exports = names
}

// Exports returns the explicitly declared export list, or nil if the library
// never called (export ...) and therefore exports all its definitions.
func (e *Environment) Exports() []string {
	return e.exports
}

// NewEnvironment creates a root environment populated with the given builtins
// and special forms. Pass StandardBuiltins() and StandardForms() for the
// default set, or customised maps to restrict or extend the environment.
func NewEnvironment(builtins map[string]BuiltinFn, forms map[string]FormFn) *Environment {
	env := &Environment{vals: make(map[string]Expr)}
	for name, fn := range builtins {
		env.Bind(name, &BuiltinExpr{name: name, fn: fn})
	}
	for name, fn := range forms {
		env.RegisterForm(name, fn)
	}
	return env
}
