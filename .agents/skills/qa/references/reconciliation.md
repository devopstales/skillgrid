# Reconciliation Protocol (R32)

What happens after a drift finding is resolved: which artifact changes, what
disposition the drift report records, and how the run lands in the review
ledger. The report's top-level shape is R31's strict format, owned by
`drift-report.md` (Task 28) — not restated here.

## Two paths (R32)

Every drift finding is actionable via one of two paths:

- **Code was intentional** (the design doc is stale) → update
  `briefing.md`/`blueprint.md`/ADR to reflect reality. The doc is the
  artifact that changes.
- **Design was violated** (the code is wrong) → file a Backlog task (via
  `ticketing`) or fix the code directly. The code is the artifact that
  changes.

The user decides which path for each finding: an interactive gate, same
pattern as R10 — the five options per
`.agents/skills/writing-blueprints/references/goldfish-resolve.md` (Task 22,
cited not restated).

## Dispositions

Each finding in the drift report gets exactly one disposition:

| Disposition | Meaning |
|---|---|
| `doc-updated` | the design artifact was rewritten to reflect reality |
| `task-filed` | a Backlog task was filed; the ticket is the artifact |
| `code-fixed` | the code was changed to match the design |
| `accepted-drift` | the drift stands as-is, with a note |

**Ledger mapping (R26):** `doc-updated` / `code-fixed` ≈ `fixed`;
`task-filed` ≈ `fixed` with the ticket as the artifact; `accepted-drift` ≈
`parked` with a ruling. Drift debt therefore surfaces at qa time via the
ledger's debt-surfacing step (Task 12).

## Report details

The artifact's top-level shape is R31's strict format (owned by
`drift-report.md`, Task 28; do not restate it here). This section defines
what the disposition column and follow-up pointers carry:

- **Per-finding disposition**: one of the four values in the Dispositions
  section.
- **Follow-up pointer**: the ticket id or change id the fix lands in, or
  `none` when the finding is closed in place.
- **Summary counts**: by category and by disposition, carried in the report
  header.

## Ledger rounds

Each drift run writes a ledger round per
`.agents/skills/_shared/craft/review-ledger.md`; severities normalized at the
record boundary (same rule as Task 20): findings carry their category and the
reviewer vocabulary in transit, and the ledger entry normalizes severity to
CRITICAL/WARNING/SUGGESTION. That entry is the single normalization point.
