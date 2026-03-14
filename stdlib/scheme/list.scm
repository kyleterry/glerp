; (scheme list) — list utilities
(export #t)

;; Returns the number of elements in lst.
(define (length lst)
  (if (empty? lst)
      0
      (+ 1 (length (cdr lst)))))

;; Returns a new list consisting of the elements of lst1 followed by
;; the elements of lst2.
(define (append lst1 lst2)
  (if (empty? lst1)
      lst2
      (cons (car lst1) (append (cdr lst1) lst2))))

;; Returns a new list with the elements of lst in reverse order.
(define (reverse lst)
  (define (rev-iter lst acc)
    (if (empty? lst)
        acc
        (rev-iter (cdr lst) (cons (car lst) acc))))
  (rev-iter lst '()))

;; Returns the element at zero-based index n in lst.
;; Signals an error if n is out of range.
(define (list-ref lst n)
  (if (= n 0)
      (car lst)
      (list-ref (cdr lst) (- n 1))))

;; Returns the sublist of lst starting at zero-based index n.
;; Signals an error if n is out of range.
(define (list-tail lst n)
  (if (= n 0)
      lst
      (list-tail (cdr lst) (- n 1))))

;; Returns a new list formed by applying f to each element of lst.
(define (map f lst)
  (if (empty? lst)
      '()
      (cons (f (car lst)) (map f (cdr lst)))))

;; Returns a new list containing only the elements of lst for which
;; pred returns a truthy value.
(define (filter pred lst)
  (if (empty? lst)
      '()
      (if (pred (car lst))
          (cons (car lst) (filter pred (cdr lst)))
          (filter pred (cdr lst)))))

;; Applies f to each element of lst in order, for side effects.
;; Returns an unspecified value.
(define (for-each f lst)
  (if (not (empty? lst))
      (begin
        (f (car lst))
        (for-each f (cdr lst)))))

;; Left fold: combines elements of lst using f, starting from init.
;; (fold f init '(a b c)) => (f (f (f init a) b) c)
(define (fold f init lst)
  (if (empty? lst)
      init
      (fold f (f init (car lst)) (cdr lst))))
