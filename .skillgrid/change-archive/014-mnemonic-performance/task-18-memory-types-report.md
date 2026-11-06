# Task 18 — Typed memory categories + LLM dedup + async two-phase commit

Change `014-mnemonic-performance`, step `18-memory-types`. Branch `feat/v2-structure`.

## Summary

Implemented the three RED sub-tasks additively: a new nullable `memory_type`
column with 9 fine-grained typed categories (18.1), an opt-in LLM semantic-dedup
pre-write check with a hash fallback (18.2), and a two-phase session commit
whose sync phase is durable before return and whose async phase (LLM extraction
+ dedup + `memory_diff.json`) runs in a detached, testable goroutine (18.3).

The LLM is OPT-IN throughout (follows the step-05 `ExtractionLLM` seam pattern:
an injectable interface, not a hard-coded HTTP call). No `mem_*`/`code_*`/`web_*`
tool contract changed shape; `modernc.org/sqlite` + no CGO unchanged; all schema
changes are additive (new nullable column + new NOT-NULL-DEFAULT-0 column).

## Per sub-task

### 18.1 — `memory_type` column supports 9 typed categories

- **Migration `028_memory_type.sql`** adds a nullable `memory_type TEXT` column
  to `observations` + a partial index for the typed-list read path. (Also adds
  `sessions.compression_index` for 18.3 — see below.)
- **9 type constants** (in `memory/service.go`), exported so the CLI and async
  extractor reference canonical spellings:
  - `MemoryTypeProfile` = `profile`
  - `MemoryTypePreferences` = `preferences`
  - `MemoryTypeEntities` = `entities`
  - `MemoryTypeEvents` = `events`
  - `MemoryTypeIdentity` = `identity`
  - `MemoryTypeSoul` = `soul`
  - `MemoryTypeCases` = `cases`
  - `MemoryTypeTrajectories` = `trajectories`
  - `MemoryTypeExperiences` = `experiences`
- `IsValidMemoryType` (case-insensitive) + `memoryTypes` set are the single
  source of truth for validation.
- `SaveInput.MemoryType` (optional) is validated on save: empty → column stays
  NULL (not typed, no validation); set-but-unknown → rejected with a clear
  error naming the value (`invalid memory_type "quantum" (allowed: ...)`)
  before any write.
- Stored on the INSERT (`nullString(memoryType)`) and read back through
  `obsSelectCols`/`scanObservations`/`Get` (additive `omitempty` JSON field
  `memory_type`). The existing coarse `type` column is untouched — the two
  coexist.
- `Service.RecentWithType(ctx, category, limit)` is the read path behind
  `mem list --type`; it validates the category and returns only typed rows
  matching it (NULL / other-category rows excluded).
- **CLI** (`cmd/skillgrid/mem.go`, manual dispatcher): added a `list`
  subcommand honoring `--type <category>` (`runMemList`) and a `memory-type`
  subcommand (`runMemMemoryType`) that lists the 9 valid categories. `--type`
  flag added; usage text updated.

### 18.2 — LLM dedup detects semantic duplicates before write

- New injectable seam `DedupLLM` (in `memory/types.go`):
  `Dedup(ctx, newContent string, candidates []string) (duplicate bool, duplicateID int64, err error)`.
  Mirrors the step-05 `ExtractionLLM` pattern — a small interface, no hard-coded
  HTTP client.
- Opt-in via `SetDedupLLM` + `EnableDedupLLM` (config `mnemonic.dedup.llm`
  analogue; default off). A seam present but unenabled keeps `Save()` on the
  hash fallback.
- **Flow** (pre-write, in `Save()` after the existing 24h hash-dedup lookup):
  1. **Pre-filter** — `dedupPreFilter` SQL-queries the project's non-deleted
     observations (most recent first, capped at 20) as the candidate contents.
     This is the cheap stand-in for vector pre-filtering (no embeddings
     required).
  2. **LLM check** — `runDedupCheck` hands the new content + candidates to the
     seam and asks "is this a semantic duplicate of any of these?".
  3. **Skip/merge** — on a duplicate verdict the save is **merged** into the
     existing observation (`BumpDuplicate` on the flagged id, no new row, return
     that id). On a non-duplicate or no-candidates, the save proceeds exactly as
     before.
