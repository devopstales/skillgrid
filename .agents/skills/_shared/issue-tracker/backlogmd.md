# Issue tracker: Backlog.md

Issues and specs for this repo live as markdown files managed by the `backlog` CLI. Storage: `.backlog/tasks/<ID>.md` (configured via `backlog_directory: .backlog` in `backlog.config.yml`).

## Conventions

- One file per ticket: `.backlog/tasks/<ID>.md`, ID assigned by the backlog CLI.
- Triage state is a `Status:` line near the top of each file; labels live in `backlog.config.yml`.
- Triage roles: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix` — keep in sync with `_shared/triage-labels.md`.
- Comments and conversation append to the bottom under a `## Comments` heading.
- Completed work moves to `.backlog/completed/`, archive material to `.backlog/archive/`.

## When a skill says "publish to the issue tracker"

Create a ticket with the `backlog` CLI; the file lands in `.backlog/tasks/`.

## When a skill says "fetch the relevant ticket"

Read `.backlog/tasks/<ID>.md`. Duplicate-search first: `grep -ri "<keyword>" .backlog/tasks/` before creating a new ticket.

## issue-creation mapping

The `issue-creation` skill maps `openspec/changes/<id>/tasks.md` items → one backlog ticket per task, with dependency notes referencing sibling task IDs.
