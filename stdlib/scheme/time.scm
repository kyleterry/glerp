;; R7RS + extended time library — (import :scheme/time)
;;
;; Times are represented as real numbers of seconds since the POSIX epoch
;; (1970-01-01T00:00:00 UTC), consistent with R7RS current-second.
;; All operations use UTC.

(export #t)
(import :scheme/list)

;; Returns the current time as a real number of seconds since the POSIX epoch.
(define (current-second)
  (go-extern:current-second))

;; Returns the current time as an integer count of jiffies (nanoseconds).
(define (current-jiffy)
  (go-extern:current-jiffy))

;; Returns the number of jiffies per second (1,000,000,000).
(define (jiffies-per-second)
  1000000000)

;; Returns the current time as a unix timestamp (alias for current-second).
(define (current-time)
  (go-extern:current-second))

;; Constructs a time value from UTC components.
;; month is 1–12; all other fields follow natural ranges.
(define (make-time year month day hour minute second)
  (go-extern:time-make year month day hour minute second))

;; Returns the list (year month day hour minute second weekday) for time t.
;; weekday is 0 (Sunday) through 6 (Saturday).
(define (time-components t)
  (go-extern:time-components t))

;; Returns the four-digit UTC year of time t.
(define (time-year t)
  (list-ref (go-extern:time-components t) 0))

;; Returns the UTC month (1–12) of time t.
(define (time-month t)
  (list-ref (go-extern:time-components t) 1))

;; Returns the UTC day of the month (1–31) of time t.
(define (time-day t)
  (list-ref (go-extern:time-components t) 2))

;; Returns the UTC hour (0–23) of time t.
(define (time-hour t)
  (list-ref (go-extern:time-components t) 3))

;; Returns the UTC minute (0–59) of time t.
(define (time-minute t)
  (list-ref (go-extern:time-components t) 4))

;; Returns the UTC second (0–59) of time t.
(define (time-second t)
  (list-ref (go-extern:time-components t) 5))

;; Returns the day of the week as a number: 0 (Sunday) through 6 (Saturday).
(define (time-weekday t)
  (list-ref (go-extern:time-components t) 6))

;; Returns the full name of the weekday for time t, e.g. "Monday".
(define (time-weekday-name t)
  (list-ref
    '("Sunday" "Monday" "Tuesday" "Wednesday" "Thursday" "Friday" "Saturday")
    (time-weekday t)))

;; Returns the full name of the month for time t, e.g. "March".
;; month-number is 1-based, so we subtract 1 for the list index.
(define (time-month-name t)
  (list-ref
    '("January" "February" "March" "April" "May" "June"
      "July" "August" "September" "October" "November" "December")
    (- (time-month t) 1)))

;; Returns n seconds as a duration (seconds are the base unit).
(define (seconds n) n)

;; Returns n minutes expressed as seconds.
(define (minutes n) (* n 60))

;; Returns n hours expressed as seconds.
(define (hours n) (* n 3600))

;; Returns n days expressed as seconds.
(define (days n) (* n 86400))

;; Returns n weeks expressed as seconds.
(define (weeks n) (* n 604800))

;; Returns a new time that is duration seconds after t.
(define (time-add t duration)
  (+ t duration))

;; Returns a new time that is duration seconds before t.
(define (time-subtract t duration)
  (- t duration))

;; Returns the signed difference in seconds between t1 and t2 (t1 − t2).
(define (time-difference t1 t2)
  (- t1 t2))

;; Returns #t if t1 is before t2.
(define (time<? t1 t2) (< t1 t2))

;; Returns #t if t1 is after t2.
(define (time>? t1 t2) (> t1 t2))

;; Returns #t if t1 and t2 represent the same instant.
(define (time=? t1 t2) (= t1 t2))

;; Returns #t if t1 is not after t2.
(define (time<=? t1 t2) (<= t1 t2))

;; Returns #t if t1 is not before t2.
(define (time>=? t1 t2) (>= t1 t2))

;; Format strings use Go's reference time layout:
;;   Mon Jan 2 15:04:05 MST 2006  (i.e. 1=Jan, 2=day, 3=hour-12, 4=min, 5=sec, 6=year)

;; ISO 8601 / RFC 3339: "2006-01-02T15:04:05Z"
(define time-format/iso "2006-01-02T15:04:05Z07:00")

;; Date only: "2006-01-02"
(define time-format/date "2006-01-02")

;; Time of day only: "15:04:05"
(define time-format/time "15:04:05")

;; Date and time without timezone: "2006-01-02 15:04:05"
(define time-format/datetime "2006-01-02 15:04:05")

;; Returns t formatted as an ISO 8601 string.
(define (time->string t)
  (go-extern:time-format t time-format/iso))

;; Returns t formatted according to the given Go layout string.
(define (time->string/fmt t fmt)
  (go-extern:time-format t fmt))

;; Parses an ISO 8601 string and returns a time value.
(define (string->time s)
  (go-extern:time-parse time-format/iso s))

;; Parses s according to the given Go layout string and returns a time value.
(define (string->time/fmt fmt s)
  (go-extern:time-parse fmt s))
