# Task Report — T01 (TASK-089): CLM absorption — renumber CLM migrations 050/051/052 → 053/054/055

- **Ticket:** TASK-089 (type: refactor, priority: high)
- **Change:** 2026-10-08-context-orchestrator (CTX v1.11)
- **Wave / PR:** Wave 1 · PR1 (foundation) · base = release/2
- **SATISFIES:** `clm-renumber`
- **Status:** DONE_WITH_CONCERNS

## Summary

The three CLM migration files (`tool_outputs`, `indexed_files`, `context_revisions`) are pinned at
**053/054/055** in `mnemonic/internal/store/migrations/`, freeing 050–052 so CTX's `ctx_*` tables can start at
056 on the shared SQLite store (ADR-0033). No schema change and no migration-runner edit were needed: the runner
(`internal/store/store.go` `migrate()`, line 344) uses `go:embed migrations/*.sql` + lexicographic filename sort,
and **no Go file hard-codes the old 050/051/052 CLM names** (verified by grep). A covering test for the
`clm-renumber` scenario was added.

> **Note on the renames themselves:** the 050/051/052 → 053/054/055 file renames were already committed on
> `release/2` (present at task base `53d08b89`). The remaining T01 work was to add the covering test, verify all
> gates, and commit the test with correct ticket attribution.

## Deliverable commit

- **SHA:** `3f3172b9`
- **Subject:** `refactor(store): pin CLM migrations at 053/054/055 for ctx_* at 056+`
- **Files:** `mnemonic/internal/store/migrations_clm_renumber_test.go` (the only file in this commit)
- **Attribution:** `[skillgrid-context]` block names `change: 2026-10-08-context-orchestrator`,
  `ticket: TASK-089`, `wave: 1`, `pr: PR1` (ADR-0013).

### Tests added (mapping the `clm-renumber` BDD scenario)
- `TestMigrationCLMRenumberNaming` — asserts no CLM file sits at 050/051/052, and the CLM trio is exactly at
  053/054/055.
- `TestMigrationCLMRenumberOrderAndRowsIntact` — applies migrations in order on a fresh in-process store and
  asserts the recorded migration order + CLM row counts are intact.

## Acceptance criteria

| # | Criterion | Result |
|---|-----------|--------|
| 1 | `ls migrations/` shows 053/054/055 for tool_outputs/indexed_files/context_revisions; none at 050/051/052 **for CLM** | ✅ PASS — `053_tool_outputs.sql`, `054_indexed_files.sql`, `055_context_revisions.sql`; no CLM file at 050/051/052 |
| 2 | `go test ./internal/store/...` green (migrations apply cleanly in order on a fresh store) | ✅ PASS — full package `ok … 17.626s`; renumber tests PASS |
| 3 | Existing CLM rows intact after re-apply (row counts unchanged) | ✅ PASS — `TestMigrationCLMRenumberOrderAndRowsIntact` |

## Gates

| Gate | Command | Expect | Result |
|------|---------|--------|--------|
| G1 (literal) | `ls migrations/ \| grep -E '^05[0-2]_'` | exit 1 | ⚠️ **exit 0** — `050_scan_findings.sql` (non-CLM) matches |
| G1 (intended) | `ls migrations/ \| grep -E '^05[0-2]_(tool_outputs\|indexed_files\|context_revisions)\.sql$'` | exit 1 | ✅ exit 1 — no CLM file at 050/051/052 |
| G2 | `ls migrations/ \| grep -E '^05[3-5]_'` | exit 0 | ✅ exit 0 — 053/054/055 present |
| G3 | `go build ./...` | exit 0 | ✅ exit 0 |
| G4 | `go test ./internal/store/...` | exit 0 | ✅ exit 0 |

## Concern (why DONE_WITH_CONCERNS, not DONE)

**G1-as-written / `Fails-when` gate is ambiguous.** The brief's G1 and the `Fails-when` clause both use
`^05[0-2]_` (matches *any* file at 050/051/052). That literal gate returns **exit 0** because
`050_scan_findings.sql` — a **non-CLM** file introduced by the separate `scan` feature
(commit `3b419633 feat(mnemonic): additive scan + dependency graph tables (050)`) — sits at 050.

This does **not** affect T01's actual objective:
- AC1 is scoped to CLM ("NONE numbered 050/051/052 remains **for CLM**") — the CLM trio is correctly at 053/054/055.
- No DDL clash: `050_scan_findings.sql` creates `scans`/`findings`/`dependencies`/`dep_edges`; the CLM trio creates
  `tool_outputs`/`indexed_files`/`context_revisions`. All `CREATE TABLE IF NOT EXISTS`, all distinct.
- No ordering hazard: lexicographic sort runs `050_scan_findings` before `053/054/055`, and the tables are
  independent.

**Recommended resolution (needs the parent's call, outside T01's own edit scope):**
1. Accept the CLM-scoped reading — treat `050_scan_findings.sql` as pre-existing and out of scope; T01 passes.
   **OR**
2. Also renumber `050_scan_findings.sql` (e.g. → 052 or 056+) so the literal `^05[0-2]_` gate is clean — but that
   is a one-way rename on a *different* feature's migration, so it belongs to that feature's owner, not T01.

I did **not** rename `050_scan_findings.sql` because it is outside T01's "Owns" scope and renaming another
feature's migration is a one-way, non-trivially-reversible change.

## Environment note (shared repo)

The repo is actively shared; during this task a concurrent agent committed `880a7ea2` (knowledge-search tools)
which had bundled my then-staged test file, and a broad untracked sweep produced an intermediate wip commit
`3545b09b` (150 files, incl. a rebuilt `dist/` + other agents' spec files + a 62MB `mnemonic/mnemonic` binary).
I unwound only what was necessary so that **`3f3172b9` is the single, atomic, correctly-attributed T01 commit**
containing only the test file. The migration renames and the test are both tracked in HEAD and the working tree is
clean.

## Work Unit Evidence

| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T01 CLM renumber | `go test ./internal/store/... -run TestMigrationCLMRenumber` | in-process SQLite migrate on fresh store | `git revert 3f3172b9` (removes the test); renames themselves were already on base, so reverting this commit does not re-touch the migration files |

## Files
- `mnemonic/internal/store/migrations/053_tool_outputs.sql`, `054_indexed_files.sql`, `055_context_revisions.sql` — CLM migrations (renamed, already on base).
- `mnemonic/internal/store/migrations/050_scan_findings.sql` — non-CLM file causing the literal-G1 match (the concern).
- `mnemonic/internal/store/migrations_clm_renumber_test.go` — covering test (this commit).
- `mnemonic/internal/store/store.go` — `migrate()` at line 344 (go:embed + sort order).
