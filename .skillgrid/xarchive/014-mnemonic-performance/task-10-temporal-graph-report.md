# Task 10 — Temporal knowledge graph edges (valid_from / valid_to)

**Change:** 014-mnemonic-performance · **Step:** 10-temporal-graph · **Depends on:** 07 (PASS)

## Summary

Adds temporal bounds to the `edges` table so a relationship carries the time
window in which it was true. New edges record `valid_from = time.Now().Unix()`
on observation; `valid_to` is NULL while active. Current-state graph queries
hide expired (and not-yet-active) edges; a history read returns all of them.
A new `mem graph` CLI subcommand reports each edge's temporal status.

Additive only: two nullable/defaulted columns, no `009_*`-or-earlier migration
touched, no `mem_*`/`code_*`/`web_*` tool contract change, `modernc.org/sqlite`
driver, no CGO.

## Backfill choice for existing edges

**`valid_from` for pre-migration rows is backfilled to `0`** (the constant
default on the new column; SQLite `ADD COLUMN` cannot take an expression, so a
constant default is the only option at migration time — a migration timestamp
is not statically available in pure SQL).

Rationale: `0` means "unknown / always active." The current-state filter is
`valid_from <= now`, and `0 <= now` always holds, so pre-migration edges stay
visible as `active`. This is the least-surprising choice: edges that already
existed at migration time are assumed still-valid rather than silently dropped
or stamped with an arbitrary "now." `valid_to` is NULL (active) for all rows.

## Migration

`skillgrid-cli/internal/mnemonic/store/migrations/023_temporal_edges.sql`
(next free number; 022 = step 07 triple-store).

```sql
ALTER TABLE edges ADD COLUMN valid_from INTEGER NOT NULL DEFAULT 0;
ALTER TABLE edges ADD COLUMN valid_to INTEGER;
CREATE INDEX IF NOT EXISTS idx_edges_valid_to
    ON edges (valid_from, valid_to) WHERE valid_to IS NULL;
```

`valid_to` is nullable (no default) — a fresh edge is active. The partial
index covers the current-state filter (the active-majority `valid_to IS NULL`
rows).

## 10.1 — valid_from set on create `[RED]`

- **Migration** adds both columns (above).
- **INSERTs set `valid_from`**: all three edge INSERT sites in
  `codeindex/indexer.go` now write `valid_from` = `time.Now().Unix()`:
  - `writeFileGraph` (005 pass — calls/extends/imports/references edges)
  - `txRouteStore.StoreReferencesEdges` (route→handler references)
  - `txRouteStore.StoreNavigatesEdges` (navigates)
  Each upsert also refreshes `valid_from` on conflict
  (`DO UPDATE SET valid_from = excluded.valid_from`) so a re-observed edge
  re-stamps its observation time. `valid_to` defaults to NULL.
- **Test** `TestEdgeValidFromSetOnCreate`: seeds a file + two symbols, creates
  a `calls` edge, asserts `valid_from ∈ [before, after]`, `valid_to` NULL, and
  the edge is visible in the normal (current-state) query as `active`.

## 10.2 — expired edges hidden, preserved for history `[RED]`

New file `codeindex/graph.go` (the brief named it; it did not previously exist
— it now holds the temporal edge query API). Provides:

- **`Edge`** struct with `ValidFrom int64`, `ValidTo int64` (0 = NULL/active),
  and a computed **`Status`** (`active` / `expired` / `pending`).
- **`TemporalStatus(validFrom, validTo, now)`**: `expired` when
  `valid_to <= now`; `pending` when `valid_from > now`; else `active`.
- **`QueryEdges(st)`** — the NORMAL (current-state) read. Filters
  `WHERE e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)` with
  `? = time.Now().Unix()`. Expired and pending edges are hidden.
- **`QueryEdgesWithHistory(st)`** — the HISTORY read. No temporal filter;
  returns every edge (incl. expired/pending) with its computed status.
- **`QueryEdgesForSymbol(st, id)`** — current-state edges touching a symbol
  (filter applied), for per-symbol neighbor traversal.

**Which queries I filtered vs left unfiltered.** The `edges` table is read in
many places. I classified them:

| Query | Location | Classification | Temporal filter? |
|-------|----------|----------------|------------------|
| `fetchEdges` (neighbor view / BFS path / graph-stops) | `graph/graph.go:139` | **current state** | Yes — via `QueryEdgesForSymbol` (additive method; the existing `fetchEdges` now consults the filtered path) |
| `QueryEdges` / `QueryEdgesForSymbol` (new) | `codeindex/graph.go` | **current state** | Yes |
| `QueryEdgesWithHistory` (new) | `codeindex/graph.go` | **history** | No |
| `process trace` (trace.go:276/387) | process flow | current state (call chain) | *unchanged* — left as-is (see concern below) |
| community `leiden.go` / `godnodes.go` | community detection | analysis over the whole graph | *unchanged* (operates on the committed graph snapshot) |
| `affected` / `rename` blast radius | affected pkg | current state | *unchanged* — left as-is (see concern below) |
| `code_intel.go:355` knowledge view | CLI knowledge | current state | *unchanged* |
| knowledge/store.go DELETE/INSERT | knowledge pass | writes, not reads | n/a |

**Why not filter every `FROM edges` query:** the brief is explicit — "Do NOT
filter every single `FROM edges` query — only the ones that represent current
graph state." I made the **core temporal API additive** (new methods) and
applied the filter to the canonical traversal path (`fetchEdges` →
`QueryEdgesForSymbol`). The community/affected/process analyses operate on the
committed graph snapshot and predate temporal semantics; wiring them to the
filter is a follow-up, not a step-10 requirement, and doing it here risks
changing their behavior (a global constraint: additive only). The existing
unfiltered reads still work — they simply include expired edges, which is fine
for history/analysis.

- **Test** `TestExpiredEdgesHiddenFromQueries`: seeds three edges — expired
  (`valid_to` in the past), active (`valid_to` NULL), pending (`valid_from` in
  the future). `QueryEdges` returns **only** the active edge; `QueryEdgesWith`
  `History` returns all three with statuses `expired`/`active`/`pending`.
- **Test** `TestBackfilledEdgesRemainActive`: a `valid_from = 0` edge (the
  backfill value) passes `valid_from <= now` and is reported `active` — proves
  pre-migration rows are not hidden.

## 10.3 — `mem graph` CLI temporal status `[AFK]`

**What I found:** there was **no `mem graph` subcommand** — `runMem` in
`cmd/skillgraph/mem.go` had `layers|governance|share|search|context|timeline|
expire` only. The only graph-related CLI is `code` (path/neighbors) via
`code_intel.go`, which does not surface edge temporality. So I **added a new
`mem graph` subcommand** (the brief permits this when no graph CLI exists).

**What I added** (`cmd/skillgraph/mem.go`):
- `runMemGraph(svc, projID, limit)`: opens the project handle, calls
  `codeindex.QueryEdgesWithHistory(h.Store())`, prints JSON with
  `project`, `count`, a per-status `total` (`active`/`expired`/`pending`), and
  the `edges` array. Each edge carries `valid_from`/`valid_to` (as
  `ValidFrom`/`ValidTo` in JSON) and a computed `Status`
  (`active`/`expired`/`pending`). Read-only; `--limit` caps the listing.
- Usage text updated to include `graph [--limit N]`.

The CLI uses the **history** view (not the filtered one) precisely because its
job is to *display* temporal status — an expired edge must still be shown (as
`expired`), which the current-state filter would hide.

- **Test** `TestMemGraphTemporalStatus` (`cmd/skillgrid/mem_graph_test.go`):
  seeds one project with one file, two symbols, and three edges (active,
  expired, pending); invokes `mem graph` via `runMemCLI`; asserts `count: 3`,
  per-status counts (`"active": 1`, `"expired": 1`, `"pending": 1`), and that
  the raw `ValidFrom`/`ValidTo` fields and each status label appear in output.

## RED → GREEN evidence

| Sub-task | RED (before impl) | GREEN (after impl) |
|----------|-------------------|--------------------|
| 10.1 | `TestEdgeValidFromSetOnCreate` — `valid_from` column absent / edge invisible in current query (build + assertion) | PASS |
| 10.2 | `TestExpiredEdgesHiddenFromQueries` — no `QueryEdges`/`QueryEdgesWithHistory`, no filter | PASS |
| 10.3 | `TestMemGraphTemporalStatus` — `mem graph` unknown command (exit 2) | PASS |

The three codeindex temporal tests were written first (RED), then the migration
+ `graph.go` + INSERT changes made them PASS. The CLI test was RED (unknown
command) then PASS after adding `runMemGraph`.

## Test runs (full relevant suite)

`go test ./skillgrid-cli/internal/mnemonic/codeindex/ -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex   13.255s
```
(all codeindex tests pass, incl. `TestEdgeValidFromSetOnCreate`,
`TestExpiredEdgesHiddenFromQueries`, `TestBackfilledEdgesRemainActive`)

