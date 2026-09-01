# Issue tracker: GitHub

Issues and specs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, labels: [.labels[].name]}]'` with `--label` / `--state` filters.
- **Comment**: `gh issue comment <number> --body "..."`
- **Labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v`; `gh` resolves this automatically inside a clone.

Triage role labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — keep in sync with `_shared/triage-labels.md`.

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

`gh issue view <number> --comments`. Duplicate-search first: `gh issue list --state open --search "<keyword>"`.

## issue-creation mapping

The `issue-creation` skill maps `openspec/changes/<id>/tasks.md` items → one GitHub issue per task; use GitHub's native issue dependencies for blocking relations where available, otherwise a `Blocked by: #<n>` line at the top of the issue body.
