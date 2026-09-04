# Tasks: 006-structured-session-handoff — Step 05-handoff-watchdog

> Goal: Optional context-limit watchdog → `session_handoff`.
> Depends on: 02-handoff-resume

> Change-level forecast: see `../01-relay-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 05.1 Decide usage signal (client `%` vs token estimate) for the watchdog Input fraction; document choice in `skillgrid-cli/internal/mnemonic/relay/watchdog.go` comment.
- [ ] 05.2 Create `skillgrid-cli/internal/mnemonic/relay/watchdog.go` — flag/env-gated (`SKILLGRID_HANDOFF_WATCHDOG` + threshold); off by default; Check→ same `Handoff` path as explicit handoff.
- [ ] 05.3 Cover WHAT: with watchdog enabled and usage past threshold, triggers a Session Handoff via the same path as explicit handoff.
- [ ] 05.4 Cover WHAT edge: disabled/default → never auto-handoff; below threshold → no-op.
