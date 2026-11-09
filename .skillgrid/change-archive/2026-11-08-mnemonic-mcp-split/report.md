# Report — mnemonic MCP family split

> Change: `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/` (moves to `.skillgrid/change-archive/2026-11-08-mnemonic-mcp-split/` at ship)
> Generated: 2026-11-08T19:05:00+01:00 (qa)
> Gate: PASS (re-rendered 2026-11-08 after W001 fix `0b9e65cd`)
>
> Two phases, one file: **qa** writes the QA half (the sections above `## Final-State Facts`,
> through `## Gate Decision` + `## Human Override`) into the spec folder. **ship** reads the
> `## Gate Decision` verdict PRE-MOVE, then moves the folder. **reflect** completes the retro
> half (from `## Final-State Facts` onward) IN PLACE in the archive folder.
>
> **Effective rigor:** Tier T2 (blueprint) with the high-risk classification floor L4 in force
> (full suite + build + coverage + mutation sampling + Trivy advisory) — the classification
> wins where it exceeds the tier's L2+L3 floor. `testing.tdd: false` (Standard mode) — TDD
> evidence is the feature-commit → passing-gate chain (precedent: 2026-10-31-mnemonic-v2-code-index report).
> **QA-run discovery:** the verification pass found and fixed one real protocol defect — mcp-go's
> queue-full synchronous fallback allowed 2 tool calls in flight at 3+ concurrent with
> `WithQueueSize(1)` (the deep-concurrency scenario went RED, maxSeen=2). Fixed at `10e4206b`
> (`WithQueueSize(10000)` = mcp-go maxQueueSize; test extended to a 128-call deep phase).

## Test Plan

> Derived from `blueprint.md`, `tasks.md`, and `acceptance.feature`.
> Risk-ordered. Every row must be covered by a named test or scenario before the gate passes.

### Risk Ranking

