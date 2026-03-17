;;; report.scm -- weekly temperature analysis
;;;
;;; Demonstrates define-syntax macros:
;;;   when / unless  -- new control-flow forms
;;;   ->>            -- thread-last pipeline operator
;;;   check          -- assertion that captures its own source text

(import :scheme/list :scheme/math)

;; ── macros ────────────────────────────────────────────────────────────────────

;; (when test body ...) -- evaluate body forms only when test is truthy.
(define-syntax when
  (syntax-rules ()
    [(_ test body ...)
     (if test (begin body ...))]))

;; (unless test body ...) -- evaluate body forms only when test is falsy.
(define-syntax unless
  (syntax-rules ()
    [(_ test body ...)
     (if (not test) (begin body ...))]))

;; (->> val step ...) -- thread val through steps as the last argument.
;; A step can be a bare name or a partial application (f arg ...).
;;
;;   (->> '(1 2 3) (filter odd?) length)
;;   => (length (filter odd? '(1 2 3)))
(define-syntax ->>
  (syntax-rules ()
    [(_ x)                       x]
    [(_ x (f arg ...) rest ...)  (->> (f arg ... x) rest ...)]
    [(_ x f         rest ...)    (->> (f x)         rest ...)]))

;; (check expr) -- evaluate expr and report pass/fail.
;; The source text of expr is captured at expansion time via quote,
;; so the output shows the original code rather than evaluated values.
(define-syntax check
  (syntax-rules ()
    [(_ expr)
     (report-check (quote expr) expr)]))

;; ── data ──────────────────────────────────────────────────────────────────────

;; Each reading is (day temp-in-celsius).
(define readings
  '((Mon 21) (Tue 18) (Wed 24) (Thu 18) (Fri 28) (Sat 24) (Sun 21)))

(define (day r)  (car r))
(define (temp r) (car (cdr r)))

;; ── statistics ────────────────────────────────────────────────────────────────

(define temps  (map temp readings))
(define n      (length temps))
(define total  (fold + 0 temps))
(define mean   (/ total n))
(define hi     (fold max (car temps) (cdr temps)))
(define lo     (fold min (car temps) (cdr temps)))
(define spread (- hi lo))

;; Count above-average days using the thread-last macro.
;; Read as: take temps, keep those above the mean, count them.
(define above-avg
  (->> temps
    (filter (lambda (t) (> t mean)))
    length))

;; ── report ────────────────────────────────────────────────────────────────────

(print-header "Weekly Temperature Report")
(for-each (lambda (r)
            (print-reading (day r) (temp r) mean))
          readings)

(print-header "Summary")
(print-kv "days recorded"  n)
(print-kv "mean"           mean)
(print-kv "high"           hi)
(print-kv "low"            lo)
(print-kv "spread"         spread)
(print-kv "above average"  above-avg)

(when (>= mean 24)
  (print-kv "note" "warm week"))

(unless (>= mean 24)
  (print-kv "note" "cool week"))

;; ── checks ────────────────────────────────────────────────────────────────────

(print-header "Checks")
(check (= n 7))
(check (= mean 22))
(check (= hi 28))
(check (= lo 18))
(check (= spread 10))
(check (> above-avg 0))
(check (< lo mean))
(check (< mean hi))
