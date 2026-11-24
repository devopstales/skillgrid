# Slicing Part 2: Ticketing

## Part 2: Ticketing

Publish the sliced tickets to the configured tracker, record their IDs, and keep their status moving as execution waves progress. The tracker is the source of truth for status — without it, work has no recovery point and no audit trail.

### Status Machine

```
backlog → ready → in-progress → review → done
```

| Status | Meaning | Set by |
|--------|---------|--------|
| `backlog` | Ticket created, not yet started | `slicing` (at publish) |
| `ready` | All blockers done, can start next wave | `slicing` (at wave start) |
| `in-progress` | An implementer is working on it | `skillgrid:execute` (subagent or inline mode) |
| `review` | Implementation done, awaiting review | `skillgrid:execute` (subagent or inline mode) |
| `done` | Review clean, merged | `code-review` (on clean review) |

**Epic** (the parent) transitions: `backlog` → `in-progress` (first ticket starts) → `done` (all tickets done).

### Step 6: Read Input

1. Read `.skillgrid/config.yaml` for `ticketing.enabled` and `ticketing.type`. If `ticketing.enabled: false`, do not create tracker items; record a local-only waiver and let execution read from `tasks.md` directly.
2. Read `tasks.md` from `.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md`
3. Parse: epic summary, tickets (title, scope, acceptance, files, size, blocks, blocked by), dependency graph, execution order (waves)

### Step 7: Publish

