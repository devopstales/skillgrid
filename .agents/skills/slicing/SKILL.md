---
name: slicing
description: Use after writing a blueprint to break it into vertical tracer-bullet tickets with dependency edges and execution waves. Produces tasks.md for the execution skills to pick up.
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: mattpocock-skills:to-tickets + colemedin-skills:piv-slice-epic
---

# Slicing

Break a blueprint into vertical tracer-bullet tickets that each fit a single fresh agent context window.

**Announce at start:** "I'm using the skillgrid:slicing skill to break the blueprint into executable tickets."

**Config:** Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root` for tasks.md location. If the file doesn't exist, use the default `.skillgrid/specs/`. Ticket titles and scope use the glossary vocabulary from `conventions.glossary` (default `.skillgrid/glossary/`). A ticket that contradicts an in-force ADR — per the change's ADR Review Manifest at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` — must call it out, not silently override it.

## When to Use

- After `skillgrid:writing-blueprints` produces a blueprint with 3+ tasks worth of work
- When the blueprint is too large for a single execution session
- When you need parallel execution waves (independent tickets can run concurrently)

**When NOT to use:** the blueprint has 1-2 tasks. Just execute it directly with `skillgrid:subagent-execution` or `skillgrid:simple-execution`.

## Fast-Track Classification

Before slicing, classify the change per `.skillgrid/config.yaml` `rules.fast_track`:

| Class | Criteria | Slicing behavior |
|---|---|---|
| `trivial` | ≤3 files, no behavior change, no migration/dependency/trust-boundary change | Light `tasks.md` (1–3 tickets, 1–2 lines each). Waiver recorded in `briefing.md`. |
| `small` | Single file/concern, ≤10 files, no new capability | Light `tasks.md` (fewer phases, 1–2 lines per ticket). Waiver recorded in `briefing.md`. |
| `standard` / `risky` | Everything else | Full `tasks.md` with complete dependency graph, waves, and acceptance criteria. |

**Blocked from fast-track** (full pipeline mandatory): any threat-matrix row Applicable, new trust boundary or migration, multi-domain impact, 400+ line change. When in doubt, run the full pipeline.

The waiver record (class, skips, reason, approved-by) must be present in `briefing.md` before `slicing` produces a light `tasks.md`. See `../_shared/conventions/fast-track.md`.

**Downstream:** the `SATISFIES` scenario names in `tasks.md` are the traceability oracle for `skillgrid:qa` — every scenario must be referenceable from a test. Name them precisely.

## The Process

### Step 1: Read the Blueprint

1. Read `blueprint.md` (technical details, task breakdown, interfaces)
2. Read `briefing.md` (context, falsifiable requirements, acceptance criteria)
3. Identify all work units in the blueprint

### Step 2: Identify Vertical Slices

Slice the work into tickets using these rules:

**Vertical, not horizontal.** Each ticket is a narrow but COMPLETE path through every layer (UI → API → DB → config). Not "all database schemas" then "all API endpoints" then "all UI." A vertical slice touches a little of each layer and produces working behavior.

**Sized for one fresh context window.** A ticket is the right size when a single agent session can implement it, test it, and commit it without needing to re-read the blueprint. Rough guide: ~500-1500 lines of change (20-50% tests). If the agent needs to go back to the blueprint mid-ticket, the ticket is too big.

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

**Acceptance-first (BDD is always on):** every ticket carries a `SATISFIES: <scenario-name>` pointing at its acceptance scenario in `.skillgrid/specs/<id>/acceptance.feature`. The scenario's RED state is confirmed before the implementation that turns it green. Order scenario/RED tickets ahead of their implementation tickets in the graph.

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

