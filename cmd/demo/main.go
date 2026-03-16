// Command demo shows glerp used as an embedded configuration DSL with
// custom special forms.
//
// config.scm is written as a proper DSL: (server ...), (database ...),
// (GET ...), (features ...) are all custom forms registered from Go before
// the file is evaluated. Each form receives its arguments unevaluated and
// controls its own semantics — values inside forms can be arbitrary Scheme
// expressions, including computed values and conditionals.
package main

import (
	"fmt"
	"log"
	"math"
	"strings"

	"go.e64ec.com/glerp"
)

type ServerConfig struct {
	AppName      string
	Version      string
	Host         string
	Port         int
	ReadTimeout  int
	WriteTimeout int
	WorkerCount  int
}

type DBConfig struct {
	Host     string
	Port     int
	Name     string
	PoolSize int
}

type Route struct {
	Method  string
	Path    string
	Handler string
}

type AppConfig struct {
	Server   ServerConfig
	DB       DBConfig
	Routes   []Route
	Features []string
}

// subform extracts the key name and value expressions from a (key val...)
// sub-form. Used by forms that accept keyword-style children.
func subform(e glerp.Expr) (key string, vals []glerp.Expr, err error) {
	lst, ok := e.(*glerp.ListExpr)
	if !ok {
		return "", nil, fmt.Errorf("expected (key value...), got %s", e.String())
	}
	elems := lst.Elements()
	if len(elems) == 0 {
		return "", nil, fmt.Errorf("empty sub-form")
	}
	// The key is the head symbol; use String() to get its name.
	return elems[0].String(), elems[1:], nil
}

// evalStr evaluates e and asserts the result is a string.
func evalStr(e glerp.Expr, env *glerp.Environment) (string, error) {
	v, err := e.Eval(env)
	if err != nil {
		return "", err
	}
	s, ok := v.(*glerp.StringExpr)
	if !ok {
		return "", fmt.Errorf("expected string, got %s", v.String())
	}
	return s.Value(), nil
}

// evalInt evaluates e and asserts the result is an integer.
func evalInt(e glerp.Expr, env *glerp.Environment) (int, error) {
	v, err := e.Eval(env)
	if err != nil {
		return 0, err
	}
	n, ok := v.(*glerp.NumberExpr)
	if !ok {
		return 0, fmt.Errorf("expected number, got %s", v.String())
	}
	f := n.Value()
	if f != math.Trunc(f) {
		return 0, fmt.Errorf("expected integer, got %s", v.String())
	}
	return int(f), nil
}