**7a. Discovery.** Run the [Discovery](#discovery) checks first. Stop and ask if auth fails, the repo/instance is unresolvable, or policy discovery fails. Never continue from failed discovery into publication.

**7b. Duplicate check.** Search the tracker for existing tickets with the same title (per the tracker reference). If found, skip creation and use the existing ID.

**7c. Build the body.** Use the matching [formatting reference](#work-item-formatting) for the tracker's body template (epic/tracking issue + per-ticket). Fill with evidence you already have. Missing facts → ask; never invent.

**7d. Privacy review.** Run the [Pre-submission Privacy Review](#pre-submission-privacy-review) on every body immediately before publishing.

**7e. Create the epic and tickets.**

**Epic** — one parent issue/ticket:
- Title: `Epic: <epic summary from tasks.md>`
- Body: epic summary + link to `tasks.md` + dependency graph
- Status: `backlog`
- **Milestone name = spec folder name** (e.g. `2026-10-06-mnemonic-standalone`). The milestone must be named after the full spec folder under `.skillgrid/change-specs/` or `.skillgrid/change-archive/`, never a short slug or numeric ID. This makes the `milestone → spec` mapping 1:1 and recoverable from the tracker alone. It is created together with the change folder at Phase 0 (`_shared/rules/sdd-structure.md` § Writing Rules) — look it up and assign tickets to that ID; only create it if it is missing (e.g. ticketing was enabled mid-change). Never create a second milestone for the change.

**Tickets** — one per ticket in tasks.md:
- Title: per the formatting reference's title convention (e.g. `TICKET-<NN>: <title>` or `[TYPE] description (COMPONENT)`). **ASCII only** — the backlog CLI slugifies titles into filenames; em dashes and other non-ASCII characters produce unquotable filenames. Use ASCII hyphens for separators.
- Body: per the tracker's template in `references/ticketing/<tracker>/templates/ticket.md` (filled per the formatting reference)
- Blocked by: link to blocker tickets (native dependency if tracker supports it, `Blocked by:` text section otherwise)
- Status: `backlog`
- Linked to epic as child/sub-issue

**7f. Backlog.md completeness gate.** When the tracker is Backlog.md, satisfy the [completeness gate](#backlog-completeness-gate) before reporting published. Post-create: `backlog task view <ID> --plain` to verify.

**7g. Apply labels** only if the label exists and repository policy permits the actor to apply it.

### Step 8: Record IDs

Write the tracker IDs back into `tasks.md` under each ticket:

```markdown
### TICKET-01 — <title>
- **Tracker ID:** PROJ-123  (or TASK-001, or #42)
- **Scope:** ...
```

Commit the updated `tasks.md`.

### Step 9: Report

After publishing (or recording the local-only waiver), report:

**"Tasks sliced and committed to `.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md`. N tickets in M waves. [N tickets published to <tracker> as epic <EPIC_ID>.] Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per ticket, review between tickets, parallel worktrees within each wave

**2. Inline Execution** — Execute tickets in this session using `skillgrid:execute` (inline mode), wave by wave"

**Update `state.yaml`:** set `changes.<topic>.phase: slicing` and `changes.<topic>.gate: user-gate-slice` — the slice is not executed until the user confirms it (User gate contract in `_shared/rules/sdd-structure.md`; show the 30-second gate screen). `skillgrid:execute` clears the gate when it starts.

## Tracker Adapters

Read `.skillgrid/config.yaml` `ticketing.type`. Each tracker has one folder in `references/ticketing/`: the CLI standard, the formatting standard, and `templates/` (the body scaffolds to fill in). **Load only the folder for the resolved `ticketing.type`** — never the whole set.

| `ticketing.type` | Read (in this order) |
|---|---|
| `backlogmd` | [backlogmd.md](references/ticketing/backlogmd/backlogmd.md) → [backlogmd-formatting.md](references/ticketing/backlogmd-formatting.md) → `templates/` when building bodies |
| `gh` | [github.md](references/ticketing/github/github.md) → [github-formatting.md](references/ticketing/github-formatting.md) → `templates/` when building bodies |
| `glab` | [gitlab.md](references/ticketing/gitlab/gitlab.md) → [gitlab-formatting.md](references/ticketing/gitlab-formatting.md) → `templates/` when building bodies |
| `jira` | [jira.md](references/ticketing/jira/jira.md) → [jira-formatting.md](references/ticketing/jira-formatting.md) → `templates/` when building bodies |
| `hermes-kanban` | [hermes-kanban.md](references/ticketing/hermes-kanban/hermes-kanban.md) → [hermes-kanban-formatting.md](references/ticketing/hermes-kanban-formatting.md) → [templates/ticket.md](references/ticketing/hermes-kanban/templates/ticket.md) when building bodies |

For `hermes-kanban`, the **Kanban board is the source of truth** for status; a local Backlog.md entry (if any) is a human-readable mirror that carries the Kanban task ID and whose status is updated *from* the board. Read [hermes-kanban.md](references/ticketing/hermes-kanban/hermes-kanban.md) before every Kanban publish.

Triage role vocabulary: [references/triage-labels.md](references/triage-labels.md).

## Discovery

Run read-only checks first, per the tracker's CLI reference. Stop and ask if any check fails — never continue from failed discovery into publication.

- **Auth**: `gh auth status` / `glab auth status` / `jira config` / `backlog overview` / `hermes --version` (Hermes Kanban: the `hermes` binary on PATH; the board and assignee resolve from `ticketing.kanban.*`, and a missing board is created — creation failure is non-fatal, see [hermes-kanban.md](references/ticketing/hermes-kanban/hermes-kanban.md)).
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

**Decision gates (publishing):**

| Condition | Action |
|---|---|
| A discovery check fails (auth, repo/instance, policy, labels, templates) | STOP and ask — never continue from failed discovery into publication. |
| A Backlog.md ticket lacks Type / References / DoD / Implementation Plan | Not published — fix the field first (thin one-line description stubs are forbidden). |
| The body contains private data (project names, usernames, hostnames, ports, IPs) | Pre-submission privacy review — replace with placeholders before publishing. |
| `ticketing.enabled: false` or local-only waiver | Skip all tracker mutations — do not invent an ID. |
| `hermes-kanban` with a local Backlog.md mirror | The board is the source of truth; mirror status from the board at each transition — the board wins on conflict. |
| A ticket's status changes (wave start / execution / review / done) | The Ticket Lifecycle transitions apply — creation without progress/close-out is incomplete. |

**SDD tickets:** seed References to `briefing.md` / `tasks.md` / `acceptance.feature`; seed Plan from the blueprint steps; seed DoD from project defaults + change-level DoD. Thin one-line description stubs are **forbidden**. After create, write the ID into `tasks.md` **Tracker ID**. Later phases (execution / review) **must** run the [Ticket Lifecycle](#ticket-lifecycle) — creation without progress/close-out is incomplete.

**CLI crash:** filesystem fallback under `<backlog_directory>/tasks/` is allowed only if it still satisfies this gate (see [backlogmd.md](references/ticketing/backlogmd/backlogmd.md)).

Read [backlogmd.md](references/ticketing/backlogmd/backlogmd.md) before every Backlog create.

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

Shared across all trackers: split multi-component work into **one item per component** (API before UI), express blocking explicitly, only use labels/fields that exist in the project, and match the project's existing title convention.

## Slicing Strategy

| Tracker | Strategy |
|---------|----------|
| Backlog.md | One task per ticket, parented to milestone (epic equivalent) |
| GitHub | One issue per ticket, sub-issue of epic issue |
| GitLab | One issue per ticket, `/blocked_by` links |
| Jira | Domain = epic, ticket = story, acceptance criteria = sub-tasks |
| Hermes Kanban | One task per ticket on the configured board; epic = top-level task; dependencies via `--parents` / `link`; coding → `worktree` workspace, research → `scratch` |

## Ticket Lifecycle

Creation alone is not enough. When `tasks.md` has a real tracker ID (from Step 8), every later execution phase owns a tracker update. The per-tracker CLI commands live in the matching CLI reference; the canonical transitions are:

| Phase | Status | Per-tracker action (see CLI reference) |
|---|---|---|
| **Wave start** | `backlog` → `ready` | For each ticket in the wave whose blockers are all `done`: set status → `ready`; if the epic is still `backlog`: set epic → `in-progress` |
| **Execution (start)** | `backlog`/`ready` → `in-progress` | Set status in-progress; comment that apply began |
| **Execution (per commit)** | `in-progress` | Commit footer `Refs: <ID>` |
| **Review** | `in-progress` → `review` | Comment review verdict |
| **Done** | `review` → `done` | Set status done; closing commit uses `Closes <ID>`; if all tickets are `done`: set epic → `done` |

Hermes Kanban specifics (`hermes-kanban`): the board status is the truth — `unblock` at wave start, `claim` at execution start, `request-review` at review, `complete` at done (the exact commands live in [hermes-kanban.md](references/ticketing/hermes-kanban/hermes-kanban.md)). When a local Backlog.md entry exists for the ticket, mirror the board status into it at each transition — the board wins on any conflict.

If no tracker ID exists (e.g. `ticketing.enabled: false` or local-only waiver), skip all tracker mutations. Do not invent an id.

## CLI-Only Guard

**Do not edit tracker task files directly.** Use the tracker's CLI (`backlog`, `gh`, `glab`, `jira`, `hermes kanban`) so metadata, relationships, and history stay consistent. Hand-editing a Backlog.md task file, a GitHub issue's body, a Jira ticket's fields, or a Kanban task's fields breaks the audit trail and the status machine.

The only exception: writing the **Tracker ID** back into `tasks.md` (Step 8) — that is a local file, not the tracker.

## Anti-Patterns

| Anti-pattern | Symptom | Fix |
|---|---|---|
| In-head slicing | The slice exists only in the session; the executor gets a fresh context | If the slice isn't written down, it doesn't exist for them — write `tasks.md`. |
| "All related" oversized ticket | A ticket that needs the blueprint re-read mid-execution | "All related" is how 3000-line tickets get born — find the seam, split it. |
| Dependency graph skipped on 3 tickets | No graph, no waves | With 3 tickets you don't need this skill — use it at 5+. |
| Waves figured out during execution | Implicit ordering; the executor serializes everything | Waves must be explicit — parallelism is the whole point. |
| tasks.md as the tracker | No tracker ID, no status, no recovery point | tasks.md is the slice, not the tracker — publish the tickets. |
| Duplicate check skipped | A duplicate ticket in the execution loop | Search first — duplicates confuse the loop. |
| Batched status updates | Status jumps discovered at done-time | Batch updates lose the audit trail — update at each transition. |
| Untracked epic status | The epic sits in `backlog` while tickets run | The epic's status is the project's status — track it. |
| Default-GitHub assumption | Issues created in a tracker the repo didn't choose | The repository chose its tracker — read `.skillgrid/config.yaml` `ticketing.type` first. |
| Invented labels | A generic label that discovery never returned | Inventing a label is publishing noise — use discovered labels. |
| Privacy review skipped on small bugs | A username, hostname, or token in a published body | The review takes 10 seconds and prevents a leak. |
| Multi-component ticket | API + UI in one task | Multi-component tasks block both teams — split per component, link with `Blocked by:`. |
| Title-tag type | `[FEATURE]` in the title, no frontmatter | Title tag ≠ frontmatter `type:` — set `--type` / `type:` every time. |
| "DoD defaults will fill themselves" | Empty DoD in the created task | Only if `definition_of_done` is configured — verify the `view --plain` output. |
| Empty Implementation Plan | The plan section filled "later on pickup" | Seed a plan at create (blueprint steps or research→implement→verify); deepen on pickup; never leave it empty. |
| Thin stub after CLI crash | A fallback task missing type/references/DoD/plan | Filesystem fallback must still satisfy the completeness gate. |
| Labels "fixed later" | A ticket published without its labels | Labels are discoverability — apply them at publish time or not at all. |
| Hand-edited tracker files | A Backlog.md task file, issue body, or ticket field edited directly | The CLI (`backlog` / `gh` / `glab` / `jira` / `hermes kanban`) keeps metadata, relationships, and history consistent — hand-edits break the audit trail. |

## Red Flags

**Never:**
- Create a ticket that's a horizontal layer ("all DB schemas")
- Create a ticket too big for one context window
- Leave a ticket's acceptance criteria vague ("it works")
- Skip the dependency graph
- Put a ticket in a wave before its blockers
- Create a ticket without checking for duplicates first
- Skip recording the tracker ID back into tasks.md
- Update a ticket's status without the triggering event
- Hand-edit Backlog.md task files (use `backlog` CLI)
- Hand-edit any tracker's issue/ticket body or fields (use the CLI)
- Invent a tracker ID — always read it from the create response
- Leave the epic in `backlog` after the first ticket starts
- Publishing before reading the tracker's CLI + formatting reference
- Invoking `gh` / `glab` / `jira` / `backlog` / `hermes kanban` without an auth check in the current context
- Publishing a coding ticket without `--workspace worktree` (or a research ticket without `--workspace scratch`)
- Treating a Backlog.md mirror as the source of truth when the tracker is `hermes-kanban` (the board wins)
- Applying a label that was not returned by discovery
- A body containing a username, private hostname, or token that isn't a placeholder
- A "duplicate" search that returned 1000+ results and was not narrowed
- One issue covering multiple components with no `Blocked by:` relation expressed
- Backlog task missing `type:`, empty `references:`, empty DoD, or empty Implementation Plan
- Ending the turn after publishing with a description-only stub

## Verification

- [ ] `tasks.md` was produced at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md` with vertical tracer-bullet tickets (title, scope, acceptance, `SATISFIES`, files, blocks/blocked-by)
- [ ] Every ticket is dependency-ordered — no ticket appears in a wave before its blockers, and each "blocked by" edge is a real implementation dependency
- [ ] Every ticket is sized for one fresh agent context (implement + test + commit without re-reading the blueprint)
- [ ] The Fast-Track classification was applied — a `trivial`/`small` waiver record exists in `briefing.md`, or a full `tasks.md` with waves and acceptance criteria was produced
- [ ] The Slicing Self-Review passed — no oversized, non-vertical, demoable-less, or placeholder tickets
- [ ] Tickets were created in the tracker (or a local-only waiver was recorded when `ticketing.enabled: false`) and their IDs are recorded in `tasks.md` under each ticket's `Tracker ID`
- [ ] The Status Machine transitions are valid (no skipped statuses: `backlog` → `ready` → `in-progress` → `review` → `done`)
- [ ] The Pre-submission Privacy Review passed (no username, hostname, token, or private value in a ticket body)
- [ ] Ticket bodies follow the tracker's Work-Item Formatting (title convention, component split, explicit `Blocked by:`)
- [ ] When the tracker is Backlog.md, the Backlog completeness gate passed (`backlog task view <ID> --plain` shows type, references, DoD, and plan)
- [ ] When the tracker is `hermes-kanban`, every ticket appears in `hermes kanban list`, `hermes kanban show <ID>` confirms assignee + parents + workspace, and any local Backlog.md mirror carries the Kanban task ID
- [ ] `state.yaml` carries `phase: slicing` and `gate: user-gate-slice`, and the execution choice was presented

## Output Contract

The deliverables are `tasks.md` at
`.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md` (committed per the
work-unit commit contract) and the tracker state. Report back:

- The `tasks.md` path and its STATUS banner (`sliced (YYYY-MM-DD)`).
- The `## Delivery Strategy` section as written — the four plain-text guard
  lines verbatim, plus the work-unit table with each unit's review budget
  (`N / 400`).
- Per published ticket: its Tracker ID (from the create response — never
  invented) and the status it was published at.
- Any deferred decisions (e.g. `Chain strategy: pending`) named as such.

## How to measure it

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of tickets that execute in the wave the `tasks.md` predicted vs. re-sequenced | `state.yaml` / commit order | should rise |
| Lagging | Ticket size-estimate drift: planned S/M/L vs. actual PR lines changed | PR metadata | should fall |

