# SDD ledger — plan: docs/skillgrid/changes/007-mnemonic-project-handle-facade/tasks.md

## Pre-flight plan scan

| Pair / item | Finding | Ruling |
|-------------|---------|--------|
| 01 vs 02/03 (interface) | 01 produces exported `ProjectHandle` + `Open`/`OpenForCWD`/`OpenForDirectory`; 02/03 consume it. No conflict. | proceed |
| 01 vs 04 (service.go) | 01 renames `projectHandle`→`ProjectHandle` + adds guard; 04 collapses wrappers. 04 builds on 01. | proceed |
| 02 vs 03 (parallel after 01) | Touch disjoint files (mcp/* vs http/*). Safe to run sequentially; both depend only on 01. | proceed |
| 01 text self-consistency | tasks list handle.go as "optional split"; impl kept in service.go. Brief permits. | OK |
| 04 depends on 02+03 | Correct DAG. | OK |

## Tasks

- Task 01: complete (commits 51bcfa4..e3b67d1, review clean after 1 fix round)
  - Implementer: 99b97ad (abort on bad id) + 32dc818 (export seam)
  - Review round 1: approved, 1 Important (webcache never exercised) + 3 Minor
  - Fix round 1/5: e3b67d1 (seed webcache snapshot → Lookup hit; reword Open doc) — re-review: both ADDRESSED, no new breakage
  - Tests: service package 34/34 PASS; `go build ./...` OK
  - Minor (deferred): test names lack `AbortOpen` substring (matches via Empty/InvalidProject substring — brief command still works); teams.go openTeamsHandle("") whitespace pre-existing

- Task 02: complete (commits 8d34aa0..4f7e9bf, review clean — 1 plan-mandated Important parked)
  - Implementer: 4f7e9bf (MCP single-open via Project Handle)
  - Review round 1: approved, 1 Important (verbatim copy duplication) + 3 Minor
  - Ruling: the mcp↔service verbatim copies (mcpOrientSymbol, mcpCodeExplore, mcpMemoryDoctor, mcpCodeEmbeddingStatus, mcpReadIndexedCode) are plan-mandated — the handle only exposes Memory()/Web()/Store(), so MCP has no non-reopening seam to call. Correct for step 02. Consolidation is step 04's job (collapse wrappers: export db-backed impls from service/ or move to handle, delete mcp copies, add identity test). PARKED → carry into step 04 dispatch.
  - Minor (deferred): RecordOpen counts store.Open calls not handles (drift probes can add opens to other stores — target-project invariant still holds); openServiceForRepo double-resolves (pre-existing, no regression); process-global counter shared across pkgs (test-only, ResetOpenCount before assert).
  - Tests: mcp full suite PASS, store PASS, go build ./... OK, go vet clean. RED→GREEN single-open counter 2→1.
  - Env fix: TestMemSaveRemainsRegisteredAdditive pinned MNEMONIC_PROJECT=reg (pre-existing fragility, not a regression).

- Task 03: complete (commits 90df9e2..4a08eff, review clean after 1 fix round)
  - Implementer: 153c4ce (HTTP single-project routes via Project Handle)
  - Review round 1: approved, 1 Important (httpResolveEmbedder double-loads config) + 3 Minor
  - Ruling/discovery: HTTP path NEVER had the MCP-style double-open — `projectFromRequest` resolves the id with no store.Open, and each handler called exactly one facade (one open) → 0+1=1. The brief's predicted RED FAIL(2) was MCP-shaped and doesn't apply to HTTP. The rewire is still correct (MCP-consistent) and the single-open invariant is now pinned by tests (a future regression to 2 opens fails). This is a brief/prediction discrepancy, not an implementation defect.
  - Fix round 1/5: 4a08eff (httpResolveEmbedder takes cfg config.Indexing; handleCodeIndex loads config once) — re-review: ADDRESSED, no new breakage
  - Seed-merge claim verified: handleCodeIndex → openHandleForDir → OpenForDirectory → openProjectForDirectory performs the MergeProjects fold before the single open; only theoretical non-SourceIdentity branch differs (hard-abort for writes anyway).
  - Tests: http full suite 53 PASS, service/mcp/store PASS, go build ./... OK, go vet clean.
  - Verbatim copies (httpMemoryDoctor, readIndexedCode, httpResolveEmbedder, scopeForSave, CodeFiles SQL) verified byte-identical to service/ sources.

- Task 04: complete (commits 13e38bc..89cd2ca, review clean — 1 pre-existing Important parked)
  - Implementer: 89cd2ca (collapse double-open Service wrappers). NOTE: prior subagent died mid-work on API error, leaving uncommitted partial work (service.go -560 lines, double_open_test.go); resumed the same subagent task_id to finish.
  - Deleted 43 dead single-project wrappers + 4 orphaned type aliases; restored PassiveInput alias (http/server.go:417 still refs it).
  - Rewired tests: service/orient_test.go, service/compaction_test.go, integration/seed_test.go, mcp/single_open_test.go (scope normalization ""→project, personal→user preserved in saveObservation/seedObservation helpers).
  - Review round 1: approved, 1 Important (pre-existing CodeExplore→CodeImpact double-open) + 2 Minor
  - Ruling: CodeExplore (service.go:986) opens the project then calls s.CodeImpact (service.go:1044→:814) which reopens — a PRE-EXISTING double-open, not introduced by this step (both methods KEPT, neither added/removed). The "no production path double-opens" constraint is about the open-delegate-close FACADE wrappers this change collapses (now gone). Fixing CodeExplore to reuse an already-open handle (codeImpactHandle variant) is a separate deepening — PARKED as a follow-up, not a blocker for 007. The guard (double_open_test.go) pins the Open/MnemonicCommit invariants.
  - Minor (deferred): SemanticSearch/LoadFullDetails/SearchObservationsScoped report overstates "production callers" (actually test-only; correctly kept per cross-store caveat); CodeFiles rewire inlines unfiltered query (faithful to original).
  - memory/*_test.go needed NO rewiring (already on *memory.Service directly).
  - Tests: go build ./... OK; service/mcp/http/integration PASS; go vet clean; FULL SUITE 19/19 packages PASS. Double-open guard (DoubleOpen|OpenCounter|Once) PASS.
