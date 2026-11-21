# based on skillgrid-v2:_shared/issue-tracker/backlogmd-formatting.md

# Backlog.md: ticket formatting (milestones and tasks)

Shared formatting conventions for `backlog.md`-based workflows. Consumed by the `ticketing` skill when the resolved tracker is Backlog.md (see `backlogmd.md` (same directory) for CLI conventions).

**Backlog.md is file-based.** One ticket = one markdown file under `<backlog_directory>/tasks/<ID>.md`, where `<backlog_directory>` is resolved from the root-level `backlog.config.yml` field `backlog_directory` (default `.backlog/`; skillgrid convention `.skillgrid/projectwiki/backlog`). There's no server — the file *is* the tracker.

## Choosing Milestone vs Task

- **Milestone** (via `backlog milestone add`) — large feature spanning multiple components, major initiative. Epic equivalent, matching the GitHub milestone pattern.
- **Task** — a single deliverable that lands in one PR. One file per task.
- **Bug** — separate sibling task files per component; urgent, no wrapper needed.

**Standing rule (one milestone per SDD change):** Backlog.md has no real epics, so the milestone is the grouping mechanism — not a parent task. For every change: `backlog milestone add "<YYYY-MM-DD-topic>"` using the spec slug folder name verbatim, then assign the epic parent (if one was created, it stays as a pointer only) and every ticket via `backlog task edit <ID> -m <milestone-id>`. Never group a change's tickets by parent-task alone.

## Milestone Template

