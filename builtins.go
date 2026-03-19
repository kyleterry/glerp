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
		"length":                    builtinLength,
		"map":                       builtinMap,
		"apply":                     builtinApply,
		"list->vector":              builtinListToVector,
		"vector-fill!":              builtinVectorFill,
		"symbol->string":            builtinSymbolToString,
		"string->symbol":            builtinStringToSymbol,
		"gensym":                    builtinGensym,
		"datum->syntax":             builtinDatumToSyntax,
		"syntax->datum":             builtinSyntaxToDatum,
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

// checkArity returns an error if args does not have exactly n elements.
func checkArity(name string, args []Expr, n int) error {
	if len(args) != n {
		return fmt.Errorf("%s: expected %d argument(s), got %d", name, n, len(args))
	}

	return nil
}

func toNum(name string, e Expr) (float64, error) {
	n, ok := e.(*NumberExpr)
	if !ok {
		return 0, fmt.Errorf("%s: expected number, got %s", name, e.String())
	}

	return n.val, nil
}

func toList(name string, e Expr) (*ListExpr, error) {
	l, ok := e.(*ListExpr)
	if !ok {
		return nil, fmt.Errorf("%s: expected list, got %s", name, e.String())
	}

	return l, nil
}

func toVec(name string, e Expr) (*VectorExpr, error) {
	v, ok := e.(*VectorExpr)
	if !ok {
		return nil, fmt.Errorf("%s: expected vector, got %s", name, e.String())
	}

	return v, nil
}

func toStr(name string, e Expr) (string, error) {
	s, ok := e.(*StringExpr)
	if !ok {
		return "", fmt.Errorf("%s: expected string, got %s", name, e.String())
	}

	return s.val, nil
}

func toSym(name string, e Expr) (string, error) {
	s, ok := e.(*SymbolExpr)
	if !ok {
		return "", fmt.Errorf("%s: expected symbol, got %s", name, e.String())
	}

	return s.val, nil
}

// vecIndex extracts a VectorExpr from args[0] and a valid index from args[1].
func vecIndex(name string, args []Expr) (*VectorExpr, int, error) {
	vec, err := toVec(name, args[0])
	if err != nil {
		return nil, 0, err
	}

	k, err := toNum(name, args[1])
	if err != nil {
		return nil, 0, err
	}

	idx := int(k)
	if idx < 0 || idx >= len(vec.elements) {
		return nil, 0, fmt.Errorf("%s: index %d out of range for vector of length %d", name, idx, len(vec.elements))
	}

	return vec, idx, nil
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
	if err := checkArity("not", args, 1); err != nil {
		return nil, err
	}

	return boolean(isFalse(args[0])), nil
}

func builtinCar(args []Expr) (Expr, error) {
	if err := checkArity("car", args, 1); err != nil {
		return nil, err
	}

	lst, ok := args[0].(*ListExpr)
	if !ok || len(lst.elements) == 0 {
		return nil, fmt.Errorf("car: expected non-empty list, got %s", args[0].String())
	}

	return lst.elements[0], nil
}

func builtinCdr(args []Expr) (Expr, error) {
	if err := checkArity("cdr", args, 1); err != nil {
		return nil, err
	}

	lst, ok := args[0].(*ListExpr)
	if !ok || len(lst.elements) == 0 {
		return nil, fmt.Errorf("cdr: expected non-empty list, got %s", args[0].String())
	}

	return &ListExpr{elements: lst.elements[1:]}, nil
}

func builtinCons(args []Expr) (Expr, error) {
	if err := checkArity("cons", args, 2); err != nil {
		return nil, err
	}

	lst, err := toList("cons", args[1])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, 1+len(lst.elements))
	elems[0] = args[0]
	copy(elems[1:], lst.elements)

	return &ListExpr{elements: elems}, nil
}

func builtinList(args []Expr) (Expr, error) {
	return &ListExpr{elements: args}, nil
}

