package glerp

import "fmt"

func hashTableBuiltins() map[string]BuiltinFn {
	return map[string]BuiltinFn{
		"make-hash-table": builtinMakeHashTable,
		"hash-table?": typePred(
			"hash-table?",
			func(e Expr) bool { h, ok := e.(*HashTableExpr); return ok && h.data != nil },
		),
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

var (
	builtinHashTableSize = b1hash("hash-table-size", func(ht *HashTableExpr) Expr {
		return num(float64(ht.Size()))
	})
	builtinHashTableKeys = b1hash("hash-table-keys", func(ht *HashTableExpr) Expr {
		elems := make([]Expr, len(ht.order))
		for i, sk := range ht.order {
			elems[i] = ht.keys[sk]
		}
		l, _ := builtinList(elems)
		return l
	})
	builtinHashTableValues = b1hash("hash-table-values", func(ht *HashTableExpr) Expr {
		elems := make([]Expr, len(ht.order))
		for i, sk := range ht.order {
			elems[i] = ht.data[sk]
		}
		l, _ := builtinList(elems)
		return l
	})
	builtinHashTableToAlist = b1hash("hash-table->alist", func(ht *HashTableExpr) Expr {
		elems := make([]Expr, len(ht.order))
		for i, sk := range ht.order {
			pair, _ := builtinList([]Expr{ht.keys[sk], ht.data[sk]})
			elems[i] = pair
		}
		l, _ := builtinList(elems)
		return l
	})
	builtinHashTableCopy = b1hash("hash-table-copy", func(ht *HashTableExpr) Expr {
		cp := newHashTable(ht.tok)
		for _, sk := range ht.order {
			cp.Set(ht.keys[sk], ht.data[sk])
		}
		return cp
	})
)

func builtinAlistToHashTable(args []Expr) (Expr, error) {
	if err := checkArity("alist->hash-table", args, 1); err != nil {
		return nil, err
	}

	slice, err := toList("alist->hash-table", args[0])
	if err != nil {
		return nil, err
	}

	ht := newHashTable(Token{})

	for _, el := range slice {
		pairSlice, err := toSlice("alist->hash-table", el)
		if err != nil || len(pairSlice) != 2 {
			return nil, fmt.Errorf(
				"alist->hash-table: each element must be a (key value) list, got %s",
				el.String(),
			)
		}

		ht.Set(pairSlice[0], pairSlice[1])
	}

	return ht, nil
}
