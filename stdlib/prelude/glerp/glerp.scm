; glerp prelude — project-specific sugar
(import :glerp/struct)
(export empty? display-ln)

(define empty? null?)

(define (display-ln x) (display x) (newline))
