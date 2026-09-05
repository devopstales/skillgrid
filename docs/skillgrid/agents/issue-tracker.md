# Issue tracker: Backlog.md

Issues and specs for this repo live as markdown files managed by the `backlog` CLI. Storage: `.backlog/tasks/<ID>.md` (configured via `backlog_directory: .backlog` in `backlog.config.yml`).

**Formatting** (task file templates, frontmatter schema, required fields, `Blocked by` / `Blocks` arrays, multi-component split, file placement rules): see [backlogmd-formatting.md](.agents/skills/_shared/issue-tracker/backlogmd-formatting.md).

## Conventions

- One file per ticket: `.backlog/tasks/<ID>.md`, ID assigned by the backlog CLI.
- Triage state is a `Status:` frontmatter field near the top of each file; labels configured in `backlog.config.yml` / `.backlog/config.yml`.
- Triage roles: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix` — keep in sync with `_shared/triage-labels.md`.
- Comments and conversation append to the bottom under a `## Comments` heading.
- Completed work moves to `.backlog/completed/`, archive material to `.backlog/archive/`.

## Required fields (non-negotiable)

Every Backlog task must include **Type**, **References**, **Definition of Done**, and **Implementation Plan** before publish. See `.agents/skills/_shared/issue-tracker/backlogmd.md` for the gate, CLI shape, post-create verification, and filesystem fallback when the CLI SIGILLs.

Project defaults live in `.backlog/config.yml` (`types:`, `definition_of_done:`).

## When a skill says "publish to the issue tracker"

Create a ticket with the `backlog` CLI; the file lands in `.backlog/tasks/`. Verify required fields with `backlog task view <ID> --plain` (or read the file) before claiming done.

## When a skill says "fetch the relevant ticket"

Read `.backlog/tasks/<ID>.md`. Duplicate-search first: `grep -ri "<keyword>" .backlog/tasks/` before creating a new ticket.

## issue-creation mapping

The `issue-creation` skill maps each step's `tasks.md` items (`docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md`) → one backlog ticket per task, with dependency notes referencing sibling task IDs. Required-field gate still applies for `force_ticket_creation` plan/acceptance tickets.
