(check (pair? (cons 1 2)))
(check (not (pair? '())))
(check (not (pair? 42)))

(define p (cons 'a 'b))
(check (eq? (car p) 'a))
(check (eq? (cdr p) 'b))

(set-car! p 1)
(check (= (car p) 1))
(set-cdr! p 2)
(check (= (cdr p) 2))

(define nested (cons 1 (cons 2 (cons 3 '()))))
(check (= (car nested) 1))
(check (= (car (cdr nested)) 2))
(check (= (car (cdr (cdr nested))) 3))
(check (null? (cdr (cdr (cdr nested)))))

;; Test cxr builtins (cadr, caddr, etc.)
(check (= (cadr nested) 2))
(check (= (caddr nested) 3))

(check (list? nested))
(check (not (list? (cons 1 2)))) ; Dotted pair is not a proper list
(check (list? '()))
