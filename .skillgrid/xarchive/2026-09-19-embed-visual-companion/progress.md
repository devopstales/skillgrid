# SDD ledger — plan: .skillgrid/specs/2026-09-19-embed-visual-companion/tasks.md

## Setup
- 2026-09-21: branch release/2, BASE e490ed1 (docs(backlog) sync — clean base)
- Working tree carried forward (NOT part of this plan, pre-existing handoff feature WIP): .agents/hooks/checkpoint-state.sh, .agents/skills/{resume,ship,work-unit-commits}/SKILL.md, docs/user-guide/{04-hooks,09-serve-dashboard}.md, skillgrid-cli/cmd/skillgrid/main.go, http/activity{,_test}.go, http/server.go, mcp/* (server, session_handoff, teams, memory_* tests), relay/cleave.go, relay/relay{,_test}.go, skillgrid-ui/src/app.tsx, skillgrid-ui/src/components/layout/AppLayout.tsx, skills-lock.json, + untracked handoff files.
  - NOTE: http/server.go and app.tsx + AppLayout.tsx are BOTH plan files and carried-WIP files. Pre-existing dirty state will land in this plan's commits; acceptable (size:exception single-PR delivery).
- Delivery: size:exception single PR (user decision 2026-09-21). Automatic mode.
- Spec: briefing.md (authority). Blueprint: blueprint.md. Findings: findings.md.
- Door check: TICKET-02 Task-1 of blueprint — GET /mnemonic/decisions must list type=decision + parse content.state from the existing store.

## Rulings
- R1 (2026-09-21): Delivery path = size:exception single PR (user decision). Cost if wrong: one big PR to review; rollback boundary table still names per-unit revert points.
- R2 (2026-09-21): Execution on release/2 (not a feature branch) — repo's active dev branch, matches prior SDD changes (014, web-admin-dashboard executed here). Cost if wrong: plan commits sit on release/2; ship step is an archive move, no branch integration.
- R3 (2026-09-21): The 3 RED tickets (T01, T03, T05) and their implementation tickets (T02, T04, T06) are paired: dispatch ONE implementer per RED+impl pair (the RED ticket is ~150 lines of test-only and the impl ticket makes it green; splitting them costs a full dispatch+review cycle for a file the next implementer just fills in). Pair = one review surface, TDD evidence intact (RED confirmed in the pair's tests before impl). Pairs: P1=T01+T02, P2=T03+T04, P3=T05+T06. T07 standalone. Cost if wrong: slightly larger review diffs; mitigated by per-pair review packages.
- R4 (2026-09-21): Pair execution order follows the wave graph: P1 (T01+T02) → P2 (T03+T04) → P3 (T05+T06) → T07. T03's RED is independent of T02, but T04 needs T02, so P2 after P1 keeps the graph honest without a parallel dispatch (skill is sequential). Cost if wrong: none found.

## Pre-flight conflict scan
| Pair (shared file/interface) | One produces → other consumes | Found |
|---|---|---|
| T01 → T02 (decisions_test.go, UpdateContent, routes) | T01 writes RED tests against proposed UpdateContent signature + route paths | T02 scope says "fill in" the test file — consistent. T01 must not implement the routes. OK. |
| T02 → T04 (POST /mnemonic/decisions/{id}/answer) | T02 implements the route; T04's api.ts POSTs to it | Request/response shape: blueprint Task 2 defines {optionId, note?} → but tasks.md T02 says {answer, rationale?, answerer?}. CONFLICT — see ruling R5. |
| T03 → T04 (DecisionsPage/DecisionCard tests) | T03 writes RED React tests; T04 implements components | T03 says "a decision without visual renders no iframe" — iframe lands in T06, so T03's "no visual → no iframe" case is trivially true at T04 time. OK. |
| T04 → T05 → T06 (DecisionCard.tsx iframe) | T04 creates DecisionCard; T05 adds iframe RED cases; T06 wires SandboxPreview | T05 says "DecisionCard renders sandboxed iframe only when content.visual set (attr sandbox=allow-scripts, not allow-same-origin)" — T04's DecisionCard has no visual field yet; T05's test compiles against T04's component. OK. |
| T06 ↔ existing stitchFile (server.go route registration) | T06 reuses stitchFile guard shape, new root .skillgrid/prototype/ | Blueprint says register BEFORE SPA fallback and NOT in apiPrefixes catch-all. Verify against embed.go at T06 dispatch. OK (noted). |
| T07 ↔ T02/T06 (openapi + round-trip test) | T07 documents the final route set (needs T06 done) + round-trip test needs T02's answer route | Consistent with Blocked-by. OK. |
| T02 self-consistency | "parse {answer, rationale?, answerer?}" vs blueprint Task 2 "{optionId, note?}" | CONFLICT — R5. |

- R5 (2026-09-21): Answer payload conflict: tasks.md TICKET-02 says parse `{answer, rationale?, answerer?}`; blueprint Task 2 + Must-Haves #3 say `{optionId, note?}` and "records the chosen optionId (and free text if any)"; decisionContent has AnsweredOption + AnswerNote fields. Ruling: BLUEPRINT WINS (spec is binding authority; tasks.md is its slice and the field names in its scope line are loose). Answer body = `{optionId, note?}`; content gets `AnsweredOption=optionId`, `AnswerNote=note`, `State=answered`, `UpdatedBy=user:<answerer|default>`. The RED tests (T01) are written against this shape. Cost if wrong: T01 tests need a rename if the user actually wanted the tasks.md names — trivial.

## Task ownership & dependencies
- P1 (T01+T02): Owns: skillgrid-cli/internal/mnemonic/http/decisions{,_test}.go, skillgrid-cli/internal/mnemonic/memory/governance.go (+ tests), skillgrid-cli/internal/mnemonic/http/server.go (route lines only). Needs: none.
- P2 (T03+T04): Owns: skillgrid-ui/src/features/decisions/*, skillgrid-ui/src/app.tsx (route line), skillgrid-ui/src/components/layout/AppLayout.tsx (nav line). Needs: P1 (POST endpoint exists).
- P3 (T05+T06): Owns: skillgrid-cli/internal/mnemonic/http/prototype{,_test}.go, server.go (route line), skillgrid-ui/src/features/decisions/DecisionCard{,.test}.tsx (iframe). Needs: P2 (DecisionCard exists).
- T07: Owns: skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml, docs/user-guide/10-decision-companion.md, decisions_test.go (round-trip test). Needs: P3 (final route set).
- Integration gate G1: P2 — decisions page POSTs to P1's route end-to-end (live serve + browser). G2: P3 — card iframe loads P3's route (live serve + browser).

## Final review + fixes
- Final two-axis review (final-review-verdict.md): FIXES REQUIRED — 1 merge-blocking: app.tsx/AppLayout.tsx import untracked ./features/handoff/HandoffHubPage → clean checkout fails build:check. 4-way schema (Go/openapi/docs/TS) aligned; must-haves 1–11 all implemented+tested; routes clean; 6 follow-up tickets triaged from parked minors.
- Fix 1 (73e6f63): committed the 30 handoff WIP files so the PR is self-contained. Re-review CLEAN but flagged: 6 mcp *_test.go tool-count asserts (83→88) still unstaged → clean-checkout go test ./internal/mnemonic/mcp/... would fail.
- Fix 2 (3297a0c): committed the 6 mcp test files with exact `!= 88` assert. Parent re-verify: http+memory+mcp suites green (2026-09-21).
- QA gate: live serve+curl round-trip harness PASS (20/20, qa-harness.md). Pre-existing unrelated failures in go test ./... (git-hook-dependent: TestSkillgridEvalSelfCorpus, TestBindingWriteFailureAborts, TestIdentityStableAcrossRemoteChange, TestPhase2_CLI_Failure, TestEmbedderConfigDrivenSelection) — NOT caused by this plan (reproduce on e490ed1; checkpoint.json 2026-09-17 already notes pre-existing git-hook test failures).
- Branch head for ship: 3297a0c (plan commits e490ed1..3297a0c).

## Tickets
- T01+T02 (P1): complete (commits e490ed1..7cc50f4, review clean; fix round 1/5 for R6 re-answer-appends-version; 5 minors parked: see p1-review-verdict.md — deferred to final review triage)
- R6 (2026-09-21): re-answer of an answered decision APPENDS a version (blueprint threat matrix wins over the no-op reading in tasks.md). Cost if wrong: version history gets one extra row per re-answer; no-op reading would have needed a handler branch restored.
- T03+T04 (P2): complete (commits 7cc50f4..76ec335, review APPROVED; 3 LOWs parked: handoff import untracked in committed tree [handoff-owner issue], answer POST mocked at function level not fetch boundary, no live-store round-trip [runtime harness gap — to be covered at QA gate])
- T05+T06 (P3): complete (commits 76ec335..1086a2b, review APPROVED; 3 minors parked: sandbox contract duplicated in DecisionCard vs SandboxPreview [no shared const], bare-.. 307-redirect nuance documented in tests, no live-browser iframe render [QA gate harness])
- T07: complete (commits 1086a2b..8148b5d, review APPROVED; 2 LOW + 1 NIT parked: report overclaims 2nd YAML parse error [wording-only], round-trip test is httptest-based not live-MCP-stdio, doc mentions 200-row cap [informational])
- T07 side-fix (in scope, review-verified): openapi.yaml was invalid YAML pre-ticket (plain-scalar parse error at ~line 2649, Phase 7 substring tests never caught it) — fixed via `|-` conversion + new TestPhase8_OpenAPI parse+dup-key guard.
