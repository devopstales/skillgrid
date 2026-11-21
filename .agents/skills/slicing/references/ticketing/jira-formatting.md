# based on skillgrid-v2:_shared/issue-tracker/jira-formatting.md

# Jira: work-item formatting (epics and tasks)

Shared formatting conventions for Jira epics and tasks. Consumed by the `ticketing` skill when the resolved tracker is Jira (see `jira.md` for CLI conventions).

**Field IDs are instance-specific.** The placeholders below (`{{TEAM_FIELD}}`, `{{DESCR_FIELD}}`, `{{PROJECT_KEY}}`) stand for the instance's custom fields and project key — read them from `.skillgrid/config.yaml` (`ticketing.project_key`) or discover them with `jira issue view <existing-key>`. Never hardcode IDs from another instance.

## Choosing Epic vs Task

- **Epic** — large feature spanning multiple components, major initiative, or new view/subsystem. Groups tasks under one issue (`-P` parent).
- **Task** (or Story/Sub-task) — a single deliverable that lands in one PR.
- **Bug** — separate siblings per component; urgent, no business-context wrapper needed.

## Epic Template

Body scaffold: [jira/templates/epic.md](jira/templates/epic.md) — fill the placeholders and use it for the epic body. **Jira renders Wiki markup** in the web UI, not Markdown — either pass `--body` with Wiki markup (conversion cheat sheet below) or let the instance auto-convert.

### Epic title

Format: `EPIC-{Project} {Feature Name}` or `Epic: {Feature Name}` — match whatever the project's existing epics use. One word of scope in the name works well.

### Required sections

1. **Feature Overview** — what / who / why.
2. **Requirements** — grouped by functional area, specific and testable.
3. **Technical Considerations** — performance, data integration, UI components (drop any that don't apply).
4. **Implementation Checklist** — high-level deliverables, ordered by dependency (API before UI); each checkbox = a candidate child task.

### Splitting epic into tasks

From epic "Findings View", derive:

| # | Task title | Component | Blocked by |
|---|---|-----------|------------|
| 1 | Findings table with pagination | API | - |
| 2 | Findings filters - provider and account | API | 1 |
| 3 | Findings detail panel - Overview tab | UI | 1 |
| 4 | Findings bulk actions - mute/suppress | API + UI | 1, 2 |

Every child task lists its epic and blocking relations in its body.

## Task Template

Body scaffold: [jira/templates/ticket.md](jira/templates/ticket.md) — fill the placeholders and use it for each task body (convert to Wiki markup for the web UI if the instance does not auto-convert).

### Task title

Format: `[TYPE] Brief description (COMPONENT)` where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}` and `(COMPONENT)` names the affected surface(s) — omit when the project doesn't tag components. Examples:

- `[BUG] AWS GovCloud accounts cannot connect - STS region hardcoded (API + UI)`
- `[FEATURE] Add dark mode toggle (UI)`
- `[REFACTOR] Migrate E2E tests to Page Object Model (UI)`

### Multi-component work: split into multiple tasks

When work touches multiple components (API, UI, SDK), create **one task per component**, not one big task:

- Different developers can work in parallel.
- Easier to review and test.
- API task must finish before UI (dependency) — express that with `Blocked by:`.

**Bug** → sibling tasks, no parent wrapper.
**Feature** → optional parent task (business context, user story, no technical detail) + one child per component.

### Priority guidelines

| Priority | Criteria |
|----------|----------|
| **Critical** | Production down, data loss, security vulnerability |
| **High** | Blocks users, no workaround, affects paid features |
| **Medium** | Has workaround, affects subset of users |
| **Low** | Nice to have, cosmetic, internal tooling |

## Component-specific sections

- **API tasks**: serializer/endpoint changes, migration requirements, spec regeneration.
- **UI tasks**: component paths, form validation, state management impact, responsive considerations.
- **SDK tasks**: provider, service, check changes, config changes.

## Checklist before publishing

1. Title follows the project's `[TYPE] description (COMPONENT)` convention.
2. Description has Current/Expected State (tasks) or Overview (epics).
3. Acceptance Criteria are specific and testable.
4. Technical Notes include file paths.
5. Testing section covers happy path + edge cases.
6. Priority has a one-line justification.
7. Multi-component work is split into separate tasks.
8. `Related Tasks` lists parent / blocked-by / blocks.

## Creating via `jira` CLI

Worked example (placeholders):

```bash
# Epic
jira issue create \
  -t Epic \
  -s "Findings View" \
  --template epic-body.md

# Child task under the epic, blocked by the API task
jira issue create \
  -t Task \
  -s "[FEATURE] Findings filters - provider and account (UI)" \
  -P FIND-123 \
  --template task-body.md

jira issue link FIND-124 --type Blocks FIND-140
```

If the instance uses a `Team` custom field or stores the body in a non-standard description field (some instances render a `Work Item Description` custom field instead of `description`), set it explicitly with `--cf "{{TEAM_FIELD}}"=UI` / `--cf "{{DESCR_FIELD}}"=...`.

**Jira renders Wiki markup** (`h2.`, `*bold*`, `* bullet`) not Markdown in the web UI. Either pass `--body` with Wiki markup, or let the instance auto-convert — verify with `jira issue view <key>` after creating.

## Markdown → Wiki conversion cheat sheet

| Markdown | Jira Wiki |
|----------|-----------|
| `## Heading` | `h2. Heading` |
| `**bold**` | `*bold*` |
| `- item` | `* item` |
| `  - subitem` | `** subitem` |
| `1. item` | `# item` |
| `\`code\`` | `{code}code{/code}` |
