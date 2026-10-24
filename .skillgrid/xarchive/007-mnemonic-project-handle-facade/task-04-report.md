# Task 04 Report — collapse double-open Service wrappers

**Change:** 007-mnemonic-project-handle-facade, step 04 (`04-collapse-wrappers`)
**Commit:** `89cd2ca` `refactor(mnemonic): collapse double-open Service wrappers`
**Status:** DONE

## Summary

Steps 02/03 already rewired MCP and HTTP to call domain modules through
`*service.ProjectHandle` (`h.Memory()` / `h.Web()` / `h.Store()`). As a result
the wide open-delegate-close facade in `service/service.go` had no production
callers for ~43 single-project methods. This step deleted those dead wrappers
(+ one test-only helper), rewire the affected tests to the handle, restored one
type alias that production code still needs, and added a double-open regression
guard. Net diff: **-589 / +199 lines** across 6 files.

## Methods deleted (all in `skillgrid-cli/internal/mnemonic/service/service.go`)

Each was a shallow `openProject(projectID, ".")` → delegate → `close` wrapper
with no production caller after 02/03. Line numbers are pre-deletion.

**No production AND no test callers (21):**
`ObservationsRecent`, `ContextForCompaction`, `GetObservation`,
`SetObservationEmbedding`, `SearchAllProjectsInProjects`, `PinObservation`,
`UnpinObservation`, `TTLPending`, `UpdateObservation`, `DeleteObservation`,
`ObservationTimeline`, `MarkReviewReviewed`, `SetObservationReviewAfter`,
`MemoryDoctor`, `SymbolSearch`, `ReadIndexedCode`, `CodeFairCoverage`, `WebGet`,
`ListProjectsForMigration`, `SessionStartedAt`.
(Plus `CountLongTermMemories` in `compaction.go` — test-only helper.)

**Dead but had test-only callers (22) — method deleted AND test rewired:**
`SessionSetTitle`, `SessionEnd`, `SessionSummary`, `SavePrompt`, `CapturePassive`,
`LastObservationAt`, `RecordRelation`, `RemoveRelation`, `RelationsOf`,
`RelationsBetween`, `RecentContext`, `BlendedSearch`, `TTLRetire`, `SaveObservation`,
`CodeStatus`, `CodeSearch`, `CodeFiles`, `WebLookup`, `WebSave`, `WebSearch`,
`MemoryStatus`, `WebCacheStatus`, and `SearchObservations` (delegator).

### Type aliases deleted
- `type PassiveResult = memory.CapturePassiveResult` (only the `CapturePassive`
  method used it)
- `type CompactionContext = memory.CompactionContext` (only `ContextForCompaction`)
- `type SymbolHitDTO = search.SymbolHit` (only `SymbolSearch`)
- `type FairCoverageDTO = graph.CoverageLang` (only `CodeFairCoverage`)

### Type alias RESTORED (still referenced by production)
- `type PassiveInput = memory.PassiveInput` — referenced by
  `http/server.go:417` (`handleObservationPassive`). The previous attempt deleted
  it together with the `CapturePassive` method, breaking the build. Restored at
  `service.go:187` (keeps `http` unchanged, least invasive).

## Tests rewired (old facade call → new handle call)

All rewired to `svc.Open(projectID)` → `h.Memory()` / `h.Web()` / `h.Store().DB`,
preserving the normalization the old facade applied.

