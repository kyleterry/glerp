; (scheme math) — mathematical utilities
(export cube average clamp pi)

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

;; Returns pi
(define pi 3.14159265358979323846264338327950288419716939937510582097494459)
