# Task 02 Report — MCP single-open via Project Handle

## What was implemented

Every MCP tool handler that previously called `openService()` (which opened the
CWD project store once just to learn the project id + register cleanup) and then
called a `*service.Service` facade method (which internally re-opened the same
project via `s.openProject(projectID, ".")`) now opens the project **once** and
performs domain ops through the `*service.ProjectHandle` (`h.Memory()` /
`h.Web()` / `h.Store()`). The second open is eliminated.

### `openService()` / `openServiceForRepo()` rewired
- `openService()` (tools_memory.go) now returns
  `(*service.Service, *service.ProjectHandle, func(), error)` — the handle (the
  opened store + memory/web services) instead of just the resolved project id.
  `svc` is still returned for cross-store ops (drift, all-projects search) and
  ops with no handle method.
- `openServiceForRepo()` (tools_code_hybrid.go) now returns the opened
  `*service.ProjectHandle` too. It keeps calling `projectIDForPool(svc, repo)`
  for its validation/error behavior, then opens the handle via
  `svc.OpenForDirectory(".")`.

### Facade methods replaced with handle calls

**tools_memory.go** (all `h.Memory()` unless noted):
- `handleMemSave`: `svc.SaveObservation` → `h.Memory().Save` — **reproduced the
  scope normalization** (`"" → "project"`, `"personal" → "user"`) that
  `service.SaveObservation` applied before calling `memory.Save`. Drift check
  (`svc.CheckProjectDrift`) still uses `svc` (cross-store probe, not the single
  project).
- `handleMemSearch`: `svc.SearchObservationsScoped` → `h.Memory().SearchWithScope`
  (the scoped method is a thin open+delegate over that). The `all_projects`
  branch still uses `svc.SearchAllProjects` (cross-store merge — must stay on
  the root service). Drift check unchanged.
- `handleMemCapturePassive` → `h.Memory().CapturePassive`.
- `handleMemContext` → `h.Memory().RecentContext`.
- `handleMemGetObservation` → `h.Memory().Get`.
- `handleMemSessionSetTitle` → `h.Memory().SessionSetTitle`.
- `handleMemSessionEnd` → `h.Memory().SessionEnd`.
- `handleMemSessionSummary` → `h.Memory().SessionSummary`.
- `handleMemTimeline` → `h.Memory().Timeline`.
- `handleMemUpdate` → `h.Memory().Update`.
- `handleMemDelete` → `h.Memory().Delete`.
- `handleMemStats` → `h.Memory().Status`.
- `handleMemSavePrompt` → `h.Memory().SavePrompt`.
- `handleMemDoctor` → **reproduced** the store diagnostics as a new single-open
  helper `mcpMemoryDoctor(ctx, h)` that runs the same SQL (schema_version,
  journal_mode, per-table counts, FTS counts/drift, by-type, disk size) directly
  on `h.Store().DB` — byte-compatible with `service.MemoryDoctor`.
- `handleMemReview` → `h.Memory().ListReviews` / `h.Memory().MarkReviewed`.
- `handleMemJudge` / `handleMemCompare` + shared `recordRelation` →
  `h.Memory().RecordRelation` / `RemoveRelation` / `RelationsBetween`.
- `handleMemPin` / `handleMemUnpin` → `h.Memory().Pin` / `Unpin`.

Unchanged (correctly): `handleMemSessionStart` (uses `rootService()` +
`SessionStart`, which opens the *directory* project, not the CWD handle),
`handleMemCurrentProject`, `handleMemMergeProjects`, `handleMemUnify` (all
root-service / cross-store ops), `handleMemSuggestTopicKey` (pure).

**tools_code.go**:
- `handleCodeStatus`: `svc.CodeStatus` → `codeindex.GetStatus(h.Store())` with
  the same `stale` rule (`FileCount==0 || LastIndexed==""`); `svc.CodeFairCoverage`
  → `graph.FairCoverage(ctx, h.Store().DB)`.