| File | Old call | New call |
|------|----------|----------|
| `service/orient_test.go:38` | `svc.SymbolSearch(ctx, projectID, "parseConfig", 10)` | `h, cleanup := svc.Open(projectID)`; `search.SymbolFTS(h.Store().DB, "parseConfig", 10)` |
| `service/compaction_test.go:87,94,116` | `svc.CountLongTermMemories(ctx, project)` | local `countLongTermMemories(t, svc, project)` helper → `svc.Open` + `h.Store().DB.QueryRow(COUNT long_term_memories)` |
| `service/compaction_test.go:91` | `svc.SessionEnd(ctx, project, "s1", "unsaved lessons")` | `h := svc.Open(project)`; `h.Memory().SessionEnd(ctx, "s1", "unsaved lessons")`; `cleanup()` |
| `integration/seed_test.go:79` | `svc.SaveObservation(ctx, projectID, service.SaveObservationInput{...})` | `saveObservation(t, ctx, svc, projectID, memory.SaveInput{...})` helper (applies scope `""→project`, `personal→user`, then `h.Memory().Save`) |
| `integration/seed_test.go:84` | `svc.MemoryStatus(ctx, projectID)` | `handleFor(t, svc, projectID).Memory().Status(ctx)` |
| `integration/seed_test.go:103` | `svc.SearchObservations(ctx, projectID, q, "any", 10)` | `handleFor(...).Memory().SearchWithScope(ctx, q, "any", "", 10)` |
| `integration/seed_test.go:137` | `svc.CodeStatus(ctx, projectID)` | `codeindex.GetStatus(h.Store())` + stale recompute |
| `integration/seed_test.go:148` | `svc.CodeFiles(ctx, projectID)` | inline `h.Store().DB.Query("SELECT path FROM files ORDER BY path")` |
| `integration/seed_test.go:156` | `svc.CodeSearch(ctx, projectID, "Greet", 5)` | `search.CodeSearch(h.Store().DB, "Greet", 5)` |
| `integration/seed_test.go:181` | `svc.WebSave(ctx, seedProject, in)` | `handleFor(...).Web().Save(ctx, in)` |
| `integration/seed_test.go:186` | `svc.WebCacheStatus(ctx, seedProject)` | `handleFor(...).Web().CacheStatus(ctx)` |
| `integration/seed_test.go:202` | `svc.WebLookup(ctx, seedProject, in)` | `handleFor(...).Web().Lookup(ctx, in)` |
| `integration/seed_test.go:212` | `svc.WebSearch(ctx, seedProject, ...)` | `handleFor(...).Web().Search(ctx, ...)` |
| `integration/seed_test.go:240` | `svc.SaveObservation(...)` | `saveObservation(t, ctx, svc, projectID, memory.SaveInput{...})` |
| `integration/seed_test.go:252` | `svc.WebSave(ctx, projectID, in)` | `handleFor(...).Web().Save(ctx, in)` |
| `mcp/single_open_test.go:121` | `(service.New(dataDir)).SaveObservation(ctx, "shape-probe", ...)` | `seedObservation(t, dataDir, "shape-probe", memory.SaveInput{...})` helper (applies scope normalization, `h.Memory().Save`) |
| `mcp/single_open_test.go:175` | `(service.New(dataDir)).SaveObservation(ctx, "shape-search", ...)` | `seedObservation(t, dataDir, "shape-search", memory.SaveInput{...})` |

