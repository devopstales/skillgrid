# Task 03 Report — HTTP single-open via Project Handle

## What was implemented

Every single-project HTTP handler in `skillgrid-cli/internal/mnemonic/http/server.go`
now opens the project **once** per request via a `*service.ProjectHandle`
(`s.svc.Open(projectID)` / `s.svc.OpenForDirectory(dir)`) and performs the domain
op through `h.Memory()` / `h.Web()` / `h.Store()`, instead of calling a
re-opening facade method on the root `*service.Service`. This matches the MCP
single-open lifecycle from step 02.

Two small helpers were added on `*Server`:
- `openHandleFor(projectID)` → `s.svc.Open(projectID)` (config root `"."`).
- `openHandleForDir(directory)` → `s.svc.OpenForDirectory(directory)`.

### Handlers rewired (single-project → handle)

Memory:
- `handleObservationCreate` — `svc.SaveObservation` → `h.Memory().Save` with
  reproduced scope normalization (see below).
- `handleObservationPassive` → `h.Memory().CapturePassive`.
- `handleObservationUpdate` → `h.Memory().Update`.
- `handleObservationDelete` → `h.Memory().Delete`.
- `handleObservationsRecent` → `h.Memory().RecentContext`.
- `handleObservationsList` → `h.Memory().Recent`.
- `handleMemoryTimeline` → `h.Memory().Timeline`.
- `handleMemoryReviews` → `h.Memory().ListReviews`.
- `handleMemoryReviewMark` → `h.Memory().MarkReviewed`.
- `handleRelationCreate` → `h.Memory().RecordRelation`.
- `handleRelationRemove` → `h.Memory().RemoveRelation`.
- `handleRelationsOf` → `h.Memory().RelationsOf`.
- `handleRelationsBetween` → `h.Memory().RelationsBetween`.
- `handleMemoryDoctor` → `httpMemoryDoctor(ctx, h, projectID)` (reproduced, below).
- `handleMemoryStatus` → `h.Memory().Status`.
- `handleMemoryLastSaveAt` → `h.Memory().LastObservationAt`.

Session (project-scoped):
- `handleSessionEnd` → `h.Memory().SessionEnd`.
- `handleSessionSetTitle` → `h.Memory().SessionSetTitle`.
- `handleSessionGet` (the `project != ""` branch) → `h.Memory().SessionStartedAt`.

Context / prompt / search:
- `handleContext` → `h.Memory().RecentContext`.
- `handleContextCompaction` → `h.Memory().CompactionContext`.
- `handlePromptCreate` → `h.Memory().SavePrompt` (keeps the
  `memory.ErrPromptTooSmall` → 202 `captured:false` branch).
- `handleSearch` → `h.Memory().SearchWithScope(ctx, query, matchMode, "", limit)`
  (scope `""` = any, matching `svc.SearchObservations` which delegates to the
  scoped variant).

Code:
- `handleCodeStatus` → `codeindex.GetStatus(h.Store())` + the same stale rule
  (`FileCount==0 || LastIndexed==""`).
- `handleCodeFiles` → `SELECT path FROM files ORDER BY path` on `h.Store().DB`
  (reproduced inline; byte-compatible).
- `handleCodeSearch` → `search.CodeSearch(h.Store().DB, query, limit)`.
- `handleCodeRead` → `readIndexedCode(h.Store().DB, path, startLine, endLine)`
  (reproduced helper, below).
- `handleCodeIndex` (directory-rooted) → `openHandleForDir(dir)` +
  `codeindex.New(h.Store()).Run(...)`, reproducing `service.RunCodeIndex`
  (config from the directory, embedder from `config.Load(dir)`, store from the
  handle). See "Left on the root service" for the one caveat.

Web:
- `handleWebLookup` → `h.Web().Lookup`.
- `handleWebCacheSave` → `h.Web().Save`.
- `handleWebSearch` → `h.Web().Search`.
- `handleWebEntry` → `h.Web().Get`.
- `handleWebStatus` → `h.Web().CacheStatus`.

### Left on the root service (and why)
- `handleProjectsMigrate` (`svc.MigrateProjects`), `handleProjectsMerge`
  (`svc.MergeProjects`) — cross-store ATTACH/copy + alias ops; must span two
  stores, so they stay on the root service.
