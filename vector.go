package glerp

import "fmt"

func vectorBuiltins() map[string]BuiltinFn {
	return map[string]BuiltinFn{
		"vector":        builtinVector,
		"make-vector":   builtinMakeVector,
		"vector-ref":    builtinVectorRef,
		"vector-set!":   builtinVectorSet,
		"vector-length": builtinVectorLength,
		"vector?":       typePred("vector?", func(e Expr) bool { _, ok := e.(*VectorExpr); return ok }),
		"vector->list":  builtinVectorToList,
		"list->vector":  builtinListToVector,
		"vector-fill!":  builtinVectorFill,
	}
}

// vectorIndex extracts a VectorExpr from args[0] and a valid index from args[1].
func vectorIndex(name string, args []Expr) (*VectorExpr, int, error) {
	vec, err := toVector(name, args[0])
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

	vec, idx, err := vectorIndex("vector-ref", args)
	if err != nil {
		return nil, err
	}

	return vec.elements[idx], nil
}

func builtinVectorSet(args []Expr) (Expr, error) {
	if err := checkArity("vector-set!", args, 3); err != nil {
		return nil, err
	}

	vec, idx, err := vectorIndex("vector-set!", args)
	if err != nil {
		return nil, err
	}

	vec.elements[idx] = args[2]

	return Void(), nil
}

var (
	builtinVectorLength = b1vec("vector-length", func(v *VectorExpr) Expr { return num(float64(len(v.elements))) })
	builtinVectorToList = b1vec("vector->list", func(v *VectorExpr) Expr {
		elems := make([]Expr, len(v.elements))
		copy(elems, v.elements)
		l, _ := builtinList(elems)
		return l
	})
	builtinListToVector = b1lst("list->vector", func(slice []Expr) Expr {
		elems := make([]Expr, len(slice))
		copy(elems, slice)
		return &VectorExpr{elements: elems}
	})
)

func builtinVectorFill(args []Expr) (Expr, error) {
	if err := checkArity("vector-fill!", args, 2); err != nil {
		return nil, err
	}

	vec, err := toVector("vector-fill!", args[0])
	if err != nil {
		return nil, err
	}

	for i := range vec.elements {
		vec.elements[i] = args[1]
	}

	return Void(), nil
}
