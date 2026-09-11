# Delta Spec Format (skillgrid) — CANONICAL

> Canonical source. `sdd-spec` writes it, `sdd-archive` merges it (`../_shared/references/delta-spec-format.md`).
> Per-skill copies are pointers to this file — do not fork content.

## Capability → file mapping (mechanical, no inference)

From the proposal's **Capabilities** section:

```
FOR EACH entry under "New Capabilities":
  → write docs/skillgrid/changes/{change-name}/specs/<capability>/spec.md
    AS A FULL SPEC (## Purpose + ## Requirements). There is no existing
    behavior to be a delta against.

FOR EACH entry under "Modified Capabilities":
  → read docs/skillgrid/specs/<capability>/spec.md  (REQUIRED first)
  → write docs/skillgrid/changes/{change-name}/specs/<capability>/spec.md
    AS A DELTA SPEC (## ADDED / ## MODIFIED / ## REMOVED / ## RENAMED).
```

Write **New** specs before **Modified** ones. A "New" capability whose main spec already exists (or a "Modified" one with none) is a proposal bug — flag it, don't guess.

## Full spec shape (New capability)

```markdown
## Purpose
{what this capability is for}

## Requirements

### Requirement: {Name}
{Body with RFC 2119 keywords — MUST/SHALL/SHOULD/MAY, always uppercase.}

**Scenario: {happy path}**
- Given {precondition}
- When {action}
- Then {observable outcome}

**Scenario: {edge or failure}**
- Given {precondition}
- When {action}
- Then {observable outcome}
```

Every requirement: ≥1 scenario, covering a happy path **and** an edge/failure. Scenarios are 3–5 lines, testable directly. Spec is WHAT, not HOW — no file paths, line numbers, or function names in requirement text.

## Delta spec shape (Modified capability)

```markdown
## ADDED Requirements
### Requirement: {New Name}
{body + scenarios — new behavior alongside unchanged existing behavior}

## MODIFIED Requirements
### Requirement: {Name}
{COPY of the ENTIRE existing requirement block — name, body, EVERY scenario — then edited}

## REMOVED Requirements
### Requirement: {Name}
(Reason: {why}) (Migration: {what consumers/data/docs/tests must do, or None})

## RENAMED Requirements
### Requirement: {Old} → {New}
(Migration: {references, tests, docs that still point at the old name})
```

Rules:

- **MODIFIED is REPLACE semantics, not PATCH.** `sdd-archive` replaces the main-spec requirement with the MODIFIED block byte-for-byte. Any scenario not copied is **gone** after archive. Adding behavior without changing existing behavior → `## ADDED`, not `## MODIFIED`.
- **REMOVED** requires `(Reason: …)`; include `(Migration: …)` (or `Migration: None`) when consumers, persisted data, docs, or tests are affected.
- **RENAMED** requires the `{old} → {new}` heading plus a `(Migration: …)`.
- A design threat-matrix row marked Applicable MUST have a covering scenario in some spec (add it under its domain or flag the gap — never silently drop it).

## Archive merge semantics (for sdd-archive)

```
FOR EACH SECTION in delta spec:
├── ADDED     → Append the block to the main spec
├── MODIFIED  → REPLACE the same-named requirement block-for-block (name + body + all scenarios)
├── REMOVED   → Delete the same-named requirement (only with Reason; Migration where relevant)
└── RENAMED   → Rename per the {old} → {new} heading, preserving scenarios unless also modified
```

- Match by requirement name (`### Requirement: {Name}`).
- **PRESERVE all requirements not mentioned in the delta.** A MODIFIED block replaces exactly one requirement; never drop siblings. Re-read the main spec after each merge.
- Destructive merges (removing multiple requirements or large sections): WARN and name exactly what is removed before proceeding.

## RFC 2119 quick reference

- **MUST / SHALL** — required. **SHOULD** — recommended, deviate only with reason. **MAY** — optional. Always uppercase; lowercase `must`/`should` is not a keyword.
