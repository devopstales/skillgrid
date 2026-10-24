# Task Brief — T01 (TASK-089): CLM absorption — renumber CLM migrations 050/051/052 → 053/054/055

- Ticket: TASK-089 (type: refactor, priority: high)
- Change: 2026-10-08-context-orchestrator (CTX v1.11)
- Wave 1 · PR1 (foundation) · base = release/2
- Reversibility: ONE-WAY (migration file renames — a wrong rename re-applies the wrong DDL)
- Precondition: CLM spec (2026-10-06-context-harness-clm) migrations 050/051/052 exist on disk
- SATISFIES: clm-renumber

## Scope
Rename the three CLM migration files in `mnemonic/internal/store/migrations/` so CTX's `ctx_*` tables can start at 056 on the shared SQLite store (ADR-0033). No schema change — the CLM tables already exist from the CLM change. This is a pure file rename + any hard-coded path/number refs in the migration runner.

- 050_tool_outputs → 053_tool_outputs
- 051_indexed_files → 054_indexed_files
- 052_context_revisions → 055_context_revisions

## Acceptance Criteria
1. `ls mnemonic/internal/store/migrations/` shows 053/054/055 for tool_outputs/indexed_files/context_revisions; NONE numbered 050/051/052 remains for CLM.
2. `go test ./internal/store/...` green: migrations apply cleanly in order on a fresh store.
3. Existing CLM rows intact after re-apply (row counts unchanged).

## Definition of Done
- Tests pass for touched packages; lint/format pass; edge cases covered; no new warnings; CLM rows verified intact after re-apply.

## Implementation Plan
1. Locate the three CLM migration files in `mnemonic/internal/store/migrations/`.
2. Rename 050→053, 051→054, 052→055 using `git mv` to preserve history.
3. Update any hard-coded path/ref to the old numbers in the migration runner (`internal/store/migrate*`).
4. `go build ./...` and `go test ./internal/store/...` — confirm migrations apply in order on a fresh store.
5. Verify existing CLM rows intact after re-apply (row counts unchanged).

## Gates
- G1: `cd mnemonic && ls internal/store/migrations/ | grep -E '^05[0-2]_'` → EXPECT exit 1 (no 050/051/052 files remain)
- G2: `cd mnemonic && ls internal/store/migrations/ | grep -E '^05[3-5]_'` → EXPECT exit 0 (053/054/055 present)
- G3: `cd mnemonic && go build ./...` → EXPECT exit 0
- G4: `cd mnemonic && go test ./internal/store/...` → EXPECT exit 0
- Fails-when: `go test ./internal/store/...` exit != 0 OR any file matches `05[0-2]_`

## ADR Constraints (do not re-litigate)
- ADR-0033: ctx is a `mnemonic` subcommand; ctx_* tables live on the shared SQLite store (no 2nd DB). This renumber makes room for ctx_* at 056+.
- ADR-0013: repo is source of truth; commit carries a `[skillgrid-context]` block naming this ticket.

## Owns (edit only under these)
- `mnemonic/internal/store/migrations/05{3,4,5}_*` (renamed)
- any `mnemonic/internal/store/migrate*` refs to the old numbers

## Work Unit Evidence (fill in report)
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T01 CLM renumber | `go test ./internal/store/...` | in-process SQLite migrate on fresh store | `git revert <sha>` (3 file renames + runner refs) |