- **Hash fallback** — when the LLM is disabled, absent, or returns an error,
  `runDedupCheck` returns 0 and the save proceeds to the existing
  normalized-hash dedup (exact content match) — the default path is unchanged.
  An LLM error is logged to stderr and never propagated (mirrors the step-05
  LLM-failure fallback).

### 18.3 — Async two-phase commit: sync write + async LLM extraction

- Migration adds `sessions.compression_index INTEGER NOT NULL DEFAULT 0` (the
  per-session "commit" counter).
- `Service.SessionCommit(ctx, sessionID, summary)`:
  - **SYNC phase (durable, must complete before return):** writes the session
    message (the close summary → `sessions.summary` + title) and increments
    `sessions.compression_index` by 1. Both must succeed before return; the
    returned `compressionIndex` is read back from the store (durable).
  - **ASYNC phase (non-blocking, best-effort):** when the LLM extraction pass is
    enabled + a seam is attached, a **detached goroutine** runs
    `asyncExtractAndDiff` (LLM extraction → dedup each item → write
    `memory_diff.json`). It never blocks the sync return and never fails the
    commit.
- **`memory_diff.json`** is written to the project's data directory
  (`<dataDir>/<project>/memory_diff.json`) by `writeMemoryDiff`. It records the
  session id, project, timestamp, each extracted item (title/type), whether it
  was deduped (`duplicate`), and the `merged_into` id; an extraction error is
  captured in an `error` field. Best-effort: a write failure is logged to
  stderr, never returned.
- **Testable async phase** (no `time.Sleep`): a package-level test seam
  `SetAsyncCommitGate(*sync.WaitGroup)` installs a WaitGroup the async phase
  signals (one `Done()` per spawned goroutine). The test `wg.Add(2)` (two
  commits), calls both commits, then blocks on `wg.Wait()` — a deterministic
  channel/WaitGroup wait, not a sleep. The test asserts: sync return is prompt,
  `compression_index` is incremented + durable (1 then 2), the session message
  is durable, the LLM was invoked in the background (`callCount >= 2`), and
  `memory_diff.json` exists with the extracted item + correct session id.

## Migration 028 (quoted)

```sql
-- 028: fine-grained typed memory categories (change 014, step 18).
--
-- Additive: one new nullable TEXT column on observations. No existing schema
-- is touched; the existing `type` column (coarse taxonomy: learning, decision,
-- bug, session_log, ...) is left untouched. pre-028 rows read back with
-- memory_type NULL (not typed), so untyped saves keep working unchanged.
--
--   memory_type — one of the 9 typed categories (step 18):
--                  profile, preferences, entities, events, identity, soul,
--                  cases, trajectories, experiences.
--                NULL = the observation was not given a fine-grained type.
--                Validation (rejecting unknown values) happens in Go, not the
--                schema, so the column stays nullable and pre-existing rows
--                are never invalid.
--
-- The partial index lets `mem list --type <cat>` (the RecentWithType read
-- path) seek straight to typed rows without scanning the whole table. NULL
-- memory_type rows are excluded, matching "only typed observations".

ALTER TABLE observations ADD COLUMN memory_type TEXT;

CREATE INDEX IF NOT EXISTS idx_obs_memory_type
    ON observations (project, memory_type)
    WHERE memory_type IS NOT NULL AND deleted_at IS NULL;

-- Async two-phase session commit (step 18.3): compression_index is the per-
-- session "commit" counter the SYNC phase of SessionCommit increments. It is
-- additive (NOT NULL DEFAULT 0) so pre-028 sessions read back with 0 and the
-- commit path can advance it without a backfill.
ALTER TABLE sessions ADD COLUMN compression_index INTEGER NOT NULL DEFAULT 0;
```

## RED → GREEN evidence

- **18.1** `TestMemoryTypeCategories`: RED — build failure (`unknown field
  MemoryType`, `RecentWithType undefined`). GREEN after adding the column,
  constants, validation, storage, `RecentWithType`, and CLI.
- **18.2** `TestLLMDedupDetectsSemanticDuplicates`: RED — build failure
  (`SetDedupLLM`/`EnableDedupLLM undefined`). GREEN after the `DedupLLM` seam,
  pre-filter, `runDedupCheck`, and the `Save()` merge branch.
- **18.3** `TestAsyncTwoPhaseCommit`: RED — runtime failure
  (`SessionCommit: session ... not found` → fixed by creating the session row
  in the fixture project bucket; before `compression_index`/`SessionCommit`
  existed it would not compile). GREEN after the two-phase `SessionCommit`,
  the async goroutine, `memory_diff.json`, and the WaitGroup gate.

