# GitLab: issue formatting (epics and issues)

Shared formatting conventions for GitLab work. Consumed by the `issue-creation` skill when the resolved tracker is GitLab (see `gitlab.md` for CLI conventions).

**GitLab supports three work-item shapes.** Match what the project's existing conventions use:

- **Group/Project Epic** (paid feature on GitLab.com, self-hosted with premium EE enabled) — the canonical "large initiative" wrapper.
- **Board list** or **label-based grouping** (free/CE) — an epics-equivalent built from a label + board list.
- **Issues** in general — the atomic unit.

## Choosing Epic vs Issue

- **Epic** — large feature spanning multiple components. Only when the instance has Epics enabled and the project uses them.
- **Issue** — a single deliverable that lands in one MR.
- **Bug** — separate siblings per component; urgent, no wrapper needed.

## Epic / Tracking Issue Template

```markdown
# {Epic Title}

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

### Performance
- {requirements}

### Data Integration
- {sources, APIs, relationships}

### UI Components
- {reusable components, design system usage}

## Child Issues

- [ ] !N — {child issue title} (`api`)
- [ ] !N — {child issue title} (`ui`)
- [ ] !N — {child issue title} (`api`, `ui`)

## Diagrams

{Mermaid: architecture, data flow, state, ER — as applicable}
```

### Epic / tracking issue title

Match the project's existing style. Common patterns: `Epic: {Feature Name}` / `{Feature Name}`.

### Splitting epic into issues

From "Findings View", derive:

| # | Issue title | Component label | Blocked by |
|---|---|---|---|
| 1 | Findings table with pagination | `api` | - |
| 2 | Findings filters - provider and account | `api` | !1 |
| 3 | Findings detail panel - Overview tab | `ui` | !1 |
| 4 | Findings bulk actions - mute/suppress | `api`, `ui` | !1, !2 |

Each child issue lists its epic label and its `Blocked by:`.

## Issue Template

```markdown
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

## Related Issues

- Epic: !N (or "—")
- Blocked by: !N (or "—")
- Blocks: !N (or "—")
```

## Issue title

Format: `[TYPE] Brief description (COMPONENT)` where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. Examples:

- `[BUG] AWS GovCloud accounts cannot connect - STS region hardcoded (API + UI)`
- `[FEATURE] Add dark mode toggle (UI)`
- `[REFACTOR] Migrate E2E tests to Page Object Model (UI)`

Match the project's existing style if it differs (e.g. `bug:` / `feat:` prefixes).

## Labels

Apply both a **type** and a **component** label (when the repository has them):

- Type: `bug` / `feature` / `enhancement`
- Component: `api` / `ui` / `sdk` / any project-specific component
- Triage: `needs-triage` / `needs-info` / `ready-for-agent` / `ready-for-human` / `wontfix` (see `_shared/triage-labels.md`)

GitLab creates labels on first use. Discover existing labels with `glab api "projects/:id/labels"` before applying.

## Priorities (issues don't have a native field)

GitLab issues (unlike MRs) have no native priority field. Use labels where the project maintains them, or a `## Priority` line in the body (preferred when labels are sparse):

| Priority | Criteria |
|---|---|
| **Critical** | Production down, data loss, security vulnerability |
| **High** | Blocks users, no workaround, affects paid features |
| **Medium** | Has workaround, affects subset of users |
| **Low** | Nice to have, cosmetic, internal tooling |

## Multi-component work: split into multiple issues

When work touches multiple components, create **one issue per component**. Express blocking via:

**Preferred:** GitLab's **`blocked_by` link** (paid feature on GitLab.com, EE on self-hosted) — create via the web UI or by adding `Blocked by: !N` in the body (GitLab parses this into a dependency link).

**Fallback:** `Blocked by: !N` at the top of the body.

**Bug** → sibling issues, no wrapper.
**Feature** → optional epic/tracking issue + one issue per component.

## Component-specific sections

- **API issues**: serializer/endpoint changes, migration requirements, spec regeneration.
- **UI issues**: component paths, form validation, state management impact, responsive considerations.
- **SDK issues**: provider, service, check changes, config changes.

## Checklist before publishing

1. Title follows the project's `[TYPE] description (COMPONENT)` convention.
2. Description has Current/Expected State (issues) or Overview (epics).
3. Acceptance Criteria are specific and testable.
4. Technical Notes include file paths.
5. Testing section covers happy path + edge cases.
6. Priority has a one-line justification.
7. Multi-component work is split into separate issues.
8. `Related Issues` lists epic / blocked-by / blocks.

## Creating via `glab`

Worked example (placeholders):

```bash
# Tracking issue (epic equivalent)
glab issue create \
  --title "Epic: Findings View" \
  --description-file epic-body.md \
  --label epic --label feature

# Issues under it
glab issue create \
  --title "[FEATURE] Findings filters - provider and account (UI)" \
  --description-file issue-body.md \
  --label feature --label ui
```

Then link: edit via the web UI ("Related issues" panel) or via API (`glab api` to `projects/:id/issues/:iid/links`).

If the repo uses **issue templates** (`.gitlab/issue_templates/*.md`), pass `--template <name>`:

```bash
glab issue create --title "..." --description-file body.md --template bug
```

## Markdown rendering notes

GitLab renders full CommonMark + GFM. Checkboxes in the body render as task lists and can be ticked in the UI — prefer `- [ ]` for acceptance criteria. Mermaid renders natively in issue bodies on `gitlab.com` and most self-hosted instances.
