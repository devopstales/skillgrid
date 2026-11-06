# Task 23 — Hub Impact Analysis: Review

FRESH reviewer. Scope: 2 commits (`e2bded5`, `8b55459`), 4 files. Verdict below.

## Verdict: `PASS WITH WARNINGS`

The code is clean, the schema is additive, the tool contract is untouched, and the
TDD discipline is real. The SQL counting logic is internally correct (DISTINCT
importing files, right kind, right temporal window, right denominator). There is,
however, a **major** production-realism gap: the importer count is keyed on
`edges.to_id`, but the codeindex does **not** resolve import edges to a target
symbol id (`to_id` is NULL for most imports). The whole feature is therefore
vacuously 0 against real indexed code, and the tests pass by seeding the invariant
rather than the data. That is a scope/feature-correctness warning, not a defect in
the code as written, and it does not change any `mem_*`/`code_*`/`web_*` contract.

---

## Findings

### F1 — MAJOR — `importersForFiles` counts `to_id`-resolved imports, which the codeindex never writes

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:161-170` (also the
dependent-count subquery at `hub.go:521-527`)

The counting query is:
```sql
SELECT s2.file_id AS to_file, COUNT(DISTINCT e.file_id) AS importers
FROM edges e
JOIN symbols s2 ON s2.id = e.to_id
WHERE e.kind = 'imports'
  AND e.to_id IS NOT NULL
  AND e.valid_from <= ?
  AND (e.valid_to IS NULL OR e.valid_to > ?)
  AND s2.file_id IN (...)
GROUP BY s2.file_id
```

This counts importers **only for edges whose `to_id` is a resolved symbol in the
target file**. The SQL shape is correct (DISTINCT `e.file_id` = the *importing*
file, not a raw edge count; `e.to_id` = the *target* symbol via `s2.file_id`), but
the `e.to_id IS NOT NULL` premise is the load-bearing assumption, and it is not
met in production.

Evidence from the codebase:
- `extract/adapter.go:181-192` — `addImport` sets `kind="imports"`, `toName=target`,
  `target=target`, and **never sets `toUID`**.
- `extract/adapter.go:282-303` — the from-UID stamping loop `continue`s for
  `kind == "imports"`, so imports also have no `fromUID`.
- `codeindex/indexer.go:1161-1167` — `toID` is only set when `e.ToUID != ""`. For
  an import, `ToUID` is empty, so `to_id` is stored **NULL**. `from_id` falls back
  to the file's first symbol (the package symbol).
- `codeindex` never resolves an import *path* to a target symbol id. There is no
  `target_path → symbol` resolution that writes `to_id` for imports (confirmed: no
  `UPDATE edges SET to_id` for imports anywhere).
- The existing blast-radius walker knows this: `graph/impact.go:118-126` traverses
  reverse import dependents with the **dual** predicate
  `e.to_id = ? OR (e.to_id IS NULL AND e.to_name IN (SELECT name FROM symbols WHERE id = ?))`.

So in real indexed code, an import from `a.go` into `b.go` is stored as
`kind='imports', from_id=<a.go pkg sym>, file_id=a.go, to_id=NULL, to_name='b.go'`.
The hub query's `JOIN symbols s2 ON s2.id = e.to_id` / `e.to_id IS NOT NULL` drops
that edge entirely. `IdentifyHubFiles` returns no hubs; every `hub_score` is 0;
`RecomputeRiskScores` stamps every `risk_score` 0; `mem graph --risk` (default
threshold 0.5) prints an empty `entries` list.

**Why the tests pass:** `hub_test.go:597-619` `seedImporterFile` writes
`to_id = hubSymID` (non-NULL) on purpose — it seeds the exact invariant the
production extractor does not guarantee. The RED/GREEN cycle is therefore
validating the query against a synthetic premise, not against the real import
edge shape.

**The fix (for a follow-up, not a blocker on this step):** count importers the way
the existing impact walker does — resolve the import target by *name/path*, not by
`to_id`. E.g. count `DISTINCT e.file_id` for `kind='imports'` where the target is
this file, matching on `to_name`/`target_path` against the file (or its package
symbol name), mirroring `graph/impact.go` and `affected/affected.go`. As written,
the step is "correct against the edges table schema" but "wrong against the data
the codeindex actually writes."

Note this is a **major** warning, not a blocker: it does not break, delete, or
reshape anything; it only means the feature yields all-zero advisory scores against
production data until import targets are resolved. The schema, contract, and code
are all sound.

---

### F2 — MAJOR — `risk_score` is `hub_score` only; it does not combine `importance_score` (step 13)

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:388-407` (`RecomputeRiskScores`)

The brief's Interfaces section (line 29) says step 23 **consumes** "importance
scoring from step 13." The review's check 5 asks whether `risk_score` is a
**combination** of `importance_score` + hub impact. It is not:

