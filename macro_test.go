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
