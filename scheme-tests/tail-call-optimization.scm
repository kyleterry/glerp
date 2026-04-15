;;; Tail call optimization tests
;;;
;;; These tests verify that tail calls do not grow the Go stack. A depth
;;; of 1 000 000 would overflow without TCO, so surviving the call is
;;; the assertion itself.

(import :scheme/list)

;; simple tail-recursive countdown
(define (countdown n)
  (if (= n 0) 0
      (countdown (- n 1))))

(check (= (countdown 1000000) 0) "simple countdown")

;; accumulator-style tail recursion
(define (sum-to n acc)
  (if (= n 0) acc
      (sum-to (- n 1) (+ n acc))))

(check (= (sum-to 100000 0) 5000050000) "accumulator sum")

;; mutual tail recursion
(define (my-even? n)
  (if (= n 0) #t
      (my-odd? (- n 1))))

(define (my-odd? n)
  (if (= n 0) #f
      (my-even? (- n 1))))

(check (my-even? 100000)       "mutual recursion even")
(check (not (my-odd? 100000))  "mutual recursion odd")
(check (my-odd? 99999)         "mutual recursion odd true")

;; tail position in if consequent and alternate
(define (if-tail n)
  (if (> n 0)
      (if-tail (- n 1))
      "done"))

(check (equal? (if-tail 1000000) "done") "if branches are tail position")

;; tail position in cond clauses
(define (cond-tail n)
  (cond
    ((= n 0) "zero")
    ((> n 0) (cond-tail (- n 1)))
    (else    "negative")))

(check (equal? (cond-tail 1000000) "zero") "cond body is tail position")

;; tail position in case
(define (case-tail n)
  (case n
    ((0) "done")
    (else (case-tail (- n 1)))))

(check (equal? (case-tail 100000) "done") "case body is tail position")

;; tail position in let body
(define (let-tail n)
  (let ((x n))
    (if (= x 0) "done"
        (let-tail (- x 1)))))

(check (equal? (let-tail 1000000) "done") "let body is tail position")

;; tail position in let* body
(define (let*-tail n)
  (let* ((x n)
         (half (/ x 2)))
    (if (= x 0) half
        (let*-tail (- x 1)))))

(check (= (let*-tail 1000000) 0) "let* body is tail position")

;; tail position in begin
(define (begin-tail n)
  (begin
    (+ 1 2)
    (if (= n 0) "done"
        (begin-tail (- n 1)))))

(check (equal? (begin-tail 1000000) "done") "begin last expr is tail position")

;; tail position in and
(define (and-tail n)
  (if (= n 0) #t
      (and #t (and-tail (- n 1)))))

(check (and-tail 1000000) "and last expr is tail position")

;; tail position in or
(define (or-tail n)
  (if (= n 0) #t
      (or #f (or-tail (- n 1)))))

(check (or-tail 1000000) "or last expr is tail position")

;; tail position in do result
(define (do-result n)
  (do ((i 0 (+ i 1)))
      ((= i n) (countdown 1000000))))

(check (= (do-result 3) 0) "do result is tail position")

;; map with a tail-recursive callback
(define (reverse-acc lst acc)
  (if (null? lst) acc
      (reverse-acc (cdr lst) (cons (car lst) acc))))

(check (equal? (reverse-acc '(1 2 3 4 5) '())
               '(5 4 3 2 1))
       "tail-recursive reverse")

;; apply with a tail-recursive procedure
(check (= (apply sum-to '(100000 0)) 5000050000) "apply + tail recursion")

;; nested tail calls through multiple forms
(define (nested-tail n)
  (cond
    ((= n 0) "done")
    ((even? n)
     (let ((m (- n 1)))
       (begin
         (and #t (nested-tail m)))))
    (else
     (or #f (nested-tail (- n 1))))))

(check (equal? (nested-tail 100000) "done") "nested tail positions compose")
