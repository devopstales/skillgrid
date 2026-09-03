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

Location: `.backlog/tasks/<ID>.md` (ID assigned by the `backlog` CLI).

```markdown
---
Title: [TYPE] {Brief description} ({COMPONENT})
Status: needs-triage
Labels: [feature, ui]
Blocked by: []
Blocks: []
Related: [TASK-M]
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

- [ ] {Specific, testable requirement}
- [ ] {Another requirement}

## Technical Notes

- Affected files:
  - `{file path 1}`
  - `{file path 2}`
- {Implementation hints}

## Testing

- [ ] {Test case 1}
- [ ] {Test case 2}

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

Match the project's existing style if it differs.

### Frontmatter fields

| Field | Notes |
|---|---|
| `Title` | `[TYPE] description (COMPONENT)` |
| `Status` | one of: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`, `in-progress`, `done`, `blocked` (see `_shared/triage-labels.md`) |
| `Labels` | YAML array; project-defined + component names |
| `Blocked by` | YAML array of task IDs |
| `Blocks` | YAML array of task IDs |
| `Related` | YAML array of task IDs (loose links) |

## Labels

Apply both a **type** and a **component** label (when the project's `backlog.config.yml` defines them):

- Type: `bug` / `feature` / `enhancement`
- Component: `api` / `ui` / `sdk` / any project-specific component
- Triage: the five canonical roles (see `_shared/triage-labels.md`)

Only use labels declared in `backlog.config.yml`.

## Priorities (no native field)

Backlog.md has no native priority field. Put a `## Priority` section in the body (preferred) — one of {Critical, High, Medium, Low} plus a one-line justification:

| Priority | Criteria |
|---|---|
| **Critical** | Production down, data loss, security vulnerability |
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

1. Title follows the project's `[TYPE] description (COMPONENT)` convention.
2. Frontmatter is valid YAML with `Title`, `Status`, `Labels` at minimum.
3. Description has Current/Expected State (tasks) or Overview (projects).
4. Acceptance Criteria are specific and testable.
5. Technical Notes include file paths.
6. Testing section covers happy path + edge cases.
7. Priority has a one-line justification.
8. Multi-component work is split into separate task files.
9. `Blocked by` / `Blocks` / `Related` frontmatter lists sibling IDs.

## Creating via `backlog` CLI

Worked example (placeholders):

```bash
# Create the task — ID is assigned and printed
backlog new --title "[FEATURE] Findings filters - provider and account (UI)" \
  --frontmatter-file task-frontmatter.yml \
  --body-file task-body.md

# Add blocking relation
backlog add-blocked-by <NEW_ID> TASK-3

# Discover sibling IDs
backlog list --status ready-for-agent
```

If the project has templates in `backlog.config.yml`, pass `--template <name>`.

## File placement rules

- Active tasks: `.backlog/tasks/<ID>.md`
- Completed: move (or copy) to `.backlog/completed/<ID>.md`
- Archived: `.backlog/archive/<ID>.md`

Do not hand-edit files outside these directories — the CLI tracks IDs and moves.

## Markdown rendering notes

Files are raw markdown. When rendered (VS Code preview, GitHub Pages of the backlog, etc.):

- Frontmatter (YAML) is metadata, not rendered.
- `- [ ]` checkboxes render as clickable task lists in supporting renderers.
- Mermaid blocks render with the `mermaid` code fence: ```` ```mermaid ````.
