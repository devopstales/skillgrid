# 03-autosync-watcher — implementation report

## Per sub-task

### 03.1 `[RED]` fsnotify auto-sync watcher with debounce (RED → GREEN)
- Created `codeindex/watch.go`: `Watcher` (fsnotify + debounce + source filter). `DefaultDebounce=2000ms`, `clampDebounce` clamps to `[100ms, 60s]`. `Start(ctx)` opens fsnotify on the index root and runs a background loop; a burst of create/modify/delete events on **source** files (005 include/exclude, `IsSource`) collapses into a single `OnSync` call after the debounce window. Non-source, chmod, rename, and out-of-root events are ignored. `MarkPending`/`ClearPending`/`Pending` expose the pending set (feeds the staleness banner). `WatchDisabled()` reads `SKILLGRID_NO_WATCH`.
- `go.mod` + `go.sum`: added `github.com/fsnotify/fsnotify v1.5.1` (pure-Go FSEvents/inotify/KQueue, no CGo; now a **direct** dep after `go mod tidy`).
- Test `watch_test.go:TestWatcher` (RED: `undefined: NewWatcher/WatchConfig` → compile FAIL; GREEN: PASS).
- Commit `c436a30`.

### 03.2 `[RED]` copy-and-swap index publication (RED → GREEN)
- Created `codeindex/publish.go`: `Publish{LivePath, Build, SupportSidecar, InPlace, Copy, MaxInPlaceRetries}`. Sidecar mode = `Copy(live→sidecar)` + `Build(sidecar)` + `os.Rename(sidecar→live)` (atomic; a reader sees old-or-new, never torn; sidecar removed on failure). In-place fallback (sidecar unsupported): retries on **pre-write** failure (`NewPreWriteFailure`), and **stops** with `ErrMayHaveMutated` once a mid-write failure may have mutated the live index (the old index stays live). `IsMayHaveMutated` + `isPreWriteFailure` are exported for the failure test.
- Test `publish_test.go:TestPublish` (atomic swap, no leftover sidecar) + `TestPublishInPlaceRetry` (pre-write retry succeeds; mid-write stops after 1 attempt with `ErrMayHaveMutated`). RED: `undefined: Publish` → compile FAIL; GREEN: PASS.
- Commit `9d097eb`.

### 03.3 `[RED]` reader auto-reopen of published index (RED → GREEN)
- Created `codeindex/reopen.go`: `ReopenTracker` (injectable `reopenWindow`, `now`). `Check()` stats the live index file; if it's newer than the recorded open time **and** the window (default `DefaultReopenWindow=5s`, injectable) has elapsed, it fires `OnReopen(livePath)` and re-records the open time (no restart). A second `Check` immediately after does not reopen again.
- `mcp/freshness.go` (response path): `InitFreshness(svc, projectID, root, cfg, reopenWindow)` records the live index path + the gate context; `preQuery` runs `tracker.Check()` (auto-reopen) before answering a `code_*` query.
- Tests: `codeindex/reopen_test.go:TestAutoReopen` (RED: `undefined: NewReopenTracker`) + `mcp/freshness_test.go:TestAutoReopen` (the next tool call reopens after a publish, no restart; injected 10ms window). GREEN: PASS.
- Commit `07bec0a`.

### 03.4 `[RED]` connect-time size/mtime/hash catch-up (RED → GREEN)
- `codeindex/catchup_test.go:TestCatchUp`: after an index + `StoreFingerprint`, an **offline** edit (watcher off) to `b.go` is detected by the fingerprint `(size, mtime)` stat-walk (`FingerprintDriftDB`) as exactly `b.go` (not `a.go`); the reconcile re-index (`idx.Run`, the 005 single content-hash guard) re-indexes only `b.go` (1 indexed, 1 skipped) and the new content lands in chunks.
- RED: `undefined: FingerprintDrift/StoreFingerprint` → compile FAIL; GREEN: PASS. (Committed in `bee853c` alongside 03.6, which shares the catch-up file.)

