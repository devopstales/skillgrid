# Backlog.md: ticket formatting (projects and tasks)

Shared formatting conventions for `backlog.md`-based workflows. Consumed by the `issue-creation` skill when the resolved tracker is Backlog.md (see `backlogmd.md` for CLI conventions).

**Backlog.md is file-based.** One ticket = one markdown file under `.backlog/tasks/<ID>.md`. There's no server — the file *is* the tracker.

## Choosing Project vs Task

- **Project** (top-level entry in `backlog.config.yml`, or a section file) — large feature spanning multiple components, major initiative.
- **Task** — a single deliverable that lands in one PR. One file per task.
- **Bug** — separate sibling task files per component; urgent, no wrapper needed.

## Project / Initiative File Template

If the project tracks initiatives, put the overview at:

- `.backlog/projects/<slug>.md`, or
- A top-level heading in `Backlog.md` with one bullet per task.

```markdown
# {Project Title}

**Figma:** {figma link if available}

## Feature Overview

{2-3 paragraphs: what it does, who uses it, why it's needed}

## Requirements

### {Section 1: Major Functionality Area}
- Requirement 1
- Requirement 2

### {Section 2: Another Major Area}
- Requirement 1
- Requirement 2

## Technical Considerations

- **Performance** — {requirements}
- **Data Integration** — {sources, APIs, relationships}
- **UI Components** — {reusable components, design system usage}

## Tasks

- [ ] `TASK-N` — {task title} (api)
- [ ] `TASK-N` — {task title} (ui)
- [ ] `TASK-N` — {task title} (api, ui)

## Diagrams

{Mermaid: architecture, data flow, state, ER — as applicable}
```

### Splitting project into tasks

From "Findings View", derive:

| # | Task title | Component | Blocked by |
|---|---|---|---|
| 1 | Findings table with pagination | api | - |
| 2 | Findings filters - provider and account | api | 1 |
| 3 | Findings detail panel - Overview tab | ui | 1 |
| 4 | Findings bulk actions - mute/suppress | api, ui | 1, 2 |

Every child task lists its project and blocking relations in its file.

## Task File Template

Location: `.backlog/tasks/<ID>.md` (ID assigned by the `backlog` CLI). Prefer `backlog task create` / `edit`; use this shape for verification and filesystem fallback.

```markdown
---
id: TASK-NNN
title: '[FEATURE] Brief description (COMPONENT)'
status: needs-triage
assignee: []
created_date: 'YYYY-MM-DD'
labels: []
dependencies: []
priority: medium
type: feature
references:
  - docs/skillgrid/changes/<NNN-slug>/change.md
  - path/or/url/related
documentation:
  - docs/skillgrid/changes/<NNN-slug>/change.md
---

## Description

{Brief explanation of the problem or feature.}

**Current State:**
- {What's happening now / what's broken}
- {Impact on users}

**Expected State:**
- {What should happen}
- {Desired behavior}

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 {Specific, testable requirement}
- [ ] #2 {Another requirement}
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 {Task-specific DoD item}
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. {Research / confirm deps}
2. {Follow change.md Step Blueprint or concrete implement steps}
3. {Verify with tests; mark AC/DoD}
<!-- SECTION:PLAN:END -->

## Technical Notes

- Affected files:
  - `{file path 1}`
  - `{file path 2}`
- {Implementation hints}

## Priority

{High/Medium/Low} — {justification}

## Comments

<!-- Conversation appends here. -->
```

### Task title

Format: `[TYPE] Brief description (COMPONENT)` where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. Examples:

- `[BUG] AWS GovCloud accounts cannot connect - STS region hardcoded (API + UI)`
- `[FEATURE] Add dark mode toggle (UI)`
- `[REFACTOR] Migrate E2E tests to Page Object Model (UI)`

Match the project's existing style if it differs. Title `[TYPE]` is **not** a substitute for frontmatter `type:` — both are required.

### Frontmatter fields

| Field | Required | Notes |
|---|---|---|
| `id` / `title` / `status` | yes | Canonical Backlog.md fields |
| `type` | **yes** | One of project `types:` (e.g. `feature`, `bug`). Never omit. |
| `references` | **yes** | Non-empty list of paths/URLs an agent needs. SDD tickets: always include `change.md` (+ `tasks.md` / `acceptance.feature` as relevant). |
| `documentation` | recommended | Design/spec docs for context |
| `priority` | yes | One of project `priorities:` |
| `labels` | when configured | YAML array; project-defined + component names |
| `dependencies` | when blocked | YAML array of task IDs (`TASK-N`) |

Legacy `Blocked by` / `Blocks` / `Related` arrays in older templates map to `dependencies` / comments — prefer CLI `--dep` so metadata stays consistent.

## Labels

Apply both a **type** and a **component** label (when the project's `backlog.config.yml` defines them):

- Type: `bug` / `feature` / `enhancement`
- Component: `api` / `ui` / `sdk` / any project-specific component
- Triage: the five canonical roles (see `_shared/triage-labels.md`)

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
**Feature** → optional project/initiative file + one task file per component.

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
  --ref docs/skillgrid/changes/<NNN-slug>/change.md \
  --plan $'1. Research filter API\n2. Implement\n3. Verify tests'

backlog task edit TASK-N --dep TASK-M   # blocking
backlog task view TASK-N --plain        # verify required fields
```

If the project has templates in config, pass `--template <name>`. On CLI crash, write the **complete** template above under `.backlog/tasks/` (filesystem fallback) — still with type / references / DoD / plan.

## File placement rules

- Active tasks: `.backlog/tasks/<ID>.md`
- Completed: move (or copy) to `.backlog/completed/<ID>.md`
- Archived: `.backlog/archive/<ID>.md`

Prefer the CLI so IDs and metadata stay consistent. Filesystem fallback is allowed only when the CLI is broken, and must still satisfy the required-field gate.

## Markdown rendering notes

Files are raw markdown. When rendered (VS Code preview, GitHub Pages of the backlog, etc.):

- Frontmatter (YAML) is metadata, not rendered.
- `- [ ]` checkboxes render as clickable task lists in supporting renderers.
- Mermaid blocks render with the `mermaid` code fence: ```` ```mermaid ````.
