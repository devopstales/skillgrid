---
name: issue-creation
description: >
  Create, triage, and publish issues to the repository's tracker — Jira, GitHub, GitLab, or Backlog.md.
  Trigger: when the user asks to create an issue/epic/task/ticket, report a bug, file a feature request,
  or map SDD tasks to tracker issues.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
---

# Issue Creation

Create and triage issues across **Jira, GitHub, GitLab, or Backlog.md**. Which one to use is repository policy, not this skill's choice — discover it first.

## Tracker Resolution

Read the project's `docs/agents/issue-tracker.md` (written by `sdd-init`). It names one of:

| Tracker | Reference | CLI |
|---|---|---|
| Jira | [`../_shared/issue-tracker/jira.md`](../_shared/issue-tracker/jira.md) | `jira` CLI |
| GitHub | [`../_shared/issue-tracker/github.md`](../_shared/issue-tracker/github.md) | `gh` CLI |
| GitLab | [`../_shared/issue-tracker/gitlab.md`](../_shared/issue-tracker/gitlab.md) | `glab` CLI |
| Backlog.md | [`../_shared/issue-tracker/backlogmd.md`](../_shared/issue-tracker/backlogmd.md) | `backlog` CLI |

Read the referenced file before publishing. It carries the tracker-specific CLI syntax, conventions, label vocabulary, and SDD→tracker mapping.

If `docs/agents/issue-tracker.md` doesn't exist, ask which tracker the project uses (never guess), and run `sdd-init` to bootstrap the conventions.

## Core Rule

Discover the repository's actual contribution workflow before proposing or publishing. Templates, labels, approval gates, and issue forms are **repository policy**, not universal behavior. A skill that invents labels or bypasses an approval gate is publishing noise.

## Discovery (per tracker)

Run read-only checks first, per the tracker's reference file. Common checks:

- **Auth**: `gh auth status` / `glab auth status` / `jira config` / `backlog status`.
- **Repo/instance**: resolve from `git remote -v` or the tracker's config.
- **Policy**: `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/` (GitHub), issue form / issue templates (GitLab), project config (Jira), `backlog.config.yml` (Backlog).
- **Labels**: only apply labels that exist and that repository policy permits the actor to use. Jira: `jira issue list` + `jira project` to discover. GitHub: `gh api "repos/$REPO/labels"`. GitLab: `glab api "projects/:id/labels"`. Backlog: `backlog.config.yml`.
- **Templates**: GitHub `ISSUE_TEMPLATE/*.md|yml`; GitLab issue templates; Jira `issue_type` defaults; Backlog `backlog.config.yml` templates.

Stop and ask if authentication fails, the repository/instance is unresolvable, or policy discovery fails. Never continue from failed discovery into publication.

## Workflow

1. **Classify the issue.** One sentence for what's being reported or requested. Decide: bug, feature, task, enhancement, or epic (Jira), or the tracker's issue type.
2. **Duplicate-search first.** Per tracker:
   - GitHub: `gh issue list --state all --search "$QUERY" --limit 1000`
   - GitLab: `glab issue list --state opened --search "$QUERY"`
   - Jira: `jira issue list -q 'project = PROJ AND summary ~ "<query>" AND resolution is empty'`
   - Backlog: `grep -ri "$QUERY" .backlog/tasks/`

   If something already covers this behavior, **comment** on the existing issue. Do not create a duplicate.
