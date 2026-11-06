# EGM Goldfish: Shootdown + Plan Review + Drafting Disciplines + Drift Report — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-11-02)

> **Role:** the *what* — validated design/spec: problem, intent, requirements,
> success criteria. The companion `blueprint.md` is the *how*. Do not rename.

## Problem / Intent

This change ports the **Elephant-Goldfish Model (EGM)** onto skillgrid in four
tracks. Tracks A and B address the same root failure mode — **same-context
rubber-stamping** — at two different points in the pipeline. Track C addresses
a different failure mode — **one-shot drafting without incremental
verification** — at the same pipeline stage as Track B (blueprint writing) but
at a different seam (the drafting process, not the validation seam). Track D
addresses a fourth failure mode — **post-implementation documentation drift** —
at a different point in the pipeline entirely (after code, periodic, not a
per-change gate).

**Track A — Brainstorm shootdown (before design, on the anchored hypothesis).**
skillgrid's `brainstorming` skill is a structured 4-path classifier with a hard
approval gate, but it has **no adversarial review of the idea itself** before the
design is committed. The author and the user design together in the same
context, which means sycophancy risk: the agent affirms the user's premise
rather than stress-testing it. EGM's `egm-brainstorm` skill solves this by
spawning six parallel "Goldfish" subagents that attack the anchored hypothesis
from six conflicting lenses (YAGNI, cost economics, UX, migration,
observability, tech debt) before the design is written. Track A ports this
shootdown phase into `brainstorming`'s New Project / New Function paths,
inserted after the codebase feasibility check and before writing the spec.

**Track B — Plan review (before code, on `blueprint.md`).**
skillgrid's before-code review of a `blueprint.md` is thin: a same-context
Self-Review (the author grading its own plan) plus a single generic "Critic
subagent" pass and one structural Plan Review. All three run with the author's
assumptions already in context, so they rubber-stamp the very blind spots they
are meant to catch — unstated assumptions, terms a fresh reader cannot
resolve, altitude drops, and gaps that make the plan unbuildable by a
zero-context implementer. Track B ports EGM's `egm-review` + `egm-review-loop`
+ `egm-resolve` as a **new top-level skill** (`skillgrid:goldfish-review`) with
a 5-stage fresh-context pipeline, 4 execution presets, a multi-round
sharpening loop (convergence = 2 consecutive clean rounds, cap = 5), and an
Elephant-side resolution workflow (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE with
interactive gate). `writing-blueprints` invokes `skillgrid:goldfish-review` on
`blueprint.md` in its validation seam — the skill is independently invocable
(review any design doc, not just blueprints), but in this change it is
invoked by `writing-blueprints`. The sharpening loop and resolution workflow
are reference files inside `goldfish-review`, not separate skills — they are
tightly coupled to the orchestrator and not independently useful.

**Track C — Drafting disciplines (during blueprint writing, on the
drafting process).**
skillgrid's `writing-blueprints` skill writes the blueprint in one shot: the
agent drafts the entire plan, then runs Self-Review + Critic + User Review.
EGM's `egm-draft` skill writes the design document **section-by-section** with
per-section approval gates: draft a section → user approves → next section.
This catches a wrong interface, a missing file, or a bad test strategy *before*
the agent writes 15 more tasks. Track C ports this section-by-section model
into `writing-blueprints`, along with eight other drafting disciplines from
`egm-draft`: explicit ingestion, spec reconciliation, owner-decides-
architecture, alternatives-from-briefing, anti-sycophancy, artifact-first
delivery, and plain-English.

**Track D — Drift report (after code, periodic, on the codebase vs design
docs).**
skillgrid has no mechanism to detect that the codebase has diverged from its
design documents after implementation. Quick point-fixes, untracked refactors,
and AI-agent edits leave `briefing.md`, `blueprint.md`, and ADRs outdated. When
a human or AI agent later consults an outdated spec to design a new feature,
they make flawed assumptions that lead to broken implementations. EGM's
`egm-drift-report` skill solves this by auditing checked-in design documents
against source code to identify architectural divergence in four categories
(Missing Features, Incomplete Models, Logic Shortcuts, Architectural
Divergence). Track D ports this as a new `skillgrid:drift-report` skill — a
periodic ritual (not a per-change gate) that compares spec artifacts against
the codebase and produces an actionable drift report. It reuses
`goldfish-review`'s resolution workflow (R10) for findings.

**The four tracks are complementary gates:**
- Track A (shootdown) asks **"is this idea worth building at all?"** before any
  design is committed — it validates the *premise*.
- Track B (plan review) asks **"will this plan, if followed, produce the right
  thing?"** before any code is written — it validates the *plan*.
- Track C (drafting disciplines) asks **"was this plan built the right way?"**
  during blueprint writing — it validates the *process*.
- Track D (drift report) asks **"does the code still match the design?"**
  after the code is written, periodically — it validates the *synchronization*.
- `qa` asks **"did the code produce the right thing?"** after the code is
  written — it validates the *code*.

All four are subflows or rituals at different pipeline stages. None replaces
the others.

## Purpose & Success Criteria

- **Purpose:** give `brainstorming` a fresh-context, multi-lens shootdown
  (Track A), create a new `skillgrid:goldfish-review` skill with a 5-stage
  fresh-context pipeline + sharpening loop + resolution workflow (Track B),
  and give `writing-blueprints` section-by-section drafting with per-section
  approval gates plus eight drafting disciplines (Track C). Track B is a new
  top-level skill (independently invocable, reusable across design docs);
  Tracks A and C are subflows of existing skills. No new pipeline phase.
- **Success criteria:**
  - **Track A:** a New Project or New Function brainstorm spawns at least six
    **fresh** subagents (one per lens) that were not given the author's
    conversation context, after the codebase feasibility check and before
    `briefing.md` is written. Blocker findings must be dispositioned (resolved /
    accepted-with-note / downgraded-to-non-goal) before the spec is written.
  - **Track B:** a blueprint with ≥3 tasks (or any one-way-door) is reviewed by
    `skillgrid:goldfish-review` — a 5-stage fresh-context pipeline (zoom-out /
    comprehension / critic / intermediate zoom-out / implementation-readiness)
    with a multi-round sharpening loop (convergence = 2 consecutive clean
    rounds, cap = 5) and an Elephant-side resolution workflow, before
    `acceptance.feature` is written. The comprehension reviewer's paraphrase
    of the blueprint is written to an artifact; any section it could not
    paraphrase, or any term it had to guess, is a recorded finding. The review
    is independently invocable (`/goldfish-review <path>`) but is invoked by
    `writing-blueprints` in this change.
  - **Track C:** the blueprint is drafted section-by-section with per-section
    approval gates. Each task in the Tasks section gets its own approval gate.
    The agent reconciles the blueprint against `briefing.md` (contradictions +
    significant additions) before Self-Review. Architectural decisions (data
    model, component boundaries, storage, protocol) are presented to the owner
    without ranking or default — the owner picks.
  - **Track D:** `/drift-report <spec-dir>` (per-change) or `/drift-report`
    (project-wide) compares spec artifacts (`briefing.md`, `blueprint.md`,
    `adr.md`, `acceptance.feature`) against the codebase and produces a drift
    report with findings in 4 categories (Missing Features, Incomplete Models,
    Logic Shortcuts, Architectural Divergence). Findings are dispositioned
    using `goldfish-review`'s resolution workflow (R10). The report is
    committed to the spec dir. It is a periodic ritual, not a per-change gate.
  - Review findings are dispositioned and the review is committed in the spec
    zone before execution handoff (Track B) or before `writing-blueprints` is
    invoked (Track A).
  - `qa`, `requesting-code-review`, `review-ledger`, and `parallel-code-review`
    are **unchanged** — the after-code half is untouched.
- **Out of scope:**
  - No change to `qa`, the after-code quality gate.
  - One new top-level skill (`skillgrid:goldfish-review`) — Track B. It is
    independently invocable and reusable across design docs, not just
    blueprints. Tracks A and C are subflows of existing skills.
  - No separate `goldfish-review-loop` or `goldfish-resolve` skill — they are
    reference files inside `goldfish-review` (tightly coupled to the
    orchestrator, not independently useful).
  - No `egm-bootstrap` (per-directory README contracts), no separate
    `concepts_brief.md` artifact, no synthesizer subagent-of-subagents layer.
  - No change to the blueprint template's task structure (Files / Interfaces /
    SATISFIES / Steps with actual code) — the blueprint is the *how*, code is
    correct. Track C changes the *approval cadence*, not the task format.
  - No `design_draft.md` working file — `writing-blueprints` writes
    `blueprint.md` directly. A separate draft file would be a second source of
    truth.
  - No "no code" rule in the blueprint (unlike `egm-draft`'s design doc) — the
    blueprint's task steps include actual code (failing test, implementation,
    commit) by design.
  - No enforcement of drift-report cadence (weekly gardener rotation is
    suggested, not required). The skill is invocable on-demand.
  - No automatic drift detection in CI (the drift report is an agent-driven
    ritual, not a CI check).
  - No new third-party dependency (prose-only change; the only code is skill
    markdown + reference docs).

## Context

- `brainstorming` (`.agents/skills/planning/brainstorming/SKILL.md`) is the
  skill Track A upgrades. Its New Project / New Function paths already have the
  seam: interview → approaches → design → **codebase feasibility check** → ADR
  manifest → acceptance contract → spec. Track A inserts the shootdown between
  the feasibility check and the ADR manifest. The 6-lens Goldfish swarm
  replaces nothing — it is additive. The existing structured `question` tool
  mandate (used by `interviewing`) stays intact; the anti-sycophancy rules apply
  to *tone*, not *tooling*.
- `writing-blueprints` (`.agents/skills/planning/writing-blueprints/SKILL.md`,
  `large` tier, 26KB) is the skill Tracks B and C upgrade. Track B replaces the
  validation seam: **Self-Review** → **Critic subagent** → **Owed-Decision Gate**
  → **Plan Review** (`references/plan-review.md`) becomes **Self-Review** →
  **Owed-Decision Gate** → **`skillgrid:goldfish-review`** → **Execution
  Handoff**. The single generic Critic subagent and the structural Plan Review
  are replaced by the 5-stage `goldfish-review` skill. Track C upgrades the
  drafting process: the one-shot model is replaced by section-by-section
  drafting with per-section approval gates, plus explicit ingestion, spec
  reconciliation, owner-decides-architecture, and anti-sycophancy rules.
- `skillgrid:goldfish-review` (NEW, `.agents/skills/planning/goldfish-review/`)
  is the new top-level skill for Track B. It is independently invocable
  (`/goldfish-review <path>`) and reusable across design docs (blueprints,
  briefings, ADRs). It contains: `SKILL.md` (5-stage pipeline + 4 presets +
  checkpoint gates + pause/resume), `references/goldfish-review-loop.md`
  (sharpening loop: multi-round, convergence, settled/fixed tracking, churn,
  turn-taking), `references/goldfish-resolve.md` (Elephant-side resolution:
  CONTEST/ESCALATE/PROPOSE-FIX/DECLINE categorization rules + interactive gate
  + report format). It is invoked by `writing-blueprints` in the validation
  seam, but can also be invoked standalone.
