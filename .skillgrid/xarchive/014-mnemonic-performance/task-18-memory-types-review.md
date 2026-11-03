# Task 18 Review — Typed memory categories + LLM dedup + async two-phase commit

Change `014-mnemonic-performance`, step `18-memory-types`. FRESH reviewer, no prior context.
Reviewed the single step-18 commit `d35b96a` (8 files, 1085 insertions / 20 deletions) via the
scoped diff plus the actual source. Re-ran the focused trio independently:

```
$ go test ./internal/mnemonic/memory/ -run 'TestMemoryTypeCategories|TestLLMDedupDetectsSemanticDuplicates|TestAsyncTwoPhaseCommit' -race -count=1
--- PASS: TestMemoryTypeCategories (2.64s)
--- PASS: TestLLMDedupDetectsSemanticDuplicates (5.27s)
    --- PASS: .../LLM_flags_semantic_duplicate_and_merges (2.52s)
    --- PASS: .../hash_fallback_when_LLM_unavailable (2.75s)
--- PASS: TestAsyncTwoPhaseCommit (2.01s)
```

All three pass **race-clean** under `-race -count=1`, independent of the implementer's report.

## Verdict

**PASS WITH WARNINGS**

## Findings

### F1 — minor — `candidateDedupResult` is dead code
`types.go:50`
```go
// candidateDedupResult is the outcome of the pre-write dedup check.
type candidateDedupResult struct {
    MergedInto int64
    Skipped bool
    Reason string
}
```
The struct is declared but **never constructed or read anywhere** (confirmed by grep: the only
non-comment reference is the declaration). `runDedupCheck` returns a plain `(int64, string)`
instead. It is unexported, so it does not pollute the package API and it does not break the build,
but it is unused scaffolding. Fix: delete the type, or actually return it from `runDedupCheck`.

