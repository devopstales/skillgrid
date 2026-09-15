---
name: writing-blueprints
description: Use when you have a spec or requirements for a multi-step task, before touching code
license: MIT
metadata:
  author: devopstales
  version: "1.0"
  part-of: skillgrid
  based_on: superpowers:writing-plans
---

# Writing Blueprints

## Overview

Write comprehensive implementation plans assuming the engineer has zero context for our codebase and questionable taste. Document everything they need to know: which files to touch for each task, code, testing, docs they might need to check, how to test it. Give them the whole plan as bite-sized tasks. DRY. YAGNI. TDD. Frequent commits.

Assume they are a skilled developer, but know almost nothing about our toolset or problem domain. Assume they don't know good test design very well.

**Announce at start:** "I'm using the skillgrid:writing-blueprints skill to create the implementation plan."

**Config:** Read `.skillgrid/config.yaml` before starting. Use `conventions.specs_root` for blueprint location. If the file doesn't exist, use the default `.skillgrid/specs/`. Use the glossary vocabulary from `conventions.glossary` (default `.skillgrid/glossary/`) for names and concepts. Constrain the blueprint by the in-force decisions: read the change's ADR Review Manifest at `.skillgrid/specs/YYYY-MM-DD-<topic>/adr.md` (produced by brainstorming) and respect every in-force ADR it names — a blueprint that contradicts an in-force ADR must either follow it or record a new superseding ADR.

