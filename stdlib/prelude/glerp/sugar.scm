;;; sugar: syntactic conveniences
;;;
;;; (define-syntax* (name stx) (literal ...) clause ...)
;;;   Desugars to syntax-case wrapped in a lambda:
;;;   (define-syntax name
;;;     (lambda (stx)
;;;       (syntax-case stx (literal ...) clause ...)))
;;;
;;; (define-syntax* name (literal ...) clause ...)
;;;   Desugars to syntax-rules:
;;;   (define-syntax name
;;;     (syntax-rules (literal ...) clause ...))

(export define-syntax*)

(define-syntax define-syntax*
  (lambda (stx)
    (syntax-case stx ()
      [(_ (name arg) (lit ...) clause ...)
       (syntax
         (define-syntax name
           (lambda (arg)
             (syntax-case arg (lit ...) clause ...))))]
      [(_ name (lit ...) clause ...)
       (syntax
         (define-syntax name
           (syntax-rules (lit ...) clause ...)))])))
