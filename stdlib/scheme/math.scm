; (scheme math) — mathematical utilities

;; Returns the absolute value of x.
(define (abs x)
  (if (< x 0) (- x) x))

;; Returns the larger of a and b.
(define (max a b)
  (if (> a b) a b))

;; Returns the smaller of a and b.
(define (min a b)
  (if (< a b) a b))

;; Returns x multiplied by itself.
(define (square x)
  (* x x))

;; Returns x raised to the third power.
(define (cube x)
  (* x x x))

;; Returns the arithmetic mean of a and b.
(define (average a b)
  (/ (+ a b) 2))

;; Returns x clamped to the closed interval [lo, hi].
;; If x < lo, returns lo. If x > hi, returns hi. Otherwise returns x.
(define (clamp x lo hi)
  (min (max x lo) hi))
