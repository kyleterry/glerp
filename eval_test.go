package glerp_test

import (
	"testing"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestEval(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		// literals
		{"number int", "42", "42"},
		{"number float", "3.14", "3.14"},
		{"string", `"hello"`, `"hello"`},
		{"bool true", "#t", "#t"},
		{"bool false", "#f", "#f"},
		{"empty list", "'()", "()"},

		// arithmetic
		{"add", "(+ 1 2)", "3"},
		{"add variadic", "(+ 1 2 3 4)", "10"},
		{"add zero args", "(+)", "0"},
		{"sub", "(- 10 3)", "7"},
		{"sub unary", "(- 5)", "-5"},
		{"mul", "(* 4 5)", "20"},
		{"mul variadic", "(* 2 3 4)", "24"},
		{"div", "(/ 10 2)", "5"},
		{"nested arith", "(+ (* 2 3) (- 10 4))", "12"},

		// comparisons
		{"less true", "(< 1 2)", "#t"},
		{"less false", "(< 2 1)", "#f"},
		{"greater true", "(> 2 1)", "#t"},
		{"greater false", "(> 1 2)", "#f"},
		{"num eq true", "(= 3 3)", "#t"},
		{"num eq false", "(= 3 4)", "#f"},
		{"less eq", "(<= 3 3)", "#t"},
		{"greater eq", "(>= 4 3)", "#t"},

		// logic
		{"not false", "(not #f)", "#t"},
		{"not true", "(not #t)", "#f"},
		{"not truthy", "(not 42)", "#f"},

		// define & lookup
		{"define var", "(define x 10) x", "10"},
		{"define overwrites", "(define x 1) (define x 2) x", "2"},

		// lambda
		{"lambda call", "((lambda (x) (* x x)) 5)", "25"},
		{"lambda closure", "(define make-adder (lambda (n) (lambda (x) (+ n x)))) ((make-adder 3) 7)", "10"},

		// define function shorthand
		{"define fn", "(define (square x) (* x x)) (square 7)", "49"},
		{"define fn multi-body", "(define (abs x) (if (< x 0) (- x) x)) (abs -5)", "5"},

		// if
		{"if true branch", "(if #t 1 2)", "1"},
		{"if false branch", "(if #f 1 2)", "2"},
		{"if truthy", "(if 42 \"yes\" \"no\")", `"yes"`},
		{"if no else true", "(if #t 99)", "99"},
		{"if no else false", "(if #f 99)", "#f"},

		// let
		{"let basic", "(let ((x 3) (y 4)) (+ x y))", "7"},
		{"let shadow", "(define x 10) (let ((x 99)) x)", "99"},
		{"let parallel", "(define x 1) (let ((x 2) (y x)) y)", "1"},

		// let*
		{"let* sequential", "(let* ((x 3) (y (* x 2))) y)", "6"},

		// set!
		{"set!", "(define x 1) (set! x 99) x", "99"},

		// quote
		{"quote shorthand", "'(1 2 3)", "(1 2 3)"},
		{"quote long form", "(quote (a b c))", "(a b c)"},
		{"quote atom", "'hello", "hello"},

		// list operations
		{"car", "(car '(1 2 3))", "1"},
		{"cdr", "(cdr '(1 2 3))", "(2 3)"},
		{"cdr single", "(cdr '(1))", "()"},
		{"cons", "(cons 1 '(2 3))", "(1 2 3)"},
		{"cons to empty", "(cons 42 '())", "(42)"},
		{"empty? true", "(empty? '())", "#t"},
		{"empty? false", "(empty? '(1))", "#f"},
		{"list", "(list 1 2 3)", "(1 2 3)"},
		{"list empty", "(list)", "()"},

		// recursion
		{"factorial", "(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (fact 5)", "120"},
		{"fibonacci", "(define (fib n) (if (< n 2) n (+ (fib (- n 1)) (fib (- n 2))))) (fib 10)", "55"},

		// higher-order
		{"map-like", `
			(define (my-map f lst)
			  (if (empty? lst)
			      '()
			      (cons (f (car lst)) (my-map f (cdr lst)))))
			(my-map (lambda (x) (* x x)) '(1 2 3 4 5))
		`, "(1 4 9 16 25)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment()
			results, err := glerp.Eval(tt.src, env)
			is.NoErr(err)
			is.True(len(results) > 0)
			is.Equal(results[len(results)-1].String(), tt.want)
		})
	}
}

func TestEvalErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"unbound var", "x"},
		{"wrong arg count", "((lambda (x) x) 1 2)"},
		{"car empty list", "(car '())"},
		{"cdr empty list", "(cdr '())"},
		{"div by zero", "(/ 1 0)"},
		{"not a procedure", "(1 2 3)"},
		{"set! unbound", "(set! undefined-var 42)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment()
			_, err := glerp.Eval(tt.src, env)
			is.True(err != nil)
		})
	}
}
