# Divergence Categories (R27)

The 4-class taxonomy the drift workflow's Categorize step (drift-report.md,
Task 28) classifies findings into. Every finding is classified into
**exactly one** category — the most-specific match (R27's acceptance); no
dual classification.

## Missing Features

Logic or components described in `briefing.md`/`blueprint.md` but completely
absent in source code.

**Canonical example (R27):** a requirement in the briefing with no
corresponding code.

**More pairs:**

- Design: a webhook delivery retry queue with exponential backoff. Code: no
  queue anywhere; failed deliveries are logged and dropped.
- Design: rate limiting on the public API. Code: no limiter in the request
  path.

**Detection:** for each named requirement, component, file, symbol, or
endpoint in the design docs, grep/glob the codebase for it. An unresolvable
reference (the doc names something the codebase does not have — see
drift-report.md, Sequential processing + source grounding) is a Missing
Features candidate.

## Incomplete Models

Structs, schemas, dataclasses, or classes missing fields specified in the
design.

**Canonical example (R27):** the blueprint specifies a struct with 5 fields,
the code has 3.

**More pairs:**

- Design: an `Order` schema with `id`, `amount`, `currency`, `tax`,
  `status`. Code: `Order` carries only `id` and `amount`.
- Design: an `audit_log` table in the migration plan. Code: no model and no
  migration for it.

**Detection:** extract the field list from the design (schema block, struct
spec, migration diff), extract the declaration from the code (grep the type
name), and diff the two lists field by field.

## Logic Shortcuts

Hardcoded values, mock fallbacks, or bypasses where dynamic logic was
designed.

**Canonical example (R27):** the blueprint specifies a dynamic quota lookup,
the code uses a hardcoded constant.

**More pairs:**

- Design: role-based access checked via the roles service. Code: a literal
  `role == "admin"` comparison.
- Design: behavior gated by a feature flag read from config. Code: the flag
  read replaced with a `true` literal, or a `TODO` around it.

**Detection:** in the code path the design describes, grep for magic
literals, `TODO`/`FIXME`, `mock`, `stub`, `hardcod`, and constant
assignments where the design names a lookup, service, or config read.

## Architectural Divergence

Code implemented using different design patterns, frameworks, or dependencies
than planned.

**Canonical example (R27):** the blueprint specifies a gRPC service, the
implementation uses custom HTTP handlers.

**More pairs:**

- Design: background jobs on a queue (worker + job definitions). Code:
  inline synchronous processing with `setTimeout`.
- Design: PostgreSQL-backed persistence. Code: an in-memory map as the store.

**Detection:** compare the dependency manifests (`package.json`, `go.mod`,
`requirements.txt`) against the frameworks and protocols the design names;
grep for the planned pattern's signatures (service stubs, queue client, ORM)
and for the substitute's.

## Choosing the most-specific match

When two categories plausibly apply, the more specific one wins: a missing
field on a struct that otherwise exists is Incomplete Models, not Missing
Features; a hardcoded value inside a correctly structured service is Logic
Shortcuts, not Architectural Divergence. The category names above are the
report's vocabulary — a drift report's comparability depends on them staying
verbatim.
