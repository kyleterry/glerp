# glerp

A small Scheme interpreter for embedding in Go programs. It serves as a
scripting and configuration layer for larger applications. You can evaluate
Scheme expressions, extend the language with Go procedures, or build custom
DSLs by registering special forms.

```bash
go get go.e64ec.com/glerp
```

## Language

glerp implements a practical subset of Scheme with extensions for embedding.

### Literals

```scheme
42          ; integer
3.14        ; float
"hello"     ; string
#t #f       ; booleans
'()         ; empty list
'(1 2 3)    ; quoted list
#(1 2 3)    ; vector
{ "k" "v" } ; hash table
```

Square brackets can be used anywhere in place of parentheses. They are common
in binding lists:

```scheme
(let [(x 3) (y 4)] (+ x y))
```

### Core Forms

```scheme
(define x 10)                           ; variable
(define (square n) (* n n))             ; function shorthand
(define (f x . rest) rest)              ; variadic function
(lambda (x y) (+ x y))                  ; anonymous function
(set! x 99)                             ; mutation
(if (> x 0) "pos" "neg")                ; conditional
(cond [(= x 1) "one"] [else "?"])       ; multi-branch
(case x [(1 2) "low"] [else "hi"])      ; dispatch on eqv?
(let [(a 1) (b 2)] (+ a b))             ; parallel binding
(let* [(a 1) (b (* a 2))] b)            ; sequential binding
(begin expr ...)                        ; sequence
(and expr ...) (or expr ...)            ; short-circuit logic
(quote x) 'x                            ; prevent evaluation
(define-values (lo hi) (values 3 7))    ; multiple values
```

### Macros

glerp supports hygienic macros via `define-syntax` with `syntax-rules` or
`syntax-case`.

```scheme
(define-syntax when
  (syntax-rules ()
    [(when test body ...)
     (if test (begin body ...))]))
```

### String Interpolation

The `$"..."` syntax embeds Scheme expressions in strings. Expressions inside
`{...}` are evaluated and converted to strings.

```scheme
(define name "Alice")
$"Hello {name}!"          ; "Hello Alice!"
$"squared: {(* 7 7)}"     ; "squared: 49"
```

### Iteration

```scheme
(do [(i 0 (+ i 1)) (s 0 (+ s i))]
    [(= i 5) s]
  (display i))
```

### Built-in Procedures

```text
+ - * /                arithmetic
< > <= >= =            numeric comparison
not                    boolean negation
car cdr cons           list primitives
list empty?            list utilities
vector vector-ref      vector primitives
hash-table-ref         hash table access
values                 multiple return values
display display-ln     output
```

### Libraries

Import libraries with `(import :prefix/name)`. Core libraries include
`:scheme/list`, `:scheme/math`, and `:scheme/time`.

```scheme
(import :scheme/list)
(import (only :scheme/list map filter))
```

## Embedding

### Evaluate Expressions

```go
env := glerp.NewEnvironment(glerp.DefaultConfig())
results, err := glerp.Eval(`(+ 1 2)`, env)
fmt.Println(results[len(results)-1]) // 3
```

### Register Go Procedures

Add custom builtins to the config before creating the environment.

```go
cfg := glerp.DefaultConfig()
cfg.Builtins["http-get"] = func(args []glerp.Expr) (glerp.Expr, error) {
    // args are pre-evaluated
    return glerp.Void(), nil
}
env := glerp.NewEnvironment(cfg)
```

### Configuration DSLs

Use `Load` to evaluate a file and extract typed values.

```go
cfg, _ := glerp.Load("config.scm")
host, _ := cfg.String("host")
port, _ := cfg.Int("port")
```

### Custom Special Forms

Special forms receive *unevaluated* arguments, allowing you to define custom
evaluation logic or keyword-based DSLs.

```go
cfg.Forms["routes"] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
    for _, arg := range args {
        // manually evaluate or inspect args
    }
    return glerp.Void(), nil
}
```

## CLI

```bash
go install go.e64ec.com/glerp/cmd/glerp@latest

# REPL
glerp

# file runner
glerp script.scm

# eval expressions from stdin
echo '(display-ln "hi")' | glerp -
```

## Development

```bash
task check  # run linter, vetter and tests
```
