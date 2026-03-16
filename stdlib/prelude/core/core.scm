; core prelude — standard Scheme fundamentals
(import :core/list)
(import (only :go/time current-second current-jiffy))

(export length append reverse list-ref list-tail
        map filter for-each fold
        zero? positive? negative? even? odd?
        abs max min square
        current-second current-jiffy jiffies-per-second)

;; Numeric predicates
(define (zero? n) (= n 0))
(define (positive? n) (> n 0))
(define (negative? n) (< n 0))
(define (even? n) (= (modulo n 2) 0))
(define (odd? n) (not (even? n)))

;; Numeric utilities (R5RS/R7RS)
(define (abs x) (if (< x 0) (- x) x))
(define (max a b) (if (> a b) a b))
(define (min a b) (if (< a b) a b))
(define (square x) (* x x))

;; Time (R7RS)
(define (jiffies-per-second) 1000000000)