```sql
UPDATE observations
SET risk_score = COALESCE((
    SELECT COALESCE(s.hub_score, 0)
    FROM symbols s
    WHERE s.id = observations.graph_ref
), 0)
WHERE project = ? AND deleted_at IS NULL
```

`risk_score` is a pure copy of the referenced file's `hub_score`. An observation
about a hub file scores higher (good — the test `TestRiskScoreOnObservations`
verifies this), but step 13's `importance_score` contributes **nothing**. There is
no reference to `importance` in `hub.go` or `mem.go`. The two scores are
independent columns on the same row, never combined.

This is a **brief/implementation mismatch**, not a code defect: the brief's own
task 23.3.c says "compute it as the `hub_score` of the referenced file (0 for
non-hub files)" — which is exactly what was implemented. So the code matches the
*task*, while the *Interfaces* line over-promises by listing step 13 as an input.
Either the Interfaces line should drop "importance scoring from step 13," or
`risk_score` should actually blend it. As-is, an observation about a hub file that
is also highly-important and recent gets no boost from importance. Recommend
recording this as a scope decision in the change's `design.md`/`change.md` so the
next reviewer isn't surprised.

---

### F3 — MINOR — `totalIndexedFileCount` denominator is correct, but `IdentifyHubFiles` returns early on an empty index without stamping

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:255-262`

`total <= 0 → return nil, nil` is correct for divide-by-zero safety (check 2), but
it means an empty index stamps nothing. That is fine (no symbols to stamp), and the
comment says so. Flagging only to confirm the divide-by-zero guard is present and
correct — it is. `hf.HubScore = float64(n)/float64(total)` at `hub.go:281` is
reached only when `total > 0`. No action needed.

---

### F4 — NIT — `timeUnixNow()` helper is defined but never used

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:200-206`

`timeUnixNow()` is a named helper with a comment, but every call site uses
`time.Now().Unix()` inline (`importersForFiles:243`, `RiskReport:533`). Dead helper.
Harmless (package-local, unexported, not a contract), but dead code in a "minimal
implementation" step. Remove it or use it.

---

### F5 — NIT — `RiskReport` counts dependents by `to_id = o.graph_ref`, same `to_id`-resolution gap as F1

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:521-527`

```sql
COALESCE((SELECT COUNT(DISTINCT e2.file_id)
    FROM edges e2
    WHERE e2.kind = 'imports' AND e2.to_id = o.graph_ref
      AND e2.valid_from <= ? AND (e2.valid_to IS NULL OR e2.valid_to > ?)), 0)
