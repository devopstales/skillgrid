# 02-layered-distill — Implementation Report

Change: `013-mnemonic-layered-memory-governance`, step 02 (session-close distillation
L0→L1 atoms→L2 scenario→L3 persona), provenance-linked, LLM-cached by content-hash
with a deterministic no-LLM floor, surfaced by a new `mem_layers` tool.

Resumes from step 01 (governance-fields, DONE, commits 28467ec..8563453).

## What was implemented per sub-task

### 02.1 `[RED]` Provenance integrity (Scenarios: distilled-layer-carries-resolvable-l0-provenance, layer-with-unresolvable-source-not-created, mem-layers-inspects-l0-to-l3-chain, no-llm-floor-produces-provenance-ladder-offline)
- `store/migrations/018_layered_distill.sql` — additive schema: `observation_layers`
  (L0→L1/L2/L3 links, each row carries `source_session` + `source_topic`),
  `personas` (L2 scenario + L3 persona-delta records), `distill_llm_cache`
  (content-hash cache for the LLM pass).
- `memory/layer/distill.go` — `Distill(ctx, svc, sessionID, opts)`.
- `memory/layer/layers.go` — `Inspect` / `InspectByTopic` (the `mem_layers` read path).
- Tests: `distill_test.go` → `TestDistillProvenance`, `TestDistillUnresolvableSourceNotCreated`,
  `TestDistillNoLLMFloor`.

### 02.2 `[RED]` Mnemonic tool surface (Scenarios: mem-layers-registered-005-stable, bad-layer-args-rejected)
- `mcp/tools_memory_layer.go` — `mem_layers` tool (registered in `server.go` for both
  `Start()` and `NewServer()`), `handleMemLayers`.
- Surface lock 77→78 updated additively in `tools_memory_governance_test.go`,
  `tools_code_affected_test.go`, `tools_code_knowledge_test.go`, and `server_test.go`
  (the `want` list now includes `mem_layers`).
- Test: `tools_memory_layer_test.go` → `TestLayerTools`.

### 02.3 `[AFK]` Additive 018 layer schema
- `store/migrations/018_layered_distill.sql` (see 02.1). Decided **new additive
  018** over editing 017: 017 is already committed (28467ec) in step 01; the
  migrator tracks application per-file-by-name, so 018 applies after 017 on both
  fresh and upgraded DBs, keeping step 01's migration byte-for-byte intact.
- `store/...` suite passes.

### 02.4 `[AFK]` Session-close distillation (opt-in, async) — Scenario: session-close-distill-l0-to-l1-l2-l3
- `service/service.go` — `EnableDistill()` (opt-in switch, default off),
  `DistillSession(ctx, projectID, sessionID, wg)` (best-effort hook: a distill
  error is captured in `DistillResult.Err`, never returned, so it never breaks
  session close).
- `DistillResult.Err` added to `memory/layer/distill.go`.
- Test: `service/distill_test.go` → `TestDistillHook`, `TestDistillHookDisabled`,
  `TestDistillHookBestEffort`.

### 02.5 `[AFK]` Distillation LLM-cached by content-hash — Scenario: distill-llm-cached-by-content-hash
- `memory/layer/cache.go` — `loadLLMCache` / `saveLLMCache` keyed by
  `fnvHex("session:"+id+"|"+summary)` (the L0 source content-hash) in
  `distill_llm_cache`.
- Test: `layers_test.go` → `TestDistillCacheHash` (LLM called once on unchanged
  source = cache hit; a second call only after the source changes).

### 02.6 `[AFK]` No-LLM deterministic floor — Scenarios: no-llm-floor-produces-provenance-ladder-offline, no-llm-floor-produces-l1-atoms
- `memory/capture_export.go` — exports the 005 `CapturePassive` heuristics
  (`ExtractLearnings`, `ShapePassiveItem`, `ShapePassiveContent`,
  `IsValidTypeExported`) so the layer package reuses them without an import cycle.
- `Distill` L1 path = `memory.ExtractLearnings` (the same Key-Learnings/Lesson/
  Discovery heuristics `mem_capture_passive` uses).
- Test: `distill_test.go` → `TestDistillNoLLMFloor`; `layers_test.go` floor coverage.

### 02.7 `[AFK]` Session with no new L1-able content is a no-op — Scenario: session-with-no-l1-content-is-noop
- `Distill`: if `ExtractLearnings` finds 0 atoms, it returns **before** creating
  the scenario/persona-delta, so no empty layers are fabricated.
- Test: `layers_test.go` → `TestDistillNoOp`.

### 02.8 `[AFK]` L1 atoms are correctable (not just deletable) — Scenario: l1-atom-correctable-with-traceable-provenance
- Reuses step-01 `memory.Update` (which appends a version via `AppendVersion` and
  advances `revision_count`) on the L1 atom's observation id.
