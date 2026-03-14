# glerp

A small Scheme interpreter for embedding in Go programs. It is designed to
serve as a scripting and configuration layer inside larger applications -- you
can evaluate Scheme expressions, extend the language with Go-backed procedures,
or build a fully custom DSL by registering your own special forms.

```
go get go.e64ec.com/glerp
```


## Language

glerp implements a practical subset of Scheme.

### Literals

```scheme
42          ; integer
3.14        ; float
"hello"     ; string
#t  #f      ; booleans
'()         ; empty list
'(1 2 3)    ; quoted list
'[1 2 3]    ; square brackets are interchangeable with parentheses
```

Square brackets may be used anywhere in place of parentheses. They are
especially useful for binding lists in `let`, `do`, and similar forms, since
the visual distinction helps separate the bindings from the body:

```scheme
(let  [(x 3) (y 4)] (+ x y))
(let* [(x 3) (y (* x 2))] y)
(do   [(i 0 (+ i 1))] [(= i 5) i])
```

### Core forms

```scheme
(define x 10)                        ; variable
(define (square n) (* n n))          ; function shorthand
(lambda (x y) (+ x y))              ; anonymous function
(set! x 99)                          ; mutation
(if (> x 0) "pos" "neg")            ; conditional (else clause optional)
(cond [(= x 1) "one"] [else "?"])   ; multi-branch conditional
(case x [(1 2) "low"] [else "hi"])  ; dispatch on eqv? value
(let  [(a 1) (b 2)] (+ a b))        ; parallel bindings
(let* [(a 1) (b (* a 2))] b)        ; sequential bindings
(begin expr ...)                     ; sequence, returns last
(and expr ...)                       ; short-circuit and
(or  expr ...)                       ; short-circuit or
(quote x)  'x                        ; quoting
```

### Lists

```scheme
(cons 1 '(2 3))   ; => (1 2 3)
(car '(1 2 3))    ; => 1
(cdr '(1 2 3))    ; => (2 3)
(list 1 2 3)      ; => (1 2 3)
(empty? '())      ; => #t
```

### Iteration

```scheme
(do [(i 0 (+ i 1))       ; var init step
     (s 0 (+ s i))]
    [(= i 5) s]           ; test result-expr
  (display i))            ; body (optional, for side effects)
```

### Multiple values

```scheme
(define-values (lo hi) (values 3 7))
(+ lo hi)  ; => 10
```

### Standard library

Import one or more library specs at the top of a file. All definitions from
the library are bound in the current scope.

```scheme
(import :scheme/list)
(import :scheme/math)
(import :scheme/time)

; Selective import -- only bind the named symbols.
(import (only :scheme/list map filter))

; Multiple specs in one call.
(import :scheme/list :scheme/math)
```

**`:scheme/list`** -- `length`, `append`, `reverse`, `list-ref`, `list-tail`,
`map`, `filter`, `for-each`, `fold`

**`:scheme/math`** -- `abs`, `max`, `min`, `square`, `cube`, `average`,
`clamp`

**`:scheme/time`** -- `current-time`, `current-second`, `current-jiffy`,
`jiffies-per-second`, `make-time`, `time-components`, `time-year`,
`time-month`, `time-day`, `time-hour`, `time-minute`, `time-second`,
`time-weekday`, `time-weekday-name`, `time-month-name`, `seconds`, `minutes`,
`hours`, `days`, `weeks`, `time-add`, `time-subtract`, `time-difference`,
`time<?`, `time>?`, `time=?`, `time<=?`, `time>=?`, `time->string`,
`time->string/fmt`, `string->time`, `string->time/fmt`, `time-format/iso`,
`time-format/date`, `time-format/time`, `time-format/datetime`

### Built-in procedures

```
+  -  *  /             arithmetic (variadic where it makes sense)
<  >  <=  >=  =        numeric comparison
not                    boolean negation
car  cdr  cons         list constructors and accessors
list  empty?           list utilities
values                 multiple return values
display  display-ln    output (display omits surrounding quotes on strings)
newline                print a newline
```


