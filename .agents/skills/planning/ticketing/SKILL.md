---
name: ticketing
description: Use after slicing to publish tickets to the configured tracker, track their status through execution, and close them when done. Reads tracker config from .skillgrid/config.yaml.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: skillgrid-v2:issue-creation + skillgrid-v2:_shared/issue-tracker + mattpocock-skills:triage + BMAD:bmad-sprint-planning
---

# Ticketing

Publish sliced tickets to the configured tracker, track their status through execution, and close them when done.

**Announce at start:** "I'm using the skillgrid:ticketing skill to publish and track tickets."

## Overview

Publishes sliced tickets to the configured tracker, drives their status through the Status Machine as execution waves progress, and closes them on a clean review. The tracker is the source of truth for status — without it, work has no recovery point and no audit trail.

## When to Use

- When a change needs to be tracked as work tickets.
- When creating, updating, or closing tickets in the configured tracker (Backlog.md, GitHub, GitLab, Jira).

**When NOT to use:** when ticketing is disabled in config (`ticketing.enabled: false`) — don't create tickets if the project doesn't track them.

## Config

Read `.skillgrid/config.yaml` before starting. This skill reads:

| Config key | Used for |
|------------|----------|
| `ticketing.enabled` | If `false`, skip this skill entirely |
| `ticketing.type` | Tracker type: `backlogmd` \| `gh` \| `glab` \| `jira` |
| `ticketing.project_key` | Jira project key (when `ticketing.type: jira`) |
| `ticketing.backlogmd.dir` | Backlog.md directory (default `.backlog/`) |
| `ticketing.backlogmd.task_prefix` | Backlog.md task ID prefix (default `task`) |
| `ticketing.backlogmd.zero_padded_ids` | Backlog.md task ID zero-padding (default `3`) |
| `ticketing.gh.labels` | GitHub triage label strings |
| `ticketing.glab.labels` | GitLab triage label strings |
| `conventions.specs_root` | Where `tasks.md` lives (input) |
| `testing.runner` | Test command for DoD seeding (Backlog.md) |

If `.skillgrid/config.yaml` doesn't exist, invoke `skillgrid:onboarding` first.

## Status Machine

```
backlog → ready → in-progress → review → done
```

| Status | Meaning | Set by |
|--------|---------|--------|
| `backlog` | Ticket created, not yet started | `ticketing` (at publish) |
| `ready` | All blockers done, can start next wave | `ticketing` (at wave start) |
| `in-progress` | An implementer is working on it | `subagent-execution` / `simple-execution` |
| `review` | Implementation done, awaiting review | `subagent-execution` / `simple-execution` |
| `done` | Review clean, merged | `requesting-code-review` (on clean review) |

**Epic** (the parent) transitions: `backlog` → `in-progress` (first ticket starts) → `done` (all tickets done).

## The Process

### Step 1: Read Input

1. Read `.skillgrid/config.yaml` for `ticketing.enabled` and `ticketing.type`. If `ticketing.enabled: false`, stop — skip this skill entirely. Execution reads from `tasks.md` directly.
2. Read `tasks.md` from `.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`
3. Parse: epic summary, tickets (title, scope, acceptance, files, size, blocks, blocked by), dependency graph, execution order (waves)

### Step 2: Publish

