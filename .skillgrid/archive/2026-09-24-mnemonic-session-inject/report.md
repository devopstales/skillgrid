# QA Report — 2026-09-24-mnemonic-session-inject

Tier: T2 (blueprint) | Floor: L2 + L3 | quality: configured (coverage_min 0 = advisory, mutation disabled, p0 100 / p1 95)

## Verdict

**PASS** — 0 CRITICAL, 0 unchecked tickets, all gates green.

## Test Plan

| # | Scenario (acceptance.feature) | Layer | Test (named) | Cadence | Rank |
|---|-------------------------------|-------|--------------|---------|------|
| 1 | L1 summary is deterministic | unit | `TestDistillSummary_Deterministic` | pr | P0 |
| 2 | L1 summary respects token cap | unit | `TestDistillSummary_TokenCap`, `TestDistillSummary_TokenCapSmallCap` | pr | P0 |
| 3 | L1 summary excludes sensitive events | unit | `TestDistillSummary_ExcludesSensitive` | pr | P0 |
| 4 | L1 summary is empty for few events | unit | `TestDistillSummary_FewEvents` | pr | P1 |
| 5 | Auto-prepend on resume returns summary | unit | `TestAutoPrepend_Resume` | pr | P0 |
| 6 | Auto-prepend on fresh session returns empty | unit | `TestAutoPrepend_FreshSession` | pr | P0 |
| 7 | Hybrid retrieve BM25-only when no embedder | unit | `TestHybridRetrieve_BM25Only` | pr | P0 |
| 8 | Hybrid retrieve vector leg when embedder active | unit | `TestHybridRetrieve_VectorLeg` | pr | P0 |
| 9 | Hybrid retrieve is project-scoped | unit | `TestHybridRetrieve_ProjectScope` | pr | P1 |
| 10 | Hybrid retrieve all projects | unit | `TestHybridRetrieve_AllProjects` | pr | P1 |
| 11 | mem_inject_session tool is registered | unit | `TestMemInjectSession_Registered` | pr | P0 |
| 12 | mem_inject_session returns ranked items | unit | `TestMemInjectSession_ReturnsRankedItems` | pr | P0 |
| 13 | mem_inject_session all projects | unit | `TestMemInjectSession_AllProjects` | pr | P1 |
| 14 | mem_inject_session degraded flag | unit | `TestMemInjectSession_Degraded` | pr | P0 |
| 15 | Context block includes token costs | unit | `TestRenderContextBlock_Format` | pr | P0 |
| 16 | Context block shows BM25-only when degraded | unit | `TestRenderContextBlock_Degraded` | pr | P1 |

Supporting (not scenario rows): `TestLatestCompletedSession` (status='ended' query), `TestRenderContextBlock_Empty`.

Out of Scope: live CLI resume call-site (no hook exists — TICKET-02 ruled it dead code; the shipped surface is the T5 MCP tool); cross-bucket embedder availability at runtime (covered at unit level via SetDirEmbedder + env gate).

## Goal-Backward Verification

| Truth | Level | Evidence | Status |
|-------|-------|----------|--------|
| L1 summary is deterministic, token-capped, privacy-filtered | Truth/Artifact/KeyLink/DataFlow | `TestDistillSummary_*` (4) pass over seeded events → `DistillSummary` in summary.go | VERIFIED |
| Auto-prepend returns the last ended session's L1 on resume, empty when none | Truth/KeyLink | `TestAutoPrepend_{Resume,FreshSession}` + `TestLatestCompletedSession` (`status='ended'`) | VERIFIED |
| HybridRetrieve fuses BM25+vector (RRF), degrades to BM25-only without embedder | Truth/KeyLink/DataFlow | `TestHybridRetrieve_{BM25Only,VectorLeg,ProjectScope,AllProjects}` over seeded stores + HashEmbedder | VERIFIED |
| RenderContextBlock annotates every item + footer with token cost, BM25-only when degraded | Truth/Artifact | `TestRenderContextBlock_{Format,Empty,Empty}` | VERIFIED |
| `mem_inject_session` MCP tool is registered and wraps HybridRetrieve+Render | KeyLink/DataFlow | `TestMemInjectSession_{Registered,ReturnsRankedItems,AllProjects,Degraded}`; server surface 92→93 + name list | VERIFIED |

## Traceability Matrix (15 scenarios)