Create via `backlog milestone add` (epic equivalent, matching GitHub's milestone pattern). The milestone groups child tasks; each task file references its milestone in the body's `## Related Issues` section.

**Milestone name = spec folder name.** The milestone title must be the full spec folder name from `.skillgrid/change-specs/` (or `.skillgrid/change-archive/`), e.g. `2026-10-06-mnemonic-standalone`. Do not use short slugs like `mnemonic-standalone-module` or numeric IDs like `m-8`. This makes the `milestone → spec` mapping 1:1 and recoverable from the tracker alone.

Body scaffold: [backlogmd/templates/epic.md](backlogmd/templates/epic.md) — fill the placeholders and use it for the milestone body.

### Splitting milestone into tasks

From "Findings View", derive:

| # | Task title | Component | Blocked by |
|---|---|---|---|
| 1 | Findings table with pagination | api | - |
| 2 | Findings filters - provider and account | api | 1 |
| 3 | Findings detail panel - Overview tab | ui | 1 |
| 4 | Findings bulk actions - mute/suppress | api, ui | 1, 2 |

Every child task lists its milestone and blocking relations in its file.

## Task File Template

Location: `<backlog_directory>/tasks/<ID>.md` (ID assigned by the `backlog` CLI). Prefer `backlog task create` / `edit`; use this shape for verification and filesystem fallback.

Full file scaffold (frontmatter + body): [backlogmd/templates/ticket.md](backlogmd/templates/ticket.md) — fill the placeholders; the `type`, `references`, DoD, and Plan sections are non-negotiable (see the [checklist below](#checklist-before-publishing)).

### Task title

Format: `[TYPE] Brief description (COMPONENT)` where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. Examples:

- `[BUG] AWS GovCloud accounts cannot connect - STS region hardcoded (API + UI)`
- `[FEATURE] Add dark mode toggle (UI)`
- `[REFACTOR] Migrate E2E tests to Page Object Model (UI)`

**ASCII only.** The backlog CLI slugifies the title into a filename. Em dashes (`—`), en dashes (`–`), curly quotes, and other non-ASCII characters survive the slugify and produce unquotable filenames. Use ASCII hyphens (`-`) for separators, and avoid dots in file-relevant parts of the title (e.g. `SOUL.md` → `SOUL-md` in the title if it becomes part of the filename). The pre-commit hook enforces this.

Match the project's existing style if it differs. Title `[TYPE]` is **not** a substitute for frontmatter `type:` — both are required.

### Frontmatter fields

| Field | Required | Notes |
|---|---|---|
| `id` / `title` / `status` | yes | Canonical Backlog.md fields |
| `type` | **yes** | One of project `types:` (e.g. `feature`, `bug`). Never omit. |
| `references` | **yes** | Non-empty list of paths/URLs an agent needs. SDD tickets: always include `briefing.md` (+ `tasks.md` / `acceptance.feature` as relevant). |
| `documentation` | recommended | Design/spec docs for context |
| `priority` | yes | One of project `priorities:` |
| `labels` | when configured | YAML array; project-defined + component names |
| `dependencies` | when blocked | YAML array of task IDs (`TASK-N`) |

Legacy `Blocked by` / `Blocks` / `Related` arrays in older templates map to `dependencies` / comments — prefer CLI `--dep` so metadata stays consistent.

## Labels

Apply both a **type** and a **component** label (when the project's `backlog.config.yml` defines them):

- Type: `bug` / `feature` / `enhancement`
- Component: `api` / `ui` / `sdk` / any project-specific component
- Triage: the five canonical roles (see [triage-labels.md](../triage-labels.md))

Only use labels declared in `backlog.config.yml`.

## Priorities

Set frontmatter `priority:` from project `priorities:` (CLI `--priority`). Optionally keep a `## Priority` body section with a one-line justification:

| Priority | Criteria |
|---|---|
| **Critical** / **high** | Production down, data loss, security vulnerability |
| **High** | Blocks users, no workaround, affects paid features |
| **Medium** | Has workaround, affects subset of users |
| **Low** | Nice to have, cosmetic, internal tooling |

## Multi-component work: split into multiple task files

When work touches multiple components, create **one task file per component** — the file is the atomic unit. Express blocking via the frontmatter `Blocked by:` / `Blocks:` arrays.

**Bug** → sibling task files, no wrapper.
**Feature** → optional milestone (epic equivalent) + one task file per component.

## Component-specific sections

- **API tasks**: serializer/endpoint changes, migration requirements, spec regeneration.
- **UI tasks**: component paths, form validation, state management impact, responsive considerations.
- **SDK tasks**: provider, service, check changes, config changes.

## Checklist before publishing

**Hard fail** (do not report published until fixed):

1. Frontmatter `type:` set to a configured type.
2. Frontmatter `references:` non-empty (SDD → change artifacts).
3. `## Definition of Done` with `<!-- DOD:BEGIN -->` items (not empty / "No Definition of Done").
4. `## Implementation Plan` with `<!-- SECTION:PLAN:BEGIN -->` non-empty steps.

**Also required:**

5. Title follows `[TYPE] description (COMPONENT)`.
6. Description has Current/Expected State.
7. Acceptance Criteria specific and testable (`<!-- AC:BEGIN -->`).
8. `priority:` set; optional body justification.
9. Multi-component work split into separate task files with `dependencies:`.
10. Post-create: `backlog task view <ID> --plain` (or file read) confirms 1–4.

## Creating via `backlog` CLI

```bash
backlog task create '[FEATURE] Findings filters - provider and account (UI)' \
  --type feature \
  --priority medium \
  -d $'Current State:\n- …\n\nExpected State:\n- …' \
  --ac 'Filters apply to provider and account' \
  --dod 'Tests pass' \
  --ref .skillgrid/change-specs/YYYY-MM-DD-<topic>/briefing.md \
  --plan $'1. Research filter API\n2. Implement\n3. Verify tests'

backlog task edit TASK-N --dep TASK-M   # blocking
backlog task view TASK-N --plain        # verify required fields
```

If the project has templates in config, pass `--template <name>`. On CLI crash, write the **complete** template above under `<backlog_directory>/tasks/` (filesystem fallback) — still with type / references / DoD / plan.

## File placement rules

Active tasks live at `<backlog_directory>/tasks/<ID>.md`; on completion move (or copy) to `<backlog_directory>/completed/<ID>.md`; archive at `<backlog_directory>/archive/<ID>.md`. Prefer the CLI so IDs and metadata stay consistent. Filesystem fallback is allowed only when the CLI is broken, and must still satisfy the required-field gate.

## Markdown rendering notes

Files are raw markdown. When rendered (VS Code preview, GitHub Pages): frontmatter is metadata (not rendered), `- [ ]` checkboxes become clickable task lists in supporting renderers, and Mermaid blocks use the ` ```mermaid ` fence.
