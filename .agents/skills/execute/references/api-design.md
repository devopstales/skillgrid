# API Design

How execute shapes the REST surface a change adds or touches: name resources by convention, return HTTP status codes semantically, wrap responses in a consistent envelope, and implement pagination, filtering, auth, rate limiting, and versioning the way a production API should. Use it when a task creates a new endpoint, reshapes an existing contract, or is an "API design / review the API" request.

**Announce when designing or reviewing an API:** "I'm designing the API surface as part of skillgrid:execute."

## When to Use

- **New endpoint** — the task adds a resource or route, so its URL, method, status codes, and response shape follow the conventions from the start instead of being retrofitted.
- **Reshape** — the task changes an existing contract (renames a resource, adds a field, changes a type, splits a route), so the change is breaking-vs-non-breaking is worked out before the code.
- **Review** — the task is an "API design" or "review the API" request on the surface the change touches, rather than fixing one specific bug.

**When NOT to use:** a single bug fix inside an endpoint with no contract change (just fix the bug); an internal script or CLI with no REST surface; a one-off job/queue consumer that is not a request/response API.

## Part 1 — Resource naming

### URL structure

```
# Resources are nouns, plural, lowercase, kebab-case
GET    /api/v1/users
GET    /api/v1/users/:id
POST   /api/v1/users
PUT    /api/v1/users/:id
PATCH  /api/v1/users/:id
DELETE /api/v1/users/:id

# Sub-resources for relationships
GET    /api/v1/users/:id/orders
POST   /api/v1/users/:id/orders

# Actions that don't map to CRUD (use verbs sparingly)
POST   /api/v1/orders/:id/cancel
POST   /api/v1/auth/login
```

### Naming rules

```
# GOOD
/api/v1/team-members          # kebab-case for multi-word resources
/api/v1/orders?status=active  # query params for filtering
/api/v1/users/123/orders      # nested resources for ownership

# BAD
/api/v1/getUsers              # verb in URL
/api/v1/user                  # singular (use plural)
/api/v1/team_members          # snake_case in URLs
/api/v1/users/123/getOrders   # verb in nested resource
```

## Part 2 — HTTP methods and status codes

### Method semantics

| Method | Idempotent | Safe | Use For |
|--------|-----------|------|---------|
| GET | Yes | Yes | Retrieve resources |
| POST | No | No | Create resources, trigger actions |
| PUT | Yes | No | Full replacement of a resource |
| PATCH | No* | No | Partial update of a resource |
| DELETE | Yes | No | Remove a resource |

*PATCH can be made idempotent with proper implementation.

### Status code reference

```
# Success
200 OK                    — GET, PUT, PATCH (with response body)
201 Created               — POST (include Location header)
204 No Content            — DELETE, PUT (no response body)

# Client Errors
400 Bad Request           — Validation failure, malformed JSON
401 Unauthorized          — Missing or invalid authentication
403 Forbidden             — Authenticated but not authorized
404 Not Found             — Resource doesn't exist
409 Conflict              — Duplicate entry, state conflict
422 Unprocessable Entity  — Semantically invalid (valid JSON, bad data)
429 Too Many Requests     — Rate limit exceeded

# Server Errors
500 Internal Server Error — Unexpected failure (never expose details)
502 Bad Gateway           — Upstream service failed
503 Service Unavailable   — Temporary overload, include Retry-After
```

### The mistakes that show up in code

```
# BAD: 200 for everything
{ "status": 200, "success": false, "error": "Not found" }

# GOOD: use HTTP status codes semantically
HTTP/1.1 404 Not Found
{ "error": { "code": "not_found", "message": "User not found" } }

# BAD: 500 for validation errors
# GOOD: 400 or 422 with field-level details

# BAD: 200 for created resources
# GOOD: 201 with Location header
HTTP/1.1 201 Created
Location: /api/v1/users/abc-123
```

## Part 3 — Response envelope

One shape for the whole API, chosen once and kept consistent. Distinguish success from error by **both** the HTTP status code and the body shape.

### Success (single resource)

```json
{
  "data": {
    "id": "abc-123",
    "email": "alice@example.com",
    "name": "Alice",
    "created_at": "2025-01-15T10:30:00Z"
  }
}
```

### Collection (with pagination)

```json
{
  "data": [
    { "id": "abc-123", "name": "Alice" },
    { "id": "def-456", "name": "Bob" }
  ],
  "meta": {
    "total": 142,
    "page": 1,
    "per_page": 20,
    "total_pages": 8
  },
  "links": {
    "self": "/api/v1/users?page=1&per_page=20",
    "next": "/api/v1/users?page=2&per_page=20",
    "last": "/api/v1/users?page=8&per_page=20"
  }
}
```

### Error

