---
name: slicing
description: Use after writing a blueprint to break it into vertical tracer-bullet tickets with dependency edges and execution waves, then publish the tickets to the configured tracker, track their status through execution, and close them when done. Produces tasks.md for the execution skills to pick up and the tracker records that give execution a recovery point.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: skillgrid:slicing 1.1 + skillgrid:ticketing 1.0
---

# Sliece

## Contents
- Config / When to Use / Fast-Track Classification
- Part 1: Slicing / Part 2: Ticketing / Tracker Adapters / Discovery / Backlog completeness gate
- Pre-submission Privacy Review / Work-Item Formatting / Slicing Strategy / Ticket Lifecycle / CLI-Only Guard
- Common Rationalizations / Red Flags / Verification / How to measure it

Break a blueprint into vertical tracer-bullet tickets that each fit a single fresh agent context window, then publish them to the configured tracker and drive their status through execution to closure.

**Announce at start:** "I'm using the skillgrid:slicing skill to slice the blueprint into executable tickets and publish them to the tracker."

This skill merges two former skills: `slicing` (blueprint → `tasks.md`) and `ticketing` (tracker publication + status lifecycle). The flow is one continuous pass: slice, then publish, then record IDs — no handoff between skills.

## Config

Read `.skillgrid/config.yaml` before starting. This skill reads:

| Config key | Used for |
|------------|----------|
| `conventions.specs_root` | Where `tasks.md` lives (default `.skillgrid/change-specs/`) |
| `conventions.artifacts` | Terms vocabulary for ticket titles and scope (default `.skillgrid/projectwiki/glossary/01-business-terms.md` + `02-technical-terms.md`) |
| `rules.fast_track` | Fast-track classification criteria |
| `ticketing.enabled` | Tracker publication switch: `false` means record a local-only waiver after slicing; do not skip the ticketing phase itself |
| `ticketing.type` | Tracker type: `backlogmd` \| `gh` \| `glab` \| `jira` \| `hermes-kanban` |
| `ticketing.project_key` | Jira project key (when `ticketing.type: jira`) |
| `ticketing.kanban.board` | Hermes Kanban board slug (when `ticketing.type: hermes-kanban`; default: the `project` name) |
| `ticketing.kanban.assignee` | Hermes profile to assign tickets to (when `ticketing.type: hermes-kanban`; default `default`) |
| `ticketing.backlogmd.dir` | Backlog.md directory (default `.backlog/`; skillgrid convention: `.skillgrid/projectwiki/backlog`, resolved from root `backlog.config.yml` `backlog_directory`) |
| `ticketing.backlogmd.task_prefix` | Backlog.md task ID prefix (default `task`) |
| `ticketing.backlogmd.zero_padded_ids` | Backlog.md task ID zero-padding (default `3`) |
| `ticketing.gh.labels` | GitHub triage label strings |
| `ticketing.glab.labels` | GitLab triage label strings |
| `testing.runner` | Test command for DoD seeding (Backlog.md) |

If `.skillgrid/config.yaml` doesn't exist, invoke `skillgrid:onboarding` first. A ticket that contradicts an in-force ADR — per the change's ADR Review Manifest at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/adr.md` (in-force set from the `## LOCKED` ADR index in `.skillgrid/projectwiki/ASSUMPTIONS.md`) — must call it out, not silently override it.

## When to Use

- After `skillgrid:blueprint` produces a blueprint with 3+ tasks worth of work
- When the blueprint is too large for a single execution session
- When you need parallel execution waves (independent tickets can run concurrently)
- When a change needs to be tracked as work tickets, or tickets need creating, updating, or closing in the configured tracker (Backlog.md, GitHub, GitLab, Jira, Hermes Kanban)