- `references/plan-review.md` (existing) is **replaced** by
  `skillgrid:goldfish-review`. Its 6 checks (failure modes / scope /
  feasibility / ordering / test strategy / ADR compliance) are subsumed by the
  5-stage pipeline. The file is deleted (or marked superseded) in this change.
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
- EGM source: `github.com/google/elephant-goldfish-model` (Apache-2.0).
  Personas ported (shared by both tracks): `zoom_out`, `comprehension`,
  `design_critic`, `implementation_readiness`, `codebase_details`,
  `structural`. Folded into `review-discipline.md`: `style`. Dropped:
  `synthesizer` (the main agent synthesizes inline).
  Track A (shootdown) uses all 6 lenses. Track B (plan review) uses 3 stages
  (zoom-out / comprehension / critic = design-critic + implementation-readiness
  + codebase-details). Track C (drafting disciplines) ports `egm-draft`'s
  section-by-section model + 8 drafting disciplines; it uses no Goldfish
  personas (it is a process change, not a review change).

## Approaches Considered

- **Chosen:** Three-track upgrade. Track A inserts a 6-lens Goldfish shootdown
  into `brainstorming`'s New Project / New Function paths (after feasibility
  check, before spec). Track B creates a new `skillgrid:goldfish-review`
  skill (5-stage pipeline + 4 presets + sharpening loop + resolution
  workflow) that `writing-blueprints` invokes in its validation seam. Track C
  upgrades the `writing-blueprints` drafting process with section-by-section
  drafting + 8 drafting disciplines from `egm-draft`. Tracks A and B share a
  `_shared/elephant-goldfish/` persona-brief directory and one shared
  `review-discipline.md`. The `goldfish-review` skill contains its own
  `references/goldfish-review-loop.md` and `references/goldfish-resolve.md`
  (not separate skills — tightly coupled to the orchestrator). SKILL.md edits
  stay thin; protocol detail lives in `references/shootdown.md` (Track A),
  `goldfish-review/` (Track B), and `references/section-by-section.md`
  (Track C).
- **Rejected:** Add a new top-level `goldfish-review` skill for Track B —
  duplicates the seam, adds a phase and a resume marker, and forces a change to
  `sdd-structure.md`'s phase table. The review is a subflow of blueprint
  validation, not a new pipeline phase.
- **Rejected:** Add a new top-level `goldfish-brainstorm` skill for Track A —
  same reasoning. The shootdown is a subflow of `brainstorming`'s existing New
  Project / New Function paths, not a new pipeline phase.
- **Rejected:** Add the shootdown to `qa` — wrong time and wrong target (code,
  not idea). Would conflate the before-design, before-code, and after-code
  failure classes.
- **Rejected:** Register new named OpenCode subagents per persona — the harness
  does not make that cheap, and EGM's documented fallback (general-purpose
  subagent + "read this persona file first") is the faithful, lower-friction
  port.
- **Rejected:** Port EGM's `concepts_brief.md` as a separate artifact —
  skillgrid already has `briefing.md` as the spec. A second artifact would be
  a second source of truth. The shootdown findings land in `briefing.md` as a
  new `## Shootdown Findings` section.
- **Rejected:** Port EGM's `synthesizer` subagent — the main agent (the
  Elephant) synthesizes inline. A subagent-of-subagents layer adds a dispatch
  round-trip without adding value; the main agent already has the conversation
  context to dedupe and group findings.
- **Rejected:** Port EGM's `design_draft.md` as a working file for Track C —
  skillgrid writes `blueprint.md` directly. A separate draft file would be a
  second source of truth. The section-by-section model writes each section
  directly to `blueprint.md`, not to a scratch file.
- **Rejected:** Port EGM's "no code" rule to the blueprint — the blueprint is
  the *how* and its task steps include actual code (failing test,
  implementation, commit) by design. `egm-draft`'s "no code" rule applies to
  the *design doc* (the *what*), which skillgrid already splits into
  `briefing.md`. Porting the rule would strip code from the blueprint's task
  steps, making them unbuildable.
- **Rejected:** Port EGM's target directory probe + SITEMAP.md — skillgrid has
  its own spec directory convention (`.skillgrid/change-specs/YYYY-MM-DD-<topic>/`).
  No SITEMAP.
- **Rejected:** Port EGM's Critical User Journeys (CUJs) section — already
  covered by `acceptance.feature` (BDD scenarios).
- **Rejected:** Port EGM's Open Questions section to the blueprint — already
  covered by the briefing's `Open Questions & Assumptions` + the blueprint's
  `ASSUMED` block.
- **Rejected:** Make the drift report a separate spec — the user chose Track D
  in this spec. The drift report shares the resolution workflow (R10) and the
  persona files (R11) with Track B, keeping it in one spec makes the
  dependencies explicit. It is a different failure mode (post-implementation,
  periodic) but the change is cohesive: "port EGM onto skillgrid."

## Requirements

### Track A — Brainstorm Shootdown

1. **R1 — Six fresh-context lenses (the load-bearing rule).**
   - **Current:** `brainstorming`'s New Project / New Function paths have no
     adversarial review of the anchored hypothesis before the design is
     committed. The author and user design together in the same context.
   - **Target:** after the codebase feasibility check passes and before the ADR
     manifest is written, `brainstorming` spawns **six parallel read-only
     subagents**, one per lens:
     (a) **Contrarian / YAGNI** — *Can existing code/libs solve 90% of this
     without new code?*
     (b) **Cost, Quota & Token Economics** — *Peak QPS, memory, storage, token
     budget at scale?*
     (c) **UX & Developer Ergonomics** — *Is the CLI/API intuitive? Are errors
     actionable?*
     (d) **Migration & Compatibility** — *How do existing clients/data migrate?
     Schema breaks?*
     (e) **Observability & SRE** — *Can oncall triage this at 3am? Are metrics
     traceable?*
     (f) **Tech Debt & Maintenance** — *Tightly-coupled state machines that
     hurt in 2 years?*
     Each subagent is a freshly spawned `reviewer` (fallback `general`) given
     ONLY the anchored hypothesis + the relevant existing code/files — no
     conversation history, no author narrative. A review run in the author's
     own context is not a Goldfish shootdown and MUST NOT count toward
     satisfying R1.
   - **Acceptance:** the reviewer prompt names the anchored hypothesis and the
     reading list; the prompt contains no summary of the brainstorming
     conversation; the subagent is a freshly spawned one (not the session's own
     context); all six are dispatched in parallel (not sequentially).
   - **Acceptance scenario:** `happy path six-lens-shootdown` →
     `acceptance.feature`.

2. **R2 — Shootdown runs on New Project / New Function only.**
   - **Current:** `brainstorming` has four paths (prototype / bounded / new
     project / new function), each with its own checklist.
   - **Target:** the shootdown is inserted into the New Project / New Function
     checklists only. Bounded (one-file small) and prototype (already a probe)
     skip it. The insertion point is after the codebase feasibility check
     passes and before the ADR manifest is written. The path ratchet is
     unchanged: if a bounded task upgrades to New Function mid-task, the
     shootdown becomes mandatory at the new path.
   - **Acceptance:** `SKILL.md`'s New Project and New Function path
     descriptions name the shootdown; the Bounded and Prototype descriptions do
     not; `references/paths.md`'s checklists reflect the insertion point.
   - **Acceptance scenario:** `happy path shootdown-path-scope` →
     `acceptance.feature`.

3. **R3 — Blocker findings must be dispositioned before the spec is written.**
   - **Current:** no mechanism to disposition shootdown findings before the
     spec is committed.
   - **Target:** after the six lenses report, the main agent (the Elephant)
     synthesizes inline: dedupes overlapping findings, groups by severity
     (🔴 Blocker / 🟡 Major / 🟢 Nit). The user is shown the 🔴 Blockers and
     asked to disposition each: **resolved** (design revised, loop back to
     feasibility check), **accepted-with-note** (recorded in the spec's
     `## Shootdown Findings` section as a known risk), or **downgraded-to-
     non-goal** (moved to the spec's Out of Scope). 🟡 Major and 🟢 Nit
     findings are recorded but do not block. The spec is NOT written until all
     🔴 Blockers are dispositioned.
   - **Acceptance:** a blueprint with an undispositioned 🔴 Blocker finding
     does not have a committed `briefing.md`; a blueprint with all 🔴
     Blockers dispositioned has a `## Shootdown Findings` section in
     `briefing.md` with the severity table and dispositions.
   - **Acceptance scenario:** `happy path blocker-disposition-gate` →
     `acceptance.feature`.

4. **R4 — Shootdown findings land in `briefing.md` as a new section.**
   - **Current:** `briefing.md` has no section for adversarial review findings.
   - **Target:** `templates/briefing.md` gains a `## Shootdown Findings`
     section between "Implementation Decisions" and "Testing Decisions". The
     section contains: (1) the anchored hypothesis (2-3 sentences, as
     confirmed by the user at the readiness gate), (2) the severity table
     (🔴/🟡/🟢 with lens, description, disposition), (3) a one-line summary of
     the overall verdict (survived / survived-with-notes / needs-revision).
     No separate `concepts_brief.md` artifact — `briefing.md` is the single
     source of truth.
   - **Acceptance:** `templates/briefing.md` contains the `## Shootdown
     Findings` section with the three sub-parts; a filled-in `briefing.md` for
     a New Project / New Function change has the section populated (or an
     explicit "Skipped: bounded/prototype path" note for those paths).
   - **Acceptance scenario:** `happy path shootdown-findings-in-briefing` →
     `acceptance.feature`.

5. **R5 — Anti-sycophancy & stress-testing rules apply to the shootdown
   conversation.**
   - **Current:** `brainstorming`'s tone is governed by `prose-style.md` (21
     rules) but has no explicit no-flattery / stress-test mandate during the
   interview and shootdown phases.
   - **Target:** `SKILL.md` gains an **Anti-Sycophancy & Stress-Testing**
     section that distills EGM's execution constraints:
     - **Default to stress-testing:** never agree by default. Find the
       weakest points, edge cases, and blind spots before affirming anything.
     - **Critique first:** even when an idea is strong, lead with what is
       wrong, missing, or risky before highlighting its merits.
     - **No glazing:** no "great," "brilliant," "smart," "correct," "perfect,"
       "excellent," or "spot-on" when referring to the user's inputs.
     - **No apologies:** no "I apologize," "I am sorry," "Forgive me" when
       challenged or corrected. Outline the corrected logic directly.
     - **Generate counter-arguments:** for every strategy proposed, actively
       outline the strongest counter-argument.
     - **Reject echo-chamber framing:** never parrot framing back to the user.
     - **Plain-English standard:** no ungrounded promotional adjectives
       (seamless, robust, blazingly fast) or decorative adverbs (seamlessly,
       effortlessly, literally, drastically, obviously).
     These rules apply to the *tone* of the interview and shootdown
     conversation. They do not change the *tooling* — the structured `question`
     tool mandate (used by `interviewing`) stays intact. The rules are cited
     from `review-discipline.md` (shared with Track B).
   - **Acceptance:** `SKILL.md` contains the Anti-Sycophancy & Stress-Testing
     section; every rule is cited from `review-discipline.md` (not restated);
     the banned-adjective list is in exactly one place (`review-discipline.md`).
   - **Acceptance scenario:** `happy path anti-sycophancy-rules` →
     `acceptance.feature`.

### Track B — Goldfish Review Skill