```json
{
  "error": {
    "code": "validation_error",
    "message": "Request validation failed",
    "details": [
      { "field": "email", "message": "Must be a valid email address", "code": "invalid_format" },
      { "field": "age", "message": "Must be between 0 and 150", "code": "out_of_range" }
    ]
  }
}
```

### Envelope variants (pick one, apply to every endpoint)

```typescript
// Option A: envelope with data wrapper (recommended for public APIs)
interface ApiResponse<T> {
  data: T;
  meta?: PaginationMeta;
  links?: PaginationLinks;
}

interface ApiError {
  error: {
    code: string;
    message: string;
    details?: FieldError[];
  };
}

// Option B: flat response (simpler, common for internal APIs)
// Success: return the resource directly. Error: return the error object.
// Distinguish by HTTP status code.
```

`error.code` is a **stable machine-readable identifier** (snake_case, e.g. `validation_error`, `email_taken`, `rate_limit_exceeded`); `error.message` is human-readable and may be localized. Clients branch on `code`, never on `message`.

## Part 4 — Pagination

### Offset-based (simple)

```
GET /api/v1/users?page=2&per_page=20

# Implementation
SELECT * FROM users
ORDER BY created_at DESC
LIMIT 20 OFFSET 20;
```

**Pros:** easy to implement, supports "jump to page N." **Cons:** slow on large offsets (`OFFSET 100000`), inconsistent under concurrent inserts.

### Cursor-based (scalable)

```
GET /api/v1/users?cursor=eyJpZCI6MTIzfQ&limit=20

# Implementation
SELECT * FROM users
WHERE id > :cursor_id
ORDER BY id ASC
LIMIT 21;  -- fetch one extra to determine has_next
```

```json
{
  "data": [...],
  "meta": {
    "has_next": true,
    "next_cursor": "eyJpZCI6MTQzfQ"
  }
}
```

**Pros:** consistent performance regardless of position, stable under concurrent inserts. **Cons:** cannot jump to an arbitrary page, cursor is opaque.

### When to use which

| Use case | Pagination type |
|----------|----------------|
| Admin dashboards, small datasets (<10K) | Offset |
| Infinite scroll, feeds, large datasets | Cursor |
| Public APIs | Cursor (default) with offset (optional) |
| Search results | Offset (users expect page numbers) |

## Part 5 — Filtering, sorting, search, sparse fieldsets

### Filtering

```
# Simple equality
GET /api/v1/orders?status=active&customer_id=abc-123

# Comparison operators (bracket notation)
GET /api/v1/products?price[gte]=10&price[lte]=100
GET /api/v1/orders?created_at[after]=2025-01-01

# Multiple values (comma-separated)
GET /api/v1/products?category=electronics,clothing

# Nested fields (dot notation)
GET /api/v1/orders?customer.country=US
```

### Sorting

```
# Single field (prefix - for descending)
GET /api/v1/products?sort=-created_at

# Multiple fields (comma-separated)
GET /api/v1/products?sort=-featured,price,-created_at
```

### Full-text search

```
# Search query parameter
GET /api/v1/products?q=wireless+headphones

# Field-specific search
GET /api/v1/users?email=alice
```

### Sparse fieldsets

```
# Return only specified fields (reduces payload)
GET /api/v1/users?fields=id,name,email
GET /api/v1/orders?fields=id,total,status&include=customer.name
```

## Part 6 — Authentication and authorization

```
# Bearer token (user-facing)
GET /api/v1/users
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

# API key (server-to-server)
GET /api/v1/data
X-API-Key: sk_live_abc123
```

```typescript
// Resource-level: check ownership
app.get("/api/v1/orders/:id", async (req, res) => {
  const order = await Order.findById(req.params.id);
  if (!order) return res.status(404).json({ error: { code: "not_found" } });
  if (order.userId !== req.user.id) return res.status(403).json({ error: { code: "forbidden" } });
  return res.json({ data: order });
});

// Role-based: check permissions
app.delete("/api/v1/users/:id", requireRole("admin"), async (req, res) => {
  await User.delete(req.params.id);
  return res.status(204).send();
});
```

**401 vs 403:** `401` = the request didn't say who it is (missing/invalid credential). `403` = it said who it is, but that identity may not do this. Returning 401 for a permission problem (or 403 for a missing token) is the classic tell of an unreviewed API.

## Part 7 — Rate limiting

### Headers

```
HTTP/1.1 200 OK
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1640000000

# When exceeded
HTTP/1.1 429 Too Many Requests
Retry-After: 60
{
  "error": {
    "code": "rate_limit_exceeded",
    "message": "Rate limit exceeded. Try again in 60 seconds."
  }
}
```

### Rate limit tiers