**Read the topic's findings before writing.** If `.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md` exists, read it in full before drafting the blueprint. It is the single consolidated file for the topic — spike verdicts, liftable modules, and the sketch's chosen winner + constraints all live there. Every design decision in the blueprint that rests on a feasibility result or a chosen layout must cite it (the header's **Spike findings** / **Sketch findings** lines carry the path). If the file is absent, the change ran no spike/sketch — proceed and omit those header lines.

**Context:** If working in an isolated worktree, it should have been created via the `skillgrid:isolated-workspace` skill at execution time.

**Save plans to:** `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md`
- Copy `templates/blueprint.md` from this skill's directory and fill it in.
- Commit the blueprint after saving (`git add` + `git commit`) — the artifact is a checkpoint alongside the code. The `pre-commit` zone guard (skillgrid:work-unit-commits) blocks a commit that mixes blueprint/spec changes with code.
- (User preferences for plan location override this default)

## When to Use

- When a change needs an implementation blueprint/plan before execution
- When converting a spec or approved design into a buildable plan

**When NOT to use:** for a trivial one-file change that doesn't need a plan — a blueprint is for non-trivial multi-task work.

## Scope Check

If the spec covers multiple independent subsystems, it should have been broken into sub-project specs during brainstorming. If it wasn't, suggest breaking this into separate plans — one per subsystem. Each plan should produce working, testable software on its own.

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

Every blueprint MUST open with a falsifiable hypothesis:

```markdown
## Hypothesis

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
Mnemonic tool contracts, or any `_shared/conventions/*` file, include a threat
matrix section per `../_shared/references/threat-matrix.md`:

```markdown
## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| {boundary} | Applicable / N/A: {reason} | {response} | {concrete test} |
```

- Mark every row `Applicable` or explicit `N/A: reason`.
- Every Applicable row MUST have a concrete RED test that becomes a task in `tasks.md` and a scenario in `acceptance.feature`.
- Omit the section entirely if no boundary is touched.

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

Record the classification in the blueprint header. The `qa` skill uses it to select the verification level. See `../_shared/conventions/verification-ladder.md`.

## Task Right-Sizing

A task is the smallest unit that carries its own test cycle and is worth a
fresh reviewer's gate. When drawing task boundaries: fold setup,
configuration, scaffolding, and documentation steps into the task whose
deliverable needs them; split only where a reviewer could meaningfully
reject one task while approving its neighbor. Each task ends with an
independently testable deliverable.

## Bite-Sized Task Granularity

**Each step is one action (2-5 minutes):**
- "Write the failing test" - step
- "Run it to make sure it fails" - step
- "Implement the minimal code to make the test pass" - step
- "Run the tests and make sure they pass" - step
- "Commit" - step

## Plan Document Header

**Every plan MUST start with this header** (the `templates/blueprint.md` file
includes it plus a sample task — use it as the starting scaffold):

```markdown
# [Feature Name] Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** [One sentence describing what this builds]

**Architecture:** [2-3 sentences about approach]

**Tech Stack:** [Key technologies/libraries]

**Spec:** [path to the spec/design doc this plan implements — the plan
argues from the spec, so the spec travels with it; executors read both]

**Research:** [path to the change's `.skillgrid/specs/YYYY-MM-DD-<topic>/research.md`
if one exists — produced by `skillgrid:research` or `skillgrid:deep-research`.
The blueprint cites it for every decision that rests on a fact not in the
codebase; the spec argues from the research, so it travels with the plan. Omit
if the change needed no external research.]

**Spike findings:** [path to the change's consolidated
`.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md`, "Spike" sections only —
produced by `skillgrid:spike`. Cite it for every design decision that rests on
a feasibility result (e.g. "X is liftable", "approach A is cheaper than B").
Omit if the change ran no spike.]

**Sketch findings:** [path to the change's consolidated
`.skillgrid/specs/YYYY-MM-DD-<topic>/findings.md`, "Sketch" sections only —
produced by `skillgrid:sketch`. Cite it for the chosen layout/interaction, the
marked winner, and the constraints the build must preserve. Omit if the change
skipped a sketch.]

## Global Constraints

[The spec's project-wide requirements — version floors, dependency limits,
naming and copy rules, platform requirements — one line each, with exact
values copied verbatim from the spec. Every task's requirements implicitly
include this section.]

---
```

## Task Structure

````markdown
### Task N: [Component Name]

**Files:**
- Create: `exact/path/to/file.py`
- Modify: `exact/path/to/existing.py:123-145`
- Test: `tests/exact/path/to/test.py`

**Interfaces:**
- Consumes: [what this task uses from earlier tasks — exact signatures]
- Produces: [what later tasks rely on — exact function names, parameter
  and return types. A task's implementer sees only their own task; this
  block is how they learn the names and types neighboring tasks use.]

**SATISFIES:** [scenario-name]
(BDD is always on: name the acceptance scenario in
`.skillgrid/specs/<id>/acceptance.feature` that this task makes green.
The RED step in Step 1/2 targets this scenario; TDD Evidence references
the scenario name.)

- [ ] **Step 1: Write the failing test**

```python
def test_specific_behavior():
    result = function(input)
    assert result == expected
```

- [ ] **Step 2: Run test to verify it fails**

Run: `pytest tests/path/test.py::test_name -v`
Expected: FAIL with "function not defined"

- [ ] **Step 3: Write minimal implementation**

```python
def function(input):
    return expected
```

- [ ] **Step 4: Run test to verify it passes**

Run: `pytest tests/path/test.py::test_name -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add tests/path/test.py src/path/file.py
git commit -m "feat: add specific feature"
```
````

## No Placeholders

Every step must contain the actual content an engineer needs. These are **plan failures** — never write them:
- "TBD", "TODO", "implement later", "fill in details"
- "Add appropriate error handling" / "add validation" / "handle edge cases"
- "Write tests for the above" (without actual test code)
- "Similar to Task N" (repeat the code — the engineer may be reading tasks out of order)
- Steps that describe what to do without showing how (code blocks required for code steps)
- References to types, functions, or methods not defined in any task

## Self-Review

After writing the complete plan, look at the spec with fresh eyes and check the plan against it. This is a checklist you run yourself — not a subagent dispatch.

**1. Spec coverage:** Skim each section/requirement in the spec. Can you point to a task that implements it? List any gaps.

**2. Must-haves coverage:** Does the Must-Haves section (truths, artifacts, key links) cover every spec requirement? If a spec requirement has no corresponding truth, artifact, or key link, the plan won't verify correctly — add it.

**3. One-way-door completeness:** Are all hard-to-reverse decisions listed in the Must-Haves section AND tagged with `> ⚠ one-way:` on the task that implements them? If a migration, API change, or contract break is in a task but not flagged, add the tag.

**4. Placeholder scan:** Search your plan for red flags — any of the patterns from the "No Placeholders" section above. Fix them.

**5. Type consistency:** Do the types, method signatures, and property names you used in later tasks match what you defined in earlier tasks? A function called `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

If you find issues, fix them inline. No need to re-review — just fix and move on. If you find a spec requirement with no task, add the task.

## Plan Review

After self-review passes, run a fresh-eyes structural review of the blueprint
before handing off to execution. This is NOT a re-do of self-review — it
checks what self-review can't: whether the plan will survive contact with
reality.

### When to run

- Always, for blueprints with 3+ tasks or any one-way-door decision
- Skip for 1-2 task blueprints with no migrations or contract breaks
  (self-review is sufficient)

### How to run

**Option A — Subagent (preferred for 3+ tasks):**
Dispatch a `general-purpose` subagent with the blueprint file path, the spec
file path, and the in-force ADR manifest. Prompt:

> You are a plan reviewer. Read the blueprint and spec. Do NOT review code
> quality — review the PLAN. Check:
> 1. **Failure modes:** For each task, what can go wrong at runtime that the
>    plan doesn't address? (network failure, partial state, concurrent access,
>    empty data, timeout) Name specific failure → specific mitigation.
> 2. **Scope coherence:** Does the plan do MORE than the spec asks (scope
>    creep) or LESS (scope gap)? Flag both.
> 3. **Feasibility:** Does any task assume a capability the codebase doesn't
>    have (library, API, infra)? Check against the codebase if needed.
> 4. **Ordering:** Are dependencies correct? Could a later task be done
>    first (parallelism) or does an earlier task block on something that
>    doesn't exist yet?
> 5. **Test strategy:** Does each task have a verifiable "done" state? Are
>    the acceptance scenarios (BDD) sufficient to prove the feature works?
> 6. **ADR compliance:** Does any task contradict an in-force ADR? If so,
>    flag it.
>
> Output: a findings list (Critical / Important / Minor) with file:line
> references to the blueprint, plus a verdict: READY FOR EXECUTION |
> NEEDS REVISION.

**Option B — Inline (1-2 tasks, no one-way doors):**
Run the same 6 checks yourself against the blueprint. If all pass, proceed.
If any fail, fix inline.

### Acting on findings

Apply `skillgrid:receiving-code-review` triage rules:
- **Fix now:** structural gaps that would block execution (missing dependency,
  wrong ordering, ADR violation) — fix the blueprint, re-commit
- **Defer:** nice-to-have refinements (extra test cases, edge case notes) —
  add as a note in the blueprint's "Global Constraints" or the task's step
- **Human look:** scope questions the user should answer — surface and ask
- **Noise:** reviewer overreach — note and drop

If the verdict is NEEDS REVISION: fix the Critical/Important items, re-run
the self-review checklist, then hand off. Do NOT enter a full review loop —
one pass is the cap. The execution phase has its own review; this gate
catches structural problems, not code quality.

### Output

Append a brief review summary to the blueprint file:

```markdown
## Plan Review

- Verdict: READY FOR EXECUTION
- Findings: 0 Critical, 1 Important (fixed: added error handling for
  timeout in Task 3), 2 Minor (deferred: retry logic, logging)
- Reviewed: [date]
```

## Execution Handoff

After the plan review gate passes, check if slicing is needed:

**If the blueprint has 3+ tasks worth of work** — invoke `skillgrid:slicing` to break it into vertical tracer-bullet tickets with dependency edges and execution waves. Slicing produces `tasks.md` alongside the blueprint.

**If the blueprint has 1-2 tasks** — skip slicing, execute directly.

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
pass (`skillgrid:requesting-code-review`). For a large (50+ changed lines) or
high-risk blueprint (auth, data migration, money, concurrency, public API),
escalate the final review to `skillgrid:parallel-code-review` — multi-reviewer
fan-out. The execution skill decides when to escalate; you just tell the user
the option exists when the blueprint looks risky.

## Common Rationalizations

| Rationalization | Reality |
|---|---|
| "The plan is obvious, skip the Must-Haves" | Goal-Backward Verification derives the observable outcomes the verifier checks against; without them a task can be "done" and the spec still unmet. |
| "I'll leave a placeholder and fill it in later" | No Placeholders calls TBD, "implement later", and "similar to Task N" plan failures — the engineer may read tasks out of order and a placeholder is a hole, not a shortcut. |
| "I'll skip the one-way-door checkpoints, they're slow" | A one-way decision is expensive to undo; the `> ⚠ one-way:` stop is the only gate that surfaces a migration or contract break before the build commits to it. |
| "Each task is small, granularity rules don't matter" | Right-Sizing + Bite-Sized Granularity keep every task an independently testable deliverable with its own review gate — a vague task is one a reviewer can't reject or approve. |

## Red Flags

- A task contains a placeholder (TBD, "implement later", "similar to Task N") — No Placeholders is violated.
- A task is not independently testable, or two tasks share a file edit with no ordering — not a vertical tracer bullet.
- A one-way-door decision (migration, contract break, data model) has no `> ⚠ one-way:` stop.
- The Must-Haves / Goal-Backward Verification section is missing or empty — the verifier has nothing to check against.
- A task's scope exceeds one fresh agent context window with no split proposed.

## Verification

- [ ] The blueprint file exists at `.skillgrid/specs/YYYY-MM-DD-<feature-name>/blueprint.md` and opens with the Plan Document Header (Goal, Architecture, Tech Stack, Spec).
- [ ] The Must-Haves (Goal-Backward Verification) section is present and covers every spec requirement with verifiable truths, artifacts, and key links.
- [ ] A placeholder scan (the No Placeholders patterns) returns zero hits across all tasks.
- [ ] Every one-way-door decision is listed in the Must-Haves section and tagged `> ⚠ one-way:` on the task that implements it.
- [ ] Self-Review and the Plan Review gate both ran and produced a verdict (READY FOR EXECUTION, or findings fixed inline).
- [ ] The Execution Handoff is present — slicing decision made and an execution approach (Subagent-Driven or Inline) is offered, so the plan is ready to hand off.
