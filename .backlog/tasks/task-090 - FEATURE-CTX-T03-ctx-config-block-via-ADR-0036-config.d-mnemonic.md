---
id: TASK-090
title: '[FEATURE] CTX T03 ctx: config block via ADR-0036 config.d (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-10-09 13:33'
updated_date: '2026-10-09 13:39'
labels:
  - mnemonic
  - config
milestone: m-10
dependencies: []
references:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/acceptance.feature
documentation:
  - .skillgrid/specs/2026-10-08-context-orchestrator/briefing.md
  - .skillgrid/specs/2026-10-08-context-orchestrator/tasks.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Create mnemonic/internal/ctx/config/ that loads a ctx: block using the ADR-0036 config.d convention (builtin -> ~/.skillgrid/config.d/ctx.yaml -> first-found walking up), reusing the findConfigD shape from internal/config/pricing.go. Defaults for all keys; per-key merge with repo-local winning. The clm: block is still read from .skillgrid/config.yaml (unchanged).

Current State:
- No ctx: config block exists; findConfigD shape already proven in internal/config/pricing.go (ADR-0036)

Expected State:
- ctx: block loaded via config.d precedence (repo-local > machine-local > builtin)
- All keys have safe defaults
- clm: block resolution from .skillgrid/config.yaml unchanged
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/config/... green: repo-local key wins over machine-local over builtin
- [ ] #2 Absent repo-local falls to machine-local; neither present -> defaults
- [ ] #3 clm block still resolves from .skillgrid/config.yaml (sentinel value)
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
1. Create mnemonic/internal/ctx/config/config.go mirroring the findConfigD shape from internal/config/pricing.go.
2. Define the CtxConfig struct with all keys + safe defaults (port, modes, retry, thresholds, wrap agents, judgment, observability).
3. Implement Load(ctx): builtin defaults -> ~/.skillgrid/config.d/ctx.yaml -> first-found walking up; per-key merge, repo-local wins.
4. Add tests: repo-local beats machine-local beats builtin; absent repo-local falls to machine-local; neither -> defaults.
5. Add a sentinel test proving the clm: block still resolves from .skillgrid/config.yaml.
6. SATISFIES scenarios: config-precedence, config-defaults, config-clm-unchanged.
<!-- SECTION:PLAN:END -->