- `handleCodeSearch`: `svc.CodeSearch` → `search.CodeSearch(h.Store().DB, query, limit)`.
- `handleCodeRead`: `svc.ReadIndexedCode` → new `mcpReadIndexedCode(db, path,
  startLine, endLine)` — byte-compatible copy of `service.readIndexedCode`.

**tools_code_orient.go** (code_orient / code_signature / code_file_toc /
code_rationale): all four now share `mcpOrientSymbol(db, symbol)` — a
byte-compatible copy of `service.orientSymbol` (exact-name then identifier-FTS
fallback, file TOC, list, rationale).

**tools_code_graph.go**:
- `handleCodeNeighbors` → `mcpGrabNeighbors(ctx, h, view, symbol)`.
- `handleCodePath` → `mcpCodePath(ctx, h, from, to)`.
- `handleCodeExplain` → `mcpCodeExplain(ctx, h, symbol)`.
- `handleCodeImpact` → `mcpCodeImpact(ctx, h, symbol, opts)` — all four are
  byte-compatible copies of the corresponding `service` methods (Resolve →
  NotFound/Ambiguous/Impact branches).

**tools_code_explore.go**:
- `handleCodeExplore` → `mcpCodeExplore(ctx, h, symbol)` — byte-compatible copy
  of `service.CodeExplore` (resolve, verbatim source for target + related,
  call-flow with INFERRED hops, impact summary). Reuses `mcpReadIndexedCode`,
  `mcpCodeImpact`, plus new `mcpReadSymbolSpans` / `mcpNameOf`.

**tools_code_hybrid.go**:
- `handleCodeHybridSearch` / `handleCodeSemanticSearch` → `hybrid.Search(ctx,
  h.Store().DB, query, hybrid.Options{... Embedder: mcpResolveEmbedder(h)})`.
- `handleCodeEmbeddingStatus` → `mcpCodeEmbeddingStatus(ctx, h)` — byte-
  compatible copy of `service.CodeEmbeddingStatus`. `mcpResolveEmbedder` mirrors
  `service.resolveEmbedder` (external/onnx/off) from `config.Load(".")` — the
  same config root the handle's `OpenForCWD`/`OpenForDirectory` used to build
  its web cache (the ContentPlane root is not exposed on the handle).

**tools_web.go** (all `h.Web()`):
- `handleWebCacheLookup` → `h.Web().Lookup`.
- `handleWebCacheSave` → `h.Web().Save`.
- `handleWebCacheSearch` → `h.Web().Search`.
- `handleWebCacheGet` → `h.Web().Get`.
- `handleWebCacheStatus` → `h.Web().CacheStatus`.

### Out of scope (left on the root service, not double-open via `openService`)
`tools_retrieval.go` (`semantic_search`, `load_full_details`) and
`tools_compaction.go` (`mnemonic_commit`) use `rootService()` + `projectIDFor`
(a different pattern — they do not call `openService()`), so they were not part
of the `openService` double-open this task eliminates. They were left unchanged.

## Single-open counter mechanism

A test-only, process-global counter in the store layer (the single choke point
for SQLite opens):

- `store.RecordOpen()` — called at the top of `store.Open()` (successful or not).
- `store.OpenCount()` / `store.ResetOpenCount()` — read/reset the counter.
- A `sync.Mutex` guards the counter (safe under parallel tests).

