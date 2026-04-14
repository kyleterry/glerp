(define (make-adder n)
  (lambda (x) (+ n x)))

(define add3 (make-adder 3))
(check (= (add3 7) 10))

(define (factorial n)
  (if (<= n 1)
      1
      (* n (factorial (- n 1)))))

(check (= (factorial 5) 120))
(check (= (factorial 0) 1))

(define (fib n)
  (if (< n 2)
      n
      (+ (fib (- n 1)) (fib (- n 2)))))

(check (= (fib 10) 55))

;; local binding shadow
(define x 10)
(check (let ((x 20)) (= x 20)))
(check (= x 10))