## Embedding glerp

### Evaluate expressions

```go
import "go.e64ec.com/glerp"

env := glerp.NewEnvironment(glerp.StandardBuiltins(), glerp.StandardForms())

results, err := glerp.Eval(`(define (fact n) (if (= n 0) 1 (* n (fact (- n 1))))) (fact 6)`, env)
if err != nil {
    log.Fatal(err)
}
fmt.Println(results[len(results)-1]) // 720
```

### Evaluate a file

```go
env := glerp.NewEnvironment(glerp.StandardBuiltins(), glerp.StandardForms())
if err := glerp.EvalFile("script.scm", env); err != nil {
    log.Fatal(err)
}
```

### Register a Go procedure

Add custom procedures before creating the environment by extending the builtins
map. The function receives pre-evaluated arguments.

```go
builtins := glerp.StandardBuiltins()
builtins["http-get"] = func(args []glerp.Expr) (glerp.Expr, error) {
    if len(args) != 1 {
        return nil, fmt.Errorf("http-get: expected 1 argument")
    }
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
    return &glerp.StringExpr{}, nil // simplified
}

env := glerp.NewEnvironment(builtins, glerp.StandardForms())
```

### Extract typed values

After evaluating a config file, use `NewConfig` to pull out typed values:

```go
cfg, err := glerp.Load("config.scm")
if err != nil {
    log.Fatal(err)
}

host, _    := cfg.String("host")
port, _    := cfg.Int("port")
debug, _   := cfg.Bool("debug")
tags, _    := cfg.Strings("tags")
```

`Load` evaluates the file in a fresh standard environment. For more control,
use `EvalFile` with a prepared environment.


## Building a DSL

glerp's special form mechanism lets you register Go functions that receive
their arguments *unevaluated*, giving you full control over evaluation
semantics. This is how you build keyword-style DSLs.

The following example defines a minimal HTTP routing DSL.

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

    forms := glerp.StandardForms()

    // (GET path handler) and (POST path handler)
    for _, method := range []string{"GET", "POST", "PUT", "DELETE"} {
        m := method
        forms[m] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
            if len(args) != 2 {
                return nil, fmt.Errorf("%s: expected (path handler)", m)
            }
            path, err := args[0].Eval(env)
            if err != nil {
                return nil, err
            }
            handler, err := args[1].Eval(env)
            if err != nil {
                return nil, err
            }
            routes = append(routes, Route{
                Method:  m,
                Path:    path.(*glerp.StringExpr).Value(),
                Handler: handler.(*glerp.StringExpr).Value(),
            })
            return glerp.Void(), nil
        }
    }

    // (routes body...) -- evaluate each child route form
    forms["routes"] = func(args []glerp.Expr, env *glerp.Environment) (glerp.Expr, error) {
        for _, arg := range args {
            if _, err := arg.Eval(env); err != nil {
                return nil, err
            }
        }
        return glerp.Void(), nil
    }

    env := glerp.NewEnvironment(glerp.StandardBuiltins(), forms)
    if err := glerp.EvalFile("routes.scm", env); err != nil {
        log.Fatal(err)
    }

    for _, r := range routes {
        fmt.Printf("%-8s %-24s -> %s\n", r.Method, r.Path, r.Handler)
    }
}
```

Because form handlers receive unevaluated arguments, you can also accept bare
symbols, nested sub-forms, or any mix of evaluated and literal syntax -- the
`cmd/demo` directory contains a more complete example with a server, database,
and feature-flag DSL.


## REPL and file runner

The `cmd/glerp` binary provides a read-eval-print loop and a file runner.

```
go install go.e64ec.com/glerp/cmd/glerp@latest
```

```
glerp              # start the REPL
glerp script.scm  # evaluate a file
```

## Development

```
nix develop   # enter the dev shell (go 1.26, gopls, golangci-lint, gotools)

go test ./...
go vet ./...
golangci-lint run ./...
```