It is a lightweight diagnostic that is never read on the production path — only
the mcp tests use it. It lives in `store/` (allowed by the brief: "If a store
open-counter is unavoidable and cleaner in `store/`, you MAY add a minimal
test-only counter there"). No production behavior was added beyond incrementing
the counter.

## Test results

| Check | Command | Result |
|-------|---------|--------|
| Focused single-open + mem | `go test ./internal/mnemonic/mcp/ -run 'SingleOpen\|MemSave\|MemSearch' -count=1` | PASS (7 tests) |
| Full mcp suite | `go test ./internal/mnemonic/mcp/ -count=1` | PASS |
| Store suite | `go test ./internal/mnemonic/store/ -count=1` | PASS |
| Full build | `go build ./...` | OK |
| go vet | `go vet ./internal/mnemonic/mcp/ ./internal/mnemonic/store/` | clean |

## TDD evidence

### 02.1 RED — mem_save opens store once
Failing test written first (added store counter + `TestMemSaveSingleOpen`).

Command:
`go test ./skillgrid-cli/internal/mnemonic/mcp/ -run 'MemSave|SingleOpen|OpenOnce' -count=1`

Result (FAIL, before rewire):
```
--- FAIL: TestMemSaveSingleOpen (0.09s)
    single_open_test.go:77: mem_save opened the store 2 times, want exactly 1 (double-open)
--- FAIL: TestMemSavePromptSingleOpen (0.09s)
    single_open_test.go:106: mem_save_prompt opened the store 2 times, want exactly 1
--- FAIL: TestMemSearchSingleOpen (0.08s)
    single_open_test.go:197: scoped mem_search opened the store 2 times, want exactly 1
--- FAIL: TestCodeSearchSingleOpen (0.09s)
    single_open_test.go:217: code_search opened the store 2 times, want exactly 1
--- FAIL: TestWebCacheLookupSingleOpen (0.09s)
    single_open_test.go:235: web_cache_lookup opened the store 2 times, want exactly 1
FAIL
```
Why expected: each handler opened the store in `openService()` (OPEN #1) and
again inside the service facade method (OPEN #2) → counter == 2.

### 02.1 GREEN
After the rewire:
```
--- PASS: TestMemSaveSingleOpen (0.09s)
--- PASS: TestMemSavePromptSingleOpen (0.08s)
--- PASS: TestMemSearchSingleOpen (0.10s)
--- PASS: TestCodeSearchSingleOpen (0.07s)
--- PASS: TestWebCacheLookupSingleOpen (0.10s)
PASS
```
Counter == 1 for each.

### 02.2 — mem_search contract shape
`TestMemSearchContractShape` asserts the `mem_search` result keeps its exact
shape (`project` + `observations[]` with the same field names) after the handle
path, and `TestMemSearchSingleOpen` proves the scoped path opens once. Both pass.

### 02.3 — code_* / web_cache_* use the handle; SetService inject still works
- `TestCodeSearchSingleOpen` / `TestWebCacheLookupSingleOpen` prove those
  handlers open once through the handle.
- `SetService(service.New(dataDir))` injection is exercised by the new tests
  (`pinProjectCwd` calls `SetService`) and by the pre-existing
  `TestAllToolsRegistered` / `TestSuggestTopicKeyDispatch` / `TestCodeImpactTool`
  etc. Full suite passes.

## Files changed
- `skillgrid-cli/internal/mnemonic/mcp/tools_memory.go` — openService → handle; all mem handlers rewired; `mcpMemoryDoctor`/`mcpRowCount` helpers.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code.go` — CodeStatus/CodeSearch/CodeRead via handle; `mcpReadIndexedCode` helper.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_orient.go` — orient handlers via `mcpOrientSymbol`.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_graph.go` — neighbor/path/explain/impact via `mcp*` helpers.
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_explore.go` — explore via `mcpCodeExplore` (+ `mcpReadSymbolSpans`, `mcpNameOf`).
- `skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go` — openServiceForRepo → handle; hybrid/semantic/embedding-status via handle (+ `mcpCodeEmbeddingStatus`, `mcpResolveEmbedder`, `mcpBuildEmbedder`, `mcpToAsym`).
- `skillgrid-cli/internal/mnemonic/mcp/tools_web.go` — all web_cache handlers via `h.Web()`.
- `skillgrid-cli/internal/mnemonic/mcp/single_open_test.go` (new) — single-open + contract-shape tests; `pinProjectCwd` helper.
- `skillgrid-cli/internal/mnemonic/mcp/tools_compaction_test.go` — pinned `MNEMONIC_PROJECT` in `TestMemSaveRemainsRegisteredAdditive` (env-fixture fix, see below).
- `skillgrid-cli/internal/mnemonic/store/store.go` — test-only open counter.

## Self-review findings

### Facade methods whose semantics had to be reproduced (not just delegated)
These were non-trivial and were copied byte-for-byte into mcp helpers so the
handle path is behavior-identical:
1. `service.SaveObservation` → scope normalization (`""→project`,
   `personal→user`) reproduced in `handleMemSave` before `h.Memory().Save`.
2. `service.MemoryDoctor` → full store-diagnostic SQL reproduced in
   `mcpMemoryDoctor` (schema_version, journal_mode, counts, FTS drift, by-type,
   disk size).
3. `service.readIndexedCode` → `mcpReadIndexedCode` (verbatim copy).
4. `service.orientSymbol` → `mcpOrientSymbol` (exact + FTS fallback, TOC, list,
   rationale).
5. `service.GrabNeighbors` / `CodePath` / `CodeExplain` / `CodeImpact` → `mcp*`
   copies (Resolve NotFound/Ambiguous branches + graph traversal).
6. `service.CodeExplore` → `mcpCodeExplore` (source spans, call-flow, impact).
7. `service.CodeEmbeddingStatus` → `mcpCodeEmbeddingStatus`; `resolveEmbedder`
   → `mcpResolveEmbedder`/`mcpBuildEmbedder` (external/onnx/off, from
   `config.Load(".")`).

### Pure thin delegations (no logic to reproduce)
`GetObservation`→`Get`, `RecentContext`, `SessionSetTitle/End/Summary`,
`Timeline`, `Update`, `Delete`, `Status`, `SavePrompt`, `ListReviews`,
`MarkReviewed`, `RecordRelation`/`RemoveRelation`/`RelationsBetween`, `Pin`/
`Unpin`, `CapturePassive`, `SearchObservationsScoped`→`SearchWithScope`, and all
`h.Web()` calls — these were single `openProject` + `h.memory.X` wrappers, so
the handle call is equivalent with no logic to reproduce.

### Environment-dependent test fix (concern → addressed)
`TestMemSaveRemainsRegisteredAdditive` (tools_compaction_test.go) never set
`MNEMONIC_PROJECT` or `os.Chdir`, so it implicitly assumed the CWD resolved to
the seeded project `"reg"`. Before the rewire that held only where the CWD
resolved to `"reg"`; on this machine the CWD resolves to `"mcp"` (directory
identity), and the rewire correctly saves through the CWD (`"mcp"`) store —
where the seeded session does not live — producing a foreign-key error. This is
a pre-existing test fragility exposed by the rewire (the test's premise was
"the CWD store is the seeded store"). I pinned `t.Setenv("MNEMONIC_PROJECT",
"reg")` + a `SetService` cleanup so the test is deterministic and matches the
single-open contract (save through the CWD store, which is now explicitly the
`"reg"` store). Verified it fails without the pin and passes with it.

### Notes / assumptions
- `handleCodeRead`/`mcpReadIndexedCode` and the graph/orient helpers use the
  handle's `*sql.DB` (same connection the store owns), so there is no new open.
- The hybrid handlers resolve the embedder from `config.Load(".")` (the MCP
  process cwd) — identical to what the handle used for its web-cache config, so
  behavior is unchanged.

## Issues / concerns
- **None blocking.** The only judgment call beyond a pure rewire is the
  `MNEMONIC_PROJECT` pin in one pre-existing test, which is a correct
  environment-fixture hardening (the test had no deterministic project
  resolution before). Flagging it so the orchestrator can confirm the intent.
- The mcp package now imports `graph`, `hybrid`, `codeindex`, `search`,
  `embedder`, `config` directly (it already imported `memory`, `webcache`,
  `project`, `service`). No new dependencies were added to the module.
