# Task 09 — Session-to-Graph Auto-Promotion Review

Reviewer: fresh (no prior context). Scope: `review-step09-scoped.diff` (4 files) + existing
schema (`store/migrations/001`, `011`, `021`, `022`), `memory/service.go`, fixture.
Tests re-run locally: `go test ./internal/mnemonic/memory/ -run 'TestSessionEnd' -count=1` → all 3 PASS.

## Verdict

**PASS WITH WARNINGS**

Core behavior is correct: node creation, threshold gating, and idempotency all hold, the
representation is sound and purely additive, and the tool contracts are untouched. One
**major** gap: the idempotency test proves reuse only for the *identical-summary* re-trigger,
yet the production dedup key includes `content` — a changed summary between two `SessionEnd`
calls yields a second node observation. This is exactly the scenario the brief's "dedup on
session_id" was meant to close.

## Findings

### MAJOR — Idempotency is content-keyed, not session-keyed; the changed-summary case is untested
`session_promotion.go:389-398` — `upsertPromotedObservation` dedup query:

```sql
SELECT id FROM observations
WHERE project = ? AND type = 'session_log' AND title = ? AND content = ?
  AND deleted_at IS NULL
ORDER BY id LIMIT 1
```

The title `[session <id>] Session summary` encodes the session (stable), but the key **also
includes `content`** (the summary). The symbol side *is* session-stable (`uid = 'session:<id>'`,
UNIQUE), and the `promotes` edges upsert on the (kind, from_id, file_id, to_id, to_name,
target_path, line) unique key with all-non-NULL values.

Consequence: re-trigger `SessionEnd` for the same session with a **changed** summary (the
natural "re-derivation" case — the L0 summary is refreshed during the session, or an operator
re-closes with an improved summary). Then:
- dedup `content = ?` misses → a **second** node observation is inserted (new id).
- the symbol is reused (uid unique) → `UPDATE observations SET graph_ref = ? WHERE id = <new id>`
  re-binds the *new* observation to the *same* symbol.
- net: **1 symbol, 2 node observations, 2 FTS-indexed rows** for one session.

`TestSessionEndIdempotentPromotion` uses `promotionSummary` **verbatim on both ends**, so it
proves only the same-content path (observation reuse, symbol reuse, no duplicate edges). It does
**not** exercise the changed-summary path. The brief says "dedup on session_id"; the symbol half
honors that, the observation half does not.

Fix: make the observation dedup session-stable — `WHERE project = ? AND type = 'session_log' AND
topic_key = 'session:<id>'` (the INSERT already writes `topic_key = nullString("session:"+sessionID)`),
or `AND title = ?` without the `content = ?` clause — and `UPDATE ... SET content = ?` in place so
the FTS update trigger reindexes. Then add a test: end with summary A (≥threshold), end again with
a *different* summary B (≥threshold), assert `COUNT(*) FROM observations WHERE project='mem-promo'
AND type='session_log' AND title='[session s1] Session summary'` == 1 and `content` == B.

### MINOR — Re-trigger with a *shortened* summary degrades the node content to the stub
Same function. Because the dedup key includes content, a re-trigger whose summary fails the
threshold (e.g. re-close with `"done"`) takes no branch at all (`promotionMeetsThreshold` is false
in `SessionEnd`/`PromoteSession`) — fine, the original node survives. But if the summary *changes
and still clears* the bar while differing in length, the original node's `content` is never
refreshed (only `last_seen_at`/`updated_at` are touched on the found-row path, and a new row is
created on the miss path). The node's signature/content on the symbol is likewise frozen at first
promotion (the symbol INSERT only happens when `SELECT ... WHERE uid` returns NoRows). So the
"permanent" node can lag the session's final summary. Low impact (the node is a pointer +
snapshot), but worth a deliberate decision: snapshot-at-first vs. refresh-on-retrigger.

### NIT — `PromoteSession` logs "skipping" even when `SessionEnd` already skipped
`session_promotion.go:254-270`. `PromoteSession` calls `SessionEnd` (which, when the threshold is
met, already promotes *and* logs `promoted to graph node`), then re-checks the threshold and may
log `summary below promotion threshold, skipping graph promotion` — the *second* log line for the
sub-threshold path is correct, but for the above-threshold path the flow promotes inside
`SessionEnd` **and** then calls `promoteSessionToGraph` again (the idempotent upserts make it a
no-op, and it logs a second `promoted to graph node`). Two "promoted" lines for one close. Cosmetic;
the upserts keep it correct. Consider having `PromoteSession` delegate fully to `SessionEnd`
(promotion already runs there) rather than re-running it.

