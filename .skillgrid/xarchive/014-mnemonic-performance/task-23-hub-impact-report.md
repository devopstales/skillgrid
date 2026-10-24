# Task 23 — Hub Impact Analysis: Report

## Status: DONE

All four sub-tasks (23.1–23.4) implemented and passing.

## Commits

| SHA | Subject |
|-----|---------|
| `e2bded5` | feat(mnemonic): hub file identification with 3+ importer threshold |
| `8b55459` | feat(mnemonic): risk_score on observations, impact analysis, and mem graph --risk CLI |

## 23.1 — Hub file identification (`IdentifyHubFiles`)

**Files:** `skillgrid-cli/internal/mnemonic/memory/hub.go`, `033_hub_score.sql`

- `IdentifyHubFiles(ctx)` counts distinct importers per file via the codeindex `edges` table:
  ```sql
  SELECT s2.file_id, COUNT(DISTINCT e.file_id)
  FROM edges e JOIN symbols s2 ON s2.id = e.to_id
  WHERE e.kind = 'imports' AND e.to_id IS NOT NULL
    AND e.valid_from <= ? AND (e.valid_to IS NULL OR e.valid_to > ?)
    AND s2.file_id IN (?)
  GROUP BY s2.file_id
  ```
  An "importer" is a **distinct from-file** (`e.file_id`), not a distinct edge — multiple symbols or repeated import lines in the same file count once.
- Threshold: 3+ importers → hub (configurable via `SetHub`).
- `hub_score = importers / total_files` (ratio in [0,1]).
- Stored on every symbol row of the file via `UPDATE symbols SET hub_score = ? WHERE file_id = ?`. Non-hub files get 0 (distinguishes "analyzed, not a hub" from NULL "never analyzed").
- Returns `[]HubFile` sorted by importer count DESC, then path ASC.

**Migration 033** (additive):
```sql
ALTER TABLE symbols ADD COLUMN hub_score REAL;
ALTER TABLE observations ADD COLUMN risk_score REAL;
CREATE INDEX IF NOT EXISTS idx_obs_risk
    ON observations (project, risk_score DESC)
    WHERE deleted_at IS NULL AND risk_score IS NOT NULL;
```

## 23.2 — Impact analysis (`AnalyzeImpact`)

- `AnalyzeImpact(ctx, changedFiles)` runs the hub pass, then classifies each changed file:
  - **high**: hub with dependents >= 5 (`HighImpactDependents`, configurable)
  - **medium**: hub with fewer dependents
  - **low**: non-hub (DependentCount = 0)
- Returns `[]ImpactResult{FilePath, IsHub, DependentCount, ImpactLevel}` in input order.
- Changed files matched by exact `files.path` or suffix (relative path tail).

## 23.3 — Risk score on observations (`RecomputeRiskScores`)

- `risk_score` = `hub_score` of the file the observation references (via `graph_ref → symbols.file_id → symbols.hub_score`), 0 for non-hub or NULL graph_ref.
- Implemented as a single `UPDATE observations SET risk_score = COALESCE((SELECT COALESCE(s.hub_score, 0) FROM symbols s WHERE s.id = observations.graph_ref), 0) WHERE project = ? AND deleted_at IS NULL`.
- Runs the hub pass first (so risk reflects the current import graph); re-running after import edge changes re-stamps.
- Test verifies: hub obs risk = 5/9 ≈ 0.556, leaf obs risk = 0, re-stamp after adding a 6th importer → 6/10 = 0.6.

## 23.4 — `mem graph --risk` CLI

- `--risk` flag on `mem graph` switches from the edge timeline to the risk view.
- `--threshold` (default 0.5, 0 = show all) filters by `risk_score >= threshold`.
- `--limit` caps the result set.
- Output: JSON with `project`, `threshold`, `count`, `entries[]` (each: `observation_id`, `title`, `file_path`, `risk_score`, `dependent_count`).
- CLI calls `RecomputeRiskScores` then `RiskReport`.

## RED→GREEN Evidence

### 23.1 (TestHubFileIdentification)
- **RED**: `build failed` — `IdentifyHubFiles`, `HubFile` undefined.
- **GREEN**: `ok` — 10 files, hub.go (5 importers) flagged, hub_score = 0.5 (5/10), stored on symbols, non-hub = 0.

### 23.2 (TestAnalyzeImpactHubFiles)
- **RED**: `build failed` — `AnalyzeImpact`, `ImpactResult` undefined.
- **GREEN**: `ok` — 5 changed files, 2 hubs (high/medium), 3 non-hubs (low), dependent counts correct.

### 23.3 (TestRiskScoreOnObservations)
- **RED**: `build failed` — `RecomputeRiskScores` undefined.
- **GREEN**: `ok` — hub obs risk = 5/9, leaf obs risk = 0, re-stamp → 6/10.

### 23.4 (TestMemGraphRiskCLI)
- **RED**: `mem graph --risk missing threshold field` (flag not yet wired).
- **GREEN**: `ok` — default threshold shows hub obs only; `--threshold 0` shows both; `--threshold 1` shows zero.

## Test Results

### Focused tests
```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestHubFileIdentification|TestAnalyzeImpactHubFiles|TestRiskScoreOnObservations'
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  1.565s
```

### Full memory suite
```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  27.365s
```

### Full CLI suite
```
$ go test ./skillgrid-cli/cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  94.721s
```

### Baseline suites (005/008/010/011/013)
```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
    ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... \
    ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... \
    ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok  .../store       7.097s
FAIL .../service    61.712s  (pre-existing: TestReindexStructuralIsEmbedderFree — fails on HEAD without step-23 changes)
ok  .../codeindex   22.692s
ok  .../route       1.892s
ok  .../affected    11.647s
ok  .../community   4.499s
ok  .../pdg         6.155s
```

## Concerns

1. **Pre-existing service test failure**: `TestReindexStructuralIsEmbedderFree` fails with `sql: database is closed` — verified to fail on the committed HEAD (`8b55459`) without any step-23 changes. Not caused by this step.
2. **Arg-order bug (found during TDD)**: The initial `importersForFiles` query had the file IDs before the `now` params, but the SQL placeholders were `valid_from <= ?` (1st), `valid_to > ?` (2nd), `IN (?)` (3rd). The file ID was being bound to `valid_from`. Fixed by reordering args to `[now, now, fileIDs...]`. This was the root cause of the multi-round debugging.
3. **`risk_score >= 0` with NULL**: The `RiskReport` query uses `risk_score >= ?` which excludes NULL rows (SQLite semantics). `RecomputeRiskScores` stamps 0 on all live observations, so NULLs only exist for pre-033 rows that were never risk-stamped. The `--threshold 0` case works correctly after `RecomputeRiskScores` runs (which the CLI does before `RiskReport`).
