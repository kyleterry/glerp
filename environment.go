package glerp

import (
	"fmt"
	"io/fs"
)

// Environment is a lexically-scoped map of variable bindings. Each environment
// optionally references an outer (enclosing) scope, forming the chain used for
// variable lookup and lambda closures.
type Environment struct {
	vals    map[string]Expr
	outer   *Environment
	exports []string
	reg     *registry
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

// StandardForms returns the default set of special forms. The returned map is
// a fresh copy — callers may add, remove, or replace entries before passing it
// to an EnvironmentConfig.
func StandardForms() map[string]FormFn {
	return map[string]FormFn{
		"define":     evalDefine,
		"lambda":     evalLambda,
		"if":         evalIf,
		"let":        evalLet,
		"let*":       evalLetStar,
		"set!":       evalSetBang,
		"quote":      evalQuote,
		"quasiquote": evalQuasiquote,
		"unquote":    func(_ []Expr, _ *Environment) (Expr, error) { return nil, fmt.Errorf("unquote: not inside quasiquote") },
		"unquote-splicing": func(_ []Expr, _ *Environment) (Expr, error) {
			return nil, fmt.Errorf("unquote-splicing: not inside quasiquote")
		},
		"define-values": evalDefineValues,
		"case":          evalCase,
		"do":            evalDo,
		"begin":         evalBegin,
		"cond":          evalCond,
		"and":           evalAnd,
		"or":            evalOr,
		"import":        evalImport,
		"export":        evalExport,
		"define-syntax": evalDefineSyntax,
		"syntax-rules":  evalSyntaxRules,
		"syntax-case":   evalSyntaxCase,
		"syntax":        evalSyntax,
		"with-syntax":   evalWithSyntax,
		"quasisyntax":   evalQuasisyntax,
		"unsyntax": func(_ []Expr, _ *Environment) (Expr, error) {
			return nil, fmt.Errorf("unsyntax: not inside quasisyntax")
		},
		"unsyntax-splicing": func(_ []Expr, _ *Environment) (Expr, error) {
			return nil, fmt.Errorf("unsyntax-splicing: not inside quasisyntax")
		},
	}
}

// Extend creates a child environment with this environment as the outer scope.
func (e *Environment) Extend() *Environment {
	return &Environment{
		vals:  make(map[string]Expr),
		outer: e,
		reg:   e.reg,
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

// AllNames returns every name visible from this scope, including those
// inherited from outer scopes. Inner bindings shadow outer ones; each name
// appears at most once. Useful for tab completion.
func (e *Environment) AllNames() []string {
	seen := make(map[string]bool)
	var names []string

	for cur := e; cur != nil; cur = cur.outer {
		for name := range cur.vals {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
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

// Prelude is a named filesystem containing Scheme source files that are
// evaluated into every new environment. The entry point is Name.scm in the
// root of FS. Files within FS can import each other via (import :Name/path).
type Prelude struct {
	Name string
	FS   fs.FS
}

// Library is a named set of importable Scheme or Go-backed procedures.
// Set FS for a Scheme-file library importable as (import :Prefix/path),
// or set Builtins for a Go-backed library importable as (import :Prefix).
type Library struct {
	Prefix   string
	FS       fs.FS
	Builtins map[string]BuiltinFn
}

// EnvironmentConfig holds all the settings for creating a new environment.
type EnvironmentConfig struct {
	Builtins  map[string]BuiltinFn
	Forms     map[string]FormFn
	Preludes  []Prelude
	Libraries []Library
}

// DefaultConfig returns an EnvironmentConfig populated with the standard
// builtins, forms, preludes, and libraries. The returned config is a fresh
// copy — callers may modify any field before passing it to NewEnvironment.
func DefaultConfig() *EnvironmentConfig {
	return &EnvironmentConfig{
		Builtins:  StandardBuiltins(),
		Forms:     StandardForms(),
		Preludes:  StandardPreludes(),
		Libraries: StandardLibraries(),
	}
}

// NewEnvironment creates a root environment from the given config. It
// installs Go-level builtins and forms, registers libraries, then evaluates
// preludes. Use DefaultConfig() for the standard setup.
func NewEnvironment(cfg *EnvironmentConfig) *Environment {
	reg := &registry{
		builtins: cfg.Builtins,
		forms:    cfg.Forms,
	}

	reg.libs = append(reg.libs, cfg.Libraries...)

	env := newBaseEnvironment(cfg.Builtins, cfg.Forms, reg)
	loadPreludes(env, cfg.Preludes)

	return env
}

// newBaseEnvironment creates a root environment populated with the given
// builtins and special forms, without loading preludes.
func newBaseEnvironment(builtins map[string]BuiltinFn, forms map[string]FormFn, reg *registry) *Environment {
	env := &Environment{vals: make(map[string]Expr), reg: reg}

	for name, fn := range builtins {
		env.Bind(name, &BuiltinExpr{name: name, fn: fn})
	}

	for name, fn := range forms {
		env.RegisterForm(name, fn)
	}

	return env
}
