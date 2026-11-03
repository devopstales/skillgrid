---
name: writing-blueprints
description: Use when you have a spec or requirements for a multi-step task, before touching code
license: MIT
metadata:
  author: devopstales
  version: "1.1"
  part-of: skillgrid
  based_on: superpowers:writing-plans
---

# Writing Blueprints

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the skillgrid:writing-blueprints skill to create the implementation plan."

**Config:** Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root` for blueprint location. If the file doesn't exist, use the default `.skillgrid/specs/`. Use the terms vocabulary from `conventions.artifacts` (default `.skillgrid/artifacts/01-business-terms.md` + `02-technical-terms.md`) for names and concepts. Constrain the blueprint by the in-force decisions: read the change's ADR Review Manifest at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` (produced by brainstorming; in-force set from the `## LOCKED` ADR index in `.skillgrid/ASSUMPTIONS.md`) and respect every in-force ADR it names — a blueprint that contradicts an in-force ADR must either follow it or record a new superseding ADR file.

**The blueprint is pure-technical.** It is the *how*: architecture, file layout, tasks, tests, constraints. The product *why* — user value, personas, success metrics — lives in the spec (`briefing.md`) and in `.skillgrid/artifacts/00-prd.md`; the blueprint **cites** those (a `Spec:` line, a `per .skillgrid/artifacts/00-prd.md` or `per .skillgrid/artifacts/04-adr-NNNN-slug.md` constraint) and does not restate them (per `_shared/craft/cite-dont-restate.md`). Operational rules are cited as `per AGENTS.md § Rules`.

