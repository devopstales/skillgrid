# Task Brief — T02 (TASK-091): ctx_ schema — migrations 056–061 (6 tables) + store package

- Ticket: TASK-091 (type: feature, priority: high)
- Change: 2026-10-08-context-orchestrator (CTX v1.11)
- Wave 1 · PR1 (foundation) · base = release/2
- Reversibility: ONE-WAY (additive migration DDL — a wrong column/type is baked into the shared store)
- Precondition: T01/TASK-089 done — CLM trio is at 053/054/055, so 056+ is free (verified on base).
- SATISFIES: schema-ctx-tables, fail-open-degrade (store-level: accessors must not error a missing ctx_ table into a hard fail; endpoint 200-empty is asserted in T10)

## Scope
Create `mnemonic/internal/ctx/store/` and six additive migrations **056–061** in
`mnemonic/internal/store/migrations/`. Each migration creates EXACTLY one `ctx_*` table
using the plan §9.3 schema verbatim (the column definitions are below — copy them, do not
re-derive). Provide store accessors that open via the **shared** `*store.Store` handle
(never a 2nd DB / 2nd connection — ADR-0033) with typed CRUD per table + the aggregate
rollup query for `ctx_stats_history`.

## The six tables + migrations (plan §9.3 verbatim — DO NOT alter columns)

| Migration file | Table | PK |
|---|---|---|
| `056_ctx_proxy_sessions.sql` | `ctx_proxy_sessions` | session_id |
| `057_ctx_proxy_savings.sql` | `ctx_proxy_savings` | saving_id |
| `058_ctx_proxy_mode_history.sql` | `ctx_proxy_mode_history` | change_id |
| `059_ctx_session_summaries.sql` | `ctx_session_summaries` | session_id |
| `060_ctx_stats_history.sql` | `ctx_stats_history` | bucket_id |
| `061_ctx_traffic_memories.sql` | `ctx_traffic_memories` | memory_id |

Column definitions (verbatim from `.skillgrid/artifacts/context-orchestrator-plan.md` §9.3).
Write each as `CREATE TABLE IF NOT EXISTS <table> ( ... )` with the columns below. INTEGER
columns are unix-millis epoch (the codebase convention); add the inline `-- comment` for
each column as shown (comments are part of the DDL and aid future readers):

```
ctx_proxy_sessions(
  session_id            TEXT PRIMARY KEY,
  agent                 TEXT,     -- detected from headers/UA
  mode                  TEXT,     -- token | cache | passthrough
  started_at            INTEGER,
  ended_at              INTEGER,
  requests              INTEGER,
  tokens_before         INTEGER,
  tokens_after          INTEGER,
  cache_hits            INTEGER,
  cache_invalidations   INTEGER
)

ctx_proxy_savings(
  saving_id             TEXT PRIMARY KEY,
  session_id            TEXT,
  turn                  INTEGER,
  upstream              TEXT,
  mode                  TEXT,
  tokens_before         INTEGER,
  tokens_after          INTEGER,
  savings_ratio         REAL,
  created_at            INTEGER
)

ctx_proxy_mode_history(
  change_id             TEXT PRIMARY KEY,
  session_id            TEXT,
  old_mode              TEXT,
  new_mode              TEXT,
  reason                TEXT,
  changed_at            INTEGER
)

ctx_session_summaries(
  session_id            TEXT PRIMARY KEY,
  agent                 TEXT,
  duration_ms           INTEGER,
  turns                 INTEGER,
  tokens_saved          INTEGER,
  judgment_keeps        INTEGER,
  judgment_truncs       INTEGER,
  prefilter_pins        INTEGER,
  prefilter_deads       INTEGER,
  cache_invalidations   INTEGER,
  shaping_applied       BOOLEAN,
  effort_routed         INTEGER,
  traffic_memories      INTEGER,
  summary_json          TEXT,     -- the full wide event
  created_at            INTEGER
)

ctx_stats_history(
  bucket_id             TEXT PRIMARY KEY,
  granularity           TEXT,     -- hourly | daily | weekly | monthly
  bucket_start          INTEGER,
  tokens_saved          INTEGER,
  requests              INTEGER,
  sessions              INTEGER,
  cache_hits            INTEGER,
  created_at            INTEGER
)

ctx_traffic_memories(
  memory_id             TEXT PRIMARY KEY,
  session_id            TEXT,
  category              TEXT,     -- error_recovery | environment | preference | architecture
  memory_text           TEXT,
  evidence_json         TEXT,
  confidence            REAL,
  created_at            INTEGER
)
```

**Table-name authority:** the table names above (from blueprint §9.3 + plan §9.3) are the
source of truth. `tasks.md` TICKET-02 body names two of them differently
(`ctx_proxy_stats`, `ctx_judgment_cache`) — that is a plan typo/stale draft. Use
`ctx_proxy_sessions` + `ctx_proxy_savings` (what blueprint + plan + the later T04/T09/T10
tickets reference). There is NO `ctx_judgment_cache` table in v1.11 §9.3; the judgment
layer (T14) has no dedicated store table in this ticket.

