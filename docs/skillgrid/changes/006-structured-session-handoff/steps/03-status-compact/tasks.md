# Tasks: 006-structured-session-handoff — Step 03-status-compact

> Goal: `session_status` + thin `knowledge_compact`.
> Depends on: 02-handoff-resume

> Change-level forecast: see `../01-relay-schema/tasks.md`
> Decision needed before apply: Yes · Chained PRs recommended: Yes · Chain strategy: pending · 400-line budget risk: High

## Execution

- [ ] 03.1 Create `skillgrid-cli/internal/mnemonic/relay/status.go` — Status aggregation (handoff count + last known cost/context when caller supplies stats).
- [ ] 03.2 Create `skillgrid-cli/internal/mnemonic/relay/compact.go` — thin `CompactKnowledge` refreshes `.cleave/KNOWLEDGE.md` from handoff inputs/session notes only (no Fact Memory).
- [ ] 03.3 RED (Mnemonic tool surface): in `skillgrid-cli/internal/mnemonic/mcp/server_test.go`, assert `session_status` and `knowledge_compact` are registered; `mem_save` still works; `knowledge_compact` succeeds with **no** Fact Memory dependency (compounding file only) — expect fail until tools land.
- [ ] 03.4 Make 03.3 pass — create `skillgrid-cli/internal/mnemonic/mcp/tools_session_status.go` registering `session_status` + `knowledge_compact` via the step-02 registrar hook **without** re-editing `server.go`.
- [ ] 03.5 Cover WHAT: `session_status` reports handoff count and last cost/context; thin `knowledge_compact` refreshes `KNOWLEDGE.md` without facts.
- [ ] 03.6 Cover WHAT edge: no handoffs yet → zero counts / empty knowledge file, not a crash.
