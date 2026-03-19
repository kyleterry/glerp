; glerp prelude — project-specific sugar
(import :glerp/sugar
        :glerp/struct)

(export define-syntax*
        struct
        empty?
        display-ln)

(define empty? null?)

(define (display-ln x) (display x) (newline))
