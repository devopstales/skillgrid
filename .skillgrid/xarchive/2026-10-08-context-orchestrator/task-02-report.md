# Task Report — T02 (TASK-091): ctx_ schema — migrations 056–061 (6 tables) + store package

- Ticket: TASK-091 (type: feature, priority: high)
- Change: 2026-10-08-context-orchestrator (CTX v1.11)
- Wave 1 · PR1 (foundation) · base = release/2
- Commit: `3cbd14c3` (9 files, +1093)
- Status: COMPLETE — all gates pass

## What was done

### 1. Six additive migrations (one `CREATE TABLE IF NOT EXISTS` each)
Written verbatim from brief §9.3 (column definitions copied, not re-derived).
Auto-picked-up by the existing `go:embed migrations/*.sql` lexicographic runner — no runner edit.

| Migration | Table | PK |
|---|---|---|
| `056_ctx_proxy_sessions.sql` | `ctx_proxy_sessions` | session_id |
| `057_ctx_proxy_savings.sql` | `ctx_proxy_savings` | saving_id |
| `058_ctx_proxy_mode_history.sql` | `ctx_proxy_mode_history` | change_id |
| `059_ctx_session_summaries.sql` | `ctx_session_summaries` | session_id |
| `060_ctx_stats_history.sql` | `ctx_stats_history` | bucket_id |
| `061_ctx_traffic_memories.sql` | `ctx_traffic_memories` | memory_id |

Table-name authority honored: `ctx_proxy_sessions` + `ctx_proxy_savings` (NOT the stale
`ctx_proxy_stats` / `ctx_judgment_cache` in tasks.md). No `ctx_judgment_cache` table.

### 2. `mnemonic/internal/ctx/store/`
- `store.go` — `Store` struct wrapping a passed-in `*store.Store` (ADR-0033: no 2nd DB /
  2nd connection). `New(shared)`, `DB()`, `Shared()`. Fail-open primitives:
  - `failOpenOpen(err, table)` — parses the driver's `no such table: <name>` message,
    compares the parsed name to the accessor's own table so a *different* table's
    missing error is not swallowed.
  - `queryFailOpen()` — on a missing-table error returns `nil, nil` (NOT a fabricated
    `*sql.Rows`, which would panic in `Next()`/`Close()` under `modernc.org/sqlite`).
  - `execFailOpen()` — swallows a missing-table error on inserts so a pre-migration
    store degrades to a no-op write.
- `tables.go` — typed row structs + CRUD per table + `Rollup()` for
  `ctx_stats_history` (GROUP BY granularity, sums tokens_saved/requests/sessions/cache_hits).
  Every List/Rollup method guards `rows == nil` before `defer rows.Close()` / `for rows.Next()`.

### 3. Tests (`store_test.go`) — 5 functions, all pass
| Test | AC covered |
|---|---|
| `TestCtxTablesCreatedOnFreshStore` | AC2a (6 tables on fresh store, via `sqlite_master`) + AC2c (all carry `ctx_` prefix) |
| `TestCtxTableRoundTrip` | AC2b (per-table typed insert + query round-trip) |
| `TestCtxSharedHandleNoSecondConnection` | AC3 (built from existing `*store.Store`, `OpenCount()` unchanged, same `Path()`) |
| `TestCtxFailOpenMissingTable` | DoD fail-open (query a not-created ctx_ table → empty, no panic) |
| `TestCtxRollup` | aggregate rollup grouped by granularity |

Notable test gotchas resolved:
- `SessionSummary.ShapingApplied` is `*bool`; the driver returns a fresh pointer per scan,
  so `reflect.DeepEqual`/`!=` on the struct fails on pointer identity. Added `summaryEqual`
  helper comparing the pointed-to values.
- `queryFailOpen` must return `nil, nil` (not `&sql.Rows{}`) — a zero-value `*sql.Rows`
  panics in `Next()`.

## Gates (all pass)
- G1: `cd mnemonic && go build ./...` → exit 0
- G2: `cd mnemonic && go test ./internal/ctx/store/...` → exit 0 (ok, 5/5 tests)
- G3: `cd mnemonic && go test ./internal/store/...` → exit 0 (ok; 056-061 apply cleanly after 055, CLM trio + existing tables unaffected)
- G4: `cd mnemonic && ls internal/store/migrations/ | grep -E '^05[6-9]_|^06[01]_'` → exit 0 (6 files listed)
- G5: `cd mnemonic && go vet ./internal/ctx/store/...` → exit 0
- gofmt: `gofmt -l internal/ctx/store/ internal/store/migrations/` → clean

## ADR Constraints honored
- ADR-0033 / ADR-0012: ctx store opens NO second handle; asserts `OpenCount()` unchanged.
  Creates ONLY `ctx_`-prefixed tables.
- ADR-0016: fail-open — missing/unmigrated ctx_ table → empty result, never a hard error.
- ADR-0013: commit carries `[skillgrid-context]` block naming TASK-091.

## Work Unit Evidence
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T02 ctx_ schema + store | `go test ./internal/ctx/store/...` (5/5 pass) | in-process SQLite via `store.Open` on a fresh store (shared handle) | `git revert 3cbd14c3` (6 new migration files + internal/ctx/store package; all additive) |

## Files
- `mnemonic/internal/ctx/store/store.go` — Store struct, New/DB/Shared, failOpenOpen, queryFailOpen, execFailOpen, name-parse helpers.
- `mnemonic/internal/ctx/store/tables.go` — typed rows, CRUD, Rollup; nil-rows guards on all List/Rollup.
- `mnemonic/internal/ctx/store/store_test.go` — 5 test functions + `summaryEqual` helper.
- `mnemonic/internal/store/migrations/056_ctx_proxy_sessions.sql` … `061_ctx_traffic_memories.sql` — 6 DDL files.