**2a. Discovery.** Run the [Discovery](#discovery) checks first. Stop and ask if auth fails, the repo/instance is unresolvable, or policy discovery fails. Never continue from failed discovery into publication.

**2b. Duplicate check.** Search the tracker for existing tickets with the same title (per the tracker reference). If found, skip creation and use the existing ID.

**2c. Build the body.** Use the matching [formatting reference](#work-item-formatting) for the tracker's body template (epic/tracking issue + per-ticket). Fill with evidence you already have. Missing facts → ask; never invent.

**2d. Privacy review.** Run the [Pre-submission Privacy Review](#pre-submission-privacy-review) on every body immediately before publishing.

**2e. Create the epic and tickets.**

**Epic** — one parent issue/ticket:
- Title: `Epic: <epic summary from tasks.md>`
- Body: epic summary + link to `tasks.md` + dependency graph
- Status: `backlog`

**Tickets** — one per ticket in tasks.md:
- Title: per the formatting reference's title convention (e.g. `TICKET-<NN>: <title>` or `[TYPE] description (COMPONENT)`)
- Body: per the formatting reference's issue template
- Blocked by: link to blocker tickets (native dependency if tracker supports it, `Blocked by:` text section otherwise)
- Status: `backlog`
- Linked to epic as child/sub-issue

**2f. Backlog.md completeness gate.** When the tracker is Backlog.md, satisfy the [completeness gate](#backlog-completeness-gate) before reporting published. Post-create: `backlog task view <ID> --plain` to verify.

**2g. Apply labels** only if the label exists and repository policy permits the actor to apply it.

### Step 3: Record IDs

Write the tracker IDs back into `tasks.md` under each ticket:

```markdown
### TICKET-01 — <title>
- **Tracker ID:** PROJ-123  (or TASK-001, or #42)
- **Scope:** ...
```

Commit the updated `tasks.md`.

### Step 4: Wave Lifecycle

As execution progresses (driven by `subagent-execution` or `simple-execution`), update ticket status per the [Ticket Lifecycle](#ticket-lifecycle) table. The canonical transitions:

**Wave start:**
- For each ticket in the wave whose blockers are all `done`: set status → `ready`
- If the epic is still `backlog`: set epic → `in-progress`

**Ticket picked up:**
- Set ticket status → `in-progress`

**Implementation complete, review starts:**
- Set ticket status → `review`

**Review clean, merged:**
- Set ticket status → `done`
- If all tickets are `done`: set epic → `done`

### Step 5: Report

After publishing, report:

**"N tickets published to <tracker> as epic <EPIC_ID>. M waves. Ticket IDs recorded in tasks.md. Execution can begin — `skillgrid:subagent-execution` or `skillgrid:simple-execution` will pick up tickets in wave order."**

## Tracker Adapters

Read `.skillgrid/config.yaml` `ticketing.type`. Before publishing, read the matching **CLI** reference and the matching **formatting** reference. They carry the tracker-specific CLI syntax, conventions, label vocabulary, body templates, and the ticket lifecycle.

| `ticketing.type` | CLI reference | Formatting reference |
|---|---|---|
| `backlogmd` | [references/backlogmd.md](references/backlogmd.md) | [references/backlogmd-formatting.md](references/backlogmd-formatting.md) |
| `gh` | [references/github.md](references/github.md) | [references/github-formatting.md](references/github-formatting.md) |
| `glab` | [references/gitlab.md](references/gitlab.md) | [references/gitlab-formatting.md](references/gitlab-formatting.md) |
| `jira` | [references/jira.md](references/jira.md) | [references/jira-formatting.md](references/jira-formatting.md) |

Triage role vocabulary: [references/triage-labels.md](references/triage-labels.md).

## Discovery

Run read-only checks first, per the tracker's CLI reference. Stop and ask if any check fails — never continue from failed discovery into publication.

- **Auth**: `gh auth status` / `glab auth status` / `jira config` / `backlog status`.
- **Repo/instance**: resolve from `git remote -v` or the tracker's config.
- **Policy**: `CONTRIBUTING.md`, `.github/ISSUE_TEMPLATE/` (GitHub), issue templates (GitLab), project config (Jira), `backlog.config.yml` (Backlog).
- **Labels**: only apply labels that exist and that repository policy permits. Jira: `jira issue list` + `jira project`. GitHub: `gh api "repos/$REPO/labels"`. GitLab: `glab api "projects/:id/labels"`. Backlog: `backlog.config.yml`.
- **Templates**: GitHub `ISSUE_TEMPLATE/*.md|yml`; GitLab issue templates; Jira `issue_type` defaults; Backlog `backlog.config.yml` templates.

## Backlog completeness gate

When the resolved tracker is Backlog.md, a ticket is **not published** until all four are present:

| Field | Frontmatter / body | Fail signal |
|---|---|---|
| **Type** | `type:` | blank / missing |
| **References** | `references:` (non-empty) | empty list |
| **Definition of Done** | `## Definition of Done` + `<!-- DOD:BEGIN -->` items | "No Definition of Done items defined" |
| **Implementation Plan** | `## Implementation Plan` + `<!-- SECTION:PLAN:BEGIN -->` | missing or empty |

Also require Description (Current/Expected), Acceptance Criteria, and `priority:`.

**SDD tickets:** seed References to `briefing.md` / `tasks.md` / `acceptance.feature`; seed Plan from the blueprint steps; seed DoD from project defaults + change-level DoD. Thin one-line description stubs are **forbidden**. After create, write the ID into `tasks.md` **Tracker ID**. Later phases (execution / review) **must** run the [Ticket Lifecycle](#ticket-lifecycle) — creation without progress/close-out is incomplete.

**CLI crash:** filesystem fallback under `.backlog/tasks/` is allowed only if it still satisfies this gate (see `references/backlogmd.md`).

Read `references/backlogmd.md` before every Backlog create.

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

## Work-Item Formatting

Every tracker has a per-tracker formatting reference next to its CLI file. It carries the initiative/epic template, the task/issue template, title conventions, priority guidelines, component-specific sections, and the multi-component split-by-component rule. Read the one for the resolved tracker (see the [Tracker Adapters](#tracker-adapters) table).

Shared across all four: split multi-component work into **one item per component** (API before UI), express blocking explicitly, only use labels/fields that exist in the project, and match the project's existing title convention.

## Ticket Lifecycle

Creation alone is not enough. When `tasks.md` has a real tracker ID (from Step 3), every later execution phase owns a tracker update. The per-tracker CLI commands live in the matching CLI reference; the canonical transitions are:

| Phase | Status | Per-tracker action (see CLI reference) |
|---|---|---|
| **Execution (start)** | `backlog` → `in-progress` | Set status in-progress; comment that apply began |
| **Execution (per commit)** | `in-progress` | Commit footer `Refs: <ID>` |
| **Review** | `in-progress` → `review` | Comment review verdict |
| **Done** | `review` → `done` | Set status done; closing commit uses `Closes <ID>` |

If no tracker ID exists (e.g. `ticketing.enabled: false`), skip all tracker mutations. Do not invent an id.

## Slicing Strategy

| Tracker | Strategy |
|---------|----------|
| Backlog.md | One task per ticket, parented to epic task |
| GitHub | One issue per ticket, sub-issue of epic issue |
| GitLab | One issue per ticket, `/blocked_by` links |
| Jira | Domain = epic, ticket = story, acceptance criteria = sub-tasks |

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll skip the tracker and just use tasks.md" | tasks.md is the slice, not the tracker. The tracker gives you status, visibility, and a recovery point. Publish. |
| "The duplicate check is overkill" | Duplicate tickets confuse the execution loop. Search first. |
| "I'll update the status later in a batch" | Batch status updates lose the audit trail. Update at each transition. |
| "The epic is just a container, I don't need to track its status" | The epic's status is the project's status. Track it. |
| "I'll just create a GitHub issue, it's the default" | The repository chose its tracker. Read `.skillgrid/config.yaml` `ticketing.type` first. |
| "Labels don't matter, I'll add a generic one" | Inventing a label is publishing noise. Use discovered labels. |
| "This bug is small, no need for a privacy review" | The privacy review takes 10 seconds and prevents a leak. |
| "I'll combine API + UI into one task, it's simpler" | Multi-component tasks block both teams. Split per component; link with `Blocked by:`. |
| "Type is in the title as [FEATURE]" | Title tag ≠ frontmatter `type:`. Set `--type` / `type:` every time. |
| "DoD defaults will fill themselves" | Only if `definition_of_done` is configured. Verify the `view --plain` output. |
| "Implementation Plan comes later on pickup" | Seed a plan at create (blueprint steps or research→implement→verify). Deepen on pickup; never leave the section empty. |
| "CLI crashed, so a thin stub is fine" | Filesystem fallback must still include type, references, DoD, and plan. |
| "I'll publish and fix labels later" | Labels are discoverability. Apply them at publish time or not at all. |

## CLI-Only Guard

**Do not edit tracker task files directly.** Use the tracker's CLI (`backlog`, `gh`, `glab`, `jira`) so metadata, relationships, and history stay consistent. Hand-editing a Backlog.md task file, a GitHub issue's body, or a Jira ticket's fields breaks the audit trail and the status machine.

The only exception: writing the **Tracker ID** back into `tasks.md` (Step 3) — that is a local file, not the tracker.

## Red Flags

**Never:**
- Create a ticket without checking for duplicates first
- Skip recording the tracker ID back into tasks.md
- Update a ticket's status without the triggering event
- Hand-edit Backlog.md task files (use `backlog` CLI)
- Hand-edit any tracker's issue/ticket body or fields (use the CLI)
- Invent a tracker ID — always read it from the create response
- Leave the epic in `backlog` after the first ticket starts
- Publishing before reading the tracker's CLI + formatting reference
- Invoking `gh` / `glab` / `jira` / `backlog` without an auth check in the current context
- Applying a label that was not returned by discovery
- A body containing a username, private hostname, or token that isn't a placeholder
- A "duplicate" search that returned 1000+ results and was not narrowed
- One issue covering multiple components with no `Blocked by:` relation expressed
- Backlog task missing `type:`, empty `references:`, empty DoD, or empty Implementation Plan
- Ending the turn after publishing with a description-only stub

## Verification

- [ ] Tickets were created in the tracker and their IDs are recorded in `tasks.md` under each ticket's `Tracker ID`
- [ ] The Status Machine transitions are valid (no skipped statuses: `backlog` → `ready` → `in-progress` → `review` → `done`)
- [ ] The Pre-submission Privacy Review passed (no username, hostname, token, or private value in a ticket body)
- [ ] Ticket bodies follow the tracker's Work-Item Formatting (title convention, component split, explicit `Blocked by:`)
- [ ] When the tracker is Backlog.md, the Backlog completeness gate passed (`backlog task view <ID> --plain` shows type, references, DoD, and plan)
