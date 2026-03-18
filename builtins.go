package glerp

import (
	"fmt"
	"maps"
	"math"
	"os"
	"strings"
)

// BuiltinFn is the signature for a Go-implemented Scheme procedure. Arguments
// are already evaluated before the function is called.
type BuiltinFn func([]Expr) (Expr, error)

// StandardBuiltins returns the default set of built-in procedures, including
// arithmetic, list operations, and I/O. The returned map is a fresh copy —
// callers may modify it on an EnvironmentConfig before calling NewEnvironment.
//
// Go-backed libraries (e.g. time) are not included here; they are registered
// as Libraries on the config and accessed with (import :go/time) from Scheme.
func StandardBuiltins() map[string]BuiltinFn {
	m := map[string]BuiltinFn{
		"+":                         builtinAdd,
		"-":                         builtinSub,
		"*":                         builtinMul,
		"/":                         builtinDiv,
		"<":                         builtinLess,
		">":                         builtinGreater,
		"<=":                        builtinLessEq,
		">=":                        builtinGreaterEq,
		"=":                         builtinNumEq,
		"not":                       builtinNot,
		"car":                       builtinCar,
		"cdr":                       builtinCdr,
		"cons":                      builtinCons,
		"null?":                     typePred("null?", func(e Expr) bool { l, ok := e.(*ListExpr); return ok && len(l.elements) == 0 }),
		"pair?":                     typePred("pair?", func(e Expr) bool { l, ok := e.(*ListExpr); return ok && len(l.elements) > 0 }),
		"list?":                     typePred("list?", func(e Expr) bool { _, ok := e.(*ListExpr); return ok }),
		"number?":                   typePred("number?", func(e Expr) bool { _, ok := e.(*NumberExpr); return ok }),
		"string?":                   typePred("string?", func(e Expr) bool { _, ok := e.(*StringExpr); return ok }),
		"boolean?":                  typePred("boolean?", func(e Expr) bool { _, ok := e.(*BoolExpr); return ok }),
		"symbol?":                   typePred("symbol?", func(e Expr) bool { _, ok := e.(*SymbolExpr); return ok }),
		"procedure?":                typePred("procedure?", func(e Expr) bool { _, okL := e.(*LambdaExpr); _, okB := e.(*BuiltinExpr); return okL || okB }),
		"eq?":                       builtinEq,
		"equal?":                    builtinEqual,
		"modulo":                    builtinModulo,
		"remainder":                 builtinRemainder,
		"list":                      builtinList,
		"display":                   builtinDisplay,
		"display-ln":                builtinDisplayLn,
		"newline":                   builtinNewline,
		"values":                    builtinValues,
		"string-append":             builtinStringAppend,
		"->string":                  builtinToString,
		"get-environment-variable":  builtinGetEnvVar,
		"get-environment-variables": builtinGetEnvVars,
		"vector":                    builtinVector,
		"make-vector":               builtinMakeVector,
		"vector-ref":                builtinVectorRef,
		"vector-set!":               builtinVectorSet,
		"vector-length":             builtinVectorLength,
		"vector?":                   typePred("vector?", func(e Expr) bool { _, ok := e.(*VectorExpr); return ok }),
		"vector->list":              builtinVectorToList,
		"list->vector":              builtinListToVector,
		"vector-fill!":              builtinVectorFill,
	}

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

func typePred(name string, pred func(Expr) bool) BuiltinFn {
	return func(args []Expr) (Expr, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("%s: expected 1 argument, got %d", name, len(args))
		}

		return boolean(pred(args[0])), nil
	}
}

func builtinEq(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("eq?: expected 2 arguments, got %d", len(args))
	}

	return boolean(eqv(args[0], args[1])), nil
}

func deepEqual(a, b Expr) bool {
	la, okA := a.(*ListExpr)
	lb, okB := b.(*ListExpr)

	if okA && okB {
		if len(la.elements) != len(lb.elements) {
			return false
		}
		for i := range la.elements {
			if !deepEqual(la.elements[i], lb.elements[i]) {
				return false
			}
		}

		return true
	}

	va, okVA := a.(*VectorExpr)
	vb, okVB := b.(*VectorExpr)

	if okVA && okVB {
		if len(va.elements) != len(vb.elements) {
			return false
		}
		for i := range va.elements {
			if !deepEqual(va.elements[i], vb.elements[i]) {
				return false
			}
		}

		return true
	}

	return eqv(a, b)
}

func builtinEqual(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("equal?: expected 2 arguments, got %d", len(args))
	}

	return boolean(deepEqual(args[0], args[1])), nil
}

