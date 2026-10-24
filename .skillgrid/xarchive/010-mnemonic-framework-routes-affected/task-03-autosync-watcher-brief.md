## 03-autosync-watcher

### Goal

fsnotify watcher + **pull-at-query fingerprint gate** + copy-and-swap publication + reader auto-reopen + connect-time catch-up + staleness banner + warm embedder daemon — keep the graph fresh *and* the embedder warm as the agent edits. The watcher is the *comfort* layer (near-zero-latency background sync); the **fingerprint gate** is the *correctness backstop* — a ~3ms `(size, mtime)` stat-walk per query that runs a structural-only re-index on drift under the writer lock, so every `code_*` query is fresh even with the watcher off.

### Out of scope / Non-Goals

- Route edges + the drop-not-guess policy (step 01); `code_affected`/`code_rename` (step 02)
- Manual full re-index (still available); shared-daemon watcher mode (later optimization)

### Definition of Done

This step is done only when:

- [ ] All `### Tasks` checkboxes below are `[x]`
- [ ] All `@step-03` scenarios in `acceptance.feature` pass
- [ ] `### Verification` Verdict is `PASS` or `PASS WITH WARNINGS`
- [ ] Depends-on step(s) already PASS / PASS WITH WARNINGS
- [ ] No Global Constraint violated

> Depends on: 02-code-affected

**Files:**
- Create: `skillgrid-cli/internal/mnemonic/codeindex/watch.go`
- Create: `skillgrid-cli/internal/mnemonic/codeindex/fingerprint.go`
- Create: `skillgrid-cli/internal/mnemonic/codeindex/publish.go`
- Create: `skillgrid-cli/internal/mnemonic/embedder/warm.go`
- Modify: `skillgrid-cli/internal/mnemonic/mcp/server.go` (wire staleness banner + auto-reopen + warm-embedder heartbeat into response path)
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (`serve`/MCP watcher + fingerprint gate + warm-embedder wiring + `skillgrid index status` pending-sync section)
- Modify: `skillgrid-cli/go.mod` (add `github.com/fsnotify/fsnotify`)
- Test: `skillgrid-cli/internal/mnemonic/codeindex/...`, `skillgrid-cli/internal/mnemonic/embedder/...`, `skillgrid-cli/internal/mnemonic/mcp/...`, `skillgrid-cli/cmd/skillgrid/...`

**Interfaces:**
- Consumes: 005's single-transaction `(size, mtime)` + content-hash guard; fsnotify (FSEvents/inotify/KQueue, pure-Go); 005's ONNX `Embedder` interface; 005 include/exclude + `.gitignore` filters; the same writer lock as the watcher (fingerprint gate re-index)
- Produces: debounced incremental re-index (create/modify/delete, source files only); **pull-at-query fingerprint gate** (`codeindex/fingerprint.go` — ~3ms `(size, mtime)` stat-walk per query, structural-only re-index on drift under the writer lock, extractor-stamp keyed, never touches the embedder leg); copy-and-swap publication (sidecar → atomic publish) + in-place write-failure guard; reader auto-reopen (~5s, no restart); connect-time `(size, mtime)` + content-hash catch-up; `⚠️` staleness banner (pending referenced file) + footer (un-referenced); warm embedder cache (load-once, `idle_timeout` evict default 10min, MCP-heartbeat keep-alive); `skillgrid index status` `### Pending sync:` section; writer-lock for single-writer

### Tasks

- [ ] 03.1 `[RED]` Index freshness / concurrency — saving a source file triggers a debounced incremental re-index (Scenario: Saving a source file triggers a debounced re-index) — threat: Index freshness / concurrency
  - [ ] 03.1.a Write failing test — with a running MCP/serve watcher, save (create/modify/delete) of a source file fires fsnotify; a burst of rapid edits collapses into ONE sync after the debounce window (default 2000ms, clamped [100ms, 60s]); only the changed surface is re-indexed; non-source files (filtered by 005 include/exclude + `.gitignore`) are ignored
  - [ ] 03.1.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Watcher -count=1` — Expected: FAIL
  - [ ] 03.1.c Minimal implementation — `codeindex/watch.go` (fsnotify + debounce + source filter) + `go.mod` dep
  - [ ] 03.1.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Watcher -count=1` — Expected: PASS
  - [ ] 03.1.e Commit — `feat(mnemonic): fsnotify auto-sync watcher with debounce`
- [ ] 03.2 `[RED]` Index freshness / concurrency — copy-and-swap: reader never sees a torn index (Scenario: Reader sees old or new index never torn) — threat: Index freshness / concurrency
  - [ ] 03.2.a Write failing test — during a re-index publication, a concurrent reader observes either the old or the new index, never a partial one; on a sidecar-unsupported filesystem the in-place fallback retries on pre-write failure and stops when it may have mutated the live index
  - [ ] 03.2.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Publish -count=1` — Expected: FAIL
  - [ ] 03.2.c Minimal implementation — `codeindex/publish.go` (sidecar build → atomic rename/swap; in-place write-failure guard)
  - [ ] 03.2.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run Publish -count=1` — Expected: PASS
  - [ ] 03.2.e Commit — `feat(mnemonic): copy-and-swap index publication`
