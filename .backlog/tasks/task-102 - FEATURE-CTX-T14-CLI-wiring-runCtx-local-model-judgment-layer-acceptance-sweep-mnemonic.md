---
id: TASK-102
title: >-
  [FEATURE] CTX T14 CLI wiring (runCtx) + local-model judgment layer +
  acceptance sweep (mnemonic)
status: done
assignee: []
created_date: '2026-10-09 13:39'
updated_date: '2026-10-13 19:03'
labels:
  - mnemonic
  - cli
  - judgment
  - acceptance
milestone: m-10
dependencies:
  - TASK-092
  - TASK-094
  - TASK-095
  - TASK-096
  - TASK-097
  - TASK-098
  - TASK-099
  - TASK-100
references:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Wire all ctx subcommands (proxy, wrap, learn, stats) into the runCtx dispatcher (cmd/mnemonic/ctx.go) with a clean --help. Add the local-model judgment layer (ADR-0034): local Ollama, truncate-only, fingerprint cache, fail-open (Ollama down -> deterministic pre-filter, unit not dropped). Final acceptance sweep: all 23 scenarios covered.

Current State:
- Individual pieces (proxy, wrap, learn, observability) exist; not yet wired into the CLI dispatcher, no judgment layer, no final acceptance sweep

Expected State:
- All ctx subcommands wired + clean --help
- Judgment layer fail-open against local Ollama
- All 23 acceptance scenarios covered
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go build ./... and go test ./... green at the repo root
- [ ] #2 ctx --help lists all subcommands (proxy, wrap, learn, stats); an unknown subcommand errors clearly
- [ ] #3 Judgment layer: Ollama down -> deterministic pre-filter, no unit dropped (fail-open)
- [ ] #4 All 23 acceptance.feature scenarios are covered by a passing test or manual check; none left uncovered
- [ ] #5 No new 3rd-party dependency in go.mod (stdlib net/http + httputil.ReverseProxy only)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Wire all ctx subcommands (proxy, wrap, learn, stats) into runCtx in cmd/mnemonic/ctx.go; ctx --help lists them; unknown subcommand -> clear error.
2. Add the local-model judgment layer (ADR-0034): call local Ollama, truncate-only, fingerprint cache in ctx_judgment_cache.
3. Fail-open: Ollama down -> deterministic pre-filter, no unit dropped.
4. Acceptance sweep: map all 23 acceptance.feature scenarios to a passing test or documented manual check; confirm none uncovered.
5. Repo-level gates: go build ./... + go test ./... green at the root; no new 3rd-party dep in go.mod.
6. SATISFIES scenarios: cli-help, judgment-fail-open, judgment-cache, acceptance-sweep.
<!-- SECTION:PLAN:END -->