- `handleProjects` (`svc.ListProjects`) — cross-store directory listing.
- `handleMemoryCurrentProject` (`svc.CurrentProject`) — cwd resolution +
  cross-store project list.
- `handleSessionCreate` (`svc.SessionStart` / `svc.SessionStartByClientID`) —
  directory-rooted; the facade opens via `openProjectForDirectory` (which can
  do a seed-merge fold before opening). Reproducing that fold cleanly through a
  handle was not worth forcing, so it stays on the root service (one open).
- `handleCodeIndex` — **was** directory-rooted via `svc.RunCodeIndex`; it is now
  rewired through `openHandleForDir(dir)` + an in-handler indexing run (one
  open). The only behavioral difference: a non-`SourceIdentity` directory
  resolution (e.g. an ambiguous directory-hash fallback) does not run the
  best-effort seed-merge fold the facade performs. For the primary case
  (explicit `?dir=` of a resolvable repo, or `MNEMONIC_PROJECT` pin) the
  resolution is `SourceIdentity`, so `res.SeedID == res.ID` and the fold is a
  no-op — behavior is identical. See concerns.

## How facade semantics were reproduced

Non-trivial reproductions (not just thin delegations):

1. **`SaveObservation` → scope normalization.** `service.SaveObservation`
   normalizes `scope: ""→"project"`, `"personal"→"user"` before
   `memory.Save`. Reproduced as `scopeForSave(scope)` in
   `handleObservationCreate`, then `h.Memory().Save(memory.SaveInput{...})`
   with all fields (Title/Type/Content/Scope/TopicKey/SessionID/CapturePrompt/
   ProjectName/ToolName) mapped.
2. **`MemoryDoctor` → `httpMemoryDoctor`.** Byte-compatible copy of
   `service.MemoryDoctor`'s read-only SQL (schema_version, PRAGMA
   journal_mode, per-table counts, FTS counts, by-type GROUP BY, disk size,
   FTS drift/integrity), run on `h.Store().DB`. Returns the exported
   `service.MemoryDoctor` DTO so the JSON field names are unchanged.
3. **`readIndexedCode` → in-handler helper.** Byte-compatible copy of
   `service.readIndexedCode` (file lookup, optional line range, chunk join,
   first/last line computation), run on `h.Store().DB`.
4. **`CodeStatus` → stale rule.** `codeindex.GetStatus(h.Store())` +
   `stale = FileCount==0 || LastIndexed==""` (same as the facade).
5. **`RunCodeIndex` → in-handler run.** `config.Load(dir)` for the index
   config, `codeindex.New(h.Store())`, `httpResolveEmbedder(dir)` (mirror of
   `service.resolveEmbedder`: external/onnx/off from config), then
   `idx.Run(ctx, dir, idxCfg)`.
6. **`CodeFiles` → inline SQL.** `SELECT path FROM files ORDER BY path`.

Pure thin delegations (no logic to reproduce): everything else above —
`RecentContext`, `Recent`, `Timeline`, `Update`, `Delete`, `ListReviews`,
`MarkReviewed`, `RecordRelation`/`RemoveRelation`/`RelationsOf`/
`RelationsBetween`, `Status`, `LastObservationAt`, `SessionEnd`/`SetTitle`/
`StartedAt`, `CompactionContext`, `SavePrompt`, `CapturePassive`,
`SearchWithScope`, and all `h.Web()` calls.

## The single-open test and what it asserts

`skillgrid-cli/internal/mnemonic/http/single_open_test.go` (new) uses the
step-02 test-only, mutex-guarded `store` open counter
(`store.OpenCount()` / `store.ResetOpenCount()` — `store.RecordOpen()` is
called at the top of `store.Open()`). Because the counter is process-global,
the test resets it before each request and asserts the count for the request
under test.

