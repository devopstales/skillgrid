# Issue tracker: GitLab

Issues and specs for this repo live as GitLab issues. Use the `glab` CLI (https://gitlab.com/gitlab-org/cli) for all operations.

## Conventions

- **Create an issue**: `glab issue create --title "..." --description "..."`.
- **Read an issue**: `glab issue view <number>` (add `--web` only when a browser view is genuinely needed).
- **List issues**: `glab issue list --state opened -F json` with label filters; pipe through `jq` to keep output small.
- **Comment**: `glab mr note <number> -m "..."` / issue equivalent `glab api projects/:id/issues/<n>/notes -X POST -f body="..."`.
- **Labels**: `glab issue update <number> --label "..."` / `--unlabel "..."`.
- **Close**: `glab issue close <number>` — comment first via the notes API if closure needs context.

Infer the repo from `git remote -v` (`gitlab.com/...` or self-hosted host).

Triage role labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — GitLab creates labels on first use; keep in sync with `_shared/triage-labels.md`.

## When a skill says "publish to the issue tracker"

Create a GitLab issue with `glab issue create`.

## When a skill says "fetch the relevant ticket"

`glab issue view <number>`. Duplicate-search first: `glab issue list --state opened --search "<keyword>"`.

## issue-creation mapping

The `issue-creation` skill maps `openspec/changes/<id>/tasks.md` items → one GitLab issue per task; blocking relations go via GitLab issue `blocked_by` links where the instance supports them, otherwise a `Blocked by: !<n>` line at the top of the body.
