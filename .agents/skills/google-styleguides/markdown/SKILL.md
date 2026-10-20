---
name: markdown
description: Google's official Markdown style guide. Covers formatting, structure, links, lists, and Markdown best practices for documentation.
---

# Google Markdown Style Guide

> Official Google Markdown formatting standards for consistent documentation.

## Quick Reference

### Golden Rules

1. **One sentence per line** — easier diffs and reviews
2. **ATX-style headers** — use `#` not underlines
3. **Fenced code blocks** — use ``` with language
4. **Reference-style links** — for repeated URLs
5. **Consistent list markers** — `-` for unordered, `1.` for ordered
6. **Blank lines around blocks** — improve readability
7. **80-character line limit** — for prose

### Headers (Preview)

```markdown
# Top Level Header

## Second Level

### Third Level
```

### Code Blocks (Preview)

````markdown
```python
def hello():
    print("Hello, world!")
```
````

### Lists (Preview)

```markdown
- First item
- Second item
  - Nested item
  - Another nested

1. Ordered first
2. Ordered second
```

### Links (Preview)

```markdown
[Google](https://google.com)

[Google][google-link]

[google-link]: https://google.com
```

## When to Use This Guide

- Writing documentation
- README files
- Technical writing
- Code reviews
- Onboarding new team members

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/markdown
```

## Full Guide

See [markdown.md](markdown.md) for complete details, examples, and edge cases.
