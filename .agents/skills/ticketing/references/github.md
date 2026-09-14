# based on skillgrid-v2:_shared/issue-tracker/github.md

# Issue tracker: GitHub

Issues and specs for this repo live as GitHub issues. Use the `gh` CLI for all operations.

**Formatting** (milestone/tracking-issue templates, title conventions, label conventions, multi-component split, blocking links): see [github-formatting.md](github-formatting.md).

## Conventions

- **Create an issue**: `gh issue create --title "..." --body "..."`. Use a heredoc for multi-line bodies.
- **Read an issue**: `gh issue view <number> --comments`.
- **List issues**: `gh issue list --state open --json number,title,body,labels,comments --jq '[.[] | {number, title, labels: [.labels[].name]}]'` with `--label` / `--state` filters.
- **Comment**: `gh issue comment <number> --body "..."`
- **Labels**: `gh issue edit <number> --add-label "..."` / `--remove-label "..."`
- **Close**: `gh issue close <number> --comment "..."`

Infer the repo from `git remote -v`; `gh` resolves this automatically inside a clone.

Triage role labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — keep in sync with [triage-labels.md](triage-labels.md).

## When a skill says "publish to the issue tracker"

Create a GitHub issue.

## When a skill says "fetch the relevant ticket"

`gh issue view <number> --comments`. Duplicate-search first: `gh issue list --state open --search "<keyword>"`.

## Ticketing mapping

The `ticketing` skill maps each ticket in `tasks.md` (`.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`) → one GitHub issue per ticket; use GitHub's native issue dependencies for blocking relations where available, otherwise a `Blocked by: #<n>` line at the top of the issue body.

## Ticket lifecycle (mandatory when a Tracker ID is set)

| Phase | Tracker action |
|---|---|
| **Execution** | Keep issue open; every step commit uses `Refs #<n>`; optional progress comment |
| **Review** | Comment review verdict |
| **Done** | Closing commit / PR uses `Closes #<n>` (or `gh issue close`); ticket close is part of archive, not optional |
