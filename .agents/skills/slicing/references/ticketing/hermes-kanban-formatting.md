# Hermes Kanban: task formatting (epic and ticket cards)

Shared formatting conventions for Hermes Kanban boards. Consumed by the
`slicing` skill when the resolved tracker is `hermes-kanban` (see
[hermes-kanban/hermes-kanban.md](hermes-kanban/hermes-kanban.md) for CLI
conventions).

**The board is the source of truth.** A local Backlog.md entry (if one exists
for the ticket) is a human-readable mirror: it carries the Kanban task ID as a
reference and its status flows *from* the board, never the other way around.

## Choosing Epic vs Task

- **Epic** — one top-level tracking card per SDD change. No code changes — it
  is a status rollup.
- **Task** — one Kanban task per ticket in `tasks.md`. The atomic unit.
- **Bug** — a sibling task under the same epic, no wrapper.

**Standing rule (one epic card per SDD change):** every ticket of a change is a
child of the change's epic card, and the epic card's title is derived from the
spec folder name (see below). A change's tickets are never grouped by
dependency edges alone.

## Epic Card

Title: `Epic: <epic summary from tasks.md>`.

- Workspace: `--workspace scratch` (no code changes — it is a tracking card).
- Assignee: the profile from `ticketing.kanban.assignee`.
- Parent: none (top-level card).
- Milestone mapping: the epic card stands in for the other trackers'
  milestone, created at ticketing publish time (this tracker has no milestone
  object, so there is nothing to create at Phase 0). The full spec folder name (e.g. `2026-10-06-mnemonic-standalone`)
  goes in the epic card's body as a `Spec: <spec-folder-name>` line so the
  `epic → spec` mapping stays 1:1 and recoverable from the board alone. Never
  use a short slug or numeric ID there.
- Status mirror: ready when the first wave starts, done when all tickets are
  done (see the lifecycle table in
  [hermes-kanban/hermes-kanban.md](hermes-kanban/hermes-kanban.md)).

## Task Title

Format: `TICKET-<NN>: <title>` (ASCII only).

- `<NN>` is the zero-padded ticket number from `tasks.md` (`01`, `02`, …).
- **ASCII only.** The Hermes worker toolchain slugifies task titles into
  workspace and log paths; em dashes, en dashes, curly quotes, and other
  non-ASCII characters produce unquotable paths. Use ASCII hyphens (`-`) for
  separators.

Match the project's existing board style if it differs (some boards use
`[TYPE] description (COMPONENT)` instead). The `TICKET-<NN>:` prefix is not a
substitute for the workspace flag — both are required.

## Task Body

Body scaffold: [hermes-kanban/templates/ticket.md](hermes-kanban/templates/ticket.md)
— fill the placeholders from the ticket's fields in `tasks.md`. Keep the body
self-documenting: the native `--parent` links already carry the dependency,
the `Blocked by:` line is redundant on purpose so the body stands alone.

Required sections (hard fail if missing):

1. **Scope** — one line: what this ticket builds.
2. **Acceptance** — the falsifiable criteria from `tasks.md` (the ticket's
   `SATISFIES` scenario named).
3. **Fails-when** — the output/exit-code that means the verify command failed.
4. **Blocked by** — the parent Kanban task IDs (or `none`).

Also required: the ticket's `SATISFIES` scenario name, and the component
section for multi-component work (below).

## Workspace

The workspace flag decides where the worker executes. Set it at create:

| Ticket type | Flag |
|---|---|
| Coding (implementation, refactoring, migration) | `--workspace worktree` |
| Research (investigation, documentation, no code changes) | `--workspace scratch` |

A coding ticket without `--workspace worktree` runs on a shared checkout and
races other workers. A research ticket in a worktree burns a branch it never
merges.

## Multi-component work: split into multiple tasks

When work touches multiple components (API, UI, SDK), create **one task per
component** — the task is the atomic unit. Express blocking via native
`--parent` links (and the body's `Blocked by:` line).

**Bug** → sibling tasks under the epic, no wrapper.
**Feature** → the epic card + one task per component.

## Component-specific sections

- **API tasks**: serializer/endpoint changes, migration requirements, spec regeneration.
- **UI tasks**: component paths, form validation, state management impact, responsive considerations.
- **SDK tasks**: provider, service, check changes, config changes.

## Checklist before publishing

1. Epic card created first (top-level, `--workspace scratch`, `Spec:` line).
2. Every task title is `TICKET-<NN>: <title>`, ASCII only.
3. Body has Scope, Acceptance, Fails-when, and `Blocked by:` (or `none`).
4. Workspace matches the ticket type (`worktree` / `scratch`).
5. Multi-component work split into separate tasks with native `--parent` links.
6. Published in wave order — parents before children — so `--parent` works at create time.
7. Post-create: `hermes kanban --board <slug> show <task-id>` confirms title,
   assignee, parent list, and workspace (mandatory — see the CLI reference).

## Backlog.md mirror (optional)

When a ticket was also published to Backlog.md (or a Backlog.md entry exists
for it): add the Kanban task ID to the entry's **References** (or a
`Kanban task:` line near the top), and mirror the board status into the local
entry at each transition. The board wins on any conflict.

When the resolved tracker is **only** Hermes Kanban, Backlog.md may hold no
entry at all — `tasks.md` then carries the task ID and is the local mirror.
