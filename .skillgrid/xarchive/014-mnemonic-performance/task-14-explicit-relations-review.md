# Step 14 — Explicit Relations (`@relation` annotations) — Review

**Verdict: PASS WITH WARNINGS**

Fresh reviewer. No prior context. Scoped to the 5 step-14 files in
`review-step14-scoped.diff` (587 lines): `cmd/skillgrid/mem.go`,
`cmd/skillgrid/mem_relations_test.go`, `memory/observation_relations.go` [NEW],
`memory/observation_relations_test.go` [NEW], `store/migrations/026_observation_relations.sql` [NEW].

All three step-14 tests pass locally; `go vet` on the touched packages is clean
(the one vet warning is pre-existing in `budget.go:114`, not step 14).

## Findings

### F1 — major — two parallel relation systems (DESIGN, not correctness)
**File:** `memory/observation_relations.go` (new) vs `memory/relations.go` (pre-existing, 006)

There are now **two** typed-relation systems in the same package:

| | pre-existing `relations.go` | new `observation_relations.go` |
|---|---|---|
| Table | `memory_relations` (006) | `observation_relations` (026) |
| Types | `related, compatible, scoped, conflicts_with, supersedes` (Engram-parity verdict vocab) | `mentions, depends_on, contradicts, supports, references` (new @relation vocab) |
| Confidence | `*float64` (nullable) | `float64` (required, validated `[0,1]`) |
| Extra cols | `reason`, `project`, `deleted_at` (soft delete), auto `id` | none (hard delete, composite PK) |
| Upsert | manual SELECT-then-UPDATE/INSERT | `ON CONFLICT … DO UPDATE` |
| Self-rel | **rejected** (`srcID == dstID` error, `relations.go:79`) | **allowed** (no check) |
| Consumers | MCP `tools_memory.go:944,982,994` + HTTP `server.go:981,1017,1042,1075` | `mem relations` CLI only |
| Bidirectional read | `RelationsOf`/`RelationsBetween` (both directions) | `GetRelations` (`source_id = ? OR target_id = ?`) |

**Assessment:** This is a **deliberate, defensible split**, not accidental duplication:
- The old system is the **Engram-parity verdict system** — its vocabulary is the
  `mem_judge`/`mem_compare` verdict set (`conflicts_with`, `supersedes`, …), it is
  **soft-deleted**, carries `reason` (judge rationale), and is **actively consumed
  by MCP/HTTP**. Its contract (tool shapes) is frozen by the Global Constraints and
  must NOT be touched here. Merging into it would risk those contracts.
- The new system is a **cleaner, purpose-built** typed edge: required
  confidence-weighted field, canonical 5-type `@relation` vocabulary, hard delete,
  composite-PK upsert, a dedicated bidirectional read with a confidence floor.

The two are distinguishable by **vocabulary** (zero overlap between the two type
sets), **storage model** (soft vs hard delete), and **consumer surface** (MCP/HTTP
vs CLI). For *this step* the split is acceptable: it is additive, it does not break
the frozen tool contracts, and the new table is the right shape for confidence
filtering (a nullable-`confidence` + soft-delete table makes `WHERE confidence >= ?`
awkward).

**Why it is a WARNING, not a nit:** the cost is real. Two near-identical
"typed edge between observations" concepts now live side by side with overlapping
naming (`Relation` vs `ObservationRelation`, `RecordRelation` vs `AddRelation`,
`RemoveRelation` vs `DeleteRelation`, `RelationsOf` vs `GetRelations`). The
implementer's report and file-header comments do a good job of distinguishing them,
but a future contributor can easily write to the wrong table. This is a
**consolidation candidate for a later change** (migrate the CLI onto the MCP
verdict table, or vice-versa, once the frozen contracts are revisited). It does not
block step 14.

### F2 — minor — self-relation: allowed at the store layer, but the CLI mislabels direction
**File:** `cmd/skillgrid/mem.go:566-572`

The `@edge` acceptance scenario ("Self-relation is allowed and queryable",
`acceptance.feature:505-509`) is satisfied at the `memory` layer — I verified
`AddRelation(a, a, "mentions", 0.6)` round-trips through `GetRelations` (stored +
returned as `{src:a, tgt:a}`).

However the CLI's direction logic assumes `source_id != target_id`:

```go
dir := "outgoing"
other := r.TargetID
if r.SourceID != id {   // for a self-relation this is FALSE
    dir = "incoming"
    other = r.SourceID
}
```

