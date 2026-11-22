# Blueprint Plan Specification

## File Structure

Before defining tasks, map out which files will be created or modified and what
each one is responsible for. This is where decomposition decisions get locked in.

- Design units with clear boundaries and well-defined interfaces. Each file
  should have one clear responsibility.
- You reason best about code you can hold in context at once, and your edits are
  more reliable when files are focused. Prefer smaller, focused files over large
  ones that do too much.
- Files that change together should live together. Split by responsibility, not
  by technical layer.
- In existing codebases, follow established patterns. If the codebase uses large
  files, don't unilaterally restructure — but if a file you're modifying has
  grown unwieldy, including a split in the plan is reasonable.

This structure informs the task decomposition. Each task should produce
self-contained changes that make sense independently.

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
the plan, and they are the spec-time signal that `skillgrid:code-review`
consumes: its verification-gap lens abstains (routes to `could-not-verify`
rather than a confident pass) on a check a `backstop` truth designates as
exogenous. Omit the tag for truths a plain diff read settles.

## Hypothesis

Every blueprint MUST open with one falsifiable hypothesis. Copy this block into
`blueprint.md` under a `## Hypothesis` heading:

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

## Validation Matrix

Before defining tasks, derive the **`## Validation`** section from the
spec's requirements — the plan's statement of *how it will be proven*, written
before slicing. One row per in-scope requirement: the happy path, the edge,
the failure mode; the check (a runnable command or a named gate); and the
falsifier — the result that would falsify the plan. An out-of-scope row names
what is deliberately not proven.

- The **Check** column merges with the per-requirement test command (every
  in-scope requirement names a runnable signal before implementation).
- The **Happy / Edge / Failure** columns must line up with the
  `happy path` / `edge` / `failure` scenarios in `acceptance.feature` (Phase 5).
- Coverage is mechanical: a requirement with no row is a plan defect. `skillgrid:slicing`
  checks it before publishing, and `skillgrid:test`'s intent area re-runs the
  rows at gate time. A green suite is not the verdict — the matrix is.
- A `backstop` Must-Have truth (one the diff alone cannot confirm) gets its
  held-out test named in this matrix's Check column.

Copy the section into `blueprint.md` per the template block.

## Threat Matrix

If the blueprint touches routing, shell commands, subprocesses, version-control
automation, PR automation, executable-file classification, process integration,
Mnemonic tool contracts, or any `_shared/{planning,craft,verification,knowledge}/`
file, include a threat matrix section per `../_shared/references/threat-matrix.md`.
Copy this block into `blueprint.md` under a `## Threat Matrix` heading:

```markdown
| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| {boundary} | Applicable / N/A: {reason} | {response} | {concrete test} |
```

- Mark every row `Applicable` or explicit `N/A: reason`.
- Every Applicable row MUST have a concrete RED test that becomes a task in
  `tasks.md` and a scenario in `acceptance.feature`.
- Omit the section entirely if no boundary is touched.

## Build Shape

Record the change's **build shape** in the blueprint header (`**Build shape:**
<name>`). The shape is *how the work is ordered* — which behavior exists first —
and it shapes the first execution wave. These are decades-old delivery
strategies; the pick is a recorded decision (with a one-line why), not a vibe.
Each shape name is a leading word (per `_shared/craft/leading-words.md`): it
recruits the pretraining prior for "how this gets built," so the agent thinks
in the shape while ordering the first wave.

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

A **one-way-door** decision is hard to reverse: it needs a migration, breaks a
published contract, changes a public API shape, or is otherwise expensive to
undo. When a task implements such a decision, tag it with
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

Record the classification in the blueprint header. The `qa` skill uses it to
select the verification level. See `../_shared/rules/verification-ladder.md`.

## Task Right-Sizing

A task is the smallest unit that carries its own test cycle and is worth a
fresh reviewer's gate. When drawing task boundaries: fold setup, configuration,
scaffolding, and documentation steps into the task whose deliverable needs them;
split only where a reviewer could meaningfully reject one task while approving
its neighbor. Each task ends with an independently testable deliverable.

## Plan header and task structure

Bite-sized steps, the plan document header, task structure, and the
no-placeholders rule: [task-structure.md](references/task-structure.md).

## Self-Review

After the blueprint is fully drafted (section-by-section, per Phase 2) and
re-run at Phase 6, look at the spec with fresh eyes and check the plan against
it. This is a checklist you run yourself — not a subagent dispatch.

**1. Spec coverage:** Skim each section/requirement in the spec. Can you point
to a task that implements it? List any gaps.

**2. Must-haves coverage:** Does the Must-Haves section (truths, artifacts, key
links) cover every spec requirement? If a spec requirement has no corresponding
truth, artifact, or key link, the plan won't verify correctly — add it.

**2. Must-haves + validation coverage:** does the Must-Haves section (truths, artifacts, key
links) cover every spec requirement? If a spec requirement has no corresponding
truth, artifact, or key link, the plan won't verify correctly — add it. Does
`## Validation` carry a row for every
in-scope spec requirement (happy / edge / failure / check / falsifier), with
the check naming a runnable command or named gate? A requirement with no row
is a plan defect — add the row. A `backstop` truth's held-out test must appear
in its Check cell.

