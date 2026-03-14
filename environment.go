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

// RegisterForm binds name to a custom special form in this environment.
// Unlike RegisterBuiltin / Bind with a BuiltinExpr, form handlers receive
// their arguments unevaluated and are free to control evaluation themselves.
func (e *Environment) RegisterForm(name string, fn func([]Expr, *Environment) (Expr, error)) {
	e.Bind(name, &FormExpr{name: name, fn: fn})
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

// NewEnvironment creates a root environment populated with standard builtins.
func NewEnvironment() *Environment {
	env := &Environment{vals: make(map[string]Expr)}
	for name, fn := range builtins {
		env.Bind(name, &BuiltinExpr{name: name, fn: fn})
	}
	env.RegisterForm("define-values", evalDefineValues)
	env.RegisterForm("case", evalCase)
	env.RegisterForm("begin", evalBegin)
	env.RegisterForm("cond", evalCond)
	env.RegisterForm("and", evalAnd)
	env.RegisterForm("or", evalOr)
	env.RegisterForm("import", evalImport)
	env.RegisterForm("export", evalExport)
	return env
}