| Tier | Limit | Window | Use case |
|------|-------|--------|----------|
| Anonymous | 30/min | Per IP | Public endpoints |
| Authenticated | 100/min | Per user | Standard API access |
| Premium | 1000/min | Per API key | Paid API plans |
| Internal | 10000/min | Per service | Service-to-service |

## Part 8 — Versioning

### URL path versioning (recommended)

```
/api/v1/users
/api/v2/users
```

**Pros:** explicit, easy to route, cacheable. **Cons:** URL changes between versions.

### Header versioning

```
GET /api/users
Accept: application/vnd.myapp.v2+json
```

**Pros:** clean URLs. **Cons:** harder to test, easy to forget.

### Versioning strategy

```
1. Start with /api/v1/ — don't version until you need to.
2. Maintain at most 2 active versions (current + previous).
3. Deprecation timeline:
   - Announce deprecation (6 months notice for public APIs).
   - Add Sunset header: Sunset: Sat, 01 Jan 2026 00:00:00 GMT.
   - Return 410 Gone after the sunset date.
4. Non-breaking changes don't need a new version:
   - Adding new fields to responses.
   - Adding new optional query parameters.
   - Adding new endpoints.
5. Breaking changes require a new version:
   - Removing or renaming fields.
   - Changing field types.
   - Changing URL structure.
   - Changing the authentication method.
```

**The decision that matters in a change:** is the edit non-breaking (rule 4) or breaking (rule 5)? A breaking change without a new version is the most common silent API regression.

## Implementation patterns

### TypeScript (Next.js API route)

```typescript
import { z } from "zod";
import { NextRequest, NextResponse } from "next/server";

const createUserSchema = z.object({
  email: z.string().email(),
  name: z.string().min(1).max(100),
});

export async function POST(req: NextRequest) {
  const body = await req.json();
  const parsed = createUserSchema.safeParse(body);

  if (!parsed.success) {
    return NextResponse.json({
      error: {
        code: "validation_error",
        message: "Request validation failed",
        details: parsed.error.issues.map(i => ({
          field: i.path.join("."),
          message: i.message,
          code: i.code,
        })),
      },
    }, { status: 422 });
  }

  const user = await createUser(parsed.data);

  return NextResponse.json(
    { data: user },
    {
      status: 201,
      headers: { Location: `/api/v1/users/${user.id}` },
    },
  );
}
```

### Go (net/http)

```go
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeError(w, http.StatusBadRequest, "invalid_json", "Invalid request body")
        return
    }

    if err := req.Validate(); err != nil {
        writeError(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
        return
    }

    user, err := h.service.Create(r.Context(), req)
    if err != nil {
        switch {
        case errors.Is(err, domain.ErrEmailTaken):
            writeError(w, http.StatusConflict, "email_taken", "Email already registered")
        default:
            writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
        }
        return
    }

    w.Header().Set("Location", fmt.Sprintf("/api/v1/users/%s", user.ID))
    writeJSON(w, http.StatusCreated, map[string]any{"data": user})
}
```

## Red Flags (api design)

- 200 for everything, with a `success: false` boolean in the body instead of a real status code.
- A verb in the URL (`getUsers`, `createOrder`) or a singular resource (`/user`).
- 500 for a validation error (should be 400/422 with field-level details).
- 201 returned without a `Location` header on the created resource.
- `error.message` used as the machine code (it's localized and changes); no stable `error.code`.
- List endpoint with no pagination, or offset pagination on a large dataset / feed.
- 401 returned for a permission problem (or 403 for a missing token).
- A breaking change (renamed/removed field, changed type, changed URL) shipped without a new version.
- Response leaks internal details (stack traces, SQL errors, model names).
- A new endpoint whose naming or status codes diverge from the existing surface.

## Verification (api design)

- [ ] Resource URL follows the naming conventions (plural, kebab-case, no verbs), consistent with the existing endpoints.
- [ ] Correct HTTP method used (GET for reads, POST for creates, PUT for full replace, PATCH for partial, DELETE for remove).
- [ ] Appropriate status codes returned (not 200 for everything); 201 carries a `Location` header; validation errors are 400/422, not 500.
- [ ] Input validated with a schema (Zod, Pydantic, Bean Validation) before it reaches the handler.
- [ ] Error responses follow the standard envelope with a stable `error.code` and human `error.message`; internal details (stack traces, SQL) are not leaked.
- [ ] Pagination implemented for list endpoints (cursor for large/feeds, offset for small/dashboards/search).
- [ ] Authentication required (or explicitly marked public); authorization checked so a user reaches only their own resources (401 vs 403 used correctly).
- [ ] Rate limiting configured with the right tier; `Retry-After` on 429.
- [ ] Breaking-vs-non-breaking decided; a breaking change opened a new version (or the change is confirmed non-breaking per the versioning strategy).
- [ ] OpenAPI/Swagger spec updated to match the endpoint actually shipped.
