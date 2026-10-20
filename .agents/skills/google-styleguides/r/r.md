# Google R Style Guide

> Source: https://google.github.io/styleguide/Rguide.html

## Golden Rules

1. **BigCamelCase for functions** — to distinguish from objects
2. **snake_case for variables** — lowercase with underscores
3. **Explicit `return()`** — don't rely on implicit returns
4. **Qualify namespaces** — use `package::function()`
5. **No `attach()`** — avoid namespace pollution
6. **No right-hand assignment** — use `<-` on left only

---

## 1. Naming

| Element | Convention | Example |
|---|---|---|
| Functions | BigCamelCase | `CalculateAverage()` |
| Private functions | .BigCamelCase | `.HelperFunction()` |
| Variables | snake_case | `user_count` |
| Constants | snake_case | `max_iterations` |

```r
# CORRECT - function naming
CalculateAverage <- function(x) {
  return(mean(x, na.rm = TRUE))
}

# CORRECT - private function
.ValidateInput <- function(x) {
  return(is.numeric(x))
}

# CORRECT - variable naming
user_count <- 10
total_sales <- sum(sales_data$amount)
```

---

## 2. Assignment

```r
# CORRECT - use <- for assignment
x <- 5
result <- CalculateTotal(data)

# AVOID - right-hand assignment
data %>%
  filter(active == TRUE) -> filtered_data  # AVOID

# AVOID - = for assignment (use for function arguments only)
x = 5  # AVOID
```

---

## 3. Functions

```r
# CORRECT - explicit return
CalculateTotal <- function(values) {
  total <- sum(values, na.rm = TRUE)
  return(total)
}

# AVOID - implicit return
CalculateTotal <- function(values) {
  sum(values, na.rm = TRUE)  # AVOID
}

# CORRECT - function with multiple returns
CheckValue <- function(x) {
  if (x < 0) {
    return("negative")
  }
  if (x == 0) {
    return("zero")
  }
  return("positive")
}
```

---

## 4. Namespace Qualification

```r
# CORRECT - explicit namespace
result <- dplyr::filter(data, active == TRUE)
plot <- ggplot2::ggplot(data, ggplot2::aes(x, y))

# AVOID - importing everything
library(dplyr)  # AVOID in packages
filter(data, active == TRUE)

# CORRECT - in package DESCRIPTION, use Imports not Depends
```

---

## 5. Pipes

```r
# CORRECT - pipe usage
result <- data %>%
  dplyr::filter(active == TRUE) %>%
  dplyr::group_by(category) %>%
  dplyr::summarize(total = sum(amount))

# CORRECT - line breaks in pipes
result <- data %>%
  dplyr::filter(
    active == TRUE,
    amount > 100
  ) %>%
  dplyr::arrange(desc(amount))
```

---

## 6. Data Manipulation

```r
# CORRECT - dplyr for data manipulation
filtered_data <- data %>%
  dplyr::filter(year == 2024) %>%
  dplyr::select(id, name, amount) %>%
  dplyr::mutate(
    amount_usd = amount * exchange_rate,
    category = dplyr::case_when(
      amount < 100 ~ "small",
      amount < 1000 ~ "medium",
      TRUE ~ "large"
    )
  )
```

---

## 7. Control Flow

```r
# CORRECT - if/else
if (condition) {
  DoSomething()
} else if (other_condition) {
  DoSomethingElse()
} else {
  DoDefault()
}

# CORRECT - for loop
for (i in seq_along(items)) {
  ProcessItem(items[[i]])
}

# CORRECT - while loop
while (condition) {
  DoSomething()
  condition <- CheckCondition()
}
```

---

## 8. Documentation

```r
#' Calculate the average of a numeric vector
#'
#' This function calculates the mean of a numeric vector,
#' removing NA values by default.
#'
#' @param x A numeric vector
#' @param na_rm Logical, whether to remove NA values
#' @return The mean of x
#' @examples
#' CalculateAverage(c(1, 2, 3, 4, 5))
#' CalculateAverage(c(1, 2, NA, 4), na_rm = TRUE)
#' @export
CalculateAverage <- function(x, na_rm = TRUE) {
  if (!is.numeric(x)) {
    stop("x must be numeric")
  }
  return(mean(x, na.rm = na_rm))
}
```

---

## 9. Package Development

```r
# CORRECT - package structure
# R/
#   calculate.R
#   validate.R
# tests/
#   testthat/
#     test-calculate.R
# DESCRIPTION
# NAMESPACE

# CORRECT - in DESCRIPTION file
Imports:
    dplyr,
    ggplot2,
    purrr

# CORRECT - in NAMESPACE (via roxygen2)
#' @importFrom dplyr filter mutate
#' @importFrom ggplot2 ggplot aes
```

---

## 10. Testing

```r
# CORRECT - testthat tests
test_that("CalculateAverage returns correct mean", {
  result <- CalculateAverage(c(1, 2, 3, 4, 5))
  expect_equal(result, 3)
})

test_that("CalculateAverage handles NA values", {
  result <- CalculateAverage(c(1, 2, NA, 4), na_rm = TRUE)
  expect_equal(result, 7/3)
})

test_that("CalculateAverage errors on non-numeric input", {
  expect_error(
    CalculateAverage(c("a", "b", "c")),
    "x must be numeric"
  )
})
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Right-hand assignment `->` | Use left-hand `<-` |
| Implicit returns | Use explicit `return()` |
| Using `attach()` | Access data frame columns directly |
| Missing namespace | Use `package::function()` |
| `=` for assignment | Use `<-` (save `=` for arguments) |
| Importing all functions | Import specific functions only |

