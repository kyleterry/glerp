package glerp_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

// writeTemp writes src to a temporary .scm file and returns its path.
func writeTemp(t *testing.T, src string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.scm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(src); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	return f.Name()
}

func TestLoad(t *testing.T) {
	is := is.New(t)

	path := writeTemp(t, `
(define host "localhost")
(define port 8080)
(define debug #t)
(define tags '("web" "api"))
(define ratio 1.5)
`)
	cfg, err := glerp.Load(path)
	is.NoErr(err)

	host, err := cfg.String("host")
	is.NoErr(err)
	is.Equal(host, "localhost")

	port, err := cfg.Int("port")
	is.NoErr(err)
	is.Equal(port, 8080)

	debug, err := cfg.Bool("debug")
	is.NoErr(err)
	is.Equal(debug, true)

	tags, err := cfg.Strings("tags")
	is.NoErr(err)
	is.Equal(tags, []string{"web", "api"})

	ratio, err := cfg.Float("ratio")
	is.NoErr(err)
	is.Equal(ratio, 1.5)
}

func TestLoadMissingFile(t *testing.T) {
	is := is.New(t)
	_, err := glerp.Load(filepath.Join(t.TempDir(), "missing.scm"))
	is.True(err != nil)
}

func TestConfigTypeMismatches(t *testing.T) {
	is := is.New(t)

	path := writeTemp(t, `
(define n 42)
(define s "hello")
(define b #t)
(define lst '(1 2))
`)
	cfg, err := glerp.Load(path)
	is.NoErr(err)

	_, err = cfg.String("n")
	is.True(err != nil)

	_, err = cfg.Int("s")
	is.True(err != nil)

	_, err = cfg.Bool("n")
	is.True(err != nil)

	_, err = cfg.Float("s")
	is.True(err != nil)

	_, err = cfg.Strings("n")
	is.True(err != nil)
}

func TestConfigUnbound(t *testing.T) {
	is := is.New(t)

	cfg, err := glerp.Load(writeTemp(t, `(define x 1)`))
	is.NoErr(err)

	_, err = cfg.String("missing")
	is.True(err != nil)

	_, err = cfg.Int("missing")
	is.True(err != nil)

	_, err = cfg.Bool("missing")
	is.True(err != nil)

	_, err = cfg.Float("missing")
	is.True(err != nil)

	_, err = cfg.Strings("missing")
	is.True(err != nil)

	_, err = cfg.List("missing")
	is.True(err != nil)
}

func TestConfigIntRejectsFloat(t *testing.T) {
	is := is.New(t)

	cfg, err := glerp.Load(writeTemp(t, `(define x 3.14)`))
	is.NoErr(err)

	_, err = cfg.Int("x")
	is.True(err != nil)
}

func TestConfigStringsList(t *testing.T) {
	is := is.New(t)

	cfg, err := glerp.Load(writeTemp(t, `(define mixed '("a" 1))`))
	is.NoErr(err)

	_, err = cfg.Strings("mixed")
	is.True(err != nil)
}

func TestEvalFile(t *testing.T) {
	is := is.New(t)

	path := writeTemp(t, `(define result (+ 1 2))`)
	env := glerp.NewEnvironment(glerp.DefaultConfig())
	err := glerp.EvalFile(path, env)
	is.NoErr(err)

	cfg := glerp.NewConfig(env)
	n, err := cfg.Int("result")
	is.NoErr(err)
	is.Equal(n, 3)
}

func TestEvalFileMissing(t *testing.T) {
	is := is.New(t)

	env := glerp.NewEnvironment(glerp.DefaultConfig())
	err := glerp.EvalFile(filepath.Join(t.TempDir(), "missing.scm"), env)
	is.True(err != nil)
}

func TestEvalFilePreRegisteredForm(t *testing.T) {
	is := is.New(t)

	type server struct {
		host string
		port int
	}
	var got server

	ecfg := glerp.DefaultConfig()
	ecfg.Forms["server"] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		host, err := args[0].Eval(env)
		if err != nil {
			return nil, err
		}
		port, err := args[1].Eval(env)
		if err != nil {
			return nil, err
		}
		got.host = host.(*glerp.StringExpr).Value()
		got.port = int(port.(*glerp.NumberExpr).Value())
		return glerp.Void(), nil
	}

	path := writeTemp(t, `(server "0.0.0.0" 9090)`)
	env := glerp.NewEnvironment(ecfg)
	err := glerp.EvalFile(path, env)
	is.NoErr(err)
	is.Equal(got.host, "0.0.0.0")
	is.Equal(got.port, 9090)
}
