# Briefing — Bi-temporal Observations + AUDN Save Classification

**Topic:** 2026-09-24-bitemporal-audn
**Date:** 2026-09-24
**Classification:** risky (T2) — schema migration + save-path contract change
**Build shape:** Tracer thread
**ADR:** `.skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md`
**ADR manifest:** `.skillgrid/specs/2026-09-24-bitemporal-audn/adr.md`
**Findings:** `.skillgrid/specs/2026-09-24-mnemon-comparison/findings.md`
**Spec:** `.skillgrid/specs/2026-09-24-mnemonic-memory-improvements/briefing.md` (parent spec — this is the bi-temporal + AUDN slice)

## Why

The memory-improvements comparison (Mnemon + mnemonic-ai) identified four
out-of-scope capabilities. This spec covers two that are tightly coupled:
bi-temporal observations and the AUDN save classifier. The delete arm of the
AUDN classifier IS the auto-supersede mechanism — without bi-temporal columns,
the delete arm has nowhere to point; without the classifier, the bi-temporal
columns are never populated.

## What's already built (do NOT re-plan)

| Capability | Where | Status |
|---|---|---|
| `status = 'superseded'` + `SetStatus` | `memory/governance.go:23-27, 213-232` | Built (manual only) |
| `supersedes` edge in `memory_relations` | `memory/relations.go:20` | Built (006) |
| `DedupLLM` seam (binary) | `memory/types.go:37-42` | Built (opt-in, test-only) |
| `runDedupCheck` (binary) | `memory/types.go:127-156` | Built |
| Save cascade (hash→LLM→topic→insert) | `memory/service.go:425-603` | Built |
| `BumpDuplicate` | `memory/lifecycle.go:104-116` | Built |
| `observation_versions` (content history) | `memory/governance.go:144-164` | Built (017) |
| AKL importance (`importance_score`, `maturity_tier`) | `memory/importance.go` | Built (014 step 13) |
| `expires_at` TTL filter pattern | `memory/service.go:697` | Built |

## Capabilities to add

### C1: Bi-temporal columns

**New columns** (migration 043, separate file `store/migrations/043_bitemporal.sql`):
```sql
-- ===== 043_bitemporal.sql =====
ALTER TABLE observations ADD COLUMN valid_at TEXT;
ALTER TABLE observations ADD COLUMN invalid_at TEXT;
ALTER TABLE observations ADD COLUMN superseded_by INTEGER;

CREATE INDEX IF NOT EXISTS idx_obs_invalid_at
    ON observations (project, invalid_at)
    WHERE invalid_at IS NOT NULL AND deleted_at IS NULL;
```

- `valid_at` — when the fact became true. New rows: `= created_at`.
- `invalid_at` — when the fact stopped being true. NULL = still true.
- `superseded_by` — back-pointer to the observation that supersededed this one. NULL = not superseded. Stored in this spec for the four-edge traversal spec (next change) which will walk the chain backward. Not consumed by any query in this spec.

**Observation struct** (service.go:332-377): add 3 fields:
- `ValidAt string` (omitempty)
- `InvalidAt string` (omitempty)
- `SupersededBy sql.NullInt64` (omitempty)

**obsSelectCols** (service.go:2058-2063): add `o.valid_at, o.invalid_at, o.superseded_by`.

**scanObservations** (service.go:2069-2078): add 3 scan vars.

**4 inline FTS/admin SELECTs** (must add 3 columns to each — they don't use
`obsSelectCols`):
- service.go:689 (`SearchWithScope`)
- service.go:748 (`SearchOwnerScoped`)
- governance.go:446 (`SearchOwner`)
- governance.go:492 (`AdminCrossOwnerList`)

**Save INSERT** (service.go:571-579): add 3 columns:
`valid_at = created_at, invalid_at = NULL, superseded_by = NULL`.

**Bi-temporal filter** on 7 primary read sites:
```sql
AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', 'now'))
```
Sites: service.go:696, 755, 792, 827, 847; governance.go:453, 498.

**`MarkSuperseded`** (new, lifecycle.go):
```go
func (s *Service) MarkSuperseded(ctx context.Context, id int64, supersededByID int64) {
    now := time.Now().UTC().Format(time.RFC3339)
    _, _ = s.store.DB.ExecContext(ctx, `
        UPDATE observations
        SET invalid_at = ?, superseded_by = ?, status = 'superseded'
        WHERE id = ? AND project = ? AND deleted_at IS NULL`,
        now, supersededByID, id, s.projectID,
    )
}
```

### C1.5: Time-travel query

**New method** `ValidAtTime` (service.go):
```go
func (s *Service) ValidAtTime(ctx context.Context, query string, atTime string) ([]Observation, error)
```