The most likely production break: a client that mounts the server by name/capabilities or
calls an old CLI path — the cutover is clean (no aliases), so the single biggest risk is a
wrong tool/verb set reaching a client (isolation leak or stale command in a fresh install).
Second: the serial-stdio protocol invariant — response interleaving or concurrent processing
corrupts a client that correlates by id; this is where the QA run itself found a real defect
(mcp-go's queue-full synchronous fallback, see Findings). Third: the code response contract —
the quality floor (blast radius + verbatim source) regressing to a thinner shape without any
error.

### Plan

| # | Risk / Behavior | Seam | Layer | Test / Scenario | Priority | Cadence | Status |
|---|-----------------|------|-------|-----------------|----------|---------|--------|
| 1 | Old CLI paths dispatch or alias silently / new paths misroute | CLI dispatch (`cmd/mnemonic`) | unit | `cli_family_test.go::TestFamilyDispatch`, `TestOldPathsSuggestTheirReplacement`, `TestOldSubcommandPathsRefuse`, `TestBareCodeDefaultsToIndex` | P0 | pr | covered |
| 2 | Rename leaves `mnemonic-v2` module references (build drift) | go.mod / go.work / Taskfile | build + grep gate | workspace build + full suite at `mnemonic/` (row 15) + grep of go.mod/go.work/Taskfile (live: 0 hits) | P0 | pr | covered |
| 3 | Tool isolation leak (a client sees another family's tools) | MCP `tools/list` per family × profile | integration (in-process stdio) | `server_family_test.go::TestToolIsolationPerFamily`, `contract_test.go::TestAllProfileIsAgentPlusAdmin` | P0 | pr | covered |
| 4 | Serial stdio broken (concurrent processing / interleaved responses) | stdio loop (`StartOn`) | integration (in-process stdio) | `server_family_test.go::TestSerialToolCalls` (128-call deep phase; the WithWorkerPoolSize mutation gate) | P0 | pr | covered |
| 5 | Non-protocol bytes reach stdout post-startup | stdio loop + `Logf` stderr seam | integration (in-process stdio) | `server_family_test.go::TestStdoutProtocolOnly` | P0 | pr | covered |
| 6 | Parallel family servers lock/double-push the store | SQLite store + sync lease across processes | integration (two servers, one store) | `server_family_test.go::TestTwoServersOneStoreParallel` | P0 | pr | covered |
| 7 | Full-suite race regression (both modules) | workspace | integration | `go test ./... -race` (mnemonic + skillgrid-cli) | P0 | pr | covered (named exception: 4 pre-existing v1 failures — see Findings) |
| 8 | initialize shape drift vs the baselines (identity / capabilities / instructions) | MCP `initialize` per family | integration (in-process stdio) | `server_family_test.go::TestInitializeShapePerFamily` (byte-identical pins vs the engram/openwiki baselines) | P1 | pr | covered |
| 9 | Unknown profile accepted | CLI mcp-verb flag parse | unit | `cli_family_test.go::TestUnknownProfileRefuses` | P1 | pr | covered |
| 10 | `code_search` hit loses blast radius / snippet (quality floor) | `CodeHandler.Search` response | unit | handler `TestSearchHitBlastRadiusShape`, `TestSearchUnresolvedHitDegrades`, `TestSearchDuplicateEdgesOnce` | P1 | pr | covered |
| 11 | `code_explore` loses a baseline element / lists a duplicate edge | `RenderExplore` | unit | handler `TestExploreResponseShape` (the mutation gate), `TestExploreTagsInferredEdges`, `TestExploreEmptyIndex`, `TestExploreDegradedLeg` | P1 | pr | covered |
| 12 | Stale old-path references survive in live skills/docs/vault | repo convention files | unit (grep boundary test) | `cmd/mnemonic::TestNoStalePathReferences` | P1 | pr | covered |
| 13 | Fresh install writes the dead bare `mnemonic mcp` form | installer config writers | unit | `skillgrid-cli/internal/setup` hermes/protocol/cursor tests (family command-line assertions) | P1 | pr | covered |
| 14 | Architecture page anchors stale vs the new tree | projectwiki pin-check | manual gate (CLI) | `mnemonic projectwiki validate ARCHITECTURE.md` (same-edit reindex) | P1 | pr | covered |
| 15 | ADR missing / mis-indexed / number collision | ADR invariants | unit (node script) | `node scripts/check-adr-invariants.mjs` | P1 | pr | covered |
| 16 | mcp-go pool option mutates (serial invariant unguarded) | `StartOn` pinned options | mutation sampling | live mutation: drop `WithWorkerPoolSize(1)` → `TestSerialToolCalls` fails (RED, this session) | P0 | pr | covered |
| 17 | explore renderer mutates (shape unguarded) | `RenderExplore` | mutation sampling | live mutation: drop the blast-radius section → `TestExploreResponseShape` fails (RED, this session) | P1 | pr | covered |

**Layer** — unit where the seam is in-process (CLI dispatch, renderer, handler); integration for the stdio-loop and cross-process behaviors (the layer list is `[unit, integration]`; the e2e layer is unavailable, and the in-process stdio loop over pipes is the highest available layer for the protocol invariants). Duplicate Coverage Guard: the CLI refuse cases (row 1) are not re-covered at the stdio layer; the isolation counts (row 3) are not re-covered by the initialize pins (row 8).

**Cadence** — all `pr`: every row is a merge blocker for a public-contract cutover; none is a regression net only.

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| CLI cutover | removed paths (`mnemonic index` / `mem save` / `mcp` / `serve`) | non-zero exit + did-you-mean naming the new path | `TestOldPathsSuggestTheirReplacement`, `TestOldSubcommandPathsRefuse` | covered |
| CLI cutover | bare `mnemonic code` (no sub-verb) | runs the index (default verb unchanged) | `TestBareCodeDefaultsToIndex` | covered |
| CLI cutover | unknown `--profile` on a family mcp verb | non-zero usage refusal naming the valid profiles | `TestUnknownProfileRefuses` | covered |
| serial stdio | 128 tool calls in flight before any response (beyond the queue) | max in-flight = 1 | `TestSerialToolCalls` deep phase | covered |
| code_search | hit whose symbol did not resolve | blast-radius fields omitted, no error | `TestSearchUnresolvedHitDegrades` | covered |
| code_search | same caller edge recorded twice | caller counted exactly once | `TestSearchDuplicateEdgesOnce` | covered |
| code_explore | empty index / non-git directory | `Found 0 symbols across 0 files.`, no source blocks, no degraded flag, no error | `TestExploreEmptyIndex` | covered |
| code_explore | one search leg degraded | degraded flag naming the leg | `TestExploreDegradedLeg` | covered |
| code_explore | INFERRED call edge | tagged `(inferred)`; EXTRACTED edges untagged | `TestExploreTagsInferredEdges` | covered |
| store | two family servers on one store, autosync on | ≤1 sync-lease holder; no `database is locked` | `TestTwoServersOneStoreParallel` | covered |
| store | fresh-store TOCTOU (two processes race the lease) | single winner, sequenced like `TestAutosyncSingleLease` | `TestTwoServersOneStoreParallel` (sequenced phase) | covered |

### Out of Scope

- **v1 `skillgrid-cli` behavior** beyond the installer's written command strings — v1 is untouched by design (R1).
- **HTTP transport** — v2's `serve` is removed; no HTTP surface exists to test (out of scope per briefing).
- **Store/index semantics** — unchanged by design; R6 is response-shape only over the existing graph.
- **MCP client behavior** — mcp-go internals beyond the pinned options (worker pool / queue size) are the library's contract, tested only at the seam.
- **Embedding provider availability** — the vector leg's degradation path is exercised (`TestExploreDegradedLeg`); a live Ollama endpoint is deployment state, not this change.

## Goal-Backward Verification

**Stated goal** (from briefing.md): split the surface into three single-purpose MCP servers under a family-first CLI, rename the module, make responses strictly serial, and raise the code-family tool responses to the quality bar of the gitnexus/CodeGraph baselines.

**Assumption:** The goal was NOT achieved until the evidence below proves it.

| Level | Item | Evidence | Status |
|-------|------|----------|--------|
| Truth | three single-purpose servers, one mounted per client, each with its baseline initialize + family-only tools | `TestInitializeShapePerFamily` + `TestToolIsolationPerFamily` (fresh -count=1) + live stdio smoke on the built binary: `mnemonic-memory` 0.2.0 / `{"tools":{"listChanged":true}}` / 22 tools (agent) / 28 (all); `mnemonic-code` 0.2.0 / `{prompts:{},resources:{},tools:{}}` / 0B instructions / 20 tools; `mnemonic-projectwiki` 0.2.0 / `{"tools":{"listChanged":true}}` / 8 tools | VERIFIED |
| Truth | family-first CLI with clean cutover (no aliases; did-you-mean on removed paths) | `TestFamilyDispatch`, `TestBareCodeDefaultsToIndex`, `TestOldPathsSuggestTheirReplacement`, `TestOldSubcommandPathsRefuse` + live binary smoke: all four old paths exit 2 with `"<path>" was renamed: mnemonic …` | VERIFIED |
| Truth | responses strictly serial, stdout protocol-only | `TestSerialToolCalls` (128 calls in flight, maxSeen=1), `TestStdoutProtocolOnly` + live mutation sampling (pool option removed → RED) + live smoke (stdout = protocol frames only) | VERIFIED |
| Truth | code responses equivalent-or-better (per-hit blast radius + snippet; code_explore composite at CodeGraph baseline) | `TestSearchHitBlastRadiusShape`, `TestSearchUnresolvedHitDegrades`, `TestSearchDuplicateEdgesOnce`, `TestExploreResponseShape`, `TestExploreTagsInferredEdges`, `TestExploreEmptyIndex`, `TestExploreDegradedLeg` + live mutation sampling (blast-radius section dropped → RED) | VERIFIED |
| Truth | parallel family servers share one store safely (no double-push, no database-locked) | `TestTwoServersOneStoreParallel` (two processes, one store file, ≤1 lease holder) | VERIFIED |
| Truth | module rename mechanical and complete | full suite green at `mnemonic/` (18/18 packages, -race, this session) + grep gate go.mod/go.work/Taskfile = 0 hits + v1 module builds without the renamed path | VERIFIED |
| Truth | the decision is recorded, indexed, and the live surface points at the new paths | `check-adr-invariants.mjs` OK (58 files/59 rows), `TestNoStalePathReferences`, setup tests, `projectwiki validate ARCHITECTURE.md` ok | VERIFIED |
| Artifact | `mnemonic/` module + `cmd/mnemonic` family dispatch | suite ran at the new path; `main.go` run() 78.4% covered; dispatch tests green | VERIFIED |
| Artifact | `StartOn` pinned serial options + `Logf` stderr seam | `server.go` 90.1% covered; the two stdio invariants tested over the production loop | VERIFIED |
| Artifact | `RenderExplore` + enriched `CodeHandler.Search` | `explore_format.go` 94.6% covered; `handler_code.go` changed paths exercised by the T4/T5 pins | VERIFIED |
| Key Link | CLI mcp verb → per-family server (family threading through `runMCP`) | `TestFamilyDispatch` + live smoke of all three family servers | VERIFIED |
| Key Link | handler → MCP dispatch envelope (`{"response": …}`) | `TestExploreResponseShape` (envelope + shape) | VERIFIED |
| Data Flow | initialize/tools/list frames over real stdio | live binary smoke, 3 servers + the all-profile variant | VERIFIED |
| Data Flow | 128-call burst over the stdio pipe, serial drain, all 128 responses | `TestSerialToolCalls` deep phase (exact-ID match per response) | VERIFIED |
| Data Flow | two processes on one store: lease election, no lock errors | `TestTwoServersOneStoreParallel` | VERIFIED |

**Status definitions:** `VERIFIED` — a test exercises this at the behavior level and passed. `PRESENT_BEHAVIOR_UNVERIFIED` — code exists but no test exercises the state transition. `UNVERIFIED` — no evidence found. (No PRESENT_BEHAVIOR_UNVERIFIED or UNVERIFIED rows.)

## Traceability Matrix

| Scenario (from acceptance.feature) | Test / Scenario (from test plan) | Ran | Result |
|--------------------------------------|----------------------------------|-----|--------|
| rename-builds-and-tests-green-at-new-path | full suite at `mnemonic/` (-race, this session) | yes | pass |
| rename-leaves-no-module-references | grep gate go.mod/go.work/Taskfile (0 hits) + suite at new path | yes | pass |
| rename-keeps-v1-untouched | `skillgrid-cli` build + setup suite (no import of renamed path) | yes | pass |
| new-family-paths-dispatch | `TestFamilyDispatch` | yes | pass |
| bare-code-defaults-to-index | `TestBareCodeDefaultsToIndex` | yes | pass |
| old-paths-suggest-their-replacement | `TestOldPathsSuggestTheirReplacement` + live binary smoke | yes | pass |
| serve-is-gone | `TestOldPathsSuggestTheirReplacement` (the `serve` case) + live smoke | yes | pass |
| memory-server-serves-memory-only | `TestToolIsolationPerFamily` | yes | pass |
| all-profile-adds-admin-tools-only | `TestToolIsolationPerFamily` (all rows) + `TestAllProfileIsAgentPlusAdmin` + live smoke (28) | yes | pass |
| code-and-wiki-servers-are-isolated | `TestToolIsolationPerFamily` | yes | pass |
| unknown-profile-refuses | `TestUnknownProfileRefuses` | yes | pass |
| memory-initialize-follows-engram | `TestInitializeShapePerFamily` + live smoke (identity/caps/instr 2597B) | yes | pass |
| projectwiki-initialize-follows-openwiki | `TestInitializeShapePerFamily` + live smoke (instr 2590B) | yes | pass |
| code-initialize-follows-gitnexus | `TestInitializeShapePerFamily` + live smoke (caps `{prompts:{},resources:{},tools:{}}`, 0B instructions) | yes | pass |
| concurrent-calls-answer-one-at-a-time | `TestSerialToolCalls` | yes | pass |
| deep-concurrency-stays-serial | `TestSerialToolCalls` deep phase (128 calls) | yes | pass |
| stdout-carries-only-protocol-frames | `TestStdoutProtocolOnly` + live smoke | yes | pass |
| search-hit-carries-blast-radius-and-snippet | `TestSearchHitBlastRadiusShape` | yes | pass |
| unresolved-hit-degrades-gracefully | `TestSearchUnresolvedHitDegrades` | yes | pass |
| duplicate-edges-are-reported-once | `TestSearchDuplicateEdgesOnce` | yes | pass |
| explore-returns-the-full-baseline-shape | `TestExploreResponseShape` | yes | pass |
| explore-tags-inferred-edges | `TestExploreTagsInferredEdges` | yes | pass |
| explore-on-empty-index-degrades-silently | `TestExploreEmptyIndex` | yes | pass |
| explore-declares-a-degraded-leg | `TestExploreDegradedLeg` | yes | pass |
| two-servers-one-store-parallel | `TestTwoServersOneStoreParallel` | yes | pass |
| no-stale-path-references-remain | `TestNoStalePathReferences` | yes | pass |
| fresh-install-writes-live-mcp-commands | skillgrid-cli setup suite (hermes/protocol/cursor family-command assertions) | yes | pass |
| architecture-page-matches-the-new-tree | `projectwiki validate ARCHITECTURE.md` → ok (not_grounded) | yes | pass |
| adr-lands-and-indexes | `check-adr-invariants.mjs` → OK 58 files/59 rows | yes | pass |
| full-suite-green-race | `go test ./... -race` both modules — mnemonic 18/18 packages green; skillgrid-cli all packages green (4 pre-existing `internal/install` failures fixed in re-verification, `0b9e65cd`) | yes | pass |
| mutation-of-serial-option-is-caught | live mutation: `WithWorkerPoolSize(1)` removed → `TestSerialToolCalls` FAIL (RED, this session) | yes | pass |
| mutation-of-renderer-is-caught | live mutation: blast-radius section removed → `TestExploreResponseShape` FAIL (RED, this session) | yes | pass |

**Coverage:** 32/32 scenarios covered by a test or named verification that ran this session; all 32 pass clean (the one named pre-existing exception, W001, was fixed in re-verification — `0b9e65cd`).

## Verification-Gap Audit

| Gap | Location | Shape | Evidence | Smallest Regression |
|-----|----------|-------|----------|---------------------|
| (FOUND + FIXED this session) `TestSerialToolCalls` drove only 2 concurrent calls against the `deep-concurrency-stays-serial` scenario (100+); the 2-call shape could not see the queue-full path | `internal/transport/mcp/server_family_test.go:379` | broken-verification (the tag claimed more than the body exercised) | extending to 128 calls went RED against `WithQueueSize(1)` — mcp-go's dispatch falls back to synchronous processing when the queue is full (`stdio.go:614 default:`), so 3+ concurrent calls ran 2-in-flight | a client sending a burst of 3+ parallel tool calls sees two handlers in flight (completion-order responses), violating R5 |
| (FOUND + FIXED this session, same root) `StartOn` pinned `WithQueueSize(1)` — with 1 worker + 1 queue slot, the 3rd in-flight call is processed synchronously in the dispatch goroutine | `internal/transport/mcp/server.go:565` | regression (production option below the scenario's floor) | RED: maxSeen=2 at 128 calls; mcp-go source `stdio.go:604-621` (blocking send + `default:` synchronous fallback) | any burst beyond 2 in-flight calls loses strict seriality |

Both closed at `10e4206b` (queue → `maxQueueSize` 10000 + 128-call deep phase with exact-ID matching). No other gaps: every production seam this change introduces (dispatch, StartOn, RenderExplore, enriched Search, installer builders) has a named pin that ran this session.

**Gap shapes:** regression gap / missing-adoption gap / broken-verification gap (definitions per the skill).

## TDD Evidence Audit

> `testing.tdd: false` (Standard mode) — strict-TDD RED/GREEN/REFACTOR commit archaeology is not the project floor. Evidence below is the feature-commit → passing-gate chain (git history + current run), per the change's execution record.

| Task | SATISFIES | Feature commit | Gate (ran this session) | Verdict |
|---|---|---|---|---|
| TICKET-01 | rename-builds-and-tests-green-at-new-path, rename-leaves-no-module-references, rename-keeps-v1-untouched | `08f8984c` | full suite green at `mnemonic/` (-race) + grep gate 0 hits + v1 build | OK |
| TICKET-02 | new-family-paths-dispatch, bare-code-defaults-to-index, old-paths-suggest-their-replacement, serve-is-gone | `5b047ddb` | 5 CLI tests green (-count=1) + live binary smoke (4 old paths exit 2 + renamed) | OK |
| TICKET-03 | memory-server-serves-memory-only, all-profile-adds-admin-tools-only, code-and-wiki-servers-are-isolated, unknown-profile-refuses, memory/projectwiki/code-initialize-*, concurrent-calls-answer-one-at-a-time, deep-concurrency-stays-serial, stdout-carries-only-protocol-frames, two-servers-one-store-parallel, mutation-of-serial-option-is-caught | `39b7e2ed` (+ QA fix `10e4206b`) | 6 mcp tests green (-count=1) + live smoke (3 servers + all profile) + mutation sampling RED | OK |
| TICKET-04 | search-hit-carries-blast-radius-and-snippet, unresolved-hit-degrades-gracefully, duplicate-edges-are-reported-once | `59195870` | 3 handler tests green (-count=1) | OK |
| TICKET-05 | explore-returns-the-full-baseline-shape, explore-tags-inferred-edges, explore-on-empty-index-degrades-silently, explore-declares-a-degraded-leg, mutation-of-renderer-is-caught | `541ab49a` | 4 handler tests green (-count=1) + mutation sampling RED | OK |
| TICKET-06 | no-stale-path-references-remain, fresh-install-writes-live-mcp-commands, architecture-page-matches-the-new-tree | `a49f1839` spec / `9c9d2138` code / `b0917d4d` docs | `TestNoStalePathReferences` + setup suite green + `validate ARCHITECTURE.md` ok | OK |
| TICKET-07 | adr-lands-and-indexes | `f0eb2dfc` spec / `e3b3e145` vault | `check-adr-invariants.mjs` OK 58/59 | OK |

**Summary:** 7/7 tickets have a feature commit and a passing gate (re-run fresh this session).

## Assertion Quality Audit

> Re-scanned all test files created/modified by this change: `cli_family_test.go`, `stale_refs_test.go`, `server_family_test.go`, `tools_code_test.go`, `explore_format_test.go`, setup `{hermes,protocol,cursor}_test.go`.

| File | Line | Pattern | Evidence | Severity |
|---|---|---|---|---|
| `server_family_test.go` (pre-fix) | old `waitForID` | impl-detail (substring ID match) | `strings.Contains(string(m.ID), "1")` matched IDs 10, 11, … — a latent wrong-pass at 10+ responses; fixed to exact match in `10e4206b` | SUGGESTION (fixed this session) |

**Notes:** `TestNoStalePathReferences` asserts on repo file text — deliberate: the guarded production artifact IS the repo convention files (boundary test, not a broken verification). No tautologies, ghost loops, or assertions without a production call.

**Summary:** 0 CRITICAL, 0 WARNING, 1 SUGGESTION (fixed this session).

## Changed-File Coverage

> `testing.coverage` = "" and `quality.coverage_min` = 0 → **advisory only, never blocking.** Numbers below are from a 3-package run (`./internal/transport/mcp/ ./internal/transport/handler/ ./cmd/mnemonic/`, -count=1); shared files (`handler.go`, `handler_code.go`) also serve the CLI surface and are under-counted here vs a whole-module run.

| File | Line % | Uncovered lines | Rating |
|---|---|---|---|
| `mnemonic/internal/transport/mcp/server.go` | 90.1% | 18 stmts (teardown edges) | Acceptable |
| `mnemonic/internal/transport/handler/explore_format.go` | 89.4% | 7 stmts (empty-variant branches) | Acceptable |
| `mnemonic/internal/transport/mcp/dispatch.go` | 42.0% | admin-profile dispatch rows | Low |
| `mnemonic/internal/transport/mcp/tools_code.go` | 54.7% | per-tool adapter rows | Low |
| `mnemonic/cmd/mnemonic/main.go` | 61.6% | `main`/`printUsage`/git helpers (wiring) | Low |
| `mnemonic/internal/transport/handler/handler_code.go` | 26.6% | non-changed file regions (changed paths: Search enrichment + Explore — exercised by the T4/T5 pins) | Low (file-level; changed paths covered) |
| `mnemonic/internal/transport/handler/handler.go` | 15.1% | shared file; this change's edits minimal | Low (file-level; changed paths covered) |

**Threshold:** `quality.coverage_min` = 0 → gate N/A. **Average changed-file coverage:** 53.0% (informational).

## Test Quality Audit

| Test | Contract Violated | Evidence |
|------|-------------------|----------|
| — | — | — |

**Notes:** real code (the production stdio loop over pipes, real CLI dispatch, real two-process store); no vacuous truths (both mutation samplings flipped a pin to RED); no pass-always; claimed paths tested; fixtures realistic (the handler pins use fixture stores shaped like the production rows); counter-tests present (`TestSearchUnresolvedHitDegrades`, `TestExploreEmptyIndex`).

**Summary:** 0 violations.

## Test Strategy Audit

**Available layers** (from `testing.layers`): unit, integration.

| Behavior | Assigned Layer | Degrade? | Duplicate Coverage? | Rationale |
|----------|---------------|----------|---------------------|-----------|
| CLI dispatch / refuse / default-verb | unit | no | no | in-process dispatch, no I/O beyond a temp project |
| initialize shape + tools/list isolation | integration (in-process stdio over pipes) | yes (e2e unavailable) | no | the highest available layer that drives the real protocol loop |
| serial stdio + protocol-only stdout | integration (in-process stdio) | yes (e2e unavailable) | no | the loop + options are the seam; a real client adds nothing |
| two-servers-one-store parallelism | integration (two processes, one store file) | no | no | cross-process store behavior needs real processes |
| search enrichment / explore render | unit | no | no | pure/handler-level over fixtures |
| installer command writers | unit | no | no | config-file assertions |

**Layer distribution:** 12 unit, 7 integration, 0 e2e. Reasonable for a transport-contract cutover: the risk concentrates at the protocol loop and the store, both at the highest available layer.

## Security Audit

**Mode:** Trivy

### Trivy Findings (Mode A)

**Command:** `trivy fs . --scanners vuln --severity CRITICAL,HIGH,MEDIUM,LOW` (config: `fail_on: ""` → report only)

| Scanner | Finding | Severity | Location | Fix | Classification |
|---------|---------|----------|----------|-----|----------------|
| vuln | golang.org/x/text CVE-2026-56851 (DoS) | HIGH | `.worktrees/mnemonic-llm-provider/skillgrid-cli/go.mod` (installed v0.39.0) | v0.41.0 | SUGGESTION — a separate in-flight change's worktree; main-tree Go modules: 0 findings |
| vuln | DOMPurify GHSA-cg2g-xh64-v4q7 + 4 HIGH + 4 LOW + 3 MEDIUM (SPA) | CRITICAL×1 | `.worktrees/mnemonic-llm-provider/skillgrid-ui/package-lock.json` | 3.4.17 | SUGGESTION — same worktree; the v1 SPA is legacy surface (ADR-0059) |

**Main tree (this change's footprint):** 0 findings at CRITICAL/HIGH (root `package-lock.json`: 0; all Go modules: 0). The change adds no dependencies.

**Trivy gate:** `fail_on: ""` → N/A (advisory, never blocks).

### Manual Findings (Mode B)

No user-facing input handling changed in this change beyond what the stdio protocol already covers (the MCP-spawn trust boundary, ADR-0005, is unchanged: spawned process, project read access). No new secrets, no new config surface with credentials.

**Security verdict:** PASS for the change's footprint — 0 findings in the main tree at any severity above LOW; all findings are inside `.worktrees/mnemonic-llm-provider` (a separate in-flight change).

## Code Quality Gate

| Gate | Command | Threshold (config) | Actual | Result |
|------|---------|--------------------|--------|--------|
| Coverage (whole project) | not configured | 0 | — | N/A |
| Changed-file coverage | not configured (informational run) | 0 | 53.0% avg (see table) | N/A |
| Mutation | not configured (L4 floor via live sampling) | 0 | 2/2 mutations caught (RED) | N/A (floor met) |
| Lint | not configured (go vet via `go test`) | errors only | 0 errors | PASS |
| Typecheck | not configured (Go compilation) | errors only | 0 errors | PASS |
| P0 pass rate | plan rows 1,2,3,4,5,6,7,16 | 100 | 100 (row 7: both modules green — the 4 v1 `internal/install` failures were fixed in re-verification, `0b9e65cd`) | PASS |
| P1 pass rate | plan rows 8–17 | 95 | 100 | PASS |
| Trivy security | `trivy fs . --scanners vuln …` | report only | 0 in main tree | N/A |
| Duplication (jscpd) | `npx jscpd` (changed files) | report only | 72 clones · 2.69% lines (pre-existing, see below) | N/A |

**Quality config status:** configured (`quality:` present).

## Duplication

> Scoped to the changed files (`git diff --name-only 09b99d9a..HEAD -- '*.go'` — 195 files; the module rename moved the whole existing tree, so the clone base is the pre-existing code).

**Command:** `npx jscpd --min-tokens 70 --ignore '**/*_test.go' <changed go files>`

**Result:** 72 clones · 928 lines (2.69%) / 7441 tokens (3.75%) over 103 analyzed files — scope: COMPLETE.

| Files (lines) | Kind | Introduced? | Severity | Note |
|---------------|------|-------------|----------|------|
| `handler_code.go:402-403` region + siblings (neighbor-view error paths) | near | pre-existing | SUGGESTION | moved by the rename, not written by this change |

**Advisory:** no clone introduced by this change — its net-new code is the thin adapter seams (`StartOn`, family registration, `RenderExplore`, the enriched `Search` path), none of which appears in the clone list.

## State Drift

> `node .agents/skills/qa/scripts/state-drift-check.mjs` — read-only; never CRITICAL, never affects the verdict.

**Verdict:** PARSE ERROR

| Field | Stale (state.yaml) | Derived (spec zone) |
|-------|-------------------|---------------------|
| `pipeline.current_phase` | `slicing` | `qa` |

**Fix applied:** `pipeline.current_phase` set to `qa` in this gate's state commit.
**Parse error (pre-existing):** the multi-segment entry format (`key: "text" (annotation) "text2" …`) is not strict YAML — the guard's `yaml` package fails identically on the pre-change (`88ffc471`) and current content (both: "Nested mappings are not allowed in compact mappings at line 9, column 26"). The guard is dead against the current format — W002.

**Scope (from the guard's `SCOPE:` line):** n/a (parse error precedes derivation).

## Verification Scope

| Derivation | Scope | Stale? |
|---|---|---|
| Traceability matrix | COMPLETE (32/32 scenarios; `acceptance.feature` read in full) | none |
| Verification-gap audit | COMPLETE over the change's diff (195 files: the bulk is the content-identical module rename, verified by build+suite+grep rather than per-file review; ~25 substantive files reviewed at the seam level: dispatch, StartOn, RenderExplore, Search, installer builders) | none |
| State Drift (9.5) | PARSE ERROR (pre-existing) | n/a |
| Structure Drift (9.6) | n/a (guard parse error) | n/a |

**Stale-verification:** `STALE: none` — the only code-zone change since the last full verification is `10e4206b` (the queue fix), and the full suite (-race), the named tests, the mutation sampling, and the live smoke all ran after it.

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward verification (weakest truth) | VERIFIED (7/7 truths) |
| Traceability (weakest scenario) | VERIFIED (`full-suite-green-race` now clean after W001 fix) |
| Verification-gap audit | SUGGESTION (2 gaps found + fixed this session; 0 open) |
| TDD evidence (weakest ticket) | OK (7/7) |
| Assertion quality audit (weakest finding) | SUGGESTION (1 fixed this session; 0 open) |
| Changed-file coverage (weakest file) | N/A (threshold 0; 15.1% file-level on the shared `handler.go`, changed paths covered) |
| Test quality audit | none |
| Code-quality gates (weakest gate) | PASS (all PASS or N/A) |
| Security audit | SUGGESTION (worktree findings only; main tree clean) |
| Verification scope (composite) | COMPLETE (state-drift row: PARSE ERROR, pre-existing, no verdict effect) |

**FLOOR:** all floors at or above their thresholds (weakest: Traceability — VERIFIED after the W001 fix). No floor caps the gate.

## Findings

### CRITICAL (must fix before merge)

- None.

### WARNING (should fix before archive)

- None open. **W001 — 4 pre-existing v1 suite failures** in `skillgrid-cli/internal/install` (`TestMemoryAndPersonasBoundary`, `TestCollectNPMPackages`, `TestInstallNPMBatchAllowScripts`, `TestPrePushBlocksMalformedCloses`) — **FIXED in re-verification (`0b9e65cd`)** after the user chose fix-now. Verified pre-existing at the change's baseline (`f30aea50`, worktree run); fixed with three root causes: (a) `TestMemoryAndPersonasBoundary` — the V2-logo refactor dropped `ensureConfigFile(tui.json)` from the opencode path, a real source regression: fresh-home installs crash at `installAgentConfig` (restored in `agents.go`, matching the kilo path); (b) `TestCollectNPMPackages` / `TestInstallNPMBatchAllowScripts` — test expectations stale on `opencode-ai` where the source correctly uses `@opencode/cli` (package verified on npm, v2.2.6); (c) `TestPrePushBlocksMalformedCloses` — the developer's global `core.hooksPath` (`~/.skillgrid/git-hooks`) leaked into the throwaway git repos and its pre-commit footer check killed the test's own commit; the tests now run git with `GIT_CONFIG_GLOBAL/SYSTEM=/dev/null` (hermetic, via the shared `gitIn` helper + both repo-setup loops). Also fixed one pre-existing `gofmt` drift in `agents.go` (const-block comment alignment). Full `skillgrid-cli` suite `-race` green after the fix.

### SUGGESTION (nice to have)

- **W002 — state-drift-check.mjs cannot parse the multi-segment state.yaml format** (pre-existing; both pre- and post-change content fail identically). Either make the state format strict YAML or make the guard lenient; until then the drift guard is dead.
- **Deleted draft spec folders (disposition record):** three v1-era draft specs (`2026-10-05-mnemonic-central-pg`, `2026-10-06-mnemonic-standalone`, `2026-10-07-mnemonic-compaction-v2` — all `status: draft`, never executed) were deleted outright in the D1 rename commit (`08f8984c`) without a recorded disposition (code-review finding I2). Restored to `change-specs/` at review time; they are superseded by the v2 repo-map redesign (2026-11-01) and are kept as historical drafts.
- **Trivy (worktree):** `.worktrees/mnemonic-llm-provider` carries golang.org/x/text CVE-2026-56851 (HIGH, fix v0.41.0) + 12 SPA findings (1 CRITICAL DOMPurify) — fix on that branch, not this one.
- **7 advisory STATUS-banner findings** (pre-existing: 6 old briefings + the current change's banner, now `draft` vs derived phase `qa`).

## Gate Decision

**Verdict:** PASS

**Reasoning:** Every one of the 9 requirements' scenarios is VERIFIED by named tests that ran fresh this session, a live stdio smoke across all three family servers, and 2/2 live mutation samplings — and the QA pass itself caught and fixed a real protocol defect (mcp-go's queue-full synchronous fallback; `10e4206b`). The one named exception from the first render (W001: 4 pre-existing v1 suite failures) was fixed in re-verification (`0b9e65cd`) at the user's choice — full `skillgrid-cli` suite `-race` green. Remaining findings are SUGGESTION-level only (W002 tooling gap, worktree-scoped Trivy findings, pre-existing status-banner drift) — none blocks PASS. No CRITICAL findings.

**Open items (if CONCERNS):** none — gate PASS. W002 (SUGGESTION) remains for a future tooling change.

## Human Override

> A human decision always overrides this machine verdict.
> An epic that fails its criteria with no human decision is recorded as **not accepted** — never as silently accepted.

**Human decision:** not applicable — gate PASS (no override needed). Re-rendered after the user chose fix-now on W001.

<!-- reflect completes this half at archive time -->

## Final-State Facts

**Shipped:** <what actually shipped>
**Base branch:** <base-branch> · **Chain strategy:** <strategy>
**Integration:** <merged @ <commit> | PR <url> | kept branch <name>> (from ship context)

## Gates

| Gate | Result |
|------|--------|
| Ship gate | <pending — ship> |
| QA gate | ✅ PASS (re-rendered after W001 fix `0b9e65cd`) |
| Verdict gate (advisory) | <pending — human> |

## Decisions

| Decision | Tradeoff | Why | Source |
|----------|----------|-----|--------|
| <reflect> | | | |

## Lessons

| Lesson | Root Cause | Do Differently | Source |
|--------|-----------|----------------|--------|
| <reflect> | | | |

## Patterns

| Pattern | Reuse | Source |
|---------|-------|--------|
| <reflect> | | |

## Surprises

| Surprise | Signal | Source |
|----------|--------|--------|
| mcp-go's queue-full fallback processes the call synchronously (2-in-flight at 3+ concurrent) | TestSerialToolCalls deep phase RED (maxSeen=2) on `WithQueueSize(1)` | `mcp-go@v0.44.0/server/stdio.go:604-621` / commit 10e4206b / scenario deep-concurrency-stays-serial |

## Environment Retro

**Written by `skillgrid:environment-retro` (a reflect sub-phase), not qa.** Improvements to *the agent environment*, never the code.

| Category | Finding | Fix lands in | Source |
|----------|---------|--------------|--------|

## Acceptance Verdict

**Verdict:** <accepted | accepted-with-open-items | rejected>

**Grounding:** <goal from briefing.md + the Goal-Backward Verification / Traceability Matrix evidence above that supports the verdict>

**Reasoning:** <2–3 sentences: what drove the verdict>

## Open Items (→ next change)

- W001: 4 pre-existing v1 suite failures (skillgrid-cli/internal/install) — **closed in QA re-verification (`0b9e65cd`)**; one source regression (opencode `tui.json` ensure) fixed in `agents.go`.
- W002: state-drift-check.mjs vs the multi-segment state.yaml format — tooling fix or format decision.

## Web-cache promotion candidates

<!-- reflect: proposals only. Replace with one line if none. -->

## Prior-Change Follow-Through

| Prior open item (from previous archived change) | Addressed by this change? | Evidence |
|--------------------------------------------------|---------------------------|----------|
| <reflect> | | |

## Move Evidence (from ship context)

**From:** `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/` → **To:** `.skillgrid/change-archive/2026-11-08-mnemonic-mcp-split/`
**`diff -r` readback:** <pending — ship>

## Overrides / Waivers / Contradictions

- <pending — human override if any>

## Lineage (observation IDs)

- briefing: <reflect>
- blueprint: <reflect>
- tasks: <reflect>
- ship: <reflect>
- report (QA half): this file
- research / findings / ADRs: ADR-0061 (04-adr-0061-mcp-family-split.md)
