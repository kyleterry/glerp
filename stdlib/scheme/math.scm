; (scheme math) — mathematical utilities
(export cube average clamp)

;; Returns x raised to the third power.
(define (cube x)
  (* x x x))

;; Returns the arithmetic mean of a and b.
(define (average a b)
  (/ (+ a b) 2))

;; Returns x clamped to the closed interval [lo, hi].
;; If x < lo, returns lo. If x > hi, returns hi. Otherwise returns x.
(define (clamp x lo hi)
  (if (< x lo) lo (if (> x hi) hi x)))
