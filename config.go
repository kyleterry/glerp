package glerp

import (
	"fmt"
	"math"
	"os"

	"go.e64ec.com/glerp/token"
)

// Config wraps an Environment and provides typed value extraction for use
// as an embedded configuration DSL.
type Config struct {
	env *Environment
}

// NewConfig wraps env for typed value extraction.
func NewConfig(env *Environment) *Config {
	return &Config{env: env}
}

// EvalFile evaluates all top-level expressions in the file at path within env.
// Use this when you need to register custom forms before loading the config:
//
//	env := glerp.NewEnvironment(glerp.StandardBuiltins(), glerp.StandardForms())
//	env.RegisterForm("server", ...)
//	if err := glerp.EvalFile("config.scm", env); err != nil { ... }
func EvalFile(path string, env *Environment) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck

	lexer, err := token.NewLexer(f)
	if err != nil {
		return fmt.Errorf("lexer: %w", err)
	}

	p := NewParser(lexer)
	exprs, err := p.Run()
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	for _, expr := range exprs {
		if _, err := expr.Eval(env); err != nil {
			return fmt.Errorf("eval: %w", err)
		}
	}
	return nil
}

// Load reads and evaluates the glerp config file at path in a fresh
// environment, returning a Config ready for typed extraction.
// For form-based DSLs that need pre-registered forms, use EvalFile instead.
func Load(path string) (*Config, error) {
	env := NewEnvironment(StandardBuiltins(), StandardForms())
	if err := EvalFile(path, env); err != nil {
		return nil, err
	}
	return NewConfig(env), nil
}

// String returns the string value bound to name.
func (c *Config) String(name string) (string, error) {
	expr, err := c.env.Find(name)
	if err != nil {
		return "", err
	}
	s, ok := expr.(*StringExpr)
	if !ok {
		return "", fmt.Errorf("config: %s is not a string (got %s)", name, expr.String())
	}
	return s.val, nil
}

// Int returns the integer value bound to name.
func (c *Config) Int(name string) (int, error) {
	expr, err := c.env.Find(name)
	if err != nil {
		return 0, err
	}
	n, ok := expr.(*NumberExpr)
	if !ok {
		return 0, fmt.Errorf("config: %s is not a number (got %s)", name, expr.String())
	}
	if n.val != math.Trunc(n.val) {
		return 0, fmt.Errorf("config: %s is not an integer (%s)", name, expr.String())
	}
	return int(n.val), nil
}

// Float returns the float64 value bound to name.
func (c *Config) Float(name string) (float64, error) {
	expr, err := c.env.Find(name)
	if err != nil {
		return 0, err
	}
	n, ok := expr.(*NumberExpr)
	if !ok {
		return 0, fmt.Errorf("config: %s is not a number (got %s)", name, expr.String())
	}
	return n.val, nil
}

// Bool returns the boolean value bound to name.
func (c *Config) Bool(name string) (bool, error) {
	expr, err := c.env.Find(name)
	if err != nil {
		return false, err
	}
	b, ok := expr.(*BoolExpr)
	if !ok {
		return false, fmt.Errorf("config: %s is not a boolean (got %s)", name, expr.String())
	}
	return b.val, nil
}

// List returns the list expression bound to name.
func (c *Config) List(name string) (*ListExpr, error) {
	expr, err := c.env.Find(name)
	if err != nil {
		return nil, err
	}
	lst, ok := expr.(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("config: %s is not a list (got %s)", name, expr.String())
	}
	return lst, nil
}

// Strings returns a slice of string values from the list bound to name.
func (c *Config) Strings(name string) ([]string, error) {
	lst, err := c.List(name)
	if err != nil {
		return nil, err
	}
	result := make([]string, len(lst.elements))
	for i, el := range lst.elements {
		s, ok := el.(*StringExpr)
		if !ok {
			return nil, fmt.Errorf("config: %s[%d] is not a string (got %s)", name, i, el.String())
		}
		result[i] = s.val
	}
	return result, nil
}