6. **R6 — New top-level skill: `skillgrid:goldfish-review`.**
   - **Current:** no standalone Goldfish review skill. The before-code review
     is a subflow of `writing-blueprints` (Self-Review + Critic subagent +
     Plan Review).
   - **Target:** new skill at
     `.agents/skills/planning/goldfish-review/` with:
     - `SKILL.md` — the 5-stage pipeline, 4 execution presets, checkpoint
       gates, pause/resume protocol, and execution constraints (artifact-first,
       plain-English, anti-sycophancy). Independently invocable:
       `/goldfish-review <TargetFile> [GoldfishContext] [Step]`.
     - `references/goldfish-review-loop.md` — the sharpening loop: multi-round
       iteration (up to 5), convergence rules (2 consecutive clean rounds),
       settled vs fixed finding tracking, churn detection, turn-taking
       (idle between rounds, user confirms "run another round"), unconditional
       line-range re-scan before each round.
     - `references/goldfish-resolve.md` — the Elephant-side resolution
       workflow: CONTEST/ESCALATE/PROPOSE-FIX/DECLINE categorization rules
       (contest must be checkable, don't add context for subagents, don't
       contest comprehension gaps, don't change severity), interactive gate
       (auto-accept / continue / interactive breakdown / comment on file /
       custom instruction), report format (`goldfish_review_report.md` with AI
       suggestion + justification + proposed edit per finding).
     The loop and resolve are reference files (not separate skills) — they are
     tightly coupled to the orchestrator and not independently useful.
     `goldfish-resolve` runs in the Elephant context (the main agent), not a
     subagent — it needs institutional knowledge to write good fixes.
   - **Acceptance:** `SKILL.md` exists with the 5-stage pipeline + 4 presets;
     `references/goldfish-review-loop.md` exists with convergence rules;
     `references/goldfish-resolve.md` exists with categorization rules +
     interactive gate; the skill is invocable via `/goldfish-review <path>`.
   - **Acceptance scenario:** `happy path goldfish-review-skill` →
     `acceptance.feature`.

7. **R7 — Five-stage fresh-context pipeline.**
   - **Current:** one generic Critic subagent covers "architectural gaps,
     missing edge cases, untested assumptions, scope creep, unclear interfaces"
     in a single pass, plus a structural Plan Review.
   - **Target:** the single pass is replaced by a 5-stage pipeline:
     (a) **Stage 5 — Initial Zoom Out** (full `blueprint.md`): YAGNI, reuse,
     anti-weeds, altitude → `zoom_out_report.md`;
     (b) **Stage 6 — Comprehension Test** (`blueprint.md` + referenced files):
     paraphrase first, self-containment → `comprehension_report.md`;
     (c) **Stage 7 — Critic Review** (`blueprint.md`, the *how*): design-
     critic + structural + style → `critic_report.md`;
     (d) **Stage 8 — Intermediate Zoom Out** (full `blueprint.md`, re-read
     after Stages 6+7 fixes): catches altitude drift introduced by fixes →
     `intermediate_zoom_out_report.md`;
     (e) **Stage 9 — Implementation Readiness** (`blueprint.md` task steps +
     codebase): actionability, codebase accuracy → `implementation_readiness_
     report.md`.
     Each stage dispatches fresh `reviewer` (fallback `general`) subagents
     given ONLY the target file path(s) and the files that target references —
     no conversation history, no author narrative. A review run in the
     author's own context is not a Goldfish review and MUST NOT count toward
     satisfying R7. Stage 8 is the key addition over the old 3-stage model:
     it re-checks the whole doc after Stages 6+7 findings are addressed,
     catching weeds or scope creep introduced by fixes.
   - **Acceptance:** `goldfish-review/SKILL.md` names all five stages and their
     targets; the stage list is not a single combined prompt; Stage 8 re-runs
     zoom-out on the updated doc after Stage 7 remediation.
   - **Acceptance scenario:** `happy path five-stage-pipeline` →
     `acceptance.feature`.

8. **R8 — Four execution presets.**
   - **Current:** the review always runs the full pass. No mode selection.
   - **Target:** `goldfish-review` supports 4 presets, selected by the user or
     inferred from the invocation context:
     - **Interactive** (checkpoint after each stage): full 5-stage pipeline
       with user checkpoint gates between stages. Default for New Project /
       New Function blueprints.
     - **Consolidated** (no pauses): full 5-stage pipeline, no checkpoint
       gates, findings consolidated into `consolidated_review_report.md` with
       cross-stage dedup + traceability matrix (main agent synthesizes inline
       — there is no synthesizer subagent). For time-constrained reviews.
     - **Feature-spec-only** (stages 5-8, skip 9): for blueprints that are
       architecture-heavy but have straightforward task steps. Skips
       implementation readiness.
     - **Implementation-only** (stage 9 only): for blueprints where the
       architecture is already validated (e.g., a bounded task that was
       upgraded) and only the task steps need checking.
     The preset is a single-select gate (if not provided at invocation). The
     topological dependency 5→6→7→8→9 is preserved — multi-select is
     prohibited.
   - **Acceptance:** `goldfish-review/SKILL.md` documents all 4 presets; the
     preset gate is single-select; the topological order is preserved.
   - **Acceptance scenario:** `happy path four-presets` →
     `acceptance.feature`.

9. **R9 — Sharpening loop (multi-round with convergence).**
   - **Current:** the Critic pass runs once. Max 1 re-review. No convergence
     tracking.
   - **Target:** each stage runs in a **sharpening loop** (documented in
     `references/goldfish-review-loop.md`):
     - **Multi-round:** up to 5 rounds per stage. Each round: spawn
       reviewers → collect findings → resolve (categorize + fix) → re-scan
       line ranges.
     - **Convergence:** 2 consecutive **clean rounds** (no new Blocker or
       Major finding that is not settled). One clean round is not enough — a
       fresh reviewer can find a different gap in the same text.
     - **Settled findings:** contested or accepted-without-fix. Kept across
       rounds. If a new reviewer raises a settled finding again (same scope,
       same issue), do NOT re-open it.
     - **Fixed findings:** an applied fix does NOT settle a finding. If a later
       reviewer raises the same issue at the same scope, the fix failed.
       Re-open.
     - **Churn:** a new finding is at the scope of an applied fix, and
       reverting that fix would resolve it. On churn, ask the user: accept
       and proceed / continue past the cap / return to drafting.
     - **Turn-taking:** after each round's resolution, the loop goes idle and
       waits for the user to confirm "run another round." The agent does not
       auto-loop.
     - **Line-range re-scan:** before each round, unconditionally re-read
       `blueprint.md` to recalculate line numbers (they shift after edits).
   - **Acceptance:** `references/goldfish-review-loop.md` documents all 6
     rules; the loop does not auto-advance (turn-taking); convergence requires
     2 clean rounds; settled findings are not re-opened.
   - **Acceptance scenario:** `happy path sharpening-loop-convergence` →
     `acceptance.feature`.

10. **R10 — Elephant-side resolution workflow.**
    - **Current:** `receiving-code-review` triage (fix now / defer / human look
      / noise) is applied to the Critic pass. No interactive resolution gate.
    - **Target:** each round's findings are resolved by the **Elephant** (the
      main agent, not a subagent) per `references/goldfish-resolve.md`:
      - **Categorize** each finding into exactly one: **CONTEST** (inaccurate;
        cite evidence the reviewer could read) / **ESCALATE** (inaccurate but
        evidence is out-of-doc) / **PROPOSE FIX** (valid; name the edit) /
        **DECLINE** (nit; record, don't fix).
      - **Rules:** a contest must be checkable ("this was intentional" is not
        evidence). Do not add context only to satisfy a reviewer (if only a
        zero-context subagent needs it → ESCALATE). Do not contest a
        comprehension gap. Do not change a severity. You wrote the doc — a wish
        to contest is not evidence the reviewer is wrong.
      - **Interactive gate** (user picks one): **Auto accept** (batch-apply all
        PROPOSE FIX) / **Continue** (no changes) / **Interactive breakdown**
        (item-by-item walkthrough: Accept / Skip / Discuss per item) /
        **Comment on file** (user edits the report directly) / **Custom
        instruction**.
      - **Report:** `goldfish_review_report.md` with per-finding AI suggestion
        + justification + proposed edit.
      The resolution runs in the Elephant context — the agent has the
      institutional knowledge to write good fixes and defend design choices.
   - **Acceptance:** `references/goldfish-resolve.md` documents the 4
     categories + 4 rules + 5 gate options; the resolution runs in the main
     agent context (not a subagent); the report format matches the spec.
   - **Acceptance scenario:** `happy path elephant-resolution` →
     `acceptance.feature`.

11. **R11 — Persona briefs live in `_shared/elephant-goldfish/`.**
    - **Current:** no shared reviewer-persona files.
    - **Target:** new files
      `_shared/elephant-goldfish/{zoom-out,comprehension,design-critic,
      structural,style,implementation-readiness,codebase-details}.md`, each a
      self-contained read-only persona brief (role, isolation mandate, audit
      vectors, output format). The `style` persona is a standalone file (used
      by Stages 7 and 9). All six lenses used by Track A (shootdown) are
      present: `zoom-out` (YAGNI), `design-critic` (cost economics),
      `comprehension` (UX), `implementation-readiness` (migration),
      `codebase-details` (observability), `structural` (tech debt). Stage 7
      uses `design-critic` + `structural` + `style`. Stage 9 uses
      `implementation-readiness` + `codebase-details` + `style`.
    - **Acceptance:** each file exists, is ≤150 lines, names its audit vectors,
      and its output format matches the finding schema. No `synthesizer.md`.
    - **Acceptance scenario:** `happy path persona-briefs-present` →
      `acceptance.feature`.

12. **R12 — Review-discipline reference + anti-sycophancy rule (shared).**
    - **Current:** `prose-style.md` has 21 rules but no explicit no-flattery /
      stress-test rule; reviewer personas do not share a common discipline doc.
    - **Target:** new `_shared/craft/review-discipline.md` distills EGM's
      execution constraints (stress-test by default, critique-first, generate a
      counter-argument per assertion, no glazing, banned validating adjectives,
      no apologies, banned decorative adverbs) into one citable reference.
      `prose-style.md` gains one short "No flattery" rule that points at it.
      Every goldfish persona brief cites `review-discipline.md`. `goldfish-
      review/SKILL.md` cites it. Track A cites it.
    - **Acceptance:** `review-discipline.md` exists and is cited by every file
      in `_shared/elephant-goldfish/` AND by `goldfish-review/SKILL.md`;
      `prose-style.md` contains a "No flattery" section; the banned-adjective
      list is in exactly one place (`review-discipline.md`).
    - **Acceptance scenario:** `happy path review-discipline-cited` →
      `acceptance.feature`.

13. **R13 — Reports are committed artifacts in the change folder.**
    - **Current:** the single Critic pass has no committed report artifact.
    - **Target:** each `goldfish-review` stage writes its findings to
      `.skillgrid/change-specs/<topic>/review/<stage>.md` (`zoom_out_report.md`,
      `comprehension_report.md`, `critic_report.md`,
      `intermediate_zoom_out_report.md`, `implementation_readiness_report.md`).
      The resolution report is `goldfish_review_report.md`. The consolidated
      report (if Consolidated preset) is `consolidated_review_report.md`.
      Track A's shootdown findings land in `briefing.md`'s `## Shootdown
      Findings` section (not a separate file). A failed/timed-out reviewer is
      recorded as `not run` (never counted as zero findings) and retried once.
      The review is committed in the spec zone (spec-before-code zone guard).
      **Chat-response contract (all presets):** the agent works WITH the user,
      not just for them — a review that ends in silence plus a file path is a
      failed delivery. After the final stage (Interactive) or after
      consolidation (Consolidated), the agent posts a **consolidated chat
      response** containing: (1) stage/reviewer status table
      (`completed` / `not run`), (2) severity breakdown (🔴/🟡/🟢 counts),
      (3) every 🔴 Blocker and 🟡 Major finding in one or two lines each
      (description + affected scope + the agent's proposed disposition: fix /
      contest / decline), (4) the path to the full report
      (`consolidated_review_report.md` or the `review/` dir for the per-stage
      files). The full report is NOT dumped in chat (the artifact-first rule
      still holds — this is a working summary, not a paste of the file). The
      same contract applies to Track A's shootdown synthesis: the chat shows
      the severity table + each 🔴 Blocker with its disposition options, not
      just a pointer at `briefing.md`.
    - **Acceptance:** after a `goldfish-review` of a ≥3-task blueprint, the
      stage report files exist (or a `not run` marker for a failed stage) and
      are committed before `acceptance.feature` is written. After a Track A
      shootdown, `briefing.md`'s `## Shootdown Findings` section is populated
      and committed. After any review run, the chat turn ends with the
      consolidated response (status + severity + Blocker/Major findings +
      report path); a turn that ends with only a file path does not satisfy
      this requirement.
    - **Acceptance scenario:** `happy path review-artifacts-committed` →
      `acceptance.feature`.

14. **R14 — `writing-blueprints` invokes `skillgrid:goldfish-review`.**
    - **Current:** `writing-blueprints` has its own validation block (Self-
      Review + Critic subagent + Plan Review).
    - **Target:** the validation block is rewritten: **Self-Review** →
      **Owed-Decision Gate** → **`skillgrid:goldfish-review`** (invoked on
      `blueprint.md`) → **User Review Gate** → **Execution Handoff**. The
      single generic Critic subagent and the structural Plan Review
      (`references/plan-review.md`) are **replaced** by `goldfish-review`.
      `references/plan-review.md` is deleted (or marked superseded). The
      `goldfish-review` invocation passes: `TargetFile` = `blueprint.md`,
      `GoldfishContext` = `briefing.md` + referenced files, `Step` = inferred
      from the path classification (Interactive for New Project, Consolidated
      for New Function with ≥3 tasks).
    - **Acceptance:** `writing-blueprints/SKILL.md`'s validation block names
      `skillgrid:goldfish-review`; `references/plan-review.md` is deleted or
      marked superseded; the invocation passes the correct TargetFile and
      GoldfishContext.
    - **Acceptance scenario:** `happy path writing-blueprints-invokes-
      goldfish-review` → `acceptance.feature`.

15. **R15 — SKILL.md stays within tier and thin.**
    - **Current:** `writing-blueprints` SKILL.md is 26KB of a 38KB (`large`)
      ceiling; `brainstorming` SKILL.md is ~14KB of a 22KB (`medium`) ceiling.
    - **Target:** the Goldfish protocol detail (stage list, persona dispatch,
      presets, loop, resolution) lives in `goldfish-review/SKILL.md` + its
      `references/` (Track B). The shootdown detail lives in
      `references/shootdown.md` (Track A). The drafting detail lives in
      `references/section-by-section.md` (Track C). The `writing-blueprints`
      SKILL.md change is a pointer to `skillgrid:goldfish-review` plus short
      rewrites. After the change, `skill-size-budget.mjs check` still exits 0
      for all three skills.
    - **Acceptance:** `node .agents/skills/verification/qa/scripts/skill-size-
      budget.mjs check` exits 0 for `brainstorming`, `writing-blueprints`, and
      `goldfish-review`; the SKILL.md diffs are net-neutral or small.
    - **Acceptance scenario:** `happy path skill-size-within-tier` →
      `acceptance.feature`.

16. **R16 — The after-code half is untouched.**
    - **Current:** `qa`, `requesting-code-review`, `parallel-code-review`,
      `review-ledger` own the code-review half.
    - **Target:** this change modifies `brainstorming` (SKILL.md +
      `references/` + `templates/`), `writing-blueprints` (SKILL.md +
      `references/` — `plan-review.md` deleted), `_shared/craft/` (new
      `review-discipline.md`), `_shared/elephant-goldfish/` (new persona
      briefs), `_shared/rules/prose-style.md`, and creates
      `goldfish-review/` (new skill). It does not edit `qa`,
      `requesting-code-review`, `parallel-code-review`, or `review-ledger`,
      and does not add a phase or resume marker to `sdd-structure.md`.
    - **Acceptance:** `git diff --name-only` for the change lists only the
      files in R16's scope; `sdd-structure.md` is not modified.
    - **Acceptance scenario:** `happy path after-code-untouched` →
      `acceptance.feature`.

17. **R17 — Findings feed forward to `qa` without merging the gates.**
    - **Current:** a blueprint review finding resolved "defer" is a note in the
      blueprint's Global Constraints; `qa` does not see it as decision-debt.
    - **Target:** a CRITICAL Goldfish finding that is dispositioned **parked**
      (not fixed, not waived, not contested-away) is recorded in the blueprint
      as an open design gap. `qa`'s existing decision-debt path (which already
      surfaces `**Status: ASSUMED**` blueprints as a WARNING) also surfaces an
      open parked blueprint-review finding as a WARNING. This is the only link
      between the two seams — it does not move the review into `qa`. Track A's
      shootdown findings do NOT feed into `qa` directly; they feed into the
      spec, which feeds into the blueprint, which feeds into `qa`.
    - **Acceptance:** a blueprint with one parked CRITICAL finding, when `qa`
      runs, shows a WARNING naming the parked finding and its path; the gate
      verdict is otherwise unaffected (WARNING, never CRITICAL).
    - **Acceptance scenario:** `happy path parked-finding-surfaced-at-qa` →
      `acceptance.feature`.