// registerForms installs the custom DSL forms into env. Each form is a
// closure that populates the appropriate field of cfg when evaluated.
func registerForms(env *glerp.Environment, cfg *AppConfig) {
	// (server (key value) ...) — populates cfg.Server
	env.RegisterForm("server", func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		for _, arg := range args {
			key, vals, err := subform(arg)
			if err != nil {
				return nil, fmt.Errorf("server: %w", err)
			}
			if len(vals) != 1 {
				return nil, fmt.Errorf("server/%s: expected exactly 1 value", key)
			}
			switch key {
			case "host":
				cfg.Server.Host, err = evalStr(vals[0], env)
			case "port":
				cfg.Server.Port, err = evalInt(vals[0], env)
			case "workers":
				cfg.Server.WorkerCount, err = evalInt(vals[0], env)
			case "read-timeout":
				cfg.Server.ReadTimeout, err = evalInt(vals[0], env)
			case "write-timeout":
				cfg.Server.WriteTimeout, err = evalInt(vals[0], env)
			default:
				return nil, fmt.Errorf("server: unknown key %q", key)
			}
			if err != nil {
				return nil, fmt.Errorf("server/%s: %w", key, err)
			}
		}
		return glerp.Void(), nil
	})

	// (database (key value) ...) — populates cfg.DB
	env.RegisterForm("database", func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		for _, arg := range args {
			key, vals, err := subform(arg)
			if err != nil {
				return nil, fmt.Errorf("database: %w", err)
			}
			if len(vals) != 1 {
				return nil, fmt.Errorf("database/%s: expected exactly 1 value", key)
			}
			switch key {
			case "host":
				cfg.DB.Host, err = evalStr(vals[0], env)
			case "port":
				cfg.DB.Port, err = evalInt(vals[0], env)
			case "name":
				cfg.DB.Name, err = evalStr(vals[0], env)
			case "pool-size":
				cfg.DB.PoolSize, err = evalInt(vals[0], env)
			default:
				return nil, fmt.Errorf("database: unknown key %q", key)
			}
			if err != nil {
				return nil, fmt.Errorf("database/%s: %w", key, err)
			}
		}
		return glerp.Void(), nil
	})

	// HTTP method forms: (GET path handler), (POST path handler), …
	// Each one appends a Route to cfg.Routes.
	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"} {
		m := method
		env.RegisterForm(m, func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("%s: expected (path handler), got %d args", m, len(args))
			}
			path, err := evalStr(args[0], env)
			if err != nil {
				return nil, fmt.Errorf("%s path: %w", m, err)
			}
			handler, err := evalStr(args[1], env)
			if err != nil {
				return nil, fmt.Errorf("%s handler: %w", m, err)
			}
			cfg.Routes = append(cfg.Routes, Route{Method: m, Path: path, Handler: handler})
			return glerp.Void(), nil
		})
	}

	// (routes body...) — evaluates each child route form in order.
	env.RegisterForm("routes", func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		for _, arg := range args {
			if _, err := arg.Eval(env); err != nil {
				return nil, err
			}
		}
		return glerp.Void(), nil
	})

	// (features name ...) — each argument is a bare symbol used as the
	// feature name directly, without environment lookup.
	env.RegisterForm("features", func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		for _, arg := range args {
			argVal, err := arg.Eval(env)
			if err != nil {
				return nil, err
			}

			cfg.Features = append(cfg.Features, argVal.String())
		}
		return glerp.Void(), nil
	})

	// (app name version sub-form...) — top-level DSL entry point.
	// Sets name/version then evaluates the sub-forms (server, database, …).
	env.RegisterForm("app", func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
		if len(args) < 2 {
			return nil, fmt.Errorf("app: expected (app name version ...forms)")
		}
		name, err := evalStr(args[0], env)
		if err != nil {
			return nil, fmt.Errorf("app name: %w", err)
		}
		version, err := evalStr(args[1], env)
		if err != nil {
			return nil, fmt.Errorf("app version: %w", err)
		}
		cfg.Server.AppName = name
		cfg.Server.Version = version

		// Evaluate each sub-form. Because (server ...), (database ...),
		// (routes ...), and (features ...) are all registered forms in env,
		// calling Eval() on each dispatches to their handlers above.
		for _, sub := range args[2:] {
			if _, err := sub.Eval(env); err != nil {
				return nil, err
			}
		}
		return glerp.Void(), nil
	})
}

func printConfig(app *AppConfig) {
	const width = 52
	bar := strings.Repeat("─", width)

	fmt.Printf("\n┌%s┐\n", bar)
	fmt.Printf("│  %-*s│\n", width-2, fmt.Sprintf("%s  v%s", app.Server.AppName, app.Server.Version))
	fmt.Printf("└%s┘\n\n", bar)

	fmt.Println("Server")
	fmt.Printf("  address       %s:%d\n", app.Server.Host, app.Server.Port)
	fmt.Printf("  workers       %d\n", app.Server.WorkerCount)
	fmt.Printf("  read timeout  %ds\n", app.Server.ReadTimeout)
	fmt.Printf("  write timeout %ds\n\n", app.Server.WriteTimeout)

	fmt.Println("Database")
	fmt.Printf("  host      %s:%d\n", app.DB.Host, app.DB.Port)
	fmt.Printf("  name      %s\n", app.DB.Name)
	fmt.Printf("  pool size %d\n\n", app.DB.PoolSize)

	fmt.Println("Routes")
	for _, r := range app.Routes {
		fmt.Printf("  %-8s %-24s → %s\n", r.Method, r.Path, r.Handler)
	}
	fmt.Println()

	fmt.Println("Features")
	for _, f := range app.Features {
		fmt.Printf("  ✓ %s\n", f)
	}
	fmt.Println()
}

func main() {
	cfg := &AppConfig{}

	// Build an environment with standard builtins, then layer our DSL forms
	// on top. EvalFile evaluates config.scm into this prepared environment.
	env := glerp.NewEnvironment(glerp.DefaultConfig())
	registerForms(env, cfg)

	if err := glerp.EvalFile("config.scm", env); err != nil {
		log.Fatalf("config error: %v", err)
	}

	printConfig(cfg)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Listening on %s\n", addr)
}
