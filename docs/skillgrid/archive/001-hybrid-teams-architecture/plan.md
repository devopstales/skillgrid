# Plan: 001-hybrid-teams-architecture — Hybrid Agent Teams

## Technical Approach
SQLite control plane + FS data plane from `docs/plan/04-teams-sub-agents-memory.md`, on existing Go Mnemonic (`service.Service` → MCP `AddTool` → HTTP `mux` + `requireWriteAuth`). Migration `009_*` only (**003** owns `010_*`). Six Intent MCP tools; no embeddings, Python, session relay, or tracker. Optional content-write **Seam** on brief/output for later **003** tier hooks (no L0/L1/L2 here).

## Architecture Decisions

### Decision: Hybrid control + data plane
**Module / Interface / Seam / Adapter / Depth**: Teams facade; spawn/pull/read/submit/review/done; FS-then-SQL; SQLite + FS markdown; deep.
**Choice**: SQL = status/ownership/paths; content under `{project}/.skillgrid/files/{tasks,messages,reviews}/…`.
**Alternatives considered**: SQLite BLOBs; Python `file_store`.
**Rationale**: Intent rules + Go Mnemonic; small DB, Git-friendly markdown.

### Decision: Content-plane writer with optional hook
**Module / Interface / Seam / Adapter / Depth**: `files.ContentPlane`; Write/Read; content-write **Seam**; FS writer (+ later 003 hook); deep.
**Choice**: FS first → SQL insert → delete file on SQL failure; no-op post-write hook for brief/output.
**Alternatives considered**: Inline `os.WriteFile` per method; full tier registry now.
**Rationale**: One atomicity pattern; stays in scope for **003**.

### Decision: Facade + MCP/HTTP adapters
**Module / Interface / Seam / Adapter / Depth**: `service.Service` teams methods; openProject; MCP + HTTP **Adapter**s; deep.
**Choice**: Domain on `Service`; `tools_teams.go` + `registerTeamsTools`; HTTP `/teams/…` (not `/memory/reviews`).
**Alternatives considered**: Separate teams store package; MCP-only.
**Rationale**: Matches `tools_memory.go` / `http/server.go`; avoids review-route clash.

## Data Flow

```
Lead ─team_spawn_task─▶ ContentPlane(brief.md) ─▶ SQL tasks(pending)
Worker ─pull─▶ claim pending ─read─▶ brief.md
     ─submit_output─▶ output.md + task_results ─▶ status=review_spec
Reviewer ─submit_review─▶ reviews/*.md + reviews.passed
     ─mark_done─▶ status=done
```

## Impacted Files Map
| File | Action | Step | Description |
|------|--------|------|-------------|
| `skillgrid-cli/internal/mnemonic/store/migrations/009_teams_schema.sql` | Create | 01 | teams, team_members, tasks, messages, task_results, reviews |
| `skillgrid-cli/internal/mnemonic/files/content.go` | Create | 01 | ContentPlane Write/Read + FS-first rollback |
| `docs/skillgrid/agents/glossary/technical.md` | Modify | 01 | Team Task, Team Member, Agent Review, Inbox |
| `skillgrid-cli/internal/mnemonic/service/teams.go` | Create | 02 | Spawn/Pull/Read/SubmitOutput/SubmitReview/MarkDone |
| `skillgrid-cli/internal/mnemonic/service/service.go` | Modify | 02 | Wire teams via openProject if needed |
| `skillgrid-cli/internal/mnemonic/mcp/tools_teams.go` | Create | 03 | Six tools + handlers |
| `skillgrid-cli/internal/mnemonic/mcp/server.go` | Modify | 03 | `registerTeamsTools` in Start/NewServer |
| `skillgrid-cli/internal/mnemonic/mcp/server_test.go` | Modify | 03 | Expect six new tool names |
| `skillgrid-cli/internal/mnemonic/http/server.go` | Modify | 04 | `/teams/…` CRUD; writes `requireWriteAuth` |
| `skillgrid-cli/internal/mnemonic/service/teams_test.go` | Create | 05 | Atomicity, pull priority, status transitions |
| `skillgrid-cli/internal/mnemonic/mcp/tools_teams_test.go` | Create | 05 | Tool dispatch + FS/SQL parity |
| `skillgrid-cli/internal/mnemonic/http/teams_test.go` | Create | 05 | Auth-gated write / open read |

## Step WHAT

### Step 01-schema — What it delivers
- As operator: store open creates teams/tasks/messages/reviews tables without rewriting observations.
- Given a **Team Task** write, markdown lands under `.skillgrid/files/`; SQL stores paths/status only.
- Edge: migration idempotent via `index_meta`; SQL failure after FS write leaves no orphan file.

### Step 02-service-facade — What it delivers
- As orchestrator: spawn a **Team Task** with brief content and get a task id.
- Given pending work, pull claims highest-priority unassigned task; read returns brief markdown.
- Edge: output → `review_spec`; **Agent Review** sets `passed`; mark done → `done` + task_results row.

### Step 03-mcp-tools — What it delivers
- As agent: call `team_spawn_task`, `agent_pull_next_task`, `agent_read_task`, `agent_submit_output`, `agent_submit_review`, `agent_mark_done`.
- Given spawn, pull/read/submit round-trip keeps SQL/FS consistent.
- Edge: unknown id or no pending work → clear tool error (no panic).

### Step 04-http-routes — What it delivers
- As HTTP client: CRUD under `/teams/…`; writes need bearer when `SKILLGRID_HTTP_TOKEN` is set.
- Given token set, unauthenticated POST → 401; GETs stay open.
- Edge: no collision with `/memory/reviews`.

### Step 05-tests — What it delivers
- As maintainer: `go test ./…` covers facade atomicity, MCP registration/dispatch, HTTP write auth.
- Given FS ok and SQL fail, tests see file rollback.
- Edge: registry includes the six new tool names.

## Interfaces / Contracts
Tables per design doc §2 with `_path` columns. MCP names = Intent UAT. ContentPlane: Write then SQL; SQL error deletes file. Files root: project dir + `.skillgrid/files/`.

## Mnemonic Integration
New `team_*` / `agent_*` tools only — no `mem_*` / `code_*` / `web_cache_*` contract delta. Topic key: `sdd/001-hybrid-teams-architecture/plan`.

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED tests | Owner step |
|---|---|---|---|---|
| Documentation-like paths | N/A: only `.md` under `.skillgrid/files/`; no exec classification | — | — | — |
| Git / commit / push / PR | N/A: no VCS or PR automation in this change | — | — | — |
| **Mnemonic tool surface** | Applicable — six new MCP tools | `registerTeamsTools` only; mem/code/web unchanged | RED: `TestAllToolsRegistered` until six names; bad spawn → tool error not panic | 03-mcp-tools, 05-tests |
| **Shared-convention drift** | N/A: no `_shared/conventions/*` edits | — | — | — |

## Migration / Rollout
Additive `009_teams_schema.sql` via embed + `index_meta`. No flag. Rollback: drop migration + teams code (all additive).

## Open Questions
- [ ] Defer `agent_send_message` / `agent_read_inbox` (design doc) to a follow-on — Intent omits them; `messages` table kept for HTTP readiness.