For a self-relation it emits `direction=outgoing, other_id=<self>` — correct enough
(the edge *is* outgoing and the other endpoint *is* the self), but there is no
`"self"` label, so the output is slightly misleading for the one edge case the
acceptance feature calls out by name. Cosmetic; the scenario's observable
contract (stored + queryable) holds. **Fix (optional):** add a
`if r.SourceID == id && r.TargetID == id { dir = "self" }` branch.

Note the **inconsistent** treatment vs the old system: `relations.go:79` *rejects*
self-relations (`source and destination observation IDs must differ`), while the new
system *allows* them. The acceptance feature's `@edge` scenario is what forced the
new system to allow them, so the new behavior is spec-correct — but the two systems
now disagree on the same input. Part of the F1 consolidation debt.

### F3 — minor — `DeleteRelation` is unexercised dead code in this step
**File:** `memory/observation_relations.go:116-134`

`DeleteRelation` is defined (14.1 requires it) but has **no caller** — not in the
`mem relations` CLI (read-only subcommand), not in any MCP/HTTP handler, and not in
any of the 3 step-14 tests (which exercise `AddRelation` + `GetRelations` only). It
is correct and compiles, but nothing proves it at runtime this step. It is a public
`*Service` method, so it is not flagged by the linter. **Fix (optional):** add a
small round-trip test (Add then Delete, assert `true` returned and `GetRelations`
empty), or wire a `mem relations --remove` / delete path in a follow-up. Acceptable
as-is since the brief only required the method to exist and be correct.

### F4 — nit — `GetRelations` silently clamps negative `minConfidence` to 0.0
**File:** `memory/observation_relations.go:87-89`

```go
if minConfidence < 0.0 {
    minConfidence = 0.0
}
```

The CLI validates `--min-confidence` to `[0,1]` (exit 2) before it reaches here, so
this clamp is dead-in-practice for the CLI. But `GetRelations` is a public method; a
future MCP/HTTP caller passing `-0.5` is silently reinterpreted as "return all"
rather than rejected. The sibling `AddRelation` *does* range-check `confidence`
strictly (`> 1.0` errors) — so the two halves are inconsistent about input strictness.
**Fix (optional):** either error on `minConfidence < 0.0 || > 1.0` (match
`AddRelation`) or document the clamp. Non-blocking.

### F5 — nit — brief deviation on method name (`RemoveRelation` → `DeleteRelation`)
**File:** `memory/observation_relations.go:112-115`

The brief names the delete `RemoveRelation`; the implementer named it
`DeleteRelation` to avoid clashing with the pre-existing
`relations.go:145 RemoveRelation(ctx, srcID, dstID, relation)`. **Justified** — Go
has no overloading and both methods sit on `*Service`; the rename is documented in
the method comment and the report. Behavior (delete one typed edge by full triple)
matches the brief's intent. Not a defect; flagged for the record so the deviation is
deliberate, not accidental.

## Evidence checked (7 checks)

1. **Schema (14.1)** — ✅ `026_observation_relations.sql` creates
   `observation_relations` with `source_id INTEGER NOT NULL, target_id INTEGER NOT
   NULL, relation_type TEXT NOT NULL, confidence REAL NOT NULL, created_at TIMESTAMP
   DEFAULT CURRENT_TIMESTAMP, PRIMARY KEY (source_id, target_id, relation_type)`.
   Additive: `CREATE TABLE IF NOT EXISTS`. Both bidirectional indexes present
   (`idx_obs_rel_source`, `idx_obs_rel_target`). Matches the brief exactly.
2. **AddRelation (14.1)** — ✅ Validates the 5 canonical types
   (`canonicalRelationTypes` map, `observation_relations.go:23-29`) and rejects
   unknowns with the valid set listed. Validates `confidence ∈ [0.0, 1.0]`
   (`:52`). Validates both obs are live via the shared `obsExists`
   (`:55-58`). Composite PK makes re-assertion an UPSERT. INSERT quoted:
   ```sql
   INSERT INTO observation_relations (source_id, target_id, relation_type, confidence)
   VALUES (?, ?, ?, ?)
   ON CONFLICT (source_id, target_id, relation_type)
   DO UPDATE SET confidence = excluded.confidence,
                 created_at = excluded.created_at
   ```
   (Contrast: the pre-existing `RecordRelation` cannot use `ON CONFLICT` because
   006 uses a *partial* unique index + soft delete; the new 026 table can, which is
   a genuine simplification.)
