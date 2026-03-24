package glerp

import "fmt"

func hashTableBuiltins() map[string]BuiltinFn {
	return map[string]BuiltinFn{
		"make-hash-table":      builtinMakeHashTable,
		"hash-table?":          typePred("hash-table?", func(e Expr) bool { h, ok := e.(*HashTableExpr); return ok && h.data != nil }),
		"hash-table-ref":       builtinHashTableRef,
		"hash-table-set!":      builtinHashTableSet,
		"hash-table-delete!":   builtinHashTableDelete,
		"hash-table-contains?": builtinHashTableContains,
		"hash-table-size":      builtinHashTableSize,
		"hash-table-keys":      builtinHashTableKeys,
		"hash-table-values":    builtinHashTableValues,
		"hash-table->alist":    builtinHashTableToAlist,
		"alist->hash-table":    builtinAlistToHashTable,
		"hash-table-copy":      builtinHashTableCopy,
	}
}

func toHash(name string, e Expr) (*HashTableExpr, error) {
	h, ok := e.(*HashTableExpr)
	if !ok || h.data == nil {
		return nil, fmt.Errorf("%s: expected hash-table, got %s", name, e.String())
	}

	return h, nil
}

func builtinMakeHashTable(args []Expr) (Expr, error) {
	if err := checkArity("make-hash-table", args, 0); err != nil {
		return nil, err
	}

	return newHashTable(Token{}), nil
}

func builtinHashTableRef(args []Expr) (Expr, error) {
	if len(args) < 2 || len(args) > 3 {
		return nil, fmt.Errorf("hash-table-ref: expected 2 or 3 arguments, got %d", len(args))
	}

	ht, err := toHash("hash-table-ref", args[0])
	if err != nil {
		return nil, err
	}

	v, ok := ht.Get(args[1])
	if !ok {
		if len(args) == 3 {
			return args[2], nil
		}
		return nil, fmt.Errorf("hash-table-ref: key %s not found", args[1].String())
	}

	return v, nil
}

func builtinHashTableSet(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-set!", args, 3); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-set!", args[0])
	if err != nil {
		return nil, err
	}

	ht.Set(args[1], args[2])

	return Void(), nil
}

func builtinHashTableDelete(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-delete!", args, 2); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-delete!", args[0])
	if err != nil {
		return nil, err
	}

	ht.Delete(args[1])

	return Void(), nil
}

func builtinHashTableContains(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-contains?", args, 2); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-contains?", args[0])
	if err != nil {
		return nil, err
	}

	_, ok := ht.Get(args[1])

	return boolean(ok), nil
}

func builtinHashTableSize(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-size", args, 1); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-size", args[0])
	if err != nil {
		return nil, err
	}

	return num(float64(ht.Size())), nil
}

func builtinHashTableKeys(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-keys", args, 1); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-keys", args[0])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, len(ht.order))

	for i, sk := range ht.order {
		elems[i] = ht.keys[sk]
	}

	return &ListExpr{elements: elems}, nil
}

func builtinHashTableValues(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-values", args, 1); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-values", args[0])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, len(ht.order))

	for i, sk := range ht.order {
		elems[i] = ht.data[sk]
	}

	return &ListExpr{elements: elems}, nil
}

func builtinHashTableToAlist(args []Expr) (Expr, error) {
	if err := checkArity("hash-table->alist", args, 1); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table->alist", args[0])
	if err != nil {
		return nil, err
	}

	elems := make([]Expr, len(ht.order))

	for i, sk := range ht.order {
		elems[i] = &ListExpr{elements: []Expr{ht.keys[sk], ht.data[sk]}}
	}

	return &ListExpr{elements: elems}, nil
}

func builtinAlistToHashTable(args []Expr) (Expr, error) {
	if err := checkArity("alist->hash-table", args, 1); err != nil {
		return nil, err
	}

	lst, err := toList("alist->hash-table", args[0])
	if err != nil {
		return nil, err
	}

	ht := newHashTable(Token{})

	for _, el := range lst.elements {
		pair, ok := el.(*ListExpr)
		if !ok || len(pair.elements) != 2 {
			return nil, fmt.Errorf("alist->hash-table: each element must be a (key value) list, got %s", el.String())
		}

		ht.Set(pair.elements[0], pair.elements[1])
	}

	return ht, nil
}

func builtinHashTableCopy(args []Expr) (Expr, error) {
	if err := checkArity("hash-table-copy", args, 1); err != nil {
		return nil, err
	}

	ht, err := toHash("hash-table-copy", args[0])
	if err != nil {
		return nil, err
	}

	cp := newHashTable(ht.tok)

	for _, sk := range ht.order {
		cp.Set(ht.keys[sk], ht.data[sk])
	}

	return cp, nil
}
