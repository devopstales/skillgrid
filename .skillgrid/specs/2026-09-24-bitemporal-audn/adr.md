# ADR Review Manifest — 2026-09-24-bitemporal-audn

## In-force ADRs reviewed

| # | Title | In force | Relevance |
|---|-------|----------|-----------|
| 0001 | PRD scope is the whole Hub Product | yes | Context: engine-first framing |
| 0004 | PRD component map is a 4-way decomposition | yes | Context: Mnemonic Engine is one component |
| 0005 | Trust boundary: MCP-spawned process | yes | Context: `mem_save` is an MCP tool |
| 0006 | Vector search: in-memory brute-force cosine | yes | No conflict: bi-temporal is orthogonal to vector search |
| 0009 | Vector search: in-SQL sqlite-vec latency corrected | yes | No conflict: bi-temporal is orthogonal to vector storage |

## New ADRs created

| # | Title | File |
|---|-------|------|
| 0011 | Observations are bi-temporal; the save path classifies each write as Add/Update/Delete/Noop | `04-adr-0011-observations-are-bitemporal.md` |

## Superseded ADRs

None. ADR-0011 is additive — it extends the observation store with temporal
columns and a save-path classifier. It does not supersede any existing ADR.

## Constraints checked

- **#2 (No new dependencies without an ADR)**: Satisfied. The bi-temporal
  columns are plain SQLite (`TEXT`, `INTEGER`). No new Go package. The ADR is
  the vehicle documenting the schema change.
- **#7/#9 (Session-inject two-layer / project-scoped)**: Not implicated.
  Bi-temporal rows with `invalid_at IS NULL` are visible to session-inject
  exactly as today. Superseded rows (`invalid_at` set) are excluded from search
  — and session-inject reads from search — so they're naturally excluded from
  injection. No visibility gate change needed.
