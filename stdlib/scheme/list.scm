; (scheme list) — list utilities
(export range)

(import (only :core/list reverse))

;; Returns a list of integers from start (inclusive) to end (exclusive),
;; stepping by step. With one argument, returns 0..n-1. With two arguments,
;; returns start..end-1. With three arguments, applies the given step.
;; (range 5)       => (0 1 2 3 4)
;; (range 2 5)     => (2 3 4)
;; (range 0 10 2)  => (0 2 4 6 8)
(define (range start-or-end . rest)
  (let* [(start (if (null? rest) 0 start-or-end))
        (end (if (null? rest) start-or-end (car rest)))
        (step (if (or (null? rest) (null? (cdr rest))) 1 (cadr rest)))
        (build (lambda (i acc)
                 (if (if (> step 0) (>= i end) (<= i end))
                    (reverse acc)
                    (build (+ i step) (cons i acc)))))]
    (build start '())))