```

`DependentCount` in the `--risk` view is keyed on `e2.to_id = o.graph_ref` — the
*same* `to_id`-resolution assumption as F1. So the `dependent_count` column in the
CLI output is also 0 against production data. Consequence of F1, listed separately
because it is a second, independent query with the same premise.

---

### F6 — NIT — `matchIndexedPath` suffix match can over-match

**File:** `skillgrid-cli/internal/mnemonic/memory/hub.go:370-378`

`SELECT path FROM files WHERE path = ? OR path LIKE '%' || ? LIMIT 1` — the `LIKE
'%' || ?` matches *any* indexed path whose tail equals the changed path. A changed
path `main.go` matches `/a/b/main.go`, `/c/d/main.go`, etc. `LIMIT 1` (no ordering)
picks an arbitrary one, which may not be the hub, causing a non-hub result for a
changed file that is actually a hub. Low real-world impact (the changed path is
usually unique enough), but the `LIMIT 1` with no `ORDER BY` is non-deterministic.
The brief's 23.2 does not require path resolution beyond "cross-reference changed
files with the hub list," so this is a robustness nit, not a spec violation.

---

## Evidence checked (the 8 checks)

1. **Importer counting (23.1, CRITICAL)** — Quoted the counting query
   (`hub.go:161-170`). **Verdict: the SQL is structurally correct** — it counts
   `kind='imports'`, keys the target on `to_id` (`JOIN symbols s2 ON s2.id = e.to_id`,
   `s2.file_id`), counts **`DISTINCT e.file_id`** (the importing file, not a raw edge
   count), and applies the step-10 temporal window
   (`valid_from <= now AND (valid_to IS NULL OR valid_to > now)`, matching
   `graph.go:54`). The 3+ threshold is `n >= cfg.MinImporters` with
   `MinHubImporters = 3` (`hub.go:278`, `:32`) — correct. **BUT** it requires
   `e.to_id IS NOT NULL`, which the codeindex does not satisfy for imports (see F1).
   So: *correct query, wrong data premise.* The DISTINCT-on-importing-file logic is
   right; the `to_id` join is the gap.

2. **hub_score formula (23.1)** — `hf.HubScore = float64(n) / float64(total)`
   (`hub.go:281`), where `total = SELECT COUNT(*) FROM files`
   (`totalIndexedFileCount`, `hub.go:211-217`) — i.e. **total indexed files, not
   total edges**. Ratio in [0,1]. Divide-by-zero guarded: `total <= 0` returns early
   (`hub.go:259-262`) and the division is inside `if total > 0` (`hub.go:280`).
   **Correct.** (Test asserts `5/10 = 0.5` — `hub_test.go:663`.)

3. **hub_score storage (23.1)** — `033_hub_score.sql:26` `ALTER TABLE symbols ADD
   COLUMN hub_score REAL;` — **additive, nullable, no default** (NULL until first
   pass). Stamped per file via `UPDATE symbols SET hub_score = ? WHERE file_id = ?`
   (`hub.go:291-292`), 0 for non-hub, the file's ratio for hubs. Stored on every
   symbol row of the file (file-level value, per the migration comment). Updated on
   every `IdentifyHubFiles` run (idempotent re-stamp). **Correct.**

4. **Impact analysis (23.2)** — `AnalyzeImpact` runs the hub pass, builds
   `hubByPath`, then classifies each changed file: high = hub with
   `ImporterCount >= HighImpactDependents` (5), medium = hub with fewer, low =
   non-hub (`hub.go:336-363`). `DependentCount = h.ImporterCount` for hubs, 0 for
   non-hubs. Levels match the brief (high = many dependents, medium = few, low =
   non-hub). **Correct** (test `TestAnalyzeImpactHubFiles`: hub1 6→high, hub2
   4→medium, 3 non-hubs→low — `hub_test.go:701-759`). Caveat: `DependentCount`
   inherits F1's `to_id` gap.

5. **Risk score (23.3)** — `risk_score` is the `hub_score` of the file the
   observation references (`graph_ref → symbols.file_id → hub_score`), 0 otherwise
   (`hub.go:395-402`). Stored on the observation (`033_hub_score.sql:27`). An
   observation about a hub file **does** get a higher score (test verifies 5/9 for
   hub, 0 for leaf, re-stamp to 6/10). **BUT** it is **not** a combination with
   `importance_score` (step 13) — see F2. Proportional to hub_score: yes.

6. **CLI (23.4)** — `--risk` flag added to `mem graph` (`mem.go:15`, `:25-29`);
   `runMemGraphRisk` calls `RecomputeRiskScores` then `RiskReport`
   (`mem.go:627-655`). `--threshold` (default 0.5, 0 = all), `--limit`. Output:
   JSON with `project`/`threshold`/`count`/`entries[]`. Handles no-hub-files
   gracefully: returns `count: 0, entries: []` (no error path on empty). **Correct**
   in shape; will show all-zero/empty against production data due to F1.

7. **Additive-ness / tool contract** — **No `mem_*`/`code_*`/`web_*` MCP tool
   signature changed.** The two new columns are nullable REAL with no default
   (additive; `033_hub_score.sql:26-27`). The one new index is a partial index on
   `observations` (additive, `IF NOT EXISTS`). The only Go change to an existing
   file is `service.go` adding a `hubCfg` field (additive struct field) and
   `mem.go` adding a flag + a new subcommand handler. **Contract held.**

8. **TDD discipline** — Four tests, all in `hub_test.go` + `mem_test.go`:
   (a) `TestHubFileIdentification` — 3+ threshold + hub_score 0.5 + storage on
   symbol (hub + non-hub=0) — yes.
   (b) `TestAnalyzeImpactHubFiles` — hub vs non-hub, dependent count, high/med/low —
   yes.
   (c) `TestRiskScoreOnObservations` — hub obs > leaf obs, proportional, re-stamp —
   yes.
   (d) `TestMemGraphRiskCLI` (in `mem_test.go`, 127 lines) — default threshold,
   `--threshold 0`, `--threshold 1` — yes.
   **All four sub-tasks are genuinely RED→GREEN** (the report shows `build failed`
   REDs, which is the honest shape for new symbols). Caveat: the tests seed `to_id`
   non-NULL, so they do not exercise the production import-edge shape (F1).

---

## Approved to merge: **yes** (with the two MAJOR findings recorded)

F1 and F2 are **production-realism / scope** warnings, not correctness defects in
the code as written: the SQL is internally consistent, the schema is additive, the
tool contract is untouched, and the tests honestly validate the implemented
behavior. Merge is safe. Recommend, before `sdd-archive`, either (a) recording F1
as a known limitation in the change's `design.md` (the feature is dormant until
import edges carry a resolvable `to_id`), or (b) opening a follow-up task to make
`importersForFiles`/`RiskReport` resolve import targets by `to_name`/`target_path`
the way `graph/impact.go` does. F2 should be recorded as a scope decision (risk is
hub-score-only; step 13 importance is a separate, combinable signal).
