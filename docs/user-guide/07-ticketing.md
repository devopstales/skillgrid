# Ticketing

The **`ticketing`** skill maps SDD work to an issue tracker. Default: **Backlog.md**. Alternatives: GitHub, GitLab, Jira.

Tracker choice is recorded in `.skillgrid/config.yaml`. Formatting seeds and per-tracker references live under `.agents/skills/ticketing/references/` (`backlogmd`, `github`, `gitlab`, `jira` + `*-formatting.md` + `triage-labels.md`).

## Quick path

| Tracker | id | Signal |
|---------|-----|--------|
| Backlog.md | `backlogmd` | Default; local markdown + `backlog` CLI |
| GitHub | `gh` | `github.com` remote |
| GitLab | `glab` | GitLab remote |
| Jira | `jira` | Configured instance + project key |

The skill runs a status machine: `backlog → ready → in-progress → review → done`, plus a privacy review before publishing.

## Backlog.md (default)

```bash
backlog init "<name>" --integration-mode cli --backlog-dir .backlog ...
backlog task list --plain
backlog task view TASK-123 --plain
```

- Storage: `.backlog/tasks/<ID>.md`
- Do **not** edit task files by hand — use the `backlog` CLI
- Session start: `backlog instructions overview`

## Mapping SDD → tickets

| SDD artifact | Typical ticket mapping |
|--------------|------------------------|
| Blueprint (`blueprint.md`) | Epic / parent issue / tracking ticket |
| Slice in `tasks.md` | Story / issue / backlog task |
| Acceptance scenarios (`acceptance.feature`) | Linked in description; verify against them |

### Strategy A — Blueprint = issue, slices = checklist

Good for small teams on GitHub/GitLab Issues: one issue per change; `tasks.md` items become checklist lines in the body.

### Strategy B — Domain = epic, change = story

Good for Jira (and larger GitHub Projects): long-lived domain epic; each change is a story; slices become sub-tasks.

## Agent obligations

When the skill says "publish to the issue tracker":

1. Search for duplicates first
2. Create via the tracker's CLI/API (never invent IDs)
3. Link paths: `blueprint.md`, `tasks.md`, `acceptance.feature`
4. Respect `Blocked by` / depends edges from `tasks.md`

## Next step

[Concepts](08-concepts.md)