| Scenario | Covering test | Ran | Result | Status |
|----------|---------------|-----|--------|--------|
| L1 summary is deterministic | TestDistillSummary_Deterministic | yes | pass | COMPLIANT |
| L1 summary respects token cap | TestDistillSummary_TokenCap(+SmallCap) | yes | pass | COMPLIANT |
| L1 summary excludes sensitive events | TestDistillSummary_ExcludesSensitive | yes | pass | COMPLIANT |
| L1 summary is empty for few events | TestDistillSummary_FewEvents | yes | pass | COMPLIANT |
| Auto-prepend on resume returns summary | TestAutoPrepend_Resume | yes | pass | COMPLIANT |
| Auto-prepend on fresh session returns empty | TestAutoPrepend_FreshSession | yes | pass | COMPLIANT |
| Hybrid retrieve BM25-only when no embedder | TestHybridRetrieve_BM25Only | yes | pass | COMPLIANT |
| Hybrid retrieve vector leg when embedder active | TestHybridRetrieve_VectorLeg | yes | pass | COMPLIANT |
| Hybrid retrieve is project-scoped | TestHybridRetrieve_ProjectScope | yes | pass | COMPLIANT |
| Hybrid retrieve all projects | TestHybridRetrieve_AllProjects | yes | pass | COMPLIANT |
| mem_inject_session tool is registered | TestMemInjectSession_Registered | yes | pass | COMPLIANT |
| mem_inject_session returns ranked items | TestMemInjectSession_ReturnsRankedItems | yes | pass | COMPLIANT |
| mem_inject_session all projects | TestMemInjectSession_AllProjects | yes | pass | COMPLIANT |
| mem_inject_session degraded flag | TestMemInjectSession_Degraded | yes | pass | COMPLIANT |
| Context block includes token costs | TestRenderContextBlock_Format | yes | pass | COMPLIANT |
| Context block shows BM25-only when degraded | TestRenderContextBlock_Degraded | yes | pass | COMPLIANT |

**scenarios: 16/16 COMPLIANT** (feature has 16 scenarios; the 15 above + the 16th render-degraded — all land COMPLIANT).

## TDD Evidence Audit

| Ticket | RED | GREEN | Scenario match | Verdict |
|--------|-----|-------|----------------|---------|
| T1 summary core | RED (summary pkg undefined) | GREEN (5 tests) | match | OK |
| T2 AutoPrepend | RED (autoprepend undefined) | GREEN (3 tests) | match | OK |
| T3 HybridRetrieve | RED (retrieve undefined) | GREEN (4 tests) | match | OK |
| T4 RenderContextBlock | RED (render undefined) | GREEN (3 tests) | match | OK |
| T5 mem_inject_session | RED (HandleMemInjectSessionForTest undefined) | GREEN (4 tests) | match | OK |

## Verification-Gap Audit

- No regression gaps: every changed behavior has a passing covering test in the same package.
- Adoption: `mem_inject_session` registered in BOTH `Start()` and `NewServer()` (server.go); T3/T4 wired into the T5 handler (thin wrapper). No missing-adoption.
- Broken-verification: T3 AllProjects test strengthened to fail under concatenation (verified RED vs old). No broken-oracle.
- Missing-oracle: none — every scenario has a named passing test.

## Gates (run this session)

- `go test ./internal/mnemonic/session_inject/ ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ -count=1` → all `ok`
- `go build ./...` → ok
- Trivy: advisory only (`fail_on: ""`) — findings reported, never blocking. No new dependencies (go.mod untouched) → dependency surface unchanged.

## Findings

- **CRITICAL:** none.
- **WARNING (non-blocking, parked):**
  - T5 alias-drift: `TestMemInjectSession_Registered` drives `HandleMemInjectSessionForTest` (pure alias); registration is independently proven by the server surface name-list + 92→93 count pins. Optional one-line hardening (assert `NewServer().ListTools()` contains the tool name).
  - T4 test helper `containsAll` naming.
  - T2 CLI resume call-site absent (ruling, not a defect — recorded in T2 report Concerns).
- **Minor:** T5 pin-comment style (pre-existing convention).

## Note on known pre-existing noise (not this change)
- `cmd/skillgrid` CLI full-suite failures stem from the unbuilt `ui/dist` embed (passes in isolation); not introduced by this change (no cmd/ files touched).
- `go vet` context-cancel leak at `internal/mnemonic/memory/budget.go:114` — pre-existing, untouched by this change.