**Read the topic's findings before writing.** If `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` exists, read it in full before drafting the blueprint. It is the single consolidated evidence file for the topic — research findings (cited), prototype verdicts + liftable modules, and the sketch's chosen winner + constraints all live there as typed sections. Every design decision in the blueprint that rests on a research fact, a feasibility result, or a chosen layout must cite it (the header's **Findings** line carries the path). If the file is absent, the change ran no research/prototype/sketch — proceed and omit that header line.

**Context:** If working in an isolated worktree, it should have been created via the `skillgrid:isolated-workspace` skill at execution time.

**Save plans to:** `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md`

**Blueprint validation (mandatory, two-stage — after writing, before proceeding):**
1. **Critic subagent:** Dispatch a dedicated critic subagent with the blueprint content. The critic reviews for: architectural gaps, missing edge cases, untested assumptions, scope creep, and unclear interfaces. The critic returns a structured critique (issues + severity).
2. **User review:** Present the blueprint + critic findings to the user. For each issue, present options with benefits and weaknesses. Use the agent's answer-select tool for the user to choose a resolution path. The user must approve the final blueprint before `acceptance.feature` is written.

Never proceed to the next artifact without the blueprint being validated by both stages.

**Role (do not rename):** `blueprint.md` is the *how* (implementation plan —
steps, files, code, verification); the companion `briefing.md` is the *what*
(requirements & intent). The header must read
`# <Topic> — Implementation Blueprint (steps, files, verification)`. `brainstorming`
and `resume` key phase state off these exact filenames — never rename to
`plan.md`/`design.md`, and never move the folder out of `specs/`.
- Copy `templates/blueprint.md` from this skill's directory and fill it in.
- Commit the blueprint after saving (`git add` + `git commit`) — the artifact is a checkpoint alongside the code. The `pre-commit` zone guard (skillgrid:work-unit-commits) blocks a commit that mixes blueprint/spec changes with code.
- (User preferences for plan location override this default)

## Multi-Phase Change Folders

When a blueprint spans multiple phases, each phase folder is a normal change folder (`briefing.md`, not `spec.md`). Layout and rules: [multi-phase.md](references/multi-phase.md). The spec-folder tree is in `_shared/rules/sdd-structure.md`.

## When to Use

- When a change needs an implementation blueprint/plan before execution
- When converting a spec or approved design into a buildable plan

**When NOT to use:** for a trivial one-file change that doesn't need a plan — a blueprint is for non-trivial multi-task work.

## Scope Check

If the spec covers multiple independent subsystems, it should have been broken into sub-project specs during brainstorming. If it wasn't, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

### Per-phase change folders

When a blueprint is large enough to warrant phase separation (typically 4+ phases or 20+ tasks), split it into **one change folder per phase** as siblings under `.skillgrid/specs/`, plus a parent folder holding the shared blueprint index and shared context:

- **Parent folder** (`YYYY-MM-DD-<topic>/`) — not a change. Holds `blueprint.md` (status, tier, goal, architecture, tech stack, spec ref, and a phases table linking to each phase) and `shared/context.md` (hypothesis, terms, must-haves, global constraints, file structure shared across all phases).
- **Phase folders** (`YYYY-MM-DD-<topic>-phase-<N>/`) — each a normal change folder. `briefing.md` holds the phase goal. Later artifacts (`blueprint.md`, `tasks.md`, `review.md`) follow `_shared/rules/sdd-structure.md`.
- **Naming:** `<parent-change-name>-phase-<N>` (e.g. `2026-10-01-kubedash-5.0-phase-1`).
- **Back-links:** each phase `briefing.md` links to `../<parent>/blueprint.md` and `../<parent>/shared/context.md`. The parent's phases table links to `../<parent>-phase-N/`.
- **Do not nest** phase folders inside the parent — they must be siblings at the `.skillgrid/specs/` level.
- Each phase is sliced, executed, and shipped independently.

For smaller changes, use the standard single-folder layout. See the [Multi-phase changes](../../../docs/user-guide/01-layout.md#multi-phase-changes) section in the Layout reference.

## File Structure

Before defining tasks, map out which files will be created or modified and what each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are more reliable when files are focused. Prefer smaller, focused files over large ones that do too much.
- Files that change together should live together. Split by responsibility, not by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large files, don't unilaterally restructure - but if a file you're modifying has grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce self-contained changes that make sense independently.

## Must-Haves (Goal-Backward Verification)

Before defining tasks, derive the **Must-Haves** section from the spec's
requirements. These are the observable outcomes that MUST be true when the
plan is complete. The verifier checks the result against this list, not just
that tasks exist. Three categories:

- **Truths:** observable behaviors that must hold (e.g., "calling X returns Y")
- **Artifacts:** files that must exist with real implementation, not stubs
- **Key links:** critical connections between artifacts that must work together

Every item must be verifiable. If you can't state a truth, artifact, or link
for a spec requirement, the requirement is ambiguous — go back to the spec.

Tag a Must-Have truth **`backstop`** when it cannot be confirmed by reading the
diff alone — it needs a held-out test, a runtime, or state the diff doesn't
show (a behavior only observable through a real integration, a timing property,
a cross-component contract). `backstop` truths get a concrete held-out test in
the plan, and they are the spec-time signal that `skillgrid:parallel-code-review`
consumes: its verification-gap lens abstains (routes to `could-not-verify`
rather than a confident pass) on a check a `backstop` truth designates as
exogenous. Omit the tag for truths a plain diff read settles.

## Hypothesis

Every blueprint MUST open with one falsifiable hypothesis. Copy this block into `blueprint.md` under a `## Hypothesis` heading:

```markdown
**Claim:** [what the blueprint asserts will be true when complete]
**Right condition:** [observable evidence that the claim held]
**Wrong condition:** [observable evidence that the claim failed]
**Thinnest MVP:** [the smallest slice that would prove/falsify the claim]
**Door check:** [the first task that tests the hypothesis — if it fails, stop]
```

The hypothesis is the thinnest experiment that would prove or kill the
approach before the full build commits to it. The **door check** is the first
task in the blueprint that tests it — if the door check fails, the blueprint
is invalidated and the user decides whether to revise or abandon.

## Threat Matrix

If the blueprint touches routing, shell commands, subprocesses, version-control
automation, PR automation, executable-file classification, process integration,
Mnemonic tool contracts, or any `_shared/{planning,craft,verification,knowledge}/` file, include a threat
matrix section per `../../_shared/references/threat-matrix.md`. Copy this block into `blueprint.md` under a `## Threat Matrix` heading:

```markdown
| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| {boundary} | Applicable / N/A: {reason} | {response} | {concrete test} |
```

- Mark every row `Applicable` or explicit `N/A: reason`.
- Every Applicable row MUST have a concrete RED test that becomes a task in `tasks.md` and a scenario in `acceptance.feature`.
- Omit the section entirely if no boundary is touched.

## Build Shape

Record the change's **build shape** in the blueprint header (`**Build shape:**
<name>`). The shape is *how the work is ordered* — which behavior exists first —
and it shapes the first execution wave. These are decades-old delivery
strategies; the pick is a recorded decision (with a one-line why), not a vibe:

| Shape | What it means | When it's right |
|---|---|---|
| **Tracer thread** | A thin end-to-end path through every layer (UI → API → DB) works first, then thickens | You want something real and connected as early as possible; integration risk is the unknown |
| **Smallest usable whole** | The thinnest version a person could actually use ships first, then grows | Ship-and-learn; scope is uncertain but the core value is clear |
| **Facade** | The UI shell stands up on placeholder data first; the real backend wires in later | The look needs sign-off before the plumbing earns its investment; a prototype path |
| **Journey** | One user path is completed fully — every state of it — before the next begins | Each path must feel finished; the paths are genuinely independent |

Recommend the shape that fits (one line why) and record it. `skillgrid:slicing`
reads the recorded shape to order the first wave. The shape is orthogonal to
the risk-first / contract-first ordering strategies in slicing (those order
*within* waves; the shape orders the *first* wave).

## One-Way-Door Checkpoints

A **one-way-door** decision is hard to reverse: it needs a migration, breaks
a published contract, changes a public API shape, or is otherwise expensive
to undo. When a task implements such a decision, tag it with
`> ⚠ one-way: <decision>` at the top of the task and **STOP to get explicit
user approval before proceeding**. List all one-way-door decisions in the
Must-Haves section. If none, write "None".

## Change Classification

Classify the blueprint before writing tasks:

| Class | Verification Floor | When |
|---|---|---|
| `trivial` | L1 (focused test) | ≤3 files, no behavior change |
| `small` | L1 | ≤10 files, single concern |
| `standard` | L2 (full suite + build) | Normal feature/bug fix |
| `risky` | L3 (+ coverage + lint) | New trust boundary, migration |
| `high-risk` | L4 (+ mutation + security) | Auth, data migration, public API, money, concurrency |

Record the classification in the blueprint header. The `qa` skill uses it to select the verification level. See `../../_shared/rules/verification-ladder.md`.

## Task Right-Sizing

A task is the smallest unit that carries its own test cycle and is worth a
fresh reviewer's gate. When drawing task boundaries: fold setup,
configuration, scaffolding, and documentation steps into the task whose
deliverable needs them; split only where a reviewer could meaningfully
reject one task while approving its neighbor. Each task ends with an
independently testable deliverable.

## Plan header and task structure

Bite-sized steps, the plan document header, task structure, and the no-placeholders rule: [task-structure.md](references/task-structure.md).

## Self-Review

After writing the complete plan, look at the spec with fresh eyes and check the plan against it. This is a checklist you run yourself — not a subagent dispatch.

**1. Spec coverage:** Skim each section/requirement in the spec. Can you point to a task that implements it? List any gaps.

**2. Must-haves coverage:** Does the Must-Haves section (truths, artifacts, key links) cover every spec requirement? If a spec requirement has no corresponding truth, artifact, or key link, the plan won't verify correctly — add it.

**3. One-way-door completeness:** Are all hard-to-reverse decisions listed in the Must-Haves section AND tagged with `> ⚠ one-way:` on the task that implements them? If a migration, API change, or contract break is in a task but not flagged, add the tag.

**4. Placeholder scan:** Search your plan for red flags — any of the patterns in [task-structure.md](references/task-structure.md) under "No Placeholders". Fix them.

**5. Type consistency:** Do the types, method signatures, and property names you used in later tasks match what you defined in earlier tasks? A function called `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

If you find issues, fix them inline. No need to re-review — just fix and move on. If you find a spec requirement with no task, add the task.

## Owed-Decision Gate (input coverage)

After self-review passes and before the Plan Review gate, run the **input
coverage check** to catch decisions nobody made yet.

**Do NOT judge this by introspection** ("do I feel like I'm inventing
something?"). The build model happily rationalizes a real decision as "just
wiring" and pushes past it. The check is mechanical:

1. Enumerate **every value this build must produce, compute, or display** —
   from the Must-Haves truths and the spec's requirements (a total, a date, a
   status, a count, a URL, a permission).
2. For each, ask: **does the blueprint name where it comes from?** A named
   source is: a spec input, a data column, a derivation from another named
   value, or an in-force ADR.
3. Any required value with **no named source is an owed decision.**

A decision is also owed when the build would have to invent:
- a provider, library, integration, or data model;
- a whole screen or page whose design the blueprint does not pin down
  (layout, composition, asset strategy);
- a feature's behavior the acceptance criteria constrain but the blueprint
  never settles ("what exactly should it do?" is still open).

**Local implementation detail** is the narrow exception: a choice among
options the named sources already permit — a loop style, a variable name,
which helper to call. The moment a choice determines a value's source or a
behavior an acceptance criterion constrains, it is load-bearing by definition,
however small it looks.

**No owed decisions** → proceed to the Plan Review gate.

**Owed decisions exist** → present the panel (recommended first, one-line why;
never a neutral menu, never a silent decision):

> This blueprint owes a decision before it can be executed: `<name the
> specific load-bearing choice, e.g. "which date a "today" total is computed
> from — session time or the user's last-read row">`. How do you want to
> handle it?
>
> 1. **Resolve first** (recommended) — settle the decision in
>    brainstorming/interviewing now, then return to this blueprint. The build
>    has a real decision, not a guess.
> 2. **Not a real decision** — you've judged there's nothing load-bearing
>    here. Proceed; record the judgment as a one-line `Decision note:` in the
>    blueprint header.
> 3. **Build on the assumption** — keep moving. Write the **Assumed block**
>    below into the blueprint, then proceed. The assumption now lives in a
>    file, not in this chat — it survives `/clear`, teammates read it, and a
>    later run builds against it instead of guessing again. It stays flagged
>    as owing ratification until `skillgrid:writing-blueprints` (Ratify mode)
>    settles it. The flag never blocks execution.

### The Assumed block

On **Build on the assumption**, write this block into `blueprint.md` directly
after the header (before `## Must-Haves`), and set the header line
`**Status: ASSUMED**` (replace the default `**Status: PROPOSED**`). The
assumed values in the task steps are the named content — no placeholders for
them:

```markdown
**Status: ASSUMED** — owes ratification (run `skillgrid:writing-blueprints`
with this blueprint as the topic to ratify)

| | |
|---|---|
| Owed decision | {the specific load-bearing choice that was not made} |
| Assumption built on | {the concrete assumption this build will use} |
| Code area | {the paths this build will touch} |
| Authorized by | {user name/handle}, at the blueprint gate |
```

Rules:
- Only this gate (via option 3) creates an `ASSUMED` blueprint. Nothing else
  sets the status.
- The Assumed block records the assumption — it does NOT carry options,
  rationale, or alternatives. That content is written by Ratify.
- `ASSUMED` never blocks execution, slicing, QA, or ship. It is surfaced
  (WARNING, never CRITICAL) by `skillgrid:qa`, carried into the ship Return
  Envelope, and listed in the final `report.md` until ratified.
- A blueprint stays `ASSUMED` through the whole change. Only Ratify clears it.

## Ratify

**When to use:** a change's blueprint carries `**Status: ASSUMED**` and the
decision is now being deliberated — often phrased "ratify <feature>", or
surfaced by `skillgrid:qa`'s decision-debt WARNING, the ship Return Envelope,
or `skillgrid:reflect`'s report.

**Process:**
1. Read the Assumed blueprint in full. Its Owed decision, Assumption built on,
   and Code area tell you what was decided provisionally and where the code
   lives.
2. Run the design conversation anchored to what was **actually built** (the
   same infer / ask / recommend discipline as brainstorming). Deliberate the
   decision properly.
3. Two outcomes:
   - **The assumption holds.** Fill in the real decision record in the
     change's `adr.md` (ADR + options considered + rationale per the
     `adr_style` from config), then clear the flag: set the status line to
     `**Status: PROPOSED**` (the build is still in flight) or
     `**Status: ACCEPTED**` (built and verified). Delete nothing else.
   - **The assumption was wrong.** Write a corrected blueprint (a new
     `.skillgrid/specs/YYYY-MM-DD-<topic>/blueprint.md` for the same topic),
     mark the old one `**Status: SUPERSEDED by <new blueprint path>**`, and
     tell the user the build rests on a wrong assumption and should be redone
     against the corrected blueprint.

**Never leave a blueprint ASSUMED after a ratify run.** Ratification is the
only path out of the state — that is what keeps an override honest without
ever blocking the work.

## Plan Review

Fresh-eyes structural review before execution handoff: [plan-review.md](references/plan-review.md).

## Execution Handoff

After the plan review gate passes, check if slicing is needed:

**If the blueprint has 3+ tasks worth of work** — invoke `skillgrid:slicing` to break it into vertical tracer-bullet tickets with dependency edges and execution waves. Slicing produces `tasks.md` alongside the blueprint.

**If the blueprint has 1-2 tasks** — skip slicing, execute directly.

**Update `state.yaml`:** set `pipeline.current_change: <topic>` and `pipeline.current_phase: blueprint`.

Then offer execution choice:

**"Blueprint written and committed to `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task (or per ticket, if sliced), review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using skillgrid:simple-execution, batch execution with checkpoints

**Which approach?"**

**If Subagent-Driven chosen:**
- **REQUIRED SUB-SKILL:** Use skillgrid:subagent-execution
- Fresh subagent per task + two-stage review

**If Inline Execution chosen:**
- **REQUIRED SUB-SKILL:** Use skillgrid:simple-execution
- Batch execution with checkpoints for review

**Review at the end (both options):** the default is the lightweight two-axis
pass (`skillgrid:requesting-code-review`). For a large or high-risk blueprint
(at the review escalation threshold, per `_shared/planning/rigor-tiers.md`),
escalate the final review to `skillgrid:parallel-code-review` — multi-reviewer
fan-out. The execution skill decides when to escalate; you just tell the user
the option exists when the blueprint looks risky.

## How to measure it

Per `_shared/craft/measurement.md`.

| | Indicator | Data source | Direction |
|---|-----------|-------------|-----------|
| Leading | % of changes whose merged diff matches `blueprint.md` with no out-of-scope files | PR metadata (files changed) vs. `blueprint.md` scope | should rise |
| Lagging | Rework cycles: commits touching the change's files after the first merge that revert or re-scope them | Git history | should fall |

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The plan is obvious, skip the Must-Haves" | Goal-Backward Verification derives the observable outcomes the verifier checks against; without them a task can be "done" and the spec still unmet. |
| "I'll leave a placeholder and fill it in later" | No Placeholders calls TBD, "implement later", and "similar to Task N" plan failures — the engineer may read tasks out of order and a placeholder is a hole, not a shortcut. |
| "I'll skip the one-way-door checkpoints, they're slow" | A one-way decision is expensive to undo; the `> ⚠ one-way:` stop is the only gate that surfaces a migration or contract break before the build commits to it. |
| "Each task is small, granularity rules don't matter" | Right-Sizing + Bite-Sized Granularity keep every task an independently testable deliverable with its own review gate — a vague task is one a reviewer can't reject or approve. |
| "That's just wiring — I'll decide it during execution" | The build model rationalizes real decisions as wiring and buries the choice in code. If a value's source isn't named in the blueprint, the Owed-Decision Gate says it is owed. Resolve it, or record it as Assumed — don't invent it mid-build. |
| "The Assumed flag will get in the way later" | An `ASSUMED` blueprint never blocks execution, QA, or ship. What blocks later is the *silent* decision — the one that lived only in chat and got re-guessed by the next session. The flag is cheap insurance that the assumption stays visible until it's ratified. |

## Red Flags

- A task contains a placeholder (TBD, "implement later", "similar to Task N") — No Placeholders is violated.
- A task is not independently testable, or two tasks share a file edit with no ordering — not a vertical tracer bullet.
- A one-way-door decision (migration, contract break, data model) has no `> ⚠ one-way:` stop.
- A task step uses a value whose source is named nowhere in the blueprint and no `ASSUMED` block covers it — an owed decision passed the gate.
- An `ASSUMED` block with an empty "Assumption built on" cell — the assumption must be concrete, not "TBD".
- The Must-Haves / Goal-Backward Verification section is missing or empty — the verifier has nothing to check against.
- A task's scope exceeds one fresh agent context window with no split proposed.

## Verification

- [ ] The blueprint file exists at `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md` and opens with the Plan Document Header (Status, Goal, Architecture, Tech Stack, Spec).
- [ ] The Owed-Decision Gate (input coverage check) ran: every value the build must produce has a named source, or the blueprint carries a filled `ASSUMED` block (no empty cells).
- [ ] The Must-Haves (Goal-Backward Verification) section is present and covers every spec requirement with verifiable truths, artifacts, and key links.
- [ ] A placeholder scan (the No Placeholders patterns) returns zero hits across all tasks.
- [ ] Every one-way-door decision is listed in the Must-Haves section and tagged `> ⚠ one-way:` on the task that implements it.
- [ ] Self-Review and the Plan Review gate both ran and produced a verdict (READY FOR EXECUTION, or findings fixed inline).
- [ ] The Execution Handoff is present — slicing decision made and an execution approach (Subagent-Driven or Inline) is offered, so the plan is ready to hand off.
