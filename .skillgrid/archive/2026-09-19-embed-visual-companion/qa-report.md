# QA Report — 2026-09-19-embed-visual-companion

**Date:** 2026-09-21 · **Branch:** release/2 · **HEAD:** 3297a0c · **Plan commits:** e490ed1..3297a0c

## Verdict: PASS

## Scope
Visual Companion (Mnemonic Decision Bridge): decision content convention over the existing `observations` table; `GET /mnemonic/decisions` + `POST /mnemonic/decisions/{id}/answer` (write-gated, versioned); React `/decisions` view (lazy chunk, polling, answer round-trip, sandboxed-iframe visual); `GET /prototype/{id...}` (traversal-guarded serve of `.skillgrid/prototype/`); openapi + user guide + MCP round-trip guard.

## Verification evidence

### Unit tests (parent re-verified 2026-09-21, `go test -count=1`)
- `./internal/mnemonic/http/...` — ok (decision + prototype + openapi suites; 12 TestDecision* + TestDecisionRoundTrip + TestPrototype* + TestPhase8_OpenAPI)
- `./internal/mnemonic/memory/...` — ok (UpdateContent + governance)
- `./internal/mnemonic/mcp/...` — ok (surface frozen; tool-count asserts `!= 88` incl. 5 handoff tools)
- `skillgrid-ui`: `npx vitest run` — 31/31 (9 decisions: inbox render, answer POST, answered state, no-iframe-without-visual, iframe-with-visual sandbox contract); `npm run build:check` — ok (tsc + vite; Decisions is its own lazy chunk `DecisionsPage-*.js`)

### Runtime harness (live `skillgrid serve` + curl, throwaway store, qa-harness.md)
20/20 PASS: seed via mem surface → list → answer → state flip + note → re-answer appends 2nd `observation_versions` row (SQLite-checked) → 404 bad id → 409 non-decision → prototype 200 text/html + nosniff / 404 missing / 400 encoded traversal (+ documented 307 for bare `..` via mux clean-path, no-SECRET invariant) → auth pass (token set: 401 no-Bearer, 200 with Bearer).

### TDD evidence audit
RED→GREEN per pair, tests-only RED commits: P1 c3b07ca→8bc12bf (+R6 fix 7cc50f4), P2 b7e39a4→76ec335, P3 4c884d2→1086a2b, T07 49b3a25→8148b5d. Scenario traceability: every blueprint threat-matrix RED test name is present and passing (skips-malformed, upsert-bumps-revision, answer-records, answer-appends-version, visible-to-reader, requires-auth, prototype-traversal, MCP round-trip).

### Verification-gap audit
- Must-haves 1–11 (blueprint Goal-Backward): each implemented + covered (final-review-verdict.md table). Backstops 1/4/9 held out to the live harness: covered (round-trip survives restart — same SQLite store; agent read-back via fresh mem_search in TestDecisionRoundTrip).
- 4-way schema consistency (Go struct ↔ openapi.yaml ↔ user guide ↔ TS type): aligned (final review).
- Gaps accepted (parked minors, follow-up tickets): live-browser iframe render (vitest covers the contract), fetch-boundary POST test (function-level mock), answerer identity (UI sends no `answerer` → `user:unknown`; backend supports it), 200-row list cap (documented in guide), sandbox attr duplication (DecisionCard vs SandboxPreview, no shared const).

### Pre-existing failures (NOT caused by this plan — reproduce at base e490ed1)
`go test ./...` has git-hook-dependent failures: TestSkillgridEvalSelfCorpus, TestBindingWriteFailureAborts, TestIdentityStableAcrossRemoteChange (commit-msg hook in temp repos), TestPhase2_CLI_Failure (gh CLI timing), TestEmbedderConfigDrivenSelection (env embedder). Checkpoint 2026-09-17 already records pre-existing git-hook test failures. All packages touched by this plan are green.

## Gate notes
- QA gate: PASS (no CRITICAL open; no waiver).
- Review gate: final two-axis review ran (FIXES REQUIRED → both merge-blocking findings fixed: 73e6f63 handoff UI files committed, 3297a0c mcp test-count asserts; scoped re-reviews CLEAN).
