---
name: r
description: Google's official R style guide. Covers naming conventions, formatting, functions, documentation, and R best practices for data analysis.
---

# Google R Style Guide

> Official Google R coding standards for consistent, maintainable data analysis code.

## Quick Reference

### Golden Rules

1. **Use snake_case** — for functions and variables
2. **Meaningful names** — describe what, not how
3. **Limit line length** — 80 characters max
4. **Comment your code** — explain why, not what
5. **Use roxygen2** — for function documentation
6. **Avoid global state** — functions should be pure when possible
7. **Use tidyverse** — consistent, modern R patterns

### Naming Conventions (Preview)

| Element | Convention | Example |
|---|---|---|
| Functions | snake_case | `calculate_mean` |
| Variables | snake_case | `user_count` |
| Constants | UPPER_SNAKE_CASE | `MAX_ITERATIONS` |

### Basic Patterns (Preview)

```r
# ✓ CORRECT
calculate_user_stats <- function(users, min_age = 18) {
  users %>%
    filter(age >= min_age) %>%
    summarize(
      count = n(),
      avg_age = mean(age)
    )
}
```

## When to Use This Guide

- Writing new R code
- Data analysis scripts
- R package development
- Code reviews
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/r
```

## Full Guide

See [r.md](r.md) for complete details, examples, and edge cases.
