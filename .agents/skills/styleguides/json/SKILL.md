---
name: json
description: Google's official JSON style guide. Covers formatting, naming conventions, structure, and JSON best practices for APIs and configuration.
---

# Google JSON Style Guide

> Official Google JSON formatting standards for consistent API responses and configuration.

## Quick Reference

### Golden Rules

1. **Use camelCase** — for property names
2. **Consistent structure** — predictable response format
3. **Include metadata** — version, timestamps when relevant
4. **Use arrays for collections** — even single items
5. **Null vs omission** — be consistent
6. **ISO 8601 for dates** — standard format
7. **Pretty print in development** — minify in production

### Property Naming (Preview)

```json
{
  "userId": 123,
  "firstName": "Alice",
  "createdAt": "2024-01-15T10:30:00Z",
  "isActive": true
}
```

### Response Structure (Preview)

```json
{
  "data": {
    "users": [
      {"id": 1, "name": "Alice"},
      {"id": 2, "name": "Bob"}
    ]
  },
  "meta": {
    "total": 2,
    "page": 1
  }
}
```

## When to Use This Guide

- Designing REST APIs
- Writing configuration files
- Data interchange formats
- Code reviews
- API documentation

## Install

```bash
npx skills add testdino-hq/google-styleguides-skills/json
```

## Full Guide

See [json.md](json.md) for complete details, examples, and edge cases.
