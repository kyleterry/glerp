(define x 1)
(define y 2)

(check (equal? `(1 ,(+ 1 1) 3) '(1 2 3)))
(check (equal? `(1 ,@'(2 3) 4) '(1 2 3 4)))

(define-syntax my-if
  (syntax-rules ()
    [(_ test then else) (if test then else)]))

(check (= (my-if #t 10 20) 10))
(check (= (my-if #f 10 20) 20))

(define-syntax my-let
  (syntax-rules ()
    [(_ ((var val) ...) body ...)
     ((lambda (var ...) body ...) val ...)]))

(check (= (my-let ((x 5) (y 10)) (+ x y)) 15))

(check (= (syntax-case #'(1 2 3) ()
           [(a b c) (syntax->datum #'b)])
          2))
