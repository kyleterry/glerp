// Example syntax-macros shows glerp's syntax macro system in action.
//
// report.scm defines four macros using define-syntax / syntax-rules:
//
//	when / unless  control flow (new forms, not built into the interpreter)
//	->>            thread-last pipeline operator
//	check          assertion form that captures its source expression
//
// The Go host registers three formatting procedures and then evaluates the
// file. The macros and the Go-backed procedures work together without either
// side knowing about the other's implementation.
package main

import (
	"fmt"
	"log"
	"strings"

	"go.e64ec.com/glerp"
)

func main() {
	cfg := glerp.DefaultConfig()

	// print-header prints a titled section separator.
	cfg.Builtins["print-header"] = func(args []glerp.Expr) (glerp.Expr, error) {
		if len(args) != 1 {
			return nil, fmt.Errorf("print-header: expected 1 argument, got %d", len(args))
		}
		title := displayString(args[0])
		fmt.Printf("\n%s\n%s\n", title, strings.Repeat("-", len(title)))
		return glerp.Void(), nil
	}

	// print-kv prints a right-aligned label: value pair.
	cfg.Builtins["print-kv"] = func(args []glerp.Expr) (glerp.Expr, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("print-kv: expected 2 arguments, got %d", len(args))
		}
		fmt.Printf("  %-16s %s\n", displayString(args[0]), displayString(args[1]))
		return glerp.Void(), nil
	}

	// print-reading prints one day's temperature with a bar chart and a marker
	// showing whether it is above or below the mean.
	cfg.Builtins["print-reading"] = func(args []glerp.Expr) (glerp.Expr, error) {
		if len(args) != 3 {
			return nil, fmt.Errorf("print-reading: expected 3 arguments, got %d", len(args))
		}
		day := displayString(args[0])
		temp, ok1 := args[1].(*glerp.NumberExpr)
		mean, ok2 := args[2].(*glerp.NumberExpr)
		if !ok1 || !ok2 {
			return nil, fmt.Errorf("print-reading: temp and mean must be numbers")
		}
		t := int(temp.Value())
		bar := strings.Repeat("#", t-10)
		marker := " "
		if temp.Value() > mean.Value() {
			marker = "^"
		}
		fmt.Printf("  %-3s  %2d C  %-24s %s\n", day, t, bar, marker)
		return glerp.Void(), nil
	}

	// report-check is called by the (check expr) macro. It receives the quoted
	// source expression and its evaluated boolean result, then prints pass/fail.
	cfg.Builtins["report-check"] = func(args []glerp.Expr) (glerp.Expr, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("report-check: expected 2 arguments, got %d", len(args))
		}
		src := args[0].String()
		pass := true
		if b, ok := args[1].(*glerp.BoolExpr); ok {
			pass = b.Value()
		}
		status := "PASS"
		if !pass {
			status = "FAIL"
		}
		fmt.Printf("  %-30s %s\n", src, status)
		return glerp.Void(), nil
	}

	env := glerp.NewEnvironment(cfg)
	if err := glerp.EvalFile("report.scm", env); err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println()
}

// displayString returns the natural display form of an expression: strings
// without quotes, everything else via String().
func displayString(e glerp.Expr) string {
	if s, ok := e.(*glerp.StringExpr); ok {
		return s.Value()
	}
	return e.String()
}
