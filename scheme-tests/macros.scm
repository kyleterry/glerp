(define-syntax when
  (syntax-rules ()
    [(_ test body ...)
     (if test (begin body ...))]))

(define x 0)
(when #t (set! x 1))
(check (= x 1))

(when #f (set! x 2))
(check (= x 1))

(define-syntax swap!
  (syntax-rules ()
    [(_ a b)
     (let ((tmp a))
       (set! a b)
       (set! b tmp))]))

(define a 1)
(define b 2)
(swap! a b)
(check (= a 2))
(check (= b 1))

(define-syntax check-not
  (syntax-rules ()
    [(_ expr) (check (not expr))]))

(check-not (= 1 2))
