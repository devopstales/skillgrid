# Step 14 — Explicit Relations (`@relation` annotations between observations)

**Status: PASS** (one pre-existing, unrelated service-test failure — see Concerns)

## What was implemented per sub-task

### 14.1 — Relation edges store source_id, target_id, relation_type, confidence

- Created the `observation_relations` table via **migration `026_observation_relations.sql`** (the next free number; 025 = step-13 importance). It lives in the same per-project SQLite memory store (step-07 confirmed single store). It is **distinct** from `memory_relations` (006, the `mem_judge` verdict table) and from the codeindex `edges` table (symbol-level graph edges): this is a typed, confidence-weighted relation between **observation** ids.
- Created `skillgrid-cli/internal/mnemonic/memory/observation_relations.go` with:
  - `ObservationRelation` struct (`SourceID`, `TargetID`, `RelationType`, `Confidence`).
  - `AddRelation(ctx, sourceID, targetID int64, relationType string, confidence float64)` — validates the 5 canonical types + the `[0.0, 1.0]` confidence range + that both observation ids are live in the project, then upserts (the composite PK makes `(source,target,type)` unique, so re-assertion updates confidence in place).
  - `GetRelations(ctx, obsID int64, minConfidence float64)` — **bidirectional** (`WHERE source_id = ? OR target_id = ?`) with a `WHERE confidence >= ?` floor.
  - `DeleteRelation(ctx, sourceID, targetID int64, relationType string)` — deletes the exact triple; returns whether a row was removed.
  - `ObsTitle(ctx, id int64)` — fetches a live observation's title for CLI display.

> **Naming note:** the brief names the delete method `RemoveRelation`, but `memory.Service` **already** has a `RemoveRelation(ctx, src, dst int64, relation string) (bool, error)` on the 006 `memory_relations` table (used by `mem_judge`/`mem_compare` at `http/server.go:1017` and `mcp/tools_memory.go:982`). Go has no overloading, so the new method is named **`DeleteRelation`** to avoid the clash. Its behavior (delete a single typed edge by full triple) matches the brief's `RemoveRelation` intent.

**The 5 canonical relation types:** `mentions`, `depends_on`, `contradicts`, `supports`, `references` — enforced by a `canonicalRelationTypes` map in `AddRelation`; unknown types are rejected with a clear error listing the valid set.

### 14.2 — `mem relations` CLI returns all related observations with relation types