**3. One-way-door completeness:** Are all hard-to-reverse decisions listed in
the Must-Haves section AND tagged with `> ⚠ one-way:` on the task that
implements them? If a migration, API change, or contract break is in a task but
not flagged, add the tag.

**4. Placeholder scan:** Search your plan for red flags — any of the patterns in
[task-structure.md](references/task-structure.md) under "No Placeholders".
Fix them.

**5. Type consistency:** Do the types, method signatures, and property names
you used in later tasks match what you defined in earlier tasks? A function
called `clearLayers()` in Task 3 but `clearFullLayers()` in Task 7 is a bug.

**6. Spec reconciliation:** re-read `briefing.md` and compare it against the
blueprint per [section-by-section.md](references/section-by-section.md) — a
**contradiction** (the briefing decided a question, the blueprint decides it
differently) is reported every time, quoting both; an **addition** (the
blueprint decides something the briefing never raised) is reported only if it
changes architecture, an external interface, or a dependency. Contradictions
are listed first. Do NOT change the blueprint to match the briefing on your own
— drafting reads code the brainstorm did not. If nothing diverges, say so in one
line.

**7. Wiki reconciliation (Phase 3 ran):** does the project wiki reflect every
durable decision, research source, and technical detail the blueprint pins
down? A `decisions/` ADR for each locked ADR-shaped choice, a `sources/` +
`syntheses/` hub for each cited research result, an `architecture/` or
`gotchas/` page for each pinned technical detail. A vault that has none of these
is a document archive, not a wiki — `status --json` `by_type` skew is the tell.

If you find issues, fix them inline. No need to re-review — just fix and move
on. If you find a spec requirement with no task, add the task.

## Owed-Decision Gate (input coverage)

After self-review passes and before the Goldfish review, run the **input
coverage check** to catch decisions nobody made yet.

**Do NOT judge this by introspection** ("do I feel like I'm inventing
something?"). The build model happily rationalizes a real decision as "just
wiring" and pushes past it. The check is mechanical:

1. Enumerate **every value this build must produce, compute, or display** —
   from the Must-Haves truths and the spec's requirements (a total, a date, a
   status, a count, a URL, a permission).
2. For each, ask: **does the blueprint name where it comes from?** A named
   source is: a spec input, a data column, a derivation from another named
   value, an in-force ADR, or a projectwiki page.
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

**No owed decisions** → proceed to the Goldfish review.

**Owed decisions exist** → present the panel:

> This blueprint owes a decision before it can be executed: `<name the
> specific load-bearing choice, e.g. "which date a "today" total is computed
> from — session time or the user's last-read row">`. How do you want to
> handle it?
>
> 1. **Resolve first** — settle the decision in
>    brainstorming/interviewing now, then return to this blueprint. The build
>    has a real decision, not a guess.
> 2. **Not a real decision** — you've judged there's nothing load-bearing
>    here. Proceed; record the judgment as a one-line `Decision note:` in the
>    blueprint header.
> 3. **Build on the assumption** — keep moving. Write the **Assumed block**
>    below into the blueprint, then proceed. The assumption now lives in a
>    file, not in this chat — it survives `/clear`, teammates read it, and a
>    later run builds against it instead of guessing again. It stays flagged
>    as owing ratification until `skillgrid:blueprint` (Ratify mode)
>    settles it. The flag never blocks execution.

**Presenting the options (R22):** when the owed decision is a **load-bearing**
choice (data model, component boundaries, storage, protocol) with ≥2
defensible options, present each with its **mechanism, trade-offs, and the
conditions under which it fails**. The agent does not rank the options, does
not mark a default, does not pick one — the owner chooses. If the owner says
"just pick one," restate the trade-offs and present the same gate again.
**Mechanical** choices (file naming, formatting, variable names) are decided by
the agent without a gate — only architectural choices go to the owner. The
gate's input-coverage mechanics (the value-source enumeration above) are
unchanged — this is the tone shift from "here's my recommendation, please
confirm" to "here are the options and their failure conditions, you decide."

