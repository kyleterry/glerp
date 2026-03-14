package glerp

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed stdlib
var stdlibFS embed.FS

func evalImport(args []Expr, env *Environment) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("import: expected 1 argument, got %d", len(args))
	}
	spec, ok := args[0].(*ListExpr)
	if !ok || len(spec.elements) == 0 {
		return nil, fmt.Errorf("import: argument must be a non-empty list, e.g. (import (scheme list))")
	}
	parts := make([]string, len(spec.elements))
	for i, el := range spec.elements {
		sym, ok := el.(*SymbolExpr)
		if !ok {
			return nil, fmt.Errorf("import: module name components must be symbols, got %s", el.String())
		}
		parts[i] = sym.val
	}
	path := "stdlib/" + strings.Join(parts, "/") + ".scm"
	data, err := stdlibFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("import: no such module (%s)", strings.Join(parts, " "))
	}
	_, err = Eval(string(data), env)
	if err != nil {
		return nil, fmt.Errorf("import (%s): %w", strings.Join(parts, " "), err)
	}
	return Void(), nil
}
