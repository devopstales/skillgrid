# Task 19 — Directory Retrieval Report

## Status: PASS

## Commits

| SHA | Subject |
|-----|---------|
| 35d927c | feat(mnemonic): directory-level recursive retrieval with drill-down |
| 173b1dc | feat(mnemonic): preserve retrieval trajectory for debugging |
| ec53b28 | feat(mnemonic): intent analysis for query classification |
| 559c9b9 | feat(mnemonic): depth limit and caching for directory retrieval |

## Sub-task summary

### 19.1 — Directory retrieval drill-down [RED→GREEN]

**Test:** `TestDirectoryRetrievalDrillDown` (retrieval_dir_test.go)
- Seeds hierarchical observations: `project/app/core` (2 obs, "auth middleware"), `project/app/api` (1 obs), `project/docs` (1 obs)
- Verifies trajectory: `[project → project/app → project/app/core]` (3 steps, depths 1-3)
- Verifies results scoped to leaf `project/app/core`

**Implementation:** `retrieval.go` — `DirectoryRetrieve(ctx, RetrieveDirOptions) (*RetrieveDirResult, error)`
- Loop: score children of current dir → pick best → drill in → repeat until leaf or depth 5
- Scoring: FTS5 term density (matches/total_terms) + max embedding cosine similarity (if embedder attached)
- Leaf detection: `hasDirectoryChildren` (no observations under `dir/%`)

### 19.2 — Trajectory preserved [RED→GREEN]

**Test:** `TestRetrievalTrajectoryPreserved` (retrieval_trail_test.go)
- Runs retrieval → verifies in-memory trajectory (path, score, depth, path_prefix)
- Verifies `GetTrajectory(queryID)` returns identical steps
- Spot-checks `retrieval_trails` row exists
- Unknown query ID → empty slice, not error

**Implementation:**
- `recordRetrievalTrail`: INSERT into `retrieval_trails (project, query, path, scores, depth, created_at)`
- `GetTrajectory(queryID)`: SELECT path → JSON unmarshal to `[]RetrievalStep`
- Best-effort: insert failure logged, never propagated (query ID stays 0)

### 19.3 — Intent analysis + `--trajectory` CLI [RED→GREEN]

**Test:** `TestIntentAnalysisClassification` + `TestSearchTrajectoryCLI`
- Classifies: "why does the build fail"→debugging, "what files are in the auth module"→exploration, "review the changes to payment.go"→review, "how should I split this function"→refactor
- CLI test: `mem search auth middleware --trajectory` → output includes `"trajectory"`, `"query_id"`, drill-down paths, `"depth": 1`

**Implementation:**
- `ClassifyIntent(query) Intent`: keyword matching, checked in order (debugging→review→refactor→exploration)
- `intentParams(intent)`: debugging → wider fanout (40), others → 20
- CLI: `--trajectory` flag on `mem search` → calls `svc.DirectoryRetrieval()` → appends `"trajectory"` to JSON output
- Service wrapper: `Service.DirectoryRetrieval(ctx, projectID, configRoot, query, limit)` in service/retrieval.go

### 19.4 — Depth limit + caching [AFK]

**Test:** `TestDirectoryRetrievalDepthLimit` + `TestDirectoryRetrievalCache`
- 10-level hierarchy (a/b/c/d/e/f/g/h/i/j) → stops at depth 5, leaf = `a/b/c/d/e`
- Cache: second identical retrieval returns same leaf + trajectory (scores memoized)

**Implementation:**
- `retrievalMaxDepth = 5` enforced in the drill-down `for depth := 1; depth <= retrievalMaxDepth` loop
- `dirScoreCache` (map, mutex-guarded, 256-entry cap with full-flush eviction)
- Cache key: `query|root|matchMode`; stores `map[string]float64` (dir→score)

## Directory hierarchy choice

**Field used:** `topic_key` (existing column on `observations`, no schema change needed)

**Why:** `topic_key` is already a slash-separated path (e.g. `project/app/core`). The hierarchy is derived by splitting on `/`. The `scope` column is a flat visibility tag (`project|user|global`), not a directory path. No new `scope_path` column needed — `topic_key` already carries the hierarchical structure.