### The Assumed block

On **Build on the assumption**, write this block into `blueprint.md` directly
after the header (before `## Must-Haves`), and set the header line
`**Status: ASSUMED**` (replace the default `**Status: PROPOSED**`). The
assumed values in the task steps are the named content — no placeholders for
them:

```markdown
**Status: ASSUMED** — owes ratification (run `skillgrid:blueprint`
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
     change's `adr.md` (ADR + options considered + rationale, body per the
     `adr_style` template selected in `_shared/adr-records/README.md`), then
     clear the flag: set the status line to
     `**Status: PROPOSED**` (the build is still in flight) or
     `**Status: ACCEPTED**` (built and verified). Delete nothing else.
   - **The assumption was wrong.** Write a corrected blueprint (a new
     `.skillgrid/change-specs/YYYY-MM-DD-<topic>/blueprint.md` for the same
     topic), mark the old one `**Status: SUPERSEDED by <new blueprint path>**`,
     and tell the user the build rests on a wrong assumption and should be
     redone against the corrected blueprint.

**Never leave a blueprint ASSUMED after a ratify run.** Ratification is the
only path out of the state — that is what keeps an override honest without ever
blocking the work.

## Goldfish Review

The validation seam (run on `blueprint.md` after Phase 2/4 and before Phase 5):
the 5-stage fresh-context pipeline per [goldfish-review.md](references/goldfish-review.md).
The invocation passes `TargetFile` = `blueprint.md`, `GoldfishContext` =
`briefing.md` + referenced files + `findings.md`, `Step` = inferred from the
path classification (Interactive for New Project, Consolidated for New Function
with ≥3 tasks). Rounds iterate under the sharpening loop per
[goldfish-review-loop.md](references/goldfish-review-loop.md); each round's
findings resolve per [goldfish-resolve.md](references/goldfish-resolve.md).
Stage reports commit under the change folder's `review/` directory.

**User Review Gate** — the decide-then-fix loop: findings are routed back to
the user *while the blueprint is still PROPOSED and cheap to change*; for each
finding the user decides the resolution path, the blueprint is updated to
reflect it, and only the *final* updated blueprint is approved. On entering the
gate, write `changes.<topic>.gate: blueprint-validation` to `state.yaml` and
show the 30-second gate screen (User gate contract in
`_shared/rules/sdd-structure.md`); wait for the user's decision.

Never proceed to Phase 5/6 without the user approving the final blueprint. A
critique surfaced only after the blueprint is committed — with no user decision
feeding back into it — has missed its purpose.

## Execution Handoff

After the Goldfish review passes and the files are written (Phase 6), check if
slicing is needed:

**Decision gates:**

| Condition | Action |
|---|---|
| Blueprint carries `Status: ASSUMED` and the decision is being deliberated | Run **Ratify**: assumption holds → fill `adr.md`, clear the flag to PROPOSED/ACCEPTED; wrong → corrected blueprint, old one `SUPERSEDED`, build redone against it. |
| A blueprint is still `ASSUMED` after a ratify run | Not allowed — ratification is the only path out of the state. |
| Goldfish review findings not dispositioned with a user decision | No Phase 5/6 — the user review gate runs while the blueprint is still PROPOSED and cheap to change. |
| 3+ tasks worth of work | Invoke `skillgrid:slicing` (vertical tickets, dependency edges, waves). |
| 1–2 tasks | Skip slicing, execute directly. |
| Large or high-risk blueprint (review escalation threshold) | Final review escalates to `skillgrid:code-review` fan-out (multi-reviewer), not the lightweight pass. |
| A spec requirement has no truth/artifact/key link in Must-Haves | The plan won't verify correctly — add it before the self-review passes. |
| A requirement is missing from `## Validation` | Not publishable — the matrix covers the briefing or the slice descopeing it silently. |

After the Goldfish review passes and the files are written (Phase 6), check if
slicing is needed:

**If the blueprint has 3+ tasks worth of work** — invoke `skillgrid:slicing` to
break it into vertical tracer-bullet tickets with dependency edges and execution
waves. Slicing produces `tasks.md` alongside the blueprint.

**If the blueprint has 1-2 tasks** — skip slicing, execute directly.

