package glerp_test

import (
	"testing"
	"testing/fstest"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestCustomLibrary(t *testing.T) {
	is := is.New(t)

	testFS := fstest.MapFS{
		"utils.scm": &fstest.MapFile{
			Data: []byte(`(define (square x) (* x x))`),
		},
		"math/extra.scm": &fstest.MapFile{
			Data: []byte(`(define (cube x) (* x x x))`),
		},
	}

	cfg := glerp.DefaultConfig()
	cfg.Libraries = append(cfg.Libraries, glerp.Library{
		Prefix: "testpkg",
		FS:     testFS,
	})

	tests := []struct {
		name string
		src  string
		want string
	}{
		{"top-level lib", "(import :testpkg/utils) (square 5)", "25"},
		{"nested lib path", "(import :testpkg/math/extra) (cube 3)", "27"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(cfg)
			results, err := glerp.Eval(tt.src, env)
			is.NoErr(err)
			is.Equal(results[len(results)-1].String(), tt.want)
		})
	}
}