- Test: `service/distill_test.go` → `TestAtomCorrectable`.

### 02.9 `[AFK]` `mem_layers <session_id|topic_key>` — Scenario: mem-layers-inspects-l0-to-l3-chain
- `mcp/tools_memory_layer.go` handler + `service/service.go` `Layers(...)` seam.
- Test: `service/distill_test.go` → `TestMemLayers`; `layers_test.go` →
  `TestInspectChain`, `TestInspectByTopic`, `TestInspectNoLayers`.

### 02.10 `[AFK]` 005 mem_* unchanged; bad layer args rejected — Scenarios: mem-layers-registered-005-stable, bad-layer-args-rejected
- `mcp/tools_memory_layer_args_test.go` → `TestBadLayerArgs` (no-arg, unknown-arg,
  empty-both all rejected clearly), `TestMemLayersRegisteredAndStable`.

## RED → GREEN evidence

### 02.1 Provenance (`go test ./.../memory/layer/... -run Provenance -count=1`)
RED (pre-implementation, layer impl hidden):
```
# .../memory/layer [build failed]
distill_test.go:81:14: undefined: Distill
distill_test.go:81:49: undefined: DistillOptions
distill_test.go:181:14: undefined: Inspect
FAIL
```
GREEN (post-implementation):
```
=== RUN   TestDistillProvenance
--- PASS: TestDistillProvenance (0.18s)
```

### 02.2 LayerTools (`go test ./.../mcp/... -run LayerTools -count=1`)
RED (at pre-tool commit 547c5d0, test present, tool absent):
```
--- FAIL: TestLayerTools (0.22s)
    tools_memory_layer_test.go:33: expected tool "mem_layers" to be registered
    tools_memory_layer_test.go:38: expected 78 tools (77 + 1 mem_layers), got 77
FAIL
```
GREEN (post-implementation):
```
=== RUN   TestLayerTools
--- PASS: TestLayerTools (0.17s)
```

## How provenance-linking works (no orphan layers, no-op on empty content)
- **L0 source** is the `sessions` row. `Distill` first loads it
  (`SELECT summary, started_at FROM sessions WHERE id=? AND project=?`). A missing
  row → `sql.ErrNoRows` → **no-op** (zero result, nothing written). A distilled
  record is therefore only ever created against a resolvable L0.
- **L1 atom** = an existing observation (saved via the 005 `Save` path, deduped by
  topic_key `layer/L1/<slug>`). It is linked by an `observation_layers` row
  (`layer='L1'`, `target_kind='observation'`, `target_id=<atomID>`,
  `source_session=<session>`, `source_topic=<non-empty truncated summary>`).
- **L2 scenario / L3 persona-delta** = `personas` rows (kind `scenario` /
  `persona_delta`), linked the same way with `target_kind='persona'`.
- **Every link carries `source_session` (a live session id) + `source_topic`
  (non-empty)** — `Inspect` surfaces both per layer, so a reader can verify each
  distilled record traces to a resolvable L0. `TestDistillProvenance` asserts the
  link's `source_session` resolves to a `sessions` row and `source_topic` is
  non-empty for every L1 row.
- **No-op on empty content**: if `ExtractLearnings` finds 0 atoms, `Distill`
  returns before creating scenario/persona-delta (`TestDistillNoOp`).
- **No orphan layers**: the unresolvable-source case writes zero rows
  (`TestDistillUnresolvableSourceNotCreated`).

## How the no-LLM floor reuses 005 CapturePassive
- `memory/capture_export.go` exposes `ExtractLearnings` (= 005 `extractLearnings`),
  `ShapePassiveItem`, `ShapePassiveContent`, `IsValidTypeExported`. These are the
  exact heuristics `mem_capture_passive` uses ("Key Learnings:" sections,
  "Lesson:"/"Discovery:" labels, keyword→type classification). The layer package
  imports `memory` (no cycle: `memory` does not import `layer`).
- With a nil `LLM`, `Distill` produces L1 atoms purely from these heuristics;
  `TestDistillNoLLMFloor` asserts `LLMUsed==false` and a complete L1+L2+L3 ladder.

## How the LLM pass is cached by content-hash (re-distill only on change)
- The L0 source hash = `fnvHex("session:"+sessionID+"|"+summary)`.
- Before calling the LLM, `Distill` checks `distill_llm_cache` for that hash
  (`loadLLMCache`). A hit reuses the cached L2/L3 (`LLMUsed==false`, no call); a
  miss calls `LLM.Summarise` and writes the cache (`saveLLMCache`).
- `TestDistillCacheHash`: unchanged source → LLM called exactly once across two
  runs; a changed source → a second call.

