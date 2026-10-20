---
name: html-css
description: Google's official HTML/CSS style guide. Covers formatting, naming, semantics, accessibility, and best practices for web markup and styling.
---

# Google HTML/CSS Style Guide

> Official Google HTML and CSS coding standards for consistent web development.

## Quick Reference

### Golden Rules

1. **Semantic HTML** — use appropriate elements
2. **Lowercase everything** — tags, attributes, values
3. **Close all tags** — even optional ones
4. **Use meaningful class names** — describe purpose, not appearance
5. **Accessibility first** — ARIA, alt text, semantic structure
6. **Mobile-first CSS** — start with mobile, enhance for desktop
7. **BEM or similar** — consistent naming methodology

### HTML Conventions (Preview)

```html
<!-- ✓ CORRECT -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>Page Title</title>
  </head>
  <body>
    <header class="site-header">
      <h1>Welcome</h1>
    </header>
  </body>
</html>
```

### CSS Conventions (Preview)

```css
/* ✓ CORRECT */
.user-profile {
  display: flex;
  padding: 1rem;
}

.user-profile__avatar {
  width: 48px;
  height: 48px;
}
```

## When to Use This Guide

- Writing new HTML/CSS
- Refactoring existing markup/styles
- Code reviews
- Setting up linting rules
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/html-css
```

## Full Guide

See [html-css.md](html-css.md) for complete details, examples, and edge cases.