### NIT — Pseudo-file rows accumulate in `files`
`session_promotion.go:365-382`. Each promoted session adds a `files` row (`path='session:<id>'`)
as the `edges.file_id` NOT-NULL anchor. Inert to the codeindex indexer (disk-discovery only) but
visible in any `files` row count / `SELECT DISTINCT file_id` traversal. Implementer already flagged
this (report Concern #3); acceptable under the additive constraint. No action required.

### NIT — Empty marker commit `37b819b`
Confirmed: `git diff 1e2c0ac..37b819b` is empty. The dedup upserts + NULL-unique fix shipped in
`1e3fd23` (verified: that commit already contains `ON CONFLICT(...)` and the `upsertPromotedObservation`
select). This is a process/commit-hygiene nit, **not** a correctness issue — the 09.3 RED→GREEN
cycle (the re-trigger test) is real and green. Drop the empty commit or fold its message into
`1e3fd23` if hygiene matters.

## Evidence checked (the 9 checks)

1. **Idempotency (CRITICAL)** — VERIFIED, with the MAJOR caveat above. Dedup is on
   `(project, type, title, content)` — title is session-stable, **content is not**. Symbol is
   session-stable (`uid='session:<id>'` UNIQUE). Edges upsert correctly (all unique-key columns
   non-NULL — the implementer's NULL-unique diagnosis is correct: SQLite UNIQUE treats NULLs as
   distinct, so a NULL in the key defeats `ON CONFLICT`). Re-trigger with *identical* summary: no
   duplicate (proven by the test). Re-trigger with *changed* summary: **second node observation**
   (untested gap). See MAJOR.
2. **Graph-node representation** — REASONABLE. Node = `observations` row (type `session_log`, an
   existing valid type in `validTypes`, `service.go:1577`) + `symbols` row (kind `session`, uid
   `session:<id>` UNIQUE, stable). `promotes` edges link node symbol → each session observation id
   (`to_id`), `to_name='obs:<id>'`, `target_path='observation:<id>'`, `line=0`, confidence
   `EXTRACTED`, anchored on a session pseudo-file. `kind='promotes'` does not collide with the 011
   indexer kinds (`calls/extends/implements/imports/references/inherits`). Endpoint is
   polymorphic (021 relaxed the symbols FK). The cross-link (`observations.graph_ref → symbols.id`)
   matches the step-07 shape. Sound.
3. **Quality threshold (09.2)** — PRESENT. `promotionMeetsThreshold`: non-empty (after TrimSpace)
   AND ≥ `MinLength` (default 100 runes, configurable via `mnemonic.promotion.min_length`) AND ≥
   `MinSections` `## ` headings (default 1). Empty summary → skip. Reasonable: a one-line `"done"`
   or title-only stub is rejected; a real structured summary passes. `MinSections` is fixed (not
   configurable) — documented.
4. **Queryability** — VERIFIED. The node observation is inserted via the same table, so the
   existing `observations_fts` insert trigger (`001_initial.sql:42`) indexes it; `Search`/
   `SearchWithScope`/`SearchOwnerScoped` (the `mem_search` path) find it via the FTS JOIN with no
   new query. `TestSessionEndCreatesGraphNode` asserts a search for `"token rotation"` returns the
   node observation id. Passes locally.
5. **SessionEnd ordering** — CORRECT. Promotion runs after the sessions-row update + RowsAffected
   check, **before** the distill hook (`service.go:971-975`, hook at 985-989). Synchronous, so the
   node is durable when `SessionEnd` returns. Best-effort: a promotion error is logged to stderr
   (`graph promotion failed (best-effort)`) and does **not** fail the close. The distill hook stays
   detached/async. Existing contract unchanged (same signature/error behavior).
6. **Empty 09.3 commit** — VERIFIED empty (`git diff 1e2c0ac..37b819b` → no files). Process nit
   only; behavior is in `1e3fd23` and is green.
7. **Additive-ness** — VERIFIED. No new tables, no new columns, no migration. Reuses `observations`,
   `symbols`, `edges`, `files`. The only new persistent rows are data rows (session_log observation,
   session symbol, promotes edges, session pseudo-file). Diff is purely additive to
   `config/load.go` (new `Promotion` struct + merge), `memory/service.go` (new field + `SetPromotion`
   + the SessionEnd block), and two new files. Existing schema untouched.
8. **Tool contract** — VERIFIED unchanged. Scoped diff touches no MCP tool registration. The only
   non-diff files changed since baseline (`http/server.go`, `http/ui.go`, `integration/step06_test.go`)
   belong to the intervening step (015, the `/sessions` endpoints), not step 09. `mem_*`/`code_*`/
   `web_*` signatures intact.
9. **TDD discipline** — MOSTLY VERIFIED. (a) node created + mem_search-queryable: `TestSessionEndCreatesGraphNode`
   asserts node count==1, graph_ref set, `promotes` edges to both obs, graph traversal, and a
   mem_search hit. (b) no node below threshold + observations still accessible: `TestSessionEndNoPromotionBelowThreshold`
   covers empty + `"done"` and asserts obs still searchable. (c) no duplicate on re-trigger:
   `TestSessionEndIdempotentPromotion` asserts node count stays 1, same observation id, same symbol,
   no duplicate edges — **but only for the identical-summary re-trigger** (see MAJOR). The three
   tests are well-targeted and green; the one gap is the changed-summary idempotency case.

## Approved to merge

**Yes — with the MAJOR idempotency gap tracked.** The shipped behavior is correct for the
same-summary re-trigger (the realistic double-close case) and fully additive. The changed-summary
second-node case should be closed by re-keying the observation dedup on `topic_key`/title (session-stable)
and adding the one-line test above; it can land as a small follow-up without blocking this step,
since it is an edge case (re-close with a materially different summary) and not a regression.
