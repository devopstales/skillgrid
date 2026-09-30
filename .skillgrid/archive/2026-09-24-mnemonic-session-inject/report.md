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

## Gate Decision

**Verdict:** PASS
**Gate:** qa (T2 floor, L2+L3) · Trivy advisory-only (no new deps; dependency surface unchanged)
**Blockers:** none.
**Human Override:** none.

## Final-State Facts

- **Shipped to:** `release/2` (long-lived integration branch; work delivered as a single commit-range, kept as-is per option 3 — NOT pushed).
- **Code range:** `4f83b7d3..30d52142` (7 commits: T1 `6c1a1f5e` + fix `dff44175`; T2 `0013a49b`; T3 `7153302a` + fix `c0974e87`; T4 `bfba6215`; T5 `a409da00`; doc fix `30d52142`).
- **Close-out commit:** `58521d94` (archive-move `specs/`→`archive/` + 4 technical terms + `state.yaml`).
- **Files:** 27 changed, +1448/−20. New: `session_inject/{summary,privacy,autoprepend,retrieve,render}.go` + 5 test files; `memory/{latest_completed,recent_observations}.go` + tests; `mcp/tools_session_inject.go` + test. Supporting seams: `memory/service.go` (`DirEmbedder`/`StorePath`/`RecentObservations`), `memory/retrieval.go` (`DirEmbedder` reader), `mcp/server.go` (2 registration lines), 6 tool-count pins 92→93.
- **Rollback:** `git revert 4f83b7d3..30d52142`.
- **Integration evidence:** `go build ./...` ok; `go test ./internal/mnemonic/session_inject/ ./internal/mnemonic/memory/ ./internal/mnemonic/mcp/ -count=1` all `ok` on the integrated tree (release/2 head).

## Sourced Learnings

### Decisions
- **Wrap `BlendedSearch`, don't rebuild fusion** — T3 delegates per-bucket fused search to the existing `memory.Service.BlendedSearch` (search_embed.go:117, already RRF k=60 + FTS floor + vector leg). The ONLY custom fusion is the cross-bucket global RRF in `crossProjectSearch` (retrieve.go:95, `score+=1/(60+rank+1)`, key `project:id`). Tradeoff: reuses calibrated fusion + keeps a single source of truth; cost: cross-bucket fusion is bespoke because `ReciprocalRankFusion` only blends two lists. — source: retrieve.go:95, search_embed.go:117; obs 54.
- **Deliver on `release/2` as a single commit-range (not a feature-branch-chain)** — risk was High (400-line budget) and the delivery strategy said `feature-branch-chain`, but `release/2` IS the integration branch, so carving PR1/PR2/PR3 happens at ship; not a one-way door (additive, no migration/contract change). — source: tasks.md Delivery Strategy; obs 59.
- **AutoPrepend has no CLI call-site by design** — no resume hook exists (relay removed in TICKET-07); the function is ready-to-wire, the shipped surface is the T5 MCP tool. Tradeoff: L1 is currently not auto-invoked by any command; cost: a future resume path must wire it. — source: obs 53; T2 report Concerns.

### Lessons
- **The blueprint's Task-3 pseudo-code was wrong — verify the seam before delegating.** The blueprint showed hand-rolled `vectorRetrieve`+`rrfFuse`; the fused search already existed in `BlendedSearch`. Reading the real code first turned a 3-ticket re-implementation into a thin wrapper. — source: tasks.md verified-seam corrections; obs 54.
- **"Completed" is `status='ended'` in this codebase, not `'completed'`** — `SessionEnd` sets `status='ended'`; the blueprint said 'completed'. A literal read would have queried zero rows. — source: obs 53; latest_completed.go.

### Patterns
- **Cross-bucket dedup keys on `project:id`, not bare `id`** — each sibling store auto-increments `observations.id` from 1, so bare-id dedup silently drops sibling hits. The `project:id` key is the reusable rule for any cross-project merge. — source: retrieve.go:95; obs 54 (was the initial AllProjects test failure).
- **MCP registration proof = the `ForTest` alias + the tool-count pin, not an in-process JSON-RPC call** — mcp-go v0.58 `NewServer()` nil-panics on in-process JSON-RPC without `WithHooks`; drive `Handle*ForTest` and assert the surface count/name list. — source: obs 56; tools_session_inject_test.go:160.
- **`pinProjectCwd` for MCP tests** — pin `MNEMONIC_PROJECT` + `SetService` + `chdir` (with `t.Cleanup` restoring both) so the handler's CWD-resolved store is the seeded temp store; FK trap: seed the session on the target project's handle. — source: obs 56.

### Surprises
- **`degraded` can overstate the vector leg** — with an embedder attached but `MNEMONIC_EMBED` off (or no stored vectors), `queryVector` returned `degraded=false` even though `BlendedSearch`'s vector leg never ran (it gates on the env). The final whole-change review caught it (Important); fixed by documenting the `queryVector` precondition (embedder ∧ env ∧ embed) at retrieve.go:79. — source: final review; commit 30d52142.
- **Tool-count pins are load-bearing in the error strings too** — 5 of the 6 pins had the stale count in the comparison literal AND the error message; bumping only one would leave a misleading failure. — source: obs 56; T5 diff.

## Acceptance Verdict

**accepted** — goal met, no open items. Both layers are wired and verified: L1 auto-prepend on resume (6/6 scenarios) and L2 `mem_inject_session` hybrid retrieval + render (10/10 scenarios). Goal-backward: every truth VERIFIED; traceability 16/16 COMPLIANT; 0 CRITICAL. The single Important finding (degraded flag) was fixed pre-ship.

## Prior-Change Follow-Through

- **Addressed:** Wave 3 (session-inject) — the next item flagged in the monitoring ship context (obs 50: "Next in dependency order: Wave 3 = 2026-09-24-mnemonic-session-inject (needs slicing first)"). Now shipped.
- **Still open (→ waves 4a/4b/5):** migration `043_*.sql` double-booked (bitemporal-audn vs second-brain); ADR-0011 double-claimed (bitemporal-audn vs memory-improvements). These predate this change and are unaffected by it.
- **No open items from this change** that need a follow-up.

## Lineage

- **Ship context (Mnemonic):** obs 59 — `skillgrid/2026-09-24-mnemonic-session-inject/ship`.
- **Ticket observations (Mnemonic):** 53 (T2 autoprepend) · 54 (T3 hybrid) · 56 (T5 MCP tool).
- **Review:** no standalone `review.md` for this change — review ran as per-ticket two-axis (Spec + Quality) via the subagent-execution review-package, plus a final whole-change two-axis review over `4f83b7d3..30d52142`. Floor met on both axes; the whole-change review's independence grade is high (fresh subagent, diff-only input). Pre-merge catches: T1 (token-cap headroom), T3 (concatenation→global-RRF, sibling-hidden), T5 (tool-count pins verified count-only), whole-change (degraded flag — the one Important). No Critical escaped to QA.
- **Git:** code `4f83b7d3..30d52142` on release/2; close-out `58521d94`.

## Move Evidence

- `git mv .skillgrid/specs/2026-09-24-mnemonic-session-inject .skillgrid/archive/2026-09-24-mnemonic-session-inject` (close-out commit `58521d94`).
- `diff -r` against a pre-move recursive snapshot: **empty** (status 0) — move exact, no truncation/alteration.
- `report.md` (this file, QA half) moved with the folder; retro half appended in place by reflect.
