---
name: common-lisp
description: Google's official Common Lisp style guide. Covers naming conventions, formatting, macros, packages, documentation strings, and Lisp best practices.
---

# Google Common Lisp Style Guide

> Official Google Common Lisp coding standards for consistent, maintainable Lisp code.

## Golden Rules

1. **Use lowercase with hyphens** — for all symbols (kebab-case)
2. **Descriptive names** — clarity matters more than brevity
3. **Document functions** — docstrings for all exported symbols
4. **Avoid side effects** — prefer pure functions where possible
5. **Use packages** — organize code into namespaces
6. **Macros sparingly** — only when functions won't work
7. **Follow community conventions** — consistency with the Lisp ecosystem

## Quick Reference

### Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Functions | lowercase-with-hyphens | `get-user-by-id` |
| Variables | lowercase-with-hyphens | `user-count` |
| Constants | +surrounded-by-plus+ | `+max-retries+` |
| Special globals | *surrounded-by-asterisks* | `*database-connection*` |
| Predicates | ends with `-p` | `valid-user-p` |
| Constructors | `make-` prefix | `make-user` |
| Converters | `noun->noun` | `string->integer` |
| Packages | lowercase-with-hyphens | `my-app.users` |

### Packages

```lisp
;; ✓ CORRECT - define a package with explicit exports
(defpackage #:my-app.users
  (:use #:cl)
  (:export
   #:user
   #:make-user
   #:user-name
   #:get-user-by-id))

(in-package #:my-app.users)
```

### Functions and Documentation

```lisp
;; ✓ CORRECT - function with docstring
(defun get-user-by-id (user-id)
  "Retrieves a user record by USER-ID.
Returns the user struct, or NIL if not found."
  (find user-id *users* :key #'user-id))

;; ✓ CORRECT - predicate function (ends in -p)
(defun valid-user-p (user)
  "Returns T if USER is a valid user struct with required fields."
  (and (user-p user)
       (stringp (user-name user))
       (integerp (user-id user))))
```

### Variables and Constants

```lisp
;; ✓ CORRECT - constant (surrounded by +)
(defconstant +max-retry-count+ 3
  "Maximum number of retry attempts for database operations.")

;; ✓ CORRECT - dynamic/special variable (surrounded by *)
(defvar *current-user* nil
  "The currently authenticated user, or NIL if unauthenticated.")

;; ✓ CORRECT - local binding
(let ((user-name "Alice")
      (user-age 30))
  (format t "User: ~a, Age: ~d~%" user-name user-age))
```

### Structures and Classes

```lisp
;; ✓ CORRECT - defstruct with documentation
(defstruct (user (:constructor make-user (id name email)))
  "Represents an application user."
  (id    nil :type (or null integer) :read-only t)
  (name  ""  :type string)
  (email ""  :type string))

;; ✓ CORRECT - CLOS class
(defclass account ()
  ((balance
    :initarg :balance
    :initform 0
    :accessor account-balance
    :documentation "The current account balance.")
   (owner
    :initarg :owner
    :reader account-owner
    :documentation "The account owner."))
  (:documentation "A bank account."))
```

### Macros

```lisp
;; ✓ CORRECT - macro only when needed (can't do with function)
(defmacro with-database-connection ((conn db-spec) &body body)
  "Evaluates BODY with CONN bound to a database connection.
Ensures the connection is closed on exit."
  `(let ((,conn (open-connection ,db-spec)))
     (unwind-protect
         (progn ,@body)
       (close-connection ,conn))))

;; Usage
(with-database-connection (conn *db-spec*)
  (query conn "SELECT * FROM users"))
```

### Error Handling

```lisp
;; ✓ CORRECT - use conditions system
(define-condition user-not-found (error)
  ((user-id :initarg :user-id :reader user-not-found-id))
  (:report (lambda (condition stream)
             (format stream "User ~a not found."
                     (user-not-found-id condition)))))

(defun get-user (id)
  (or (find-user id)
      (error 'user-not-found :user-id id)))

;; ✓ CORRECT - handling conditions
(handler-case (get-user 42)
  (user-not-found (c)
    (format t "Error: ~a~%" c)
    nil))
```

### Formatting

```lisp
;; ✓ CORRECT - indent 2 spaces, align arguments
(defun process-users (users predicate)
  (loop for user in users
        when (funcall predicate user)
          collect user into valid-users
        finally (return valid-users)))

;; ✓ CORRECT - closing parens on same line (no separate lines)
(let ((x 1)
      (y 2))
  (+ x y))
```

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| CamelCase or underscores | Use lowercase-with-hyphens |
| Missing docstrings | Document all exported symbols |
| Using macros for everything | Prefer functions; use macros only when necessary |
| Ignoring packages | Always define and use packages |
| Side effects in pure functions | Keep computation separate from I/O |
| `(car ...)` / `(cdr ...)` on lists | Use `(first ...)` / `(rest ...)` for clarity |
| Closing parens on separate lines | Keep closing parens on same line |

## When to Use This Guide

- Writing new Common Lisp code
- Refactoring existing Lisp projects
- Code reviews
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/common-lisp
```

## Full Guide

See [common-lisp.md](common-lisp.md) for complete details, examples, and edge cases.