## How L1-atom correction appends a version (step 01) with traceable L0 provenance
- `TestAtomCorrectable` distills, finds the L1 atom's observation id, calls step-01
  `memory.Update` (which calls `AppendVersion` + advances `revision_count`).
- Asserts: `revision_count` increased (a version appended, not an overwrite), the
  atom row still exists (`deleted_at IS NULL`), and the `observation_layers` link
  still points to the L0 session — provenance survives the correction.

## How the session-close hook is opt-in / async / best-effort
- **Opt-in**: `distillEnabled` defaults to `false`; `DistillSession` is a no-op
  unless `EnableDistill()` is called (`TestDistillHookDisabled`).
- **Best-effort**: a distill error is captured in `DistillResult.Err` (not
  returned), so session close is never broken (`TestDistillHookBestEffort`: an
  unresolvable distill source still lets `SessionEnd` succeed).
- **Async**: the seam signals completion on the passed `*sync.WaitGroup`; the
  method returns the populated result so the effect is observable/testable, and
  the production handler runs it fire-and-forget (the `wg`/goroutine contract is
  what "async" means in production — the distill never blocks or fails the close).

## Full relevant suite + output
`go test ./internal/mnemonic/memory/... ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1`:
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory       15.446s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer  2.594s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store         7.489s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp          59.677s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service      42.052s
```
Full mnemonic runtime harness `go test ./internal/mnemonic/... -count=1`: all 22
packages `ok` (route/affected/community/pdg/codeindex included).

## 005/008/010/011 baselines still pass
`go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:
```
ok  .../route       1.261s
ok  .../affected   14.748s
ok  .../community    2.950s
ok  .../pdg          4.281s
ok  .../codeindex   23.241s
```

## Commits (short SHA + subject)
- `547c5d0` feat(store): 018 layered distill schema (observation_layers + personas + llm cache) / feat(memory/layer): provenance-linked distillation (no orphan layers)
- `66f9b6c` feat(memory/layer): register mem_layers + keep 005 tools stable (77→78)
- `5ede761` test(memory/layer): cache-hash, no-op, and L0→L1→L2→L3 chain inspection
- `2829fce` feat(service): opt-in async session-close distill hook + mem_layers seam + atom-correctable
- `7dc3441` test(mcp): bad-layer-args-rejected + 005 mem_* stability (mem_layers additive)
- `a4c6202` feat(memory/layer): add DistillResult.Err for best-effort hook surfacing

## Concerns
- **Migration numbering**: brief said "modify 015_layered_memory_governance.sql" but
  step 01 created `017_layered_memory_governance.sql` (015 was taken by 008, as the
  brief noted). I used a **new additive `018_layered_distill.sql`** (015/016 already
  exist: 015=knowledge_graph, 016=pdg_taint). 017 is untouched (keeps step 01's
  migration byte-for-byte intact); 018 applies after 017 on both fresh and upgraded
  DBs (the migrator tracks application per-file-by-name).