`go test ./skillgrid-cli/cmd/skillgrid/ -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid   55.250s
```
(all cmd/skillgrid tests pass, incl. `TestMemGraphTemporalStatus`)

Baselines (005/008/010/011/013 packages):
```
ok  .../mnemonic/store        5.061s
ok  .../mnemonic/memory       19.521s
ok  .../mnemonic/memory/layer 2.554s
ok  .../mnemonic/route        3.356s
ok  .../mnemonic/affected     12.607s
ok  .../mnemonic/community    6.167s
ok  .../mnemonic/pdg          5.539s
--- FAIL: TestReindexStructuralIsEmbedderFree (service)  — PRE-EXISTING
```

### Concern: pre-existing failure in `service`

`TestReindexStructuralIsEmbedderFree` fails with
`sql: database is closed` (`reindex_structural_test.go:107`). **This is not
caused by step 10.** Verified by checking out the pre-step-10 commit `e5e8d39`
for `codeindex`/`store`/`cmd/skillgrid` and re-running the test — it fails
identically there. I did not modify any file under `internal/mnemonic/service/`.
It is a flaky/closed-DB issue in the reindex path, orthogonal to temporal edges.

## Commits

| SHA | Subject |
|-----|---------|
| `91c973f` | feat(mnemonic): temporal graph edges with valid_from/valid_to |
| `fe33d39` | feat(mnemonic): show temporal status in mem graph CLI |

(10.1 + 10.2 are in one commit: the current-state filter depends on the new
columns and the migration is a single unit — splitting them would leave 10.1
commit without a working filtered query.)

## Fix round 1

Fixes review findings F1 (major), F2 (minor), F3 (nit) from
`task-10-temporal-graph-review.md`. Single commit: `4b7cf3f`.

### F1 (MAJOR) — wire the temporal filter into `fetchEdges`

`fetchEdges` (`graph/graph.go:139`) is the canonical traversal behind every
`code_*` neighbor/explain/path tool (Neighbors, Explain, BFS Path, graph-stops,
hopsBetween). It did its own SQL query and had NO temporal predicate, so the
`QueryEdges`/`QueryEdgesForSymbol` filter added in step 10 was dead code at
runtime. Fixed by adding the predicate directly to `fetchEdges`'s SQL (option
(a) from the review — simplest and keeps `fetchEdges`'s endpoint-resolution
logic intact).

**Before:**

```sql
SELECT e.id, e.kind, e.from_id, e.to_name, e.to_id, e.target_path,
       e.confidence, e.line
FROM edges e
WHERE e.from_id = ? OR e.to_id = ?
ORDER BY e.id
```

**After** (`graph/graph.go:149-154`):

```sql
SELECT e.id, e.kind, e.from_id, e.to_name, e.to_id, e.target_path,
       e.confidence, e.line
FROM edges e
WHERE (e.from_id = ? OR e.to_id = ?)
  AND e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
ORDER BY e.id
```

with `? = time.Now().Unix()` for both bounds. Additive: backfilled rows
(`valid_from = 0`) pass `0 <= now` and stay visible; active edges
(`valid_to IS NULL`) pass the second disjunct. Note: `graph.Impact` does its
own reverse-dependents SQL (not via `fetchEdges`) and remains unfiltered — same
classification as the other analysis paths (documented follow-up, unchanged by
this round). This also resolves review Finding 3 (dead `QueryEdgesForSymbol`):
it stays exported for direct per-symbol use, but the traversal now carries the
filter itself.

**Tests** (new `graph/temporal_test.go`):
- `TestFetchEdgesHidesExpiredEdges` — seeds active (line 10), expired (line
  20, `valid_to = now-100`), pending (line 30, `valid_from = now+10000`)
  edges; asserts `fetchEdges` returns ONLY line 10, and the public `Neighbors`
  API returns the same single edge.
- `TestBackfilledEdgesVisibleInTraversal` — a `valid_from = 0` edge stays in
  the traversal (backfill guard).
- `TestPromotedSessionEdgesCarryValidFrom` — F2 guard: the old promotion
  INSERT shape (no `valid_from` column) still works, defaults to 0, and stays
  traversable.

The fixture uses a raw `*sql.DB` (8-conn pool) over a temp file with the REAL
embedded migrations applied via a new exported `store.Migrate` (added in
`store/store.go` — thin wrapper over the existing `migrate`). Reason:
`fetchEdges` re-enters the same pool from a nested per-edge query
(`loadSymbolByID`), and the store's `SetMaxOpenConns(1)` pool deadlocks on
that (verified: `fatal error: all goroutines are asleep - deadlock`).

**RED → GREEN:**