**Normalization preserved:** `SaveObservation` normalized `scope: ""→project` and
`personal→user` before delegating to `memory.Save`. Both new test helpers
(`saveObservation` in seed_test.go and `seedObservation` in single_open_test.go)
reproduce exactly that. The `SaveObservationInput` struct was kept (still used by
`http/server.go:844` and by the seed loop's field list); only the method was
removed.

**Note on `memory/*_test.go`:** the survey listed `memory/capture_test.go`,
`service_test.go`, `embed_gap_test.go`, `identity_features_test.go` as candidates,
but on inspection those tests already call `*memory.Service` directly
(`fx.svc` / `svc` is a `*memory.Service`, not `*service.Service`). They required
no rewiring — confirmed by `rg "service\." internal/mnemonic/memory/*_test.go`
returning nothing and the `memory` package tests passing.

## Methods KEPT (and why — non-obvious ones)

- `Open` / `OpenForCWD` / `OpenForDirectory` — the new public handle seam (MCP/HTTP entry).
- `MnemonicCommit` — MCP production caller (`tools_compaction.go`).
- `SemanticSearch` / `LoadFullDetails` — MCP production callers (`retrieval.go`).
- `SessionStart` / `SessionStartByClientID` — MCP + HTTP + test production callers.
- `ListReviews` — NOT in the survey's delete list; has live MCP/HTTP callers
  (`tools_memory.go:731`, `http/server.go:597`). Kept.
- `SearchObservationsScoped` — the leaf helper that cross-store root
  `SearchObservationsAll` calls (per caveat). Kept (not inlined).
- `CodeExplore` (KEEP, CLI caller) internally calls the also-KEPT `CodeImpact`;
  both still exist, no double-open introduced.
- `SaveObservationInput` struct + `PassiveInput` alias — kept/restored because
  `http/server.go` references them (DTO/alias stability, no route change).

## Double-open regression test

New file `service/double_open_test.go`:

- **`TestServiceFacadePathOpensOnce`** — opens a project through the public
  `Service.Open` seam (the production path MCP/HTTP use) and asserts
  `store.OpenCount() == 1` for that logical op. Pins the invariant: any future
  wrapper that re-opens the project inside `Open` pushes the counter to 2 and
  fails.
- **`TestMnemonicCommitOpensOnce`** — exercises `MnemonicCommit` (a KEEP method
  with a live MCP caller that internally opens the project) and snapshots the
  counter immediately after the synchronous commit returns, asserting `== 1`.
  The async tiering hook reopens the store in a background goroutine (a separate
  logical op, not a double-open of the commit); the test snapshots before that
  goroutine starts and then awaits the wait group before exiting so TempDir
  cleanup is not racy.

Both use the step-02 mutex-guarded counter `store.OpenCount()` /
`store.ResetOpenCount()` (incremented by `store.Open`).

## TDD evidence (04.1)

**RED (written before the collapse; the collapse was already applied in the
prior partial attempt, so the meaningful RED/GREEN pair is compile-fix → green):**

Command:
```
go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1
```

- Initial RED (compile): the new `double_open_test.go` did not build against the
  in-progress tree until the `PassiveInput` alias was restored and the
  `cleanup()` no-value fix applied →
  ```
  internal/mnemonic/service/double_open_test.go:32:12: cleanup() (no value) used as value
  FAIL ... [build failed]
  ```
  Expected: the guard test + the alias it depends on had to exist before the
  invariant could be pinned.
- A second RED surfaced while finalizing `TestMnemonicCommitOpensOnce`:
  ```
  --- FAIL: TestMnemonicCommitOpensOnce
      testing.go:1464: TempDir RemoveAll cleanup: unlinkat .../001: directory not empty
  warn: tiered hook: read L2: ... no such file or directory
  ```
  Root cause: the tiering hook goroutine outlived the test (no wait group).
  Fix: pass a real `sync.WaitGroup`, snapshot the counter immediately, then await
  the hook. This is the genuine RED→GREEN for the commit guard.

**GREEN:**
```
$ go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ -run 'DoubleOpen|OpenCounter|Once' -count=1
--- PASS: TestServiceFacadePathOpensOnce (0.09s)
--- PASS: TestMnemonicCommitOpensOnce (0.10s)
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service  1.755s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp      (no matching tests beyond the package build)
```

## Test results (from `skillgrid-cli`)

| Check | Command | Result |
|-------|---------|--------|
| Build | `go build ./...` | PASS |
| service + mcp + http + integration | `go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/ ./internal/mnemonic/integration/ -count=1` | PASS (all 4 `ok`) |
| Vet | `go vet ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ ./internal/mnemonic/http/` | PASS |
| **Full suite** | `go test ./... -count=1` | PASS (all 19 packages `ok`, 0 FAIL) |
| Focused 04.1 | `go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ -run 'DoubleOpen\|OpenCounter\|Once' -count=1` | PASS |

## Files changed

- `skillgrid-cli/internal/mnemonic/service/service.go` — deleted 43 dead wrappers
  + 4 type aliases; restored `PassiveInput` alias.
- `skillgrid-cli/internal/mnemonic/service/double_open_test.go` — NEW regression guard.
- `skillgrid-cli/internal/mnemonic/service/orient_test.go` — `SymbolSearch` → `search.SymbolFTS` via handle.
- `skillgrid-cli/internal/mnemonic/service/compaction_test.go` — `CountLongTermMemories`/`SessionEnd` → handle + local helper.
- `skillgrid-cli/internal/mnemonic/integration/seed_test.go` — rewired 11 facade calls to the handle + 2 helpers.
- `skillgrid-cli/internal/mnemonic/mcp/single_open_test.go` — 2 `SaveObservation` calls → `seedObservation` helper.

No production `mcp/` or `http/` code was edited (except the alias usage which was
satisfied by restoring the alias in `service.go`).

## Self-review findings

- Confirmed every KEEP method from the brief still present via
  `rg "^func (s \*Service)"` — nothing accidentally deleted.
- Confirmed `SearchObservationsAll` (root) still calls `SearchObservationsScoped`
  (leaf) — caveat honored, leaf kept.
- Confirmed `MemoryDoctor` struct kept (MCP `tools_memory.go:657` + HTTP
  `server.go:1291` build their own via `service.MemoryDoctor{}`); only the method
  was removed.
- Confirmed the `PassiveResult`/`CompactionContext`/`SymbolHitDTO`/`FairCoverageDTO`
  aliases have no remaining references after deletion (safe to drop).
- Verified no `service.` references remain in `memory/*_test.go` (those tests were
  already on `*memory.Service`).

## Issues / concerns

- **None blocking.** The only surprise vs. the brief was that the survey's
  "dead but test-only caller" list for `memory/*_test.go` required no changes —
  those tests were already exercising `*memory.Service` directly. The brief's
  rewire template still applied to `service/`, `mcp/`, and `integration/` tests.
- The `CodeFiles` behavior (ordered file list) was inlined as a query in
  `seed_test.go` rather than re-exposed as a handle method, matching the facade's
  original SQL exactly (no new SQL, no schema change).
