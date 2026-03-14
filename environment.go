package glerp

import "fmt"

// Environment holds variable bindings and a reference to an enclosing scope.
type Environment struct {
	vals  map[string]Expr
	outer *Environment
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

// NewEnvironment creates a root environment populated with standard builtins.
func NewEnvironment() *Environment {
	env := &Environment{vals: make(map[string]Expr)}
	for name, fn := range builtins {
		env.Bind(name, &BuiltinExpr{name: name, fn: fn})
	}
	return env
}
