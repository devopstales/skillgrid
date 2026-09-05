# Issue tracker: Jira

Issues and specs for this repo live as Jira issues. Use the `jira` CLI ([jira-cli](https://github.com/ankitpokhrel/jira-cli)) for all operations. Referenced by issue key (`PROJ-123`), not number.

**Formatting** (epic vs task templates, title conventions, priority, multi-component split, Wiki-markup cheat sheet): see [jira-formatting.md](jira-formatting.md).

## Setup (first use in a project)

- `jira init` — interactive wizard: cloud vs server, auth type (`basic` email+API token, `bearer` PAT, `mtls`), server URL, default project and board. Stored at `~/.config/jira-cli/.config.yml`.
- Multiple Jira instances: use `--config <file>` or `JIRA_CONFIG_FILE`; verify the right one with `jira config`.
- Project key (e.g. `PROJ`) must be recorded in `docs/skillgrid/agents/issue-tracker.md` — skills never guess it.
- Auth order: `JIRA_API_TOKEN` env → `.netrc` → OS keyring.

## Conventions

- **Create an issue**: `jira issue create -t <Type> -s "summary" -b "body" [-l label] [-P PROJ-123]`. Use `--template` for a multi-line description from a file/stdin. Record the returned issue key.
- **Read an issue**: `jira issue view <KEY> --comments 10`.
- **List issues**: `jira issue list -q 'project = PROJ AND labels in (needs-triage, ready-for-agent)'` — prefer raw JQL via `-q --jql` over stacking filters; add `--plain` for compact output.
- **Comment**: `jira issue comment add <KEY> "body text"` — the body is a positional arg (or `--template FILE` / stdin with `--template -`); there is no `--body` flag. `--internal` for non-public notes.
- **Labels**: `jira issue edit <KEY> --label <name>` to add, `--label -<name>` to remove.
- **Close**: there is no `close` command — transition with `jira issue move <KEY> <Closed status> [--comment "..." -R <Resolution>]`. Check `jira issue move <KEY>` (no status) to list valid transition targets for the workflow.
- **Link**: `jira issue link <KEY> --type <linktype> <OTHER-KEY>` (e.g. blocking via an issue link type; `remote` for web links).

Triage role labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`) — keep in sync with `_shared/triage-labels.md`; Jira issues are searchable by label directly in JQL.

## When a skill says "publish to the issue tracker"

Create a Jira issue with `jira issue create` (type per the project's issue workflow — `Task`/`Story`/`Sub-task`; use `-P` for a sub-task under the change's parent issue).

## When a skill says "fetch the relevant ticket"

`jira issue view <KEY> --comments`. Duplicate-search first: `jira issue list -q 'project = PROJ AND summary ~ "<keyword>" AND resolution is empty'`.

## issue-creation mapping

The `issue-creation` skill maps each step's `tasks.md` items (`docs/skillgrid/changes/<NNN-slug>/steps/<NN-name>/tasks.md`) → one Jira issue per task, all under the change's parent Epic/Story (`-P <EPIC-KEY>`). Blocking relations: prefer a native issue link type (e.g. `Blocks`) where the instance defines one; otherwise a `Blocked by: <KEY>` line at the top of the issue body. Record every issue key back into the step's `tasks.md` (and the Mnemonic `sdd/<NNN-slug>/issue-creation` observation) for traceability.

## force_ticket_creation

When `force_ticket_creation` is `true`, the `issue-creation` skill MUST be invoked to create the ticket for the `change.md` and `tasks.md` artifacts at the `sdd-propose` and `sdd-spec` phases.
