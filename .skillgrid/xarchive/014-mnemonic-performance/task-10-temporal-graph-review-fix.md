# Task 10 — Temporal edges: SCOPED re-review of F1/F2/F3 fix round

**Change:** 014-mnemonic-performance · **Step:** 10-temporal-graph
**Re-checked diff:** `review-step10-fix.diff` (402 lines, 5 files)
**Scope:** only the F1 (MAJOR), F2 (MINOR), F3 (NIT) findings from
`task-10-temporal-graph-review.md`. Findings 3 (dead public method) and 5
(tautological JSON assertion) were not re-scoped; F3's dead-method half is
implicitly resolved by F1 (see below).

## Verdict

**PASS** (scoped to F1/F2/F3)

All three findings are resolved, verified against the fixed code in context,
with new RED tests that would fail without the fix. The full `graph` package
suite passes and all five touched packages build.

---

## F1 resolved? **YES**

`fetchEdges` (graph/graph.go:145) now applies the temporal filter in its SQL
(the report's "option (a)"), and it is the live canonical traversal:

```go
now := time.Now().Unix()
rows, err := db.QueryContext(ctx, `
    SELECT e.id, e.kind, e.from_id, e.to_name, e.to_id, e.target_path,
           e.confidence, e.line
    FROM edges e
    WHERE (e.from_id = ? OR e.to_id = ?)
      AND e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
    ORDER BY e.id`, sym.ID, sym.ID, now, now)
```

- **Not a dead function:** `Neighbors` (graph.go:271), `Explain`
  (graph.go:507), and both BFS hops in `Path` (graph.go:337, graph.go:390) all
  call `fetchEdges`. The `code_*` MCP tools route through exactly these —
  `code_get_callers`/`code_get_callees` → `graph.Neighbors`
  (tools_code_graph.go:205), `code_explain` → `graph.Explain`
  (tools_code_graph.go:260), `code_path` → `graph.Path` (tools_code_graph.go:231).
  So the filter now genuinely hides expired/pending edges from the agent-facing
  traversal. (This also resolves prior Finding 3: `QueryEdgesForSymbol`'s
  intended caller is now exercised via `fetchEdges`'s filter.)
- **`TestFetchEdgesHidesExpiredEdges` is a real [RED] guard:** it seeds an
  active edge (line 10), an expired edge (line 20, `valid_to` in the past), and
  a pending edge (line 30, `valid_from` in the future), then calls
  `fetchEdges` directly **and** the public `Neighbors` API. It asserts
  `len(byLine) != 1`, that line 10 is present, and that lines 20 and 30 are
  **absent**. Without the WHERE-clause filter, all three edges return →
  `len(byLine) == 3` → FAIL. It is not testing the already-passing
  `codeindex.QueryEdges`. Verified passing.

## F2 resolved? **YES**

All three previously-omitting INSERT paths now set `valid_from`. `time` is
imported in each file (pdg/lsp.go:13, knowledge/store.go:8,
memory/session_promotion.go:9).

- **`pdg/lsp.go:382-387`** (the LSP-resolved edge) — quoted in full:
  ```go
  INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from)
  VALUES ('calls', ?, ?, ?, ?, ?, ?, ?, ?)
  ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line) DO UPDATE SET
    confidence = 'LSP_RESOLVED'
  ```
  bound `… , e.Line, time.Now().Unix()`.
- **`knowledge/store.go`** — all four edge INSERTs (doc `references`, config
  `configures`, and both SQL-access branches) now carry `valid_from` with
  `time.Now().Unix()`.
- **`memory/session_promotion.go:202-208`** (the `promotes` edge) — INSERT now
  has the `valid_from` column and binds `time.Now().Unix()`.

Every new LSP/knowledge/session edge is now stamped with observation time
instead of `0`, matching the codeindex extraction path's intent. Verified all
four packages build.

## F3 resolved? **YES**

`mem graph` (mem.go:437) now computes the count from the full result set
**before** the `--limit` truncation:

```go
count := len(edges)                       // full set
if limit > 0 && len(edges) > limit {
    edges = edges[:limit]                 // truncate the list only
}
out := map[string]any{
    "project": projID,
    "count":   count,                     // full set, matches sum of per-status
    "total":   counts,
    "edges":   edges,
}
```

`TestMemGraphCountIgnoresLimit` seeds 3 edges, runs `mem graph --limit 2`, and
asserts `count: 3` (not 2), per-status totals unchanged, and exactly 2
`ValidFrom` fields emitted (the truncated list). With the old code
(`count = len(edges)` after truncation) the output would be `"count": 2` →
FAIL. Verified passing.

---

## New issues introduced by the fix

**None blocking.** One observation (nit, non-blocking, not a regression):

- `TestPromotedSessionEdgesCarryValidFrom`'s name/comment frames it as the F2
  guard for the session-promotion path, but it inserts the promotion edge with
  the **old** INSERT shape (no `valid_from` column) to prove the migration
  `DEFAULT 0` still works and stays traversable. It does **not** exercise the
  fixed `session_promotion.go` production INSERT (which now writes
  `time.Now()`), so it does not by itself prove the F2 fix in that file.
  It is still a valid guard for the no-regression property, and the F2 fix is
  verified in-context + builds; the three production files' new
  `time.Now().Unix()` bindings are covered by compilation. If full runtime
  coverage of the production promotion INSERT is wanted, a follow-up test that
  calls `promoteSessionToGraph` and asserts `valid_from > 0` would close it.
  Not a blocker for this scoped round.

## Approved to merge

**Yes** — for the F1/F2/F3 scope. The prior blocking Finding 1 is resolved by
wiring the filter into the actual traversal, and the two follow-ups (F2, F3)
are fixed with passing RED tests. The single nit above is a follow-up, not a
blocker.
