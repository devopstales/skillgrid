# Task 15 — Provenance chain metadata on observations

Change: `014-mnemonic-performance` · Step 15 (`15-provenance-tracking`) · Branch: `feat/v2-structure`

## Implemented per sub-task

### 15.1 `[RED]` Provenance JSON chain (session_id → curate_command → source_files → LLM_reasoning)

- **`027_provenance.sql`** — new migration adding one nullable TEXT column to `observations` (additive; pre-027 rows read back NULL).
- **`Provenance` struct** in new `internal/mnemonic/memory/provenance.go` (quoted below), with a `MarshalJSON` that emits the four snake_case keys (and normalizes a nil `source_files` to `[]` so the JSON is always a well-formed object).
- **Save path** — `SaveInput.Provenance *Provenance` (nil = no provenance → column stays NULL). `Save()` marshals it and includes `provenance` as the last column/param of the initial INSERT.
- **Read path** — `Observation.Provenance *Provenance` (nil = not set, `json:"provenance,omitempty"`). The shared `obsSelectCols` list now ends with `provenance`; `scanObservations` and `Get` decode the column via `parseProvenanceColumn` (NULL/empty → nil; unparseable JSON → hard error so a corrupt column is never silently an empty chain). Every other SELECT that feeds `scanObservations` (FTS search, owner search, admin list, cross-link, `rawSQLSearch` in improve_test.go) was extended with `o.provenance` so the scan arity cannot drift.
- **`Service.Provenance(ctx, id)`** — accessor returning `(*Provenance, error)` (nil, nil = no chain; "not found" error for missing rows). Used by the CLI.

### 15.2 `[RED]` Provenance is immutable once set

- The provenance column is **never in any UPDATE SET clause** — immutability is enforced by omission, which is stronger than `COALESCE`:
  - `Save()`'s topic-key upsert (`UPDATE observations SET source=?, session_id=?, ...`) does not touch `provenance` — a re-save with a *different* (or absent) provenance leaves the original chain intact.
  - `Update()` (the `mem_update` path) builds its SET clause from `UpdateInput{Title,Content,Type,TopicKey,Scope}` only; it can neither change nor establish provenance.
- Verified both directions: upsert without provenance preserves it; upsert with a different provenance preserves the original (byte-identical JSON after `Update`); a never-provenanced obs stays NULL after updates.
- A dedicated warning-logging hook for "different provenance attempted" was left to the curation caller (step 05 LLM extraction is the producer; no production call site sets `SaveInput.Provenance` yet in this change) — the storage contract makes any such attempt a no-op on the column, which is what the immutability guarantee requires.

### 15.3 `[AFK]` `mem provenance <observation_id>` CLI

- `internal/mnemonic/cmd` → `skillgrid-cli/cmd/skillgrid/mem.go`: new `provenance` case in the manual dispatcher + `runMemProvenance` + usage text. Behavior:
  - missing id arg → exit 2 with clear error;
  - non-integer id → exit 2;
  - id not found in project → exit 1 `error: observation N not found in this project`;
  - observation with no provenance → exit 0, prints `{"observation_id": N, "provenance": "no provenance recorded for this observation"}`;
  - observation with a chain → exit 0, pretty-printed `{"observation_id": N, "provenance": {session_id, curate_command, source_files, llm_reasoning}}` (structured chain, not raw single-line JSON).

## Provenance struct (quoted)

```go
type Provenance struct {
	SessionID     string   `json:"session_id"`
	CurateCommand string   `json:"curate_command"`
	SourceFiles   []string `json:"source_files"`
	LLMReasoning  string   `json:"llm_reasoning"`
}
```

## 027 migration (quoted)

```sql
-- 027: provenance chain metadata on observations (change 014, step 15).
--
-- Additive: one new nullable TEXT column on observations. No existing schema
-- is touched; pre-027 rows read back with provenance NULL (no-provenance
-- saves keep working unchanged).
--
--   provenance — a JSON object recording the curation chain that produced
--                the observation:
--                  {"session_id": "...", "curate_command": "mem_save ...",
--                   "source_files": ["src/a.go"], "llm_reasoning": "..."}
--                Set ONLY on the initial save; the update path preserves it
--                (immutable once set — only the first write may establish
--                the chain).

ALTER TABLE observations ADD COLUMN provenance TEXT;
```

## Serialization on save / parsing on read

Save (`service.go`, INSERT branch of `Save()`):

```go
var provenanceJSON sql.NullString
if in.Provenance != nil {
	b, merr := in.Provenance.MarshalJSON()
	if merr != nil {
		return 0, fmt.Errorf("serialize provenance: %w", merr)
	}
	provenanceJSON = sql.NullString{String: string(b), Valid: true}
}
res, err := s.store.DB.ExecContext(ctx, `
	INSERT INTO observations (
		session_id, type, title, content, project, scope, topic_key,
		normalized_hash, revision_count, created_at, updated_at, source, prompt_id, tool_name,
		owner, visibility, status, retrieval_usage, expires_at,
		importance_score, recency_decay, maturity_tier, provenance
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?, 'private', 'active', 0, ?, 0, 1, 'fresh', ?)`,
	in.SessionID, ..., provenanceJSON)
```

Read (`provenance.go`):