3. **Split multi-component work** into separate issues (one per component: API, UI, SDK, or whatever the project uses). API before UI (dependency). Express blocking via the tracker's native links (GitHub issue dependencies, GitLab `blocked_by`, Jira `Blocks`, or a `Blocked by:` line at the top of each issue body).
4. **Choose the right template** (or the project's template type) and fill it with evidence you already have. Missing facts → ask; never invent.
5. **Privacy review** the body before publishing (table below).
6. **Apply labels** only if the label exists and repository policy permits the actor to apply it.
7. **Publish** via the tracker's CLI (see the reference file).
8. **Record the issue key/number** back into any SDD `tasks.md` that this issue covers, and into the Mnemonic `sdd/<NNN-slug>/issue-creation` observation if the issue is part of a change.

## Work-Item Formatting

Every tracker has a per-tracker formatting reference next to its CLI file. It carries the initiative/epic template, the task/issue template, title conventions, priority guidelines, component-specific sections, and the multi-component split-by-component rule. Read the one for the resolved tracker:

| Tracker | Formatting reference |
|---|---|
| Jira | [`../_shared/issue-tracker/jira-formatting.md`](../_shared/issue-tracker/jira-formatting.md) — Epic + Task; custom field IDs (`{{TEAM_FIELD}}`, `{{DESCR_FIELD}}`, `{{PROJECT_KEY}}`) are placeholders the project's `issue-tracker.md` fills in |
| GitHub | [`../_shared/issue-tracker/github-formatting.md`](../_shared/issue-tracker/github-formatting.md) — Milestone/Tracking issue + Issue; blocking via native dependencies or `Blocked by: #N` |
| GitLab | [`../_shared/issue-tracker/gitlab-formatting.md`](../_shared/issue-tracker/gitlab-formatting.md) — Epic/Tracking issue + Issue; blocking via `blocked_by` link or `Blocked by: !N` |
| Backlog.md | [`../_shared/issue-tracker/backlogmd-formatting.md`](../_shared/issue-tracker/backlogmd-formatting.md) — Project/initiative file + Task file; blocking via frontmatter `Blocked by` / `Blocks` arrays |

Shared across all four: split multi-component work into **one item per component** (API before UI), express blocking explicitly, only use labels/fields that exist in the project, and match the project's existing title convention.

## Pre-submission Privacy Review

Mandatory for every tracker. Scan the body **immediately before publishing**. Replace environment-specific data with explicit placeholders — the reproduction must still teach what to fill in.

| Category | Replace with | Example (before → after) |
|---|---|---|
| Private project names | `<project-name>` | `my-private-project` → `<project-name>` |
| Usernames | `<user>` | `~/go/bin` where `~` resolves to a real user path → `/home/<user>/go/bin` |
| Hostnames | `<hostname>` | `devbox-macbook.local` → `<hostname>` |
| API keys, tokens, passwords | `<token>` / `<password>` | `ghp_abc123...` → `<token>` |
| Internal ports / IPs | `<host>:<port>` | `10.0.0.42:5432` → `<host>:<port>` |

Do NOT redact intentionally public identifiers: tool names, package names, public doc URLs, `example.com`, `localhost`.

**Rule of thumb:** if the reader can run the reproduction after the replacement, sanitization is correct. If a step becomes impossible because the placeholder consumed a needed value, mark it `<value-required>` and add a note to the body saying what to fill in.

## Labels and Approval

- Use only labels returned by discovery.
- Follow contribution guidance for who may apply each label.
- Wait when policy requires maintainer approval (GitHub: check `CONTRIBUTING.md` and existing issue templates; Jira: respect workflow transitions; GitLab: respect instance label permissions; Backlog: respect `backlog.config.yml`).
- Do not invent a status or priority taxonomy when none is documented.

## SDD Task → Issue Mapping

When the trigger is "create issues for this SDD change," use the tracker's `issue-creation mapping` section (in its `_shared/issue-tracker/*.md` file) as the binding rule. Common shape:

- One issue per `tasks.md` item under `docs/skillgrid/changes/<NNN-slug>/steps/*/`.
- Blocking relations: tracker-native link types where available; otherwise a `Blocked by: <id>` line at the top of the body.
- Record every issue key back into the step's `tasks.md` and the Mnemonic `sdd/<NNN-slug>/issue-creation` observation for traceability.

## Triage Decision

Before approving, closing, or re-labelling any issue:

- It describes a concrete bug or scoped improvement (not an unsupported question).
- It is not a duplicate (searched above).
- Report carries enough evidence for an implementation decision.
- Requested behavior is in scope.
- Labels and status changes follow the current repository's policy.

If any point is uncertain, keep the issue in the repository's current review state and request the smallest missing evidence.

## Common Rationalizations

| Excuse | Reality |
|---|---|
| "I'll just create a GitHub issue, it's the default" | The repository chose its tracker. Read `docs/agents/issue-tracker.md` first. |
| "Labels don't matter, I'll add a generic one" | Inventing a label is publishing noise. Use discovered labels. |
| "The Jira project key is probably the same as last time" | Different instances, different IDs. Read the project's `issue-tracker.md` or discover with `jira project`. |
| "This bug is small, no need for a privacy review" | The privacy review takes 10 seconds and prevents a leak. |
| "I'll combine API + UI into one task, it's simpler" | Multi-component tasks block both teams. Split per component; link with `Blocked by:`. |
| "Templates are ceremony" | Templates encode the project's required fields and approval gates. Skipping them = publishing an unapproved issue. |
| "The duplicate might be stale, I'll create a new one anyway" | Comment on the existing one with fresh evidence; if it's really stale, close the old and re-open with a reference. |
| "I'll publish and fix labels later" | Labels are discoverability. Apply them at publish time or not at all. |

## Red Flags

- Publishing before reading `docs/agents/issue-tracker.md`.
- Invoking `gh` / `glab` / `jira` / `backlog` without an auth check in the current context.
- Applying a label that was not returned by discovery.
- A body containing a username, private hostname, or token that isn't a placeholder.
- A "duplicate" search that returned 1000+ results and was not narrowed.
- One issue covering multiple components with no `Blocked by:` relation expressed.

## References

- [`../_shared/issue-tracker/jira.md`](../_shared/issue-tracker/jira.md) — Jira CLI conventions and SDD mapping.
- [`../_shared/issue-tracker/jira-formatting.md`](../_shared/issue-tracker/jira-formatting.md) — Jira epic + task templates, title conventions, priority, component split.
- [`../_shared/issue-tracker/github.md`](../_shared/issue-tracker/github.md) — GitHub CLI conventions and SDD mapping.
- [`../_shared/issue-tracker/github-formatting.md`](../_shared/issue-tracker/github-formatting.md) — GitHub milestone/issue templates, labels, blocking links.
- [`../_shared/issue-tracker/gitlab.md`](../_shared/issue-tracker/gitlab.md) — GitLab CLI conventions and SDD mapping.
- [`../_shared/issue-tracker/gitlab-formatting.md`](../_shared/issue-tracker/gitlab-formatting.md) — GitLab epic/issue templates, labels, `blocked_by` links.
- [`../_shared/issue-tracker/backlogmd.md`](../_shared/issue-tracker/backlogmd.md) — Backlog.md conventions and SDD mapping.
- [`../_shared/issue-tracker/backlogmd-formatting.md`](../_shared/issue-tracker/backlogmd-formatting.md) — Backlog.md project/task file templates, frontmatter schema, blocking arrays.
- [`../_shared/triage-labels.md`](../_shared/triage-labels.md) — shared triage role vocabulary.
