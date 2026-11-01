# Goldfish Plan Review — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-01)

> **Role:** the *what* — validated design/spec: problem, intent, requirements,
> success criteria. The companion `blueprint.md` is the *how*. Do not rename.

## Problem / Intent

skillgrid's before-code review of a `blueprint.md` is thin: a same-context
Self-Review (the author grading its own plan) plus a single generic "Critic
subagent" pass and one structural Plan Review. All three run with the author's
assumptions already in context, so they rubber-stamp the very blind spots they
are meant to catch — unstated assumptions, terms a fresh reader cannot
resolve, altitude drops, and gaps that make the plan unbuildable by a
zero-context implementer.

This change ports the **Elephant-Goldfish Model (EGM)** review idea onto the
existing `writing-blueprints` validation seam: a fresh, zero-context subagent
reviewer that has seen *only* the blueprint and the files it references. A
review performed in the author's context is not a review — it inherits the
author's assumptions. A fresh reviewer exposes exactly what the same-context
passes rationalized away.

The review lands **before execution**, on the document — it is not `qa`, which
verifies the *code* after it is written. The two are complementary gates:
Goldfish plan review asks "will this plan, if followed, produce the right
thing?"; `qa` asks "did the code produce the right thing?"

## Purpose & Success Criteria

- **Purpose:** give `writing-blueprints` a fresh-context, multi-persona plan
  review (zoom-out / comprehension / design-critic / implementation-readiness)
  that runs between Self-Review and the Execution Handoff, and that produces
  committed, per-persona report artifacts in the change folder.
- **Success criteria:**
  - A blueprint with ≥3 tasks (or any one-way-door) is reviewed by at least one
    **fresh** subagent that was not given the author's conversation context,
    before `acceptance.feature` is written.
  - The comprehension reviewer's paraphrase of the blueprint is written to an
    artifact; any section it could not paraphrase, or any term it had to guess,
    is a recorded finding.
  - Review findings are dispositioned (fixed / parked / waived / contested) and
    the review is committed in the spec zone before execution handoff.
  - `qa`, `requesting-code-review`, `review-ledger`, and `parallel-code-review`
    are **unchanged** — the after-code half is untouched.
- **Out of scope:**
  - No change to `qa`, the after-code quality gate.
  - No new top-level skill (the protocol is a subflow of `writing-blueprints`
    plus shared persona/discipline reference files).
  - No `egm-bootstrap` (per-directory README contracts), no separate
    `concepts_brief.md` artifact, no synthesizer subagent-of-subagents layer.
  - Drift report (doc-vs-merged-code) is a separate later change.
  - No new third-party dependency (prose-only change; the only code is skill
    markdown + reference docs).

## Context

- `writing-blueprints` (`.agents/skills/planning/writing-blueprints/SKILL.md`,
  `large` tier, 26KB) already has the seam this change upgrades: **Self-Review**
  → **Critic subagent** (the "Blueprint validation, two-stage" block) → **Owed-
  Decision Gate** → **Plan Review** (`references/plan-review.md`) → **Execution
  Handoff**. The single generic Critic subagent is replaced/augmented by the
  3-stage Goldfish review.
- `references/plan-review.md` already dispatches a `general-purpose` subagent
  with a fixed prompt (failure modes / scope / feasibility / ordering / test
  strategy / ADR compliance). The Goldfish personas subsume and sharpen these.
- skillgrid's split is a **file boundary** (`briefing.md` = what, `blueprint.md`
  = how), which is cleaner than EGM's header-scan split — the review applies
  different rubrics to each file without parsing `## Detailed Implementation
  Plan`.
- Shared conventions to respect: `_shared/rules/sdd-structure.md` (artifact
  order, status contract, phase derivation), `_shared/rules/prose-style.md`
  (21 prose rules), `_shared/rules/verification-ladder.md` (the gate stays the
  gate), `_shared/craft/skill-anatomy.md` + `skill-creator` (keep SKILL.md under
  its tier ceiling; push detail to `references/`).