All three focused tests PASS (and race-clean under `-race`).

## Test results

### Focused trio
```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -run 'TestMemoryTypeCategories|TestLLMDedupDetectsSemanticDuplicates|TestAsyncTwoPhaseCommit'
--- PASS: TestMemoryTypeCategories (0.43s)
--- PASS: TestLLMDedupDetectsSemanticDuplicates (0.72s)
--- PASS: TestAsyncTwoPhaseCommit (0.18s)
PASS
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  1.988s
```

### Full relevant suite
```
$ go test ./skillgrid-cli/internal/mnemonic/memory/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory  20.017s

$ go test ./skillgrid-cli/cmd/skillgrid/ -count=1
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid  75.014s
```

### 005/008/010/011/013 baselines
```
$ go test ./skillgrid-cli/internal/mnemonic/store/... ./skillgrid-cli/internal/mnemonic/service/... \
          ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/route/... \
          ./skillgrid-cli/internal/mnemonic/affected/... ./skillgrid-cli/internal/mnemonic/community/... \
          ./skillgrid-cli/internal/mnemonic/pdg/... -count=1
ok   .../internal/mnemonic/store       5.891s
FAIL .../internal/mnemonic/service     53.549s   <- TestReindexStructuralIsEmbedderFree (pre-existing)
ok   .../internal/mnemonic/codeindex   17.406s
ok   .../internal/mnemonic/route       0.740s
ok   .../internal/mnemonic/affected    12.640s
ok   .../internal/mnemonic/community   5.465s
ok   .../internal/mnemonic/pdg         6.816s
```

**Pre-existing failure (not caused by this step):** `TestReindexStructuralIsEmbedderFree`
in `internal/mnemonic/service` fails with `count gamma: sql: database is closed`.
It fails identically on clean `HEAD` (verified by `git stash` of all step-18
changes → still FAILS), and it reproduces consistently in isolation and with
`-count=2`. It is a store-pooling/teardown flake in a step-01 reindex test,
orthogonal to the additive memory_type/dedup/commit changes (which touch only
`memory/` + one migration + the `mem.go` CLI). No service-package behavior
changed by this step.

## Files changed

- `skillgrid-cli/internal/mnemonic/store/migrations/028_memory_type.sql` — NEW
- `skillgrid-cli/internal/mnemonic/memory/types.go` — NEW (DedupLLM seam,
  pre-filter, runDedupCheck, SessionCommit two-phase, asyncExtractAndDiff,
  writeMemoryDiff, memoryDiff, SetAsyncCommitGate)
- `skillgrid-cli/internal/mnemonic/memory/types_test.go` — NEW (3 tests)
- `skillgrid-cli/internal/mnemonic/memory/service.go` — MODIFIED (MemoryType on
  SaveInput/Observation, 9 constants + IsValidMemoryType, dedupLLM fields +
  setters, Save() validation + memory_type INSERT + dedup merge branch,
  obsSelectCols/scanObservations/Get + RecentWithType, SearchWithScope/
  SearchOwnerScoped SELECTs)
- `skillgrid-cli/internal/mnemonic/memory/governance.go` — MODIFIED (2 SELECTs
  → +o.memory_type)
- `skillgrid-cli/internal/mnemonic/memory/crosslink.go` — MODIFIED (SELECT →
  +o2.memory_type)
- `skillgrid-cli/internal/mnemonic/memory/improve_test.go` — MODIFIED (test
  SELECT → +o.memory_type)
- `skillgrid-cli/cmd/skillgrid/mem.go` — MODIFIED (`list` + `memory-type`
  subcommands, `--type` flag, usage)

## Concerns

- `compression_index` is incremented on `SessionCommit` (the new two-phase
  entry point) rather than on `SessionEnd`, since the brief frames "session
  commit" as its own sync+async path. `SessionEnd` behavior is unchanged.
- The LLM dedup `duplicateID` returned by the seam is trusted when positive;
  otherwise it falls back to the most-recent candidate row. A richer seam
  contract (per-candidate id) would be more precise but is beyond the minimal
  step scope.
- `memory_diff.json` is written to `<dataDir>/<project>/memory_diff.json`
  (best-effort; overwrite-on-each-commit, last-wins). If per-session audit
  files are desired later, the path can key on `sessionID`.
