# glerp

A small Scheme interpreter for embedding in Go programs. It is designed to
serve as a scripting and configuration layer inside larger applications. You
can evaluate Scheme expressions, extend the language with Go-backed procedures,
or build a fully custom DSL by registering your own special forms.

```
go get go.e64ec.com/glerp
```

## Language

glerp implements a practical subset of Scheme with a few non-standard
extensions that suit its use as an embedded DSL.

### Literals

```scheme
42          ; integer
3.14        ; float
"hello"     ; string
#t  #f      ; booleans
'()         ; empty list
'(1 2 3)    ; quoted list
```

Square brackets may be used anywhere in place of parentheses. They are
especially useful for binding lists in `let`, `do`, and similar forms:

```scheme
(let  [(x 3) (y 4)] (+ x y))
(let* [(x 3) (y (* x 2))] y)
```

### Core forms

```scheme
(define x 10)                        ; variable
(define (square n) (* n n))          ; function shorthand
(define (f x . rest) rest)           ; variadic function
(lambda (x y) (+ x y))              ; anonymous function
(set! x 99)                          ; mutation
(if (> x 0) "pos" "neg")            ; conditional (else clause optional)
(cond [(= x 1) "one"] [else "?"])   ; multi-branch conditional
(case x [(1 2) "low"] [else "hi"])  ; dispatch on eqv? value
(let  [(a 1) (b 2)] (+ a b))        ; parallel bindings
(let* [(a 1) (b (* a 2))] b)        ; sequential bindings
(begin expr ...)                     ; sequence, returns last
(and expr ...)  (or expr ...)        ; short-circuit logic
(quote x)  'x                        ; prevent evaluation
(define-values (lo hi) (values 3 7)) ; multiple values
```

### Quasiquote

`` ` `` is shorthand for `quasiquote`, `,` for `unquote`, and `,@` for
`unquote-splicing`.

```scheme
(define x 42)
(define xs '(2 3))

`(a ,x c)          ; => (a 42 c)
`(a ,@xs d)        ; => (a 2 3 d)
`(a ,(+ 1 2) c)    ; => (a 3 c)
```

### String interpolation

The `$"..."` syntax embeds Scheme expressions inside string literals. Any
expression inside `{...}` is evaluated and converted to a string with
`->string`.

```scheme
(define name "Alice")
$"Hello {name}!"          ; => "Hello Alice!"
$"squared: {(* 7 7)}"     ; => "squared: 49"
```

### Iteration

```scheme
(do [(i 0 (+ i 1))       ; var init step
     (s 0 (+ s i))]
    [(= i 5) s]           ; test result-expr
  (display i))            ; body (optional, for side effects)
```

### Built-in procedures

```
+  -  *  /             arithmetic (variadic)
<  >  <=  >=  =        numeric comparison
not                    boolean negation
car  cdr  cons         list primitives
caar cadr ... cddddr   car/cdr compositions (up to 4 deep)
list  empty?           list utilities
values                 multiple return values
string-append          concatenate strings
->string               convert any value to a string
display  display-ln    output
newline                print a newline
```

### Libraries

glerp ships with a few importable libraries (lists, math, time). Import them
with `(import :prefix/name)`:

```scheme
(import :scheme/list)
(import (only :scheme/list map filter))  ; selective import
(import :scheme/list :scheme/math)       ; multiple in one call
```

You can also create your own libraries — both Scheme-file and Go-backed — and
register them when building an environment (see embedding below).

## Embedding glerp

### Evaluate expressions

```go
env := glerp.NewEnvironment(glerp.DefaultConfig())

results, err := glerp.Eval(`(+ 1 2)`, env)
if err != nil {
    log.Fatal(err)
}
fmt.Println(results[len(results)-1]) // 3
```

### Register a Go procedure

Add custom builtins to the config before creating the environment. The
function receives pre-evaluated arguments.

```go
cfg := glerp.DefaultConfig()
cfg.Builtins["http-get"] = func(args []glerp.Expr) (glerp.Expr, error) {
    url, ok := args[0].(*glerp.StringExpr)
    if !ok {
        return nil, fmt.Errorf("http-get: expected string url")
    }
    resp, err := http.Get(url.Value())
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    body, _ := io.ReadAll(resp.Body)
    // ... use body ...
    return glerp.Void(), nil
}

env := glerp.NewEnvironment(cfg)
```

### Extract typed values

Use `Load` to evaluate a config file and pull out typed values:

```go
cfg, err := glerp.Load("config.scm")
if err != nil {
    log.Fatal(err)
}

host, _  := cfg.String("host")
port, _  := cfg.Int("port")
debug, _ := cfg.Bool("debug")
tags, _  := cfg.Strings("tags")
```

For more control, use `EvalFile` with a prepared environment.

### Register a custom library

Add libraries to the config to make them importable via `(import
:prefix/name)`. Libraries can be Scheme files (via `go:embed`) or Go-backed
builtins.

```go
//go:embed mylibs
var myLibs embed.FS

cfg := glerp.DefaultConfig()
cfg.Libraries = append(cfg.Libraries, glerp.Library{
    Prefix: "myapp",
    FS:     myLibs,
})

env := glerp.NewEnvironment(cfg)
```

```scheme
(import :myapp/utils)
(greet "world")  ; calls a function defined in mylibs/utils.scm
```

## Building a DSL

Special forms receive their arguments *unevaluated*, giving you full control
over evaluation semantics. This is how you build keyword-style DSLs.

**`routes.scm`**

```scheme
(routes
  (GET  "/health"    "health-check")
  (GET  "/api/users" "list-users")
  (POST "/api/users" "create-user"))
```

**`main.go`**

```go
package main

import (
    "fmt"
    "log"

    "go.e64ec.com/glerp"
)

type Route struct {
    Method, Path, Handler string
}

func main() {
    var routes []Route

    cfg := glerp.DefaultConfig()

    for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
        m := method
        cfg.Forms[m] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
            path, _ := args[0].Eval(env)
            handler, _ := args[1].Eval(env)
            routes = append(routes, Route{
                Method:  m,
                Path:    path.(*glerp.StringExpr).Value(),
                Handler: handler.(*glerp.StringExpr).Value(),
            })
            return glerp.Void(), nil
        }
    }

    cfg.Forms["routes"] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
        for _, arg := range args {
            if _, err := arg.Eval(env); err != nil {
                return nil, err
            }
        }
        return glerp.Void(), nil
    }

    env := glerp.NewEnvironment(cfg)
    if err := glerp.EvalFile("routes.scm", env); err != nil {
        log.Fatal(err)
    }

    for _, r := range routes {
        fmt.Printf("%-8s %-24s -> %s\n", r.Method, r.Path, r.Handler)
    }
}
```

## REPL and file runner

```
go install go.e64ec.com/glerp/cmd/glerp@latest
```

```
glerp                          # start the REPL
glerp script.scm               # evaluate a file
echo '(display "hi")' | glerp  # evaluate from stdin
```

## Development

```
nix develop   # enter the dev shell (go 1.26, gopls, golangci-lint, gotools)

go test ./...
go vet ./...
golangci-lint run ./...
```