- The reviewer subagent pool in this harness is fixed (`general`, `explore`,
  `researcher`, `reviewer`, `verifier`, `implementer`, `debugger`). EGM's named
  Goldfish subagents are realized as **persona brief files** the `reviewer`/
  `general` subagent reads first — EGM's own documented fallback for harnesses
  that do not register named subagents.
- EGM source: `github.com/google/elephant-goldfish-model` (Apache-2.0). Personas
  ported: `zoom_out`, `comprehension`, `design_critic`,
  `implementation_readiness`, `codebase_details`; folded: `structural`,
  `style`; dropped: `synthesizer`.

## Approaches Considered

- **Chosen:** Upgrade the existing `writing-blueprints` validation seam with a
  3-stage fresh-context Goldfish review, backed by a `goldfish/` persona-brief
  directory under `_shared/craft/` and one shared `review-discipline.md`. Keep
  the SKILL.md edits thin (the file is already 26KB of a 38KB ceiling); the
  protocol detail lives in a new `references/goldfish-review.md`.
- **Rejected:** Add a new top-level `goldfish-review` skill — duplicates the
  seam, adds a phase and a resume marker, and forces a change to
  `sdd-structure.md`'s phase table. The review is a subflow of blueprint
  validation, not a new pipeline phase.
- **Rejected:** Add the review to `qa` — wrong time and wrong target (code, not
  document). Would conflate the before-code and after-code failure classes.
- **Rejected:** Register new named OpenCode subagents per persona — the harness
  does not make that cheap, and EGM's documented fallback (general-purpose
  subagent + "read this persona file first") is the faithful, lower-friction
  port.

## Requirements