### F2 — minor — `dedupCandidateID` ignores the `candidates` parameter
`types.go:95`
```go
func (s *Service) dedupCandidateID(ctx context.Context, llmID int64, candidates []string) int64 {
    if llmID > 0 { ... }
    // Fall back to the most recent candidate row (the head of the pre-filter).
    var id int64
    if err := s.store.DB.QueryRowContext(ctx,
        `SELECT id FROM observations WHERE project = ? AND deleted_at IS NULL ORDER BY created_at DESC, id DESC LIMIT 1`,
        s.projectID,
    ).Scan(&id); err == nil { return id }
    return 0
}
```
The `candidates []string` argument is never used in the body — the fallback re-queries the DB for
the most recent row rather than matching the LLM-flagged candidate. This is the **documented**
behavior (the report's "Concerns" acknowledges it: "The LLM dedup `duplicateID` ... is trusted when
positive; otherwise it falls back to the most-recent candidate row"). Consequence: if the LLM
returns `duplicateID=0` (or an id not in this project) on a true duplicate, the merge lands on the
*most recent* observation, which may be the wrong row. Not a correctness bug for the tested paths
(the mock returns `candidateID()==1`, and `llmID>0` is trusted), but it is an under-specified seam
contract. Fix: remove the unused parameter, or have the fallback intersect the LLM's choice with
`candidates`.

### F3 — minor — LLM error path in `runDedupCheck` returns `("hash")` that the caller discards
`types.go:143`
```go
dup, llmID, err := s.dedupLLM.Dedup(ctx, content, candidates)
if err != nil {
    fmt.Fprintf(logWriter(), "mnemonic: dedup LLM failed, falling back to hash: %v\n", err)
    return 0, "hash"
}
```
And in `service.go:443`:
```go
if mergeInto, _ := s.runDedupCheck(ctx, in.Content); mergeInto > 0 {
```
The `reason` return value (`"hash"`, `"llm"`, `""`) is computed by `runDedupCheck` but **always
discarded** by the caller (`_`). The LLM-failure branch returns `"hash"` to signal "fall back," but
nothing consumes that signal — the hash fallback is simply the next statement in `Save()` regardless.
The behavior is correct (on LLM error, `mergeInto==0`, so `Save()` proceeds to the existing
normalized-hash dedup, which is the intended fallback), but the `reason` channel is vestigial in the
synchronous path. Harmless; worth a comment or removal to avoid implying it is observed.

### F4 — nit — the `--type` CLI flag is a generic flag name scoped by subcommand
`mem.go:66`
```go
fs.StringVar(&memType, "type", "", "list: filter by fine-grained memory_type ...")
```
`--type` is parsed for *all* `mem` subcommands (the flag set is shared), but it is only honored in
`runMemList`. There is no other `mem` subcommand that already used `--type` (verified: the only
`"type"` flag in `mem.go` is this one), so no collision today. The risk is a future `mem` subcommand
that legitimately wants a coarse `--type`; the fine-grained `memory_type` filter would silently win.
Minor naming/precedent concern, not a defect in this step. The flag is read-only and additive.

### F5 — nit — async phase writes a single last-wins `memory_diff.json`
`types.go:336`
```go
path := filepath.Join(dir, "memory_diff.json")
```
Each `SessionCommit` overwrites the same `<dataDir>/<project>/memory_diff.json`, so only the most
recent commit's diff is auditable. The report acknowledges this ("overwrite-on-each-commit,
last-wins"). For an "audit" file this is lossy, but the brief only requires that `memory_diff.json`
be written after the async phase, which it is. Not a step-18 defect; noted for a future step.

## Evidence checked (8/8)

1. **memory_type vs type (18.1) — VERIFIED.** `memory_type` is a **separate** new nullable column;
   the existing coarse `type` column (`learning`, `decision`, ...) is untouched. Migration
   `028_memory_type.sql:24` is `ALTER TABLE observations ADD COLUMN memory_type TEXT;` (additive,
   nullable → NULL = not typed). All 9 constants are defined (`service.go` `MemoryTypeProfile`…
   `MemoryTypeExperiences`) backed by the `memoryTypes` set, the single source of truth. Validation is
   correct: `service.go:392` — `if in.MemoryType != "" && !IsValidMemoryType(...)` → empty is allowed
   (NULL, no validation), set-but-unknown is rejected **before any write** with a clear error naming
   the value. `mem list --type` filters correctly via `RecentWithType` (`service.go:743`): validates
   the category, then `WHERE deleted_at IS NULL AND project = ? AND memory_type = ?` — returns only
   typed rows of that category (NULL/other excluded). The test asserts a `--type preferences`
   equivalent returns exactly the 1 preference row.

2. **LLM dedup (18.2) — VERIFIED, DEDUP VERDICT: CORRECT.** `DedupLLM` (`types.go:37`) is an
   injectable interface mirroring step-05 `ExtractionLLM` (same seam pattern, no hard-coded HTTP):
   ```go
   type DedupLLM interface {
       Dedup(ctx context.Context, newContent string, candidates []string) (duplicate bool, duplicateID int64, err error)
   }
   ```
   Opt-in via `SetDedupLLM` + `EnableDedupLLM` (config `mnemonic.dedup.llm` analogue, default off —
   zero-value `dedupLLMEnabled=false`). The flow is pre-filter → LLM check → skip/merge:
   - **Pre-filter** `dedupPreFilter` (`types.go:64`) SQL-queries the project's non-deleted rows,
     most-recent-first, capped at 20 (`dedupPreFilterLimit`).
   - **LLM check** `runDedupCheck` (`types.go:127`) → `s.dedupLLM.Dedup(ctx, content, candidates)`.
   - **Skip/merge** `service.go:443`: `if mergeInto, _ := s.runDedupCheck(...); mergeInto > 0 { s.BumpDuplicate(ctx, mergeInto); return mergeInto, nil }` — on a duplicate verdict the second write is
     **merged** (bump `duplicate_count`, no new row, return the existing id), not a duplicate row.
   - **Hash fallback**: when the LLM is disabled/absent/errors, `runDedupCheck` returns `0` and
     `Save()` proceeds to the pre-existing 24h normalized-hash dedup (the default path is unchanged).
   - **Test uses a MOCK LLM** (`dedupLLM` struct, `types_test.go:991`) — no real call. It asserts the
     LLM was invoked (`llm.called != 0`), the semantic duplicate merges (`second == first`, exactly 1
     row, `duplicate_count==1`), and a genuinely different observation is **not** flagged
     (`llm.duplicate=false` → new row, 2 total). The hash-fallback subtest (no LLM) confirms an exact
     duplicate is caught and a reworded one is not. **The dedup logic is sound and matches the brief.**

3. **Async two-phase commit (18.3) — VERIFIED, DURABILITY VERDICT: DURABLE, CORRECT.** `SessionCommit`
   (`types.go:170`) splits cleanly:
   - **SYNC phase** (`types.go:183-220`) runs to completion **before** the async goroutine is spawned:
     `UPDATE sessions SET summary=... title=...` (write message), verify the session exists, then
     `UPDATE sessions SET compression_index = COALESCE(compression_index,0)+1`, then **read the
     `compression_index` back** and return it. Every statement is a synchronous `*sql.DB` call with
     its error checked — with `modernc.org/sqlite` (no CGO, no deferred flush) these are durably
     committed at the point the call returns. **The DB write is committed before `SessionCommit`
     returns; it is NOT deferred to the async phase.** The brief's durability guarantee is met.
   - **ASYNC phase** (`types.go:228-236`) is a **detached goroutine** (`go func(){ s.asyncExtractAndDiff(...); wg.Done() }()`),
     armed only when `extractionLLMEnabled && extractionLLM != nil`. It runs LLM extraction → dedup
     each item → writes `memory_diff.json` (`writeMemoryDiff`, `types.go:318`). It **never blocks**
     the sync return (the `go` is issued, then `return compressionIndex, nil`) and never fails the
     commit.
   - **Test waits via `sync.WaitGroup`, not `time.Sleep`** (`types_test.go:1209-1273`):
     `wg.Add(2)` for two commits, `SetAsyncCommitGate(&wg)` (a package-level test seam, `types.go:373`),
     two `SessionCommit` calls (returns promptly, `ci==1` then `ci==2`), then `wg.Wait()`. It asserts
     the sync increment is **durable** (read `compression_index` straight from the store → 1 then 2),
     the session message is durable, the LLM ran in the background (`callCount >= 2`), and
     `memory_diff.json` exists with the extracted item + correct `session_id`. No `time.Sleep` anywhere.
   - **Race safety (check 8) — race-clean.** Re-ran with `-race`: clean. The shared `*Service` fields
     read by the async goroutine (`projectID`, `store`, `extractionLLM`, `dedupLLM`, `dedupLLMEnabled`)
     are set by the test **before** the goroutine is spawned, so the goroutine's read happens-after the
     setter's write via the `go` statement's synchronization — no unsynchronized concurrent access.
     `asyncExtractAndDiff`/`writeMemoryDiff`/`isExtractedDuplicate` do not mutate `Service` fields;
     the `extractionLLMStub` counter is guarded by its own mutex (`types_test.go:1158`). The gate's
     `Add`/`Done`/`Wait` is mutex-guarded at the seams. The report's "race-clean" claim is **credible
     and independently confirmed**.

4. **Migration (18.1) — VERIFIED, purely additive.** `028_memory_type.sql`:
   `ALTER TABLE observations ADD COLUMN memory_type TEXT;` (nullable, NULL = not typed) + a partial
   index on `(project, memory_type) WHERE memory_type IS NOT NULL AND deleted_at IS NULL`, plus
   `ALTER TABLE sessions ADD COLUMN compression_index INTEGER NOT NULL DEFAULT 0;` for the two-phase
   commit. Both are `ALTER TABLE ADD COLUMN` / `CREATE INDEX IF NOT EXISTS` — additive only; no
   existing column is dropped, renamed, or re-typed. Pre-028 rows read back as `memory_type` NULL and
   `compression_index` 0, so untyped saves and existing sessions keep working unchanged.

5. **Additive-ness / tool contract — VERIFIED.** No `mem_*`/`code_*`/`web_*` MCP tool signature
   changed. The `Observation` struct gains an **additive** `MemoryType string \`json:"memory_type,omitempty"\``
   field (`service.go:240`) — an `omitempty` field addition does not change the shape of any existing
   tool contract. `SaveInput` gains an optional `MemoryType` field; `SaveInput.MemoryType` defaults
   empty, so the default `Save()` path (no `memory_type`, dedup disabled) is byte-for-byte the same
   behavior as before: validation is skipped (`in.MemoryType==""`), `runDedupCheck` returns 0
   (`dedupLLMEnabled==false`), and the INSERT's new `memory_type` column binds NULL via
   `nullString("")` (`service.go:1904` returns `sql.NullString{}` for empty). The existing coarse
   `type` column and `IsValidType` are untouched; both columns coexist.

6. **Small changes to crosslink.go / governance.go / improve_test.go — VERIFIED safe.** Each is a
   one-line SELECT change appending `o.memory_type` (or `o2.memory_type`) to the column list — they
   are required because `obsSelectCols` / the hand-written SELECTs now select the new column, so the
   corresponding `scanObservations`/`Scan` expects it. Without these, the `scanObservations` (which
   now scans `&memType` last) would mismatch the column count and fail. They are read-only SELECT
   additions; no logic change. `improve_test.go` is the test-side mirror for the same reason. Safe and
   necessary.

7. **TDD discipline — VERIFIED.** The three tests map 1:1 to the brief's RED sub-tasks:
   (a) `TestMemoryTypeCategories` → 9 types stored + round-tripped (raw column + `Get`), untyped
       reads back empty (rollback boundary), `RecentWithType("preferences")` returns only that type,
       invalid `"quantum"` rejected with a clear error **and no row written** (row count asserted
       before/after). (b) `TestLLMDedupDetectsSemanticDuplicates` → semantic duplicate flagged +
       merged (no new row, `duplicate_count` bumped), different observation not flagged, hash
       fallback catches exact / not reworded. (c) `TestAsyncTwoPhaseCommit` → sync durable (read back
       from store), async non-blocking (returns before `wg.Wait()`), LLM ran in background,
       `memory_diff.json` written with correct session id, WaitGroup gate (no sleep). All RED
       evidence is build-failure (new symbols absent) for 18.1/18.2 and a runtime session-not-found for
       18.3, consistent with the report.

8. **Race safety — VERIFIED** (see check 3). Independently re-ran the trio with `-race -count=1`:
   clean. The report's claim is credible.

## Brief-vs-implementation gap (not a defect in scope, noted)

The brief lists a **18.4 `[AFK]`** sub-task (`TestMemoryTypeFallbackToAutoClassification` — fall back
to keyword-based auto-classification when the LLM type assignment fails). The implementation and the
report do **not** include 18.4: there is no `TestMemoryTypeFallbackToAutoClassification` and no
keyword-based type-inference path in the commit. The report frames the step as "the three RED
sub-tasks" (18.1/18.2/18.3) and 18.4 is marked `[AFK]` (asynchronous/optional). This matches the
report's own scope statement, so it is consistent — but if 18.4 is considered part of this step's DoD
("All `### Tasks` checkboxes `[x]`"), it is **not** done. Flagging for the orchestrator to confirm
whether 18.4 was intentionally deferred; it does not block 18.1/18.2/18.3.

## Global Constraints

- **All changes additive** — yes (new nullable column, new NOT-NULL-DEFAULT-0 column, additive
  `omitempty` struct field, additive CLI subcommands/flag).
- **`mem_*`/`code_*`/`web_*` tool contracts unchanged in shape** — yes (verified check 5).
- **modernc stays, no CGO** — yes (no new deps; `types.go` imports only stdlib + the existing
  `database/sql`/`store`).
- **LLM is opt-in (injectable)** — yes (`DedupLLM` + `EnableDedupLLM` default off; async extraction
  reuses the step-05 opt-in `ExtractionLLM` gate).

## Approved to merge

**Yes** — with the minor findings above (F1–F5) as follow-ups. The three load-bearing guarantees
(18.1 typed categories + `--type` filter, 18.2 opt-in LLM dedup with hash fallback, 18.3 durable
sync / non-blocking async two-phase commit) are all implemented correctly and race-clean under
`-race`. The dead `candidateDedupResult` type (F1) is the only thing I would fix before considering
this "clean," but it does not affect behavior.
