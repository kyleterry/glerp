package glerp_test

import (
	"testing"
	"testing/fstest"

	"github.com/matryer/is"
	"go.e64ec.com/glerp"
)

func TestPrelude(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		// core prelude: numeric predicates
		{"zero? true", "(zero? 0)", "#t"},
		{"zero? false", "(zero? 5)", "#f"},
		{"positive? true", "(positive? 3)", "#t"},
		{"positive? false", "(positive? -1)", "#f"},
		{"negative? true", "(negative? -3)", "#t"},
		{"negative? false", "(negative? 1)", "#f"},
		{"even? true", "(even? 4)", "#t"},
		{"even? false", "(even? 3)", "#f"},
		{"odd? true", "(odd? 3)", "#t"},
		{"odd? false", "(odd? 4)", "#f"},

		// core prelude: math utilities
		{"abs positive", "(abs 5)", "5"},
		{"abs negative", "(abs -7)", "7"},
		{"max", "(max 3 7)", "7"},
		{"min", "(min 3 7)", "3"},
		{"square", "(square 4)", "16"},

		// core prelude: list functions (available without import)
		{"length", "(length '(a b c))", "3"},
		{"append", "(append '(1 2) '(3 4))", "(1 2 3 4)"},
		{"reverse", "(reverse '(3 2 1))", "(1 2 3)"},
		{"map", "(map (lambda (x) (* x 2)) '(1 2 3))", "(2 4 6)"},
		{"filter", "(filter (lambda (x) (> x 2)) '(1 2 3 4))", "(3 4)"},
		{"fold", "(fold + 0 '(1 2 3 4 5))", "15"},

		// core prelude: R7RS time procedures (no import needed)
		{"current-second positive", "(> (current-second) 0)", "#t"},
		{"jiffies-per-second", "(jiffies-per-second)", "1000000000"},
		{"current-jiffy positive", "(> (current-jiffy) 0)", "#t"},
		{"jiffy elapsed", "(let ((a (current-jiffy)) (b (current-jiffy))) (>= b a))", "#t"},

		// glerp prelude: empty? alias
		{"empty? alias true", "(empty? '())", "#t"},
		{"empty? alias false", "(empty? '(1))", "#f"},
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

func TestStdlibImport(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		// :scheme/list
		{"list length", "(import :scheme/list) (length '(a b c d))", "4"},
		{"list append", "(import :scheme/list) (append '(1 2) '(3 4))", "(1 2 3 4)"},
		{"list reverse", "(import :scheme/list) (reverse '(1 2 3))", "(3 2 1)"},
		{"list map", "(import :scheme/list) (map (lambda (x) (* x 2)) '(1 2 3))", "(2 4 6)"},
		{"list filter", "(import :scheme/list) (filter (lambda (x) (> x 2)) '(1 2 3 4))", "(3 4)"},
		{"list fold", "(import :scheme/list) (fold + 0 '(1 2 3 4 5))", "15"},
		{"list list-ref", "(import :scheme/list) (list-ref '(a b c) 1)", "b"},
		{"list list-tail", "(import :scheme/list) (list-tail '(a b c d) 2)", "(c d)"},

		// :scheme/math
		{"math abs pos", "(import :scheme/math) (abs 5)", "5"},
		{"math abs neg", "(import :scheme/math) (abs -7)", "7"},
		{"math max", "(import :scheme/math) (max 3 7)", "7"},
		{"math min", "(import :scheme/math) (min 3 7)", "3"},
		{"math square", "(import :scheme/math) (square 4)", "16"},
		{"math cube", "(import :scheme/math) (cube 3)", "27"},
		{"math average", "(import :scheme/math) (average 4 6)", "5"},
		{"math clamp lo", "(import :scheme/math) (clamp -5 0 10)", "0"},
		{"math clamp hi", "(import :scheme/math) (clamp 15 0 10)", "10"},
		{"math clamp in", "(import :scheme/math) (clamp 5 0 10)", "5"},

		// :scheme/time
		{"time make-time year", "(import :scheme/time) (time-year (make-time 2024 3 15 12 0 0))", "2024"},
		{"time make-time month", "(import :scheme/time) (time-month (make-time 2024 3 15 12 0 0))", "3"},
		{"time make-time day", "(import :scheme/time) (time-day (make-time 2024 3 15 12 0 0))", "15"},
		{"time make-time hour", "(import :scheme/time) (time-hour (make-time 2024 3 15 12 30 45))", "12"},
		{"time make-time minute", "(import :scheme/time) (time-minute (make-time 2024 3 15 12 30 45))", "30"},
		{"time make-time second", "(import :scheme/time) (time-second (make-time 2024 3 15 12 30 45))", "45"},
		{"time weekday", "(import :scheme/time) (time-weekday (make-time 2024 3 15 0 0 0))", "5"},
		{"time weekday-name", `(import :scheme/time) (time-weekday-name (make-time 2024 3 15 0 0 0))`, `"Friday"`},
		{"time month-name", `(import :scheme/time) (time-month-name (make-time 2024 3 15 0 0 0))`, `"March"`},
		{"time duration seconds", "(import :scheme/time) (seconds 5)", "5"},
		{"time duration minutes", "(import :scheme/time) (minutes 2)", "120"},
		{"time duration hours", "(import :scheme/time) (hours 1)", "3600"},
		{"time duration days", "(import :scheme/time) (days 1)", "86400"},
		{"time duration weeks", "(import :scheme/time) (weeks 1)", "604800"},
		{"time time-add", "(import :scheme/time) (let ((t (make-time 2024 1 1 0 0 0))) (time-year (time-add t (days 366))))", "2025"},
		{"time time-difference", "(import :scheme/time) (time-difference (seconds 100) (seconds 30))", "70"},
		{"time time<?", "(import :scheme/time) (time<? (seconds 1) (seconds 2))", "#t"},
		{"time time>?", "(import :scheme/time) (time>? (seconds 2) (seconds 1))", "#t"},
		{"time time=?", "(import :scheme/time) (time=? (seconds 5) (seconds 5))", "#t"},
		{"time time<=?", "(import :scheme/time) (time<=? (seconds 3) (seconds 3))", "#t"},
		{"time time>=?", "(import :scheme/time) (time>=? (seconds 4) (seconds 3))", "#t"},
		{"time time->string", `(import :scheme/time) (time->string (make-time 2024 3 15 12 0 0))`, `"2024-03-15T12:00:00Z"`},
		{"time string->time round-trip", `(import :scheme/time) (time-day (string->time "2024-03-15T12:00:00Z"))`, "15"},
		{"time time->string/fmt date", `(import :scheme/time) (time->string/fmt (make-time 2024 3 15 0 0 0) time-format/date)`, `"2024-03-15"`},
		{"time time-components length", `(import :scheme/time) (length (time-components (make-time 2024 1 1 0 0 0)))`, "7"},

		// multiple specs in one import
		{"multi import", "(import :scheme/list :scheme/math) (cube (length '(a b c)))", "27"},

		// (only ...) selective import
		{"import only", "(import (only :core/list map filter)) (map (lambda (x) (* x x)) '(1 2 3))", "(1 4 9)"},
		{"import only excludes others", `
			(import (only :scheme/math cube))
			(define average "not imported")
			average
		`, `"not imported"`},
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

func TestStdlibImportErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"unknown library", "(import :scheme/nonexistent)"},
		{"unrecognized path", "(import foo/bar)"},
		{"only nonexported", "(import (only :scheme/list nonexistent-fn))"},
		{"only unknown modifier", "(import (xyzzy :scheme/list map))"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(glerp.DefaultConfig())
			_, err := glerp.Eval(tt.src, env)
			is.True(err != nil)
		})
	}
}

func TestCustomLibrary(t *testing.T) {
	testFS := fstest.MapFS{
		"utils.scm": &fstest.MapFile{
			Data: []byte(`(define (square x) (* x x))`),
		},
		"math/extra.scm": &fstest.MapFile{
			Data: []byte(`(define (cube x) (* x x x))`),
		},
	}

	cfg := glerp.DefaultConfig()
	cfg.Libraries = append(cfg.Libraries, glerp.Library{
		Prefix: "testpkg",
		FS:     testFS,
	})

	tests := []struct {
		name string
		src  string
		want string
	}{
		{"top-level lib", "(import :testpkg/utils) (square 5)", "25"},
		{"nested lib path", "(import :testpkg/math/extra) (cube 3)", "27"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			is := is.New(t)
			env := glerp.NewEnvironment(cfg)
			results, err := glerp.Eval(tt.src, env)
			is.NoErr(err)
			is.Equal(results[len(results)-1].String(), tt.want)
		})
	}
}
