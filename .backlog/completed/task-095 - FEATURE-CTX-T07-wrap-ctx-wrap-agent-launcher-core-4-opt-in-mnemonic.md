---
id: TASK-095
title: '[FEATURE] CTX T07 wrap: ctx wrap <agent> launcher (core-4 + opt-in) (mnemonic)'
status: wontfix
assignee: []
created_date: '2026-10-09 13:35'
updated_date: '2026-10-12 10:34'
labels:
  - mnemonic
  - wrap
milestone: m-10
dependencies:
  - TASK-092
  - TASK-093
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
mnemonic/internal/ctx/wrap/ — launch a named agent through the proxy. Starts the proxy, injects the agent's config to point at the proxy, backs up the previous config before writing, launches the agent (subprocess). Core-4 (claude + others) always available; opt-in agents (e.g. aider) honored from wrap config. Idempotent (re-running does not double-inject or re-backup). Agent code is never modified.

Current State:
- No ctx wrap launcher exists

Expected State:
- ctx wrap <agent> starts proxy + injects config + backup + launches agent
- Core-4 always available; opt-in agents honored
- Idempotent; agent source tree unchanged
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 go test ./internal/ctx/wrap/... green: wrap claude starts the proxy, injects config, writes a backup, launches the stub
- [ ] #2 The agent's turn through the proxy completes with savings headers
- [ ] #3 Re-running wrap is idempotent (no double-inject or re-backup)
- [ ] #4 Opt-in wrap aider works when aider is in config; a non-opt-in agent is rejected with a clear error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Agent source tree byte-identical before/after launch (agent code never modified)
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Create mnemonic/internal/ctx/wrap/wrap.go: launch a named agent through the proxy.
2. Start the loopback proxy, inject the agent's config to point at it, BACK UP the previous config before writing, then launch the agent as a subprocess.
3. Core-4 (claude + others) always available; opt-in agents (e.g. aider) honored from wrap config; non-opt-in -> clear error.
4. Idempotency: re-running wrap does not double-inject or re-backup.
5. Agent code is NEVER modified (only config injection).
6. Tests: wrap claude starts proxy + injects config + writes backup + launches stub; turn completes with savings headers; re-run idempotent; opt-in aider works, non-opt-in rejected.
7. SATISFIES scenarios: wrap-launch, wrap-idempotent, wrap-optin.
<!-- SECTION:PLAN:END -->

## Final Summary

<!-- SECTION:FINAL_SUMMARY:BEGIN -->
T07 (wrap launcher) complete. Impl ba1e4ab1 + I1/M1/M2 fix 52f56e48. Review APPROVED (0 Crit, 1 Imp closed, 2 Minor parked). `ctx wrap <agent>` starts loopback proxy, backs up pre-inject state, env-injects the base-URL, launches the agent subprocess; core-4 + opt-in, idempotent across CLI re-runs via persisted wrap-state marker; agent code never modified. SATISFIES wrap-core4-launches + wrap-optin-agent.
<!-- SECTION:FINAL_SUMMARY:END -->