### 03.5 `[RED]` staleness banner on pending files (RED → GREEN)
- `mcp/freshness.go`: `applyFreshness(res, referencedPaths)` prepends `⚠️ <file> is pending sync — Read it directly` for a referenced pending file and appends a small footer (`⚠️ pending sync (not yet indexed): …`) for un-referenced pending files; it clears once the pending set drains (after sync). `SetPendingSource` (test hook) + `AttachWatcher` wire the pending source into the response path. `hitPaths` extracts the referenced paths from `code_search` hits. `handleCodeSearch`/`handleCodeRead` funnel through `applyFreshness`.
- Test `mcp/freshness_test.go:TestStalenessBanner` (banner names a referenced pending file; un-referenced → footer only; after sync the banner clears). RED: `undefined: applyFreshness/SetPendingSource` → compile FAIL; GREEN: PASS.
- Commit `cb49fcf`.

### 03.6 `[RED]` single-writer lock (RED → GREEN)
- `codeindex/watch.go`: `WriterLock` (in-process; `Acquire` returns `*LockError{Holder}` naming the live writer; `Release`; `Holder`) + `AcquireFileLock(path, holder)` (cross-process O_EXCL; a second process gets a `*LockError` naming the live holder read from the file). `IsLockError` walks the cause chain. The fingerprint gate acquires `WriterLock("query-gate")` around its structural re-index (same lock family as the watcher).
- `mcp/freshness.go`: `preQuery` holds `fresh.lock` (`*codeindex.WriterLock`) during the gate's re-index.
- Test `codeindex/writerlock_test.go:TestWriterLock` (first writer acquires; second gets a clear `*LockError` naming the live holder; release → re-acquire; cross-process O_EXCL variant names the live holder). RED: `undefined: NewWriterLock/AcquireFileLock` → compile FAIL; GREEN: PASS.
- Commit `bee853c` (with 03.4 catch-up).

### 03.7 `[RED]` pull-at-query fingerprint gate (structural-only, embedder-free) (RED → GREEN)
- Created `codeindex/fingerprint.go`:
  - `ExtractorStamp()` → `"v1"` (the fingerprint is **keyed by the extractor stamp**; a stamp change invalidates it, forcing a full re-walk).
  - `FingerprintDrift(root, cfg, stamp)` / `FingerprintDriftDB(db, root, cfg, stamp)`: a ~3ms `(size, mtime)` **stat-walk** (metadata only, no file content) compared against the stored fingerprint (the `files` table's `mtime_ns`/`size` — the indexer's single-transaction `(size,mtime)`+content-hash guard). Returns the changed (added/modified/deleted) source files. A stored-stamp mismatch forces a full re-walk.
  - `StoreFingerprint(db, root, cfg, stamp)` records the stamp in a lazily-created `fingerprint_meta` table.
- `service.ReindexStructural(ctx, directory, cfg)`: a **structural-only** incremental re-index — `codeindex.New(store).Run(...)` with **no embedder attached** (the eager `embedPass` only runs when `idx.emb != nil`), so **no model load / embedding call** happens during the gate. It then persists the extractor stamp. Advisory: a failure keeps the old index live.
- `mcp/freshness.go`: `preQuery` runs the gate **before the answer** (stat-walk → on drift, structural re-index under the writer lock → drain). It never touches the embedder leg.
- Tests: `codeindex/fingerprint_test.go:TestFingerprintGate` (drift on an edited working-tree file; structural re-index under the writer lock; the query then reflects the edited tree — the new `gamma` symbol is indexed; no drift after re-index) + `TestFingerprintStampInvalidation` (same stamp + unchanged tree → no drift; a stamp change → full re-walk, both files changed). RED: `undefined: FingerprintDrift/StoreFingerprint/ExtractorStamp` → compile FAIL; GREEN: PASS.
- **Load-bearing property**: the gate's re-index is built with a nil embedder (asserted structurally — `ReindexStructural` attaches no embedder, so `embedPass` is skipped). The test uses `newTestIndexer` (no `WithEmbedder`) and proves the edited tree is picked up with no embedding call.
- Commit `eeb9275`.

