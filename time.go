package glerp

import (
	"fmt"
	"time"
)

// timeBuiltins returns Go-backed procedures for time operations. These are
// included in StandardBuiltins and expose the go-extern: names used by
// stdlib/scheme/time.scm.
func timeBuiltins() map[string]BuiltinFn {
	return map[string]BuiltinFn{
		"go-extern:current-second":  builtinCurrentSecond,
		"go-extern:current-jiffy":   builtinCurrentJiffy,
		"go-extern:time-make":       builtinTimeMake,
		"go-extern:time-components": builtinTimeComponents,
		"go-extern:time-format":     builtinTimeFormat,
		"go-extern:time-parse":      builtinTimeParse,
	}
}

func builtinCurrentSecond(args []Expr) (Expr, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("go-extern:current-second: expected 0 arguments, got %d", len(args))
	}

	return num(float64(time.Now().UnixNano()) / 1e9), nil
}

func builtinCurrentJiffy(args []Expr) (Expr, error) {
	if len(args) != 0 {
		return nil, fmt.Errorf("go-extern:current-jiffy: expected 0 arguments, got %d", len(args))
	}

	return num(float64(time.Now().UnixNano())), nil
}

// builtinTimeMake constructs a UTC time from (year month day hour minute second)
// and returns it as a unix timestamp (float64 seconds).
func builtinTimeMake(args []Expr) (Expr, error) {
	if len(args) != 6 {
		return nil, fmt.Errorf("go-extern:time-make: expected 6 arguments, got %d", len(args))
	}

	vals := make([]int, 6)

	for i, a := range args {
		n, err := toNum("go-extern:time-make", a)
		if err != nil {
			return nil, err
		}
		vals[i] = int(n)
	}

	t := time.Date(vals[0], time.Month(vals[1]), vals[2], vals[3], vals[4], vals[5], 0, time.UTC)

	return num(float64(t.Unix())), nil
}

// builtinTimeComponents returns (year month day hour minute second weekday) for
// a unix timestamp. weekday is 0 (Sunday) through 6 (Saturday).
func builtinTimeComponents(args []Expr) (Expr, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("go-extern:time-components: expected 1 argument, got %d", len(args))
	}

	ts, err := toNum("go-extern:time-components", args[0])
	if err != nil {
		return nil, err
	}

	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()

	elems := []Expr{
		num(float64(t.Year())),
		num(float64(t.Month())),
		num(float64(t.Day())),
		num(float64(t.Hour())),
		num(float64(t.Minute())),
		num(float64(t.Second())),
		num(float64(t.Weekday())),
	}

	return &ListExpr{elements: elems}, nil
}

// builtinTimeFormat formats a unix timestamp using a Go reference-time layout string.
func builtinTimeFormat(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("go-extern:time-format: expected 2 arguments, got %d", len(args))
	}

	ts, err := toNum("go-extern:time-format", args[0])
	if err != nil {
		return nil, err
	}
	layout, ok := args[1].(*StringExpr)
	if !ok {
		return nil, fmt.Errorf("go-extern:time-format: layout must be a string, got %s", args[1].String())
	}

	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	t := time.Unix(sec, nsec).UTC()

	return &StringExpr{val: t.Format(layout.val)}, nil
}

// builtinTimeParse parses a time string using a Go reference-time layout and
// returns a unix timestamp (float64 seconds).
func builtinTimeParse(args []Expr) (Expr, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("go-extern:time-parse: expected 2 arguments, got %d", len(args))
	}

	layout, ok1 := args[0].(*StringExpr)
	value, ok2 := args[1].(*StringExpr)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("go-extern:time-parse: both arguments must be strings")
	}

	t, err := time.Parse(layout.val, value.val)
	if err != nil {
		return nil, fmt.Errorf("go-extern:time-parse: %w", err)
	}

	return num(float64(t.Unix())), nil
}
