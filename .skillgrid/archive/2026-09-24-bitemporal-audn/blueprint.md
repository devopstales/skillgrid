# Bi-temporal Observations + AUDN Save Classification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2 (from briefing `Classification: risky (T2)` — schema migration + save-path contract change; verification floor L3: full suite + build + coverage + lint)

**Build shape:** Tracer thread (briefing's recorded shape — the end-to-end path migration → columns → save writes → search filters → time-travel is the tracer; it proves the temporal window is queryable before the AUDN classifier thickens the save path)

**Goal:** Give observations a bi-temporal validity window (`valid_at`/`invalid_at`/`superseded_by`) and make the save path classify each write as add/update/delete/noop, so the store auto-supersedes stale facts and can answer "what was true at time T?"

**Architecture:** Additive migration 044 adds three nullable columns to `observations`; the save path gains a `SaveWithAction` method that routes a 4-way `DedupDecision` from an extended `DedupLLM.Classify` seam (old `Save` delegates to it, preserving its `(int64, error)` contract). The delete arm stamps `invalid_at` + `superseded_by` + `status='superseded'` and adds a `supersedes` edge in `memory_relations`. All search/read sites gain the bi-temporal filter (the `expires_at` strftime idiom) so superseded rows drop out of results. A `ValidAtTime` service method answers the time-travel query. The `mem_save` MCP response gains additive `action` + `superseded_id` fields.

**Tech Stack:** Go, SQLite (modernc.org/sqlite), FTS5, existing `memory.Service` store. No new dependencies (locked constraint #2 — the ADR-0011 manifest is the vehicle).

**Spec:** `.skillgrid/specs/2026-09-24-bitemporal-audn/briefing.md` (capabilities C1/C1.5/C2; one-way doors; threat matrix; success criteria)

**Findings:** `.skillgrid/specs/2026-09-24-mnemon-comparison/findings.md` (Mnemon + mnemonic-ai both converged on bi-temporal columns + a save-time AUDN classifier — the reference pattern this implements)

**ADR:** `.skillgrid/artifacts/04-adr-0011-observations-are-bitemporal.md` (in force — additive; locks the schema + classifier)

## Global Constraints

- Go 1.22+ minimum to build.
- No new dependencies without an ADR (per ASSUMPTIONS.md § Locked constraints #2) — satisfied: plain SQLite `TEXT`/`INTEGER`, no new Go package.
- Serial development: one change at a time, no parallel branches (per ASSUMPTIONS.md § Locked constraints).
- Spec-zone changes commit before code-zone changes (per ASSUMPTIONS.md § Locked constraints) — this blueprint is spec-zone; commit it before any code task.
- Single-open store contract: a query path opens the store exactly once (`store.OpenCount()` counts every `Open`); reuse the open handle, never a second `Open` (per `06-research-findings.md`, second-brain retro).
- LLM budgets use `ctx.WithDeadline`, never `ctx.WithTimeout` (per `06-research-findings.md`, second-brain retro).
- A new migration `ALTER TABLE`ing a pre-squash table must keep that table in the squash-shim fixture (per `06-research-findings.md`, second-brain retro) — `TestSquashShimUpgrade` already has `observations` in the fixture (added for 043), so 044's ALTERs are safe; do NOT remove it.
- Existing rows read back with NULLs for the three new columns; all read sites must treat NULL/empty `invalid_at` as "still valid".

## Hypothesis

**Claim:** When complete, every new observation carries `valid_at = created_at` / `invalid_at = NULL` / `superseded_by = NULL`; a save classified `delete` sets the old row's `invalid_at` + `superseded_by` + `status='superseded'` and adds a `supersedes` edge; superseded rows no longer appear in search; and `ValidAtTime(T)` returns exactly the facts whose validity window covered T.
**Right condition:** all 9 acceptance scenarios in `acceptance.feature` pass; the bi-temporal filter excludes a row with `invalid_at` set; `ValidAtTime` at T1 returns A-not-B and at T3 returns B-not-A.
**Wrong condition:** a superseded row still ranks in search, or `ValidAtTime` returns a row whose window does not cover T, or `Save`'s return type changed.
**Thinnest MVP:** migration 044 + the 3 struct/scan/insert changes + the bi-temporal filter on the 4 FTS read sites + `ValidAtTime` (the columns + queryable window) — the classifier is the thickening.
**Door check:** Task 1 — if the migration + `valid_at` stamping + `ValidAtTime` window logic don't hold, stop (the temporal foundation is wrong).

## Must-Haves

**Truths:**
1. A new observation has `valid_at = created_at`, `invalid_at = NULL`, `superseded_by = NULL`. (scenario: New observations are valid from creation)
2. A superseded observation (`invalid_at` set) does not appear in `SearchWithScope` / `SearchOwnerScoped` / `SearchOwner` / `AdminCrossOwnerList`. (scenario: Superseded observations are excluded from search)
3. `ValidAtTime(query, T)` returns exactly the rows where `valid_at <= T < invalid_at` (NULL/empty `invalid_at` = still valid). (scenario: Time-travel query returns the fact valid at time T) **`backstop`** — the window math is only observable through the held-out `TestValidAtTime` (two rows with adjacent windows, queried at three times); the diff alone can't settle it.
4. `SaveWithAction` returns `action` ∈ {add, update, delete, noop} and `superseded_id` for delete. (scenario: Save returns an action label)
5. The delete arm sets `invalid_at` (save time), `superseded_by` (new id), `status='superseded'`, and inserts a `supersedes` edge from the old to the new observation. (scenario: Delete arm produces the supersede chain)
6. Deterministic floor (no LLM / LLM error): hash-hit → `noop` + `BumpDuplicate`; hash-miss → `add`. LLM error is non-fatal (logged, degrades to floor). (scenarios: Deterministic floor without LLM; LLM error degrades to deterministic floor)
7. `Save`'s return type is still `(int64, error)` and its behavior is unchanged for existing callers. (scenario: Existing Save callers are unaffected)
8. The `mem_save` MCP response includes the existing `id` + `project` fields and the new additive `action` + `superseded_id` fields. (scenario: mem_save MCP response is additive)

**Artifacts:**
- `store/migrations/044_bitemporal.sql` — 3 `ALTER TABLE` adds + the partial index.
- `memory/service.go` — `SaveWithAction`, `ValidAtTime`, the 3 struct fields, the bi-temporal filter on the read sites, the 3 insert columns.
- `memory/lifecycle.go` — `MarkSuperseded`.
- `memory/types.go` — `DedupVerdict`/`DedupDecision`, `DedupLLM.Classify`, `runDedupCheck` → `Classify`.
- `memory/config` (`config/load.go`) — `Dedup` section.
- `memory/service/service.go` — `newDedupLLMBackend` + `cfg.Dedup.LLM` wiring.
- `mcp/tools_memory.go` — `handleMemSave` → `SaveWithAction` + additive fields.

**Key links:**
- `Save` → `SaveWithAction` (old callers preserved).
- `SaveWithAction` delete arm → `MarkSuperseded` + `AddRelation("supersedes")`.
- `runDedupCheck` → `DedupLLM.Classify` (save path); `isExtractedDuplicate` → `DedupLLM.Dedup` (stays binary).
- `mem_save` MCP → `SaveWithAction` → additive response.
- bi-temporal filter on every FTS/admin read site → superseded rows invisible.

**One-way-door decisions:**
- ⚠ **Migration 044: 3 new columns on `observations`** (additive, nullable, no data migration). Mitigation: `ALTER TABLE ADD COLUMN` pattern matching 040-042; existing rows read back with NULLs.
- ⚠ **`DedupLLM` interface change (binary → 4-way).** Mitigation: keep `Dedup` as a deprecated wrapper; add `Classify` as a new method; existing implementations keep compiling.
- ⚠ **`mem_save` MCP response shape (adds `action`, `superseded_id`).** Mitigation: purely additive JSON; no existing field removed or renamed.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| MCP tool contract | Applicable: `mem_save` response shape changes | Additive fields only (`action`, `superseded_id`). No existing field removed. | `TestMemSaveMCPAdditive` — response has `id`, `project`, `action`; old fields intact (scenario: mem_save MCP response is additive) |
| Schema migration | Applicable: 3 new columns on `observations` | Additive, nullable. No data migration. `IF NOT EXISTS` on index. | `TestMigration044Idempotent` (run twice, no error) + `TestMigration044Columns` (PRAGMA table_info shows 3 cols) (scenario: New observations are valid from creation) |
| Save path | Applicable: new `SaveWithAction`; `Save` return type unchanged | `Save` delegates to `SaveWithAction`, extracts `.ID`. Old callers unaffected. | `TestSaveDelegatesToSaveWithAction` — `Save` returns the same id, same row shape (scenario: Existing Save callers are unaffected) |
| Search visibility | Applicable: bi-temporal filter changes which rows are visible | Filter is additive (`AND` clause). Superseded rows excluded. | `TestSupersededExcludedFromSearch` — row with `invalid_at` set absent from all 4 read sites (scenario: Superseded observations are excluded from search) |
| No new trust boundary | N/A: all changes within the existing per-project SQLite store | — | — |

## File Structure

- `store/migrations/044_bitemporal.sql` — create: 3 `ALTER TABLE observations ADD COLUMN` + `CREATE INDEX IF NOT EXISTS idx_obs_invalid_at` (partial, WHERE `invalid_at IS NOT NULL AND deleted_at IS NULL`).
- `memory/service.go` — modify: `Observation` struct (+3 fields), `obsSelectCols` (+3 cols), `scanObservations` (+3 scan vars), the 4 inline FTS/admin SELECTs (+3 cols each), the 7 read sites (+bi-temporal filter), `Save` INSERT (+3 cols), new `SaveWithAction`, `Save` → delegate, new `ValidAtTime`.
- `memory/lifecycle.go` — modify: new `MarkSuperseded` (mirrors `BumpDuplicate`).
- `memory/types.go` — modify: `DedupVerdict`/`DedupDecision` types, `DedupLLM` +`Classify` (keep `Dedup` deprecated), `runDedupCheck` → `Classify` returning `(DedupDecision, string)`.
- `config/load.go` — modify: `Dedup` section (`mnemonic.dedup.llm`, default false) in `Config` + `mnemonicSection` + `Load` mapping.
- `service/service.go` — modify: `newDedupLLMBackend()` (wraps the extraction LLM client, different prompt) + `if cfg.Dedup.LLM { mem.SetDedupLLM(...); mem.EnableDedupLLM(true) }`.
- `mcp/tools_memory.go` — modify: `handleMemSave` calls `SaveWithAction`, adds `action` + `superseded_id` to the response map.
- Tests: `store/migrations_044_test.go` (new), `memory/bitemporal_test.go` (new), `memory/service_test.go` (add `SaveWithAction`/`ValidAtTime` cases), `memory/types_test.go` (add `Classify` routing cases), `config/load_test.go` (add `Dedup` cases), `mcp/tools_memory_test.go` (add additive-response case).

## Task 1: Migration 044 + struct/scan/select/insert wiring + bi-temporal filter + ValidAtTime

> ⚠ one-way: Migration 044 — 3 new nullable columns on `observations` + partial index.

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/store/migrations/044_bitemporal.sql`
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (Observation struct ~332-377, obsSelectCols ~2095-2100, scanObservations ~2107+, the 4 inline SELECTs at service.go:678/736 + governance.go:429/487, the 7 read sites at service.go:696/706/736/765/792/827/862 + governance.go:453/498, Save INSERT ~581-590)
- Test: `skillgrid-cli/internal/mnemonic/store/migrations_044_test.go` (new), `skillgrid-cli/internal/mnemonic/memory/bitemporal_test.go` (new)

**Interfaces:**
- Consumes: the existing `obsSelectCols`/`scanObservations` pair (must not drift), the `expires_at` strftime idiom (`service.go:707`), the existing `Save` INSERT.
- Produces: `Observation.ValidAt string`, `Observation.InvalidAt string`, `Observation.SupersededBy sql.NullInt64`; `func (s *Service) ValidAtTime(ctx context.Context, query string, atTime string) ([]Observation, error)`.

**SATISFIES:** New observations are valid from creation; Superseded observations are excluded from search; Time-travel query returns the fact valid at time T

- [ ] **Step 1: Write the failing migration + column tests**

```go
// store/migrations_044_test.go
func TestMigration044Columns(t *testing.T) {
    st, err := Open(t.TempDir(), "bitemp044")
    if err != nil { t.Fatal(err) }
    defer st.Close()
    cols := map[string]bool{}
    rows, _ := st.DB.Query(`PRAGMA table_info(observations)`)
    defer rows.Close()
    for rows.Next() { var cid int; var name, ctype string; var notnull, pk int; var dflt any
        rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
        cols[name] = true }
    for _, want := range []string{"valid_at", "invalid_at", "superseded_by"} {
        if !cols[want] { t.Errorf("observations missing column %s", want) } }
}

func TestMigration044Idempotent(t *testing.T) {
    st, _ := Open(t.TempDir(), "bitemp044x")
    defer st.Close()
    // Re-running the migration set must not error (ADD COLUMN on an existing
    // col would; the runner tracks applied migrations, so a second Open on a
    // fresh dir is the idempotency proof — apply through 044 twice via two
    // Opens on the same dir).
    if _, err := Open(t.TempDir(), "bitemp044x"); err != nil { t.Fatal(err) }
}
```

```go
// memory/bitemporal_test.go
func TestSaveStampsBiTemporalColumns(t *testing.T) {
    // save a new observation, read it back via Get
    // assert ValidAt == CreatedAt, InvalidAt == "", SupersededBy invalid
}

func TestValidAtTimeWindow(t *testing.T) {
    // seed A: valid_at=T1, invalid_at=T2 (raw SQL to control the window)
    // seed B: valid_at=T2, invalid_at=NULL
    // ValidAtTime(q, T1) -> [A] only
    // ValidAtTime(q, T3) (T3 > T2) -> [B] only
}

func TestSupersededExcludedFromSearch(t *testing.T) {
    // seed A active, B active, both matching the query
    // set A.invalid_at = now (raw SQL or MarkSuperseded)
    // SearchWithScope / SearchOwnerScoped / SearchOwner / AdminCrossOwnerList
    // -> A absent, B present in all four
}
```

- [ ] **Step 2: Run to confirm they fail**

`cd skillgrid-cli && go test ./internal/mnemonic/store/ -run TestMigration044 -count=1` → FAIL (columns absent); `go test ./internal/mnemonic/memory/ -run 'TestSaveStampsBiTemporal|TestValidAtTime|TestSuperseded' -count=1` → FAIL.

- [ ] **Step 3: Write migration 044**

```sql
-- 044: bi-temporal observations (ADR-0011). Additive only.
--   valid_at      — when the fact became true (new rows: = created_at).
--   invalid_at    — when the fact stopped being true (NULL = still true).
--   superseded_by — back-pointer to the observation that superseded this one
--                   (NULL = not superseded; walked by the four-edge spec).
ALTER TABLE observations ADD COLUMN valid_at TEXT;
ALTER TABLE observations ADD COLUMN invalid_at TEXT;
ALTER TABLE observations ADD COLUMN superseded_by INTEGER;

CREATE INDEX IF NOT EXISTS idx_obs_invalid_at
    ON observations (project, invalid_at)
    WHERE invalid_at IS NOT NULL AND deleted_at IS NULL;
```

- [ ] **Step 4: Add the 3 struct fields** (after `MemoryType` in `Observation`, service.go ~377)

```go
	// ValidAt / InvalidAt / SupersededBy are the bi-temporal validity window
	// (ADR-0011, change 044). Additive + omitempty: pre-044 rows read back
	// empty/NULL, so the JSON contract is unchanged for existing consumers.
	ValidAt      string         `json:"valid_at,omitempty"`
	InvalidAt    string         `json:"invalid_at,omitempty"`
	SupersededBy sql.NullInt64  `json:"superseded_by,omitempty"`
```

- [ ] **Step 5: Add 3 columns to `obsSelectCols` + `scanObservations`**

`obsSelectCols`: append `, valid_at, invalid_at, superseded_by` after `memory_type`.
`scanObservations`: add `var validAt, invalidAt sql.NullString; var supersededBy sql.NullInt64` and scan them in the same order; set `obs.ValidAt = validAt.String`, `obs.InvalidAt = invalidAt.String`, `obs.SupersededBy = supersededBy`.

**Do NOT forget the 4 inline FTS/admin SELECTs** (they don't use `obsSelectCols`): service.go:678 (`SearchWithScope`), service.go:736 (`SearchOwnerScoped`), governance.go:429 (`SearchOwner`), governance.go:487 (`AdminCrossOwnerList`) — add `o.valid_at, o.invalid_at, o.superseded_by` to each so `scanObservations` gets the 3 extra values.

- [ ] **Step 6: Add the bi-temporal filter to the 7 read sites**

The idiom (mirror `expires_at` at service.go:707):
```sql
AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', 'now'))
```
Sites: service.go `SearchWithScope` (~706), `SearchOwnerScoped` (~765), the list sites at ~792/827/862, governance.go `SearchOwner` (~453), `AdminCrossOwnerList` (~498). Add the clause to each `WHERE` that already has `o.deleted_at IS NULL AND o.project = ?`.

- [ ] **Step 7: Stamp the 3 columns on the Save INSERT** (service.go ~581-590)

Add `valid_at, invalid_at, superseded_by` to the column list and `?, NULL, NULL` to the VALUES (or `now, NULL, NULL`). New rows: `valid_at = created_at = now`.

- [ ] **Step 8: Implement `ValidAtTime`** (service.go, near `SearchWithScope`)

```go
// ValidAtTime is the "what was true at time T?" time-travel query (ADR-0011,
// C1.5). It reuses the FTS5 match but replaces the "now" bi-temporal filter
// with an explicit time parameter: a row is returned iff valid_at <= T and
// (invalid_at is NULL/empty or invalid_at > T). Superseded rows whose window
// ended before T are excluded; rows valid at T (even if later superseded) are
// included. Not exposed as an MCP tool in this spec.
func (s *Service) ValidAtTime(ctx context.Context, query string, atTime string) ([]Observation, error) {
    if s == nil || s.store == nil || s.store.DB == nil {
        return nil, errors.New("memory service not initialized")
    }
    ftsQuery := buildFTSQuery(query, "all")
    if ftsQuery == "" { return nil, nil }
    rows, err := s.store.DB.QueryContext(ctx, `
        `+obsSelectWithAlias(o, s)+`   // the o.-prefixed column list, same as SearchWithScope
        FROM observations o
        INNER JOIN observations_fts ON observations_fts.rowid = o.id
        WHERE observations_fts MATCH ?
          AND o.deleted_at IS NULL AND o.project = ?
          AND strftime('%s', o.valid_at) <= strftime('%s', ?)
          AND (o.invalid_at IS NULL OR o.invalid_at = '' OR strftime('%s', o.invalid_at) > strftime('%s', ?))
        ORDER BY bm25(observations_fts)
        LIMIT ` + defaultSearchLimit, ftsQuery, s.projectID, atTime, atTime)
    if err != nil { return nil, fmt.Errorf("valid_at_time: %w", err) }
    defer rows.Close()
    return scanObservations(rows)
}
```

(Extract a small `obsSelectColsAliased(alias string) string` helper so `ValidAtTime` and the 4 inline SELECTs share the `o.`-prefixed column list — DRY, prevents the 5th copy from drifting.)

- [ ] **Step 9: Run the tests**

`cd skillgrid-cli && go test ./internal/mnemonic/store/ ./internal/mnemonic/memory/ -count=1` → PASS. Also run `go test ./internal/mnemonic/store/ -run TestSquashShim -count=1` (the fixture already has `observations`; 044's ALTERs must not break it).

- [ ] **Step 10: Commit**

`git add skillgrid-cli/internal/mnemonic/store/migrations/044_bitemporal.sql skillgrid-cli/internal/mnemonic/memory/service.go skillgrid-cli/internal/mnemonic/memory/governance.go skillgrid-cli/internal/mnemonic/store/migrations_044_test.go skillgrid-cli/internal/mnemonic/memory/bitemporal_test.go && git commit -m "feat(memory): migration 044 bi-temporal columns + ValidAtTime"`

## Task 2: MarkSuperseded + the delete arm's chain (invalid_at + superseded_by + status + edge)

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/lifecycle.go` (new `MarkSuperseded`, near `BumpDuplicate` ~104)
- Test: `skillgrid-cli/internal/mnemonic/memory/bitemporal_test.go` (add `TestMarkSupersededChain`)

**Interfaces:**
- Consumes: `BumpDuplicate` pattern (lifecycle.go:104), `AddRelation`/the `memory_relations` upsert (relations.go ~95-122), `StatusSuperseded` const (governance.go:25).
- Produces: `func (s *Service) MarkSuperseded(ctx context.Context, id int64, supersededByID int64) error` — sets `invalid_at=now, superseded_by=?, status='superseded'` on `id`; the caller adds the edge.

**SATISFIES:** Delete arm produces the supersede chain

- [ ] **Step 1: Write the failing test**

```go
func TestMarkSupersededChain(t *testing.T) {
    // save A (id a), save B (id b)
    // svc.MarkSuperseded(ctx, a, b)
    // add the supersedes edge (svc.AddRelation(ctx, a, b, "supersedes", 1.0))
    // Get(a): InvalidAt set, SupersededBy == b, Status == "superseded"
    // a memory_relations row exists (src=a, dst=b, relation="supersedes")
}
```

- [ ] **Step 2: Run to confirm it fails** (no `MarkSuperseded` yet).

- [ ] **Step 3: Implement `MarkSuperseded`** (lifecycle.go, mirror `BumpDuplicate`)

```go
// MarkSuperseded stamps the bi-temporal invalidity on the superseded
// observation (ADR-0011, C1): invalid_at = now, superseded_by = the new
// observation's id, status = 'superseded'. The caller adds the `supersedes`
// edge (old -> new) — the edge is the forward pointer, superseded_by is the
// back-pointer for chain traversal. Best-effort like BumpDuplicate: returns
// the error but callers may ignore it.
func (s *Service) MarkSuperseded(ctx context.Context, id int64, supersededByID int64) error {
    if s == nil || s.store == nil || s.store.DB == nil {
        return errors.New("memory service not initialized")
    }
    now := time.Now().UTC().Format(time.RFC3339)
    _, err := s.store.DB.ExecContext(ctx, `
        UPDATE observations
        SET invalid_at = ?, superseded_by = ?, status = ?, updated_at = ?
        WHERE id = ? AND project = ? AND deleted_at IS NULL`,
        now, supersededByID, StatusSuperseded, now, id, s.projectID,
    )
    return err
}
```

- [ ] **Step 4: Run the tests** → PASS (the edge is added in the test via `AddRelation`; Task 4 wires it into the save path).

- [ ] **Step 5: Commit** — `git commit -m "feat(memory): MarkSuperseded stamps the bi-temporal invalidity"`

## Task 3: DedupLLM.Classify (4-way) + runDedupCheck switch

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/types.go` (DedupLLM ~37-42, runDedupCheck ~127-156)
- Test: `skillgrid-cli/internal/mnemonic/memory/types_test.go` (add `Classify` routing cases)

**Interfaces:**
- Consumes: `dedupPreFilter`, `dedupCandidateID` (types.go:64/95), the existing binary `Dedup` (kept deprecated).
- Produces: `type DedupVerdict string` (`add`/`update`/`delete`/`noop`), `type DedupDecision struct { Verdict DedupVerdict; TargetID int64; Confidence float64; Reason string }`, `DedupLLM.Classify(ctx, newContent, candidates) (DedupDecision, error)`, `runDedupCheck` now returns `(DedupDecision, string)` (reason ""/"llm"/"hash").

**SATISFIES:** Save returns an action label (the classifier half); LLM error degrades to deterministic floor

- [ ] **Step 1: Write the failing tests**

```go
// a fake Classify seam that returns each of the 4 verdicts; assert
// runDedupCheck returns the decision unchanged (reason "llm").
// a Classify seam that errors; assert runDedupCheck returns the zero
// decision (reason "hash") — the deterministic floor takes over, no panic.
// no seam armed; assert runDedupCheck returns the zero decision (reason "hash").
```

- [ ] **Step 2: Run to confirm they fail.**

- [ ] **Step 3: Add the types + extend the interface** (types.go ~37)

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
    // Dedup is the legacy binary check (kept so existing impls compile).
    // Deprecated: use Classify.
    Dedup(ctx context.Context, newContent string, candidates []string) (duplicate bool, duplicateID int64, err error)
    // Classify is the 4-way AUDN verdict for the save path (ADR-0011, C2).
    Classify(ctx context.Context, newContent string, candidates []string) (DedupDecision, error)
}
```

- [ ] **Step 4: Switch `runDedupCheck` to `Classify`** (types.go ~127)

Return `(DedupDecision, string)`. When armed: `candidates := dedupPreFilter(...)`; if empty → `(DedupDecision{Verdict: VerdictAdd}, "hash")` (no candidates → the floor will insert). `d, err := s.dedupLLM.Classify(...)`; on error → log warning, return `(DedupDecision{}, "hash")`. On success, if `d.TargetID` is set and not resolvable via `dedupCandidateID`, fall back to the hash floor. `isExtractedDuplicate` (types.go:282) **stays on `Dedup`** (binary) — do not change it.

- [ ] **Step 5: Update the existing `runDedupCheck` callers** — the only caller is `Save` (service.go:511); it will be replaced by `SaveWithAction` in Task 4, so for now adapt `Save` to ignore the new shape (call `runDedupCheck`, and if `Verdict == VerdictNoop && TargetID > 0` → `BumpDuplicate` + return; otherwise fall through). This keeps `Save` green until Task 4 rewrites it.

- [ ] **Step 6: Run the tests** → PASS (`go test ./internal/mnemonic/memory/ -count=1`).

- [ ] **Step 7: Commit** — `git commit -m "feat(memory): DedupLLM.Classify 4-way AUDN verdict + runDedupCheck switch"`

## Task 4: SaveWithAction (4-way routing) + Save delegates

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/memory/service.go` (Save ~435-613 → rename body to `SaveWithAction`, add the 4-way routing, new `Save` wrapper)
- Test: `skillgrid-cli/internal/mnemonic/memory/service_test.go` (add `SaveWithAction` routing cases + `TestSaveDelegatesToSaveWithAction`)

**Interfaces:**
- Consumes: `runDedupCheck` (Task 3), `BumpDuplicate`, `MarkSuperseded` (Task 2), `AppendVersion` (topic-key upsert, service.go:530), the existing hash lookup + INSERT.
- Produces: `type SaveResult struct { ID int64 \`json:"id"\`; Action string \`json:"action,omitempty"\`; SupersededID int64 \`json:"superseded_id,omitempty"\` }`; `func (s *Service) SaveWithAction(ctx context.Context, in SaveInput) (SaveResult, error)`; `func (s *Service) Save(ctx, in) (int64, error)` (delegates, returns `.ID`).

**SATISFIES:** Save returns an action label; Delete arm produces the supersede chain; Deterministic floor without LLM; LLM error degrades to deterministic floor; Existing Save callers are unaffected

- [ ] **Step 1: Write the failing routing tests**

```go
// SaveWithAction routing table — for each verdict, a fake Classify seam:
//   noop   -> BumpDuplicate(targetID), result {ID: targetID, Action:"noop"}
//   add    -> INSERT, result {ID: newID, Action:"add"}
//   update -> topic-key upsert (when in.TopicKey set) or new row, result {ID: targetID, Action:"update"}
//   delete -> MarkSuperseded(targetID, newID) + supersedes edge + INSERT,
//             result {ID: newID, Action:"delete", SupersededID: targetID}
// deterministic floor (no seam): hash-hit -> noop+BumpDuplicate; hash-miss -> add.
// LLM error (seam returns err): hash-miss -> add (non-fatal, logged).
// TestSaveDelegatesToSaveWithAction: Save returns the same ID as SaveWithAction,
// row shape unchanged (valid_at=created_at, etc.).
```

- [ ] **Step 2: Run to confirm they fail.**

- [ ] **Step 3: Rename `Save` → `SaveWithAction`** (service.go:435) and restructure the decision point (the hash hit at ~496 and the dedup check at ~511):

```go
func (s *Service) SaveWithAction(ctx context.Context, in SaveInput) (SaveResult, error) {
    // ...same validation as Save (session/title/content/type/memoryType) ...
    // hash hit:
    if err == nil { // existingID from the 24h normalized-hash lookup
        s.BumpDuplicate(ctx, existingID)
        return SaveResult{ID: existingID, Action: string(VerdictNoop)}, nil
    }
    // LLM 4-way classify (Task 3):
    if dec, reason := s.runDedupCheck(ctx, in.Content); reason == "llm" && dec.Verdict != "" {
        switch dec.Verdict {
        case VerdictNoop:
            if dec.TargetID > 0 {
                s.BumpDuplicate(ctx, dec.TargetID)
                return SaveResult{ID: dec.TargetID, Action: string(VerdictNoop)}, nil
            }
        case VerdictUpdate:
            // topic-key upsert if in.TopicKey set (reuse the AppendVersion+UPDATE
            // block at ~516-548); else a new upsert-by-id branch.
            // return SaveResult{ID: <target or new>, Action: string(VerdictUpdate)}
        case VerdictDelete:
            // INSERT the new row first (so we have its id), then MarkSuperseded
            // + the supersedes edge.
            newID, err := s.insertObservation(ctx, in, now, hash, ...) // extract the INSERT into a helper
            if err != nil { return SaveResult{}, err }
            if dec.TargetID > 0 {
                _ = s.MarkSuperseded(ctx, dec.TargetID, newID)
                _, _ = s.AddRelation(ctx, dec.TargetID, newID, "supersedes", 1.0)
                return SaveResult{ID: newID, Action: string(VerdictDelete), SupersededID: dec.TargetID}, nil
            }
        }
        // VerdictAdd or unresolvable target: fall through to the INSERT below.
    }
    // deterministic floor: topic-key upsert (if in.TopicKey) else INSERT.
    // ...existing topic-key + INSERT path, now via the insertObservation helper...
    if in.TopicKey != "" { /* existing upsert block; return {ID: topicID, Action:"update"} */ }
    newID, err := s.insertObservation(ctx, in, now, hash, ...)
    if err != nil { return SaveResult{}, err }
    return SaveResult{ID: newID, Action: string(VerdictAdd)}, nil
}

func (s *Service) Save(ctx context.Context, in SaveInput) (int64, error) {
    r, err := s.SaveWithAction(ctx, in)
    return r.ID, err
}
```

Extract the INSERT (service.go:581-611) into `func (s *Service) insertObservation(ctx, in, now, hash string, provenanceJSON sql.NullString, memoryType string) (int64, error)` so both the `delete` and `add` arms reuse it. The `delete` arm calls `MarkSuperseded` + `AddRelation` **after** the INSERT (it needs the new id for `superseded_by`).

- [ ] **Step 4: Run the tests** → PASS.

- [ ] **Step 5: Run the full memory package** — `go test ./internal/mnemonic/memory/ -count=1` (existing `Save` callers must stay green — `TestSaveDelegatesToSaveWithAction` is the backstop).

- [ ] **Step 6: Commit** — `git commit -m "feat(memory): SaveWithAction 4-way AUDN routing + Save delegates"`

## Task 5: Config `mnemonic.dedup.llm` + `newDedupLLMBackend` wiring

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/config/load.go` (`Dedup` section in `Config` ~160-190, `mnemonicSection` ~210-230, `Load` mapping ~450)
- Modify: `skillgrid-cli/internal/mnemonic/service/service.go` (`newDedupLLMBackend` + the `if cfg.Dedup.LLM` wiring near `mem.EnableExtractionLLM(cfg.Extraction.LLM)` ~411)
- Test: `skillgrid-cli/internal/mnemonic/config/load_test.go` (add `Dedup` cases)

**Interfaces:**
- Consumes: the `Extraction` section pattern (config/load.go:66, 221, 453), the `EnableExtractionLLM` wiring pattern (service.go:411), `mem.SetDedupLLM`/`mem.EnableDedupLLM` (memory/service.go:227-240).
- Produces: `Config.Dedup.DedupLLM bool` (yaml `mnemonic.dedup.llm`, default false); `newDedupLLMBackend() memory.DedupLLM` (wraps the extraction LLM client with a different prompt).

**SATISFIES:** LLM error degrades to deterministic floor (the wiring that arms the seam); Save returns an action label (the LLM path is reachable)

- [ ] **Step 1: Write the failing config tests**

```go
// Load(dir) with no dedup section -> got.Dedup.LLM == false (default)
// Load(dir) with mnemonic.dedup.llm: true -> got.Dedup.LLM == true
```

- [ ] **Step 2: Run to confirm they fail.**

- [ ] **Step 3: Add the `Dedup` section** (config/load.go, mirror `Extraction`)

```go
// Dedup is the mnemonic.dedup section (ADR-0011, C2): the 4-way AUDN
// classifier opt-in. LLM defaults to false -> the deterministic hash floor
// is the only classifier (add + noop only; no update/delete).
type Dedup struct{ LLM bool }
// in Config: Dedup Dedup
// in mnemonicSection: Dedup dedupSection `yaml:"dedup"`
// in Load: out.Dedup = Dedup{LLM: section.Dedup.LLM}
```

- [ ] **Step 4: Add `newDedupLLMBackend` + the wiring** (service/service.go ~411)

```go
// newDedupLLMBackend wraps the extraction LLM client with the AUDN classify
// prompt (ADR-0011, C2). It follows the ExtractionLLM backend pattern
// (service.go:117): a small function-interface backend; a nil client yields a
// nil seam (the deterministic floor stays in force).
func newDedupLLMBackend() memory.DedupLLM { /* wrap the shared client; return nil if absent */ }

if cfg.Dedup.LLM {
    if b := newDedupLLMBackend(); b != nil {
        mem.SetDedupLLM(b)
    }
    mem.EnableDedupLLM(true)
}
```

- [ ] **Step 5: Run the tests** — `go test ./internal/mnemonic/config/ ./internal/mnemonic/service/ -count=1` → PASS.

- [ ] **Step 6: Commit** — `git commit -m "feat(memory): mnemonic.dedup.llm config + newDedupLLMBackend wiring"`

## Task 6: mem_save MCP → SaveWithAction + additive response

**Files:**
- Modify: `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` (`handleMemSave` ~183-265)
- Test: `skillgrid-cli/internal/mnemonic/mcp/tools_memory_test.go` (add `TestMemSaveMCPAdditive`)

**Interfaces:**
- Consumes: `SaveWithAction` (Task 4), the existing `handleMemSave` request parsing.
- Produces: the `mem_save` response map gains `action` (when non-empty) + `superseded_id` (when > 0), alongside the existing `id` + `project`.

**SATISFIES:** mem_save MCP response is additive

- [ ] **Step 1: Write the failing test**

```go
func TestMemSaveMCPAdditive(t *testing.T) {
    // call handleMemSave for a new observation
    // response JSON has: id (number), project (string), action == "add"
    // for a hash-dup: action == "noop", no superseded_id
    // existing id/project fields intact (old consumers unaffected)
}
```

- [ ] **Step 2: Run to confirm it fails.**

- [ ] **Step 3: Switch `handleMemSave` to `SaveWithAction`** (tools_memory.go ~244)

```go
result, err := svc.SaveWithAction(ctx, in)
if err != nil { /* existing error path */ }
out := map[string]any{"id": result.ID, "project": projectID}
if result.Action != "" { out["action"] = result.Action }
if result.SupersededID > 0 { out["superseded_id"] = result.SupersededID }
```

- [ ] **Step 4: Run the tests** — `go test ./internal/mnemonic/mcp/ -count=1` → PASS.

- [ ] **Step 5: Commit** — `git commit -m "feat(mcp): mem_save uses SaveWithAction + additive action/superseded_id"`

## Task 7: Full-suite regression + verification floor (L3)

**Files:** none (verification only)

**SATISFIES:** All existing tests pass (no regression) — the success-criteria backstop.

- [ ] **Step 1: Build** — `cd skillgrid-cli && go build ./...` → clean.
- [ ] **Step 2: Full test suite** — `go test ./... -count=1` → all packages ok. (Watch the known pre-existing `http/tracker` flake — gh CLI 10s timeout under parallel load; passes in isolation, not a regression.)
- [ ] **Step 3: Vet** — `go vet ./...` → clean (the known pre-existing context-cancel leak at `memory/budget.go:114` is out of scope — do not "fix" it here).
- [ ] **Step 4: Coverage** — `go test ./internal/mnemonic/... -cover -count=1` → record the package coverage; the new `memory` tests must not drop the package below its prior level.
- [ ] **Step 5: Commit** — `git commit -m "test(memory): bi-temporal + AUDN regression suite"` (only if test files changed; otherwise no commit).

## Global Constraints (reminder)

- Go 1.22+; no new dependencies (ADR-0011 manifest is the vehicle).
- Single-open store contract (reuse the open handle; never a second `Open` in a query path).
- `ctx.WithDeadline` for any LLM budget, never `ctx.WithTimeout`.
- The squash-shim fixture already has `observations` (added for 043) — 044's ALTERs are safe; do not remove it.
- Existing rows read back with NULL/empty for the 3 new columns; every read site treats that as "still valid".