**Update `state.yaml`:** set `changes.<topic>.phase: blueprint` (the entry was
created by brainstorming; if it's missing, add it with `status: in_progress`).

Then offer execution choice:

**"Blueprint written and committed to
`.skillgrid/change-specs/YYYY-MM-DD-<feature-name>/blueprint.md`. Two execution
options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task (or
per ticket, if sliced), review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using
`skillgrid:execute` (inline mode), batch execution with checkpoints

**Which approach?"**

**If Subagent-Driven chosen:**
- **REQUIRED SUB-SKILL:** Use `skillgrid:execute` (subagent mode)
- Fresh subagent per task + two-stage review

**If Inline Execution chosen:**
- **REQUIRED SUB-SKILL:** Use `skillgrid:execute` (inline mode)
- Batch execution with checkpoints for review

**Review at the end (both options):** the default is the lightweight pass of
`skillgrid:code-review` (the 3-axis pass at small/low-risk). For a large or
high-risk blueprint (at the review escalation threshold, per
`_shared/planning/rigor-tiers.md`), the final review escalates to
`skillgrid:code-review` fan-out — multi-reviewer. The execution skill decides
when to escalate; you just tell the user the option exists when the blueprint
looks risky.

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
| "I'll leave a placeholder and fill in later" | No Placeholders calls TBD, "implement later", and "similar to Task N" plan failures — the engineer may read tasks out of order and a placeholder is a hole, not a shortcut. |
| "I'll skip the one-way-door checkpoints, they're slow" | A one-way decision is expensive to undo; the `> ⚠ one-way:` stop is the only gate that surfaces a migration or contract break before the build commits to it. |
| "Each task is small, granularity rules don't matter" | Right-Sizing + Bite-Sized Granularity keep every task an independently testable deliverable with its own review gate — a vague task is one a reviewer can't reject or approve. |
| "That's just wiring — I'll decide it during execution" | The build model rationalizes real decisions as wiring and buries the choice in code. If a value's source isn't named in the blueprint, the Owed-Decision Gate says it is owed. Resolve it, or record it as Assumed — don't invent it mid-build. |
| "The Assumed flag will get in the way later" | An `ASSUMED` blueprint never blocks execution, QA, or ship. What blocks later is the *silent* decision — the one that lived only in chat and got re-guessed by the next session. The flag is cheap insurance that the assumption stays visible until it's ratified. |
| "Research is overkill for this change" | Phase 1 is for facts not in the codebase. If the design rests on a library's current API or a version's behavior, the blueprint cites `findings.md` — a fact your codebase *assumes* still gets re-verified against the source that owns it. |
| "The wiki is a nice-to-have, I'll skip Phase 3" | The wiki is the durable, cross-change knowledge that outlives the change. A blueprint that locks in an ADR-shaped decision or pins a non-obvious technical detail without filing it leaves the vault an archive, not a wiki — the next session re-derives it. |
| "I'll keep the prototype code, it's almost production-ready" | A prototype's output is an answer + a liftable pure module. Keeping the *demo* is a new request — classify it as a feature and go through the normal flow. |
| "I'll write the acceptance spec after the code works" | Phase 5 authors the spec from the validated blueprint, committed before code (Zone Rule). Post-hoc specs describe the code, not the requirement. |
| "One happy-path scenario per requirement is enough" | Every requirement has ≥1 `happy path` + `edge` + `failure` scenario. One happy path proves nothing about boundaries or errors. |
| "The blueprint is clear enough — I can skip the acceptance spec" | BDD is always on. The red scenarios at propose time are the change's work list, and a green suite is the gate for archive. |

## Red Flags

- A task contains a placeholder (TBD, "implement later", "similar to Task N") — No Placeholders is violated.
- A task is not independently testable, or two tasks share a file edit with no ordering — not a vertical tracer bullet.
- A one-way-door decision (migration, contract break, data model) has no `> ⚠ one-way:` stop.
- A task step uses a value whose source is named nowhere in the blueprint and no `ASSUMED` block covers it — an owed decision passed the gate.
- An `ASSUMED` block with an empty "Assumption built on" cell — the assumption must be concrete, not "TBD".
- The Must-Haves / Goal-Backward Verification section is missing or empty — the verifier has nothing to check against.
- A task's scope exceeds one fresh agent context window with no split proposed.
- A design decision rests on a research fact, feasibility result, or chosen layout that is not cited in `findings.md` (Phase 1/4 ran but the citation is missing).
- Phase 3 ran but the vault has no `decisions/`, `sources/`, or `syntheses/` page for a decision the blueprint pins down — the knowledge stayed in the per-change file and won't be recalled next session.
- A prototype verdict is `VALIDATED` with no edge case or comparison — the evidence rule is violated.
- Phase 4 was run without the user's explicit opt-in — it is user-selectable, not default.
- A `Then` step asserts internal state instead of an observable outcome, or a scenario has no `When` / no `Then` (Phase 5 Gherkin discipline).
- Implementation jargon (class, method, DB column) or a URL appears in a step or scenario name.
- A requirement has no `edge` or `failure` scenario, or scenario names collide (they are the `SATISFIES:` anchor for tasks).
- `acceptance.feature` was written before the blueprint passed the Goldfish review + user approval.

## Verification

- [ ] **Phase 1** — the `## Research:` section exists in `findings.md` (when
      research ran), committed, every load-bearing claim cited with confidence
      flagged; OR research was explicitly skipped (answer known from the
      codebase) and the header's Findings line is omitted.
- [ ] **Phase 2** — the blueprint was drafted section-by-section with the
      Section Approval Gate per section and per task; the user approved each
      section; the final `blueprint.md` exists at
      `.skillgrid/change-specs/YYYY-MM-DD-<feature-name>/blueprint.md` and opens
      with the Plan Document Header (Status, Goal, Architecture, Tech Stack,
      Spec).
- [ ] **Phase 3** — the project wiki reflects the blueprint's durable knowledge:
      `decisions/` ADRs for each locked choice, `sources/` + `syntheses/` for
      each cited research result, `architecture/`/`gotchas/` for each pinned
      technical detail; `projectwiki index` ran after every page write
      (`status --json` `stale: 0`); `projectwiki validate` + `lint` show no new
      findings.
- [ ] **Phase 4** — either the user opted in and the prototype produced an
      evidence-gated verdict (`VALIDATED`/`INVALIDATED`/`PARTIAL`) with the
      investigation trail, the `findings.md` `## Prototype:` section, and the
      feedback loop into the draft + wiki applied; OR the user opted out and
      `Prototype: skipped (user opt-out)` is recorded in the header.
- [ ] **Phase 5** — `acceptance.feature` exists in the change folder,
      passes the Gherkin parse (dry-run) and gherkin-lint with 0 errors,
      every requirement has ≥1 `happy path` + `edge` + `failure` scenario,
      and the Section Approval Gate on the spec was approved.
- [ ] **Phase 6** — the final `blueprint.md` and `acceptance.feature` are
      committed; the Self-Review
      (all checks including validation-matrix coverage and wiki reconciliation) ran; the Goldfish review
      produced a verdict — READY FOR EXECUTION (findings resolved or parked per
      the loop), not NEEDS REVISION; the user-decide-then-fix loop ran and the
      final blueprint was approved.
- [ ] The Owed-Decision Gate (input coverage check) ran: every value the build
      must produce has a named source, or the blueprint carries a filled
      `ASSUMED` block (no empty cells).
- [ ] The Must-Haves (Goal-Backward Verification) section is present and covers
      every spec requirement with verifiable truths, artifacts, and key links.
- [ ] A placeholder scan (the No Placeholders patterns) returns zero hits across
      all tasks.
- [ ] Every one-way-door decision is listed in the Must-Haves section and tagged
      `> ⚠ one-way:` on the task that implements it.
- [ ] The Execution Handoff is present — slicing decision made and an execution
      approach (Subagent-Driven or Inline) is offered, so the plan is ready to
      hand off.

## Output Contract

The deliverables are the change-folder files (all committed per the work-unit
commit contract) plus the handoff offer:

- `blueprint.md` at `.skillgrid/change-specs/YYYY-MM-DD-<topic>/` — opens with
  the Plan Document Header (Status, Goal, Architecture, Tech Stack, Spec),
  carries the Change Classification line, the Must-Haves section, and the
  `## Validation` matrix covering every briefing requirement.
- `acceptance.feature` in the same folder — parses clean (dry-run +
  gherkin-lint, 0 errors); every requirement has happy + edge + failure
  scenarios; scenario names are the `SATISFIES:` anchors.
- `findings.md` when research or a prototype ran — the `## Research:` /
  `## Prototype:` sections with cited sources and the evidence-gated verdict.
- Project-wiki pages for the blueprint's durable knowledge (decisions /
  sources / syntheses / architecture / gotchas) — `projectwiki index` +
  `validate` + `lint` run after the writes.
- The Execution Handoff — the slicing decision (sliced / 1–2 tasks direct)
  and the execution-approach offer (Subagent-Driven / Inline).

