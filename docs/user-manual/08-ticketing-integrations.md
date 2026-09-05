# Ticketing integrations

Skillgrid maps SDD work to an issue tracker. Default: **Backlog.md**. Alternatives: GitHub, GitLab, Jira.

Tracker choice is recorded at init in `docs/skillgrid/config.yaml` and `docs/skillgrid/agents/issue-tracker.md`. Triage roles: `docs/skillgrid/agents/triage-labels.md`.

## Quick path

| Tracker | id | Signal |
|---------|-----|--------|
| Backlog.md | `backlogmd` | Default; local markdown + `backlog` CLI |
| GitHub | `gh` | `github.com` remote |
| GitLab | `glab` | GitLab remote |
| Jira | `jira` | Configured instance + project key |

Skill: **`issue-creation`** — create/triage tickets when forced or when mapping tasks.

Formatting seeds: `.agents/skills/_shared/issue-tracker/` (`backlogmd`, `github`, `gitlab`, `jira` + `*-formatting.md`).

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
| Change (`NNN-slug`) | Epic / parent issue / tracking ticket |
| Step / task in `tasks.md` | Story / issue / backlog task |
| Acceptance scenarios | Linked in description; verify against them |

### Strategy A — Change = issue, tasks = checklist

Good for small teams on GitHub/GitLab Issues: one issue per change; `tasks.md` items become checklist lines in the body.

### Strategy B — Domain = epic, change = story

Good for Jira (and larger GitHub Projects): long-lived domain epic; each change is a story; tasks become sub-tasks.

## Agent obligations

When a skill says “publish to the issue tracker”:

1. Search for duplicates first
2. Create via the tracker’s CLI/API (never invent IDs)
3. Link paths: `docs/skillgrid/changes/<NNN-slug>/…`
4. Respect `Blocked by` / depends edges from `tasks.md`

## References

- Mapping OpenSpec-style SDD to Jira: [rushis.com](https://www.rushis.com/mapping-openspec-to-jira-sdd-without-abandoning-your-backlog/)
- Related tooling ideas: [dossier](https://github.com/fselich/dossier)

## Next step

[Plugins](09-plugins.md)
