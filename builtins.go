package glerp

import (
	"fmt"
	"maps"
	"strings"
)

// BuiltinFn is the signature for a Go-implemented Scheme procedure. Arguments
// are already evaluated before the function is called.
type BuiltinFn func([]Expr) (Expr, error)

// StandardBuiltins returns the default set of built-in procedures, including
// arithmetic, list operations, I/O, and time utilities. The returned map is a
// fresh copy — callers may add, remove, or replace entries before passing it
// to NewEnvironment.
func StandardBuiltins() map[string]BuiltinFn {
	m := map[string]BuiltinFn{
		"+":             builtinAdd,
		"-":             builtinSub,
		"*":             builtinMul,
		"/":             builtinDiv,
		"<":             builtinLess,
		">":             builtinGreater,
		"<=":            builtinLessEq,
		">=":            builtinGreaterEq,
		"=":             builtinNumEq,
		"not":           builtinNot,
		"car":           builtinCar,
		"cdr":           builtinCdr,
		"cons":          builtinCons,
		"empty?":        builtinEmpty,
		"list":          builtinList,
		"display":       builtinDisplay,
		"display-ln":    builtinDisplayLn,
		"newline":       builtinNewline,
		"values":        builtinValues,
		"string-append": builtinStringAppend,
		"->string":      builtinToString,
	}
	maps.Copy(m, timeBuiltins())
	maps.Copy(m, cxrBuiltins())
	return m
}

// cxrBuiltins generates all caar/cadr/.../cddddr compositions (2–4 a/d letters).
// Each function applies car (a) or cdr (d) right-to-left, so cadr is
// equivalent to (car (cdr x)).
func cxrBuiltins() map[string]BuiltinFn {
	m := make(map[string]BuiltinFn)
	var add func(ops string)
	add = func(ops string) {
		name := "c" + ops + "r"
		m[name] = func(args []Expr) (Expr, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("%s: expected 1 argument, got %d", name, len(args))
			}
			result := args[0]
			for i := len(ops) - 1; i >= 0; i-- {
				var err error
				if ops[i] == 'a' {
					result, err = builtinCar([]Expr{result})
				} else {
					result, err = builtinCdr([]Expr{result})
				}
				if err != nil {
					return nil, fmt.Errorf("%s: %w", name, err)
				}
			}
			return result, nil
		}
		if len(ops) < 4 {
			add("a" + ops)
			add("d" + ops)
		}
	}
	add("aa")
	add("ad")
	add("da")
	add("dd")
	return m
}

func toNum(name string, e Expr) (float64, error) {
	n, ok := e.(*NumberExpr)
	if !ok {
		return 0, fmt.Errorf("%s: expected number, got %s", name, e.String())
	}
	return n.val, nil
}

func num(v float64) *NumberExpr { return &NumberExpr{val: v} }
func boolean(v bool) *BoolExpr  { return &BoolExpr{val: v} }

func builtinAdd(args []Expr) (Expr, error) {
	sum := 0.0
	for _, a := range args {
		n, err := toNum("+", a)
		if err != nil {
			return nil, err
		}
		sum += n
	}
	return num(sum), nil
}

func builtinSub(args []Expr) (Expr, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("-: requires at least 1 argument")
	}
	first, err := toNum("-", args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 1 {
		return num(-first), nil
	}
	for _, a := range args[1:] {
		n, err := toNum("-", a)
		if err != nil {
			return nil, err
		}
		first -= n
	}
	return num(first), nil
}

func builtinMul(args []Expr) (Expr, error) {
	product := 1.0
	for _, a := range args {
		n, err := toNum("*", a)
		if err != nil {
			return nil, err
		}
		product *= n
	}
	return num(product), nil
}

func builtinDiv(args []Expr) (Expr, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("/: requires at least 1 argument")
	}
	first, err := toNum("/", args[0])
	if err != nil {
		return nil, err
	}
	if len(args) == 1 {
		if first == 0 {
			return nil, fmt.Errorf("/: division by zero")
		}
		return num(1 / first), nil
	}
	for _, a := range args[1:] {
		n, err := toNum("/", a)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			return nil, fmt.Errorf("/: division by zero")
		}
		first /= n
	}
	return num(first), nil
}

