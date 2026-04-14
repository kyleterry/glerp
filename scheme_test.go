package glerp_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestSchemeFiles(t *testing.T) {
	files, err := filepath.Glob("scheme-tests/*.scm")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			is := is.New(t)
			cfg := glerp.DefaultConfig()

			// add a simple check builtin for the tests.
			// (check expr [message])
			cfg.Builtins["check"] = func(args []glerp.Expr) (glerp.Expr, error) {
				if len(args) < 1 {
					return nil, fmt.Errorf("check: expected at least 1 argument")
				}

				pass := true
				if b, ok := args[0].(*glerp.BoolExpr); ok {
					pass = b.Value()
				}

				if !pass {
					msg := "assertion failed"
					if len(args) > 1 {
						if s, ok := args[1].(*glerp.StringExpr); ok {
							msg = s.Value()
						} else {
							msg = args[1].String()
						}
					}
					t.Errorf("%s: %s (got %s)", file, msg, args[0].String())
				}

				return glerp.Void(), nil
			}

			env := glerp.NewEnvironment(cfg)
			err := glerp.EvalFile(file, env)
			is.NoErr(err)
		})
	}
}
