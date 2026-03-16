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
		{"lambda zero args", "((lambda () 42))", "42"},
		{"lambda zero args multi-body", "((lambda () (define x 7) (* x x)))", "49"},
		{"lambda zero args closure", "(define n 10) ((lambda () n))", "10"},
		{"lambda zero args returns lambda", `
			(define (make-counter)
			  (define n 0)
			  (lambda ()
			    (set! n (+ n 1))
			    n))
			(define inc (make-counter))
			(inc) (inc) (inc)
		`, "3"},
		{"lambda closure", "(define make-adder (lambda (n) (lambda (x) (+ n x)))) ((make-adder 3) 7)", "10"},

		// define function shorthand
		{"define fn", "(define (square x) (* x x)) (square 7)", "49"},
		{"define fn multi-body", "(define (abs x) (if (< x 0) (- x) x)) (abs -5)", "5"},

		// if
		{"if true branch", "(if #t 1 2)", "1"},
		{"if false branch", "(if #f 1 2)", "2"},
		{"if truthy", "(if 42 \"yes\" \"no\")", `"yes"`},
		{"if no else true", "(if #t 99)", "99"},
		{"if no else false", "(if #f 99)", ""},

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

		// string interpolation
		{"interp plain", `$"hello"`, `"hello"`},
		{"interp var", `(define name "world") $"Hello {name}!"`, `"Hello world!"`},
		{"interp expr", `$"1+2={( + 1 2)}"`, `"1+2=3"`},
		{"interp number", `(define n 7) $"n={n}"`, `"n=7"`},
		{"interp multi", `(define a "x") (define b "y") $"{a}+{b}"`, `"x+y"`},

		// quasiquote / unquote / unquote-splicing
		{"quasiquote plain", "`(1 2 3)", "(1 2 3)"},
		{"quasiquote atom", "`hello", "hello"},
		{"unquote", "(define x 42) `(a ,x c)", "(a 42 c)"},
		{"unquote expr", "`(a ,(+ 1 2) c)", "(a 3 c)"},
		{"unquote-splicing", "(define xs '(2 3)) `(1 ,@xs 4)", "(1 2 3 4)"},
		{"unquote-splicing empty", "`(1 ,@'() 2)", "(1 2)"},
		{"quasiquote nested lists", "`((a ,(+ 1 1)) (b ,(+ 2 2)))", "((a 2) (b 4))"},

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

		// do
		{"do basic counter", `
			(do [(i 0 (+ i 1))]
			    [(= i 5) i])
		`, "5"},
		{"do sum", `
			(do [(i 1 (+ i 1))
			     (sum 0 (+ sum i))]
			    [(> i 10) sum])
		`, "55"},
		{"do no result expr returns void", `
			(do [(i 0 (+ i 1))]
			    [(= i 3)])
		`, ""},
		{"do no step keeps value", `
			(do [(x 42)
			     (i 0 (+ i 1))]
			    [(= i 3) x])
		`, "42"},
		{"do parallel step update", `
			(do [(a 1 b)
			     (b 2 a)]
			    [(= a 2) (list a b)])
		`, "(2 1)"},
		{"do no vars", `
			(define x 0)
			(do []
			    [(= x 3) x]
			  (set! x (+ x 1)))
		`, "3"},
		{"do body side effects", `
			(import :scheme/list)
			(define result '())
			(do [(i 0 (+ i 1))]
			    [(= i 3) (reverse result)]
			  (set! result (cons i result)))
		`, "(0 1 2)"},

		// values / define-values
		{"values single is identity", "(values 42)", "42"},
		{"values multiple", "(values 1 2 3)", "(values 1 2 3)"},
		{"define-values basic", "(define-values (a b c) (values 1 2 3)) (+ a b c)", "6"},
		{"define-values single", "(define-values (x) (values 99)) x", "99"},
		{"define-values single non-values", "(define-values (x) 42) x", "42"},
		{"define-values from lambda", `
			(define (minmax a b)
			  (if (< a b) (values a b) (values b a)))
			(define-values (lo hi) (minmax 7 3))
			(list lo hi)
		`, "(3 7)"},

		// case
		{"case match first", `(case 1 ((1) "one") ((2) "two"))`, `"one"`},
		{"case match second", `(case 2 ((1) "one") ((2) "two"))`, `"two"`},
		{"case multi-datum", `(case 3 ((1 2) "low") ((3 4) "high"))`, `"high"`},
		{"case else", `(case 99 ((1) "one") (else "other"))`, `"other"`},
		{"case no match", `(case 5 ((1) "one") ((2) "two"))`, ``},
		{"case key is expression", `(case (+ 1 1) ((1) "one") ((2) "two"))`, `"two"`},
		{"case symbol datum", `(case 'b ((a) 1) ((b) 2) ((c) 3))`, `2`},
		{"case bool datum", `(case #t ((#f) "no") ((#t) "yes"))`, `"yes"`},
		{"case multi-expr body", `(define x 0) (case 1 ((1) (set! x 10) x))`, `10`},

		// square bracket list syntax
		{"bracket list literal", "'[1 2 3]", "(1 2 3)"},
		{"bracket let bindings", "(let [(x 3) (y 4)] (+ x y))", "7"},
		{"bracket let* bindings", "(let* [(x 3) (y (* x 2))] y)", "6"},
		{"bracket nested in parens", "(+ 1 [+ 2 3])", "6"},
		{"parens nested in brackets", "[+ 1 (+ 2 3)]", "6"},
		{"bracket let quoted value", "(let [(foo 'foo-value) (bar \"bar value\")] foo)", "foo-value"},

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

		// begin
		{"begin returns last", "(begin 1 2 3)", "3"},
		{"begin side effects", "(define x 0) (begin (set! x 1) (set! x 2)) x", "2"},
		{"begin empty", "(begin)", ""},

		// cond
		{"cond first match", "(cond ((= 1 1) \"one\") ((= 1 2) \"two\"))", `"one"`},
		{"cond second match", "(cond ((= 1 2) \"one\") ((= 1 1) \"two\"))", `"two"`},
		{"cond else", "(cond ((= 1 2) \"no\") (else \"yes\"))", `"yes"`},
		{"cond no match", "(cond (#f 1))", ""},

		// and / or
		{"and all true", "(and 1 2 3)", "3"},
		{"and short circuit", "(and 1 #f 3)", "#f"},
		{"and empty", "(and)", "#t"},
		{"or first true", "(or #f 2 3)", "2"},
		{"or all false", "(or #f #f)", "#f"},
		{"or empty", "(or)", "#f"},

		// stdlib: :scheme/list
		{"import list length", "(import :scheme/list) (length '(a b c d))", "4"},
		{"import list append", "(import :scheme/list) (append '(1 2) '(3 4))", "(1 2 3 4)"},
		{"import list reverse", "(import :scheme/list) (reverse '(1 2 3))", "(3 2 1)"},
		{"import list map", "(import :scheme/list) (map (lambda (x) (* x 2)) '(1 2 3))", "(2 4 6)"},
		{"import list filter", "(import :scheme/list) (filter (lambda (x) (> x 2)) '(1 2 3 4))", "(3 4)"},
		{"import list fold", "(import :scheme/list) (fold + 0 '(1 2 3 4 5))", "15"},
		{"import list list-ref", "(import :scheme/list) (list-ref '(a b c) 1)", "b"},
		{"import list list-tail", "(import :scheme/list) (list-tail '(a b c d) 2)", "(c d)"},

		// stdlib: :scheme/math
		{"import math abs pos", "(import :scheme/math) (abs 5)", "5"},
		{"import math abs neg", "(import :scheme/math) (abs -7)", "7"},
		{"import math max", "(import :scheme/math) (max 3 7)", "7"},
		{"import math min", "(import :scheme/math) (min 3 7)", "3"},
		{"import math square", "(import :scheme/math) (square 4)", "16"},
		{"import math cube", "(import :scheme/math) (cube 3)", "27"},
		{"import math average", "(import :scheme/math) (average 4 6)", "5"},
		{"import math clamp lo", "(import :scheme/math) (clamp -5 0 10)", "0"},
		{"import math clamp hi", "(import :scheme/math) (clamp 15 0 10)", "10"},
		{"import math clamp in", "(import :scheme/math) (clamp 5 0 10)", "5"},

		// multiple specs in one import
		{"import multi", "(import :scheme/list :scheme/math) (square (length '(a b c)))", "9"},

		// (only ...) selective import
		{"import only", "(import (only :scheme/list map filter)) (map (lambda (x) (* x x)) '(1 2 3))", "(1 4 9)"},
		{"import only excludes others", `
			(import (only :scheme/list map))
			(define length "not imported")
			length
		`, `"not imported"`},

		// stdlib: :scheme/time
		{"import time current-second positive", "(import :scheme/time) (> (current-second) 0)", "#t"},
		{"import time jiffies-per-second", "(import :scheme/time) (jiffies-per-second)", "1000000000"},
		{"import time current-jiffy positive", "(import :scheme/time) (> (current-jiffy) 0)", "#t"},
		{"import time elapsed", "(import :scheme/time) (let ((a (current-jiffy)) (b (current-jiffy))) (>= b a))", "#t"},
		{"import time make-time year", "(import :scheme/list :scheme/time) (time-year (make-time 2024 3 15 12 0 0))", "2024"},
		{"import time make-time month", "(import :scheme/list :scheme/time) (time-month (make-time 2024 3 15 12 0 0))", "3"},
		{"import time make-time day", "(import :scheme/list :scheme/time) (time-day (make-time 2024 3 15 12 0 0))", "15"},
		{"import time make-time hour", "(import :scheme/list :scheme/time) (time-hour (make-time 2024 3 15 12 30 45))", "12"},
		{"import time make-time minute", "(import :scheme/list :scheme/time) (time-minute (make-time 2024 3 15 12 30 45))", "30"},
		{"import time make-time second", "(import :scheme/list :scheme/time) (time-second (make-time 2024 3 15 12 30 45))", "45"},
		{"import time weekday", "(import :scheme/list :scheme/time) (time-weekday (make-time 2024 3 15 0 0 0))", "5"},
		{"import time weekday-name", `(import :scheme/list :scheme/time) (time-weekday-name (make-time 2024 3 15 0 0 0))`, `"Friday"`},
		{"import time month-name", `(import :scheme/list :scheme/time) (time-month-name (make-time 2024 3 15 0 0 0))`, `"March"`},
		{"import time duration seconds", "(import :scheme/time) (seconds 5)", "5"},
		{"import time duration minutes", "(import :scheme/time) (minutes 2)", "120"},
		{"import time duration hours", "(import :scheme/time) (hours 1)", "3600"},
		{"import time duration days", "(import :scheme/time) (days 1)", "86400"},
		{"import time duration weeks", "(import :scheme/time) (weeks 1)", "604800"},
		{"import time time-add", "(import :scheme/list :scheme/time) (let ((t (make-time 2024 1 1 0 0 0))) (time-year (time-add t (days 366))))", "2025"},
		{"import time time-difference", "(import :scheme/time) (time-difference (seconds 100) (seconds 30))", "70"},
		{"import time time<?", "(import :scheme/time) (time<? (seconds 1) (seconds 2))", "#t"},
		{"import time time>?", "(import :scheme/time) (time>? (seconds 2) (seconds 1))", "#t"},
		{"import time time=?", "(import :scheme/time) (time=? (seconds 5) (seconds 5))", "#t"},
		{"import time time<=?", "(import :scheme/time) (time<=? (seconds 3) (seconds 3))", "#t"},
		{"import time time>=?", "(import :scheme/time) (time>=? (seconds 4) (seconds 3))", "#t"},
		{"import time time->string", `(import :scheme/list :scheme/time) (time->string (make-time 2024 3 15 12 0 0))`, `"2024-03-15T12:00:00Z"`},
		{"import time string->time round-trip", `(import :scheme/list :scheme/time) (time-day (string->time "2024-03-15T12:00:00Z"))`, "15"},
		{"import time time->string/fmt date", `(import :scheme/list :scheme/time) (time->string/fmt (make-time 2024 3 15 0 0 0) time-format/date)`, `"2024-03-15"`},
		{"import time time-components length", `(import :scheme/list :scheme/time) (length (time-components (make-time 2024 1 1 0 0 0)))`, "7"},

		// syntax macros: define-syntax / syntax-rules
		{"macro simple substitution", `
			(define-syntax my-inc
			  (syntax-rules ()
			    [(_ x) (+ x 1)]))
			(my-inc 5)
		`, "6"},
		{"macro multiple rules", `
			(define-syntax my-and2
			  (syntax-rules ()
			    [(_ a b) (if a b #f)]))
			(my-and2 #t 99)
		`, "99"},
		{"macro literal keyword", `
			(define-syntax my-case
			  (syntax-rules (=>)
			    [(_ val (test => result)) (if (= val test) result #f)]))
			(my-case 3 (3 => 42))
		`, "42"},
		{"macro ellipsis body", `
			(define-syntax my-begin
			  (syntax-rules ()
			    [(_ e) e]
			    [(_ e1 e2 ...) (let ([_ e1]) (my-begin e2 ...))]))
			(my-begin 1 2 3)
		`, "3"},
		{"macro ellipsis args", `
			(define-syntax my-list
			  (syntax-rules ()
			    [(_ x ...) (list x ...)]))
			(my-list 10 20 30)
		`, "(10 20 30)"},
		{"macro ellipsis zero args", `
			(define-syntax my-list
			  (syntax-rules ()
			    [(_ x ...) (list x ...)]))
			(my-list)
		`, "()"},
		{"macro nested ellipsis (my-let)", `
			(define-syntax my-let
			  (syntax-rules ()
			    [(_ ((var val) ...) body ...)
			     ((lambda (var ...) body ...) val ...)]))
			(my-let ((x 3) (y 4)) (+ x y))
		`, "7"},
		{"macro recursive (my-and)", `
			(define-syntax my-and
			  (syntax-rules ()
			    [(_) #t]
			    [(_ e) e]
			    [(_ e1 e2 ...) (if e1 (my-and e2 ...) #f)]))
			(my-and 1 2 3)
		`, "3"},
		{"macro recursive short-circuit", `
			(define-syntax my-and
			  (syntax-rules ()
			    [(_) #t]
			    [(_ e) e]
			    [(_ e1 e2 ...) (if e1 (my-and e2 ...) #f)]))
			(my-and 1 #f 3)
		`, "#f"},
		{"macro hygiene (swap!)", `
			(define-syntax swap!
			  (syntax-rules ()
			    [(_ a b)
			     (let ([tmp a])
			       (set! a b)
			       (set! b tmp))]))
			(define tmp 42)
			(define x 1)
			(define y 2)
			(swap! x y)
			(list tmp x y)
		`, "(42 2 1)"},

		// export form in user code
		{"export declares exports", `
			(define (double x) (* x 2))
			(export double)
			(double 5)
		`, "10"},
		{"export #t exports all", `
			(define (double x) (* x 2))
			(export #t)
			(double 5)
		`, "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(glerp.StandardBuiltins(), glerp.StandardForms())
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
		{"import unknown library", "(import :scheme/nonexistent)"},
		{"import unrecognized path", "(import foo/bar)"},
		{"import only nonexported", "(import (only :scheme/list nonexistent-fn))"},
		{"import only unknown modifier", "(import (xyzzy :scheme/list map))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(glerp.StandardBuiltins(), glerp.StandardForms())
			_, err := glerp.Eval(tt.src, env)
			is.True(err != nil)
		})
	}
}
