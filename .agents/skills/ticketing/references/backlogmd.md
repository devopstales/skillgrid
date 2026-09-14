# based on skillgrid-v2:_shared/issue-tracker/backlogmd.md

# Issue tracker: Backlog.md

Issues and specs for this repo live as markdown files managed by the `backlog` CLI. Storage: `.backlog/tasks/<ID>.md` (configured via `backlog_directory: .backlog` in `backlog.config.yml`).

**Formatting** (project/initiative and task file templates, frontmatter schema, required fields, `Blocked by` / `Blocks` arrays, component split, file placement rules): see [backlogmd-formatting.md](backlogmd-formatting.md).

## Conventions

- One file per ticket: `.backlog/tasks/<ID>.md`, ID assigned by the backlog CLI.
- Triage state is a `Status:` line near the top of each file; labels live in `backlog.config.yml` / `.backlog/config.yml`.
- Triage roles: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix` — keep in sync with [triage-labels.md](triage-labels.md).
- Comments and conversation append to the bottom under a `## Comments` heading.
- Completed work moves to `.backlog/completed/`, archive material to `.backlog/archive/`.

## Required fields (non-negotiable)

Every new or edited Backlog task **MUST** include all of the following before you report the ticket as published. Incomplete tickets are defects — fix them in the same turn.

| Field | Where | How (CLI preferred) |
|---|---|---|
| **Type** | Frontmatter `type:` | `--type feature` (or bug / enhancement / refactor / docs / chore — must be in project `types:`) |
| **References** | Frontmatter `references:` (+ optional `documentation:`) | `--ref <path-or-url>` (repeatable); always include the SDD `briefing.md` / `tasks.md` / `acceptance.feature` when the ticket maps to a change |
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

Prefer CLI over hand-editing. Configure project defaults once in `.backlog/config.yml`:

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

If `backlog` SIGILL/segfaults (known Bun arch mismatch), write the task file under `.backlog/tasks/` matching [backlogmd-formatting.md](backlogmd-formatting.md) **complete** template — still with `type`, `references`, DoD markers, and Implementation Plan. Note the fallback in `## Comments`. Never use fallback as an excuse to omit required fields.

## When a skill says "publish to the issue tracker"

Create a ticket with the `backlog` CLI; the file lands in `.backlog/tasks/`. Run the **post-create verification** above before claiming done.

## When a skill says "fetch the relevant ticket"

Read `.backlog/tasks/<ID>.md`. Duplicate-search first: `grep -ri "<keyword>" .backlog/tasks/` before creating a new ticket.

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