### 03.8 `[AFK]` reindex failure keeps the old index live
- `codeindex/catchup_test.go:TestWatchFailure`: copy-and-swap build failure leaves the live index untouched (old stays live); in-place mid-write failure stops after 1 attempt with `ErrMayHaveMutated` (old index kept). PASS.

### 03.9 `[AFK]` watcher disabled means manual index
- `codeindex/watch_test.go:TestWatchDisabled`: `SKILLGRID_NO_WATCH=1` → `WatchDisabled()` true; `0`/unset → false. PASS. The serve wiring (`startAutoSync`) skips the watcher when disabled (manual index; the fingerprint gate is the backstop).

### 03.10 `[RED]` warm embedder load-once cache (RED → GREEN)
- Created `embedder/warm.go`: `WarmEmbedder` (`NewWarm(cfg)`). `Get(ctx)` loads the model **once** on first use (via `cfg.Factory`) and reuses it across calls; it records the access time (resets the idle-evict window). A nil/absent factory or a `Dimension()==0` (Null Adapter, `off`/external) holds no RAM — no caching, no eviction, no model load.
- `service.go`: a process-wide `warmFor(provider)` cache; `CodeHybridSearch`/`CodeSemanticSearch` now call `s.warmSearchEmbedder(h.root)` → `warmFor(provider).Get(ctx)` (the heartbeat), instead of a fresh `resolveEmbedder` per call. `off`/empty provider returns nil (FTS floor). `WarmEmbedderHandle()`/`resetWarmCache()` expose the cache for the idle-evict timer + tests.
- Test `embedder/warm_test.go:TestWarmEmbedder` (first call loads exactly once; 5 further `Get`s do not re-load → `loads==1`). RED: `undefined: NewWarm/WarmConfig` → compile FAIL; GREEN: PASS.
- Commit `8dad267`.

### 03.11 `[RED]` warm embedder idle-evict + heartbeat keep-alive (RED → GREEN)
- `embedder/warm.go`: `CheckIdle(now, connectedClients)` evicts after `IdleTimeout` (default `DefaultIdleTimeout=10min`, **injected** in tests) when `connectedClients==0`; a connected MCP client (heartbeat, `connectedClients>0`) suppresses eviction while active. `Evict()` frees a resident (non-Null) embedder; evicting a Null Adapter is a no-op. `Get` after eviction transparently reloads.
- Test `embedder/warm_test.go:TestWarmEvict` (injected 100ms window; idle with a connected client → survives (heartbeat); idle with no client → evicted; a Null Adapter never counts as resident). RED: `undefined: CheckIdle` → compile FAIL; GREEN: PASS.
- Commit `2b0d3aa` (with the search-path wiring).

### 03.12 `[AFK]` absent model degrades to FTS + signals
- `embedder/warm_test.go:TestWarmFallback`: nil factory → `Get` returns a Null Adapter (dim 0), `Embed` returns an empty vector with **no error** (FTS + signals floor, no model-load error surfaced). PASS.

### 03.13 `[AFK]` evicted-then-reused embedder reloads transparently
- `embedder/warm_test.go:TestWarmReload`: load 1 → evict after idle → reuse → transparent reload (load 2), no error surfaced, dim preserved. PASS.

### 03.14 `[AFK]` `skillgrid index status` pending-sync section
- `cmd/skillgrid/mcp.go`: `runIndexStatus` (`skillgrid index status`) + `printIndexStatus`/`printIndexStatusJSON` render index health (file/chunk counts, last-indexed, stale) + a `### Pending sync:` section (the pending set; `(none)` when empty) + the watcher state (`enabled` / `disabled (manual index; the fingerprint gate is the backstop)`). `codeindex.GetWatchStatus(w)` + `WatchStatus` support it.
- Test `cmd/skillgrid/index_status_test.go:TestIndexStatus` (section present; pending files listed; `(none)` when empty; `SKILLGRID_NO_WATCH=1` → `watcher: disabled`; JSON shape has `pending_sync` + `watch_disabled`). PASS.
- Commit `53057ea`.

