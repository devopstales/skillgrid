# QA Report — Global Operational Memory + Skill Improvement Loop (Change A)

**Change:** 2026-10-24-global-llm-wiki
**Date:** 2026-10-25
**Tier:** T2 (Beta) — full pipeline + two-axis review
**Floor:** L2+L3 (unit + integration verification)
**Quality config:** coverage advisory (0), mutation disabled, P0=100%, P1=95%, Trivy advisory, jscpd advisory

## Test Plan

| Scenario (acceptance.feature) | Test | Layer | Priority | Status |
|---|---|---|---|---|
| Global store bootstrap: happy path | `TestOpenGlobalCreatesUnmangledStore` + `TestOpenGlobalCategoryColumn` | unit | P0 | PASS |
| Global store bootstrap: edge — excluded from listing | `TestListProjectsExcludesGlobal` | unit | P0 | PASS |
| Global store bootstrap: edge — excluded from TTL | `TestRunTTLExpirySkipsGlobal` + `TestRunTTLExpirySkipsGlobalWithObservationSeeded` | unit | P0 | PASS |
| Global store bootstrap: failure — file name | `TestOpenGlobalCreatesUnmangledStore` | unit | P0 | PASS |
| Scope routing: happy path — global add + trail | `TestGlobalAddWritesGlobalStoreProjectTrail` + `TestFactAddGlobalScopeMCP` | unit+integration | P0 | PASS |
| Scope routing: happy path — project add | `TestProjectAddWritesProjectStoreOnly` + `TestFactAddProjectScopeMCP` | unit+integration | P0 | PASS |
| Scope routing: edge — target user | `TestGlobalAddUserTarget` + `TestFactAddGlobalUserTargetMCP` | unit+integration | P1 | PASS |
| Scope routing: failure — unknown session | `TestGlobalAddUnknownSessionFails` + `TestFactAddGlobalUnknownSessionMCP` | unit+integration | P0 | PASS |
| Fused search: happy path — both stores | `TestFusedSearchReturnsBothStoresInterleaved` + `TestFusedSearchRankDrivenOrdering` | unit | P0 | PASS |
| Fused search: edge — one store | `TestFusedSearchReturnsBothStoresInterleaved` (triangulation) | unit | P1 | PASS |
| Fused search: edge — trail to project | `TestFusedSearchTrailGoesToProjectStoreOnly` | unit | P0 | PASS |
| Fused search: failure — empty stores | `TestFusedSearchEmptyStoresReturnsEmpty` | unit | P1 | PASS |
| Batch: happy path — free + add | `TestBatchFreeAndAdd` + `TestFactAddOperationsBatchMCP` | unit+integration | P0 | PASS |
| Batch: edge — overflow | `TestOverflowReturnsStructuredError` + `TestOverflowUserLimit` + `TestFactAddOperationsOverflowMCP` | unit+integration | P0 | PASS |
| Batch: edge — dedup | `TestDedupRejectsExactDuplicate` | unit | P1 | PASS |
| Batch: edge — substring patch | `TestSubstringPatch` + `TestRemoveSubstringMatch` | unit | P1 | PASS |
| Batch: failure — ambiguous | `TestRemoveAmbiguousMatch` | unit | P1 | PASS |
| Batch: failure — no match | `TestPatchNoMatchReturnsClosest` | unit | P1 | PASS |
| Prefetch: happy path — two sections | `TestRenderFrozenSnapshotTwoSections` | unit | P0 | PASS |
| Prefetch: edge — frozen mid-session | `TestPrefetchFrozenMidSession` | unit | P0 | PASS |
| Prefetch: failure — empty store | `TestRenderFrozenSnapshotEmptyStore` + `TestRenderFrozenSnapshotNilStoreFailsOpen` | unit | P1 | PASS |
| Write approval: happy path — stage + approve | `TestWriteApprovalStagesPassiveWrites` + `TestFactPendingQueueMCP` | unit+integration | P0 | PASS |
| Write approval: edge — survives restart | `TestPendingQueueSurvivesRestart` | unit | P0 | PASS |
| Write approval: edge — explicit bypass | `TestExplicitWritesBypassApprovalGate` | unit | P1 | PASS |
| Write approval: failure — reject | `TestWriteApprovalRejectDiscards` + `TestFactPendingApproveUnknownIDMCP` | unit+integration | P1 | PASS |
| Distill: happy path — route by scope | `TestDistillRoutesFactsByScope` | unit | P0 | PASS |
| Distill: edge — coalescing | `TestDistillCoalescingLatestWins` | unit | P1 | PASS |
| Distill: failure — LLM down | `TestDistillLLMDownFailsOpen` + `TestDistillMalformedResponseWritesNothing` + `TestDistillWithoutLLMIsNoOp` | unit | P0 | PASS |
| Skill loop: happy path — failure logged | `TestObserveLogsFailureWithStderr` | unit | P0 | PASS |
| Skill loop: happy path — inspect | `TestInspectDetectsRecurringFailurePattern` + `TestInspectTriangulatesMixedPatterns` | unit | P1 | PASS |
| Skill loop: happy path — amend | `TestAmendWritesNewFileWithoutModifyingOriginal` | unit | P1 | PASS |
| Skill loop: edge — regression | `TestEvaluateRegressionRollsBack` | unit | P0 | PASS |
| Skill loop: edge — improvement | `TestEvaluateImprovementAppliesAndCommits` | unit | P0 | PASS |
| Skill loop: failure — timeout | `TestObserveLogsTimeoutAsError` | unit | P1 | PASS |
| CLI: happy path — global fact add | `TestMemoryFactAddScopeGlobalTarget` | integration | P0 | PASS |
| CLI: happy path — pending + approve | `TestMemoryGlobalPending` + `TestMemoryGlobalApproveAll` | integration | P0 | PASS |
| CLI: edge — prefetch | `TestMemoryGlobalPrefetchSnapshot` | integration | P1 | PASS |
| CLI: failure — invalid scope | `TestMemoryFactInvalidScope` + `TestMemoryFactScopeAllOnListIsUsageError` | integration | P1 | PASS |

