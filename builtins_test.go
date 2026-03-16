package glerp_test

import (
	"testing"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestBuiltins(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
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

		// type predicates
		{"null? empty", "(null? '())", "#t"},
		{"null? non-empty", "(null? '(1))", "#f"},
		{"null? non-list", "(null? 42)", "#f"},
		{"pair? non-empty", "(pair? '(1 2))", "#t"},
		{"pair? empty", "(pair? '())", "#f"},
		{"list? list", "(list? '(1 2))", "#t"},
		{"list? empty", "(list? '())", "#t"},
		{"list? non-list", "(list? 42)", "#f"},
		{"number? num", "(number? 42)", "#t"},
		{"number? str", `(number? "hello")`, "#f"},
		{"string? str", `(string? "hello")`, "#t"},
		{"string? num", "(string? 42)", "#f"},
		{"boolean? bool", "(boolean? #t)", "#t"},
		{"boolean? num", "(boolean? 42)", "#f"},
		{"symbol? sym", "(symbol? 'foo)", "#t"},
		{"symbol? num", "(symbol? 42)", "#f"},
		{"procedure? lambda", "(procedure? (lambda (x) x))", "#t"},
		{"procedure? builtin", "(procedure? +)", "#t"},
		{"procedure? num", "(procedure? 42)", "#f"},

		// eq? / equal?
		{"eq? same num", "(eq? 3 3)", "#t"},
		{"eq? diff num", "(eq? 3 4)", "#f"},
		{"eq? same str", `(eq? "a" "a")`, "#t"},
		{"equal? same list", "(equal? '(1 2 3) '(1 2 3))", "#t"},
		{"equal? diff list", "(equal? '(1 2 3) '(1 2 4))", "#f"},
		{"equal? nested list", "(equal? '(1 (2 3)) '(1 (2 3)))", "#t"},
		{"equal? atom", "(equal? 42 42)", "#t"},

		// modulo / remainder
		{"modulo positive", "(modulo 10 3)", "1"},
		{"modulo negative dividend", "(modulo -10 3)", "2"},
		{"modulo negative divisor", "(modulo 10 -3)", "-2"},
		{"remainder positive", "(remainder 10 3)", "1"},
		{"remainder negative dividend", "(remainder -10 3)", "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(glerp.DefaultConfig())
			results, err := glerp.Eval(tt.src, env)
			is.NoErr(err)
			is.True(len(results) > 0)
			is.Equal(results[len(results)-1].String(), tt.want)
		})
	}
}