func numCmp(name string, op func(a, b float64) bool) BuiltinFn {
	return func(args []Expr) (Expr, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("%s: expected 2 arguments, got %d", name, len(args))
		}
		a, err := toNum(name, args[0])
		if err != nil {
			return nil, err
		}
		b, err := toNum(name, args[1])
		if err != nil {
			return nil, err
		}
		return boolean(op(a, b)), nil
	}
}

var (
	builtinLess      = numCmp("<", func(a, b float64) bool { return a < b })
	builtinGreater   = numCmp(">", func(a, b float64) bool { return a > b })
	builtinLessEq    = numCmp("<=", func(a, b float64) bool { return a <= b })
	builtinGreaterEq = numCmp(">=", func(a, b float64) bool { return a >= b })
	builtinNumEq     = numCmp("=", func(a, b float64) bool { return a == b })
)

func builtinNot(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("not: expected 1 argument, got %d", len(args))
	}
	if b, ok := args[0].(*BoolExpr); ok && !b.val {
		return boolean(true), nil
	}
	return boolean(false), nil
}

func builtinCar(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("car: expected 1 argument, got %d", len(args))
	}
	lst, ok := args[0].(*ListExpr)
	if !ok || len(lst.elements) == 0 {
		return nil, fmt.Errorf("car: expected non-empty list, got %s", args[0].String())
	}
	return lst.elements[0], nil
}

func builtinCdr(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("cdr: expected 1 argument, got %d", len(args))
	}
	lst, ok := args[0].(*ListExpr)
	if !ok || len(lst.elements) == 0 {
		return nil, fmt.Errorf("cdr: expected non-empty list, got %s", args[0].String())
	}
	return &ListExpr{elements: lst.elements[1:]}, nil
}

func builtinCons(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("cons: expected 2 arguments, got %d", len(args))
	}
	lst, ok := args[1].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("cons: second argument must be a list, got %s", args[1].String())
	}
	elems := make([]Expr, 1+len(lst.elements))
	elems[0] = args[0]
	copy(elems[1:], lst.elements)
	return &ListExpr{elements: elems}, nil
}

func builtinEmpty(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("empty?: expected 1 argument, got %d", len(args))
	}
	lst, ok := args[0].(*ListExpr)
	return boolean(ok && len(lst.elements) == 0), nil
}

func builtinList(args []Expr) (Expr, error) {
	return &ListExpr{elements: args}, nil
}

func builtinDisplay(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("display: expected 1 argument, got %d", len(args))
	}
	// Strings display without surrounding quotes.
	if s, ok := args[0].(*StringExpr); ok {
		fmt.Print(s.val)
	} else {
		fmt.Print(args[0].String())
	}
	return Void(), nil
}

func builtinDisplayLn(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("display-ln: expected 1 argument, got %d", len(args))
	}
	// Strings display without surrounding quotes.
	if s, ok := args[0].(*StringExpr); ok {
		fmt.Println(s.val)
	} else {
		fmt.Println(args[0].String())
	}
	return Void(), nil
}

func builtinValues(args []Expr) (Expr, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return &ValuesExpr{vals: args}, nil
}

func builtinNewline(args []Expr) (Expr, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("newline: expected 0 arguments, got %d", len(args))
	}
	fmt.Println()
	return Void(), nil
}

func builtinStringAppend(args []Expr) (Expr, error) {
	var b strings.Builder
	for i, arg := range args {
		s, ok := arg.(*StringExpr)
		if !ok {
			return nil, fmt.Errorf("string-append: argument %d is not a string: %s", i+1, arg.String())
		}
		b.WriteString(s.val)
	}
	return &StringExpr{val: b.String()}, nil
}

func builtinToString(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("->string: expected 1 argument, got %d", len(args))
	}
	if s, ok := args[0].(*StringExpr); ok {
		return s, nil
	}
	return &StringExpr{val: args[0].String()}, nil
}
