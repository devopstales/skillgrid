---
type: TechnicalDecision
title: "skill-pi ticketing: `ticket` tool (10 actions) backed by the `backlog` MCP server over stdio, not vendored tk or direct file I/O"
status: "accepted"
date: 2026-10-09
supersedes: none
---

# skill-pi ticketing: Backlog.md adapter over the `backlog` MCP server
## Context and Problem Statement

skill-pi needs the compact ticketing UX that `pi-tk-tickets` provides (one
model-visible `ticket` tool with ~10 actions + a `/tickets` slash command,
low prompt overhead, no auto-injection of ticket lists). The question is what
backing store and access layer that extension uses. Three candidates:

1. Vendor the `tk` bash script (as `pi-tk-tickets` does) and store tickets in
   tk's `.tickets/` format.
2. Parse/write skillgrid's existing Backlog.md files (`.backlog/tasks/*.md`)
   directly from the extension.
3. Call the existing `backlog` MCP server (stdio) that already manages
   Backlog.md for opencode/cursor/kilo.

skillgrid's ticketing config is `ticketing.type: backlogmd`
(`.skillgrid/config.yaml`), so introducing tk's `.tickets/` format would put
two ticket directories in the same repo with no sync between them.

## Considered Options

- **Vendor `tk` + `.tickets/` format** — identical to `pi-tk-tickets`, zero
  skillgrid-specific code. Rejected: a second ticket format in a repo that
  already uses Backlog.md, plus a migration story.
- **Direct file I/O on `.backlog/tasks/`** — fastest reads, no server
  dependency. Rejected: duplicates the MCP server's Backlog.md parsing logic
  and must stay in sync with the format as it evolves.
- **`backlog` MCP server over stdio** — reuses the existing server and its
  parsing logic; the server is already a required part of skill-pi's tool set
  (wired in skill-pi's `mcp.json`). Accepted.

## Decision Outcome

Chosen option: "**backlog MCP server over stdio**", because it keeps Backlog.md
as the single source of truth with a single parser, and the server is already
required in skill-pi's configuration — no new dependency, no second ticket
format.

The tool is named `ticket` (not `backlog`): it is shorter, matches the
`pi-tk-tickets` interface, and stays valid if skill-pi ever supports a second
ticket format.

The extension (`skill-pi/extensions/tickets.ts`) exposes one `ticket` tool
with 10 actions — `ready`, `list`, `show`, `create`, `start`, `complete`,
`archive`, `note`, `status`, `update` — plus a `/tickets` slash command with
the same surface. Action-to-MCP mapping: `ready` → `backlog_task_list
(ready: true)`, `list` → `backlog_task_list (status)`, `show` →
`backlog_task_view`, `create` → `backlog_task_create`, `start`/`status` →
`backlog_task_edit (status)`, `complete` → `backlog_task_complete`,
`archive` → `backlog_task_archive`, `note` → `backlog_task_edit
(notesAppend)`, `update` → `backlog_task_edit` (field params). Output is
capped at `MAX_OUTPUT = 40_000` chars; `ready` results are clamped to
`limit` (default 20, max 100). The tool exposes all 9 Backlog.md statuses;
`start` is sugar for `status: "in-progress"`. v1 does not surface
`dep tree` / `blocked` / `link` / `query` or the milestone-management tools
(milestone is a pass-through field on `create`/`update` only).

### Consequences

- Good, because one parser (the MCP server) owns Backlog.md format knowledge;
  the extension is a thin action router.
- Good, because the tool name `ticket` and the 10-action surface mirror
  `pi-tk-tickets`, so agents that know that interface port over unchanged.
- Good, because `create`/`update` carry a `milestone` field, exposing a
  Backlog.md capability tk lacks, without growing the tool surface.
- Bad, because every action pays a process-boundary cost through the MCP
  server (a process spawn / stdio round-trip per action).
- Bad, because the extension does not work if the `backlog` MCP server is
  not running; there is no direct-file fallback in v1.