- Added a `relations` subcommand to the **manual flag/command dispatcher** in `skillgrid-cli/cmd/skillgrid/mem.go` (follows the existing `switch cmd` + `flag.FlagSet` pattern — no cobra).
- `runMemRelations` queries **both outgoing and incoming** edges via `GetRelations` (which already uses `source_id = ? OR target_id = ?`) and formats each as a readable JSON object with:
  - `direction` (`outgoing` when the anchor is the source, `incoming` when it is the target),
  - `relation_type`, `confidence`, `other_id`, and `other_title` (the related observation's title, via `ObsTitle`).
- **Missing observation id** → clear error (`observation N not found in this project`, exit 1); **non-integer id / missing id** → exit 2.
- **No relations** → graceful empty list (`"relations": []`, `"count": 0`).

### 14.3 — Confidence filtering on relation queries

- `GetRelations`'s `minConfidence` parameter → `WHERE confidence >= ?` (0.0 returns all).
- Exposed via `mem relations <id> --min-confidence 0.5` (`fs.Float64Var(&minConf, "min-confidence", 0.0, ...)`), validated to `[0.0, 1.0]`.

## The `observation_relations` schema (migration quoted)

```sql
-- 026: typed @relation edges between observations (change 014, step 14).
CREATE TABLE IF NOT EXISTS observation_relations (
    source_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    relation_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type)
);

CREATE INDEX IF NOT EXISTS idx_obs_rel_source
    ON observation_relations (source_id);
CREATE INDEX IF NOT EXISTS idx_obs_rel_target
    ON observation_relations (target_id);
```

## AddRelation / GetRelations / DeleteRelation (function signatures)

```go
func (s *Service) AddRelation(ctx context.Context, sourceID, targetID int64,
    relationType string, confidence float64) (ObservationRelation, error)
// validates: 5 canonical types, confidence in [0.0,1.0], both obs live;
// upserts on the composite PK (source_id, target_id, relation_type).

func (s *Service) GetRelations(ctx context.Context, obsID int64,
    minConfidence float64) ([]ObservationRelation, error)
// bidirectional: WHERE (source_id = ? OR target_id = ?) AND confidence >= ?
// ordered by confidence DESC, then source/target/type.

func (s *Service) DeleteRelation(ctx context.Context, sourceID, targetID int64,
    relationType string) (bool, error)
// DELETE ... WHERE source_id = ? AND target_id = ? AND relation_type = ?
```

## The bidirectional CLI query

`mem relations A` resolves to (inside `GetRelations`):

```sql
SELECT source_id, target_id, relation_type, confidence
FROM observation_relations
WHERE (source_id = A OR target_id = A) AND confidence >= ?
ORDER BY confidence DESC, source_id, target_id, relation_type;
```

The CLI then labels each row `outgoing` (A is `source_id`) or `incoming` (A is `target_id`) and surfaces the *other* endpoint's id + title.

## The confidence filter

`WHERE confidence >= ?` with `minConfidence` defaulting to `0.0` (returns all). Verified in `TestRelationConfidenceFiltering` (0.3/0.7/0.95 → floor 0.5 keeps only 0.7+0.95; floor 0.0 keeps all) and in the CLI test (`--min-confidence 0.85` keeps only the 0.9 edge).

## RED → GREEN evidence

| Sub-task | RED (before impl) | GREEN (after impl) |
|----------|-------------------|--------------------|
| 14.1 `TestRelationEdgeCreation` | `svc.AddRelation undefined` / `svc.GetRelations undefined` (build failed) | `--- PASS: TestRelationEdgeCreation (0.15s)` |
| 14.3 `TestRelationConfidenceFiltering` | same build failure (methods undefined) | `--- PASS: TestRelationConfidenceFiltering (0.14s)` |
| 14.2 `TestMemRelationsCLI` | `mem.AddRelation undefined` (build failed) | `--- PASS: TestMemRelationsCLI (10.46s)` |

Test commits (RED) landed before the implementation commits (GREEN), per TDD discipline.

## Full relevant suite + output

Focused (task 14):
```
go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestRelationEdgeCreation|TestRelationConfidenceFiltering'
--- PASS: TestRelationEdgeCreation (0.15s)
--- PASS: TestRelationConfidenceFiltering (0.14s)
PASS

go test ./skillgrid-cli/cmd/skillgrid/ -run 'TestMemRelationsCLI'
--- PASS: TestMemRelationsCLI (10.46s)
PASS
```

Full packages (change 014 affected + baseline):
```
go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  16.293s

go test ./skillgrid-cli/cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  59.098s

go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
        ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... \
        ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... \
        ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok  .../internal/mnemonic/store       5.693s
FAIL .../internal/mnemonic/service   49.649s   <-- TestReindexStructuralIsEmbedderFree (pre-existing, see Concerns)
ok  .../internal/mnemonic/codeindex  19.554s
ok  .../internal/mnemonic/route      3.356s
ok  .../internal/mnemonic/affected  15.742s
ok  .../internal/mnemonic/community  2.745s
ok  .../internal/mnemonic/pdg        7.245s
```

## Commits (this step, oldest → newest)

| SHA | Subject |
|-----|---------|
| `91ef284` | test(mnemonic): add failing tests for typed observation relations |
| `5082ae5` | test(mnemonic): add failing test for mem relations CLI |
| `67ae959` | feat(mnemonic): add typed relation edges between observations |
| `9c3dcbb` | feat(mnemonic): add mem relations CLI subcommand |

(Working tree stays on `feat/v2-structure`.)

## Concerns

1. **Pre-existing, unrelated failure:** `TestReindexStructuralIsEmbedderFree` (service package) fails with `sql: database is closed` at `reindex_structural_test.go:107`. I confirmed it is **not** caused by step 14: it fails identically at the base commit `7d07632` (step 13, before any of my changes) in a clean `git worktree`. It is a codeindex re-index/store-handle lifecycle test, unrelated to the additive `observation_relations` table. It is a separate bug in the service package's store open/close handling, and is carried into this step's baseline. Flagging for a follow-up; it does not touch the `mem_*`/`code_*`/`web_*` contracts.
2. **Method rename:** brief's `RemoveRelation` is implemented as `DeleteRelation` to avoid clashing with the existing `memory_relations.RemoveRelation` (006). Documented in the code and above.
3. **Global constraints held:** additive only (new table + new methods + new CLI subcommand); no existing schema rewritten; no `mem_*`/`code_*`/`web_*` tool contract shape changed; `modernc.org/sqlite` driver retained (no CGO).
