# Report — mnemonic MCP family split

> Change: `.skillgrid/change-specs/2026-11-08-mnemonic-mcp-split/` (moves to `.skillgrid/change-archive/2026-11-08-mnemonic-mcp-split/` at ship)
> Generated: 2026-11-08T18:45:00+08:00 (qa)
> Gate: PENDING (test plan committed; gate renders after the audit)
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
| 1 | Old CLI paths dispatch or alias silently / new paths misroute | CLI dispatch (`cmd/mnemonic`) | unit | `cli_family_test.go::TestFamilyDispatch`, `TestOldPathsSuggestTheirReplacement`, `TestOldSubcommandPathsRefuse`, `TestBareCodeDefaultsToIndex` | P0 | pr | pending |
| 2 | Rename leaves `mnemonic-v2` module references (build drift) | go.mod / go.work / Taskfile | build + grep gate | workspace build + full suite at `mnemonic/` (row 15) + grep of go.mod/go.work/Taskfile (live) | P0 | pr | pending |
| 3 | Tool isolation leak (a client sees another family's tools) | MCP `tools/list` per family × profile | integration (in-process stdio) | `server_family_test.go::TestToolIsolationPerFamily`, `contract_test.go::TestAllProfileIsAgentPlusAdmin` | P0 | pr | pending |
| 4 | Serial stdio broken (concurrent processing / interleaved responses) | stdio loop (`StartOn`) | integration (in-process stdio) | `server_family_test.go::TestSerialToolCalls` (128-call deep phase; the WithWorkerPoolSize mutation gate) | P0 | pr | pending |
| 5 | Non-protocol bytes reach stdout post-startup | stdio loop + `Logf` stderr seam | integration (in-process stdio) | `server_family_test.go::TestStdoutProtocolOnly` | P0 | pr | pending |
| 6 | Parallel family servers lock/double-push the store | SQLite store + sync lease across processes | integration (two servers, one store) | `server_family_test.go::TestTwoServersOneStoreParallel` | P0 | pr | pending |
| 7 | Full-suite race regression (both modules) | workspace | integration | `go test ./... -race` (mnemonic + skillgrid-cli) | P0 | pr | pending |
| 8 | initialize shape drift vs the baselines (identity / capabilities / instructions) | MCP `initialize` per family | integration (in-process stdio) | `server_family_test.go::TestInitializeShapePerFamily` (byte-identical pins vs the engram/openwiki baselines) | P1 | pr | pending |
| 9 | Unknown profile accepted | CLI mcp-verb flag parse | unit | `cli_family_test.go::TestUnknownProfileRefuses` | P1 | pr | pending |
| 10 | `code_search` hit loses blast radius / snippet (quality floor) | `CodeHandler.Search` response | unit | handler `TestSearchHitBlastRadiusShape`, `TestSearchUnresolvedHitDegrades`, `TestSearchDuplicateEdgesOnce` | P1 | pr | pending |
| 11 | `code_explore` loses a baseline element / lists a duplicate edge | `RenderExplore` | unit | handler `TestExploreResponseShape` (the mutation gate), `TestExploreTagsInferredEdges`, `TestExploreEmptyIndex`, `TestExploreDegradedLeg` | P1 | pr | pending |
| 12 | Stale old-path references survive in live skills/docs/vault | repo convention files | unit (grep boundary test) | `cmd/mnemonic::TestNoStalePathReferences` | P1 | pr | pending |
| 13 | Fresh install writes the dead bare `mnemonic mcp` form | installer config writers | unit | `skillgrid-cli/internal/setup` hermes/protocol/cursor tests (family command-line assertions) | P1 | pr | pending |
| 14 | Architecture page anchors stale vs the new tree | projectwiki pin-check | manual gate (CLI) | `mnemonic projectwiki validate ARCHITECTURE.md` (same-edit reindex) | P1 | pr | pending |
| 15 | ADR missing / mis-indexed / number collision | ADR invariants | unit (node script) | `node scripts/check-adr-invariants.mjs` | P1 | pr | pending |
| 16 | mcp-go pool option mutates (serial invariant unguarded) | `StartOn` pinned options | mutation sampling | live mutation: drop `WithWorkerPoolSize(1)` → `TestSerialToolCalls` must fail (row 4) | P0 | pr | pending |
| 17 | explore renderer mutates (shape unguarded) | `RenderExplore` | mutation sampling | live mutation: drop the blast-radius section → `TestExploreResponseShape` must fail (row 11) | P1 | pr | pending |

**Layer** — unit where the seam is in-process (CLI dispatch, renderer, handler); integration for the stdio-loop and cross-process behaviors (the layer list is `[unit, integration]`; the e2e layer is unavailable, and the in-process stdio loop over pipes is the highest available layer for the protocol invariants). Duplicate Coverage Guard: the CLI refuse cases (row 1) are not re-covered at the stdio layer; the isolation counts (row 3) are not re-covered by the initialize pins (row 8).

**Cadence** — all `pr`: every row is a merge blocker for a public-contract cutover; none is a regression net only.

### Edge-Case Matrix

| Requirement | Edge / Boundary | Expected Behavior | Test / Scenario | Status |
|-------------|-----------------|-------------------|-----------------|--------|
| CLI cutover | removed paths (`mnemonic index` / `mem save` / `mcp` / `serve`) | non-zero exit + did-you-mean naming the new path | `TestOldPathsSuggestTheirReplacement`, `TestOldSubcommandPathsRefuse` | pending |
| CLI cutover | bare `mnemonic code` (no sub-verb) | runs the index (default verb unchanged) | `TestBareCodeDefaultsToIndex` | pending |
| CLI cutover | unknown `--profile` on a family mcp verb | non-zero usage refusal naming the valid profiles | `TestUnknownProfileRefuses` | pending |
| serial stdio | 128 tool calls in flight before any response (beyond the queue) | max in-flight = 1 | `TestSerialToolCalls` deep phase | pending |
| code_search | hit whose symbol did not resolve | blast-radius fields omitted, no error | `TestSearchUnresolvedHitDegrades` | pending |
| code_search | same caller edge recorded twice | caller counted exactly once | `TestSearchDuplicateEdgesOnce` | pending |
| code_explore | empty index / non-git directory | `Found 0 symbols across 0 files.`, no source blocks, no degraded flag, no error | `TestExploreEmptyIndex` | pending |
| code_explore | one search leg degraded | degraded flag naming the leg | `TestExploreDegradedLeg` | pending |
| code_explore | INFERRED call edge | tagged `(inferred)`; EXTRACTED edges untagged | `TestExploreTagsInferredEdges` | pending |
| store | two family servers on one store, autosync on | ≤1 sync-lease holder; no `database is locked` | `TestTwoServersOneStoreParallel` | pending |
| store | fresh-store TOCTOU (two processes race the lease) | single winner, sequenced like `TestAutosyncSingleLease` | `TestTwoServersOneStoreParallel` (sequenced phase) | pending |

### Out of Scope

- **v1 `skillgrid-cli` behavior** beyond the installer's written command strings — v1 is untouched by design (R1).
- **HTTP transport** — v2's `serve` is removed; no HTTP surface exists to test (out of scope per briefing).
- **Store/index semantics** — unchanged by design; R6 is response-shape only over the existing graph.
- **MCP client behavior** — mcp-go internals beyond the pinned options (worker pool / queue size) are the library's contract, tested only at the seam.
- **Embedding provider availability** — the vector leg's degradation path is exercised (`TestExploreDegradedLeg`); a live Ollama endpoint is deployment state, not this change.
