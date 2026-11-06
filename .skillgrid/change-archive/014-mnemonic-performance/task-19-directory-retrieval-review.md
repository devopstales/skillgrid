# Task 19 — Directory Retrieval Review

**Verdict**: `PASS WITH WARNINGS`
**Approved to merge**: yes

Scope: `retrieval.go` (new) + `retrieval_dir_test.go` + `retrieval_trail_test.go` + `029_retrieval_trails.sql` (new) + `mem.go` + `mem_search_trajectory_test.go`, 4 commits (35d927c, 173b1dc, ec53b28, 559c9b9). Tests re-run green by this reviewer:
`go test ./internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown|TestRetrievalTrajectoryPreserved|TestDirectoryRetrievalDepthLimit|TestIntentAnalysisClassification|TestDirectoryRetrievalCache' -count=1` → ok; `go test ./cmd/skillgrid/ -run TestSearchTrajectoryCLI -count=1` → ok.

## Findings

### 1. MAJOR — `--trajectory` opens the store a second time (separate `*sql.DB`) and never calls `SetDirEmbedder`

`cmd/skillgrid/mem.go:339` runs `svc.DirectoryRetrieval(...)` which (`service/retrieval.go:215`) does `s.openProject(projectID, root)` and then `h.Memory().DirectoryRetrieve(...)` (`service/retrieval.go:220`) — a fresh store open and a fresh `memory.Service` distinct from the one used by the flat search (`BudgetedRetrievalAsRootFTS`, which opens via `store.Open` in `service/retrieval.go:244`). Consequences:

- The production embedder is attached to the flat-search `memory.Service` only. The new `dirEmb` field (`service.go:106`) is set exclusively via `SetDirEmbedder`, and **no production call site exists** (grep: only `retrieval.go` self-references and tests). So in production, `DirectoryRetrieve` is FTS5-only; the embedding leg of the score is dead code in every real invocation, while the flat search beside it does embed. The report admits this ("production wiring … not yet done"), but it means the "FTS5 + embedding similarity" claim for the CLI flag is currently false at runtime.
- Two concurrent `*sql.DB` handles on the same SQLite file (one per leg of `mem search --trajectory`). Modernc WAL makes this safe in practice, but it is a second open/parse cycle and a second connection the CLI did not need.

Fix (minimal): have `DirectoryRetrieval` accept the already-opened `memory.Service` (or have the CLI call the flat-search `memory.Service` directly, reusing the same `store.Open` handle), and attach the existing `embedder.Default()` via `SetDirEmbedder` there. Or, short term, document in `DirectoryRetrieval`'s doc comment that the embedder is unwired in production and file a follow-up task.

### 2. MAJOR — `dirScoreCache` is a process-global keyed without project

`retrieval.go:152-159`:

```go
var (
	dirScoreCacheMu sync.Mutex
	dirScoreCache   = map[string]map[string]float64{}
)

func dirCacheKey(query, root, matchMode string) string {
	return query + "|" + root + "|" + matchMode
}
```

The key has no project dimension, yet scores are computed against `s.projectID` (`retrieval.go:270`). The CLI opens many projects in one process run (multi-project sessions, `mem search` across `--project` values): two projects sharing a query + root would read each other's memoized directory scores — wrong ranking, silently, with no error. The comment at `retrieval.go:148-151` says a stale entry "only changes ranking freshness, never correctness" — that holds for one project, not across projects. Fix: include `s.projectID` in `dirCacheKey` (or make the cache per-`Service` field instead of a package global).

### 3. MINOR — `scopeWider` is dead

`retrieval.go:87-94` returns two values; `retrieval.go:204-205` discards the first:

```go
intent := ClassifyIntent(query)
_, fanout := intentParams(intent)
```

`intentParams` documents "debugging → wider scope" but the only observable effect is the fanout change (40 vs 20, which only sets `LIMIT ?` = `fanout*4` in `scoreDirectories`, `retrieval.go:279`). The "wider scope" semantics are not actually implemented — no scope widening happens. Either implement it or drop the flag.

### 4. MINOR — no test proves the intent actually changes retrieval parameters

`TestIntentAnalysisClassification` (`retrieval_dir_test.go:177-193`) only checks `ClassifyIntent` label mapping. The brief (19.3.a) asks to "verify the intent classification affects the retrieval strategy (debugging → wider search, exploration → directory-first)". `intentParams` is untested directly, and no test shows the same query with different intent yields different fanout. Add a case asserting `intentParams(IntentDebugging)` returns a larger fanout than `intentParams(IntentExploration)`.

### 5. MINOR — brief's migration table diverges from what shipped; `GetTrajectory` is unscoped

Brief 19.2.c specifies `retrieval_trails` as a NEW table (`id, query, path, scores, depth, created_at`). What shipped: the table already existed from migration 010 (tiered context: `id, project, query, directories_json, files_json, result_path, corpus, created_at`); `029` only `ALTER TABLE … ADD COLUMN path/scores/depth` (all nullable, no `IF NOT EXISTS`, not idempotent on a double-apply). This is a legitimate additive extension, but:

- `GetTrajectory` (`retrieval.go:552-571`) does `SELECT path FROM retrieval_trails WHERE id = ?` with no `project` filter. 010 rows have `path` NULL, so `SELECT` returns a zero row with `pathJSON` NULL → `json.Unmarshal(nil, &steps)` errors → "decode retrieval trajectory" error for a legitimate (010-era) id, instead of the documented "empty slice, not an error". Add `AND project = ?` and treat NULL `path` as empty.
- Two different trail schemas now share one table, distinguished only by which columns are populated (the report's own "Concern 1"). A future reader of `retrieval_trails` must know both. Consider a `kind` column or a separate table; at minimum, document the dual schema at the table definition.

### 6. MINOR — `LIKE 'root/%'` is glob-unsafe for topic_keys containing `%` or `_`

`retrieval.go:272-273` and `:461` use `topic_key LIKE ?` with `root+"/%"`. SQLite `LIKE` treats `%` and `_` as wildcards; a topic_key segment like `q2_plan` contains `_`, which matches any single character — `LIKE 'q2_plan/%'` would also match `q2xplan/core`. No escaping (`ESCAPE` clause) is applied. Correct for well-formed keys, silently wrong for keys with glob metacharacters. Fix: escape `%`/`_` in the literal, or switch to `topic_key >= ? AND topic_key < ?` range queries.

### 7. MINOR — "FTS5" scoring is term containment, not the FTS5 index

`scoreDirectoryRows` (`retrieval.go:327-344`) does `strings.Contains` over `strings.Fields(query)` terms, divided by term count — a substring term-density heuristic. It never touches the `observations_fts` FTS5 table (nor bm25). The brief and doc comments call this "FTS5 relevance"; the report itself concedes "not FTS5 bm25". Substring containment also over-matches ("cat" matches "category", "auth" matches "author"). It is a defensible cheap directory-level signal, but rename/comment it as "term-density" so the FTS5 claim doesn't accumulate, and note it diverges from the flat search's actual FTS5 ranking (so a directory can be selected whose FTS5 ranking disagrees).

### 8. NIT — `matchModeFor` is a no-op placeholder

`retrieval.go:639-642` always returns `""`; it exists only to pad the cache key with a constant. Either use the real FTS match mode (which the CLI accepts via `--mode`) or drop it from the key.

### 9. NIT — `recordRetrievalTrail` swallows the id on `LastInsertId` error and shares the global `logWriter`

`retrieval.go:539-545` logs via `fmt.Fprintf(logWriter(), …)` and returns 0 on any failure — fine as best-effort, but `QueryID = 0` is indistinguishable in output between "no trail" and "trail failed", and the CLI prints no warning (it only warns on `DirectoryRetrieval` error). Cosmetic; a `trail_warning` field on `RetrieveDirResult` would make it observable.

### 10. NIT — brief/actual signature drift (accepted)

Brief 19.1.c specifies `Retrieve(ctx, query, scope) (Results, Trajectory, error)`; shipped is `DirectoryRetrieve(ctx, RetrieveDirOptions) (*RetrieveDirResult, error)` (results + trajectory + query id in one struct). This is strictly better and matches the "observable trajectory" goal; noting only as deviation from the brief's literal interface.

## Evidence checked

1. **Hierarchy choice — CORRECT.** `scope` is a flat visibility tag (`project|user|global`, see `service.go:607-608` "scope … restricts results to that visibility scope (project|user|global)"), not a path. `topic_key` is the slash-separated path: the MCP contract documents it as such (`tools_memory.go:65` "Stable key for upserts, e.g. architecture/auth-model"), and `mem.go:787` notes "a topic_key typically contains '/'". Splitting logic is `childDirectory` (`retrieval.go:587-612`):

   ```go
   func childDirectory(root, topicKey string) string {
       tk := strings.Trim(topicKey, "/")
       if root == "" {
           if i := strings.IndexByte(tk, '/'); i >= 0 {
               return tk[:i]
           }
           // A single-segment topic_key with no root is itself a top-level
           // directory.
           return tk
       }
       root = strings.Trim(root, "/")
       if tk == root {
           return ""
       }
       if !strings.HasPrefix(tk, root+"/") {
           return ""
       }
       rest := tk[len(root)+1:]
       if i := strings.IndexByte(rest, '/'); i >= 0 {
           return root + "/" + rest
       }
       return root + "/" + rest
   }
   ```

   Empty topic_keys are excluded at the SQL level (`retrieval.go:269`: `o.topic_key IS NOT NULL AND o.topic_key != ''`); flat (slash-free) keys are treated as top-level leaves by both `childDirectory` and `hasDirectoryChildren` — verified: a flat key has no `key/%` descendants, so it is a leaf at depth 1. `scope` is correctly NOT part of the hierarchy (only `project` is used for scoping, `retrieval.go:270`).

2. **Drill-down logic — CORRECT.** The loop (`retrieval.go:218-241`):

   ```go
   for depth := 1; depth <= retrievalMaxDepth; depth++ {
       scores, err := s.scoreDirectories(ctx, query, current, qv, fanout)
       if err != nil { return nil, err }
       if len(scores) == 0 { break }
       best := pickBestDirectory(scores)
       res.Trajectory = append(res.Trajectory, RetrievalStep{
           Path: best, Score: scores[best], Depth: depth,
           PathPrefix: strings.Split(best, "/"),
       })
       current = best
       if !s.hasDirectoryChildren(ctx, current) { break }
   }
   ```

   First pass scores all top-level directories (empty `root` → children are top-level segments, `retrieval.go:589-598`); it drills into the HIGHEST-scoring one via `pickBestDirectory` (max score, path tiebreak, `retrieval.go:624-634`); repeats until leaf (`hasDirectoryChildren` false) or depth 5. The depth-limit test (`TestDirectoryRetrievalDepthLimit`, `retrieval_dir_test.go:92-142`) seeds a 10-level chain `a…j` and asserts exactly 5 trajectory steps, last at depth 5, leaf `a/b/c/d/e`. Verified: the loop is iterated (not recursive), so no stack-overflow mode exists.

3. **Scoring — COMBINED, monotonic in each leg, FTS5-leg is term-density (see finding 7).** `score = ftsScore + max(0, vecScore)` (`retrieval.go:364`). No embedder → `qv` empty → `vecScore = 0`, pure FTS path, no error. Each leg is non-decreasing in its input (more matches → higher; higher cosine → higher, floored at 0). Note the vector leg takes the MAX over the directory's observations (`retrieval.go:346-360`), not the average the comment at `retrieval.go:323-324` claims ("the vector leg averages") — minor doc/code mismatch; max is the better choice.

4. **Trajectory recording — CORRECT.** `recordRetrievalTrail` (`retrieval.go:519-547`) inserts one row per RUN with `path` = JSON of all steps (each step carries path, score, depth, path_prefix array), `scores` = JSON float array, `depth` = final depth. `GetTrajectory(queryID)` (`retrieval.go:552-571`) round-trips it; unknown id → empty, not error (tested, `retrieval_trail_test.go:80-86`). `TestRetrievalTrajectoryPreserved` verifies in-memory == persisted, per-step depth/score/path_prefix invariants, and the row exists. Caveat: the trail records one row per run, not one row per directory visit (the brief's "record each directory visit" is satisfied by the JSON array column containing every visit — acceptable, but the `depth` column is the last step's depth only).

5. **Migration 029 — ADDITIVE but not idempotent.** Three `ALTER TABLE retrieval_trails ADD COLUMN …` statements; no existing column touched, 010's `insertTrail` (`service/retrieval.go:440-443`) unaffected. The 029 columns read NULL on legacy rows (verified by the 010 test suite passing). The brief's "CREATE TABLE IF NOT EXISTS" shape was replaced by the extension of the 010 table — see finding 5.

6. **Intent analysis — CORRECT as specified.** `ClassifyIntent` (`retrieval.go:70-80`): pure keyword containment, fixed order debugging → review → refactor, default `exploration`. Recognizes `exploration`, `debugging`, `review`, `refactor` exactly as 19.3.c requires. All four brief test queries classify correctly (test re-run green). Strategy effect: fanout 40 (debugging) vs 20 (others) — see findings 3-4.

7. **`--trajectory` CLI — CORRECT.** `mem.go:334-345` adds `trajectory` to the JSON output only when the flag is set; flat search output unchanged; directory-retrieval error → stderr warning, search still succeeds. Fresh query (no trail): `QueryID` 0, empty `trajectory`/`results` arrays — handled, no crash (the flag test passes; the empty-query early return is `retrieval.go:192-194`). See finding 1 for the double-open/embedder wiring.

8. **Additive-ness / tool contract — HELD.** No `mem_*`/`code_*`/`web_*` MCP tool signature changed: the diff touches only `cmd/skillgrid/mem.go` (new flag), `service/retrieval.go` (new `DirectoryRetrieval` method), `memory/service.go` (new unexported `dirEmb` field + `SetDirEmbedder`), the new `retrieval.go`, two new test files, and migration 029. The existing search path (`BudgetedRetrievalAsRootFTS` → `SearchOwnerScopedRetrieveFTS`) is byte-identical in the diff. `DirectoryRetrieve` is a new method, not a change to existing search.

9. **TDD discipline — ALL FIVE PRESENT.** (a) drill-down: `TestDirectoryRetrievalDrillDown` asserts the exact 3-step path `project → project/app → project/app/core`, depths 1-3, positive scores, leaf, and that every result is under the leaf. (b) trajectory: `TestRetrievalTrajectoryPreserved` (in-memory + persisted + row-exists + unknown-id). (c) depth limit: `TestDirectoryRetrievalDepthLimit` (10 levels → 5 steps). (d) cache: `TestDirectoryRetrievalCache` (second run reuses memoized scores). (e) intent: `TestIntentAnalysisClassification`. CLI: `TestSearchTrajectoryCLI`. Gaps: no negative test (query matching no directory → empty trajectory, leaf = root) and intent-effect test missing (finding 4).
