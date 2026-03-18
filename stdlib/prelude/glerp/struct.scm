;;; structs: vector-backed record types via syntax-case
;;;
;;; (struct name field ...)
;;;
;;; Expands to:
;;;   (make-<name> field ...)   constructor
;;;   (<name>? obj)             type predicate
;;;   (<name>-<field> obj)      getter per field
;;;   (set-<name>-<field>! obj val)  setter per field
;;;
;;; Memory layout: #(<name-tag> field0 field1 ...)

(export struct)

(define (struct-iota n)
  (define (loop i acc)
    (if (= i 0) acc
        (loop (- i 1) (cons i acc))))
  (loop n '()))

(define-syntax struct
  (lambda (stx)
    (syntax-case stx ()
      [(_ name field ...)
       (let* ([fields  (syntax (field ...))]
              [n       (length fields)]
              [ctor    (string->symbol $"make-{name}")]
              [pred    (string->symbol $"{name}?")]
              [getters (map (lambda (f) (string->symbol $"{name}-{f}")) fields)]
              [setters (map (lambda (f) (string->symbol $"set-{name}-{f}!")) fields)]
              [indices (struct-iota n)])
         (with-syntax ([ctor ctor]
                       [pred pred]
                       [(getter ...) getters]
                       [(setter ...) setters]
                       [(idx ...) indices])
           (syntax
             (begin
               (define (ctor field ...)
                 (vector (quote name) field ...))
               (define (pred obj)
                 (and (vector? obj)
                      (> (vector-length obj) 0)
                      (eq? (vector-ref obj 0) (quote name))))
               (define (getter obj) (vector-ref obj idx)) ...
               (define (setter obj val) (vector-set! obj idx val)) ...))))])))