**When NOT to use:** the blueprint has 1-2 tasks — just execute it directly with `skillgrid:execute` (inline mode). For ticket-status updates during execution (without slicing), follow the [Ticket Lifecycle](#ticket-lifecycle) section directly.

**Task creation mandate:** Ticketing ALWAYS uses the task creation skills (the tracker's task-creation tool — e.g. Backlog.md's `backlog_task_create`, or the tracker named in `.skillgrid/config.yaml` § `ticketing`). Never hand-write ticket rows without invoking the tracker's creation tool. **Prerequisite:** `acceptance.feature` must exist with valid syntax before slicing begins. **Position in artifact order:** Slicing is artifact #6 (briefing → blueprint → validation → cucumber tests → tasks → **slices**).

## Fast-Track Classification

Before slicing, classify the change per `.skillgrid/config.yaml` `rules.fast_track`:

| Class | Criteria | Slicing behavior |
|---|---|---|
| `trivial` | ≤3 files, no behavior change, no migration/dependency/trust-boundary change | Light `tasks.md` (1–3 tickets, 1–2 lines each). Waiver recorded in `briefing.md`. |
| `small` | Single file/concern, ≤10 files, no new capability | Light `tasks.md` (fewer phases, 1–2 lines per ticket). Waiver recorded in `briefing.md`. |
| `standard` / `risky` | Everything else | Full `tasks.md` with complete dependency graph, waves, and acceptance criteria. |

**Blocked from fast-track** (full pipeline mandatory): any threat-matrix row Applicable, new trust boundary or migration, multi-domain impact, 400+ line change. When in doubt, run the full pipeline.

The waiver record (class, skips, reason, approved-by) must be present in `briefing.md` before `slicing` produces a light `tasks.md`. See `../_shared/planning/fast-track.md`.

**Downstream:** the `SATISFIES` scenario names in `tasks.md` are the traceability oracle for `skillgrid:qa` — every scenario must be referenceable from a test. Name them precisely.

**Assumed blueprints:** if `blueprint.md` carries `**Status: ASSUMED**`, slicing proceeds normally (the flag never blocks). Add `Note: assumed decision (blueprint <path>)` to every ticket that implements the assumed value, so the decision debt is visible per ticket and survives into `qa`'s report and the ship Return Envelope.

## Part 1: Slicing

### Step 1: Read the Blueprint

1. Read `blueprint.md` (technical details, task breakdown, interfaces; note the `**Status:**` line — `ASSUMED` carries the debt note above)
2. Read `briefing.md` (context, falsifiable requirements, acceptance criteria)
3. Identify all work units in the blueprint

### Step 2: Identify Vertical Slices

**Build shape (orders the first wave):** read `**Build shape:**` from the
blueprint header. It decides which behavior exists *first*:
- **Tracer thread** → ticket 1 is the thin end-to-end path through every layer; later tickets thicken it.
- **Smallest usable whole** → ticket 1 is the thinnest usable product (one complete, shippable behavior); later tickets grow it.
- **Facade** → the UI shell ticket comes first (on placeholder data), with `Fails-when:` on the wire-up boundary that separates shell from real backend.
- **Journey** → one user path's tickets form the early waves, completed fully before the next path's tickets start.

The shape orders the *first* wave only — risk-first / contract-first ordering
still applies *within* and after it.

Slice the work into tickets using these rules:

**Vertical, not horizontal.** Each ticket is a narrow but COMPLETE path through every layer (UI → API → DB → config). Not "all database schemas" then "all API endpoints" then "all UI." A vertical slice touches a little of each layer and produces working behavior.

**Sized for one fresh context window.** A ticket is right-sized when the *full execution context* — source files read + implementation + test output + iteration headroom — completes within **20% of a fresh context window**. The diff alone undersells the cost: a 300-line ticket that reads 8 files and runs a full test suite is easily 40-50k tokens. Practical guide: ≤3 files modified, ≤2 subsystems crossed, ≤500 lines changed (20-50% tests). If the agent needs to go back to the blueprint mid-ticket, the ticket is too big.

**Demoable on its own.** Each ticket produces something testable or verifiable independently. Not "set up the schema" (invisible) but "user can create an account and see it in the list" (visible).

**Prefactors first.** Shared infrastructure (DB schema, base classes, config) goes into early tickets that feature tickets depend on.

**Split by dependency, concern, or slim end-to-end.** If a ticket is too big, split along a natural seam:
- By dependency: the thing that must exist first becomes its own ticket
- By concern: two independent behaviors become two tickets
- By slim end-to-end: strip out secondary features, ship the core path first

**Order by risk when it matters.** The *first* ticket should de-risk the
change. Two named strategies, applied on top of vertical slicing:
- **Risk-first:** slice the riskiest / most-uncertain piece first (the novel
  integration, the unproven API, the migration). If it fails, you learn before
  investing in the dependent slices.
- **Contract-first:** when two sides (e.g. backend + frontend) would otherwise
  block each other, slice 0 is a frozen API contract (types / OpenAPI); each
  side then slices against it independently and integrates last.

**Wide refactors are the exception.** A single mechanical change with huge blast radius (e.g., rename a column across 40 files) can't be a tracer bullet. Sequence it as:
1. Expand ticket (add the new thing alongside the old)
2. Migrate tickets (move batches, each blocked by expand)
3. Contract ticket (remove the old, blocked by all migrate batches)

### Step 3: Map Dependency Edges

For each ticket, determine:
- **Blocked by:** which tickets must be *implemented* (not just sliced) before this one can start
- **Blocks:** which tickets wait on this one

**Dependency = implemented, not sliced.** Ticket B is blocked by ticket A until A is done, not until A is written down. This matters for execution: the dependency graph drives wave ordering.

**Acceptance-first (BDD is always on):** every ticket carries a `SATISFIES: <scenario-name>` pointing at its acceptance scenario in `.skillgrid/change-specs/<id>/acceptance.feature`. The scenario's RED state is confirmed before the implementation that turns it green. Order scenario/RED tickets ahead of their implementation tickets in the graph.

### Step 3.5: Forecast Review Workload

Before grouping into waves, estimate whether the implementation is likely to exceed the **400 changed-line review budget** (`additions + deletions`). This is a planning guard, not an exact diff count. Use available signals: number of tickets, files, integration points, tests, migrations, and how many concerns the change crosses.

| Risk | Estimate | Action |
|---|---|---|
| **Low** | ≤ 400 lines | `Chained PRs recommended: No`. Single PR is fine. |
| **Medium** | 400–800 lines | `Chained PRs recommended: No`, but note the estimate. One PR with a clear review path. |
| **High** | > 800 lines | `Chained PRs recommended: Yes`. Split into work units. |

**If High**, split into **work units** that can become chained or stacked PRs — each with a clear start, clear finish, autonomous scope, verification, and a rollback boundary. Then set the **delivery strategy**:

| Strategy | Meaning |
|---|---|
| `ask-on-risk` (default) | `Decision needed before apply: Yes` — the orchestrator asks the user which chain strategy to use before execution |
| `auto-chain` | `Decision needed before apply: No` — proceed with slice 1 using the chosen strategy |
| `single-pr` | `Decision needed before apply: Yes` — requires `size:exception` before execution |
| `exception-ok` | `Decision needed before apply: No` — maintainer already accepted `size:exception` |

**Chain strategy** (a team decision, not the slicer's) — the three definitions and the `feature-branch-chain` base-boundary rule live in `_shared/rules/chain-strategy.md`; cite, don't restate. Record the choice in `tasks.md`, or defer with `Chain strategy: pending` when it is not the slicer's to decide.

**Fill in the `## Delivery Strategy` section** in `tasks.md` (the table + four plain-text guard lines + work-unit table). The four plain-text guard lines are the **contract** — `skillgrid:execute` matches them literally. Do not reword or drop any.

### Step 4: Group into Execution Waves

1. **Wave 1:** all tickets with no blockers (run in parallel)
2. **Wave 2:** tickets blocked only by Wave 1 tickets
3. **Wave N:** tickets blocked by Wave N-1 tickets

Independent tickets within a wave run in parallel worktrees. A ticket in a later wave waits for its dependency to be *complete* (reviewed, committed), not just started.

### Step 5: Write tasks.md

1. Copy `templates/tasks.md` from this skill's directory
2. Fill in the `> **STATUS:** \`sliced\` (YYYY-MM-DD)` banner on line 3 (see `_shared/rules/sdd-structure.md` § STATUS Banner)
3. Fill in:
   - Epic summary (2-3 lines from the blueprint)
   - Each ticket: title, scope, acceptance criteria, `SATISFIES` (scenario — BDD is always on), files, size, blocks, blocked by
   - **Ticket titles are ASCII only.** The backlog CLI slugifies titles into filenames; em dashes, en dashes, curly quotes, and other non-ASCII characters produce unquotable filenames. Use ASCII hyphens (`-`) for separators. See `references/ticketing/backlogmd-formatting.md` § Task title.
   - Dependency graph (mermaid)
   - Execution order (waves, acceptance-first)
   - When a ticket's scope or rationale rests on a constraint/decision it did
     not earn (an operational rule or a prior ADR), CITE the source rather than
     restate it (`per AGENTS.md § Rules <x>` / `per .skillgrid/projectwiki/decisions/04-adr-NNNN-slug.md`).
4. Save to `.skillgrid/change-specs/YYYY-MM-DD-<topic>/tasks.md`

### Ticket Execution Contract (optional fields)

Each ticket MAY carry three optional, back-compat fields. **Absent = today's
behavior** — a ticket with none of them executes exactly as before. When present,
`skillgrid:execute` enforces them:

- **`Precondition:`** — one line of *read-only, checkable* prose on what must be
  true **before** the ticket starts (a file exists, an env var is set, a health
  ping passes). The executor asserts it (no code changes) and, if unmet, halts
  with a checkpoint instead of doing partial work. Write it as a check the
  executor can run, not a vibe.
- **`Reversibility: reversible | costly | one-way`** — how expensive undo is.
  `one-way` inserts a **human checkpoint before** the ticket (it feeds the
  existing "four things stop you" rule; declaring it makes the stop a signal,
  not a judgment call). `costly` is advisory; `reversible` (or absent) is the
  default.
- **`Fails-when:`** — for every runnable verify command, the output that
  constitutes **failure**. A verify command with no `Fails-when` can't be judged
  pass/fail — name the exit code, the assertion, or the output substring that
  means "it failed." Extends the existing rule that a `Given/When/Then` with no
  falsifiable `Then` is rejected; this applies to the verify command itself.

Use `one-way` and `Precondition:` on migration, trust-boundary, and destructive
tickets. Leave them off for routine, reversible work — the back-compat default
is the point.

5. Commit per `skillgrid:work-unit-commits` — conventional subject, a `[skillgrid-context]` block. Wave commits (between execution waves) carry the same block so a resumed session knows which wave is done and which is next.

### Slicing Self-Review

Before publishing tickets, check:

1. **Every ticket is vertical** — touches multiple layers, not just one
2. **Every ticket is demoable** — produces something testable/verifiable
3. **Every ticket fits one context window** — no ticket requires re-reading the blueprint
4. **Dependencies are real** — every "blocked by" edge represents an actual implementation dependency, not a convenience ordering
5. **Waves are correct** — no ticket appears in a wave before its blockers are in earlier waves
6. **No orphan work** — every task in the blueprint is covered by at least one ticket
7. **Acceptance criteria are falsifiable** — each ticket's acceptance criteria can be checked pass/fail

If any check fails, fix the slicing before continuing.

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
- **Milestone name = spec folder name** (e.g. `2026-10-06-mnemonic-standalone`). The milestone must be named after the full spec folder under `.skillgrid/change-specs/` or `.skillgrid/change-archive/`, never a short slug or numeric ID. This makes the `milestone → spec` mapping 1:1 and recoverable from the tracker alone.

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
| `hermes-kanban` | [hermes-kanban.md](references/ticketing/hermes-kanban/hermes-kanban.md) → [templates/ticket.md](references/ticketing/hermes-kanban/templates/ticket.md) when building bodies |

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

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just slice it in my head and tell the executor what to do" | The executor gets a fresh context. If the slice isn't written down, it doesn't exist for them. Write tasks.md. |
| "This ticket is a bit big but it's all related" | "All related" is how 3000-line tickets get born. Find the seam — there's always one. Split it. |
| "The dependency graph is overkill for 3 tickets" | With 3 tickets you don't need this skill. Use it when you have 5+. |
| "I'll figure out the waves during execution" | If the waves aren't explicit, the executor serializes everything. Parallelism is the whole point. |
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

## How to measure it

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of tickets that execute in the wave the `tasks.md` predicted vs. re-sequenced | `state.yaml` / commit order | should rise |
| Lagging | Ticket size-estimate drift: planned S/M/L vs. actual PR lines changed | PR metadata | should fall |
