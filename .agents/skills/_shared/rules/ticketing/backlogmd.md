# based on skillgrid-v2:_shared/issue-tracker/backlogmd.md

# Issue tracker: Backlog.md

Issues and specs for this repo live as markdown files managed by the `backlog` CLI. Storage: `.backlog/tasks/<ID>.md` (configured via `backlog_directory: .backlog` in `backlog.config.yml`).

**Formatting** (project/initiative and task file templates, frontmatter schema, required fields, `Blocked by` / `Blocks` arrays, component split, file placement rules): see [backlogmd-formatting.md](backlogmd-formatting.md).

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**At the beginning of each conversation in this project, run `backlog instructions overview` before answering or taking action. Re-read it only if you have not read it yet in the current conversation.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Before task lifecycle actions, read the matching detailed guide:

- `backlog instructions task-creation` before creating or splitting tasks
- `backlog instructions task-execution` before planning, changing status or assignee, adding a plan or implementation notes, or implementing task work
- `backlog instructions task-finalization` before checking acceptance criteria, writing final summaries, or moving tasks to terminal statuses

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent. The filesystem fallback below is only for a CLI crash.

The overview is the routing step, not the procedure. It says when to act; the detailed guides define how. Create a task when the work needs planning, decisions, or handoff notes — ask whether you need to think about how to do it. If yes, search for an existing task first, then create one. If no, make the small mechanical change directly. Skip task creation for questions, explanations, quick lookups, and obvious mechanical edits.

Search and read before changing anything:

```bash
backlog search "query" --plain
backlog task list --status "<todo status>" --plain
backlog task list --status "<active status>" --plain
backlog task view TASK-123 --plain
```

List windows (`--max-count`, `--skip`), `--json`, and `backlog task list --json --watch` are defined in `backlog instructions overview`. Re-read that overview for those shapes instead of copying them here.

## Conventions

- One file per ticket: `.backlog/tasks/<ID>.md`, ID assigned by the backlog CLI.
- Triage state is a `Status:` line near the top of each file; labels live in `backlog.config.yml` / `.backlog/config.yml`.
- Triage roles: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix` — keep in sync with [triage-labels.md](../../../planning/ticketing/references/triage-labels.md).
- Comments and conversation append to the bottom under a `## Comments` heading.
- Completed work moves to `.backlog/completed/`, archive material to `.backlog/archive/`.
- ADR copies live under `.backlog/decisions/decision-NNN - ADR-NNNN-slug.md`. The Skillgrid artifact is the record (ADR-0019); write the copy in the same edit as the artifact. See `skillgrid:architectural-decision-records` → Backlog.md copy.

## Required fields (non-negotiable)

Every new or edited Backlog task **MUST** include all of the following before you report the ticket as published. Incomplete tickets are defects — fix them in the same turn.

| Field | Where | How (CLI preferred) |
|---|---|---|
| **Type** | Frontmatter `type:` | `--type feature` (or bug / enhancement / refactor / docs / chore — must be in project `types:`) |
| **References** | Frontmatter `references:` (+ optional `documentation:`) | `--ref <path-or-url>` (repeatable); always include the SDD `briefing.md` / `blueprint.md` / `tasks.md` / `acceptance.feature` when the ticket maps to a change |
| **Definition of Done** | Body `## Definition of Done` with `<!-- DOD:BEGIN -->` … `<!-- DOD:END -->` | Rely on project `definition_of_done` defaults **and** add task-specific `--dod` items; never leave "No Definition of Done items defined" |
| **Implementation Plan** | Body `## Implementation Plan` with `<!-- SECTION:PLAN:BEGIN -->` … `<!-- SECTION:PLAN:END -->` | `--plan '…'`. For SDD tickets: seed numbered steps from the blueprint / `tasks.md` (no speculative redesign). For non-SDD: seed research → implement → verify steps; deepen on pickup if needed |

Also required (already common): clear **Description** (Current/Expected State), **Acceptance Criteria** (`<!-- AC:BEGIN -->`), **priority**.

### Post-create verification (mandatory)

After create or edit, run `backlog task view <ID> --plain` (or read the task file) and confirm:

1. `Type:` is set (not blank)
2. References list is non-empty
3. Definition of Done shows numbered checklist items (not "No Definition of Done items defined")
4. Implementation Plan section is present and non-empty

If any check fails → `backlog task edit` (or filesystem fallback below) **before** ending the turn. Do not leave thin stubs.

### CLI create shape (minimum)

```bash
backlog task create '[FEATURE] Brief description (mnemonic)' \
  --type feature \
  --priority medium \
  -d $'…Current/Expected State…' \
  --ac 'First criterion' \
  --ac 'Second criterion' \
  --dod 'Change-level Definition of Done in briefing.md fully checked' \
  --ref .skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md \
  --ref .skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md \
  --doc .skillgrid/specs/YYYY-MM-DD-<topic>/briefing.md \
  --plan $'1. Follow briefing.md blueprint\n2. Drive from tasks.md + acceptance.feature\n3. Verify tests on touched packages'
```

Prefer the CLI. Direct edits of task, draft, document, decision, or milestone files are not the normal path. Configure project defaults once in `.backlog/config.yml`:

```yaml
types: ["feature", "bug", "enhancement", "refactor", "docs", "chore"]
definition_of_done:
  - Tests pass
  - Lint and formatting pass
  - Edge cases covered
  - No new warnings introduced
  - Spec/docs updated if behavior changes
```

### Filesystem fallback (CLI crash / SIGILL)

This is the only exception to the no-direct-edit rule. If `backlog` SIGILL/segfaults (known Bun arch mismatch), write the task file under `.backlog/tasks/` matching [backlogmd-formatting.md](backlogmd-formatting.md) **complete** template — still with `type`, `references`, DoD markers, and Implementation Plan. Note the fallback in `## Comments`. Never use fallback as an excuse to omit required fields.

## When a skill says "publish to the issue tracker"

Create a ticket with the `backlog` CLI; the file lands in `.backlog/tasks/`. Run the **post-create verification** above before claiming done.

## When a skill says "fetch the relevant ticket"

Run `backlog task view <ID> --plain`. Duplicate-search first with `backlog search "<keyword>" --plain` before creating a new ticket. Reading `.backlog/tasks/<ID>.md` is only for understanding a task the CLI already wrote.

## Ticketing mapping

The `ticketing` skill maps each ticket in `tasks.md` (`.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`) → one backlog ticket per ticket, with dependency notes referencing sibling ticket IDs.

For plan/acceptance tickets (briefing/tasks), still fill Type, References (to the change artifacts), DoD, and an Implementation Plan seeded from the blueprint — not a one-line description stub.

## Ticket lifecycle (mandatory when a Tracker ID is set)

Creation alone is not enough. When `tasks.md` has a real tracker ID, every later execution phase owns a tracker update via the `backlog` CLI (filesystem fallback only on CLI crash):

| Phase | Tracker action |
|---|---|
| **Execution (start)** | `backlog task edit <ID> -s in-progress` (if not already); comment that execution began |
| **Execution (per commit)** | Commit footer **`Refs: <ID>`** |
| **Review** | Comment that review passed; leave status `in-progress` until done (or `ready-for-human` only if human QA is still open) |
| **Done** | `backlog task edit <ID> -s done` → `backlog task complete <ID>`; include ticket path changes in the **archive commit** |

If no Tracker ID exists, skip all tracker mutations. Do not invent an id.