### 03.15 `[RED]` tool-surface stability (RED → GREEN)
- `mcp/watcher_surface_test.go:TestWatcherSurface`: asserts (1) every 005 `code_*` tool (the `expected005ToolSurface` baseline lock) **and** the 010 step-02 `code_affected`/`code_rename` tools keep their exact names + required params with the watcher + banner wired into the response path (they are response-path concerns, not tool-schema changes); (2) the staleness banner is present on the response path (a referenced pending file is named via `applyFreshness`); (3) bad args are rejected clearly (`code_read` with a missing required `path` → `IsError`).
- The 73-tool baseline lock (`TestAllToolsRegistered` count, `tools_code_knowledge_test.go` count `73`, `TestToolSurfaceBaseline`, `TestCodeSearchSchemaStable`) is **unchanged** — the watcher adds no tools and alters no schemas.
- RED: `undefined: applyFreshness` (test) → compile FAIL; GREEN: PASS.
- Commit `8b68453`.

## How the time-based tests use injectable durations

| Behavior | Default | Injection | Test duration |
|---|---|---|---|
| Watcher debounce | `2000ms` | `WatchConfig.Debounced` (clamped `[100ms,60s]`) | `50ms` (burst collapse) |
| Reader auto-reopen window | `5000ms` | `ReopenTracker` `reopenWindow` (via `NewReopenTracker(path, win)`) | `10ms` |
| Warm embedder idle timeout | `10min` | `WarmConfig.IdleTimeout` + injectable `WarmConfig.Now` clock | `100ms` |

No test waits a real 2000ms/5s/10min: the watcher test uses a 50ms window (a 3×10ms burst), the auto-reopen tests use a 10ms window + `os.Chtimes` to make the index newer, and the warm tests use a 100ms window driven by a fake clock (`Now` func + `CheckIdle(clock, clients)`).

## Full relevant suite (final run)

