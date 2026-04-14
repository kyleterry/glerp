(import :scheme/list)

(check (equal? '(1 2 3) (list 1 2 3)))
(check (equal? '(a b c) (append '(a) '(b c))))
(check (= (length '(1 2 3 4)) 4))
(check (equal? '(3 2 1) (reverse '(1 2 3))))
(check (equal? '(2 4 6) (map (lambda (x) (* x 2)) '(1 2 3))))
(check (equal? '(3 4) (filter (lambda (x) (> x 2)) '(1 2 3 4))))
(check (= (fold + 0 '(1 2 3 4 5)) 15))
