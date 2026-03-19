package glerp_test

import (
	"testing"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestMacro(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{"simple substitution", `
			(define-syntax my-inc
			  (syntax-rules ()
			    [(_ x) (+ x 1)]))
			(my-inc 5)
		`, "6"},
		{"multiple rules", `
			(define-syntax my-and2
			  (syntax-rules ()
			    [(_ a b) (if a b #f)]))
			(my-and2 #t 99)
		`, "99"},
		{"literal keyword", `
			(define-syntax my-case
			  (syntax-rules (=>)
			    [(_ val (test => result)) (if (= val test) result #f)]))
			(my-case 3 (3 => 42))
		`, "42"},
		{"ellipsis body", `
			(define-syntax my-begin
			  (syntax-rules ()
			    [(_ e) e]
			    [(_ e1 e2 ...) (let ([_ e1]) (my-begin e2 ...))]))
			(my-begin 1 2 3)
		`, "3"},
		{"ellipsis args", `
			(define-syntax my-list
			  (syntax-rules ()
			    [(_ x ...) (list x ...)]))
			(my-list 10 20 30)
		`, "(10 20 30)"},
		{"ellipsis zero args", `
			(define-syntax my-list
			  (syntax-rules ()
			    [(_ x ...) (list x ...)]))
			(my-list)
		`, "()"},
		{"nested ellipsis (my-let)", `
			(define-syntax my-let
			  (syntax-rules ()
			    [(_ ((var val) ...) body ...)
			     ((lambda (var ...) body ...) val ...)]))
			(my-let ((x 3) (y 4)) (+ x y))
		`, "7"},
		{"recursive (my-and)", `
			(define-syntax my-and
			  (syntax-rules ()
			    [(_) #t]
			    [(_ e) e]
			    [(_ e1 e2 ...) (if e1 (my-and e2 ...) #f)]))
			(my-and 1 2 3)
		`, "3"},
		{"recursive short-circuit", `
			(define-syntax my-and
			  (syntax-rules ()
			    [(_) #t]
			    [(_ e) e]
			    [(_ e1 e2 ...) (if e1 (my-and e2 ...) #f)]))
			(my-and 1 #f 3)
		`, "#f"},
		{"hygiene (swap!)", `
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

func TestSyntaxCase(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		// basic syntax-case with lambda transformer
		{"simple transformer", `
			(define-syntax my-inc
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x) (syntax (+ x 1))])))
			(my-inc 5)
		`, "6"},

		// ellipsis in syntax-case
		{"ellipsis", `
			(define-syntax my-list
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x ...) (syntax (list x ...))])))
			(my-list 10 20 30)
		`, "(10 20 30)"},

		// nested pattern
		{"nested pattern", `
			(define-syntax my-let
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ ((var val) ...) body ...)
			       (syntax ((lambda (var ...) body ...) val ...))])))
			(my-let ((x 3) (y 4)) (+ x y))
		`, "7"},

		// fender clause
		{"fender match", `
			(define-syntax classify
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x) (number? x) (syntax "number")]
			      [(_ x) (syntax "other")])))
			(classify 42)
		`, `"number"`},
		{"fender fallthrough", `
			(define-syntax classify
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x) (number? x) (syntax "number")]
			      [(_ x) (syntax "other")])))
			(classify "hello")
		`, `"other"`},

		// body computation with pattern variables
		{"pattern var in body", `
			(define-syntax make-adder
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ name n)
			       (let ([name-str (symbol->string name)])
			         (with-syntax ([full-name (string->symbol (string-append "add-" name-str))])
			           (syntax (define (full-name x) (+ x n)))))])))
			(make-adder three 3)
			(add-three 10)
		`, "13"},

		// with-syntax binding
		{"with-syntax simple", `
			(define-syntax swap-args
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ f a b)
			       (with-syntax ([result (syntax (f b a))])
			         (syntax result))])))
			(swap-args - 1 10)
		`, "9"},

		// with-syntax ellipsis
		{"with-syntax ellipsis", `
			(define-syntax double-list
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x ...)
			       (with-syntax ([(doubled ...) (map (lambda (v) (* v 2)) (syntax (x ...)))])
			         (syntax (list doubled ...)))])))
			(double-list 1 2 3)
		`, "(2 4 6)"},

		// datum->syntax for identifier generation
		{"datum->syntax", `
			(define-syntax def-val
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ name val)
			       (with-syntax ([getter (datum->syntax name
			                       (string->symbol
			                         (string-append "get-" (symbol->string name))))])
			         (syntax (begin
			           (define getter (lambda () val)))))])))
			(def-val answer 42)
			(get-answer)
		`, "42"},

		// symbol->string and string->symbol
		{"symbol->string", `(symbol->string 'hello)`, `"hello"`},
		{"string->symbol", `(symbol->string (string->symbol "world"))`, `"world"`},

		// syntax->datum identity
		{"syntax->datum", `(syntax->datum 42)`, "42"},

		// syntax outside syntax-case acts like quote
		{"syntax as quote", `(syntax (+ 1 2))`, "(+ 1 2)"},

		// #' shorthand for syntax
		{"#' shorthand", `
			(define-syntax my-inc
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x) #'(+ x 1)])))
			(my-inc 5)
		`, "6"},

		// quasisyntax with unsyntax
		{"quasisyntax basic", `
			(define-syntax add-n
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ n body)
			       (let ([doubled (* 2 n)])
			         (quasisyntax (+ body (unsyntax doubled))))])))
			(add-n 3 10)
		`, "16"},

		// quasisyntax with unsyntax-splicing
		{"quasisyntax splicing", `
			(define-syntax wrap-list
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ x ...)
			       (let ([items (syntax (x ...))])
			         (quasisyntax (list (unsyntax-splicing items))))])))
			(wrap-list 1 2 3)
		`, "(1 2 3)"},

		// quasisyntax with pattern vars and computed value
		{"quasisyntax mixed", `
			(define-syntax make-adder2
			  (lambda (stx)
			    (syntax-case stx ()
			      [(_ name n)
			       (let ([fn-name (string->symbol
			               (string-append "add-" (symbol->string name)))])
			         (quasisyntax (define ((unsyntax fn-name) x) (+ x n))))])))
			(make-adder2 five 5)
			(add-five 10)
		`, "15"},

		// struct from prelude
		{"struct basic", `
			(struct point x y)
			(define p (make-point 3 7))
			(list (point? p) (point-x p) (point-y p))
		`, "(#t 3 7)"},
		{"struct setter", `
			(struct point x y)
			(define p (make-point 3 7))
			(set-point-x! p 10)
			(point-x p)
		`, "10"},
		{"struct predicate false", `
			(struct point x y)
			(point? 42)
		`, "#f"},
		{"struct methods", `
			(struct point x y
				(methods
					(add (p other)
						(make-point (+ (point-x p) (point-x other))
									(+ (point-y p) (point-y other))))))
			(define a (make-point 1 2))
			(define b (make-point 3 4))
			(define c (point-add a b))
			(list (point-x c) (point-y c))
		`, "(4 6)"},
		{"struct multiple methods", `
			(struct vec2 x y
				(methods
					(scale (v n)
						(make-vec2 (* (vec2-x v) n) (* (vec2-y v) n)))
					(mag-sq (v)
						(+ (* (vec2-x v) (vec2-x v)) (* (vec2-y v) (vec2-y v))))))
			(define v (make-vec2 3 4))
			(list (vec2-x (vec2-scale v 2)) (vec2-mag-sq v))
		`, "(6 25)"},

		// define-syntax* from prelude
		{"define-syntax* syntax-rules", `
			(define-syntax* my-if ()
				[(_ test then else)
				 (cond [test then] [#t else])])
			(my-if #t 1 2)
		`, "1"},
		{"define-syntax* syntax-case", `
			(define-syntax* (my-swap! stx) ()
				[(_ a b)
				 (syntax
				   (let ([tmp a])
				     (set! a b)
				     (set! b tmp)))])
			(define x 10)
			(define y 20)
			(my-swap! x y)
			(list x y)
		`, "(20 10)"},
		{"define-syntax* with literals", `
			(define-syntax* (my-when stx) ()
				[(_ test body ...)
				 (syntax (if test (begin body ...) (void)))])
			(define x 0)
			(my-when #t (set! x 1) (set! x (+ x 1)))
			x
		`, "2"},
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
