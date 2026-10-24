# Task 10 — Temporal knowledge graph edges (valid_from / valid_to) — REVIEW

**Change:** 014-mnemonic-performance · **Step:** 10-temporal-graph
**Reviewed diff:** `review-step10-scoped.diff` (599 lines, 6 files)
**Scope judged:** only the 6 step-10 files + read of existing code they touch.

## Verdict

**NEEDS FIXES**

The schema, backfill, query filter, partial index, CLI status logic, additive-ness,
and tool-contract are all correct. But the **canonical graph-traversal path is not
actually wired to the temporal filter**, and the report describes it as wired. The
filter therefore changes nothing at runtime for the agent-facing graph tools.

---

## Findings

### [major] 1. `fetchEdges` is NOT routed to `QueryEdgesForSymbol` — the report overstates the filter's effect
`skillgrid-cli/internal/mnemonic/graph/graph.go:139-145` (unchanged by the diff):

```go
func fetchEdges(ctx context.Context, db *sql.DB, sym Symbol) ([]Edge, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT e.id, e.kind, e.from_id, e.to_name, e.to_id, e.target_path,
		       e.confidence, e.line
		FROM edges e
		WHERE e.from_id = ? OR e.to_id = ?
		ORDER BY e.id`, sym.ID, sym.ID)
