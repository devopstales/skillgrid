# Triage Labels

The five canonical triage roles. Each role maps to a label/status string in the active tracker; this repo uses **Backlog.md** (`backlog` CLI).

| role | Backlog.md status / label |
|---|---|
| needs-triage | `Status: needs-triage` (default) |
| needs-info | `Status: needs-info` |
| ready-for-agent | `Status: ready-for-agent` |
| ready-for-human | `Status: ready-for-human` |
| wontfix | `Status: wontfix` |

## Rules

- Keep labels in sync across `backlog.config.yml`, this doc, and `_shared/triage-labels.md`.
- Triage state for Backlog.md is a `Status:` frontmatter field near the top of each ticket file.
- If the user's tracker already uses different label names, record the override per-role in `docs/agents/issue-tracker.md` — do not rename the roles.