- `TestObservationRoutesOpenStoreOnce` (the [RED] scenario "Observation routes
  open store once"): seeds a store, then `POST /observations` → asserts
  `OpenCount()==1` (response 201 + positive `id`), resets, then
  `GET /observations/recent` → asserts `OpenCount()==1` (response 200 +
  `sessions`). A double-open (route open #1 + facade `openProject` #2) would
  read 2.
- `TestObservationListRouteOpenStoreOnce`: `GET /observations` → `OpenCount()==1`.
- `TestSingleProjectRoutesUseHandle` (scenario "Single-project HTTP uses
  handle"): a matrix of 20 rewired single-project routes (observations,
  context, memory status/doctor/timeline/reviews, search, relations, code
  status/files/search, web lookup/search/status, session get, observation
  update, review mark). Each subtest asserts the response has its unchanged
  JSON key and `OpenCount()==1`.
- `TestSessionEndOpenStoreOnce`: `POST /sessions/{id}/end?project=` →
  `OpenCount()==1`.
- `TestMigrateMergeStayOnRoot` (scenario "Migrate and merge stay on root"):
  exercises `POST /projects/merge` (asserts `merged`+`alias_recorded` true)
  and `POST /projects/migrate` (asserts 0 rows moved) through the root
  service, then `GET /observations` and asserts `OpenCount()==1` — i.e. the
  cross-store path runs and single-project routes still open once afterward.

## Test results

| Check | Command | Result |
|-------|---------|--------|
| Focused single-open | `go test ./internal/mnemonic/http/ -run 'Observation\|SingleOpen\|Handle' -count=1` | PASS (7 tests) |
| Full http suite | `go test ./internal/mnemonic/http/ -count=1` | PASS (53 tests) |
| service/mcp/store suites | `go test ./internal/mnemonic/service/ ./internal/mnemonic/mcp/ ./internal/mnemonic/store/ -count=1` | PASS |
| Build | `go build ./...` | OK |
| go vet | `go vet ./internal/mnemonic/http/` | clean |
| gofmt | `gofmt -l internal/mnemonic/http/` | clean |

## TDD evidence

### 03.1 RED — Observation routes open store once
Failing test written first (`single_open_test.go`: `TestObservationRoutesOpenStoreOnce`
+ `TestObservationListRouteOpenStoreOnce`), asserting `store.OpenCount()==1`
after `POST /observations` and `GET /observations/recent`.

Command:
`go test ./skillgrid-cli/internal/mnemonic/http/ -run 'Observation|SingleOpen|Handle' -count=1`

**Important finding (deviation from the brief's prediction):** the focused test
**passed before the rewire** (count was already 1), not the predicted FAIL(2).
Why: unlike the MCP layer, an HTTP handler never opens the project itself —
`projectFromRequest(r)` resolves the project id with **no** `store.Open` (it
reads the `?project=` query or the body), and the handler then calls exactly one
facade method which opens once. So the pre-rewire flow was already
`0 + 1 = 1` opens. The MCP double-open was real because `openService()`
(open #1, to learn the CWD project) was *followed by* a facade call (open #2).
That second open does not exist on the HTTP path. (Verified empirically: a
probe hitting every single-project route read `OpenCount()==1` pre-rewire, and
two explicit `svc.Open` calls read 2, so the counter is live and the probe is
valid.)

The rewire is still correct and is now **pinned**: the handlers open a
`ProjectHandle` explicitly and call through it (matching MCP), and the tests
assert the single-open invariant so any future regression that introduces a
second open (e.g. a route that both opens a handle *and* calls a re-opening
facade) fails with `count 2, want exactly 1 (double-open)`.

### 03.1 GREEN / 03.2
After the rewire the full suite passes (commands + results in the table above).
`TestSingleProjectRoutesUseHandle` (20 routes) and `TestMigrateMergeStayOnRoot`
carry the 03.2 scenarios (single-project uses handle; migrate/merge stay on
root; JSON field names unchanged — each subtest checks its response key).

## Files changed
- `skillgrid-cli/internal/mnemonic/http/server.go` — added `openHandleFor` /
  `openHandleForDir` / `scopeForSave` / `httpMemoryDoctor` / `httpRowCount` /
  `httpResolveEmbedder` / `httpToAsym` / `readIndexedCode`; rewired all
  single-project handlers to open once and go through the handle; added
  imports (`context`, `database/sql`, `codeindex`, `config`, `embedder`,
  `search`).
- `skillgrid-cli/internal/mnemonic/http/single_open_test.go` (new) — single-open
  + root-service + route-matrix tests.

No other package was edited (service/, memory/, webcache/, store/ were read
only, to reproduce semantics and reuse the existing counter).

## Self-review findings
- All remaining `s.svc.` calls in server.go are exactly the intended root set:
  `Open`/`OpenForDirectory` (the helpers), `SessionStart`/`SessionStartByClientID`
  (handleSessionCreate), `MigrateProjects`, `MergeProjects`, `CurrentProject`,
  `ListProjects`. No single-project facade method is called directly anymore.
- Route paths, `requireWriteAuth` bearer behavior, and JSON request/response
  field names are unchanged (the full pre-existing suite — auth 401/200, UI
  routes, DTO shapes — passes).
- Error status codes per handler are preserved (404 for observation
  not-found, 422 for relation, 400 for bad project/id, 202 for
  `too-small` prompt).
- The counter is process-global and the tests do not call `t.Parallel()`, so
  the reset-then-assert pattern is race-free within the package.

## Issues / concerns
1. **RED did not fail before the rewire** (count was already 1, not 2). Root
   cause documented above: the HTTP handler never opened the project itself, so
   the MCP-style "two opens" did not exist on the HTTP path. The rewire is
   still the correct, MCP-consistent structure and the single-open invariant is
   now pinned by tests. This is a brief/prediction discrepancy, not a defect in
   the implementation.
2. **`handleCodeIndex` seed-merge caveat.** The facade `RunCodeIndex` opens via
   `openProjectForDirectory`, which performs a best-effort `MergeProjects`
   fold of a pre-identity directory-hash seed store (only when
   `res.SeedID != res.ID` and `res.Source == SourceIdentity`). The rewired
   handler uses `openHandleForDir` (same `openProjectForDirectory`) but then
   runs the index in-handler; the fold is still performed by the open. The only
   difference is that a non-`SourceIdentity` resolution (ambiguous directory
   hash) does not trigger the fold — but that case is a hard-abort for writes
   anyway and is a no-op for the primary `?dir=`/pinned cases. Low risk; flagged
   for the orchestrator.
 3. **Verbatim helper copies** (`httpMemoryDoctor`, `readIndexedCode`,
    `httpResolveEmbedder`) mirror the service layer — the same pattern step 02
    used for MCP (`mcpMemoryDoctor`, `mcpReadIndexedCode`, `mcpResolveEmbedder`).
    Consolidation is step 04's job (collapse the double-open wrappers / expose
    db-backed impls from service/).

---

## Review finding fix — `httpResolveEmbedder` double-loaded config

**Finding (Important):** `handleCodeIndex` (server.go) called `config.Load(dir)`
to build the index config, then called `httpResolveEmbedder(dir)` which called
`config.Load(dir)` a **second** time. The service source uses a single loaded
config root for both the index config and the embedder.

**What changed:** `httpResolveEmbedder` now accepts the already-loaded
`cfg config.Indexing` instead of `dir`, and no longer calls `config.Load`
internally. `handleCodeIndex` now calls `httpResolveEmbedder(cfg)` with the same
`cfg` it already loaded for `idxCfg`. The config is loaded **once** per request
in `handleCodeIndex`, and both the index config and the embedder are derived from
that single load.

Behavior is byte-identical: the same external/onnx/off switch, the same config
field mapping (`BaseURL`/`Model`/`APIKey`/`Dimension`/`Indexing`/`Query`), and
the same `httpToAsym` conversion. The `config` import remains in use
(`config.Load` at the handler, `config.Indexing` and `config.EmbedderParams` in
the helper signatures). Routes, auth, JSON field names, and all other packages
are unchanged.

**Files changed:**
- `skillgrid-cli/internal/mnemonic/http/server.go` — `httpResolveEmbedder`
  signature `configRoot string` → `cfg config.Indexing` (dropped the internal
  `config.Load`); `handleCodeIndex` passes `cfg` instead of `dir`.

**Covering tests:**

| Check | Command | Result |
|-------|---------|--------|
| Focused code-index + single-open | `go test ./internal/mnemonic/http/ -run 'CodeIndex\|SingleOpen\|Handle' -count=1` | PASS (TestCodeIndexEndpoint + TestSingleProjectRoutesUseHandle, 20 routes, all green) |
| Full http suite | `go test ./internal/mnemonic/http/ -count=1` | PASS |
| gofmt | `gofmt -l internal/mnemonic/http/` | clean |