func builtinLength(args []Expr) (Expr, error) {
	if err := checkArity("length", args, 1); err != nil {
		return nil, err
	}

	list, err := toList("length", args[0])
	if err != nil {
		return nil, err
	}

	return num(float64(len(list.elements))), nil
}

func builtinMap(args []Expr) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("map: expected at least 2 arguments, got %d", len(args))
	}

	proc := args[0]
	list, err := toList("map", args[1])
	if err != nil {
		return nil, err
	}

	results := make([]Expr, len(list.elements))

	for i, elem := range list.elements {
		result, err := apply(proc, []Expr{elem})
		if err != nil {
			return nil, fmt.Errorf("map: %w", err)
		}

		results[i] = result
	}

	return &ListExpr{elements: results}, nil
}

func builtinApply(args []Expr) (Expr, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("apply: expected at least 2 arguments, got %d", len(args))
	}

	proc := args[0]

	list, err := toList("apply", args[len(args)-1])
	if err != nil {
		return nil, err
	}

	// Collect leading args + spread the final list
	allArgs := make([]Expr, 0, len(args)-2+len(list.elements))
	allArgs = append(allArgs, args[1:len(args)-1]...)
	allArgs = append(allArgs, list.elements...)

	return apply(proc, allArgs)
}


func displayValue(e Expr) string {
	if s, ok := e.(*StringExpr); ok {
		return s.val
	}

	return e.String()
}

func builtinDisplay(args []Expr) (Expr, error) {
	if err := checkArity("display", args, 1); err != nil {
		return nil, err
	}

	fmt.Print(displayValue(args[0]))

	return Void(), nil
}


func builtinValues(args []Expr) (Expr, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	return &ValuesExpr{vals: args}, nil
}

func builtinNewline(args []Expr) (Expr, error) {
	if err := checkArity("newline", args, 0); err != nil {
		return nil, err
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
	if err := checkArity("->string", args, 1); err != nil {
		return nil, err
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
	if err := checkArity("eq?", args, 2); err != nil {
		return nil, err
	}

	return boolean(eqv(args[0], args[1])), nil
}

func elemsEqual(a, b []Expr) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if !deepEqual(a[i], b[i]) {
			return false
		}
	}

	return true
}

func deepEqual(a, b Expr) bool {
	if la, ok := a.(*ListExpr); ok {
		if lb, ok := b.(*ListExpr); ok {
			return elemsEqual(la.elements, lb.elements)
		}
	}

	if va, ok := a.(*VectorExpr); ok {
		if vb, ok := b.(*VectorExpr); ok {
			return elemsEqual(va.elements, vb.elements)
		}
	}

	return eqv(a, b)
}

func builtinEqual(args []Expr) (Expr, error) {
	if err := checkArity("equal?", args, 2); err != nil {
		return nil, err
	}

	return boolean(deepEqual(args[0], args[1])), nil
}

// numPairOp extracts two numbers from a 2-arg call and applies fn.
func numPairOp(name string, args []Expr, fn func(a, b float64) (Expr, error)) (Expr, error) {
	if err := checkArity(name, args, 2); err != nil {
		return nil, err
	}

	a, err := toNum(name, args[0])
	if err != nil {
		return nil, err
	}

	b, err := toNum(name, args[1])
	if err != nil {
		return nil, err
	}

	return fn(a, b)
}

func builtinModulo(args []Expr) (Expr, error) {
	return numPairOp("modulo", args, func(a, b float64) (Expr, error) {
		if b == 0 {
			return nil, fmt.Errorf("modulo: division by zero")
		}

		r := math.Mod(a, b)
		if r != 0 && (r < 0) != (b < 0) {
			r += b
		}

		return num(r), nil
	})
}

