# Triage Labels

The five canonical triage roles. Each role maps to a label string in the active tracker; the tracker's `docs/skillgrid/agents/issue-tracker.md` (or `backlog.config.yml` for Backlog.md) records the actual strings.

| role | default label |
|---|---|
| needs-triage | `needs-triage` |
| needs-info | `needs-info` |
| ready-for-agent | `ready-for-agent` |
| ready-for-human | `ready-for-human` |
| wontfix | `wontfix` |

Rules:

- Keep labels in sync across `backlog.config.yml`, GitHub/GitLab issue labels, and any override mapping recorded during sdd-init.
- If the user's tracker already uses different label names (e.g. `bug:triage`), record the override per-role in the tracker doc — do not rename the roles.
- Triage state for local trackers (Backlog.md) is a `Status:` line near the top of each ticket file.
