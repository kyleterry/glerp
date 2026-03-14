package glerp

import (
	"embed"
	"fmt"
	"os"
	"strings"
)

//go:embed stdlib
var stdlibFS embed.FS

// evalExport implements (export name ...) inside a library file.
// It declares the set of symbols this library makes available to importers.
// (export #t) explicitly exports all definitions (equivalent to no declaration).
// Libraries without an (export ...) declaration also export all their definitions.
func evalExport(args []Expr, env *Environment) (Expr, error) {
	if len(args) == 1 {
		if b, ok := args[0].(*BoolExpr); ok && b.val {
			return Void(), nil // #t: export all — leave exports as nil
		}
	}
	names := make([]string, len(args))
	for i, arg := range args {
		sym, ok := arg.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("export: expected symbol or #t, got %s", arg.String())
		}
		names[i] = sym.val
	}
	env.DeclareExports(names)
	return Void(), nil
}

// evalImport implements (import <spec> ...) where each spec is one of:
//
//	:scheme/list              — import all exports from the named stdlib library
//	./relative/path           — import all exports from a .scm file relative to CWD
//	(only <spec> name ...)    — import a named subset of a library's exports
func evalImport(args []Expr, env *Environment) (Expr, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("import: expected at least one import spec")
	}
	for _, arg := range args {
		if err := applyImportSpec(arg, env); err != nil {
			return nil, err
		}
	}
	return Void(), nil
}

func applyImportSpec(spec Expr, env *Environment) error {
	switch s := spec.(type) {
	case *SymbolExpr:
		return importAll(s.val, env)
	case *ListExpr:
		if len(s.elements) == 0 {
			return fmt.Errorf("import: empty import spec")
		}
		head, ok := s.elements[0].(*SymbolExpr)
		if !ok {
			return fmt.Errorf("import: modifier must be a symbol, got %s", s.elements[0].String())
		}
		switch head.val {
		case "only":
			if len(s.elements) < 3 {
				return fmt.Errorf("import: (only <lib> <name> ...) requires a library and at least one name")
			}
			libSym, ok := s.elements[1].(*SymbolExpr)
			if !ok {
				return fmt.Errorf("import: only: library spec must be a symbol, got %s", s.elements[1].String())
			}
			names := make([]string, len(s.elements)-2)
			for i, el := range s.elements[2:] {
				sym, ok := el.(*SymbolExpr)
				if !ok {
					return fmt.Errorf("import: only: names must be symbols, got %s", el.String())
				}
				names[i] = sym.val
			}
			return importOnly(libSym.val, names, env)
		default:
			return fmt.Errorf("import: unknown modifier %q (known: only)", head.val)
		}
	default:
		return fmt.Errorf("import: invalid spec %s", spec.String())
	}
}

func importAll(libSpec string, env *Environment) error {
	libEnv, err := loadLibEnv(libSpec)
	if err != nil {
		return err
	}
	for _, name := range exportedNames(libEnv) {
		val, _ := libEnv.Find(name)
		env.Bind(name, val)
	}
	return nil
}

func importOnly(libSpec string, names []string, env *Environment) error {
	libEnv, err := loadLibEnv(libSpec)
	if err != nil {
		return err
	}
	exported := make(map[string]bool)
	for _, n := range exportedNames(libEnv) {
		exported[n] = true
	}
	for _, name := range names {
		if !exported[name] {
			return fmt.Errorf("import: %s does not export %q", libSpec, name)
		}
		val, _ := libEnv.Find(name)
		env.Bind(name, val)
	}
	return nil
}

// loadLibEnv evaluates the library at libSpec in an isolated environment and
// returns that environment for the caller to inspect and selectively copy from.
func loadLibEnv(libSpec string) (*Environment, error) {
	data, err := readLibSource(libSpec)
	if err != nil {
		return nil, err
	}
	// Evaluate in a child of a fresh base so builtins are available but
	// user-defined names land only in libEnv (not mixed with builtins).
	libEnv := NewEnvironment().Extend()
	if _, err := Eval(string(data), libEnv); err != nil {
		return nil, fmt.Errorf("import %s: %w", libSpec, err)
	}
	return libEnv, nil
}

// exportedNames returns the names a library makes available to importers.
// If the library declared (export ...), only those names are returned.
// Otherwise every name defined in the library's own scope is returned.
func exportedNames(libEnv *Environment) []string {
	if exp := libEnv.Exports(); exp != nil {
		return exp
	}
	return libEnv.Names()
}

// readLibSource resolves a library spec to its source bytes.
//
//	:scheme/list   → embedded stdlib/scheme/list.scm
//	./my-utils     → ./my-utils.scm relative to CWD
func readLibSource(spec string) ([]byte, error) {
	if tail, ok := strings.CutPrefix(spec, ":"); ok {
		path := "stdlib/" + tail + ".scm"
		data, err := stdlibFS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("import: no such library %q", spec)
		}
		return data, nil
	}
	if strings.HasPrefix(spec, "./") || strings.HasPrefix(spec, "../") {
		data, err := os.ReadFile(spec + ".scm")
		if err != nil {
			return nil, fmt.Errorf("import: cannot read %s.scm: %w", spec, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("import: unrecognized path %q (use :lib/path or ./relative)", spec)
}
