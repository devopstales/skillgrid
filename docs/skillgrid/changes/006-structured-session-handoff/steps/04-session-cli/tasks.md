# Tasks: 006-structured-session-handoff — Step 04-session-cli

> Goal: `skillgrid session handoff|resume|status`.
> Depends on: 02-handoff-resume, 03-status-compact

> Change-level forecast: see `../01-relay-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 04.1 Create `skillgrid-cli/cmd/skillgrid/session.go` — `session handoff|resume|status` calling the same `relay` Module / project store as MCP.
- [ ] 04.2 Modify `skillgrid-cli/cmd/skillgrid/main.go` to dispatch the `session` subcommand.
- [ ] 04.3 Cover WHAT: CLI handoff/resume/status mirror MCP outcomes on the same project store.
- [ ] 04.4 Cover WHAT edge: bad flags / missing id → non-zero exit with stderr message.