1. **R1 — Fresh-context isolation (the load-bearing rule).**
   - **Current:** All three existing passes (Self-Review, Critic subagent,
     Plan Review) run with the author's assumptions in context.
   - **Target:** Each Goldfish stage dispatches a `reviewer` (fallback
     `general`) subagent given ONLY the target file path(s) and the files that
     target references — no conversation history, no author narrative. A review
     run in the author's own context is not a Goldfish review and MUST NOT count
     toward satisfying R1.
   - **Acceptance:** the reviewer prompt names the target path and the reading
     list; the prompt contains no summary of the brainstorming conversation;
     the subagent is a freshly spawned one (not the session's own context).
   - **Acceptance scenario:** `happy path fresh-context-review` →
     `acceptance.feature`.

2. **R2 — Three review stages, one per persona-group.**
   - **Current:** one generic Critic subagent covers "architectural gaps,
     missing edge cases, untested assumptions, scope creep, unclear interfaces"
     in a single pass.
   - **Target:** the single pass is replaced by three stages:
     (a) **zoom-out** (full `blueprint.md`): YAGNI, reuse, anti-weeds, altitude;
     (b) **comprehension** (`blueprint.md` + referenced files): paraphrase
     first, self-containment; (c) **critic** = design-critic +
     implementation-readiness on `blueprint.md` (and `briefing.md` for the
     what-side). Each stage is its own subagent dispatch.
   - **Acceptance:** `SKILL.md`'s validation block names all three stages and
     their target ranges; the stage list is not a single combined prompt.
   - **Acceptance scenario:** `happy path three-stage-review` →
     `acceptance.feature`.

3. **R3 — Comprehension test (self-containment).**
   - **Current:** no check that a zero-context reader can reconstruct the
     design from the artifact alone.
   - **Target:** the comprehension reviewer writes (1) a paraphrase of what the
     blueprint proposes and how the existing system works per the doc, under
     two fixed headings, and (2) a list of every term it had to guess or
     define, and every section it could not paraphrase. Each is a finding.
   - **Acceptance:** the artifact starts with the two paraphrase headings; a
     section that could not be paraphrased is recorded as a finding (never
     dropped silently).
   - **Acceptance scenario:** `happy path comprehension-paraphrase` →
     `acceptance.feature`.

4. **R4 — Persona briefs live in `_shared/craft/goldfish/`.**
   - **Current:** no shared reviewer-persona files.
   - **Target:** new files
     `_shared/craft/goldfish/{zoom-out,comprehension,design-critic,
     implementation-readiness,codebase-details}.md`, each a self-contained
     read-only persona brief (role, isolation mandate, audit vectors, output
     format). The `structural` and `style` personas are folded into
     `review-discipline.md`, not standalone files.
   - **Acceptance:** each file exists, is ≤150 lines, names its audit vectors,
     and its output format matches the finding schema in R5. No `synthesizer.md`.
   - **Acceptance scenario:** `happy path persona-briefs-present` →
     `acceptance.feature`.

5. **R5 — Review-discipline reference + anti-sycophancy rule.**
   - **Current:** `prose-style.md` has 21 rules but no explicit no-flattery /
     stress-test rule; reviewer personas do not share a common discipline doc.
   - **Target:** new `_shared/craft/review-discipline.md` distills EGM's
     execution constraints (stress-test by default, critique-first, generate a
     counter-argument per assertion, no glazing, banned validating adjectives,
     no apologies, banned decorative adverbs) into one citable reference.
     `prose-style.md` gains one short "No flattery" rule that points at it.
     Every goldfish persona brief cites `review-discipline.md`.
   - **Acceptance:** `review-discipline.md` exists and is cited by every file in
     `_shared/craft/goldfish/`; `prose-style.md` contains a "No flattery"
     section; the banned-adjective list is in exactly one place
     (review-discipline.md).
   - **Acceptance scenario:** `happy path review-discipline-cited` →
     `acceptance.feature`.

6. **R6 — Reports are committed artifacts in the change folder.**
   - **Current:** the single Critic pass has no committed report artifact.
   - **Target:** each stage writes its findings to
     `.skillgrid/specs/<topic>/review/<stage>.md` (`zoom-out.md`,
     `comprehension.md`, `critic.md`). A failed/timed-out reviewer is recorded
     as `not run` (never counted as zero findings) and retried once. The review
     is committed in the spec zone (spec-before-code zone guard).
   - **Acceptance:** after a review of a ≥3-task blueprint, the three report
     files exist (or a `not run` marker for a failed stage) and are committed
     before `acceptance.feature` is written.
   - **Acceptance scenario:** `happy path review-artifacts-committed` →
     `acceptance.feature`.

7. **R7 — Disposition + convergence cap.**
   - **Current:** `receiving-code-review` triage (fix now / defer / human look /
     noise) is applied to the single Critic pass, capped at one re-review.
   - **Target:** each finding is dispositioned with the sharper 4-way taxonomy
     **CONTEST** (inaccurate; cite evidence the reviewer could read) /
     **ESCALATE** (inaccurate but evidence is out-of-doc) / **PROPOSE FIX**
     (valid; name the edit) / **DECLINE** (nit; record, don't fix). Per stage,
     max 1 re-audit after a fix; after the cap, residual findings become
     **accepted trade-offs** recorded in the blueprint, and the review advances.
     CONTEST/ESCALATE/PROPOSE-FIX/DECLINE is documented in
     `references/goldfish-review.md` (not a new skill).
   - **Acceptance:** the reference file defines the 4-way taxonomy and the
     re-audit cap; a blueprint carrying residual findings has an
     "accepted trade-offs" note rather than an open-ended loop.
   - **Acceptance scenario:** `happy path disposition-and-cap` →
     `acceptance.feature`.

8. **R8 — SKILL.md stays within tier and thin.**
   - **Current:** `writing-blueprints` SKILL.md is 26KB of a 38KB (`large`)
     ceiling; it already inlines Self-Review, Owed-Decision Gate, Ratify.
   - **Target:** the Goldfish protocol detail (stage list, persona dispatch,
     report paths, convergence, taxonomy) lives in a new
     `references/goldfish-review.md`. The SKILL.md change is a pointer plus a
     short "Blueprint validation" rewrite that names the 3 stages and links the
     reference. After the change, `skill-size-budget.mjs check` still exits 0
     for `writing-blueprints`.
   - **Acceptance:** `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs
     check` exits 0; the SKILL.md diff is net-neutral or small (the moved block
     is replaced by a pointer, not duplicated).
   - **Acceptance scenario:** `happy path skill-size-within-tier` →
     `acceptance.feature`.

9. **R9 — The after-code half is untouched.**
   - **Current:** `qa`, `requesting-code-review`, `parallel-code-review`,
     `review-ledger` own the code-review half.
   - **Target:** this change modifies only `writing-blueprints` (SKILL.md +
     `references/`), `_shared/craft/` (new `goldfish/` dir + `review-discipline.md`),
     and `_shared/rules/prose-style.md`. It does not edit `qa`, `requesting-code-
     review`, `parallel-code-review`, or `review-ledger`, and does not add a phase
     or resume marker to `sdd-structure.md`.
   - **Acceptance:** `git diff --name-only` for the change lists only the
     files in R9's scope; `sdd-structure.md` is not modified.
   - **Acceptance scenario:** `happy path after-code-untouched` →
     `acceptance.feature`.

10. **R10 — Findings feed forward to `qa` without merging the gates.**
    - **Current:** a blueprint review finding resolved "defer" is a note in the
      blueprint's Global Constraints; `qa` does not see it as decision-debt.
    - **Target:** a CRITICAL Goldfish finding that is dispositioned **parked**
      (not fixed, not waived, not contested-away) is recorded in the blueprint
      as an open design gap. `qa`'s existing decision-debt path
      (which already surfaces `**Status: ASSUMED**` blueprints as a WARNING)
      also surfaces an open parked blueprint-review finding as a WARNING. This
      is the only link between the two seams — it does not move the review into
      `qa`.
    - **Acceptance:** a blueprint with one parked CRITICAL finding, when `qa`
      runs, shows a WARNING naming the parked finding and its path; the gate
      verdict is otherwise unaffected (WARNING, never CRITICAL).
    - **Acceptance scenario:** `happy path parked-finding-surfaced-at-qa` →
      `acceptance.feature`.

## Implementation Decisions

- **Modules to build/modify:**
  - NEW `.agents/skills/_shared/craft/goldfish/{zoom-out,comprehension,
    design-critic,implementation-readiness,codebase-details}.md` (persona briefs).
  - NEW `.agents/skills/_shared/craft/review-discipline.md`.
  - NEW `.agents/skills/planning/writing-blueprints/references/goldfish-review.md`
    (the protocol: stages, dispatch, report paths, taxonomy, convergence).
  - MODIFY `.agents/skills/planning/writing-blueprints/SKILL.md` (rewrite the
    "Blueprint validation" block to name the 3 stages + link the reference;
    keep it thin).
  - MODIFY `.agents/skills/_shared/rules/prose-style.md` (add "No flattery"
    section pointing at `review-discipline.md`).
  - MODIFY (light) `.agents/skills/verification/qa/SKILL.md` decision-debt
    paragraph — surface a parked blueprint-review finding as a WARNING.
    (This is the one R9-scoped file `qa` edit; it is additive, one sentence,
    and does not change the four-state gate.)
- **Interfaces (module names / shapes, no file paths):**
  - Persona brief contract: each file is markdown, ≤150 lines, with sections
    `# <Role>`, `## Isolation mandate`, `## Audit vectors`, `## Output format`.
    Output format is the finding schema below.
  - Finding schema (all stages): `Severity: Blocker|Major|Nit` /
    `Affected scope: <heading or line range>` / `Description` / `Rationale`.
  - Dispatch contract: the orchestrating skill spawns one `reviewer` subagent
    (fallback `general`) per stage, prompt = persona brief path + target path
    + reading list + "Mode: REVIEW". The subagent is read-only (no edits).
- **Data flow:** `blueprint.md` (+ `briefing.md` for the what-side, + the files
  they reference) → per-stage fresh subagent → `.skillgrid/specs/<topic>/review/
  <stage>.md` → disposition (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE) → blueprint
  edits (PROPOSE-FIX) or accepted-trade-off note (residual) → commit → Execution
  Handoff. A parked CRITICAL → `qa` WARNING (R10).
- **Error handling:** a reviewer that times out / returns no report → retry
  once → record `not run` (never zero findings). A stage whose reviewer is
  `not run` caps that stage's verdict at "not verified," not "clean."
- **Dependencies:** no new third-party dependency. Prose + markdown only.

## Testing Decisions

- **What makes a good test:** these are prose artifacts consumed by an agent,
  not compiled code. The "tests" are structural checks: files exist, are within
  line/byte budgets, cite the right references, and the SKILL.md pointer
  resolves. There is no runtime to exercise.
- **Modules to test:** the presence + shape of the new reference/persona files
  (a structural lint), and `skill-size-budget.mjs check` (the existing script)
  as the size gate. The persona *behavior* (a fresh agent actually finding a
  planted gap) is verified by a one-off manual eval, not a committed test —
  it is the skill-creator eval loop, run at review time, not in CI.
- **Prior art:** `skill-size-budget.mjs` is the precedent for a committed
  structural check on skill files.
- **Edge cases:** a blueprint with 1-2 tasks and no one-way-door (review
  skipped, per the existing Plan Review rule); a failed reviewer (`not run`);
  a blueprint that already has an `ASSUMED` block (R10 must not double-count
  the decision-debt WARNING).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None
- `.skillgrid/ASSUMPTIONS.md`: add one ADR pointer row under `§ LOCKED` —
  "before-code plan review is a fresh-context multi-persona Goldfish review
  (subflow of writing-blueprints), not a new pipeline phase and not part of
  qa." Record: `artifacts/04-adr-00XX-goldfish-plan-review.md`.
- `.skillgrid/ARCHITECTURE.md`: None (no new component/data store; the change
  is to the planning skill's reference set).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.85  | 0.75 | ok     | EGM port scope is explicit          |
| Boundary Clarity    | 0.80  | 0.70 | ok     | before-code only; qa untouched      |
| Constraint Clarity  | 0.78  | 0.65 | ok     | tier ceiling, no new dep, no phase  |
| Acceptance Criteria | 0.82  | 0.70 | ok     | all 10 requirements falsifiable     |
| **Clarity**         | 0.15  | ≤0.20| ok     |                                      |

**Interview log:**

| Round | Question summary                  | Decision locked                              |
|-------|-----------------------------------|----------------------------------------------|
| 1     | Is this a new skill or a subflow? | Subflow of writing-blueprints; no new phase  |
| 2     | Which EGM skills/personas to port?| 5 personas + 2 folded; drop synthesizer, bootstrap |
| 3     | Where does the review sit?        | Between Self-Review and Execution Handoff; not qa |
| 4     | How do personas run in OpenCode?  | Persona brief files read by reviewer/general subagent |

## Open Questions & Assumptions

- **Assumption:** the `reviewer` subagent can be told "read this persona file
  first, then review the target" and will honor read-only. (EGM's own fallback
  assumes the same. If the harness's `reviewer` is too opinionated, fall back
  to `general` — the brief names both.)
- **Question (open, non-blocking):** whether to also run the comprehension
  reviewer over `briefing.md` (the what-side) or only `blueprint.md`. Default
  for this change: `blueprint.md` only; `briefing.md` review is an optional
  addition the critic stage may note but not block on.

## Decisions (ADR)

- `artifacts/04-adr-00XX-goldfish-plan-review.md` — before-code plan review is
  a fresh-context multi-persona Goldfish subflow of writing-blueprints; the
  after-code half (qa / code review) is a separate, unchanged gate.

## Terms

- **Goldfish review:** a design-document review performed by a fresh,
  zero-context subagent that has seen only the target artifact and the files it
  references (contrast: a same-context review that inherits the author's
  assumptions). Link: see ADR above.
- **Elephant:** the persistent author context that wrote the document (the
  session's own context). Goldfish reviewers are deliberately not the Elephant.
- No new terms added to the terms files; the two above are local to this change
  and defined here.