`go test ./internal/mnemonic/codeindex/... ./internal/mnemonic/embedder/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1`
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  6.634s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder   0.807s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp       31.637s
ok  github.com/devopstales/skillgrid/skillgrid-cli/cmd/skillgrid              13.501s
```
005/008/010 baseline:
```
ok  .../internal/mnemonic/graph      2.324s
ok  .../internal/mnemonic/route      0.626s
ok  .../internal/mnemonic/community  1.366s
ok  .../internal/mnemonic/affected   9.356s
```
All 22 `internal/mnemonic/...` packages pass (`go test ./internal/mnemonic/... -count=1` → 22 ok, 0 fail).

Named tests (all PASS): codeindex — `TestWatcher`, `TestWatchDisabled`, `TestPublish`, `TestPublishInPlaceRetry`, `TestAutoReopen`, `TestCatchUp`, `TestWatchFailure`, `TestWriterLock`, `TestFingerprintGate`, `TestFingerprintStampInvalidation`; embedder — `TestWarmEmbedder`, `TestWarmEvict`, `TestWarmFallback`, `TestWarmReload`; mcp — `TestAutoReopen`, `TestStalenessBanner`, `TestWatcherSurface` (+ baseline `TestAllToolsRegistered`, `TestToolSurfaceBaseline`, `TestCodeSearchSchemaStable`); cmd — `TestIndexStatus`.

## Commits

| SHA | Subject |
|-----|---------|
| c436a30 | feat(mnemonic): fsnotify auto-sync watcher with debounce |
| 9d097eb | feat(mnemonic): copy-and-swap index publication |
| 07bec0a | feat(mnemonic): reader auto-reopen of published index |
| eeb9275 | feat(mnemonic): pull-at-query fingerprint gate (structural-only, embedder-free) |
| cb49fcf | feat(mnemonic): staleness banner on pending files |
| bee853c | feat(mnemonic): single-writer lock for auto-sync + connect-time catch-up |
| 8dad267 | feat(mnemonic): warm embedder load-once cache |
| 2b0d3aa | feat(mnemonic): warm embedder idle-evict and heartbeat keep-alive |
| 53057ea | feat(mnemonic): index status pending-sync section + serve watcher wiring |
| 8b68453 | feat(mnemonic): watcher tool-surface stability and status |

## Concerns

- **Warm cache is keyed by provider, not model**: `warmFor(provider)` reuses a single `WarmEmbedder` per provider. A **model-name swap** (same provider) is picked up on the next reload (the `Factory` re-reads config; the indexer's `embed_meta` model-swap guard also clears stale vectors), but a swap does not force an immediate in-process cache reset — the first reload after a restart picks it up. Acceptable: model swaps are rare and the eager embed pass re-keys vectors by model.
- **`buildEmbedderFromProvider` re-reads `config.Load(".")`**: the warm cache's factory resolves config from `CWD` at load time. In the MCP/serve process CWD is the project root, so this is correct; in a CLI search from a sub-directory it still walks up to find `config.d/indexing.yaml`. The search path itself uses `h.root` for `warmSearchEmbedder` (the provider), so the vector leg is correct; the factory's `.` is a best-effort second read.
- **The fingerprint gate re-index is advisory**: a gate re-index failure keeps the old index live (the query answers against the last good index; a stale answer is possible but the reader is never served a torn index). The `(size,mtime)` stat-walk is the cheap layer; the content-hash guard inside `idx.Run` is the authoritative change detector.
- **`openProjectDBForRoot` (package-level `FingerprintDrift`) picks the first store in the data dir**: this is a best-effort convenience; the query path and all tests use the store-scoped `FingerprintDriftDB(db, …)`, which is unambiguous.
- **WatchFailure (03.8) is at the publish layer**: the in-place "stop the watcher" behavior is expressed as `ErrMayHaveMutated` (the serve `startAutoSync` surfaces the watcher error via the watcher's error channel); the test asserts the old index stays live + the stop sentinel, which is the load-bearing contract.
- **No `go build ./...` / `go vet` / `gofmt -l`** were run per the step rules (tests are the only verification). `go mod tidy` was run to move fsnotify from indirect to direct (it is now imported directly by `codeindex/watch.go`).

---

## Review-findings fixes (step 03)

Addressed 3 review findings (2 minor, 1 important — the load-bearing property). Each is now a real assertion that fails if the property regresses (verified by reverting the fix and watching the test fail).

### 1. Fingerprint gate's embedder-free property is now asserted (was comment-only)

`fingerprint_test.go` previously said the gate "only triggers a structural re-index" but had no assertion that no model-load/embedding fires. Added a **counting spy embedder** (`countEmbedder` in `fingerprint_test.go` / `reindexSpy` in `reindex_structural_test.go`) that counts every `Model()`/`Dimension()`/`Embed()`/`EmbedQuery()` call, plus two tests that prove the property:

- **`TestReindexStructuralIsEmbedderFree`** (service, the gate's production entrypoint): drives `svc.ReindexStructural` — exactly what `freshness.preQuery` calls — through a drifted tree and asserts the re-index syncs the new symbol (`gamma`) **without** any embedder. `ReindexStructural` is embedder-free *by construction* (it builds `codeindex.New(h.store)` and never calls `WithEmbedder`), so the structural re-index the gate runs can never invoke the embedder leg.
- **`TestReindexStructuralAttachedEmbedderIgnored`** (service counterfactual): the **same** Indexer (`codeindex.New(h.Store())`, identical to what `ReindexStructural` constructs) run **WITH** a spy attached (the `WithEmbedder` path `RunCodeIndex` uses) **fires** the spy — isolating the attach as the single variable that makes a re-index invoke the embedder. The gate's entrypoint, which never attaches, is the embedder-free side of that contrast.
- **`TestFingerprintGateEmbedderFree`** (codeindex): the same counterfactual at the index layer — a spy-attached run fires the embedder (proving the spy is sensitive to attach), while the gate's no-attach re-index is embedder-free and still syncs `gamma`.

Verified the counterfactual is non-vacuous: with the spy attached, `idx.Run` fires **5 calls** (model=1, dim=1, embed=3); the gate's no-attach run fires **0**.

### 2. `warmSearchEmbedder` resolves from `h.root`, not CWD

`warmSearchEmbedder`/`buildEmbedderFromProvider` called `config.Load(".")` / `resolveEmbedder(".")` (the CWD) instead of `h.root`. A search where `h.root != CWD` resolved the embedder from the wrong config. Fixed by threading `configRoot` (= `h.root`) through `warmFor` → `buildEmbedderFromProvider` → `resolveEmbedder`. Tests (`warm_search_embedder_test.go`):

- **`TestWarmSearchEmbedderUsesProjectRoot`**: h.root's config pins `off` → nil (FTS floor); a different root pinning `onnx` → non-nil.
- **`TestWarmSearchEmbedderHRootNotCWD`**: with the CWD's repo config defaulting to onnx and h.root pinning `off`, the warm path returns nil — h.root's config wins.
- **`TestWarmSearchEmbedderOffIsNil`**: explicit `off` → nil (the FTS-floor off behavior, kept intact).

Verified non-vacuous: with the old `config.Load(".")` behavior, all three tests **fail** (they returned the CWD's onnx embedder instead of the h.root `off`).

### 3. Failed-model → FTS branch is now tested

`embedder/warm.go` degrades to the Null Adapter when the `Factory` returns a nil embedder (the observable form of a failed model load), but `TestWarmFallback`'s `wFail` used a *healthy* `NewHash` factory, so the nil-embedder branch was never exercised. Added **`TestWarmFailedFactoryDegradesToFTS`**: a `Factory` returning `nil` → the warm cache degrades to Null (dim 0), does not crash, holds no RAM (`IsLoaded()==false`), embeds an empty vector (FTS floor), stays degraded on reuse, and is never reported resident.

### Test summary (step-03 review fixes)

```
go test ./internal/mnemonic/codeindex/... ./internal/mnemonic/embedder/... ./internal/mnemonic/mcp/... ./cmd/skillgrid/... -count=1
ok  .../codeindex  6.5s
ok  .../embedder   0.9s
ok  .../mcp        29.5s
ok  .../cmd/skillgrid  16.4s
```
005/008/010 baseline + service:
```
ok  .../service    18.7s
ok  .../graph      0.9s
ok  .../route      3.5s
ok  .../community  2.5s
ok  .../affected   9.6s
```
New/fixed tests (all PASS): codeindex — `TestFingerprintGateEmbedderFree`; service — `TestReindexStructuralIsEmbedderFree`, `TestReindexStructuralAttachedEmbedderIgnored`, `TestWarmSearchEmbedderUsesProjectRoot`, `TestWarmSearchEmbedderHRootNotCWD`, `TestWarmSearchEmbedderOffIsNil`; embedder — `TestWarmFailedFactoryDegradesToFTS`. Baseline locks `TestAllToolsRegistered`, `TestToolSurfaceBaseline` (73 tools), `TestCodeSearchSchemaStable` still pass.

### Commits (review fixes)

| SHA | Subject |
|-----|---------|
| 981f128 | test(mnemonic): assert fingerprint gate re-index is embedder-free (counting spy) |
| 3da7fbb | fix(mnemonic): warm embedder resolves from project root (h.root) not CWD |
| 1ee735f | test(mnemonic): failed-factory warm embedder degrades to FTS (Null Adapter) |
