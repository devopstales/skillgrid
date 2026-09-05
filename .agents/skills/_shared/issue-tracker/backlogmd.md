# Issue tracker: Backlog.md

Issues and specs for this repo live as markdown files managed by the `backlog` CLI. Storage: `.backlog/tasks/<ID>.md` (configured via `backlog_directory: .backlog` in `backlog.config.yml`).

**Formatting** (project/initiative and task file templates, frontmatter schema, required fields, `Blocked by` / `Blocks` arrays, component split, file placement rules): see [backlogmd-formatting.md](backlogmd-formatting.md).

## Conventions

- One file per ticket: `.backlog/tasks/<ID>.md`, ID assigned by the backlog CLI.
- Triage state is a `Status:` line near the top of each file; labels live in `backlog.config.yml` / `.backlog/config.yml`.
- Triage roles: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix` — keep in sync with `_shared/triage-labels.md`.
- Comments and conversation append to the bottom under a `## Comments` heading.
- Completed work moves to `.backlog/completed/`, archive material to `.backlog/archive/`.

## Required fields (non-negotiable)

Every new or edited Backlog task **MUST** include all of the following before you report the ticket as published. Incomplete tickets are defects — fix them in the same turn.

| Field | Where | How (CLI preferred) |
|---|---|---|
| **Type** | Frontmatter `type:` | `--type feature` (or bug / enhancement / refactor / docs / chore — must be in project `types:`) |
| **References** | Frontmatter `references:` (+ optional `documentation:`) | `--ref <path-or-url>` (repeatable); always include the SDD `change.md` / `tasks.md` / `acceptance.feature` when the ticket maps to a change |
| **Definition of Done** | Body `## Definition of Done` with `<!-- DOD:BEGIN -->` … `<!-- DOD:END -->` | Rely on project `definition_of_done` defaults **and** add task-specific `--dod` items; never leave "No Definition of Done items defined" |
| **Implementation Plan** | Body `## Implementation Plan` with `<!-- SECTION:PLAN:BEGIN -->` … `<!-- SECTION:PLAN:END -->` | `--plan '…'`. For SDD tickets: seed numbered steps from the change Step Blueprint / `tasks.md` (no speculative redesign). For non-SDD: seed research → implement → verify steps; deepen on pickup if needed |

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
  --dod 'Change-level Definition of Done in change.md fully checked' \
  --ref docs/skillgrid/changes/<NNN-slug>/change.md \
  --ref docs/skillgrid/changes/<NNN-slug>/tasks.md \
  --doc docs/skillgrid/changes/<NNN-slug>/change.md \
  --plan $'1. Follow change.md Step Blueprint\n2. Drive from tasks.md + acceptance.feature\n3. Verify go test ./... on touched packages'
```

Prefer CLI over hand-editing. Configure project defaults once in `.backlog/config.yml`:

```yaml
types: ["feature", "bug", "enhancement", "refactor", "docs", "chore"]
definition_of_done:
  - Tests pass (`go test ./...` for touched packages)
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

## issue-creation mapping

The `issue-creation` skill maps each step's `tasks.md` items (`docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md`) → one backlog ticket per task, with dependency notes referencing sibling task IDs.

For `force_ticket_creation` plan/acceptance tickets (propose/spec), still fill Type, References (to the change artifacts), DoD, and an Implementation Plan seeded from the Step Blueprint — not a one-line description stub.

## force_ticket_creation

When `force_ticket_creation` is `true`, the `issue-creation` skill MUST be invoked to create the ticket for the `change.md` and `tasks.md` artifacts at the `sdd-propose` and `sdd-spec` phases. Required-field gate still applies.