- [ ] 03.3 `[RED]` Index freshness / concurrency — running process reopens the new index without restart (Scenario: Reader auto-reopens the new index) — threat: Index freshness / concurrency
  - [ ] 03.3.a Write failing test — after a publication, a running MCP/serve process opens the newly-published index on its next tool call (within ~5s) with no restart; the agent's next query reflects its own just-made edit
  - [ ] 03.3.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -run AutoReopen -count=1` — Expected: FAIL
  - [ ] 03.3.c Minimal implementation — auto-reopen hook in the MCP serve response path (`mcp/server.go`)
  - [ ] 03.3.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/mcp/... -run AutoReopen -count=1` — Expected: PASS
  - [ ] 03.3.e Commit — `feat(mnemonic): reader auto-reopen of published index`
- [ ] 03.4 `[RED]` Index freshness / concurrency — connect-time catch-up absorbs offline edits (Scenario: Connect-time catch-up absorbs offline edits) — threat: Index freshness / concurrency
  - [ ] 03.4.a Write failing test — edits made while no MCP server was running (a `git pull`, another editor) are reconciled by a `(size, mtime)` + content-hash check on (re)connect; only genuinely changed files are re-indexed
  - [ ] 03.4.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run CatchUp -count=1` — Expected: FAIL
  - [ ] 03.4.c Minimal implementation — connect-time `(size, mtime)` + hash reconciliation in `watch.go`
  - [ ] 03.4.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run CatchUp -count=1` — Expected: PASS
  - [ ] 03.4.e Commit — `feat(mnemonic): connect-time size-mtime-hash catch-up`
- [ ] 03.5 `[RED]` Index freshness / concurrency — staleness banner for a pending referenced file (Scenario: Pending referenced file gets a staleness banner) — threat: Index freshness / concurrency
  - [ ] 03.5.a Write failing test — during the debounce window, an MCP tool response that references a still-pending file prepends `⚠️ <file> is pending sync — Read it directly`; pending files not referenced surface as a small footer; after sync the banner clears
  - [ ] 03.5.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run StalenessBanner -count=1` — Expected: FAIL
  - [ ] 03.5.c Minimal implementation — pending-file tracking + banner/footer in `mcp/server.go` response path
  - [ ] 03.5.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... -run StalenessBanner -count=1` — Expected: PASS
  - [ ] 03.5.e Commit — `feat(mnemonic): staleness banner on pending files`
- [ ] 03.6 `[RED]` Index freshness / concurrency — second writer exits with a lock error (Scenario: Second writer exits with a writer-lock error) — threat: Index freshness / concurrency
  - [ ] 03.6.a Write failing test — two `serve`/MCP processes on one project: the first acquires the writer lock; the second exits with a clear writer-lock error pointing at the live writer
  - [ ] 03.6.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WriterLock -count=1` — Expected: FAIL
  - [ ] 03.6.c Minimal implementation — single-writer lock (shared/direct-mode) in `watch.go` + `main.go` serve wiring
  - [ ] 03.6.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WriterLock -count=1` — Expected: PASS
  - [ ] 03.6.e Commit — `feat(mnemonic): single-writer lock for auto-sync`
- [ ] 03.7 `[RED]` Index freshness / concurrency — pull-at-query fingerprint gate re-indexes structurally on drift with the watcher off (Scenario: Fingerprint gate re-indexes structurally with watcher off) — threat: Index freshness / concurrency; Business rule "Pull-at-query fingerprint gate"
  - [ ] 03.7.a Write failing test — with the watcher off (`SKILLGRID_NO_WATCH=1`), a `code_*` query made after an edit runs a ~3ms `(size, mtime)` stat-walk against the last index's fingerprint; on drift a **structural-only** incremental re-index runs (under the same writer lock as the watcher) **before** the answer, so the query reflects the edited tree (uncommitted edits included); assert the gate **never invokes the embedder leg** (no model load/embedding call during the gate) and that the fingerprint is keyed by the extractor stamp (a stamp/version change invalidates it, forcing a re-walk)
  - [ ] 03.7.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run FingerprintGate -count=1` — Expected: FAIL
  - [ ] 03.7.c Minimal implementation — `codeindex/fingerprint.go` (stat-walk fingerprint + extractor-stamp key + structural-only re-index under the writer lock, never the embedder leg) + query-path hook in `main.go`/`mcp/server.go`
  - [ ] 03.7.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run FingerprintGate -count=1` — Expected: PASS
  - [ ] 03.7.e Commit — `feat(mnemonic): pull-at-query fingerprint gate (structural-only, embedder-free)`