## Scoring (FTS5 + embedding)

Per directory, score = `ftsScore + max(0, vecScore)`:
- **ftsScore**: count of query terms found in concatenated title+content of the directory's observations, divided by total query terms. This is term-density (not FTS5 bm25 — bm25 requires a full FTS5 query parse; term density is cheaper and sufficient for directory-level ranking).
- **vecScore**: max cosine similarity between the query vector and any observation's stored embedding in that directory. 0 when no embedder is configured or no embeddings exist.

The embedding seam is `SetDirEmbedder` (optional, nil = FTS5-only). It adapts any `EmbedQuery(ctx, string) (Vector, error)` provider.

## Drill-down logic

```
current = scope (or "" for full hierarchy)
for depth 1..5:
    scores = scoreDirectories(current)  // children of current
    if no scores: break  // nothing matches, return current
    best = pickBestDirectory(scores)     // highest score, tiebreak by path
    record trajectory step (best, score, depth, path_prefix)
    current = best
    if no children under current: break  // leaf level
leaf = current
results = observations in leaf, ranked by same scoring
```

## Migration (029_retrieval_trails.sql)

```sql
ALTER TABLE retrieval_trails ADD COLUMN path TEXT;
ALTER TABLE retrieval_trails ADD COLUMN scores TEXT;
ALTER TABLE retrieval_trails ADD COLUMN depth INTEGER;
```

**Note:** `retrieval_trails` already existed from migration 010 (tiered context) with columns `id, project, query, directories_json, files_json, result_path, corpus, created_at`. Migration 029 adds 3 columns additively. The 010 `insertTrail` code path is unaffected (new columns read NULL for legacy rows).

## Test evidence

### Focused tests
```
$ go test ./internal/mnemonic/memory/ -run 'TestDirectoryRetrievalDrillDown|TestRetrievalTrajectoryPreserved|TestDirectoryRetrievalDepthLimit|TestIntentAnalysisClassification|TestDirectoryRetrievalCache' -count=1 -v
--- PASS: TestDirectoryRetrievalDrillDown (0.27s)
--- PASS: TestDirectoryRetrievalDepthLimit (0.35s)
--- PASS: TestDirectoryRetrievalCache (0.22s)
--- PASS: TestIntentAnalysisClassification (0.00s)
--- PASS: TestRetrievalTrajectoryPreserved (0.17s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  2.204s
```

### Full memory suite
```
$ go test ./internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  36.647s
```

### CLI suite
```
$ go test ./cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  96.828s
```

### Baseline suites
```
$ go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store       5.779s
FAIL github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service    58.110s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  19.751s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route       5.818s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected   15.389s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community   2.541s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg         3.987s
```

**Pre-existing failure:** `TestReindexStructuralIsEmbedderFree` in `service` package fails with "sql: database is closed" — verified pre-existing on parent commit `f47c6b5` via git worktree. Not caused by step 19.

## Concerns

1. **`retrieval_trails` table collision**: Migration 010 already created `retrieval_trails` for a different purpose (tiered context file-path trails). Step 19 reuses the same table for directory-retrieval trajectories. The two use cases are distinguishable by which columns are populated (010 rows have `directories_json`/`files_json` set; 19 rows have `path`/`scores`/`depth` set), but a future query on this table must be aware of both schemas.

2. **Store pooling cache**: The `store.Open` pooling cache (keyed by dbPath) can serve a stale `*sql.DB` across tests if `t.TempDir()` paths are recycled. This caused confusion during development but is a pre-existing infrastructure concern, not a step-19 bug. The `SKILLGRID_MNEMONIC_DISABLE_CACHE=1` env var bypasses it.

3. **`dirEmb` field on Service**: Added as an unexported field on `memory.Service` (not on the store). It is set via `SetDirEmbedder` and defaults to nil (FTS5-only). The production wiring (config → `SetDirEmbedder`) is not yet done — the embedder is only active when a test or future code calls `SetDirEmbedder` explicitly.