## Acceptance Criteria
1. `go build ./...` green.
2. `go test ./internal/ctx/store/...` green:
   a. each of the 6 tables is created on a fresh store (assert via `sqlite_master`).
   b. each table's typed insert + query round-trips (insert a row, read it back, compare).
   c. NO created table name lacks the `ctx_` prefix.
3. Opening the ctx store takes no handle of its own — the accessors operate on a passed-in
   `*store.Store` (or its `.DB`), and a test asserts the ctx store is constructed from an
   existing shared `*store.Store` (do not call `store.Open` to make a second connection).
   Mirror the pattern in `internal/store/store_pooling_test.go` for how a shared handle is
   proven, but assert the simpler property: `ctxstore.New(existingStore)` returns a ctx
   accessor whose queries hit the SAME database file as `existingStore.Path()`.

## Definition of Done
- Tests pass for touched packages; lint/format pass; edge cases covered; no new warnings.
- Accessors degrade fail-open at the store level: a query for a `ctx_*` table that does not
  exist (e.g. store opened before migration ran) returns an empty result set / zero value,
  NOT a hard error that would 500 an endpoint. (Full endpoint 200-empty is T10's job; here
  just ensure the accessor API doesn't panic/hard-fail on missing table.)

## Implementation Plan
1. Create the six migration files `mnemonic/internal/store/migrations/05{6..9}_*.sql`,
   `060_*.sql`, `061_*.sql`, each with the verbatim DDL above (one `CREATE TABLE IF NOT
   EXISTS` per file). They are picked up automatically by the existing `go:embed migrations/*.sql`
   + lexicographic-sort runner — NO runner edit needed (confirm by building + running a fresh-store test).
2. Create `mnemonic/internal/ctx/store/store.go`:
   - `type Store struct { db *sql.DB }` (or hold the `*store.Store`).
   - `func New(shared *store.Store) *Store` — wraps the shared handle, no new connection.
   - Typed accessors per table (insert/get/list) + `Rollup(...)` for `ctx_stats_history`
     (aggregate tokens_saved/requests/sessions/cache_hits grouped by granularity + bucket).
3. Create `mnemonic/internal/ctx/store/tables.go` (typed rows + CRUD) and `rollup.go` if you
   prefer the split the ticket names; a single `store.go` is acceptable if it stays one clear
   responsibility. Keep it lean — accessors are thin over the shared handle.
4. Tests (`mnemonic/internal/ctx/store/*_test.go`):
   - fresh store via `store.Open(t.TempDir(), "ctxstore")` → assert all 6 ctx_ tables exist.
   - per-table insert + query round-trip.
   - shared-handle assertion (AC3): build ctx accessor from an existing `*store.Store`, assert
     it queries the same file (no second connection).
   - fail-open: query a `ctx_*` table on a store where it was NOT created → empty, no panic.
5. `go build ./...`, `go test ./internal/ctx/store/... ./internal/store/...`, `gofmt`/`go vet`.

## Gates
- G1: `cd mnemonic && go build ./...` → EXPECT exit 0
- G2: `cd mnemonic && go test ./internal/ctx/store/...` → EXPECT exit 0
- G3: `cd mnemonic && go test ./internal/store/...` → EXPECT exit 0 (migrations 056-061 apply
  cleanly in order after 055 on a fresh store; CLM trio + existing tables unaffected)
- G4: `cd mnemonic && ls internal/store/migrations/ | grep -E '^05[6-9]_|^06[01]_'` → EXPECT exit 0
- G5: `cd mnemonic && go vet ./internal/ctx/store/...` → EXPECT exit 0
- Fails-when: `go test ./internal/ctx/store/...` exit != 0 OR any created table name lacks `ctx_` prefix
  OR the shared-handle assertion fails (a 2nd connection is opened).

## ADR Constraints (do not re-litigate)
- ADR-0033 / ADR-0012: ctx_* tables live on the SHARED SQLite store; the ctx store opens NO
  second DB handle / connection and creates ONLY `ctx_`-prefixed tables.
- ADR-0016: fail-open — a missing/unmigrated ctx_* table degrades to empty, never a hard error.
- ADR-0013: commit carries a `[skillgrid-context]` block naming TASK-091.

## Owns (edit only under these)
- `mnemonic/internal/store/migrations/05{6..9}_*.sql`, `060_*.sql`, `061_*.sql`
- `mnemonic/internal/ctx/store/*.go` (+ test fixtures)

## Work Unit Evidence (fill in report)
| Unit | Focused test | Runtime harness | Rollback boundary |
|------|--------------|-----------------|-------------------|
| T02 ctx_ schema + store | `go test ./internal/ctx/store/...` | in-process SQLite via `store.Open` on a fresh store (shared handle) | `git revert <sha>` (6 new migration files + internal/ctx/store package; all additive) |
