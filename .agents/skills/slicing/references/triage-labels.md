# based on skillgrid-v2:docs/skillgrid/agents/triage-labels.md

# Triage Labels

The five canonical triage roles. Each role maps to a label/status string in the active tracker.

| role | Backlog.md | GitHub | GitLab | Jira |
|---|---|---|---|---|
| needs-triage | `Status: needs-triage` (default) | label `needs-triage` | label `needs-triage` | label `needs-triage` |
| needs-info | `Status: needs-info` | label `needs-info` | label `needs-info` | label `needs-info` |
| ready-for-agent | `Status: ready-for-agent` | label `ready-for-agent` | label `ready-for-agent` | label `ready-for-agent` |
| ready-for-human | `Status: ready-for-human` | label `ready-for-human` | label `ready-for-human` | label `ready-for-human` |
| wontfix | `Status: wontfix` | label `wontfix` | label `wontfix` | label `wontfix` |

## Rules

- Keep labels in sync across `backlog.config.yml`, the tracker's label config, and this doc.
- Triage state for Backlog.md is a `Status:` frontmatter field near the top of each ticket file.
- If the user's tracker already uses different label names, record the override per-role in `.skillgrid/config.yaml` — do not rename the roles.
