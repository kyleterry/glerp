;;; glerp-demo — configuration using a custom DSL
;;;
;;; (app), (server), (database), (routes), (features), and the HTTP
;;; method forms (GET) (POST) (DELETE) are all registered from Go.
;;; Normal Scheme expressions work anywhere a value is expected.

;;; Top-level constants — available throughout the config.
(define cpu-count 4)
(define debug     #f)

(app "glerp-demo" "1.0.0"

  (server
    (host          "127.0.0.1")
    (port          8080)
    (workers       (* cpu-count 2))          ;;; → 8
    (read-timeout  (if debug 300 30))        ;;; → 30  (300 when debug)
    (write-timeout (if debug 300 30)))

  (database
    (host      "localhost")
    (port      5432)
    (name      "glerp_demo")
    (pool-size (+ cpu-count 2)))             ;;; → 6

  (routes
    (GET    "/"              "index")
    (GET    "/health"        "health-check")
    (GET    "/api/users"     "list-users")
    (POST   "/api/users"     "create-user")
    (DELETE "/api/users/:id" "delete-user")
    (GET    "/api/metrics"   "metrics"))

  (features
    'rate-limiting
    'request-logging
    'gzip-compression))
