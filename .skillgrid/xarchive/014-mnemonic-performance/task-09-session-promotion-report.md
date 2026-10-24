# Task 09 — Session-to-Graph Auto-Promotion Report

## Summary

L0/L1 session → L2/L3 graph auto-promotion on SessionEnd. A session close
with a summary that meets the quality threshold is promoted to a PERMANENT
graph node: an observation (mem_search-queryable via the existing FTS path)
plus a kind='session' symbol in the codeindex graph (traversable via the
edges table), linked to every session observation through 'promotes' edges.
All changes additive; no existing schema rewritten; the `mem_*`/`code_*`/
`web_*` tool contracts are unchanged; `modernc.org/sqlite` (no CGO) kept.

## Per sub-task

### 09.1 [RED] SessionEnd with valid summary creates permanent graph node

- `TestSessionEndCreatesGraphNode` (session_promotion_test.go): start a
  session, save two observations (one with a real graph_ref via a seeded
  symbol), end with a populated structured summary; asserts
  (1) exactly one promoted node (a kind='session' symbol, uid
  `session:<id>`), (2) the node observation links to its symbol via
  graph_ref, (3) the node has a 'promotes' edge to EVERY session
  observation, (4) the node is traversable through the codeindex graph
  (graph_ref → edges JOIN), (5) the node is queryable via mem_search
  (the existing FTS path — no new query).
- RED: `expected exactly 1 promoted graph node, got 0` → FAIL.
- GREEN: PASS (0.24s).

### 09.2 [RED] No promotion when summary is empty or below quality threshold

- `TestSessionEndNoPromotionBelowThreshold`: empty summary → no node;
  low-quality summary ("done") → no node; the session observations remain
  accessible via mem_search in both cases (promotion is additive, not a
  gate on existing access).
- RED: the below-threshold branch was not implemented (no threshold check
  → a node would be created) → FAIL.
- GREEN: PASS (0.23s).

### 09.3 [AFK] Promotion is idempotent — no duplicate graph nodes

- `TestSessionEndIdempotentPromotion`: end with a valid summary → node
  created; end the same session again (re-trigger) → no duplicate node,
  same observation id reused, same symbol id, the 'promotes' edge set is
  unchanged (each observation linked exactly once).
- RED: re-trigger created duplicate edges (4 instead of 2) → FAIL.
- GREEN: PASS (0.25s).

## Graph-node representation

The "permanent graph node" is a single row in EACH of the two stores that
make it queryable (both in the same per-project SQLite DB, per step 07):

| Store | Row | Why |
|-------|-----|-----|
| `observations` | type='session_log', title=`[session <id>] Session summary`, content=summary, topic_key=`session:<id>`, source='session_promotion' | mem_search-queryable: the existing observations_fts triggers index it on insert, so the FTS search path finds it with no new query. |
| `symbols` | kind='session', name/title=the node title, signature=summary, uid=`session:<id>` | codeindex graph node: the symbol_fts trigger indexes it; the node observation's graph_ref points at it (the step-07 triple-store cross-link shape). |

**Link to session observations**: `edges` rows with kind='promotes',
from_id = the node symbol id, to_id = each session observation id,
to_name=`obs:<id>`, target_path=`observation:<id>`, line=0, confidence
'EXTRACTED'. The endpoint is polymorphic (021 relaxed the symbols FK),
edges.file_id points at a session pseudo-file (path `session:<id>` — a
files row the codeindex indexer never discovers, so it is an inert graph
anchor that satisfies the NOT-NULL file FK), and the kind='promotes'
namespace does not collide with any codeindex edge kind (the 011 indexer
owns kinds like 'calls'/'references' and never writes 'promotes').

**Dedup on session_id**: the node observation is found by
`WHERE project = ? AND type = 'session_log' AND title = ?` with the
deterministic title `[session <id>] Session summary` (content is the
summary — an exact-match upsert, so a re-trigger with the same summary
reuses the row in place, refreshing last_seen_at/updated_at). The symbol
is found by `uid = 'session:<id>'` (unique). The 'promotes' edges are
upserted on the edges unique constraint.

## Quality threshold

`promotionMeetsThreshold` (session_promotion.go): the summary must be
non-empty AND at least **100 runes** long (after TrimSpace) AND contain at
least **1 `## ` markdown heading** (a structured session summary has at
least one section). Defaults are `defaultPromotionMinLength = 100`,
`defaultPromotionMinSections = 1`; `MinLength` is configurable via the
`mnemonic.promotion.min_length` config key (see `SetPromotion` /
`config.Promotion`); `MinSections` is fixed at the documented default.
When the threshold is not met, promotion is skipped with a debug log
(`[mnemonic] session <id>: summary below promotion threshold, skipping
graph promotion`) and the close still succeeds.

Note on `LayerSummary`: the brief references a `LayerSummary` struct that
does not exist; at SessionEnd the available layer data is the `summary`
string argument (the L0 layer summary — the distill hook consumes it as
such), so the quality check runs on that string.

## Idempotency dedup query

```sql
-- node observation (reused in place on re-trigger)
SELECT id FROM observations
WHERE project = ? AND type = 'session_log' AND title = ?
  AND deleted_at IS NULL
ORDER BY id LIMIT 1;

-- node symbol (uid is UNIQUE)
SELECT id FROM symbols WHERE uid = ?;   -- 'session:<session_id>'

-- promotes edge (upserted, never duplicated)
INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
VALUES ('promotes', ?, ?, ?, ?, ?, 'EXTRACTED', 0)
ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line)
DO UPDATE SET confidence = excluded.confidence;
```