SQL:
```sql
SELECT ... FROM observations o
INNER JOIN observations_fts ON observations_fts.rowid = o.id
WHERE observations_fts MATCH ?
  AND o.deleted_at IS NULL
  AND o.project = ?
  AND strftime('%s', o.valid_at) <= strftime('%s', ?)
  AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', ?))
ORDER BY bm25(observations_fts)
LIMIT ?
```

This is the "what was true at time T?" query. It reuses the FTS5 match but
replaces the "now" filter with an explicit time parameter. Superseded
observations whose validity window ended before T are excluded; observations
that were valid at T (even if later superseded) are included.

Not exposed as an MCP tool in this spec (the search surface is the existing
`mem_search`; time-travel is a service-layer method available to the HTTP API
and future MCP tools). The acceptance test calls it directly.

### C2: AUDN save classifier

**Extend `DedupLLM`** (types.go:37-42):
```go
type DedupVerdict string
const (
    VerdictAdd    DedupVerdict = "add"
    VerdictUpdate DedupVerdict = "update"
    VerdictDelete DedupVerdict = "delete"
    VerdictNoop   DedupVerdict = "noop"
)

type DedupDecision struct {
    Verdict    DedupVerdict
    TargetID   int64
    Confidence float64
    Reason     string
}

type DedupLLM interface {
    Dedup(ctx context.Context, newContent string, candidates []string) (duplicate bool, duplicateID int64, err error) // deprecated
    Classify(ctx context.Context, newContent string, candidates []string) (DedupDecision, error)
}
```

**Two call sites** for the `DedupLLM` seam:
1. `runDedupCheck` (types.go:127-156) — the Save pre-write path. **Switches to `Classify`.** Return `(DedupDecision, string)` instead of `(int64, string)`.
2. `isExtractedDuplicate` (types.go:288) — the async extraction dedup in `asyncExtractAndDiff` (types.go:246-277), which writes `memory_diff.json`. **Stays on `Dedup`** (binary). The extraction path only needs "is this a dup?" — it doesn't need the 4-way verdict. Stays binary, no behavior change.

**`newDedupLLMBackend()`** — new constructor in `service/service.go` (following the `ExtractionLLM` backend pattern at service.go:339-345). Config key: `mnemonic.dedup.llm` (default `false`). When enabled, the backend wraps the existing extraction LLM client (shared provider, different prompt).

**New `SaveWithAction`** (service.go):
```go
type SaveResult struct {
    ID           int64  `json:"id"`
    Action       string `json:"action,omitempty"`
    SupersededID int64  `json:"superseded_id,omitempty"`
}

func (s *Service) SaveWithAction(ctx context.Context, in SaveInput) (SaveResult, error)
```

4-way routing in `SaveWithAction` (at the decision point, service.go:501):

| Verdict | Action | Sink |
|---|---|---|
| `noop` | `BumpDuplicate(targetID)` + return `{ID: targetID, Action: "noop"}` | Existing (line 502) |
| `add` | INSERT new row + return `{ID: newID, Action: "add"}` | Existing (line 571) |
| `update` | Topic-key upsert if topic_key set; else new upsert-by-id branch. Return `{ID: targetID, Action: "update"}` | Partial (lines 506-543) |
| `delete` | `MarkSuperseded(targetID, newID)` + add `supersedes` edge + INSERT new row. Return `{ID: newID, Action: "delete", SupersededID: targetID}` | **New sink** |

**`Save` delegates to `SaveWithAction`:**
```go
func (s *Service) Save(ctx context.Context, in SaveInput) (int64, error) {
    result, err := s.SaveWithAction(ctx, in)
    return result.ID, err
}
```
Old callers (`mcp/tools_memory.go:244`) keep working.

**`handleMemSave`** (tools_memory.go:181-265): call `SaveWithAction` instead
of `Save`. Add `action` + `superseded_id` to the response map:
```go
out := map[string]any{"id": result.ID, "project": projectID}
if result.Action != "" {
    out["action"] = result.Action
}
if result.SupersededID > 0 {
    out["superseded_id"] = result.SupersededID
}
```