3. **Bidirectional query (14.2)** — ✅ `GetRelations` uses
   `WHERE (source_id = ? OR target_id = ?) AND confidence >= ?`
   (`observation_relations.go:93`). CLI (`mem.go:552-582`) labels each row
   `outgoing` (anchor is source) / `incoming` (anchor is target) and surfaces the
   other endpoint's id + title via `ObsTitle`. Missing observation id →
   `observation N not found in this project`, exit 1 (`mem.go:547-550`). No
   relations → `"relations": []`, `"count": 0` (slice pre-sized to 0 at
   `mem.go:565`). All verified in `TestMemRelationsCLI`.
4. **Confidence filtering (14.3)** — ✅ `GetRelations(obsID, minConfidence)` adds
   `AND confidence >= ?` in the *single* WHERE clause alongside the bidirectional
   OR — so the floor is applied to **both directions** (a source-edge and a
   target-edge each pass the same `confidence >= ?`). `minConfidence=0.0` returns
   all. `TestRelationConfidenceFiltering` proves 0.3/0.7/0.95 → floor 0.5 keeps
   exactly {0.7, 0.95}; floor 0.0 keeps all 3. CLI `--min-confidence` validated to
   `[0,1]` and exercised in `TestMemRelationsCLI` (`0.85` keeps only the 0.9 edge).
   **This is the filter the brief cares about and it is correct in both directions.**
5. **Two-systems concern** — ✅ See F1. Distinct, defensible design (verdict system
   vs typed @relation system); acceptable for this step; consolidation is a
   follow-up, not a blocker.
6. **Additive / tool contract** — ✅ Diff touches only the 5 step-14 files. No
   `mem_*`/`code_*`/`web_*` tool registration changed. Pre-existing
   `relations.go` (`RecordRelation`/`RemoveRelation`/`RelationsOf`/
   `RelationsBetween`) and its consumers (`tools_memory.go`, `http/server.go`) are
   **untouched** — confirmed by scoped diff + `rg` (the only MCP/HTTP refs to
   `RemoveRelation` are the old ones at `tools_memory.go:982`, `server.go:1017`).
   New methods are CLI-surface only. `modernc.org/sqlite` retained; no CGO.
7. **TDD discipline** — ✅ 3 tests, all passing locally:
   - `TestRelationEdgeCreation` — AddRelation round-trip (source/target/type/conf
     all asserted) + all 5 canonical types created on fresh pairs.
   - `TestMemRelationsCLI` — runs the *real* `mem relations` subcommand via
     `go run .`; asserts B/C/D present with types + confidences, `--min-confidence
     0.85` filters to only the 0.9 edge, isolated obs → `"relations": []`, missing
     id → error containing "not found".
   - `TestRelationConfidenceFiltering` — 0.3/0.7/0.95 → floor 0.5 keeps 2
     (asserts no below-floor row, asserts 0.7+0.95 present), floor 0.0 keeps 3.
   RED→GREEN commits (`91ef284`, `5082ae5` tests before `67ae959`, `9c3dcbb` impl)
   land in the correct order.

## Runtime evidence

```
go test ./internal/mnemonic/memory/ -run 'TestRelationEdgeCreation|TestRelationConfidenceFiltering' -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  1.053s

go test ./cmd/skillgrid/ -run 'TestMemRelationsCLI' -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  5.741s
```

Self-relation (the `@edge` scenario) verified empirically: `AddRelation(a, a,
"mentions", 0.6)` stores `{src:a, tgt:a}` and `GetRelations` returns it — the
scenario's observable contract (stored + queryable) holds at the memory layer.

Pre-existing, unrelated failure (`TestReindexStructuralIsEmbedderFree`, service
package, `sql: database is closed`) is **not** caused by step 14: the step-14
packages (`memory`, `cmd/skillgrid`) pass clean, and the new code is strictly
additive (new table + new methods + new CLI subcommand). Carried into the baseline.

## Approved to merge

**Yes** — with warnings. Step 14 delivers all three sub-tasks (14.1/14.2/14.3) per
the brief, all acceptance-feature step-14 scenarios are satisfied, Global
Constraints hold (additive; frozen `mem_*`/`code_*`/`web_*` tool contracts
untouched; no CGO), and the two-systems split (F1) is a deliberate, documented
design decision that is acceptable for this step. F1 (consolidation debt) and F2
(self-relation CLI label) should be tracked as follow-ups; F3/F4 are optional
polish.