**SQLite pitfall found during TDD**: in a UNIQUE index, NULL values are NOT
considered equal, so a NULL column in the unique key makes the upsert a
no-op (every insert is "new"). The first GREEN attempt used
`line = NULL` + `target_path = NULL` and accumulated duplicate edges on
re-trigger (verified with an isolated diagnostic: 2 inserts → 2 rows,
second pass → 4 rows). Fix: populate every unique-key column with a
non-NULL deterministic value (to_name/target_path labels, line=0 — a real
value the codeindex indexer never writes for its edges).

## Where promotion is called in SessionEnd (ordering vs the distill hook)

`SessionEnd` (memory/service.go) runs the promotion **after the sessions
row update and RowsAffected check, BEFORE the session-close distill hook**,
synchronously:

1. Validate + update the sessions row (summary, ended_at, status='ended',
   title derivation) — unchanged.
2. RowsAffected check — unchanged.
3. **Promotion** (new): if `promotionMeetsThreshold(summary)`, call
   `promoteSessionToGraph` synchronously. Best-effort: a failure logs to
   stderr and does NOT fail the close.
4. Distill hook (013, unchanged): the detached, async, best-effort
   goroutine still fires after promotion.

Ordering rationale: the promoted node must be durable the moment
SessionEnd returns (a next-session read can already see it), while the
distill hook stays detached/async and best-effort (it re-derives L1 layers
from the same L0 summary, so it does not depend on the node). The existing
SessionEnd contract is unchanged: same signature, same error behavior,
same distill-hook semantics.

A convenience `PromoteSession(ctx, sessionID, summary)` method (SessionEnd
+ promotion) is also exposed and used by the tests.

## Queryability via mem_search

The promoted node IS an observation row, so it is indexed by the existing
observations_fts triggers on insert and is found by the existing
`Search`/`SearchWithScope`/`SearchOwnerScoped` FTS path — no new query, no
tool contract change. `TestSessionEndCreatesGraphNode` asserts a search
for a term from the summary ("token rotation") returns the node
observation id. Graph traversal is asserted via a direct edges + graph_ref
JOIN (the node is a symbol in the edges table).

## RED→GREEN evidence

| Sub-task | Test | RED | GREEN |
|----------|------|-----|-------|
| 09.1 | TestSessionEndCreatesGraphNode | `expected exactly 1 promoted graph node, got 0` | PASS 0.24s |
| 09.2 | TestSessionEndNoPromotionBelowThreshold | no threshold branch → node created below threshold | PASS 0.23s |
| 09.3 | TestSessionEndIdempotentPromotion | `re-triggered end must not duplicate 'promotes' edges (got 4, want 2)` | PASS 0.25s |

## Full test output

### memory package (the changed package)

```
$ go test ./internal/mnemonic/memory/ -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	30.733s
```

### Baselines (005/008/010/011/013-related packages)

```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store	6.285s
--- FAIL: TestReindexStructuralIsEmbedderFree (0.99s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
FAIL	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service	49.703s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex	17.393s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route	3.042s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected	15.989s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community	6.759s
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg	6.122s
FAIL
```

`TestReindexStructuralIsEmbedderFree` is PRE-EXISTING: it fails identically
at baseline `fa3a64d` (step 08 complete) in a clean worktree — unrelated to
this change (no store/service wiring touched by promotion; the failure is a
"database is closed" race in the reindex path).

## Commits

- `1e3fd23` feat(mnemonic): session-to-graph promotion on SessionEnd
- `1e2c0ac` feat(mnemonic): skip promotion when summary below quality threshold
- `37b819b` feat(mnemonic): idempotent session promotion with dedup (empty marker — the dedup upserts + NULL-unique fix shipped in 1e3fd23; 09.3's RED→GREEN cycle is the re-trigger test)

## Files changed

- `skillgrid-cli/internal/mnemonic/memory/session_promotion.go` (NEW): promotion logic, threshold, config seam.
- `skillgrid-cli/internal/mnemonic/memory/session_promotion_test.go` (NEW): the three tests.
- `skillgrid-cli/internal/mnemonic/memory/service.go`: `promotionCfg` field + `SetPromotion`; promotion call in `SessionEnd` (after sessions row update, before distill hook).
- `skillgrid-cli/internal/mnemonic/config/load.go`: `Promotion` struct + `mnemonic.promotion` YAML section + merge.
- `skillgrid-cli/internal/mnemonic/service/service.go`: `SetPromotion` wiring from config.

## Concerns

1. **Empty commit for 09.3**: the dedup upserts and the NULL-unique fix are
   behavior that only the 09.3 re-trigger test exercises, so they landed in
   the 09.1 commit (1e3fd23) to keep each commit green; 37b819b is an empty
   marker commit for the 09.3 RED→GREEN cycle. If the empty commit is
   undesirable it can be dropped without behavior change.
2. **Pre-existing failure**: `TestReindexStructuralIsEmbedderFree`
   (service package) fails at baseline too — not caused by this change.
3. **Pseudo-file rows**: each promoted session adds a `files` row
   (path `session:<id>`) as the edges.file_id anchor. It is inert to the
   codeindex indexer (disk-discovery only) but appears in files-row counts.
   If that matters, a dedicated nullable anchor could replace it — but that
   would require a schema change (edges.file_id is NOT NULL), out of scope
   for the additive constraint.
4. **Promotion is always-on when the threshold is met** (not a global
   opt-in flag): the brief's "opt-in-ish" is satisfied by the threshold
   gate itself (only a valid structured summary promotes). A
   `mnemonic.promotion.enabled` kill-switch was not added — the threshold
   is the gate; adding one is a one-line follow-up if desired.
