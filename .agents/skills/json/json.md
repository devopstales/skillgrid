# Google JSON Style Guide

> Source: https://google.github.io/styleguide/jsoncstyleguide.xml

## Golden Rules

1. **Use camelCase for property names** — consistent with JavaScript
2. **Use double quotes** — for strings
3. **No trailing commas** — JSON doesn't allow them
4. **Use arrays for ordered data** — objects for unordered
5. **Keep it simple** — avoid deep nesting
6. **Use consistent date formats** — ISO 8601 recommended

---

## 1. Property Names

```json
// CORRECT - camelCase
{
  "firstName": "John",
  "lastName": "Doe",
  "emailAddress": "john@example.com",
  "phoneNumber": "+1-555-0100"
}

// AVOID - snake_case or PascalCase
{
  "first_name": "John",
  "FirstName": "John"
}
```

---

## 2. Data Types

```json
// CORRECT - use appropriate types
{
  "name": "John Doe",
  "age": 30,
  "isActive": true,
  "balance": 1234.56,
  "tags": ["developer", "designer"],
  "address": {
    "street": "123 Main St",
    "city": "New York"
  },
  "metadata": null
}
```

---

## 3. Arrays

```json
// CORRECT - arrays for ordered lists
{
  "users": [
    {
      "id": 1,
      "name": "Alice"
    },
    {
      "id": 2,
      "name": "Bob"
    }
  ]
}

// CORRECT - empty arrays
{
  "items": []
}
```

---

## 4. Objects

```json
// CORRECT - objects for key-value pairs
{
  "user": {
    "id": 123,
    "name": "John Doe",
    "email": "john@example.com"
  },
  "settings": {
    "theme": "dark",
    "language": "en",
    "notifications": true
  }
}

// CORRECT - empty objects
{
  "metadata": {}
}
```

---

## 5. Dates and Times

```json
// CORRECT - ISO 8601 format
{
  "createdAt": "2024-01-15T10:30:00Z",
  "updatedAt": "2024-01-15T14:45:30.123Z",
  "birthDate": "1990-05-20"
}

// AVOID - custom date formats
{
  "createdAt": "01/15/2024",
  "updatedAt": "15-Jan-2024"
}
```

---

## 6. Null vs Omission

```json
// CORRECT - use null for explicitly empty values
{
  "name": "John",
  "middleName": null,
  "email": "john@example.com"
}

// ALSO CORRECT - omit optional fields
{
  "name": "John",
  "email": "john@example.com"
}
```

---

## 7. Boolean Values

```json
// CORRECT - use true/false
{
  "isActive": true,
  "isVerified": false,
  "hasAccess": true
}

// AVOID - strings or numbers for booleans
{
  "isActive": "true",
  "isVerified": 0
}
```

---

## 8. Numbers

```json
// CORRECT - numbers without quotes
{
  "count": 42,
  "price": 19.99,
  "percentage": 0.15,
  "scientific": 1.23e-4
}

// AVOID - numbers as strings
{
  "count": "42",
  "price": "19.99"
}
```

---

## 9. Formatting

```json
// CORRECT - pretty-printed for readability
{
  "user": {
    "id": 123,
    "name": "John Doe",
    "roles": [
      "admin",
      "editor"
    ]
  }
}

// CORRECT - minified for production
{"user":{"id":123,"name":"John Doe","roles":["admin","editor"]}}
```

---

## 10. API Responses

```json
// CORRECT - consistent structure
{
  "status": "success",
  "data": {
    "user": {
      "id": 123,
      "name": "John Doe"
    }
  },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z",
    "version": "1.0"
  }
}

// CORRECT - error response
{
  "status": "error",
  "error": {
    "code": "INVALID_INPUT",
    "message": "Email address is required",
    "field": "email"
  }
}
```

---

## 11. Pagination

```json
// CORRECT - pagination metadata
{
  "data": [
    {"id": 1, "name": "Item 1"},
    {"id": 2, "name": "Item 2"}
  ],
  "pagination": {
    "page": 1,
    "pageSize": 20,
    "totalPages": 5,
    "totalItems": 100
  }
}
```

---

## 12. Versioning

```json
// CORRECT - include version in response
{
  "apiVersion": "2.0",
  "data": {
    "user": {
      "id": 123,
      "name": "John Doe"
    }
  }
}
```

---

## Common Mistakes

| Mistake | Correct Approach |
|---|---|
| Trailing commas | Remove all trailing commas |
| Single quotes | Use double quotes only |
| Comments | JSON doesn't support comments |
| snake_case properties | Use camelCase |
| Undefined values | Use `null` or omit the property |
| Numbers as strings | Use actual number types |
| Inconsistent date formats | Use ISO 8601 |

