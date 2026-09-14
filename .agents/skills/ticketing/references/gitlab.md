# based on skillgrid-v2:_shared/issue-tracker/gitlab.md

# Issue tracker: GitLab

Issues and specs for this repo live as GitLab issues. Use the `glab` CLI (https://gitlab.com/gitlab-org/cli) for all operations.

**Formatting** (epic/tracking-issue templates, title conventions, label conventions, `blocked_by` links, multi-component split): see [gitlab-formatting.md](gitlab-formatting.md).

## Conventions

- **Create an issue**: `glab issue create --title "..." --description "..."`.
- **Read an issue**: `glab issue view <number>` (add `--web` only when a browser view is genuinely needed).
- **List issues**: `glab issue list --state opened -F json` with label filters; pipe through `jq` to keep output small.
- **Comment**: `glab issue note <number> -m "..."` (opens an editor when no `-m`); threaded replies via the notes API: `glab api projects/:id/issues/<n>/notes -X POST -f body="..."`.
- **Labels**: `glab issue update <number> --label "..."` / `--unlabel "..."`.
- **Close**: `glab issue close <number>` — comment first via the notes API if closure needs context.

Infer the repo from `git remote -v` (`gitlab.com/...` or self-hosted host).

Triage role labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — GitLab creates labels on first use; keep in sync with [triage-labels.md](triage-labels.md).

## When a skill says "publish to the issue tracker"

Create a GitLab issue with `glab issue create`.

## When a skill says "fetch the relevant ticket"

`glab issue view <number>`. Duplicate-search first: `glab issue list --state opened --search "<keyword>"`.

## Ticketing mapping

The `ticketing` skill maps each ticket in `tasks.md` (`.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`) → one GitLab issue per ticket; blocking relations go via GitLab issue `blocked_by` links where the instance supports them, otherwise a `Blocked by: !<n>` line at the top of the body.

## Ticket lifecycle (mandatory when a Tracker ID is set)

| Phase | Tracker action |
|---|---|
| **Execution** | Keep issue open; every step commit uses `Refs #<iid>`; optional progress comment |
| **Review** | Comment review verdict |
| **Done** | Closing commit / MR uses `Closes #<iid>` (or `glab issue close`); ticket close is part of archive, not optional |