func builtinRemainder(args []Expr) (Expr, error) {
	return numPairOp("remainder", args, func(a, b float64) (Expr, error) {
		if b == 0 {
			return nil, fmt.Errorf("remainder: division by zero")
		}

		return num(math.Mod(a, b)), nil
	})
}

func builtinGetEnvVar(args []Expr) (Expr, error) {
	if err := checkArity("get-environment-variable", args, 1); err != nil {
		return nil, err
	}

	s, err := toStr("get-environment-variable", args[0])
	if err != nil {
		return nil, err
	}

	val, found := os.LookupEnv(s)
	if !found {
		return boolean(false), nil
	}

	return &StringExpr{val: val}, nil
}

func builtinGetEnvVars(args []Expr) (Expr, error) {
	if err := checkArity("get-environment-variables", args, 0); err != nil {
		return nil, err
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
	if err := checkArity("vector-ref", args, 2); err != nil {
		return nil, err
	}

	vec, idx, err := vecIndex("vector-ref", args)
	if err != nil {
		return nil, err
	}

	return vec.elements[idx], nil
}

func builtinVectorSet(args []Expr) (Expr, error) {
	if err := checkArity("vector-set!", args, 3); err != nil {
		return nil, err
	}

	vec, idx, err := vecIndex("vector-set!", args)
	if err != nil {
		return nil, err
	}

	vec.elements[idx] = args[2]

	return Void(), nil
}

func builtinVectorLength(args []Expr) (Expr, error) {
	if err := checkArity("vector-length", args, 1); err != nil {
		return nil, err
	}

	vec, err := toVec("vector-length", args[0])
	if err != nil {
		return nil, err
	}

	return num(float64(len(vec.elements))), nil
}

func builtinVectorToList(args []Expr) (Expr, error) {
	if err := checkArity("vector->list", args, 1); err != nil {
		return nil, err
	}

	vec, err := toVec("vector->list", args[0])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, len(vec.elements))
	copy(elems, vec.elements)

	return &ListExpr{elements: elems}, nil
}

func builtinListToVector(args []Expr) (Expr, error) {
	if err := checkArity("list->vector", args, 1); err != nil {
		return nil, err
	}

	lst, err := toList("list->vector", args[0])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, len(lst.elements))
	copy(elems, lst.elements)

	return &VectorExpr{elements: elems}, nil
}

func builtinVectorFill(args []Expr) (Expr, error) {
	if err := checkArity("vector-fill!", args, 2); err != nil {
		return nil, err
	}

	vec, err := toVec("vector-fill!", args[0])
	if err != nil {
		return nil, err
	}

	for i := range vec.elements {
		vec.elements[i] = args[1]
	}

	return Void(), nil
}

func builtinSymbolToString(args []Expr) (Expr, error) {
	if err := checkArity("symbol->string", args, 1); err != nil {
		return nil, err
	}

	val, err := toSym("symbol->string", args[0])
	if err != nil {
		return nil, err
	}

	return &StringExpr{val: val}, nil
}

func builtinStringToSymbol(args []Expr) (Expr, error) {
	if err := checkArity("string->symbol", args, 1); err != nil {
		return nil, err
	}

	val, err := toStr("string->symbol", args[0])
	if err != nil {
		return nil, err
	}

	return &SymbolExpr{val: val}, nil
}

func builtinGensym(args []Expr) (Expr, error) {
	if err := checkArity("gensym", args, 1); err != nil {
		return nil, err
	}

	val, err := toStr("gensym", args[0])
	if err != nil {
		return nil, err
	}

	return &SymbolExpr{val: gensym(val)}, nil
}

func builtinDatumToSyntax(args []Expr) (Expr, error) {
	if err := checkArity("datum->syntax", args, 2); err != nil {
		return nil, err
	}

	return args[1], nil
}

func builtinSyntaxToDatum(args []Expr) (Expr, error) {
	if err := checkArity("syntax->datum", args, 1); err != nil {
		return nil, err
	}

	return args[0], nil
}