```

There is **no temporal predicate** and **no call to `QueryEdgesForSymbol`**.
`QueryEdgesForSymbol` is only referenced in `codeindex/graph.go:82` (definition) and
`codeindex/graph_test.go` (tests) — it has **zero production callers**.

The report (`task-10-temporal-graph-report.md:83`) states:

> `fetchEdges` (neighbor view / BFS path / graph-stops) — **current state** —
> Yes — via `QueryEdgesForSymbol` (additive method; the existing `fetchEdges`
> now consults the filtered path)

That is inaccurate: `fetchEdges` does not consult the filtered path. Its callers —
`Neighbors` (`graph.go:263`, the MCP `code_get_callers/callees/dependents/...`
tools), `Explain` (`graph.go:499`, `code_explain`), the BFS path
(`graph.go:329`, `graph.go:382`), and `Impact` (`impact.go:120`) — all read the
raw `edges` table and **still include expired and pending edges**.

**Why major, not critical:** (a) nothing *writes* `valid_to` in the current
codebase — no INSERT sets it and there is no expire/retire path, so the filter is
inert *today* either way; (b) the filter is correctly implemented and testable in
`codeindex`. The defect is that the brief's core deliverable — "current graph
traversal hides expired edges" — is not actually achieved by the code shipped, and
the report claims it is. The new filtered API is dead code from the traversal's
point of view.

**Fix:** wire the filter into the traversal. Either (a) add the predicate to
`fetchEdges`'s SQL (`AND e.valid_from <= :now AND (e.valid_to IS NULL OR
e.valid_to > :now)`), or (b) make `fetchEdges` delegate to
`codeindex.QueryEdgesForSymbol` (matching the report's stated design) and adapt the
scan. Then re-run the `graph` package tests.

---

### [minor] 2. Two non-test edge INSERT paths still omit `valid_from` → new edges get `valid_from = 0`
`skillgrid-cli/internal/mnemonic/pdg/lsp.go:380` and
`skillgrid-cli/internal/mnemonic/knowledge/store.go:77,185,252,263` and
`skillgrid-cli/internal/mnemonic/memory/session_promotion.go:203` all do:

```go
INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
VALUES (...)
```

with **no `valid_from`**. Because the migration gives `valid_from` a `DEFAULT 0`,
any edge written by these paths today is stamped `0` = "always active / unknown
observed", indistinguishable from a backfilled pre-migration row — it does *not*
record `time.Now().Unix()`. Behaviorally safe (0 ≤ now ⇒ visible as active), but it
defeats the "edges carry the time the relationship was observed" intent for the
LSP-resolved, knowledge, and session-promotion edges.

**Fix (follow-up acceptable):** add `valid_from` = `time.Now().Unix()` to these
INSERTs, or document that only the `codeindex` extraction path records observation
time.

---

### [minor] 3. `QueryEdgesForSymbol` is a dead public method (no production caller)
`skillgrid-cli/internal/mnemonic/codeindex/graph.go:82`. It is exported,
tested, and filtered correctly, but nothing outside the tests calls it (it was
*supposed* to back `fetchEdges` — see Finding 1). As-is it is an unexercised public
API. Wiring it per Finding 1 resolves this.

---

### [nit] 4. `count` reflects the truncated slice, not the full set
`skillgrid-cli/cmd/skillgrid/mem.go:425-437`: `edges = edges[:limit]` runs *before*
`out := ...{"count": len(edges)}`, so with `--limit N < total`, `count` = N while
`total` (per-status) still reflects the full set — a confusing split. `total` is
computed from the full list, `count` from the truncated one.

**Fix:** compute `count := len(edges)` before truncation, or rename to `shown`/`count`.

---

### [nit] 5. Test JSON-field assertion is tautological
`skillgrid-cli/cmd/skillgrid/mem_graph_test.go:164-168` and `:174-178` assert the
same literals (`"active"`, `"expired"`, `"pending"`) for both the per-status count
map *and* the per-edge status label. Because the count map and the edge label share
identical JSON strings, the edge-label check is satisfied by the count map alone.
The test would pass even if `Status` were never set on the `Edge` (the count map
would still emit `"active": 1`). Not wrong — just weaker than it looks. The
`ValidFrom`/`ValidTo` assertion (`:170-172`) is sound.

---

## Checks verified

| # | Check | Result | Evidence |
|---|-------|--------|----------|
| 1 | Backfill | **PASS** | `023:29` `ALTER TABLE edges ADD COLUMN valid_from INTEGER NOT NULL DEFAULT 0`. `0 <= now` always true ⇒ pre-migration rows stay active. `TestBackfilledEdgesRemainActive` proves it. |
| 2 | New edge INSERT sets `valid_from` | **PARTIAL** | All three `codeindex/indexer.go` sites (990/1029/1169) set `valid_from = time.Now().Unix()` and refresh on conflict. But other non-test INSERT paths (`pdg/lsp.go:380`, `knowledge/store.go`, `session_promotion.go:203`) omit it → get 0. See Finding 2. |
| 3 | Query filtering | **PARTIAL** | `QueryEdges` (graph.go:48-55) filters `valid_from <= ? AND (valid_to IS NULL OR valid_to > ?)`. `QueryEdgesWithHistory` (66-78) has no filter. `QueryEdgesForSymbol` (82-96) filters. **But the canonical traversal `fetchEdges` (graph/graph.go:139) does not** — see Finding 1. Many analysis reads (leiden, godnodes, affected, process, impact, code_intel) remain unfiltered; the report documents this and it is acceptable for analysis paths. |
| 4 | Partial index | **PASS (fine)** | `023:35-37` `CREATE INDEX ... (valid_from, valid_to) WHERE valid_to IS NULL`. Correct: it serves the active-majority (`valid_to IS NULL` branch). The `valid_to > now` (future-expiry) rows are not indexed, but those are a vanishing minority (most edges never expire) — acceptable, and documented in the migration comment. |
| 5 | CLI status | **PASS** | `mem.go` `runMemGraph` shows `valid_from`/`valid_to` (as `ValidFrom`/`ValidTo`) + computed `Status`. `TemporalStatus` (graph.go:33-41): expired when `valid_to != 0 && valid_to <= now`; pending when `valid_from > now`; else active — logic correct. Uses history view (correct for a diagnostic that must *show* expired edges). |
| 6 | Additive-ness | **PASS** | `023` is purely `ADD COLUMN` ×2 + `CREATE INDEX`. No existing column or the `UNIQUE(kind,from_id,file_id,to_id,to_name,target_path,line)` constraint touched. `valid_from NOT NULL DEFAULT 0` does not break existing INSERTs (they get 0). No `009_*`-or-earlier migration touched (023 is the next free number after 022). |
| 7 | Tool contract | **PASS** | Diff touches no `mem_*`/`code_*`/`web_*` tool registration. The `code_*` neighbor/explain tools still go `service → graph.Neighbors/Explain → fetchEdges` (unchanged shapes). Only additive `codeindex` methods + a new `mem graph` CLI subcommand. |
| 8 | TDD discipline | **PASS (for the tested scope)** | (a) `TestEdgeValidFromSetOnCreate` — valid_from in [before,after], valid_to NULL, visible as active. (b) `TestExpiredEdgesHiddenFromQueries` — expired+pending hidden from `QueryEdges`, active shown; history returns all three with correct statuses. (c) `TestMemGraphTemporalStatus` — count 3, per-status counts, ValidFrom/ValidTo present. (d) `TestBackfilledEdgesRemainActive` — valid_from=0 stays active. **Gap:** no test proves the *traversal* (`fetchEdges`/`Neighbors`) actually hides expired edges — consistent with Finding 1 (it doesn't). |

---

## Approved to merge

**No** — conditional. Block on **Finding 1**: either wire the temporal filter into
`fetchEdges` (the actual canonical traversal) so expired edges are genuinely hidden
from graph traversal, or amend the report to state that the filter is implemented
and tested in `codeindex` but not yet wired into traversal (a documented follow-up).
Findings 2–5 are follow-ups/nits and do not block on their own.
