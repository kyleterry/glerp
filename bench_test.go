package glerp

import (
	"strings"
	"testing"
)

func BenchmarkTokenize(b *testing.B) {
	src := `
		(define (factorial n)
		  (if (= n 0) 1
		      (* n (factorial (- n 1)))))
		(define (fibonacci n)
		  (if (< n 2) n
		      (+ (fibonacci (- n 1)) (fibonacci (- n 2)))))
		(define (map f lst)
		  (if (empty? lst) '()
		      (cons (f (car lst)) (map f (cdr lst)))))
	`

	b.ResetTimer()

	for b.Loop() {
		tzr := NewTokenizer()
		_, err := tzr.Run(strings.NewReader(src))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	src := `
		(define (factorial n)
		  (if (= n 0) 1
		      (* n (factorial (- n 1)))))
		(define (fibonacci n)
		  (if (< n 2) n
		      (+ (fibonacci (- n 1)) (fibonacci (- n 2)))))
		(define (map f lst)
		  (if (empty? lst) '()
		      (cons (f (car lst)) (map f (cdr lst)))))
	`

	b.ResetTimer()

	for b.Loop() {
		lexer, err := NewLexer(strings.NewReader(src))
		if err != nil {
			b.Fatal(err)
		}
		p := NewParser(lexer)
		_, err = p.Run()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchSetup(b *testing.B, env *Environment, src string) {
	b.Helper()

	if _, err := Eval(src, env); err != nil {
		b.Fatal(err)
	}
}

func BenchmarkEval(b *testing.B) {
	b.Run("arithmetic", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		for b.Loop() {
			_, err := Eval("(+ (* 2 3) (- 10 (/ 20 4)))", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("factorial_10", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		benchSetup(b, env, "(define (fact n) (if (= n 0) 1 (* n (fact (- n 1)))))")

		b.ResetTimer()

		for b.Loop() {
			_, err := Eval("(fact 10)", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("fibonacci_15", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		benchSetup(b, env, "(define (fib n) (if (< n 2) n (+ (fib (- n 1)) (fib (- n 2)))))")

		b.ResetTimer()

		for b.Loop() {
			_, err := Eval("(fib 15)", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("lambda_closure", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		benchSetup(b, env, "(define (make-adder n) (lambda (x) (+ n x)))")
		benchSetup(b, env, "(define add5 (make-adder 5))")

		b.ResetTimer()

		for b.Loop() {
			_, err := Eval("(add5 10)", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("list_operations", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		benchSetup(b, env, `(define (my-length lst)
			(if (empty? lst) 0
			    (+ 1 (my-length (cdr lst)))))`)
		benchSetup(b, env, "(define xs '(1 2 3 4 5 6 7 8 9 10))")

		b.ResetTimer()

		for b.Loop() {
			_, err := Eval("(my-length xs)", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("tail_call_1000", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		benchSetup(b, env, `(define (loop n)
			(if (= n 0) "done" (loop (- n 1))))`)

		b.ResetTimer()

		for b.Loop() {
			_, err := Eval("(loop 1000)", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("let_binding", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		for b.Loop() {
			_, err := Eval("(let ((x 3) (y 4) (z 5)) (+ x (* y z)))", env)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkEnvironmentLookup(b *testing.B) {
	b.Run("shallow", func(b *testing.B) {
		env := NewEnvironment(DefaultConfig())
		env.Bind("target", &NumberExpr{val: 42})

		b.ResetTimer()

		for b.Loop() {
			_, err := env.Find("target")
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("deep_scope_chain", func(b *testing.B) {
		root := NewEnvironment(DefaultConfig())
		root.Bind("target", &NumberExpr{val: 42})

		env := root
		for i := range 10 {
			env = env.Extend()
			env.Bind("dummy", &NumberExpr{val: float64(i)})
		}

		b.ResetTimer()

		for b.Loop() {
			_, err := env.Find("target")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkNewEnvironment(b *testing.B) {
	cfg := DefaultConfig()

	b.ResetTimer()

	for b.Loop() {
		NewEnvironment(cfg)
	}
}