**Deterministic floor** (no LLM / LLM error):
- Hash hit → `noop` (call `BumpDuplicate`: increment `duplicate_count`, refresh `last_seen_at`, return existing ID — same behavior as today's hash path, service.go:487-491)
- Hash miss → `add` (insert new row)
- No `update` or `delete` without the LLM
- LLM error is non-fatal: log a warning, fall through to the deterministic floor

**Config wiring** (service/service.go, following the extraction-LLM pattern at
line 339-345):
```go
if cfg.Dedup.LLM {
    mem.SetDedupLLM(newDedupLLMBackend())
    mem.EnableDedupLLM(true)
}
```
Config key: `mnemonic.dedup.llm` (default `false`).

## One-way-door decisions

| # | Decision | Risk | Mitigation |
|---|---|---|---|
| 1 | Migration 043: 3 new columns on `observations` | Additive, nullable. No data migration. | `ALTER TABLE ADD COLUMN` pattern (separate file, matching 040-042). Existing rows read back with NULLs. |
| 2 | `DedupLLM` interface change (binary → 4-way) | Existing test mocks break. | Keep `Dedup` as deprecated wrapper. Add `Classify` as new method. Existing implementations keep compiling. |
| 3 | `mem_save` MCP response shape (adds `action`, `superseded_id`) | Additive fields. Old consumers ignore unknown fields. | Purely additive JSON. No existing field removed or renamed. |

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| MCP tool contract | Applicable: `mem_save` response shape changes | Additive fields only (`action`, `superseded_id`). No existing field removed. | "mem_save MCP response is additive" scenario |
| Schema migration | Applicable: 3 new columns on `observations` | Additive, nullable. No data migration. `IF NOT EXISTS` on index. | Migration idempotency test (run twice, no error) |
| Save path | Applicable: new `SaveWithAction` method; `Save` return type unchanged | `Save` delegates to `SaveWithAction`, extracts `.ID`. Old callers unaffected. | "Existing Save callers are unaffected" scenario |
| Search visibility | Applicable: bi-temporal filter changes which rows are visible | Filter is additive (`AND` clause). Superseded rows excluded from search. | "Superseded observations are excluded from search" scenario |
| No new trust boundary | N/A: all changes within the existing per-project SQLite store | — | — |

## Success criteria

1. New observations have `valid_at = created_at`, `invalid_at = NULL`, `superseded_by = NULL`.
2. Superseded observations (`invalid_at` set) do not appear in search results.
3. Time-travel query returns the fact valid at time T.
4. `mem_save` returns `action` label (add/update/delete/noop) + `superseded_id` when applicable.
5. Delete arm produces the full supersede chain (`invalid_at`, `superseded_by`, `status`, `supersedes` edge).
6. Deterministic floor: no LLM → hash-hit = noop, hash-miss = add. LLM error → non-fatal, degrades to floor.
7. Existing `Save` callers are unaffected (return type `(int64, error)` preserved).
8. All existing tests pass (no regression).
9. New tests cover: bi-temporal columns, supersede chain, time-travel query, AUDN 4-way routing, deterministic floor, LLM error degradation, MCP response additive.

## Token budget (delta on session-inject spec)

The session-inject L1 auto-prepend gets a `token_budget` parameter (default
800 tokens). Quality ranking: pinned first, then `importance_score ×
recency_factor` (reuses AKL columns from 014 step 13). Deterministic
rendering (byte-identical across calls, KV-cache friendly). The
`mem_inject_session` tool also accepts `token_budget`.

This is a delta on the session-inject spec (separate change). No new columns,
no new migration. Listed here for completeness; no tasks in this spec.

## Out of scope (next spec)

- **Four-edge extension** (temporal/entity/causal/semantic in
  `observation_relations`) + depth-capped BFS traversal + intent-adaptive
  weights. This is the second spec (Approach B). Needs its own briefing,
  blueprint, and ADR.
- **13-language intent detection** — stretch goal.

## Clarity Report

| Dimension | Score | Status |
|---|---|---|
| Problem clarity | 0.90 | ✅ — the failure class is concrete (superseded facts rank in search, agent can't answer "what was true when?") |
| Solution clarity | 0.85 | ✅ — the 4-way routing table is explicit, the sinks are named, the migration is defined |
| Scope clarity | 0.90 | ✅ — token budget is a delta (separate spec), four-edge is out of scope (next spec) |
| Constraint clarity | 0.95 | ✅ — ADR-0011 locks the schema + classifier; locked constraints checked (#2 satisfied, #7/#9 not implicated) |
| **Overall** | **0.90** | ✅ — clarity ≤ 0.20 (on the 0-1 ambiguity scale, 0.10) |

## Context

**Fits existing patterns:** yes.

The bi-temporal columns follow the exact `ALTER TABLE ADD COLUMN` pattern
(027 provenance, 028 memory_type). The `MarkSuperseded` method follows the
`BumpDuplicate` pattern (lifecycle.go:104). The bi-temporal filter follows the
`expires_at` strftime idiom (service.go:697). The `SaveWithAction` method
follows the additive-new-method pattern (old `Save` delegates, old callers
untouched). The `DedupLLM.Classify` method follows the `ExtractionLLM` seam
pattern (opt-in, nil-default, deterministic floor).

One note: the 4 inline FTS/admin SELECTs duplicate the column list instead of
using `obsSelectCols`. The migration task must explicitly add the 3 columns to
all 5 SELECT sites (obsSelectCols + 4 inline).
