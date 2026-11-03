# Google Common Lisp Style Guide

> Source: https://google.github.io/styleguide/lispguide.xml

## Golden Rules

1. **Use lowercase with hyphens** — for symbols
2. **2-space indentation** — for readability
3. **Descriptive names** — clarity over brevity
4. **Document all exported symbols** — with docstrings
5. **Use packages** — for namespace management
6. **Prefer functional style** — minimize side effects

---

## 1. Naming

| Element | Convention | Example |
|---|---|---|
| Functions | lowercase-with-hyphens | `calculate-total` |
| Variables | lowercase-with-hyphens | `user-count` |
| Constants | +constant-name+ | `+max-retries+` |
| Global vars | *global-var* | `*database-connection*` |
| Predicates | ends-with-p | `active-p`, `empty-p` |
| Type predicates | ends-with-p | `stringp`, `numberp` |

```lisp
;; CORRECT - naming conventions
(defun calculate-average (numbers)
  "Calculate the average of a list of numbers."
  (/ (reduce #'+ numbers) (length numbers)))

(defparameter *database-url* "localhost:5432")
(defconstant +max-connections+ 100)

(defun active-p (user)
  "Return T if user is active."
  (eq (user-status user) :active))
```

---

## 2. Functions

```lisp
;; CORRECT - function definition with docstring
(defun process-user (user)
  "Process a user record and return the result.
   USER must be a valid user object."
  (when (active-p user)
    (update-last-seen user)
    (send-notification user)))

;; CORRECT - function with multiple values
(defun divide-with-remainder (dividend divisor)
  "Return quotient and remainder."
  (values (floor dividend divisor)
          (mod dividend divisor)))

;; CORRECT - lambda functions
(mapcar (lambda (x) (* x 2)) '(1 2 3 4 5))
```

---

## 3. Variables

```lisp
;; CORRECT - let for local bindings
(let ((x 10)
      (y 20))
  (+ x y))

;; CORRECT - let* for sequential bindings
(let* ((x 10)
       (y (* x 2)))
  y)

;; CORRECT - defparameter for dynamic variables
(defparameter *default-timeout* 30
  "Default timeout in seconds.")

;; CORRECT - defvar for variables that shouldn't be reset
(defvar *connection-pool* nil
  "Global connection pool.")
```

---

## 4. Control Flow

```lisp
;; CORRECT - if for simple conditionals
(if (> x 10)
    (print "Greater")
    (print "Not greater"))

;; CORRECT - when for single branch
(when (active-p user)
  (process-user user)
  (log-activity user))

;; CORRECT - unless for negated condition
(unless (empty-p queue)
  (process-next-item queue))

;; CORRECT - cond for multiple conditions
(cond
  ((< x 0) "negative")
  ((= x 0) "zero")
  ((> x 0) "positive")
  (t "unknown"))

;; CORRECT - case for dispatch
(case status
  (:active "Active")
  (:inactive "Inactive")
  (:pending "Pending")
  (otherwise "Unknown"))
```

---

## 5. Loops

```lisp
;; CORRECT - dolist for iterating lists
(dolist (item items)
  (process-item item))

;; CORRECT - dotimes for counting
(dotimes (i 10)
  (print i))

;; CORRECT - loop macro
(loop for i from 1 to 10
      collect (* i i))

(loop for item in items
      when (active-p item)
      collect item)
```

---

## 6. Data Structures

```lisp
;; CORRECT - lists
(defvar *users* '("Alice" "Bob" "Charlie"))

;; CORRECT - property lists
(defvar *config* '(:host "localhost"
                   :port 5432
                   :database "mydb"))

;; CORRECT - hash tables
(defvar *user-cache* (make-hash-table :test 'equal))
(setf (gethash "user-123" *user-cache*) user-object)

;; CORRECT - structures
(defstruct user
  id
  name
  email
  (active t))

(defvar *user* (make-user :id 1 :name "Alice"))
```

---

## 7. Macros

```lisp
;; CORRECT - simple macro
(defmacro with-timing (&body body)
  "Execute BODY and print execution time."
  `(let ((start (get-internal-real-time)))
     (prog1
         (progn ,@body)
       (format t "Time: ~A~%"
               (- (get-internal-real-time) start)))))

;; Usage
(with-timing
  (expensive-operation))
```

---

## 8. Error Handling

```lisp
;; CORRECT - condition handling
(handler-case
    (risky-operation)
  (file-error (e)
    (format t "File error: ~A~%" e))
  (error (e)
    (format t "General error: ~A~%" e)))

;; CORRECT - unwind-protect for cleanup
(unwind-protect
     (progn
       (open-resource)
       (use-resource))
  (close-resource))
```

---

## 9. Packages

```lisp
;; CORRECT - package definition
(defpackage :myapp.users
  (:use :cl)
  (:export :user
           :make-user
           :user-name
           :user-email
           :process-user))

(in-package :myapp.users)

;; CORRECT - using symbols from other packages
(myapp.database:connect *database-url*)
```

---

## 10. Documentation

```lisp
;; CORRECT - comprehensive docstrings
(defun fetch-user (user-id)
  "Fetch user by USER-ID from the database.
   
   USER-ID must be a positive integer.
   Returns a USER object or NIL if not found.
   
   Signals DATABASE-ERROR if connection fails."
  (when (not (plusp user-id))
    (error "USER-ID must be positive"))
  (database-query "SELECT * FROM users WHERE id = ?" user-id))
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| CamelCase names | Use lowercase-with-hyphens |
| Missing docstrings | Document all exported symbols |
| Not using packages | Define packages for namespacing |
| Ignoring errors | Use proper error handling |
| Side effects everywhere | Prefer functional style |
| Inconsistent indentation | Use 2-space indentation |