- [ ] 03.8 `[AFK]` Watcher reindex failure stops in-place, keeps old on swap (Scenario: Reindex failure keeps the old index live) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... -run WatchFailure -count=1` — Expected: PASS
- [ ] 03.9 `[AFK]` Watcher disabled means manual index (Scenario: Watcher disabled means manual index) — `Run: go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/cmd/skillgrid/... -run WatchDisabled -count=1` — Expected: PASS
- [ ] 03.10 `[RED]` Warm embedder lifecycle — load-once, reused across search calls (Scenario: Warm embedder loads once and reuses) — threat: Warm embedder lifecycle
  - [ ] 03.10.a Write failing test — the ONNX model loads once and is reused across `skillgrid search` / MCP calls (no per-call model-load; the ~270MB model is not re-loaded per invocation); the first call pays the load, subsequent calls do not
  - [ ] 03.10.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmEmbedder -count=1` — Expected: FAIL
  - [ ] 03.10.c Minimal implementation — `embedder/warm.go` load-once ONNX cache + `mcp/server.go` / `main.go` wiring into the search path
  - [ ] 03.10.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmEmbedder -count=1` — Expected: PASS
  - [ ] 03.10.e Commit — `feat(mnemonic): warm embedder load-once cache`
- [ ] 03.11 `[RED]` Warm embedder lifecycle — evicts after idle_timeout, heartbeat keeps alive (Scenario: Warm embedder evicts on idle and survives heartbeat) — threat: Warm embedder lifecycle
  - [ ] 03.11.a Write failing test — after `idle_timeout` (default 10min, tunable) of no use the warm embedder is evicted from RAM; an active MCP client connection keeps it alive via heartbeat (no eviction while connected); `off`/external providers hold no RAM
  - [ ] 03.11.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmEvict -count=1` — Expected: FAIL
  - [ ] 03.11.c Minimal implementation — `idle_timeout` evict timer + MCP-heartbeat keep-alive in `embedder/warm.go` + `server.go`
  - [ ] 03.11.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmEvict -count=1` — Expected: PASS
  - [ ] 03.11.e Commit — `feat(mnemonic): warm embedder idle-evict and heartbeat keep-alive`
- [ ] 03.12 `[AFK]` Absent or failed model degrades to FTS and signals (Scenario: Absent model degrades to FTS and signals) — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... -run WarmFallback -count=1` — Expected: PASS
- [ ] 03.13 `[AFK]` Evicted-then-reused embedder reloads transparently (Scenario: Evicted embedder reloads on reuse) — `Run: go test ./skillgrid-cli/internal/mnemonic/embedder/... -run WarmReload -count=1` — Expected: PASS
- [ ] 03.14 `[AFK]` `skillgrid index status` pending-sync section (Scenario: index status shows pending sync) — `Run: go test ./skillgrid-cli/cmd/skillgrid/... -run IndexStatus -count=1` — Expected: PASS
- [ ] 03.15 `[RED]` Mnemonic tool surface — watcher alters no tool schemas, banner present, bad args rejected (Scenario: Watcher keeps tool schemas stable) — threat: Mnemonic tool surface
  - [ ] 03.15.a Write failing test — assert the watcher + fingerprint gate + banner do not alter any existing 005/008/new `code_*` tool schemas; the staleness banner is present on the response path; bad watcher/status args are rejected with a clear validation error
  - [ ] 03.15.b Run to confirm fail — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run WatcherSurface -count=1` — Expected: FAIL
  - [ ] 03.15.c Minimal implementation — schema-stability assertions + banner/arg-validation wiring
  - [ ] 03.15.d Run to confirm pass — `Run: go test ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -run WatcherSurface -count=1` — Expected: PASS
  - [ ] 03.15.e Commit — `feat(mnemonic): watcher tool-surface stability and status`

### Verification

Verdict: `PENDING`

Evidence:

| Check | Run | Expected | Result | Notes |
|-------|-----|----------|--------|-------|
| Focused test | `go test ./skillgrid-cli/internal/mnemonic/codeindex/... ./skillgrid-cli/internal/mnemonic/embedder/... ./skillgrid-cli/internal/mnemonic/mcp/... ./skillgrid-cli/cmd/skillgrid/... -count=1` | PASS | | |
| Acceptance `@step-03` / `@p0` | BDD / mapped unit scenarios | PASS | | |
| Runtime harness | `skillgrid serve` + edit a file mid-session; confirm auto-reopen + banner; `SKILLGRID_NO_WATCH=1` then edit + query to confirm the fingerprint gate re-indexes structurally (embedder untouched); `skillgrid index status` pending section | PASS | | |
| Rollback boundary | Drop `watch.go`/`fingerprint.go`/`publish.go`/`warm.go` + banner/heartbeat/auto-reopen in serve path; revert `go.mod` + `main.go` | PASS | | |
| Global Constraints | — | held | | |

### Commit

When step DoD is met: `feat(mnemonic): auto-sync watcher, fingerprint gate, copy-and-swap, and warm embedder`

---

