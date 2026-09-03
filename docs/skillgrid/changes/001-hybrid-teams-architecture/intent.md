# Intent: 001 — Hybrid Agent Teams Architecture

## Business Problem
Skillgrid's Mnemonic is a single-agent memory engine (observations, code index, web cache). It has no concept of multi-agent teamwork: there's no way for agents to claim tasks, hand off work via messages, or be peer-reviewed by other agents. The design doc `docs/plan/04-teams-sub-agents-memory.md` describes the desired architecture (SQLite Control Plane + Filesystem Data Plane), but it has never been implemented. We need this to support the SDD subagent-execution and parallel-agent dispatch skills, which delegate work to fresh agent processes that must track their work, communicate, and be reviewed.

## Target Users & Situations
- **Agent orchestrator** — spawns tasks, assigns them to sub-agents, tracks progress across the team.
- **Agent worker (sub-agent)** — pulls next available task, reads its brief, writes output, sends messages to peers, submits reviews.
- **Agent reviewer** — reads a task's output, writes spec-compliance or code-quality review.
- Urgency: Medium — the `subagent-execution` and `dispatching-parallel-agents` skills exist but cannot function end-to-end without this backend.

## Business Rules
- SQLite stores metadata only (task status, ownership, paths, relations); actual content lives as `.md` files on the filesystem under `.skillgrid/files/`.
- Write filesystem content first, then insert the SQL row; if SQL fails, delete the orphaned file (atomicity via rollback pattern).
- All new SQL is additive migrations (`009_*` and up) — no existing schema changes.
- MCP tools follow the existing `s.AddTool(toolDef, handlerFunc)` pattern in `tools_memory.go`/`tools_code.go`.
- HTTP routes follow the existing `mux.HandleFunc` pattern with bearer-token auth on writes.

## Success Criteria (UAT-level)
- [ ] `team_spawn_task` MCP tool creates a task with a `brief.md` file on disk and a SQL row, returning the task ID.
- [ ] `agent_pull_next_task` returns the highest-priority unassigned task for the calling agent.
- [ ] `agent_read_task` returns the brief markdown content from `.skillgrid/files/tasks/{id}/brief.md`.
- [ ] `agent_submit_output` writes `output.md` and advances the task to `review_spec` status.
- [ ] `agent_submit_review` writes a review markdown file and sets `passed` boolean on the `reviews` table.
- [ ] `agent_mark_done` transitions the task to `done` and populates `task_results`.
- [ ] All Go tests pass: `task test` (go vet + go test ./...).

## Scope

### In Scope
- New SQL migration: `teams`, `team_members`, `tasks`, `messages`, `task_results`, `reviews` tables.
- Service facade methods on `service.Service` for each MCP tool.
- 6 MCP tools registered in a new `tools_teams.go` file.
- Filesystem helpers for `.skillgrid/files/tasks/{id}/`, `.skillgrid/files/messages/{id}/`, `.skillgrid/files/reviews/{id}/`.
- HTTP routes for team/task/message/review CRUD (write routes auth-gated).
- Unit + integration tests following existing patterns.

### Out of Scope
- Embeddings / vector search (covered by `docs/plan/07-nemonic-hybid-search.md`, separate change).
- Python microservice (docs/plan/05 proposes this for Hermes memory — not needed for teams).
- Session relay / Cleave-style handoffs (docs/plan/06 — separate change).
- `project_name` / issue tracker integration (Backlog.md already wired via sdd-init).

## Step Blueprint
- `01-schema`: Add SQLite migration for teams/tasks/messages/reviews tables + filesystem data-plane dirs.
- `02-service-facade`: Add Service methods (spawn, pull, read, submit, review, done) following `openProject` pattern.
- `03-mcp-tools`: Register 6 new MCP tools in `tools_teams.go` with handlers.
- `04-http-routes`: Add HTTP routes for task/message/review CRUD (auth-gated writes).
- `05-tests`: Unit tests for service + integration tests for MCP tools.

## Affected Areas
| Area | Impact | Description |
|------|--------|-------------|
| `internal/mnemonic/store/migrations/009_teams_schema.sql` | New | Teams/tasks/messages/reviews SQL schema |
| `internal/mnemonic/service/service.go` | Modified | Add task/team/message/review service methods |
| `internal/mnemonic/mcp/tools_teams.go` | New | 6 MCP tool definitions + handlers |
| `internal/mnemonic/mcp/server.go` | Modified | Register `registerTeamsTools(s)` |
| `internal/mnemonic/http/server.go` | Modified | Add team/task/message/review routes |
| `internal/mnemonic/service/` | New file | Filesystem content-plane helpers (atomic write + SQL insert) |
| `docs/skillgrid/glossary/` | Modified | Add domain terms (task, team member, review, inbox) |

## Risks
| Risk | Likelihood | Mitigation |
|------|------------|------------|
| FS+SQL atomicity gap | Med | Use the proposed write-FS-first-then-SQL-with-FS-rollback pattern; wrap in a single DB transaction |
| Migration ordering | Low | `009_*` sorts after `008_*` via existing `sort.Strings(names)` |
| MCP tool surface proliferation | Med | Group all team tools in one file (`tools_teams.go`); follow existing `registerMemoryTools` pattern |
| HTTP auth parity | Low | Reuse `requireWriteAuth` for all task/team mutation routes |

## Rollback Plan
- Remove `009_teams_schema.sql` migration file — the migration system is idempotent (checks `index_meta`); existing databases will not re-run it.
- Remove `tools_teams.go`, revert `server.go` registration, revert `service.go` additions — all additive code, no existing lines altered.
- HTTP routes are additive — remove `s.mux.HandleFunc(...)` lines for team endpoints.
- Filesystem content under `.skillgrid/files/` is opt-in; no existing users have it.

## Dependencies
- None external — all Go stdlib + existing deps (`modernc.org/sqlite`, `mcp-go`).
- `docs/plan/04-teams-sub-agents-memory.md` (design reference, already in repo).
- `docs/plan/05-hermes-memory.md` (L0/L1/L2 tiered storage — separate change, not a blocker).