func builtinModulo(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("modulo: expected 2 arguments, got %d", len(args))
	}

	a, err := toNum("modulo", args[0])
	if err != nil {
		return nil, err
	}

	b, err := toNum("modulo", args[1])
	if err != nil {
		return nil, err
	}

	if b == 0 {
		return nil, fmt.Errorf("modulo: division by zero")
	}

	r := math.Mod(a, b)
	if r != 0 && (r < 0) != (b < 0) {
		r += b
	}

	return num(r), nil
}

func builtinRemainder(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("remainder: expected 2 arguments, got %d", len(args))
	}

	a, err := toNum("remainder", args[0])
	if err != nil {
		return nil, err
	}

	b, err := toNum("remainder", args[1])
	if err != nil {
		return nil, err
	}

	if b == 0 {
		return nil, fmt.Errorf("remainder: division by zero")
	}

	return num(math.Mod(a, b)), nil
}

func builtinGetEnvVar(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("get-environment-variable: expected 1 argument, got %d", len(args))
	}

	s, ok := args[0].(*StringExpr)
	if !ok {
		return nil, fmt.Errorf("get-environment-variable: expected string, got %s", args[0].String())
	}

	val, found := os.LookupEnv(s.val)
	if !found {
		return boolean(false), nil
	}

	return &StringExpr{val: val}, nil
}

func builtinGetEnvVars(args []Expr) (Expr, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("get-environment-variables: expected 0 arguments, got %d", len(args))
	}

	environ := os.Environ()
	entries := make([]Expr, 0, len(environ))

	for _, entry := range environ {
		k, v, _ := strings.Cut(entry, "=")
		entries = append(entries, &ListExpr{
			elements: []Expr{
				&StringExpr{val: k},
				&StringExpr{val: v},
			},
		})
	}

	return &ListExpr{elements: entries}, nil
}

func builtinVector(args []Expr) (Expr, error) {
	elems := make([]Expr, len(args))
	copy(elems, args)

	return &VectorExpr{elements: elems}, nil
}

func builtinMakeVector(args []Expr) (Expr, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("make-vector: expected 1 or 2 arguments, got %d", len(args))
	}

	k, err := toNum("make-vector", args[0])
	if err != nil {
		return nil, err
	}

	n := int(k)
	if n < 0 {
		return nil, fmt.Errorf("make-vector: length must be non-negative, got %d", n)
	}

	var fill Expr = num(0)
	if len(args) == 2 {
		fill = args[1]
	}

	elems := make([]Expr, n)
	for i := range elems {
		elems[i] = fill
	}

	return &VectorExpr{elements: elems}, nil
}

func builtinVectorRef(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("vector-ref: expected 2 arguments, got %d", len(args))
	}

	vec, ok := args[0].(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("vector-ref: expected vector, got %s", args[0].String())
	}

	k, err := toNum("vector-ref", args[1])
	if err != nil {
		return nil, err
	}

	idx := int(k)
	if idx < 0 || idx >= len(vec.elements) {
		return nil, fmt.Errorf("vector-ref: index %d out of range for vector of length %d", idx, len(vec.elements))
	}

	return vec.elements[idx], nil
}

func builtinVectorSet(args []Expr) (Expr, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("vector-set!: expected 3 arguments, got %d", len(args))
	}

	vec, ok := args[0].(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("vector-set!: expected vector, got %s", args[0].String())
	}

	k, err := toNum("vector-set!", args[1])
	if err != nil {
		return nil, err
	}

	idx := int(k)
	if idx < 0 || idx >= len(vec.elements) {
		return nil, fmt.Errorf("vector-set!: index %d out of range for vector of length %d", idx, len(vec.elements))
	}

	vec.elements[idx] = args[2]

	return Void(), nil
}

func builtinVectorLength(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector-length: expected 1 argument, got %d", len(args))
	}

	vec, ok := args[0].(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("vector-length: expected vector, got %s", args[0].String())
	}

	return num(float64(len(vec.elements))), nil
}

func builtinVectorToList(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("vector->list: expected 1 argument, got %d", len(args))
	}

	vec, ok := args[0].(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("vector->list: expected vector, got %s", args[0].String())
	}

	elems := make([]Expr, len(vec.elements))
	copy(elems, vec.elements)

	return &ListExpr{elements: elems}, nil
}

func builtinListToVector(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("list->vector: expected 1 argument, got %d", len(args))
	}

	lst, ok := args[0].(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("list->vector: expected list, got %s", args[0].String())
	}

	elems := make([]Expr, len(lst.elements))
	copy(elems, lst.elements)

	return &VectorExpr{elements: elems}, nil
}

func builtinVectorFill(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("vector-fill!: expected 2 arguments, got %d", len(args))
	}

	vec, ok := args[0].(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("vector-fill!: expected vector, got %s", args[0].String())
	}

	for i := range vec.elements {
		vec.elements[i] = args[1]
	}

	return Void(), nil
}