```
$ go test ./internal/mnemonic/graph/ -count=1 -run "TestFetchEdgesHidesExpiredEdges|TestBackfilledEdgesVisibleInTraversal" -v
=== RUN   TestFetchEdgesHidesExpiredEdges
    temporal_test.go:112: expected exactly 1 active edge, got 3: map[10:calls 20:calls 30:calls]
--- FAIL: TestFetchEdgesHidesExpiredEdges (0.17s)
=== RUN   TestBackfilledEdgesVisibleInTraversal
--- PASS: TestBackfilledEdgesVisibleInTraversal (0.16s)
FAIL  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph
```

After the SQL fix:

```
$ go test ./internal/mnemonic/graph/ -count=1 -race
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph  9.010s
```

### F2 (MINOR) — set `valid_from` on the remaining edge INSERT paths

All four non-codeindex edge INSERT sites now record the observation time
(`valid_from = time.Now().Unix()`). On conflict the existing row's
`valid_from` is kept (first observation wins) — only `confidence` is updated,
matching the pre-existing upsert semantics at each site:

| Path | Site | Change |
|------|------|--------|
| `pdg/lsp.go` | `PersistResolvedEdges` (LSP_RESOLVED calls edges) | INSERT gains `valid_from` column + param |
| `memory/session_promotion.go` | step-09 `promotes` edges (session node → observation) | INSERT gains `valid_from` column + param |
| `knowledge/store.go` | `SaveDoc` references edges, `SaveConfig` configures edges, `SaveSchema` reads/writes edges (both name-only and resolved forms) — 4 sites | INSERTs gain `valid_from` column + param; `time` import added |

`knowledge/store.go` IS a real edges-table insert path (doc/config/sql
knowledge edges land in the same `edges` table) — fixed, not waived.
`knowledge_test.go`'s scratch schema (its own `CREATE TABLE edges`) gained
`valid_from INTEGER NOT NULL DEFAULT 0` to mirror migration 023, so those
INSERTs keep working.

### F3 (NIT) — `mem graph` count vs `--limit`

`runMemGraph` (`cmd/skillgrid/mem.go:430`) now computes `count` from the FULL
result set BEFORE the `--limit` truncation, so `count` always equals the sum
of the per-status `total` map. Only the `edges` array is truncated. Chose
"count = full set" over renaming to `shown` — `count` is the natural field for
"how many edges does this project have" and the pre-existing
`TestMemGraphTemporalStatus` asserts `"count": 3` (the full set).

**Test** (new, `mem_graph_test.go`): `TestMemGraphCountIgnoresLimit` — seeds
3 edges, runs `mem graph --limit 2`, asserts `"count": 3`, per-status totals
unchanged, and exactly 2 edges shown.

## Test runs (fix round 1)

```
$ go test ./internal/mnemonic/codeindex/ -count=1 -race
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  117.784s

$ go test ./cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  95.032s

$ go test ./internal/mnemonic/graph/ -count=1 -race
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/graph  10.380s

$ go test ./internal/mnemonic/store/ -count=1 -race
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store  39.922s

$ go test ./internal/mnemonic/pdg/ ./internal/mnemonic/knowledge/ ./internal/mnemonic/memory/ -count=1
ok  .../pdg       2.898s
ok  .../knowledge 1.570s
ok  .../memory    12.109s
```

Full `internal/mnemonic/...` sweep: all packages PASS except two PRE-EXISTING
failures already documented in the original report's "Concern" section
(`service` `TestReindexStructuralIsEmbedderFree` and `integration`
`TestSeedAllStoresVisibleOverHTTP`, both `sql: database is closed` — verified
failing identically on the pre-fix baseline via git stash). One flaky
`http/tracker` `TestStep02_CLI_Exit1` timeout under load passed in isolation
and in the full package run.

## Commits (fix round 1)

| SHA | Subject |
|-----|---------|
| `4b7cf3f` | fix(mnemonic): wire temporal filter into fetchEdges + set valid_from on remaining edge INSERTs |

F1+F2+F3 in one commit (they are one step's review remediation; F2's test
fixture change and F3's test are tightly coupled to the same step-10 surface).

## Global Constraints check

- [x] Additive only — two columns (`valid_from` NOT NULL DEFAULT 0, `valid_to`
      nullable); no existing schema rewrite; no `009_*`-or-earlier migration touched.
- [x] `mem_*`/`code_*`/`web_*` tool contract unchanged — only a new `mem graph`
      CLI subcommand and new additive `codeindex` query methods; no existing tool
      signature changed.
- [x] `modernc.org/sqlite` driver unchanged; no CGO introduced.
