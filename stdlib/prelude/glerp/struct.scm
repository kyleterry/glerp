;;; structs: vector-backed record types via syntax-case
;;;
;;; (struct name field ...)
;;; (struct name field ...
;;;   (methods
;;;     (method-name (self arg ...) body ...) ...))
;;;
;;; Expands to:
;;;   (make-<name> field ...)   constructor
;;;   (<name>? obj)             type predicate
;;;   (<name>-<field> obj)      getter per field
;;;   (set-<name>-<field>! obj val)  setter per field
;;;   (<name>-<method> self arg ...)  method per method-def
;;;
;;; Memory layout: #(<name-tag> field0 field1 ...)

(import :glerp/sugar)

(export struct)

(define (struct-iota n)
  (define (loop i acc)
    (if (= i 0) acc
        (loop (- i 1) (cons i acc))))
  (loop n '()))

(define (struct-build-method name mdef)
  (let ([mname    (car mdef)]
        [margs    (cadr mdef)]
        [mbody    (cddr mdef)]
        [prefixed (string->symbol $"{name}-{(car mdef)}")])
    `(define (,prefixed ,@margs) ,@mbody)))

(define-syntax* (struct stx) (methods)
  [(_ name field ... (methods method-def ...))
       (let* ([fields  (syntax (field ...))]
              [n       (length fields)]
              [ctor    (string->symbol $"make-{name}")]
              [pred    (string->symbol $"{name}?")]
              [getters (map (lambda (f) (string->symbol $"{name}-{f}")) fields)]
              [setters (map (lambda (f) (string->symbol $"set-{name}-{f}!")) fields)]
              [indices (struct-iota n)]
              [mdefs   (syntax (method-def ...))]
              [mdefines (map (lambda (m) (struct-build-method name m)) mdefs)])
         (with-syntax ([ctor ctor]
                       [pred pred]
                       [(getter ...) getters]
                       [(setter ...) setters]
                       [(idx ...) indices]
                       [(mdefine ...) mdefines])
           (syntax
             (begin
               (define (ctor field ...)
                 (vector (quote name) field ...))
               (define (pred obj)
                 (and (vector? obj)
                      (> (vector-length obj) 0)
                      (eq? (vector-ref obj 0) (quote name))))
               (define (getter obj) (vector-ref obj idx)) ...
               (define (setter obj val) (vector-set! obj idx val)) ...
               mdefine ...))))]
  [(_ name field ...)
   (syntax (struct2 name field ... (methods)))])