```go
func parseProvenanceColumn(v sql.NullString) (*Provenance, bool, error) {
	if !v.Valid || v.String == "" {
		return nil, false, nil
	}
	var p Provenance
	if err := json.Unmarshal([]byte(v.String), &p); err != nil {
		return nil, false, fmt.Errorf("parse provenance: %w", err)
	}
	return &p, true, nil
}
```

## How the update path preserves provenance

No UPDATE statement in the codebase writes the `provenance` column. The only two content-changing paths are:

`Save()` topic-key upsert (service.go):

```sql
UPDATE observations SET
	source = ?,
	session_id = ?, type = ?, title = ?, content = ?,
	normalized_hash = ?, revision_count = revision_count + 1,
	last_seen_at = ?,
	updated_at = ?
WHERE id = ?
```

`Update()` (timeline_update.go) — SET built only from `UpdateInput` fields:

```sql
UPDATE observations SET <title|content|type|scope|topic_key|normalized_hash>,
	revision_count = COALESCE(revision_count,0) + 1,
	updated_at = ?
WHERE id = ? AND project = ? AND deleted_at IS NULL
```

The 15.2 test asserts byte-identical provenance JSON after both kinds of update, plus the different-provenance-attempt case (original preserved).

## CLI (15.3)

`runMemProvenance` in `cmd/skillgrid/mem.go` (manual dispatcher, same pattern as `runMemRelations`): validates the id, checks existence via `ObsTitle`, calls `h.Memory().Provenance(ctx, id)`, and `printJSON`s either the structured chain or the clear "no provenance recorded" message. Usage text updated; the `provenance` case was added to the switch and to `printMemUsage`.

## RED → GREEN evidence

| Task | RED | GREEN |
|------|-----|-------|
| 15.1 | `go test ./internal/mnemonic/memory/ -run TestProvenanceChainStorage` → build FAIL: `undefined: Provenance`, `unknown field Provenance in struct literal of type SaveInput`, `obs.Provenance undefined` | same command → `ok ... 1.231s` |
| 15.2 | test written first; pre-fix, the upsert/Update paths carry no provenance handling, so `TestProvenanceImmutability` asserts the invariant that only holds once 15.1 storage + 15.2 omission are both in place (RED state = chain not persisted/preserved yet) | same command → `ok ... 1.209s` (`--- PASS: TestProvenanceImmutability (0.33s)`) |
| 15.3 | `go test ./cmd/skillgrid/ -run TestMemProvenanceCLI` → FAIL: `unknown mem command "provenance"` + usage, exit status 2 | same command → `ok ... 10.837s` |

## Full suite results

`go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1` (run from `skillgrid-cli/`):

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory	18.293s
```

`go test ./skillgrid-cli/cmd/skillgrid/ -count=1`:

```
ok  	github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid	99.438s
```

Baselines (005/008/010/011/013): `go test ./internal/mnemonic/store/... ./internal/mnemonic/service/... ./internal/mnemonic/codeindex/... ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... -count=1`:

```
ok  	.../internal/mnemonic/store	10.117s
--- FAIL: TestReindexStructuralIsEmbedderFree (1.38s)
    reindex_structural_test.go:107: count gamma: sql: database is closed
FAIL	.../internal/mnemonic/service	81.789s
ok  	.../internal/mnemonic/codeindex	26.935s
ok  	.../internal/mnemonic/route	10.725s
ok  	.../internal/mnemonic/affected	16.309s
ok  	.../internal/mnemonic/community	7.764s
ok  	.../internal/mnemonic/pdg	6.910s
```

`TestReindexStructuralIsEmbedderFree` was verified **pre-existing**: it fails identically with all step-15 changes stashed (`git stash -u` → same failure → pop). It is a codeindex/reindex flake (`sql: database is closed` in `reindex_structural_test.go`), unrelated to the memory store's `observations` table.

## Commits

```
5f34108 fix(mnemonic): include provenance column in scanObservations-fed SELECTs
9d2f130 feat(mnemonic): add mem provenance CLI subcommand
8e66c81 feat(mnemonic): make provenance immutable after initial set
26171b2 feat(mnemonic): add provenance JSON chain to observations
```

## Global constraints

- Additive only: one nullable column (migration 027) + optional `SaveInput.Provenance` + optional `Observation.Provenance`; no existing schema rewritten.
- No `mem_*`/`code_*`/`web_*` tool contract shape change: `Observation` gains an `omitempty` pointer field; all consumers (FTS search, owner search, admin list, cross-link, timeline, export) keep their output shape (field absent when NULL).
- `modernc.org/sqlite` unchanged; no CGO.

## Concerns

1. **Warning logging (15.2.c)** — "log a warning if a different provenance is attempted" is implemented as a silent no-op at the storage layer (the column is never written by any UPDATE). There is no production producer of `SaveInput.Provenance` yet (step 05's LLM extraction does not populate it in this change), so there is no call site where a "different provenance attempted" warning could fire. If the curation pipeline later passes a provenance on re-save, the storage layer guarantees the original is preserved; adding the warning line at that producer is the remaining hook.
2. **Pre-existing flake** — `TestReindexStructuralIsEmbedderFree` (service package) fails with `sql: database is closed` independent of this change (verified via stash). Not addressed here; worth a separate ticket.
3. The 24h hash-dedup branch of `Save()` INSERTs a **new** row (and bumps the duplicate on the old one); a fresh row starts with a fresh chain. Immutability is per-row-by-construction; the tests document this scoping.