### Track C — Drafting Disciplines

18. **R18 — Ingestion step (explicit briefing read + state verification).**
    - **Current:** `writing-blueprints` assumes `briefing.md` exists and reads
      `findings.md` if present, but has no explicit ingestion step or
      confirmation gate before drafting begins.
    - **Target:** before writing any section, `writing-blueprints` runs an
      explicit ingestion step: (1) read `briefing.md` in full, (2) read
      `findings.md` if present, (3) read `adr.md` if present, (4) present a
      2-3 sentence summary of the briefing's key decisions to the user, (5)
      confirm we are starting from the right context before drafting. This is
      lightweight — `briefing.md` already exists; the step makes ingestion
      explicit and adds the confirmation gate.
    - **Acceptance:** `SKILL.md` contains an "Ingest Predecessor Context"
      section; the agent presents a summary of the briefing's key decisions
      before drafting the first section; the user confirms before drafting
      proceeds.
    - **Acceptance scenario:** `happy path ingestion-confirmation` →
      `acceptance.feature`.

19. **R19 — Section-by-section drafting with per-section approval gates.**
    - **Current:** `writing-blueprints` writes the blueprint in one shot: the
      agent drafts the entire plan, then runs Self-Review + Critic + User
      Review. The user sees the finished product only after all sections are
      written.
    - **Target:** the blueprint is drafted section-by-section. The section
      order is: Header → Hypothesis → Must-Haves → Threat Matrix (if
      applicable) → File Structure → Global Constraints → Tasks 1..N →
      Self-Review → Commit. After each section is written to `blueprint.md`,
      the agent presents in the same turn: (1) a visible chat summary (which
      section was drafted, 2-3 key decisions, path to `blueprint.md`), (2) a
      single-select approval gate ("Approve [Section] and proceed" / "Request
      revisions to [Section]"). The user can revisit any earlier section at any
      point. The protocol detail lives in `references/section-by-section.md`.
    - **Acceptance:** `SKILL.md` contains a "Section-by-Section Drafting"
      section that names `references/section-by-section.md`; the agent writes
      each section to `blueprint.md` and presents the approval gate before
      proceeding to the next; the chat summary precedes the gate (no empty
      chat bubbles).
    - **Acceptance scenario:** `happy path section-by-section-drafting` →
      `acceptance.feature`.

20. **R20 — Per-task gate (each task approved individually).**
    - **Current:** the Tasks section is written in one pass — the agent writes
      all tasks, then the user reviews the whole blueprint. A wrong interface
      in Task 1 propagates through Tasks 2-15 before the user sees it.
    - **Target:** each task in the Tasks section gets its own approval gate.
      The agent writes Task 1, presents the gate, the user approves, then the
      agent writes Task 2, and so on. The task template (Files / Interfaces /
      SATISFIES / Steps with code) is unchanged — only the approval cadence
      changes. This is the highest-value addition of Track C: it catches a
      wrong interface, a missing file, or a bad test strategy before the agent
      writes 15 more tasks.
    - **Acceptance:** the agent does not write Task N+1 until Task N is
      approved; the per-task gate is present in `references/section-by-
      section.md`; the task template is structurally unchanged.
    - **Acceptance scenario:** `happy path per-task-gate` →
      `acceptance.feature`.

21. **R21 — Spec reconciliation in Self-Review (contradictions + additions
    vs briefing).**
    - **Current:** Self-Review checks spec coverage (does each spec requirement
      have a task?) but does NOT reconcile the blueprint's *decisions* against
      the briefing's *decisions*. The agent can quietly change a design
      decision the user already made while writing the plan.
    - **Target:** Self-Review gains a new check: **Spec reconciliation.**
      Re-read `briefing.md` and compare it with the blueprint.
      - **Contradiction:** the briefing decided a question and the blueprint
        decides it differently. Report every one, quoting the briefing and the
        blueprint.
      - **Addition:** the blueprint decides a question the briefing did not
        raise. Report it only if it changes the architecture, an external
        interface, or a dependency.
      List contradictions first. Do NOT change the blueprint to match the
      briefing on your own: the blueprint can be right, because drafting reads
      code that the brainstorm did not. If nothing diverges, say so in one
      line. This is the check "no Goldfish can do, because no Goldfish reads
      the brief."
    - **Acceptance:** `SKILL.md`'s Self-Review section contains the Spec
      Reconciliation check; a blueprint with a contradiction vs the briefing
      reports it (quoting both); the agent does not silently change the
      blueprint to match the briefing.
    - **Acceptance scenario:** `happy path spec-reconciliation` →
      `acceptance.feature`.

22. **R22 — Owner Decides Architecture (no ranking, no default, owner
    picks).**
    - **Current:** the Owed-Decision Gate presents options with "recommended
      first, one-line why" — the agent recommends.
    - **Target:** when the Owed-Decision Gate presents ≥2 defensible options
      for a **load-bearing** choice (data model, component boundaries, storage,
      protocol), it presents each with its **mechanism, trade-offs, and the
      conditions under which it fails**. The agent does NOT rank the options,
      does NOT mark a default, does NOT pick one. The owner chooses. If the
      owner says "just pick one," the agent restates the trade-offs and
      presents the same gate again. **Mechanical** choices (file naming,
      formatting, variable names) are decided by the agent — only
      **architectural** choices go to the owner. This is a tone shift from
      "here's my recommendation, please confirm" to "here are the options and
      their failure conditions, you decide."
    - **Acceptance:** the Owed-Decision Gate's option presentation does not
      contain "recommended" or a default marker for architectural choices;
      each option has mechanism + trade-offs + failure conditions; mechanical
      choices are decided by the agent without a gate.
    - **Acceptance scenario:** `happy path owner-decides-architecture` →
      `acceptance.feature`.

23. **R23 — Alternatives sourced from the briefing (not memory).**
    - **Current:** the blueprint template has no explicit instruction to source
      alternatives from the briefing's `Approaches Considered` section. The
      agent may rely on memory, which fails when drafting runs in a new
      session.
    - **Target:** `SKILL.md` instructs the agent to: (1) read the briefing's
      `Approaches Considered` section before writing the blueprint, (2) note
      rejected alternatives in the blueprint (one line each: "Rejected:
      [approach] — [reason]"), (3) if the section is absent, note "No
      alternatives recorded in briefing" rather than inventing them. **Do not
      rely on memory:** if drafting runs in a new session, re-read the
      briefing.
    - **Acceptance:** `SKILL.md` contains the alternatives-from-briefing
      instruction; a blueprint with rejected alternatives in the briefing notes
      them; a blueprint with no alternatives in the briefing notes "No
      alternatives recorded in briefing."
    - **Acceptance scenario:** `happy path alternatives-from-briefing` →
      `acceptance.feature`.

24. **R24 — Anti-sycophancy + plain-English rules apply to the drafting
    conversation.**
    - **Current:** `writing-blueprints`'s tone is governed by `prose-style.md`
      (21 rules) but has no explicit no-flattery / stress-test / plain-English
      mandate during the drafting conversation.
    - **Target:** `SKILL.md` gains a **Tone & Discipline** section citing
      `_shared/craft/review-discipline.md` (the shared file Tracks A and B
      already create): stress-test by default, critique first, no glazing, no
      banned adjectives, no apologies, plain-English standard (no ungrounded
      promotional adjectives, no decorative adverbs). These rules apply to the
      *tone* of the drafting conversation and the blueprint's prose. They do
      not change the *structure* of the blueprint (which is governed by the No
      Placeholders and No Restating rules). The banned-adjective list is in
      exactly one place (`review-discipline.md`).
    - **Acceptance:** `SKILL.md` contains the Tone & Discipline section; every
      rule is cited from `review-discipline.md` (not restated); the
      banned-adjective list is in exactly one place.
    - **Acceptance scenario:** `happy path drafting-anti-sycophancy` →
      `acceptance.feature`.

25. **R25 — Artifact-first delivery (no chat dumps).**
    - **Current:** `writing-blueprints` writes to `blueprint.md` directly but
      has no explicit prohibition on dumping the blueprint content in chat.
    - **Target:** `SKILL.md` states: **Do not dump the blueprint in chat.**
      Write to `blueprint.md` using the file-writing tool and give the user the
      path. Highlight 2-3 key decisions in chat (one line each) and ask for
      review. This is already how the skill works in practice; making it
      explicit prevents the agent from pasting 200 lines of markdown into the
      chat pane.
    - **Acceptance:** `SKILL.md` contains the artifact-first delivery rule;
      the agent writes to `blueprint.md` and gives the path rather than
      printing the content in chat.
    - **Acceptance scenario:** `happy path artifact-first-delivery` →
      `acceptance.feature`.

### Track D — Drift Report

26. **R26 — New top-level skill: `skillgrid:drift-report`.**
    - **Current:** no mechanism to detect that the codebase has diverged from
      its design documents after implementation.
    - **Target:** new skill at
      `.agents/skills/verification/drift-report/` with:
      - `SKILL.md` — the drift detection protocol: intake, sequential
        processing, source grounding, 4 divergence categories, difference
        filtering, artifact output, two-way reconciliation. Independently
        invocable: `/drift-report <spec-dir>` (per-change) or `/drift-report`
        (project-wide, scans all `.skillgrid/change-specs/`).
      - `references/divergence-categories.md` — the 4 categories (Missing
        Features, Incomplete Models, Logic Shortcuts, Architectural
        Divergence) with examples and detection heuristics.
      - `references/reconciliation.md` — the two-way reconciliation protocol
        (code was intentional → update doc; design was violated → file task or
        fix code).
      The skill is under `verification/` (it verifies code against docs, like
      `qa`). It is NOT a phase in `sdd-structure.md` — it's a periodic ritual,
      not a per-change gate. It reuses `goldfish-review`'s resolution workflow
      (R10) for findings.
    - **Acceptance:** `SKILL.md` exists with the drift detection protocol;
      `references/divergence-categories.md` exists with the 4 categories;
      `references/reconciliation.md` exists with the two-way protocol; the
      skill is invocable via `/drift-report <path>`.
    - **Acceptance scenario:** `happy path drift-report-skill` →
      `acceptance.feature`.

27. **R27 — Four divergence categories.**
    - **Current:** no structured categories for doc-vs-code divergence.
    - **Target:** the drift report detects divergence in exactly 4 categories
      (documented in `references/divergence-categories.md`):
      - **Missing Features:** logic or components described in
        `briefing.md`/`blueprint.md` but completely absent in source code.
        Example: a requirement in `briefing.md` has no corresponding task in
        the code.
      - **Incomplete Models:** structs, schemas, dataclasses, or classes
        missing fields specified in the design. Example: `blueprint.md`
        specifies a struct with 5 fields, but the code implementation only
        has 3.
      - **Logic Shortcuts:** hardcoded values, mock fallbacks, or bypasses
        where dynamic logic was designed. Example: `blueprint.md` specifies a
        dynamic quota lookup, but the code uses a hardcoded constant.
      - **Architectural Divergence:** code implemented using different design
        patterns, frameworks, or dependencies than planned. Example:
        `blueprint.md` specifies a gRPC service, but the implementation uses
        custom HTTP handlers.
    - **Acceptance:** `references/divergence-categories.md` defines all 4
      categories with at least one example each; the drift report classifies
      every finding into exactly one category.
    - **Acceptance scenario:** `happy path four-divergence-categories` →
      `acceptance.feature`.

28. **R28 — Target artifacts.**
    - **Current:** no defined target set for drift detection.
    - **Target:** the drift report reads and compares:
      - **Per-change** (`/drift-report <spec-dir>`): `briefing.md`
        (requirements, Must-Haves), `blueprint.md` (file structure, tasks,
        interfaces), `adr.md` (in-force ADRs), `acceptance.feature` (BDD
        scenarios) from the specified spec dir.
      - **Project-wide** (`/drift-report`): all `.skillgrid/change-specs/` dirs +
        `.skillgrid/ARCHITECTURE.md` (structure) + `AGENTS.md` (operational
        rules) + `.skillgrid/artifacts/04-adr-*.md` (in-force ADRs).
      The comparison is: does the code match the documented design? Not: is
      the design good? (That's `goldfish-review`'s job, R7.)
    - **Acceptance:** `/drift-report <spec-dir>` reads the 4 spec artifacts;
      `/drift-report` (no arg) scans all spec dirs + ARCHITECTURE.md +
      AGENTS.md; the report is scoped to the target set.
    - **Acceptance scenario:** `happy path target-artifacts` →
      `acceptance.feature`.

29. **R29 — Sequential processing + source grounding.**
    - **Current:** no drift detection, no sequential processing constraint.
    - **Target:** the drift report processes spec dirs **one at a time**
      (sequential, not parallel) to prevent context overflow. For each doc,
      it **resolves referenced files/symbols/endpoints** via code search
      (`grep`/`glob`/`read`). If a reference is unresolvable (the doc mentions
      a file or symbol that can't be found in the codebase), the report notes
      it as a **Missing Feature** finding AND prompts the user to add an
      explicit link to the document (makes future scans cheaper). The agent
      reads files liberally to ensure it is looking at the current state of
      the code, not relying on memory.
    - **Acceptance:** spec dirs are processed sequentially (not in parallel);
      code search is used to resolve references; unresolvable references are
      reported as findings + a prompt to add links.
    - **Acceptance scenario:** `happy path sequential-source-grounding` →
      `acceptance.feature`.

30. **R30 — Difference filtering.**
    - **Current:** no drift detection, no filtering rule.
    - **Target:** the drift report reports **ONLY** actionable discrepancies.
      If the implementation matches the design completely, it outputs
      "No drift detected." It does NOT describe what is working or what
      matches. Be rigorous: if a design doc mentions a specific object, field,
      or file name and the code doesn't have it, that is a reportable drift.
      Double-check for coverage.
    - **Acceptance:** a clean spec dir produces "No drift detected"; a drifted
      spec dir produces a bulleted list of specific divergences with
      file:line references; no "what matches" descriptions.
    - **Acceptance scenario:** `happy path difference-filtering` →
      `acceptance.feature`.

31. **R31 — Artifact output.**
    - **Current:** no drift report artifact.
    - **Target:** the drift report writes:
      - **Per-change:** `.skillgrid/change-specs/<topic>/drift-report.md`
      - **Project-wide:** `.skillgrid/drift-report-{YYYY-MM-DD}.md`
      Format (strict):
      ```markdown
      # Drift Report — {topic | project-wide} ({YYYY-MM-DD})

      ## <DESIGN_DOC_NAME>

      * [Divergence item 1 — category: Missing Features — file:line]
      * [Divergence item 2 — category: Logic Shortcuts — file:line]
      <!-- Or "No drift detected." -->
      ```
      Each finding names its category (R27) and includes a file:line reference
      where applicable. The report is committed to the spec dir (spec-before-
      code zone guard).
    - **Acceptance:** the report file exists at the correct path; each finding
      has a category + file:line reference; a clean report says "No drift
      detected."
    - **Acceptance scenario:** `happy path drift-report-artifact` →
      `acceptance.feature`.

32. **R32 — Two-way reconciliation.**
    - **Current:** no reconciliation protocol for drift findings.
    - **Target:** every drift finding is actionable via one of two paths
      (documented in `references/reconciliation.md`):
      - **Code was intentional** (the design doc is stale): update
        `briefing.md`/`blueprint.md`/ADR to reflect reality. The doc is the
        artifact that changes.
      - **Design was violated** (the code is wrong): file a Backlog task
        (via `ticketing`) or fix the code directly. The code is the artifact
        that changes.
      The user decides which path for each finding (interactive gate, same
      pattern as R10). The reconciliation is recorded in the drift report
      (each finding gets a disposition: doc-updated / task-filed / code-fixed /
      accepted-drift).
    - **Acceptance:** `references/reconciliation.md` documents both paths;
      each finding in the report has a disposition; the user decides the path
      per finding.
    - **Acceptance scenario:** `happy path two-way-reconciliation` →
      `acceptance.feature`.

33. **R33 — Reuses `goldfish-review` resolution workflow.**
    - **Current:** `goldfish-review` (R10) has a resolution workflow (CONTEST/
      ESCALATE/PROPOSE-FIX/DECLINE + interactive gate) but it is only used for
      plan review findings.
    - **Target:** the drift report reuses the same resolution workflow for its
      findings. The `codebase-details` persona (from
      `_shared/elephant-goldfish/`, R11) is the primary reviewer — its
      "verify code vs doc" audit vectors are directly applicable. The
      categorization rules (R10) apply: a contest must be checkable, don't
      add context for subagents, don't contest a comprehension gap, don't
      change a severity. The interactive gate (auto-accept / continue /
      interactive breakdown / comment / custom) is the same.
    - **Acceptance:** the drift report's findings use the R10 categorization
      (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE); the `codebase-details` persona
      is the primary reviewer; the interactive gate matches R10.
    - **Acceptance scenario:** `happy path drift-reuses-resolution` →
      `acceptance.feature`.

34. **R34 — `reflect` skill gains a drift check item.**
    - **Current:** `reflect` (the post-ship retro skill) has a checklist but
      no drift check.
    - **Target:** `reflect`'s checklist gains an **advisory** item: "Run
      `/drift-report` on this change's spec dir to verify the code matches the
      design." The item is advisory (not blocking) — the reflect report notes
      drift findings (if any) but does not fail the retro. The cadence is
      suggested (weekly gardener rotation or per-ship), not enforced.
    - **Acceptance:** `reflect/SKILL.md`'s checklist contains the drift check
      item; it is marked advisory (not blocking); the reflect report notes
      drift findings if present.
    - **Acceptance scenario:** `happy path reflect-drift-check` →
      `acceptance.feature`.

35. **R35 — SKILL.md stays within tier and thin.**
    - **Current:** N/A (new skill).
    - **Target:** the drift report protocol detail (divergence categories,
      reconciliation, source grounding) lives in `references/`. The SKILL.md
      is thin: invocation, intake, the 4-category summary, the sequential
      processing rule, the artifact path, the pointer to `references/`.
      `skill-size-budget.mjs check` exits 0 for `drift-report`.
    - **Acceptance:** `node .agents/skills/verification/qa/scripts/skill-size-
      budget.mjs check` exits 0 for `drift-report`; the SKILL.md is < 100
      lines.
    - **Acceptance scenario:** `happy path drift-skill-size` →
      `acceptance.feature`.

### Cross-track — Goldfish Isolation & Harness Portability

36. **R36 — Goldfish freshness is non-negotiable; a harness without fresh-context
    subagents does NOT degrade — it STOPS.**
    - **Current:** `brainstorming`'s feasibility check and `writing-blueprints`'
      Self-Review run in the author's own context; a missing subagent tool would
      be absorbed silently (same-context review stands in).
    - **Target:** port EGM's strictness verbatim: *"If your harness has no
      subagent tool, stop and tell the user: a review in the same context is not
      a Goldfish review."* The check applies at the first dispatch point of each
      track:
      - Track A: before the six-lens shootdown is spawned.
      - Track B: before Stage 5 of `goldfish-review` is dispatched.
      - Track D: before the first `codebase-details` reviewer is dispatched.
      A review run in the author's own context is not a Goldfish review and MUST
      NOT count toward satisfying R1 or R7 (already stated there; R36 makes the
      stop-the-world behavior explicit). The stop is not a suggestion: the skill
      halts, tells the user the harness cannot spawn a fresh-context subagent,
      and names the EGM rule it is enforcing. No partial run, no "we'll count
      this one as a review anyway."
    - **Acceptance:** `brainstorming/references/shootdown.md`,
      `goldfish-review/SKILL.md`, and `drift-report/SKILL.md` each contain the
      stop rule verbatim (or its normative equivalent); a same-context run
      recorded as "review complete" in any of the three tracks is a defect.
    - **Acceptance scenario:** `happy path no-subagent-stop` →
      `acceptance.feature`.

37. **R37 — Goldfish stays a Goldfish: context isolation contract.**
    - **Current:** the harness's `reviewer`/`general` subagents inherit whatever
      the harness injects — in Claude Code that includes `CLAUDE.md` (user and
      project instruction files), in OpenCode it includes project
      instructions/skills/references. EGM's Claude adapters declare
      `omitClaudeMd: true` + `tools: [Read, Grep, Glob]` to cut both.
    - **Target:** every Goldfish dispatch (Track A lens, Track B stage reviewer,
      Track D reviewer) enforces the **four-part isolation contract**:
      1. **Fresh context** — no conversation history, no author narrative, no
         parent deliberation. (Already R1/R7.)
      2. **Read-only** — no file edits, no shell, no writes. A reviewer that
         edits the target has not reviewed; its findings are discarded.
      3. **No project instructions** — the harness's per-project instruction
         files (Claude Code: `CLAUDE.md` → `omitClaudeMd: true`; OpenCode:
         project instructions block excluded; Cursor: `rules` files not loaded;
         Pi: `AGENTS.md` not injected) must not reach the reviewer. The
         reviewer sees ONLY: the persona brief, the target file path(s), the
         explicit reading list, and the `Mode: BRAINSTORM|REVIEW` flag.
      4. **No network, no MCP** — the reviewer cannot open URLs, fetch web
         pages, or call MCP tools. Documentation the review needs is saved as a
         file in the workspace and passed by path.
      The contract is stated in each persona brief's `## Isolation mandate`
      section (R11) and re-asserted in the dispatch prompt. A harness that
      cannot enforce part 3 (no project instructions) must either (a) enforce
      it via its own mechanism (adapter / agent config) or (b) say so in the
      review report header ("harness did not exclude project instructions —
      findings may carry author context"); it must not silently pass.
    - **Acceptance:** each persona brief in `_shared/elephant-goldfish/` has a
      `## Isolation mandate` naming all four parts; the dispatch prompt in
      `brainstorming/references/shootdown.md`, `goldfish-review/SKILL.md`, and
      `drift-report/SKILL.md` re-asserts the four parts verbatim; the review
      report format (R10, R13) includes a one-line harness-isolation attestation
      (which of the four parts the harness enforced vs. could not).
    - **Acceptance scenario:** `happy path goldfish-isolation-contract` →
      `acceptance.feature`.

38. **R38 — Harness portability: Claude Code, OpenCode, Cursor, Pi.**
    - **Current:** the spec's persona-pool assumption (Open Questions) names
      only `reviewer`/`general` and assumes they honor the persona file. No
      other harness is named.
    - **Target:** the Goldfish protocol is harness-agnostic at the skill layer
      and harness-specific at the adapter layer. The skill text never names a
      harness tool; it says "spawn a fresh-context read-only subagent and tell
      it to read this persona file first." Each supported harness supplies its
      own adapter:
      - **Claude Code** (`.claude-plugin/agents/<persona>.md`):
        `omitClaudeMd: true`, `tools: [Read, Grep, Glob]`, `model: inherit`.
        Subagent tool: `Agent` (Task). Nesting allowed (up to 3 levels) — the
        main context may dispatch reviewers directly without a synthesizer.
      - **OpenCode** (`.opencode/agents/<persona>.md`):
        `mode: subagent`, `permissions: [edit: deny, shell: deny]` (allow
        read/glob/grep only). Subagent tool: `subagent`. The Goldfish plugin
        (below) strips project instructions from Goldfish child sessions via a
        `context` hook — native enforcement, not just a persona mandate.
      - **Cursor** (`.cursor/agents/<persona>.md`): read-only tools via
        Cursor's agent permission settings; subagent tool: Cursor's Task
        subagent. `rules/` files are NOT loaded for subagents by default —
        verify at install time.
      - **Pi** (pi.dev): subagent capability is detected at runtime via Pi's
        native delegation (Pi supports MCP and agent delegation as of v0.99);
        if Pi cannot spawn a fresh-context subagent in the current session,
        R36's stop rule applies — do not degrade.
      **OpenCode Goldfish plugin (`.opencode/plugins/goldfish/index.ts`):**
      OpenCode V2 agent frontmatter has no `omitClaudeMd` equivalent, but a
      server plugin provides native enforcement. The plugin registers:
      1. **`context` hook** — when a model request fires for a Goldfish child
         session (child sessions run under the goldfish reviewer agent; the
         hook detects them via session lineage / agent identity), it strips the
         instruction sources from `event.system` before dispatch: the
         `AGENTS.md` files, `instructions` entries, and skill/reference/MCP
         guidance. What remains: the provider/agent system prompt (the persona
         brief is the agent's `system`), tool guidance, and the session's own
         dispatch prompt. `metadata.experimental.instructions` lists each
         source as `{key, text}` so the hook can identify and remove the
         instruction parts. Residual: the provider base system prompt still
         loads (unavoidable; `omitClaudeMd` in Claude Code also leaves the base
         prompt) — attested in the report header.
      2. **`tool.transform`** — denies `edit`/`write`/`shell` for Goldfish
         agent sessions (read-only contract part 2, enforced natively rather
         than by prompt discipline).
      3. **`session.hook("prompt")`** — denies `webfetch`/`websearch` tool
         availability in Goldfish sessions (contract part 4, no network).
      The plugin is part of this change's OpenCode adapter (it ships in
      `.opencode/plugins/goldfish/`, loaded automatically). The persona brief's
      isolation mandate stays as defense-in-depth: the plugin is the
      enforcement, the mandate is the fallback if the plugin fails to load.
      The report-header attestation changes from "harness could not exclude
      project instructions" to "stripped by goldfish plugin, provider base
      prompt remains" — full enforcement, not partial.
      The skill must not break when a fifth harness appears: the adapter is a
      file, the skill is a protocol. Adding a harness = adding an adapter
      file + a row in the harness table in `goldfish-review/references/
      harnesses.md` (NEW — the port of EGM's `docs/harnesses.md`, scoped to
      the four supported harnesses + a "contributing a new harness" section).
    - **Acceptance:** `goldfish-review/references/harnesses.md` exists with a
      table covering Claude Code, OpenCode, Cursor, Pi (subagent tool,
      read-only mechanism, project-instruction exclusion mechanism,
      partial-enforcement note where applicable); each skill's dispatch
      section names the protocol, not a specific harness; no harness-specific
      string (`Agent`, `invoke_subagent`, `omitClaudeMd`) appears in
      `SKILL.md` or persona briefs — only in `references/harnesses.md` and the
      adapter files. The OpenCode adapter (`.opencode/plugins/goldfish/`)
      loads and strips instruction sources from a Goldfish child session in a
      smoke test (a reviewer spawned under the goldfish agent does not see the
      project `AGENTS.md` in its context); the attestation in a real
      `goldfish-review` run on OpenCode reads "stripped by goldfish plugin,
      provider base prompt remains."
    - **Acceptance scenario:** `happy path harness-portability` →
      `acceptance.feature`.
    - **Current:** N/A (new skill).
    - **Target:** the drift report protocol detail (divergence categories,
      reconciliation, source grounding) lives in `references/`. The SKILL.md
      is thin: invocation, intake, the 4-category summary, the sequential
      processing rule, the artifact path, the pointer to `references/`.
      `skill-size-budget.mjs check` exits 0 for `drift-report`.
    - **Acceptance:** `node .agents/skills/verification/qa/scripts/skill-size-
      budget.mjs check` exits 0 for `drift-report`; the SKILL.md is < 100
      lines.
    - **Acceptance scenario:** `happy path drift-skill-size` →
      `acceptance.feature`.

## Implementation Decisions

- **Modules to build/modify:**
  - **Track A (brainstorming):**
    - NEW `.agents/skills/planning/brainstorming/references/shootdown.md`
      (the phase instructions: spawn 6 parallel subagents, 6 lens definitions
      with core questions, severity triage, synthesis rules, blocker-must-
      resolve gate).
    - NEW `.agents/skills/planning/brainstorming/agents/{contrarian-yagni,
      cost-economics,ux-ergonomics,migration-compat,observability-sre,
      tech-debt}.md` (6 BRAINSTORM-only persona files, trimmed from EGM's
      dual-mode personas).
    - MODIFY `.agents/skills/planning/brainstorming/SKILL.md` (add shootdown to
      New Project / New Function paths + add Anti-Sycophancy & Stress-Testing
      section).
    - MODIFY `.agents/skills/planning/brainstorming/references/paths.md`
      (insert shootdown step into checklists + update process-flow diagram).
    - MODIFY `.agents/skills/planning/brainstorming/templates/briefing.md`
      (add `## Shootdown Findings` section).
  - **Track B (new `goldfish-review` skill + `writing-blueprints` integration):**
    - NEW `.agents/skills/planning/goldfish-review/SKILL.md` (the 5-stage
      pipeline, 4 presets, checkpoint gates, pause/resume, execution
      constraints).
    - NEW `.agents/skills/planning/goldfish-review/references/goldfish-review-
      loop.md` (sharpening loop: multi-round, convergence, settled/fixed
      tracking, churn, turn-taking, line-range re-scan).
    - NEW `.agents/skills/planning/goldfish-review/references/goldfish-resolve.
      md` (Elephant-side resolution: CONTEST/ESCALATE/PROPOSE-FIX/DECLINE
      rules + interactive gate + report format).
    - MODIFY `.agents/skills/planning/writing-blueprints/SKILL.md` (rewrite the
      validation block to invoke `skillgrid:goldfish-review`; delete or mark
      superseded `references/plan-review.md`).
    - DELETE (or mark superseded) `.agents/skills/planning/writing-blueprints/
      references/plan-review.md`.
  - **Track C (writing-blueprints — drafting process):**
    - NEW `.agents/skills/planning/writing-blueprints/references/section-by-
      section.md` (the section order table, approval gate mechanics, per-task
      gate, empty write-in guard, no-chat-dumps rule, revisit-old-sections
      rule).
    - MODIFY `.agents/skills/planning/writing-blueprints/SKILL.md` (add:
      ingestion step, section-by-section drafting section, spec reconciliation
      to Self-Review, owner-decides-architecture to Owed-Decision Gate,
      alternatives-from-briefing, tone & discipline, artifact-first delivery).
    - MODIFY `.agents/skills/planning/writing-blueprints/templates/blueprint.
      md` (add Self-Review reminder comment in header: section order +
      reconciliation check).
  - **Shared:**
    - NEW `.agents/skills/_shared/elephant-goldfish/{zoom-out,comprehension,
      design-critic,structural,style,implementation-readiness,codebase-
      details}.md` (7 persona briefs, shared by Tracks A and B; Track C uses
      no Goldfish personas. `style` is a standalone file used by Stages 7 and
      9).
    - NEW `.agents/skills/_shared/craft/review-discipline.md` (anti-sycophancy
      + stress-testing rules, shared by both tracks).
    - MODIFY `.agents/skills/_shared/rules/prose-style.md` (add "No flattery"
      section pointing at `review-discipline.md`).
  - **Track D (drift-report):**
    - NEW `.agents/skills/verification/drift-report/SKILL.md` (the drift
      detection protocol: intake, sequential processing, source grounding,
      4 categories, filtering, artifact output, reconciliation).
    - NEW `.agents/skills/verification/drift-report/references/divergence-
      categories.md` (the 4 categories with examples + detection heuristics).
    - NEW `.agents/skills/verification/drift-report/references/reconciliation.
      md` (the two-way reconciliation protocol).
    - MODIFY `.agents/skills/verification/reflect/SKILL.md` (add advisory
      drift check to the checklist).
  - **Cross-track (Goldfish isolation & harness portability, R36-R38):**
    - NEW `.agents/skills/planning/goldfish-review/references/harnesses.md`
      (port of EGM's `docs/harnesses.md`, scoped to Claude Code, OpenCode,
      Cursor, Pi: subagent tool, read-only mechanism, project-instruction
      exclusion mechanism, partial-enforcement notes, + "contributing a new
      harness" section).
    - MODIFY each persona brief in
      `.agents/skills/_shared/elephant-goldfish/` to carry a `## Isolation
      mandate` section naming all four parts of the R37 contract (fresh
      context, read-only, no project instructions, no network/MCP).
    - MODIFY `brainstorming/references/shootdown.md`,
      `goldfish-review/SKILL.md`, and `drift-report/SKILL.md` dispatch
      sections: re-assert the R37 contract in every dispatch prompt + the R36
      stop rule at the first dispatch point.
    - Review report format (R10 / R13 / R31) gains a one-line
      harness-isolation attestation: which of the four parts the harness
      enforced natively vs. which it could not (attested in the report
      header).
    - NEW `.opencode/plugins/goldfish/index.ts` (OpenCode adapter: `context`
      hook that strips instruction sources from Goldfish child sessions,
      `tool.transform` denying edit/write/shell, `session.hook("prompt")`
      denying web tools). Ships in `.opencode/plugins/`, loads automatically.
      The persona-brief isolation mandate remains as fallback if the plugin
      fails to load.
  - **Light qa edit:**
    - MODIFY (light) `.agents/skills/verification/qa/SKILL.md` decision-debt
      paragraph — surface a parked blueprint-review finding as a WARNING.
      (This is the one R14-scoped file `qa` edit; it is additive, one sentence,
      and does not change the four-state gate.)
- **Interfaces (module names / shapes, no file paths):**
  - Persona brief contract: each file is markdown, ≤150 lines, with sections
    `# <Role>`, `## Isolation mandate`, `## Audit vectors`, `## Output format`.
    Output format is the finding schema below.
  - Finding schema (all lenses, both tracks): `Severity: Blocker|Major|Nit` /
    `Affected scope: <heading or line range>` / `Description` / `Rationale`.
  - Dispatch contract: the orchestrating skill spawns one `reviewer` subagent
    (fallback `general`) per lens/stage, prompt = persona brief path + target
    path + reading list + "Mode: BRAINSTORM" (Track A) or "Mode: REVIEW"
    (Track B). The subagent is read-only (no edits).
- **Data flow:**
  - **Track A:** anchored hypothesis (user-confirmed) + relevant existing
    code/files → 6 parallel fresh subagents (one per lens) → main agent
    synthesizes inline (dedupe, group by severity) → **consolidated chat
    response** (severity table + each 🔴 Blocker with disposition options)
    → user dispositions 🔴 Blockers → `briefing.md` `## Shootdown Findings`
    section → commit → `writing-blueprints`.
  - **Track B:** `writing-blueprints` invokes `skillgrid:goldfish-review` on
    `blueprint.md` → 5-stage pipeline (each stage: spawn fresh reviewers →
    collect findings → Elephant-side resolution (categorize + interactive gate
    + apply fixes) → re-scan line ranges → next round) → convergence (2 clean
    rounds) or cap (5) or churn → `.skillgrid/change-specs/<topic>/review/<stage>.
    md` + `goldfish_review_report.md` → **consolidated chat response** (status
    + severity + Blocker/Major findings + report path) → commit → User Review
    Gate → Execution Handoff. A parked CRITICAL → `qa` WARNING (R17).
  - **Track C:** `briefing.md` (+ `findings.md` + `adr.md`) → ingestion
    confirmation → section-by-section drafting (each section written to
    `blueprint.md` + approval gate; per-task gate for Tasks section) → Self-
    Review + Spec Reconciliation → commit → Execution Handoff. The drafting
    conversation is governed by `review-discipline.md` (tone). The blueprint
    is the single source of truth — no `design_draft.md` scratch file.
  - **Track D:** `briefing.md` + `blueprint.md` + `adr.md` + `acceptance.
    feature` (per-change) or all spec dirs + `ARCHITECTURE.md` + `AGENTS.md`
    (project-wide) → sequential processing (one spec dir at a time) → source
    grounding (code search to resolve references) → 4-category divergence
    detection → difference filtering → `drift-report.md` (with harness-
    isolation attestation in the header) → resolution (reuses R10:
    CONTEST/ESCALATE/PROPOSE-FIX/DECLINE + interactive gate) → two-way
    reconciliation (update doc OR file task/fix code) → commit. Periodic
    ritual, not a per-change gate.
  - **Cross-track (all dispatch points):** before any reviewer is spawned, the
    orchestrating skill checks that the harness can spawn a fresh-context
    subagent (R36). If not: stop, tell the user, name the EGM rule. If yes:
    re-assert the R37 four-part isolation contract in the dispatch prompt
    (fresh context, read-only, no project instructions, no network/MCP) and
    record the harness's enforcement level in the report header.
- **Error handling:** a reviewer that times out / returns no report → retry
  once → record `not run` (never zero findings). A stage whose reviewer is
  `not run` caps that stage's verdict at "not verified," not "clean."
- **Dependencies:** no new third-party dependency. Prose + markdown only.

## Testing Decisions

- **What makes a good test:** these are prose artifacts consumed by an agent,
  not compiled code. The "tests" are structural checks: files exist, are within
  line/byte budgets, cite the right references, and the SKILL.md pointers
  resolve. There is no runtime to exercise.
- **Modules to test:** the presence + shape of the new reference/persona files
  (a structural lint), and `skill-size-budget.mjs check` (the existing script)
  as the size gate. The persona *behavior* (a fresh agent actually finding a
  planted gap) is verified by a one-off manual eval, not a committed test —
  it is the skill-creator eval loop, run at review time, not in CI.
- **Prior art:** `skill-size-budget.mjs` is the precedent for a committed
  structural check on skill files.
- **Edge cases:**
  - A blueprint with 1-2 tasks and no one-way-door (Track B review skipped,
    per the existing Plan Review rule).
  - A bounded or prototype brainstorm (Track A shootdown skipped).
  - A failed reviewer (`not run`).
  - A blueprint that already has an `ASSUMED` block (R15 must not double-count
    the decision-debt WARNING).
  - A shootdown where all six lenses return zero findings (the `## Shootdown
    Findings` section records "no findings" rather than being omitted).
  - A shootdown where a 🔴 Blocker is dispositioned "resolved" (the design is
    revised and the feasibility check is re-run before the spec is written).
  - A blueprint where the user requests revisions on a section (the agent
    applies revisions, re-presents the gate; the revision loop is capped at
    reasonable iterations — if the user keeps requesting revisions on the same
    section, the agent asks what specifically is wrong rather than looping
    silently).
  - A blueprint where the user revisits an earlier section (e.g., approves
    Task 5, then goes back to change Must-Haves). The agent re-presents the
    Must-Haves gate, and the user re-approves. Downstream sections that depend
    on Must-Haves are flagged for re-review.
  - A blueprint where the user says "just pick one" at the Owner Decides
    Architecture gate (the agent restates the trade-offs and presents the same
    gate again — it does not pick).
  - A briefing with no `Approaches Considered` section (the blueprint notes
    "No alternatives recorded in briefing" rather than inventing them).
  - A sharpening loop that reaches the cap (5 rounds) without convergence (the
    agent lists open findings by severity and asks the user: accept and
    proceed / continue past the cap / return to drafting).
  - A sharpening loop with churn (a new finding at the scope of an applied fix
    that would be resolved by reverting the fix — the agent detects the
    oscillation and asks the user).
  - A settled finding raised again by a new reviewer in a later round (the
    agent does NOT re-open it — same scope, same issue, already contested or
    accepted).
  - A fixed finding raised again by a new reviewer in a later round (the agent
    re-opens it — the fix failed or a later edit reintroduced the issue).
  - A `goldfish-review` invoked standalone (not by `writing-blueprints`) on a
    `briefing.md` or ADR (the skill is independently invocable; the preset is
    selected by the user).
  - A drift report on a clean spec dir (produces "No drift detected" — the
    report file still exists, with the clean verdict).
  - A drift report where a reference in `blueprint.md` is unresolvable (the
    file or symbol doesn't exist in the codebase — reported as a Missing
    Feature finding + a prompt to add an explicit link to the doc).
  - A project-wide drift report with many spec dirs (sequential processing
    prevents context overflow; the report has one section per spec dir).
  - A drift finding where the user chooses "code was intentional" (the doc
    is updated, not the code — the reconciliation records "doc-updated").
  - A drift finding where the user chooses "design was violated" (a Backlog
    task is filed or the code is fixed — the reconciliation records
    "task-filed" or "code-fixed").
  - A drift report run from `reflect` (the advisory item triggers
    `/drift-report <spec-dir>`; the reflect report notes the findings).

## Impact on Global Docs

- `.skillgrid/artifacts/00-prd.md`: None
- `.skillgrid/ASSUMPTIONS.md`: add two ADR pointer rows under `§ LOCKED`:
  1. "before-design idea validation is a fresh-context 6-lens Goldfish
     shootdown (subflow of brainstorming), not a new pipeline phase."
     Record: `artifacts/04-adr-00XX-goldfish-brainstorm-shootdown.md`.
  2. "before-code plan review is a fresh-context multi-persona Goldfish review
     (subflow of writing-blueprints), not a new pipeline phase and not part of
     qa." Record: `artifacts/04-adr-00XX-goldfish-plan-review.md`.
  3. "blueprint drafting is section-by-section with per-section approval gates
     (subflow of writing-blueprints); architectural decisions are presented to
     the owner without ranking or default." Record:
     `artifacts/04-adr-00XX-goldfish-drafting-disciplines.md`.
- `.skillgrid/ARCHITECTURE.md`: None (no new component/data store; the change
  is to the planning skills' reference set).

## Clarity Report

| Dimension           | Score | Min  | Status | Notes                              |
|---------------------|-------|------|--------|------------------------------------|
| Goal Clarity        | 0.88  | 0.75 | ok     | two-track scope is explicit         |
| Boundary Clarity    | 0.82  | 0.70 | ok     | before-design + before-code; qa     |
|                     |       |      |        | untouched                           |
| Constraint Clarity  | 0.80  | 0.65 | ok     | tier ceilings, no new dep, no       |
|                     |       |      |        | phase                               |
| Acceptance Criteria | 0.81  | 0.70 | ok     | all 35 requirements falsifiable     |
| **Clarity**         | 0.14  | ≤0.20| ok     |                                      |

**Interview log:**

| Round | Question summary                  | Decision locked                              |
|-------|-----------------------------------|----------------------------------------------|
| 1     | One skill or two?                 | Two tracks in one change: shootdown + plan   |
|       |                                   | review; no new top-level skill                |
| 2     | Where does the shootdown sit?     | After feasibility check, before spec; New    |
|       |                                   | Project / New Function only                   |
| 3     | Where do shootdown findings land? | In `briefing.md` as `## Shootdown Findings`   |
|       |                                   | section (no separate artifact)                |
| 4     | Shared or separate persona files? | Shared `_shared/elephant-goldfish/` dir; 6       |
|       |                                   | lenses for Track A, 3 stages for Track B      |
| 5     | Synthesizer subagent or inline?   | Inline (main agent synthesizes); no           |
|       |                                   | synthesizer file                              |
| 6     | Anti-sycophancy rules: tone or    | Tone only; structured `question` tool         |
|       | tooling?                          | mandate stays intact                          |
| 7     | Section-by-section or one-shot?   | Section-by-section with per-section +         |
|       |                                   | per-task approval gates (from egm-draft)      |
| 8     | Which egm-draft disciplines to    | All 8: ingestion, reconciliation, owner-      |
|       | port?                             | decides-architecture, alternatives,           |
|       |                                   | anti-sycophancy, artifact-first, plain-       |
|       |                                   | English, per-task gate. No "no code" rule,    |
|       |                                   | no design_draft.md, no SITEMAP, no CUJs, no   |
|       |                                   | Open Questions (already covered)              |
| 9     | Track B: separate skill or        | Separate skill (`goldfish-review`) with       |
|       | reference files?                  | loop + resolve as reference files (Option C). |
|       |                                   | Reusable across design docs; loop/resolve are |
|       |                                   | tightly coupled, not independently useful     |
| 10    | Track B: 3 or 5 stages?           | 5 stages (port egm-review). Stage 8           |
|       |                                   | (Intermediate Zoom Out) catches altitude      |
|       |                                   | drift from Stages 6+7 fixes                   |
| 11    | Track B: 1 re-audit or sharpening | Sharpening loop (port egm-review-loop):       |
|       | loop?                             | multi-round, 2-clean-round convergence,       |
|       |                                   | settled/fixed tracking, churn, turn-taking    |
| 12    | Track B: resolution model?        | Elephant-side (port egm-resolve): main agent  |
|       |                                   | categorizes + interactive gate + applies      |
|       |                                   | fixes. Not a subagent (needs institutional    |
|       |                                   | knowledge)                                    |
| 13    | Drift report: separate spec or    | Track D in this spec (user chose). Shares     |
|       | Track D?                          | resolution workflow (R10) + persona files     |
|       |                                   | (R11) with Track B. Different failure mode    |
|       |                                   | (post-implementation, periodic) but cohesive  |
|       |                                   | change: "port EGM onto skillgrid"             |

## Open Questions & Assumptions

- **Assumption:** the `reviewer` subagent can be told "read this persona file
  first, then review the target" and will honor read-only. (EGM's own fallback
  assumes the same. If the harness's `reviewer` is too opinionated, fall back
  to `general` — the brief names both.)
- **Assumption:** six parallel subagents can be spawned in the harness without
  hitting a concurrency limit. If the harness caps parallel subagents at N<6,
  the shootdown degrades to sequential dispatch (still fresh-context, still
  one subagent per lens) — the parallelism is an optimization, not a
  requirement. The freshness (no conversation history) is the load-bearing
  rule. If the harness has **no** subagent tool at all, there is no degradation:
  R36 stops the review and says so.
- **Assumption (R38, OpenCode-specific):** the OpenCode Goldfish plugin's
  `context` hook can identify Goldfish child sessions at request time and
  strip the instruction sources from `event.system` using
  `metadata.experimental.instructions` (the per-source `{key, text}` list).
  This metadata is marked experimental in the V2 docs — if its shape changes,
  the hook breaks loudly (instructions remain, attestation reverts to partial)
  rather than silently. The provider base system prompt cannot be stripped
  and remains in reviewer context; this matches Claude Code's `omitClaudeMd`
  behavior (base prompt loads, instruction files do not).
- **Question (open, non-blocking):** whether to also run the comprehension
  reviewer over `briefing.md` (the what-side) or only `blueprint.md`. Default
  for this change: `blueprint.md` only; `briefing.md` review is an optional
  addition the critic stage may note but not block on.
- **Question (open, non-blocking):** whether the shootdown's 6 lenses should be
  configurable (user can skip a lens they don't care about) or fixed. Default
  for this change: fixed (all 6 always run) — the lenses are cheap (read-only
  subagents) and the value is in the conflicting perspectives, so skipping one
  weakens the swarm. Configurability is a later change.

## Decisions (ADR)

- `artifacts/04-adr-00XX-goldfish-brainstorm-shootdown.md` — before-design
  idea validation is a fresh-context 6-lens Goldfish shootdown subflow of
  brainstorming; it validates the premise before the spec is written.
- `artifacts/04-adr-00XX-goldfish-plan-review.md` — before-code plan review is
  a fresh-context multi-persona Goldfish subflow of writing-blueprints; the
  after-code half (qa / code review) is a separate, unchanged gate.
- `artifacts/04-adr-00XX-goldfish-drafting-disciplines.md` — blueprint
  drafting is section-by-section with per-section approval gates (subflow of
  writing-blueprints); architectural decisions are presented to the owner
  without ranking or default; the blueprint is the single source of truth (no
  `design_draft.md` scratch file).

## Terms

- **Goldfish shootdown:** a before-design adversarial review performed by six
  fresh, zero-context subagents (one per lens) that have seen only the anchored
  hypothesis and the relevant existing code — not the author's conversation.
  Validates the *premise* before the spec is written.
- **Goldfish plan review:** a before-code design-document review performed by
  fresh, zero-context subagents that have seen only the target artifact and the
  files it references. Validates the *plan* before code is written.
- **Sharpening loop:** the multi-round review→resolve→re-review cycle that
  `goldfish-review` runs per stage. Convergence = 2 consecutive clean rounds
  (no new Blocker/Major finding that is not settled). Cap = 5 rounds. Churn =
  a new finding at the scope of an applied fix that would be resolved by
  reverting the fix. Turn-taking: the loop goes idle between rounds and waits
  for the user to confirm "run another round."
- **Settled finding:** a finding that is contested (with checkable evidence)
  or accepted-without-fix. Kept across rounds. If a new reviewer raises a
  settled finding again (same scope, same issue), it is NOT re-opened.
- **Fixed finding:** a finding that had an applied fix. An applied fix does
  NOT settle a finding — if a later reviewer raises the same issue at the same
  scope, the fix failed and the finding is re-opened.
- **Elephant-side resolution:** the resolution workflow that runs in the main
  agent's context (not a subagent). The Elephant categorizes findings
  (CONTEST/ESCALATE/PROPOSE-FIX/DECLINE), presents an interactive gate, and
  applies approved fixes. It needs institutional knowledge to write good fixes
  and defend design choices.
- **Drift report:** a post-implementation audit that compares checked-in
  design documents (`briefing.md`, `blueprint.md`, `adr.md`,
  `acceptance.feature`) against the codebase to identify architectural
  divergence. Produces an actionable report with findings in 4 categories
  (Missing Features, Incomplete Models, Logic Shortcuts, Architectural
  Divergence). A periodic ritual, not a per-change gate.
- **Divergence category:** one of the 4 structured classes of doc-vs-code
  drift: Missing Features (logic in doc but absent in code), Incomplete Models
  (structs/schemas missing fields), Logic Shortcuts (hardcoded values where
  dynamic logic was planned), Architectural Divergence (different patterns/
  libraries than proposed).
- **Two-way reconciliation:** the protocol for resolving a drift finding: if
  the code change was intentional, update the doc; if the design was violated,
  file a task or fix the code. The user decides the path per finding.
- **Elephant:** the persistent author context that wrote the document (the
  session's own context). Goldfish reviewers are deliberately not the Elephant.
- **Anchored hypothesis:** the 2-3 sentence summary of the feature concept that
  the user explicitly confirmed at the brainstorming readiness gate, before the
  shootdown is spawned. It is the input to all six lenses.
- **Consolidated chat response:** the working summary the agent posts in chat
  at the end of any review run (goldfish-review, all presets, and the Track A
  shootdown): stage/reviewer status, severity breakdown, every Blocker and
  Major finding in 1-2 lines with the proposed disposition, and the path to
  the full report. The agent works WITH the user, not just for them — the full
  report stays in the artifact, but the chat never ends with only a file path.
- **Section-by-section drafting:** the blueprint drafting model where each
  section (Header, Hypothesis, Must-Haves, etc.) is written to `blueprint.md`
  and approved by the user before the next section is written. Each task in the
  Tasks section gets its own approval gate. Contrast: one-shot drafting (the
  current model) where the entire blueprint is written before the user sees it.
- **Spec reconciliation:** the Self-Review check that compares the blueprint's
  decisions against the briefing's decisions, reporting contradictions (briefing
  decided X, blueprint decides Y) and significant additions (blueprint decides
  something the briefing didn't raise, that changes architecture/interface/
  dependency). "No Goldfish can do this check, because no Goldfish reads the
  brief."
- **Owner Decides Architecture:** the rule that architectural choices (data
  model, component boundaries, storage, protocol) are presented to the owner
  with mechanism + trade-offs + failure conditions, without ranking or default.
  The owner picks. Mechanical choices (naming, formatting) are decided by the
  agent.
- No new terms added to the terms files; the above are local to this change and
  defined here.