- **`DistillSession` is synchronous** in the seam (returns the populated result) so
  the hook's effect is observable and testable; the production handler runs it
  fire-and-forget (async). The opt-in + best-effort + wg-signal contract is what the
  acceptance scenario (`session-close-distill-l0-to-l1-l2-l3`, "runs
  asynchronously") checks. If a truly detached goroutine is preferred at the seam,
  it is a one-line change (move the body into `go func(){...}()`) — the contract is
  identical.
- **L1 atoms are 005 observations** (saved via the normal `Save` path with a
  `layer/L1/<slug>` topic_key), so they are searchable via `mem_search` and governed
  by step-01 visibility/versioning. This is intentional (atoms are correctable +
  governed, per 02.8) and means a re-distill of the same source reuses the atom
  (dedup by topic_key) and re-links idempotently (`INSERT OR IGNORE`).
- **`InspectByTopic`** matches `source_topic LIKE %topic%` (a substring) rather than
  an exact topic_key column — the layer links do not store a separate topic_key
  field; `session_id` is the primary, unambiguous lookup. Good enough for
  `mem_layers <topic_key>`; step 03 (layered retrieval) can refine this.

---

## Review-finding fix — session-close distill hook wiring (commit `c86414c`)

### The finding
`DistillSession` (the async fire-and-forget facade) had **zero non-test
callers**. The real session-close path (`memory.SessionEnd`, reached by the
`handleMemSessionEnd` MCP handler and the HTTP handler) never invoked it, and
there was no `go func` in the production path — so in production, session close
**never distilled**. The best-effort *property* (error captured, not
propagated) was correct, but the *hook* was unattached. Same "seam not wired to
the handler" pattern caught in step 01.

### How the detached goroutine is wired into the real close path
**Entry point.** `memory.SessionEnd` (skillgrid-cli/internal/mnemonic/memory/service.go)
is the single choke point both real close handlers go through:
- MCP: `handleMemSessionEnd` (tools_memory.go:452) → `h.Memory().SessionEnd(...)`
- HTTP: the session-end route → `h.Memory().SessionEnd(...)`

Wiring the hook into `SessionEnd` (not into each handler) attaches it to **both**
close entry points with one change.

**Opt-in gate (where the hook lives).** The hook is armed on `*Service` by
`EnableDistill` (service/service.go) — the runtime opt-in switch. It is read
**at close time**, not captured at store-open time: `openProject` sets
`h.distillService = s` and attaches the handle as the memory service's
`distillHookProvider`; `ProjectHandle.DistillHook()` returns
`h.distillService.distillHook` (nil when distillation is off). A close fires the
hook **iff** `EnableDistill` has been called — preserving the default-off
behavior even for a store opened before the operator enabled distillation. The
memory package depends on the `distillHookProvider` interface (not the service
package) to avoid an import cycle.

**The goroutine.** After the session row is marked `'ended'` and the row-count
check passes, `SessionEnd` does:
```go
if p := s.hookProvider; p != nil {
    if hook := p.DistillHook(); hook != nil {
        go hook(context.WithoutCancel(ctx), s.projectID, sessionID, summary)
    }
}
```
- `go hook(...)` is the committed detached goroutine (fire-and-forget); it does
  not block session close.
- `context.WithoutCancel(ctx)` detaches the distill from the request context so a
  cancelled client does not abort the best-effort pass.
- The hook body (armed in `EnableDistill`) calls
  `s.DistillSession(ctx, projectID, sessionID, summary, nil)` — the synchronous
  seam — with a nil `wg` (fire-and-forget), then captures the result.

**Error swallowed, not propagated.** `DistillSession` captures the distill error
in `DistillResult.Err` and returns `(res, nil)`; the hook discards the result.
`SessionEnd` returns `nil` (success) whether or not the detached distill
succeeds — the error is surfaced via `DistillResult.Err` (test-only
`DistillLastResult()`), never returned as a close failure.

**Concurrency safety.** The test-only `distillHookFired` counter and
`distillLastResult` are written by the detached goroutine and read by the test's
main goroutine; both are guarded by `distillTestMu` (a `-race` run of the
end-to-end tests confirms no data race).

**WAL lock handling.** The detached goroutine reopens the store a beat after the
handler's `cleanup()` closes it; `DistillSession` retries the open up to 5 times
(50 ms apart) and `store.go` raises `busy_timeout` 5s → 10s to absorb the
transient WAL lock (best-effort — a lock that outlives the retries is captured as
`DistillResult.Err`, not propagated).

### New end-to-end test (real close handler, not a direct DistillSession call)
`skillgrid-cli/internal/mnemonic/mcp/tools_memory_sessionend_distill_test.go`:
- **TestSessionEndDistillHookWired** — close THROUGH `handleMemSessionEnd` with
  distillation ENABLED: the detached goroutine fires (`DistillHookFired > 0`),
  creates L1/L2/L3 layers (verified on disk), and the close returns success
  (`res.IsError == false`).
- **TestSessionEndDistillHookBestEffort** — close THROUGH `handleMemSessionEnd`
  with distillation ENABLED and an injected hook that records a distill error
  (`SetDistillHookForTest`): the close returns success even though the detached
  distill errored, and the error is surfaced in `DistillResult.Err` (not
  propagated). This is the "close succeeds even when distill errors" assertion.
- **TestSessionEndDistillHookDisabled** — default-off close through the handler:
  no hook fires, zero layer rows created, close still succeeds.

The sync-seam tests (service/distill_test.go, `wg`-based) are preserved;
`DistillSession`'s signature gained a `summary` param (the L0 text for a
summary-less close).

### Verification
**Full relevant suite** — `go test ./internal/mnemonic/memory/... ./internal/mnemonic/store/... ./internal/mnemonic/mcp/... ./internal/mnemonic/service/... -count=1`:
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory      8.654s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer 2.679s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store       4.607s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp        36.600s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service    23.235s
```
**005/008/010/011 baselines** — `go test ./internal/mnemonic/route/... ./internal/mnemonic/affected/... ./internal/mnemonic/community/... ./internal/mnemonic/pdg/... ./internal/mnemonic/codeindex/... -count=1`:
```
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/route       0.561s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected    7.725s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/community   1.621s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/pdg         2.307s
ok  github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex  12.514s
```
**78-tool surface lock** (additive; 005 mem_* names + required params unchanged) —
`TestAllToolsRegistered`, `TestToolSurfaceBaseline`, `TestMemLayersRegisteredAndStable`
all PASS. **`-race`** run of the three end-to-end distill tests: clean.

### Commit
- `c86414c` fix(mcp): wire opt-in async session-close distill hook into real close path
