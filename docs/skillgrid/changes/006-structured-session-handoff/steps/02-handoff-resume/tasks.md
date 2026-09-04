# Tasks: 006-structured-session-handoff — Step 02-handoff-resume

> Goal: `.cleave/` + MCP handoff/resume; L0 paths as needed.
> Depends on: 01-relay-schema

> Change-level forecast: see `../01-relay-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 02.1 Create `skillgrid-cli/internal/mnemonic/relay/cleave.go` — WriteBundle/ReadBundle for `.skillgrid/.cleave/{PROGRESS,KNOWLEDGE,NEXT_PROMPT}.md`; soft-optional L0 under `.skillgrid/workspace/sessions/{id}/`.
- [ ] 02.2 Create `skillgrid-cli/internal/mnemonic/relay/relay.go` — `Handoff` / `Resume` writing SQL rows + cleave files; fail closed (no orphan row without files); clear errors for missing `.cleave/` / unknown id.
- [ ] 02.3 RED (Mnemonic tool surface): in `skillgrid-cli/internal/mnemonic/mcp/server_test.go`, assert `session_handoff` and `session_resume` are registered **and** `mem_save` remains registered and still dispatches successfully — expect fail until tools land.
- [ ] 02.4 Make 02.3 pass — create `skillgrid-cli/internal/mnemonic/mcp/tools_session_handoff.go` (`session_handoff` / `session_resume`, JSON-only) with a registrar hook; call it from `skillgrid-cli/internal/mnemonic/mcp/server.go` without dropping `mem_*`.
- [ ] 02.5 Modify `.gitignore` to ignore `.skillgrid/.cleave/` by default.
- [ ] 02.6 Cover WHAT: `session_handoff` writes three `.cleave/` files and a `session_handoffs` row; `session_resume` returns next-session prompt (optional archive when requested).
- [ ] 02.7 Cover WHAT edge: missing `.cleave/` or unknown id → clear error; no orphan SQL row without files.