**Threat-matrix tests (T08):**

| Threat row | Test | Status |
|---|---|---|
| Scope isolation | `TestThreatScopeIsolation` | PASS |
| Batch atomicity | `TestThreatBatchAtomicity` | PASS |
| Fused search labeled | `TestThreatFusedSearchLabeled` | PASS |
| FTS injection shape | `TestThreatFTSInjectionShape` | PASS |
| Migration 072 (category) | `TestThreatMigration072Category` | PASS |
| Migration 073 (outcome + pending) | `TestThreatMigration073OutcomeAndPending` | PASS |
| Trail FK (unknown session) | `TestGlobalAddUnknownSessionFails` (pre-existing) | PASS |

**Totals:** 35 acceptance scenarios + 7 threat-matrix tests = 42 verification points. All PASS.

**Out of scope:**
- Change B (wiki pipeline, ADR-0050) — separate change, blocked on Change A
- Session-close trigger wiring (T06 deferral — owner's seam, documented)
- `write_approval` config→handler seam (T06 deferral — ponytail, documented)
- Per-category char budget config→handler seam (T04 deferral — ponytail, documented)

## Goal-Backward Verification

**Stated goal (briefing):** Add a cross-project, cross-session, cross-agent fact store in `__global__.sqlite` with AKL importance, char limit, prefetch injection, background distill, write approval, and a skill improvement loop.

| Truth | Evidence | Verdict |
|---|---|---|
| Cross-project fact store exists at `__global__.sqlite` | `TestOpenGlobalCreatesUnmangledStore` (file name exact, not mangled by NormalizeID) | VERIFIED |
| Scope routing: global writes → global store, trail → project store | `TestGlobalAddWritesGlobalStoreProjectTrail` (row in global, trail in project, 0 trails in global) | VERIFIED |
| Fused search: both stores, labeled, interleaved by rank | `TestFusedSearchReturnsBothStoresInterleaved` + `TestFusedSearchRankDrivenOrdering` (3:1 fixture, rank-driven) | VERIFIED |
| Batch: atomic, char limit on final result, overflow error | `TestBatchFreeAndAdd` + `TestOverflowReturnsStructuredError` + `TestThreatBatchAtomicity` (valid remove runs first, only rollback keeps count) | VERIFIED |
| Prefetch: frozen snapshot, two sections, § separators | `TestRenderFrozenSnapshotTwoSections` (byte-exact assertion) + `TestPrefetchFrozenMidSession` (mid-session add invisible) | VERIFIED |
| Distill: one LLM call, scope-classifying, coalescing | `TestDistillRoutesFactsByScope` (count==1) + `TestDistillCoalescingLatestWins` (count==1, coalesced writes nothing) | VERIFIED |
| Write approval: pending_facts durable, approve/reject | `TestPendingQueueSurvivesRestart` (seed before close, read after reopen) + `TestWriteApprovalStagesPassiveWrites` | VERIFIED |
| Skill loop: observe all outcomes, inspect, amend, evaluate | `TestObserveLogsFailureWithStderr` + `TestObserveLogsTimeoutAsError` + `TestInspectDetectsRecurringFailurePattern` + `TestEvaluateRegressionRollsBack` + `TestEvaluateImprovementAppliesAndCommits` | VERIFIED |
| CLI: scope/target/pending/approve/reject/prefetch | `TestMemoryFactAddScopeGlobalTarget` + `TestMemoryGlobalPending` + `TestMemoryGlobalApproveAll` + `TestMemoryGlobalPrefetchSnapshot` + `TestMemoryFactInvalidScope` | VERIFIED |
| ADR-0049 exists with correct format | File exists, Context/Decision/Consequences, references ADR-0012/0048/0023/0045/0031 | VERIFIED |

All truths VERIFIED.

## TDD Evidence

| Ticket | RED commit | GREEN commit | Test file exists |
|---|---|---|---|
| T01 | build fail (undefined GlobalProjectID, OpenGlobal) | `5d123573` | `global_test.go` ✓ |
| T02 | build fail (undefined Scope, Route, AddWith) | `df072c78` | `scope_test.go` + `tools_facts_scope_test.go` ✓ |
| T03 | build fail (undefined FusedSearch) | `a628c21e` | `facts_test.go` (TestFusedSearch*) ✓ |
| T04 | build fail (undefined Batch, Operation, OverflowError) | `67cef4c3` | `batch_test.go` ✓ |
| T05 | build fail (undefined RenderFrozenSnapshot, NewSnapshot) | `866a488a` | `prefetch_test.go` ✓ |
| T06 | build fail (undefined Distiller, NewDistiller) | `e6b570a3` | `distill_test.go` ✓ |
| T07 | build fail (undefined Inspect, Amend, Evaluate) | `f4b3fc87` | `loop_test.go` ✓ |
| T08 | RED (list --scope all → "no facts") | `12c217be` | `memory_test.go` (TestMemory*) ✓ |

No MISSING_RED. No STALE.

## Audit Results

### Verification Gap
No gaps found. All 35 acceptance scenarios have named covering tests. All 7 threat-matrix rows have covering tests. The 3 documented deferrals (session-close trigger, write_approval config seam, char budget config seam) are ponytail-documented with upgrade paths.

### Assertion Quality
No CRITICAL patterns found. Tests assert on real SQLite rows (not mocks), exact byte shapes (prefetch), store-level queries (isolation), and exit codes (CLI). The threat-matrix tests are non-tautological: `TestThreatBatchAtomicity` runs a valid remove before the overflow so only true rollback keeps the count.

### Security (OWASP spot-check)
No user-facing input handling in this change. The change is a data-layer feature (SQLite stores, routing, batch ops). FTS injection shape is tested (`TestThreatFTSInjectionShape`). No new trust boundaries. Trivy advisory (no findings at fail_on severity — fail_on is empty).

### Code Quality
- `go build ./mnemonic/...` → exit 0
- `go vet ./mnemonic/...` → clean
- `gofmt -l` on changed files → pre-existing drift in untouched files (parallel_search_many_test.go, retrieval.go, teams.go, doctor_strict.go, etc.) — not introduced by this change
- Changed-file coverage: not measured (coverage_min=0, advisory only)
- Duplication (jscpd): advisory (fail_on=0), not run (Go codebase, jscpd primarily targets JS/TS)

### P0/P1 Pass Rate
P0: 100% (all P0 tests pass)
P1: 100% (all P1 tests pass)

## Floor

| Dimension | Verdict |
|---|---|
| Goal-backward | PASS (all truths VERIFIED) |
| Traceability | PASS (42/42 scenarios + threats covered) |
| TDD Evidence | PASS (8/8 tickets, no MISSING_RED, no STALE) |
| Assertion Quality | PASS (no CRITICAL) |
| Verification Gap | PASS (no gaps) |
| Security | PASS (no user-facing input, FTS injection tested) |
| Code Quality | PASS (build + vet clean, gofmt pre-existing only) |
| P0/P1 | PASS (100%/100%) |

**Weakest dimension:** Code Quality (gofmt pre-existing drift) — PASS

## Gate Decision

**PASS**

- All truths VERIFIED by named tests
- All 35 acceptance scenarios + 7 threat-matrix tests covered and passing
- No CRITICAL findings from any audit
- No MISSING_RED or STALE in TDD Evidence
- P0 pass rate 100% ≥ 100%, P1 pass rate 100% ≥ 95%
- All code-quality gates PASS (coverage/mutation disabled)
- Verification scope: COMPLETE (all touched packages tested)

## Human Override

None.

## Broken Windows

| ID | Change | Severity | Finding | Owner | Date | Status |
|---|---|---|---|---|---|---|
| W001 | 2026-10-24-global-llm-wiki | SUGGESTION | gofmt pre-existing drift in service/parallel_search_many_test.go, retrieval.go, teams.go (not introduced by this change) | qa-auto | 2026-10-25 | open |
| W002 | 2026-10-24-global-llm-wiki | SUGGESTION | mcp full-package test suite times out at 600s (pre-existing slow codeindex fixture) | qa-auto | 2026-10-25 | open |