**Chain strategy** (a team decision, not the slicer's):
- **`stacked-to-main`** — each PR merges to main in order. Fast, fix on the go. Best for independent slices.
- **`feature-branch-chain`** — PR #1 targets the feature branch; later PRs target the previous PR branch so each child diff stays focused; only the tracker merges to main. Best for rollback control.
- **`size:exception`** — keep as one PR with maintainer approval. Best for generated code, migrations, vendor diffs.

For `feature-branch-chain`, name the intended base boundary per work unit: PR #1 base = main (or tracker branch); PR #2 base = PR #1 branch; PR #3 base = PR #2 branch.

**Fill in the `## Delivery Strategy` section** in `tasks.md` (the table + four plain-text guard lines + work-unit table). The four plain-text guard lines are the **contract** — `subagent-execution` matches them literally. Do not reword or drop any.

### Step 4: Group into Execution Waves

1. **Wave 1:** all tickets with no blockers (run in parallel)
2. **Wave 2:** tickets blocked only by Wave 1 tickets
3. **Wave N:** tickets blocked by Wave N-1 tickets

Independent tickets within a wave run in parallel worktrees. A ticket in a later wave waits for its dependency to be *complete* (reviewed, committed), not just started.

### Step 5: Write tasks.md

1. Copy `templates/tasks.md` from this skill's directory
2. Fill in:
   - Epic summary (2-3 lines from the blueprint)
    - Each ticket: title, scope, acceptance criteria, `SATISFIES` (scenario — BDD is always on), files, size, blocks, blocked by
   - Dependency graph (mermaid)
   - Execution order (waves, acceptance-first)
3. Save to `.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`

### Ticket Execution Contract (optional fields)

Each ticket MAY carry three optional, back-compat fields. **Absent = today's
behavior** — a ticket with none of them executes exactly as before. When present,
`skillgrid:subagent-execution` enforces them:

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
4. Commit per skillgrid:work-unit-commits — conventional subject, a `[skillgrid-context]` block, then `checkpoint-state.sh snapshot`. Wave commits (between execution waves) carry the same block so a resumed session knows which wave is done and which is next.

### Step 6: Hand Off

Check if ticketing is enabled:

**If `ticketing.enabled: true` in `.skillgrid/config.yaml`** — invoke `skillgrid:ticketing` to publish these tickets to the configured tracker.

**If `ticketing.enabled: false`** — skip ticketing, execute directly from `tasks.md`.

Then present the execution choice:

**"Tasks sliced and committed to `.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md`. N tickets in M waves. [Tickets published to <tracker> as epic <ID>.] Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per ticket, review between tickets, parallel worktrees within each wave

**2. Inline Execution** — Execute tickets in this session using skillgrid:simple-execution, wave by wave"

## Slicing Self-Review

Before committing tasks.md, check:

1. **Every ticket is vertical** — touches multiple layers, not just one
2. **Every ticket is demoable** — produces something testable/verifiable
3. **Every ticket fits one context window** — no ticket requires re-reading the blueprint
4. **Dependencies are real** — every "blocked by" edge represents an actual implementation dependency, not a convenience ordering
5. **Waves are correct** — no ticket appears in a wave before its blockers are in earlier waves
6. **No orphan work** — every task in the blueprint is covered by at least one ticket
7. **Acceptance criteria are falsifiable** — each ticket's acceptance criteria can be checked pass/fail

If any check fails, fix the slicing before committing.

## Common Rationalizations

| Excuse | Reality |
|--------|---------|
| "I'll just slice it in my head and tell the executor what to do" | The executor gets a fresh context. If the slice isn't written down, it doesn't exist for them. Write tasks.md. |
| "This ticket is a bit big but it's all related" | "All related" is how 3000-line tickets get born. Find the seam — there's always one. Split it. |
| "The dependency graph is overkill for 3 tickets" | With 3 tickets you don't need this skill. Use it when you have 5+. |
| "I'll figure out the waves during execution" | If the waves aren't explicit, the executor serializes everything. Parallelism is the whole point. |

## Red Flags

**Never:**
- Create a ticket that's a horizontal layer ("all DB schemas")
- Create a ticket too big for one context window
- Leave a ticket's acceptance criteria vague ("it works")
- Skip the dependency graph
- Put a ticket in a wave before its blockers

## Verification

- [ ] `tasks.md` was produced at `.skillgrid/specs/YYYY-MM-DD-<topic>/tasks.md` with vertical tracer-bullet tickets (title, scope, acceptance, `SATISFIES`, files, blocks/blocked-by)
- [ ] Every ticket is dependency-ordered — no ticket appears in a wave before its blockers, and each "blocked by" edge is a real implementation dependency
- [ ] Every ticket is sized for one fresh agent context (implement + test + commit without re-reading the blueprint)
- [ ] The Fast-Track classification was applied — a `trivial`/`small` waiver record exists in `briefing.md`, or a full `tasks.md` with waves and acceptance criteria was produced
- [ ] The Slicing Self-Review passed — no oversized, non-vertical, demoable-less, or placeholder tickets
- [ ] Ticket IDs are present if `ticketing.enabled: true` (published to the configured tracker as an epic)
