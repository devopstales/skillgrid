# GitHub: issue formatting (milestones and issues)

Shared formatting conventions for GitHub work. Consumed by the `issue-creation` skill when the resolved tracker is GitHub (see `github.md` for CLI conventions).

**GitHub has no native Epic type.** For large initiatives, use a **milestone** (via `gh api`) and label child issues with it, or a plain **tracking issue** with a checklist of child issues. Match whatever the repository's existing conventions use.

## Choosing Milestone vs Issue

- **Milestone** (or tracking issue) — large feature spanning multiple components, major initiative. Groups related issues.
- **Issue** (type via labels or an issue form) — a single deliverable that lands in one PR.
- **Bug** — separate siblings per component; urgent, no wrapper needed.

## Milestone / Tracking Issue Template

```markdown
# {Milestone Title}

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

## Child Issues

- [ ] #N — {child issue title} (`api`)
- [ ] #N — {child issue title} (`ui`)
- [ ] #N — {child issue title} (`api`, `ui`)

## Diagrams

{Mermaid: architecture, data flow, state, ER — as applicable}
```

### Milestone / tracking issue title

Match the project's existing style. Common patterns: `{Feature Name}`, `Epic: {Feature Name}`, `{Feature Name} (Q{N} {Year})`.

### Splitting milestone into issues

From "Findings View", derive:

| # | Issue title | Component label | Blocked by |
|---|---|---|---|
| 1 | Findings table with pagination | `api` | - |
| 2 | Findings filters - provider and account | `api` | #1 |
| 3 | Findings detail panel - Overview tab | `ui` | #1 |
| 4 | Findings bulk actions - mute/suppress | `api`, `ui` | #1, #2 |

Each child issue lists its milestone (`gh issue edit <n> --milestone "{name}"`) and its `Blocked by:`.

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

- Milestone: {name} (if tracked)
- Blocked by: #{n} (or "—")
- Blocks: #{n} (or "—")
```

## Issue title

Format: `[TYPE] Brief description (COMPONENT)` where `[TYPE]` ∈ `{BUG, FEATURE, ENHANCEMENT, REFACTOR, DOCS, CHORE}`. Examples:

- `[BUG] AWS GovCloud accounts cannot connect - STS region hardcoded (API + UI)`
- `[FEATURE] Add dark mode toggle (UI)`
- `[REFACTOR] Migrate E2E tests to Page Object Model (UI)`

Match the project's existing style if it differs (e.g. `fix:`, `feat:` prefixes).

## Labels

Apply both a **type** and a **component** label (when the repository has them):

- Type: `bug` / `feature` / `enhancement` / `good first issue`
- Component: `api` / `ui` / `sdk` / any project-specific component
- Triage: `needs-triage` / `needs-info` / `ready-for-agent` / `ready-for-human` / `wontfix` (see `_shared/triage-labels.md`)

Only use labels returned by `gh api "repos/$REPO/labels"` — never invent.

## Priorities (labels, no native field)

GitHub has no native priority field. Use labels where the project maintains them, or a `## Priority` line in the body (preferred when labels are sparse):

| Priority | Criteria |
|---|---|
| **Critical** | Production down, data loss, security vulnerability |
| **High** | Blocks users, no workaround, affects paid features |
| **Medium** | Has workaround, affects subset of users |
| **Low** | Nice to have, cosmetic, internal tooling |

## Multi-component work: split into multiple issues

When work touches multiple components (API, UI, SDK), create **one issue per component** — GitHub issues are the atomic unit. Express blocking via:

**Preferred:** GitHub's **native issue dependencies** — create via the web UI or by adding a `Blocked by: #N` reference in the body (the GitHub UI parses this into a dependency).

**Fallback:** `Blocked by: #N` at the top of the body.

**Bug** → sibling issues, no wrapper.
**Feature** → optional tracking issue + one issue per component.

## Component-specific sections

- **API issues**: serializer/endpoint changes, migration requirements, spec regeneration.
- **UI issues**: component paths, form validation, state management impact, responsive considerations.
- **SDK issues**: provider, service, check changes, config changes.

## Checklist before publishing

1. Title follows the project's `[TYPE] description (COMPONENT)` convention.
2. Description has Current/Expected State (issues) or Overview (milestones).
3. Acceptance Criteria are specific and testable.
4. Technical Notes include file paths.
5. Testing section covers happy path + edge cases.
6. Priority has a one-line justification.
7. Multi-component work is split into separate issues.
8. `Related Issues` lists milestone / blocked-by / blocks.

## Creating via `gh`

Worked example (placeholders):

```bash
# Milestone (if the project uses them)
gh api -X POST "repos/$REPO/milestones" -f title="Findings View"

# Issues under the milestone
gh issue create \
  --title "[FEATURE] Findings filters - provider and account (UI)" \
  --body-file issue-body.md \
  --label feature --label ui

# Set milestone + add dependency
gh issue edit <N> --milestone "Findings View"
```

If the repo uses **issue forms** (`.github/ISSUE_TEMPLATE/*.yml`), open the web chooser instead:

```bash
gh issue create --web
```

Do not parse or render the form schema — it's user-completed in the UI.

## Markdown rendering notes

GitHub renders full CommonMark + GitHub Flavored Markdown. Checkboxes in the body render as task lists and can be ticked in the UI — prefer `- [ ]` for acceptance criteria. Mermaid renders natively in issue bodies on `github.com`.
