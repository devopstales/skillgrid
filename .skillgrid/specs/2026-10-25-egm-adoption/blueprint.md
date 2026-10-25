# EGM Adoption Implementation Blueprint (steps, files, verification)

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED
**Critic:** PENDING (goldfish protocol applies to this blueprint on its first run — Stage 0/1/3 mandatory, Stage 2 per tier; this is the protocol's first exercise, so the run doubles as its smoke test)

**Tier:** T2

**Build shape:** Smallest usable whole (the shared rule + the `writing-blueprints` rewire are the thinnest version a pipeline run can actually use; the other skills are bolt-ons that each stand alone, so order among them is independent)

**Goal:** Wire Google's Elephant-Goldfish Model 4-stage validation protocol (plus the Zero-Decision Rule, the debugging asymmetry, and the capped rebuttal review loop) into skillgrid-v2's SDD pipeline as shared rules + skill edits, pure spec-zone, no Go code.

**Architecture:** One new shared rule (`.agents/skills/_shared/rules/goldfish-protocol.md`) owns the protocol's mechanics — stages, personas, mandates, gating, loop. Every pipeline skill that runs the protocol *cites* the rule and states only its own dispatch specifics (cite-don't-restate). The ADR records the adoption decision and what was deliberately not adopted.

**Tech Stack:** Markdown (skill files, shared rules, ADR), YAML (config comment, ASSUMPTIONS index row). No code, no dependencies.

**Spec:** `.skillgrid/specs/2026-10-25-egm-adoption/briefing.md` (R1–R9). Product context: Google Open Source Blog 2026-10-22 "Introducing the Elephant-Goldfish Model plugin"; `github.com/google/elephant-goldfish-model` (personas in `agents/`, rubrics in `resources/document_standards/`); `github.com/vshvedov/elephant-goldfish` (debugging asymmetry, review-loop cap).

**Sources consulted (read 2026-10-25):**
- `google/elephant-goldfish-model`: `README.md`, `docs/designs/architecture.md`, `agents/comprehension.md`, `agents/zoom_out.md`, `agents/design_critic.md`, `agents/implementation_readiness.md`, `resources/document_standards/implementation-readiness-standard.md`, `resources/document_standards/zoom-out-standard.md`
- `vshvedov/elephant-goldfish`: `README.md` (pattern, pipeline, the five `eg-*` workflows)

## Terms

- **Elephant** — the deep-context design session (the pipeline skill authoring `briefing.md`/`blueprint.md`). New term introduced by this change; canonical definition lives in `.agents/skills/_shared/rules/goldfish-protocol.md` § Roles.
- **Goldfish** — a stateless, zero-context validator subagent. Defined in the same file.
- **Goldfish Protocol** — the 4-stage pre-implementation validation sequence. Same file.
- **Zero-Decision Rule** — the Stage 3 pass bar: the plan leaves zero design choices to the implementer. Adopted verbatim in spirit from `google/elephant-goldfish-model`'s `resources/document_standards/implementation-readiness-standard.md`; canonical restatement lives in `goldfish-protocol.md` § Stage 3.
- Existing terms (`blueprint`, `briefing`, `rigor tier`, `fast-track`, `fix loop`) keep their definitions in `_shared/rules/sdd-structure.md`, `_shared/planning/rigor-tiers.md`, `_shared/planning/fast-track.md` — not redefined here.

## Must-Haves (goal-backward verification)

**Truths:**
- `grep -rn "critic subagent" .agents/skills/` returns **zero** hits in pipeline files after the change (R7). The phrase may appear only inside `goldfish-protocol.md` if at all, as historical reference.
- `.agents/skills/_shared/rules/goldfish-protocol.md` contains all seven elements of R1 (four stages in order; four personas; common mandates incl. fresh-context + read-only + structured severity output; gating table; Zero-Decision Rule; synthesizer step; max-2-rounds loop with escalation) — each stated as a mechanically followable rule.
- The Stage 3 pass bar in `goldfish-protocol.md` is the Zero-Decision Rule (fresh implementer, zero design choices, file list + steps + test plan present).
- `writing-blueprints/SKILL.md` dispatches four named goldfish subagents (three always, one tier-gated) and feeds their findings to a synthesizer *before* the user-decide loop; the blueprint header recording format `Goldfish: zoom-out ✓|skipped (…) comprehension ✓ critic ✓|skipped (…) readiness ✓` is stated in the skill.
- `brainstorming/SKILL.md` contains a No-Code Rule section and a cross-reference to `goldfish-protocol.md`.
- `structured-debugging/SKILL.md` contains the Goldfish Diagnosis step with the input restriction (symptom + repro only), the convergence/divergence comparison, and the state-file record.
- `receiving-code-review/SKILL.md` caps review fix loops at 3 rounds by *citing* `_shared/planning/rigor-tiers.md` § Fix loop cap (no new number), requires fix-or-verbatim-rebuttal per finding, and escalates to the user at the cap with open findings listed.
- `rigor-tiers.md`'s T2 row and the `writing-blueprints` tier entry name the goldfish protocol; the T3 fresh-model rule extends to the Stage 2 critic goldfish with the existing "recommendation the human can decline; the decline is recorded" posture.
- `sdd-structure.md`'s Phase Order chain, skill table row, and Resume markers table all read `blueprint-validation (goldfish protocol + user)`; `using-skillgrid/SKILL.md`'s artifact-creation-order item 3 matches.
- `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md` exists with frontmatter `type: TechnicalDecision`, `status: accepted`, `date: 2026-10-25`, and a pointer row for 0050 (in force: yes) exists in `ASSUMPTIONS.md` § LOCKED.
- `config.yaml` gained a comment (no new YAML key) under `rules:` pointing at the new rule; `AGENTS.md`'s workflow line reads `writing-blueprints → blueprint-validation (goldfish protocol) → acceptance-test-authoring → …`.

**Artifacts:** (2 created + 10 edited = 12 files total)
- `.agents/skills/_shared/rules/goldfish-protocol.md` — the protocol rule (stages, personas, mandates, gating, Zero-Decision Rule, synthesizer, loop)
- `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md` — the adoption ADR
- Edited skill files (10 files): `writing-blueprints/SKILL.md`, `brainstorming/SKILL.md`, `structured-debugging/SKILL.md`, `receiving-code-review/SKILL.md`, `_shared/planning/rigor-tiers.md`, `_shared/rules/sdd-structure.md`, `lifecycle/using-skillgrid/SKILL.md`, `.skillgrid/config.yaml`, `AGENTS.md`, `.skillgrid/ASSUMPTIONS.md`

**Key links:**
- `writing-blueprints/SKILL.md`'s validation section MUST cite `_shared/rules/goldfish-protocol.md` for stage definitions and MUST NOT redefine a stage's audit vectors (cite-don't-restate); it owns only the dispatch specifics (what artifact each stage receives, the header recording format, the user-decide loop that follows).
- The synthesizer output is the *only* input to the user-decide loop in `writing-blueprints` — raw per-goldfish findings never reach the user unmerged.
- `receiving-code-review`'s round cap MUST cite `_shared/planning/rigor-tiers.md` § Fix loop cap; if that section's number ever changes, the skill inherits the change without editing (the citation is the constraint, per `_shared/craft/cite-dont-restate.md`).
- The ADR's index row in `ASSUMPTIONS.md` MUST point at the ADR file path only (short description + path, per the index convention) — the body never moves into the index.
- `sdd-structure.md` (the file that wins on conflict, per its own header) and `using-skillgrid/SKILL.md` MUST use the identical phase name `blueprint-validation (goldfish protocol + user)` — a rename in one but not the other is a shared-convention drift break.

**One-way-door decisions:** None. (Rollback is a revert of the change folder's commits; no migration, no contract break, no new dependency, no data shape change.)

## Hypothesis

```
**Claim:** A fresh-context, 4-stage goldfish validation gate, wired into
  writing-blueprints behind the rigor-tier dial, catches blueprint defects
  (ambiguity, YAGNI, unactionable steps) that the old single critic subagent
  missed, without blocking T0/T1 work.
**Right condition:** On the first ≥T2 change run through the new protocol,
  the goldfish stage surfaces at least one finding the old mechanism would
  not have produced (e.g., an undefined term, a reuse miss found by code
  search, or a step that fails the Zero-Decision Rule), AND the run completes
  to user-approved blueprint within the 2-round loop.
**Wrong condition:** Two consecutive ≥T2 runs produce zero actionable goldfish
  findings (the gate is ceremony), OR a run exceeds 2 rounds on a standard
  change (the gate is friction), OR a T0/T1 run is blocked by a stage that
  should have been skipped.
**Thinnest MVP:** Tasks 1+2 only — the shared rule + the writing-blueprints
  rewire. Everything else (debugging, review cap, rename sweep, ADR) is
  independent and can ship in a later change if the MVP proves out.
**Door check:** Task 1's self-review — if the rule cannot state a mechanical
  pass/fail for all four stages (i.e., a stage's rubric is left to judgment),
  stop: the protocol is not implementable by a subagent and the design is
  wrong, not the wording.
```

## Threat Matrix

Per `_shared/references/threat-matrix.md`. Only the Skillgrid-specific boundaries are in scope (no routing, shell, subprocess, or Mnemonic tool-contract change in this change).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Shared-convention drift** (edit to `_shared/rules/sdd-structure.md`, `_shared/planning/rigor-tiers.md`) | Applicable | One-line impact statement per edit: `sdd-structure.md` is the source of truth for phase order + artifact names — `using-skillgrid`, `resume`, `brainstorming`, `writing-blueprints`, `ship`, `reflect`, and the state-drift guard all bind to the renamed phase and must agree; `rigor-tiers.md` is the single source for tier→review topology and fix-loop caps — `requesting-code-review`, `parallel-code-review`, `qa`, `receiving-code-review` bind to it. The rename sweep (Task 8) is the design response: every referencing file is named and edited in the same change. | RED scenario `convention-drift-zero-stale-references`: after the change, `grep -rn "critic subagent" .agents/skills/` returns zero pipeline hits AND every skill that dispatches the protocol references `goldfish-protocol.md` by path. (Task 9.) |
| Mnemonic tool surface | N/A: no `mem_*`/`code_*`/`web_cache_*` tool, parameter, or return-shape change; the goldfish dispatches are in-process subagents, not MCP tools | — | — |
| Documentation-like paths | N/A: no new executable Markdown; the new `.md` files are inert rule/decision content, not `README.sh`-class executables | — | — |
| Git repository selection / Commit state / Push state / PR commands | N/A: the change is committed by the executor per `work-unit-commits`; no git automation is added or modified | — | — |

## Global Constraints

- **Cite-don't-restate** (`_shared/craft/cite-dont-restate.md`): skills cite `goldfish-protocol.md` and `rigor-tiers.md` by path; the protocol's stage definitions and the tier numbers live in exactly one place each.
- **Spec zone before code zone** (per `AGENTS.md § Rules`): all commits for this change are spec-zone (skills + `.skillgrid/`); the pre-commit zone guard applies.
- **Conventional commits only; no AI-attribution trailers** (per `AGENTS.md § Rules`).
- **No new dependencies** (per `AGENTS.md § Rules`): zero new runtime or dev dependencies — this change adds no code at all.
- **Byte budget** (`_shared/rules/sdd-structure.md` § Byte budget): `writing-blueprints` and `brainstorming` sit in the `large` tier (38 KB ceiling) per `_shared/skill-size-budget.json`; the edits net out to roughly +2 KB and −1.5 KB respectively (validation section replaced, no new section) — both stay under ceiling. After Task 4, run `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check` and confirm no new overage WARNING for the two skills; if over, trim prose in the touched sections before committing.
- **ADR IRON RULE** (per `ASSUMPTIONS.md`): never delete an ADR file or its row; 0050 is additive.
- **Language** (persona scope rule from `~/.config/opencode/AGENTS.md`): all artifact text (skill prose, rule text, ADR) is neutral English.

## File Structure

- `.agents/skills/_shared/rules/goldfish-protocol.md` (new) — owns the protocol: Roles (Elephant/Goldfish), the four stages in order with each stage's persona + audit vectors + pass criteria, the common mandates, the gating table, the Zero-Decision Rule, the synthesizer, the 2-round loop. The single authority every skill cites.
- `.agents/skills/planning/writing-blueprints/SKILL.md` (edit) — the dispatch owner: which artifact each stage receives, the tier gate application, the synthesizer → user-decide-loop handoff, the header recording format. Cites the rule.
- `.agents/skills/planning/brainstorming/SKILL.md` (edit) — No-Code Rule section + protocol cross-reference.
- `.agents/skills/verification/structured-debugging/SKILL.md` (edit) — Goldfish Diagnosis step in Phase 1.
- `.agents/skills/verification/receiving-code-review/SKILL.md` (edit) — loop cap (cited), verbatim-rebuttal requirement, escalation shape.
- `.agents/skills/_shared/planning/rigor-tiers.md` (edit) — T2 row + tier-entry naming; T3 fresh-model extension to the Stage 2 goldfish.
- `.agents/skills/_shared/rules/sdd-structure.md` (edit) — rename in Phase Order chain, skill table, Resume markers.
- `.agents/skills/lifecycle/using-skillgrid/SKILL.md` (edit) — rename in artifact-creation-order item 3.
- `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md` (new) — the adoption ADR.
- `.skillgrid/ASSUMPTIONS.md` (edit) — one index row for 0050.
- `.skillgrid/config.yaml` (edit) — comment block under `rules:` only.
- `AGENTS.md` (edit) — workflow line inside the `<!-- skillgrid:start/end -->` block only.

---

### Task 1: The goldfish protocol rule

**Files:**
- Create: `.agents/skills/_shared/rules/goldfish-protocol.md`

**Interfaces:**
- Consumes: nothing (this is the authority the others cite). Sources: the two EGM repos' `agents/*.md` personas and `resources/document_standards/*.md` rubrics (read 2026-10-25; adapted, not copied wholesale — skillgrid keeps its own terms and tier names).
- Produces: the rule every later task cites. Later tasks reference these exact section names: `## Roles`, `## Common Mandates`, `## The Four Stages` (with `### Stage 0: Zoom Out`, `### Stage 1: Comprehension`, `### Stage 2: Design Critic`, `### Stage 3: Implementation Readiness`), `## Gating`, `## The Zero-Decision Rule`, `## Synthesis`, `## The Loop`.
- Seam: none (in-process convention file).
- Deletion test: if deleted, `writing-blueprints`'s validation section has no authority to cite — the stage definitions and the Zero-Decision Rule have nowhere to live; `requesting-code-review`/`receiving-code-review` lose the loop discipline.
- Adapters: 0 (a rule, not a port).

**SATISFIES:** protocol-rule-complete; convention-drift-zero-stale-references (part 1)

**Step 1: Write the rule.**

Create `.agents/skills/_shared/rules/goldfish-protocol.md` with exactly this structure and content (adapt section prose as needed, but keep every section name, every stage's pass criteria, and the mandates verbatim — later tasks and the acceptance feature key off them):

```markdown
# Goldfish Protocol (shared convention)

Pre-implementation validation of design artifacts (briefing + blueprint).
Adopted from Google's Elephant-Goldfish Model (2026-10-22;
`github.com/google/elephant-goldfish-model`) — method adopted, plugin not.
Decision record: `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md`.
If a skill disagrees with this file, **this file wins** (same authority rule
as `sdd-structure.md`).

## Roles

- **Elephant** — the deep-context design session: the pipeline skill
  (`brainstorming`, `writing-blueprints`) that authored the artifact, with
  the full conversation history, codebase context, and the decisions made
  along the way. The Elephant designs; it does not validate its own work.
- **Goldfish** — a stateless validator subagent. It shares **no**
  conversation history with the Elephant. It receives exactly what the
  dispatch names: the target artifact (and for Stage 0, codebase read
  access for reuse search). Its blindness is the point: a goldfish that
  cannot act on the artifact alone means the artifact is wrong, not the
  goldfish.

## Common Mandates (apply to every goldfish)

1. **Fresh context.** The dispatch carries the artifact content (or its
   path) + this rule's stage section + the rubric named below. Never the
   designer conversation, never the session's hypotheses, never prior
   goldfish findings.
2. **Read-only.** No writes, no edits, no builds. The goldfish reports; it
   never fixes.
3. **No chitchat.** Output is findings only: no narration of tool usage,
   no preamble.
4. **Structured output.** Each finding: `Severity` (Blocker | Major | Nit),
   `Affected scope` (section heading / file path / line range), and the
   rationale in one or two sentences. A goldfish that finds nothing returns
   the literal line `NO FINDINGS`.
5. **Cite the rubric.** Findings that fail a named rubric criterion name
   the criterion (e.g., "fails Zero-Decision Rule: step 4 leaves the error
   format to the implementer").

## The Four Stages

Stages run in order. Stage 2's findings are the only ones that may be
skipped (Gating); a skipped stage is recorded, never silent.

### Stage 0: Zoom Out (Skeptical Architect)

**Pass bar:** the artifact solves the *root* problem in the simplest way,
reuses existing code, and stays at architectural altitude.

**Rubric (adapted from `zoom-out-standard.md`):**
1. **Context sufficiency** — can the reviewer state what the artifact is
   trying to achieve and why? If not, fail immediately with the missing
   context named.
2. **Root-problem focus** — does it fix the cause, not a symptom?
3. **YAGNI** — no abstractions, config knobs, or features for speculative
   future requirements.
4. **Reuse** — for every new utility/module/helper the artifact proposes,
   the goldfish MUST code-search the repo for an existing equivalent; a
   match found → the finding is "reuse `<path>` instead of writing new".
5. **Altitude control (anti-weeds)** — high-level sections describe
   interfaces and data flow, not loop variables and helper signatures.
6. **No premature optimization** — no caching/sharding/async without a
   proven bottleneck.

### Stage 1: Comprehension (Naive Reader)

**Pass bar:** a fresh reader with zero project history can follow every
claim in the artifact without unstated background.

**Rubric (adapted from `comprehension.md`):**
1. **Undefined terms** — every custom term, acronym, or system name is
   defined inline or linked (glossary / ADR / doc path). Check the rest of
   the document before flagging.
2. **Assumed prerequisites** — no step assumes the reader already knows an
   unstated setup step or invariant.
3. **Vague explanations** — no "the system handles this automatically"
   without stating *how* or linking to where it does.
4. **Cognitive friction** — error/edge behavior is described well enough
   that the reader can predict it.

### Stage 2: Design Critic (Hostile Reviewer)

**Pass bar:** no unmitigated architectural risk, faulty assumption, or
uncovered requirement survives.

**Rubric (adapted from `design_critic.md` + `feature-spec-standard.md`):**
1. **Standard compliance** — every core question the artifact's own
   template/standard requires is answered (for a blueprint: hypothesis,
   must-haves, threat matrix rows, task interfaces).
2. **Faulty assumptions** — assertions about system behavior, performance,
   or external capabilities that lack justification are flagged with the
   missing evidence named.
3. **Unmitigated risks** — failure modes, integration points, scale limits,
   security concerns present but not addressed.
4. **Scope creep** — anything beyond the briefing's problem statement that
   has no requirement pointing at it.

### Stage 3: Implementation Readiness (Pragmatic Tech Lead)

**Pass bar:** the **Zero-Decision Rule** (below). This is the hard gate:
a Stage 3 failure blocks slicing.

**Rubric (adapted from `implementation-readiness-standard.md`):**
1. **Target scope** — explicit list of files to create/modify/delete,
   exact paths.
2. **Actionable steps** — precise per-file changes; no "update the X
   service" without naming the function, the input, and the expected
   output.
3. **Test plan** — named test cases (happy + edge) that must exist after
   the change, with the command that runs them.
4. **Phasing** — the plan's tasks leave the codebase buildable +
   testable at every checkpoint.

## The Zero-Decision Rule

A plan passes Stage 3 iff a competent engineer — or a fresh AI session —
unfamiliar with the codebase can execute it and produce the correct
changes **on the first attempt**, without guessing, asking clarifying
questions, or making independent design choices. If the implementer must
choose anything (a schema, an API shape, a dependency, an error strategy,
an edge-case behavior), the plan **fails**: the choice must be pre-made and
written down. Local implementation detail (loop style, variable names) is
the only permitted discretion.

## Gating

| Stage | T0 | T1 | T2 | T3 |
|---|---|---|---|---|
| 0 Zoom Out | ✓ | ✓ | ✓ | ✓ |
| 1 Comprehension | ✓ | ✓ | ✓ | ✓ |
| 2 Design Critic | skip (record) | skip (record) | mandatory | mandatory, **fresh model** when available (a T3 recommendation the human can decline; the decline is recorded — same posture as the T3 reviewer rule in `rigor-tiers.md`) |
| 3 Implementation Readiness | ✓ | ✓ | ✓ | ✓ |

Fast-track changes (per `_shared/planning/fast-track.md`) run Stages
1+3 only; Stage 0 is skipped with a recorded reason, Stage 2 never runs.

## Synthesis

The orchestrating skill (today: `writing-blueprints`) collects all goldfish
findings, then **synthesizes before the human sees anything**: dedupe
overlapping findings (one finding with the union of its sources), merge
severities (Blocker > Major > Nit), and group by artifact section. The
synthesized list — not the raw per-goldfish output — is what enters the
user-decide loop.

## The Loop

Failure (any Blocker or Major open) → route the synthesized findings back
to the Elephant → the Elephant revises the artifact, addressing every
finding or rebutting it verbatim → re-dispatch the **failed stages only**
(a Stage 0/1/3 that passed is not re-run; Stage 2 re-runs when the
artifact changed materially) → **max 2 revision rounds**. After round 2,
if Blockers/Majors remain, STOP and escalate to the user with the open
findings listed verbatim. Nits do not block; they are recorded and may be
carried into `review.md` later.
```

**Step 2: Self-review (the door check).**

Re-read the file with fresh eyes and answer, in writing (append as a
`## Self-Review` note at the end of the file, or hold it in the commit
message if preferred):
1. Can a subagent follow each stage to a mechanical pass/fail with no
   judgment left open? (If any stage's rubric needs "use your discretion",
   rewrite that criterion until it doesn't — this is the door check; if it
   can't be made mechanical, STOP and raise it with the user.)
2. Does every section name match the names `writing-blueprints` will cite
   (the Interfaces block above)?
3. Is the Gating table consistent with the values in `rigor-tiers.md`
   (T3 fresh-model = recommendation, decline recorded)?
4. Placeholder scan: no TBD/TODO/"similar to" in the file.

**Step 3: Commit.**

```bash
git add .agents/skills/_shared/rules/goldfish-protocol.md
git commit -m "feat(skills): add goldfish protocol shared rule (EGM adoption)

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: 4-stage fresh-context validation rule adopted from EGM; Zero-Decision Rule as Stage 3 pass bar"
```

---

### Task 2: Rewire `writing-blueprints` to the protocol

**Files:**
- Modify: `.agents/skills/planning/writing-blueprints/SKILL.md` (the "Blueprint validation" block, currently lines 32–64)

**Interfaces:**
- Consumes: `goldfish-protocol.md` (stages, gating, synthesis, loop — cited, not restated).
- Produces: the dispatch contract for the protocol:
  - Stage 0 receives: `blueprint.md` + codebase read access (for reuse search).
  - Stage 1 receives: `briefing.md` + `blueprint.md` (comprehension audits both).
  - Stage 2 receives: `blueprint.md` (and `briefing.md` for scope-creep cross-check).
  - Stage 3 receives: `blueprint.md` only.
  - The blueprint header line format: `Goldfish: zoom-out ✓|skipped (reason) comprehension ✓ critic ✓|skipped (reason) readiness ✓|failed (n open)`.
- Seam: none (in-process skill prose).
- Deletion test: reverting this task leaves the protocol rule orphaned (nothing dispatches it) — the two tasks ship together; do not land Task 1 without Task 2 in the same change.
- Adapters: 0.

**SATISFIES:** blueprints-dispatch-goldfish; header-records-per-stage

**Step 1: Replace the validation block.**

In `.agents/skills/planning/writing-blueprints/SKILL.md`, replace the entire
"Blueprint validation (after writing, before proceeding)" block (from the
line `**Blueprint validation (after writing, before proceeding) — the critic stage is` through the line `The critic stage itself may be skipped only under the T0/T1 / fast-track condition above (and must be recorded as skipped).`) with this:

```markdown
**Blueprint validation (after writing, before proceeding) — the Goldfish
Protocol: fresh-context validators + a user-decide loop.**

The validation is the **Goldfish Protocol** per `_shared/rules/goldfish-protocol.md`
(adopted from Google's Elephant-Goldfish Model — method, not plugin;
decision: `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md`).
The stages, their rubrics, the common mandates (fresh context, read-only,
structured severity output), and the loop are defined there — cite, don't
restate. This skill owns only the dispatch specifics:

1. **Dispatch the goldfish.** Spawn one read-only subagent per stage that
   runs (Gating per `_shared/rules/goldfish-protocol.md` § Gating + the
   change's tier): Stage 0 Zoom Out, Stage 1 Comprehension, Stage 3
   Implementation Readiness always; Stage 2 Design Critic at T2+ (T3: on a
   fresh model when available — a recommendation the human can decline,
   the decline recorded, per `rigor-tiers.md`). Each dispatch carries only
   the artifact(s) named below + the rule's stage section + that stage's
   rubric — never this session's conversation:
   - Stage 0 ← `blueprint.md` + codebase read access (reuse search)
   - Stage 1 ← `briefing.md` + `blueprint.md`
   - Stage 2 ← `blueprint.md` (+ `briefing.md` for the scope-creep check)
   - Stage 3 ← `blueprint.md`
2. **Synthesize.** Merge all findings per the rule's § Synthesis (dedupe,
   severity merge, group by section). The synthesized list is the only
   critique that reaches the user.
3. **User review (always mandatory).** Present the blueprint + synthesized
   findings. For each finding, present options with benefits and
   weaknesses; use the agent's answer-select tool so the user picks the
   resolution path. **Update the blueprint to reflect each decision, then
   re-present the final blueprint.** The user approves the final blueprint
   before `acceptance.feature` is written.
4. **The loop.** Blockers/Majors unresolved → revise per the rule's
   § The Loop (failed stages only re-run; max 2 rounds; then escalate to
   the user with the open findings verbatim).

**Record the run in the blueprint header**, one line, per-stage:
`Goldfish: zoom-out ✓ comprehension ✓ critic ✓ readiness ✓` — with
`skipped (<reason>)` for any gated stage (e.g. `critic skipped (T1)`) and
`failed (<n> open)` only at escalation. A header with no `Goldfish:` line
is an unvalidated blueprint: `acceptance-test-authoring` refuses it.
```

Then update the two downstream references in the same file so they name the
new mechanism (grep the file for `critic subagent` and `Critic: skipped`
— both must be replaced):
- In "When to Use"/role prose and the `## Verification` checklist: replace
  "critic subagent" phrasing with "goldfish protocol (per
  `_shared/rules/goldfish-protocol.md`)" and replace the checklist item
  "The critic stage ran … or is recorded as `Critic: skipped …`" with
  "The Goldfish Protocol ran (per `_shared/rules/goldfish-protocol.md`):
  the header carries the `Goldfish:` line with per-stage results; gated
  stages are recorded as `skipped (<reason>)`, never silently absent."
- In the `## Common Rationalizations` table, the row about skipping the
  critic: reword to reference the gated Stage 2 (skip is T0/T1/fast-track
  only, and it is recorded).

**Step 2: Verify consistency.**

- `grep -n "critic subagent" .agents/skills/planning/writing-blueprints/SKILL.md` → zero hits.
- `grep -n "Critic: skipped" .agents/skills/planning/writing-blueprints/SKILL.md` → zero hits.
- The file's `## Verification` checklist now checks for the `Goldfish:` header line.
- Run `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check` → no new overage for `writing-blueprints` (large tier, 38 KB). If over, trim redundant prose in the replaced section (the old block is longer than the new one; a net overage means the surrounding text also grew — fix before commit).

**Step 3: Commit.**

```bash
git add .agents/skills/planning/writing-blueprints/SKILL.md
git commit -m "feat(skills): rewire writing-blueprints validation to the goldfish protocol

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: 4 fresh-context goldfish stages + synthesizer replace the single critic subagent; Goldfish: header line records the run"
```

---

### Task 3: Brainstorming No-Code Rule + cross-reference

**Files:**
- Modify: `.agents/skills/planning/brainstorming/SKILL.md`

**Interfaces:**
- Consumes: `goldfish-protocol.md` (cross-reference only).
- Produces: the Elephant's own boundary, stated where the Elephant lives.
- Seam: none. Deletion test: the No-Code Rule reappears as an unenforced
  expectation in every brainstorm session. Adapters: 0.

**SATISFIES:** brainstorming-no-code-rule

**Step 1: Add the No-Code Rule section.**

In `.agents/skills/planning/brainstorming/SKILL.md`, immediately after the
`<HARD-GATE>` block (the existing no-implementation gate), insert:

```markdown
<HARD-GATE>
**No Code (the Elephant's boundary).** This session is the *designer*,
not the builder. It produces intent, requirements, trade-offs, and ADRs —
never code, scaffolding, file layouts, or function signatures beyond what
the spec requires to be unambiguous. If you catch yourself sketching
implementation (pseudocode, struct fields, endpoint shapes), STOP: that
content belongs in the blueprint, where the Goldfish Protocol
(`_shared/rules/goldfish-protocol.md`) will validate it against the
Zero-Decision Rule before any code is written.
</HARD-GATE>
```

**Step 2: Add the cross-reference.**

In the "New Function" path prose (and the "New Project" path prose), at the
sentence that hands off to `skillgrid:writing-blueprints`, append one
sentence: "The resulting blueprint is validated by the Goldfish Protocol
before it is committed (per `_shared/rules/goldfish-protocol.md`)."

**Step 3: Verify + commit.**

- `grep -c "No Code" .agents/skills/planning/brainstorming/SKILL.md` ≥ 1; the cross-reference appears in both full paths.
- Run `node .agents/skills/verification/qa/scripts/skill-size-budget.mjs check` → no new overage for `brainstorming` (large tier).
- Commit:

```bash
git add .agents/skills/planning/brainstorming/SKILL.md
git commit -m "feat(skills): add No-Code Rule to brainstorming (Elephant boundary)

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: Elephant session stays design-only; handoff prose names the goldfish protocol"
```

---

### Task 4: Debugging asymmetry (goldfish diagnosis)

**Files:**
- Modify: `.agents/skills/verification/structured-debugging/SKILL.md` (Phase 1, after "Reproduce Consistently")

**Interfaces:**
- Consumes: the existing Debug State file format (`.skillgrid/sdd/debug/<date>-<slug>/state.md`, entries `hypothesis | evidence | verdict`).
- Produces: a new Phase 1 step; the state file gains one new field value: `source: goldfish|session` on hypothesis entries.
- Seam: none. Deletion test: anchored hypotheses go unchallenged; the first plausible cause wins by construction. Adapters: 0.

**SATISFIES:** debugging-goldfish-diagnosis

**Step 1: Add the step.**

In Phase 1, after the "2. Reproduce Consistently" item, insert as new item 3
(renumber the following items):

```markdown
3. **Goldfish Diagnosis (before you state your hypothesis as the working one)**
   - Dispatch a fresh read-only subagent that receives **only** the
     symptom + the repro steps — never your hypothesized root cause, never
     the session's investigation so far. Ask it to rank its top 1–3
     candidate root causes with file:line + the evidence that would
     falsify each.
   - **Compare:** convergence (the goldfish's top candidate matches yours)
     → proceed with confidence; note it. Divergence → the goldfish's
     candidates enter the state file's hypothesis list FIRST (they were
     found without your anchor), and you tell the user: "the goldfish
     flagged a different root cause: <candidate>; here's why I think
     <yours> is still right — do I proceed or investigate that one?"
   - Record the outcome in the state file entry:
     `goldfish: converged | diverged (candidates: …)`.
   - **Skip** for the trivial one-liner path (same "When NOT to use" gate
     as the whole skill) and when no repro exists yet — diagnosis without
     a repro is guessing.
```

**Step 2: Verify + commit.**

- The step's input restriction is explicit ("never your hypothesized root cause"); the state-file record format is stated.
- Commit:

```bash
git add .agents/skills/verification/structured-debugging/SKILL.md
git commit -m "feat(skills): add goldfish diagnosis step to structured-debugging

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: fresh subagent diagnoses from symptom+repro only (vshvedov asymmetry); convergence/divergence recorded in the debug state file"
```

---

### Task 5: Review loop cap + verbatim rebuttal

**Files:**
- Modify: `.agents/skills/verification/receiving-code-review/SKILL.md`

**Interfaces:**
- Consumes: `_shared/planning/rigor-tiers.md` § Fix loop cap (cited — the single source for the number 3).
- Produces: the loop discipline, stated where findings are processed.
- Seam: none. Deletion test: findings can be silently dropped; loops run past the cap with no escalation. Adapters: 0.

**SATISFIES:** review-loop-capped-rebuttal

**Step 1: Add the loop discipline section.**

After "## The Response Pattern" in `receiving-code-review/SKILL.md`, insert:

```markdown
## The Loop Discipline (multi-round reviews)

When a review produces findings that need fix rounds, the loop is capped
and every finding is accounted for:

1. **Cap:** review fix loops run **at most 3 rounds, then escalate to the
   human** — the cap and its rationale live in
   `_shared/planning/rigor-tiers.md` § Fix loop cap (cite, don't restate;
   if that number changes, this section inherits the change).
2. **No silent dismissals.** Every finding ends in exactly one of two
   states, recorded in `review.md`: **fixed** (commit reference) or
   **rebutted** — the rebuttal is recorded *verbatim* (the finding quoted,
   then the technical reason it is wrong for this codebase). A finding
   that is neither fixed nor rebutted is still open; it consumes the next
   round.
3. **Escalation shape.** At the cap, or when the same finding reappears at
   the same file:method across two rounds, STOP and present the user the
   open findings verbatim with a recommendation (accept / keep working /
   abandon). The user's choice is recorded in `review.md`.
```

**Step 2: Verify + commit.**

- `grep -n "3 rounds" .agents/skills/verification/receiving-code-review/SKILL.md` → the cap is cited to `rigor-tiers.md`, not defined independently (the section names the citation).
- Commit:

```bash
git add .agents/skills/verification/receiving-code-review/SKILL.md
git commit -m "feat(skills): cap review fix loops and require verbatim rebuttal

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: 3-round cap cited from rigor-tiers.md; fix-or-rebut-verbatim per finding; user escalation at the cap (vshvedov loop shape)"
```

---

### Task 6: Rigor tiers alignment

**Files:**
- Modify: `.agents/skills/_shared/planning/rigor-tiers.md`

**Interfaces:**
- Consumes: `goldfish-protocol.md` § Gating (the authoritative gating table — this file's edits must agree with it, and where they overlap, the protocol rule wins).
- Produces: tier→protocol naming in the table + the T3 fresh-model extension.
- Seam: none. Deletion test: T2's review topology is unnamed at the tier level; the T3 goldfish critic runs on the designer's model with no rule saying otherwise. Adapters: 0.

**SATISFIES:** tiers-name-protocol

**Step 1: Edit the tiers table + tier entry.**

In the tiers table, change T2's Review cell from
`two-axis (skillgrid:requesting-code-review)` to
`two-axis (skillgrid:requesting-code-review) + **goldfish protocol** Stage 2 mandatory at blueprint validation (per _shared/rules/goldfish-protocol.md)`
and T3's Review cell appends:
`+ Stage 2 goldfish on a **fresh model** when available (recommendation; decline recorded)`.

In "## What each skill does with the tier", the `skillgrid:writing-blueprints` bullet:
append `; runs the Goldfish Protocol (per _shared/rules/goldfish-protocol.md) with the tier's gating — Stage 2 mandatory at T2+, fresh-model at T3`.

**Step 2: Edit the T3 fresh-model rule.**

In "## Rules", extend the last bullet (T3's fresh-model reviewer is a
recommendation…) to name the Stage 2 goldfish:
`- T3's fresh-model rule covers both the post-implementation reviewer and
  the blueprint's Stage 2 (Design Critic) goldfish: a model reviewing its
  own design carries its own blind spots. The fresh model is a
  recommendation the human can decline; the decline is recorded in the
  blueprint header's Goldfish line.`

**Step 3: Verify + commit.**

- The T2/T3 rows and the Rules bullet all name the protocol by path; no stage's pass criteria are restated (only the gating + fresh-model posture).
- Consistency check against `goldfish-protocol.md` § Gating: T3 Stage 2 = "mandatory, fresh model when available (recommendation, decline recorded)" — both files must say the same thing.
- Commit:

```bash
git add .agents/skills/_shared/planning/rigor-tiers.md
git commit -m "feat(skills): align rigor tiers with the goldfish protocol

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: T2 mandates Stage 2; T3 fresh-model rule extended to the Stage 2 goldfish (same decline posture as the reviewer rule)"
```

---

### Task 7: ADR-0050 + index row

**Files:**
- Create: `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md`
- Modify: `.skillgrid/ASSUMPTIONS.md` (one row in § LOCKED ADR index, after row 0049)

**Interfaces:**
- Consumes: the briefing's gap analysis + the sources listed in the blueprint header.
- Produces: the durable decision record; the index row per the IRON RULE.
- Seam: none. Deletion test: the adoption is untraceable from the repo; a future change can't cite why the protocol exists. Adapters: 0.

**SATISFIES:** adr-0050-adopted

**Step 1: Write the ADR.**

```markdown
---
type: TechnicalDecision
title: "EGM Goldfish Protocol: fresh-context pre-implementation validation"
status: "accepted"
supersedes: none
date: 2026-10-25
---

# EGM Goldfish Protocol

## Context and Problem Statement

Google published the Elephant-Goldfish Model (EGM) on 2026-10-22 (Google
Open Source Blog; `github.com/google/elephant-goldfish-model`, Apache-2.0,
531★ at adoption time). EGM names four systemic failure modes of
conversational AI coding: hallucination cascades / context pollution
(a single thread designs, checks, and codes — validating its own
assumptions), premature implementation, chat-dump fatigue, and
documentation drift. Its mechanism: an **Elephant** (deep-context design
session, no code) and **Goldfish** (stateless zero-context validator
subagents) run in a fixed protocol — Zoom Out → Comprehension → Design
Critic → Implementation Readiness — each against a written rubric, with a
synthesizer merging findings before the human triages. A community
adaptation (`github.com/vshvedov/elephant-goldfish`, MIT) adds a
hidden-hypothesis diagnosis in debugging and a capped, verbatim-rebuttal
review loop.

skillgrid-v2's pipeline already has ~70% of EGM (Elephant ≈
`brainstorming` + `writing-blueprints`; persistent memory in `.skillgrid/`
OKF + Mnemonic). The gap: its only pre-implementation critic was a single
"context-bearing critic subagent" — same model, same conversation, no
written rubric, no fresh-context mandate, no readiness pass bar. The
T3 fresh-model rule covered post-implementation review only.

## Decision

Adopt the EGM **method**, not the plugin:

1. **4-stage Goldfish Protocol** as a shared rule
   (`.agents/skills/_shared/rules/goldfish-protocol.md`): Zoom Out,
   Comprehension, Design Critic, Implementation Readiness — fresh-context,
   read-only subagents, structured severity output, synthesizer before
   human triage, max-2-rounds loop with escalation.
2. **Zero-Decision Rule** as the Stage 3 (readiness) pass bar, adopted from
   EGM's `implementation-readiness-standard.md`: the plan leaves zero
   design choices to the implementer; Stage 3 failure blocks slicing.
3. **Tier gating** per `rigor-tiers.md`: Stages 0/1/3 always; Stage 2
   mandatory at T2+, fresh-model at T3 (recommendation, decline recorded).
4. **Supporting disciplines** in their existing homes: the
   hidden-hypothesis goldfish diagnosis in `structured-debugging` (Phase 1);
   the 3-round cap + verbatim rebuttal + user escalation in
   `receiving-code-review` (citing the existing fix-loop cap).
5. **No-Code Rule** in `brainstorming`: the Elephant designs, never builds.

**Not adopted** (deliberately): the Antigravity plugin itself (our SDD
skills already orchestrate); the single design doc (we keep
`briefing.md` + `blueprint.md` — the what/how split `cite-dont-restate`
enforces); `docs/designs/` as the spec location (we keep
`.skillgrid/specs/` with its archive lifecycle); `egm-drift-report` and
`egm-bootstrap` (new tools, not method — parked as follow-ups); the
divergent-lens brainstorming rework (parked until one full pipeline run on
the new protocol).

## Consequences

- Every T2+ blueprint gets 3–4 fresh subagent dispatches pre-slicing:
  higher token cost, lower first-pass rework. T0/T1 stay cheap (Stage 2
  skipped, recorded).
- The protocol rule is a new shared-convention file: it is the single
  authority for stage definitions; skills cite it. Drift risk is managed
  by the rename sweep (no "critic subagent" references survive) and the
  threat-matrix grep check.
- The Zero-Decision Rule raises the blueprint bar: blueprints must now
  pre-make every design choice. Some authors will push back — that is the
  gate working; the 2-round loop + user escalation is the pressure valve.
- Rollback: revert the change's commits; no migration, no contract break,
  no data shape change (ADR IRON RULE: this file and its row are never
  deleted — supersede if the method changes).
```

**Step 2: Add the index row.**

In `.skillgrid/ASSUMPTIONS.md` § LOCKED ADR index, after the 0049 row, add:

```markdown
| 0050 | EGM Goldfish Protocol: 4-stage fresh-context pre-implementation validation (Zoom Out → Comprehension → Design Critic → Readiness) as a shared rule; Zero-Decision Rule as the readiness bar; tier gating per rigor-tiers; plugin NOT adopted | yes | `.skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md` |
```

**Step 3: Verify + commit.**

- Frontmatter keys present (`type`, `title`, `status: accepted`, `supersedes: none`, `date: 2026-10-25`); the row is in the table with in-force `yes`.
- Commit (spec zone: both files are under `.skillgrid/`):

```bash
git add .skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md .skillgrid/ASSUMPTIONS.md
git commit -m "docs(decisions): adopt EGM goldfish protocol (ADR-0050)

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: method adopted, plugin not; Zero-Decision Rule; staged follow-ups parked"
```

---

### Task 8: Rename sweep (sdd-structure + using-skillgrid) + config/AGENTS pointers

**Files:**
- Modify: `.agents/skills/_shared/rules/sdd-structure.md` (3 sites: Phase Order chain line 10; skill table row for `writing-blueprints`; Resume markers / phase-derivation rows that name the phase)
- Modify: `.agents/skills/lifecycle/using-skillgrid/SKILL.md` (artifact-creation-order item 3)
- Modify: `.skillgrid/config.yaml` (comment block under `rules:` only — no new key)
- Modify: `AGENTS.md` (workflow line inside the `<!-- skillgrid:start/end -->` block)

**Interfaces:**
- Consumes: the phase name `blueprint-validation (goldfish protocol + user)` (must be byte-identical in every file).
- Produces: zero stale references repo-wide.
- Seam: none. Deletion test: the rename sweep deleted → `grep "critic subagent"` hits in pipeline files; resume/orchestrator prose names a mechanism that no longer exists. Adapters: 0.

**SATISFIES:** convention-drift-zero-stale-references

**Step 1: Rename in `sdd-structure.md`.**

- Phase Order chain (line 10): `brainstorming → [research | prototype | sketch] → writing-blueprints → blueprint-validation (goldfish protocol + user) → acceptance-test-authoring → …` (replace `critic subagent` with `goldfish protocol`).
- Skill table row for `writing-blueprints`: `Technical plan → blueprint.md; **must** run the Goldfish Protocol (per _shared/rules/goldfish-protocol.md) + user validation before completion`.
- The "blueprint.md (no validation)" resume-marker / phase-derivation rows: replace `(critic subagent + user)` with `(goldfish protocol + user)` wherever the phrase occurs.
- Add one line to the skill table or the Phase Order notes: `Goldfish Protocol: _shared/rules/goldfish-protocol.md (pre-implementation validation; decision: decisions/04-adr-0050-egm-goldfish-protocol.md)`.

**Step 2: Rename in `using-skillgrid/SKILL.md`.**

Artifact-creation-order item 3: `3. Blueprint validation (goldfish protocol → user review of options/benefits/weaknesses)`.

**Step 3: Config comment + AGENTS line.**

In `.skillgrid/config.yaml`, under `rules:` (after the `review:` list, before `archive:`), insert:

```yaml
  # --- Goldfish Protocol (pre-implementation design validation) ---
  # The blueprint-validation gate is the Goldfish Protocol: 4 fresh-context,
  # read-only validator stages (Zoom Out, Comprehension, Design Critic,
  # Implementation Readiness) with a synthesizer, per
  # .agents/skills/_shared/rules/goldfish-protocol.md (decision:
  # .skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md). Gating is
  # tier-driven per .agents/skills/_shared/planning/rigor-tiers.md — no
  # config key; the rule + tier file are the sources of truth.
```

In `AGENTS.md`, inside the `<!-- skillgrid:start -->` block, the Workflow line:
replace `writing-blueprints →` … `→` segment so it reads
`writing-blueprints → blueprint-validation (goldfish protocol) → acceptance-test-authoring` (matching the existing line's style; the full line keeps its other phases unchanged).

**Step 4: Verify (the drift check) + commit.**

- `grep -rn "critic subagent" .agents/skills/` → **zero** hits in pipeline files. (If `goldfish-protocol.md` itself mentions the old name, that is permitted only as historical reference in a "replaces" note — check the hit and decide; the acceptance criterion is zero hits *outside* the new rule file.)
- `grep -rn "critic subagent" .skillgrid/ AGENTS.md` → zero hits.
- Phase-name consistency: `grep -rn "goldfish protocol + user" .agents/skills/` → identical string in `sdd-structure.md` and `using-skillgrid/SKILL.md`.
- `yaml` parse check on config.yaml: `node -e "const y=require('js-yaml'); y.load(require('fs').readFileSync('.skillgrid/config.yaml','utf8')); console.log('ok')"` (or the repo's equivalent if js-yaml is not resolvable — use `python3 -c "import yaml,sys; yaml.safe_load(open('.skillgrid/config.yaml'))"` as fallback).
- Commit (single commit: the sweep is one logically atomic change — partial renames leave the drift in place):

```bash
git add .agents/skills/_shared/rules/sdd-structure.md .agents/skills/lifecycle/using-skillgrid/SKILL.md .skillgrid/config.yaml AGENTS.md
git commit -m "feat(skills): rename blueprint-validation to goldfish protocol; add pointers

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: zero stale 'critic subagent' references; config comment + AGENTS workflow line point at the rule"
```

---

### Task 9: Convention-drift RED check (whole-change verification)

**Files:**
- Modify: none (verification task; writes only the QA evidence into `report.md` later)

**Interfaces:**
- Consumes: the entire change's commits (Tasks 1–8).
- Produces: the evidence that R7's truth holds.
- Seam: none. Deletion test: the change ships with an unverified drift claim. Adapters: 0.

**SATISFIES:** convention-drift-zero-stale-references

**Step 1: Run the checks.**

```bash
# 1. Zero stale references (the R7 truth):
grep -rn "critic subagent" .agents/skills/ || echo "PASS: no stale 'critic subagent'"
grep -rn "Critic: skipped" .agents/skills/ || echo "PASS: no stale 'Critic: skipped'"

# 2. Every protocol dispatch cites the rule by path:
grep -rln "goldfish-protocol.md" .agents/skills/ | sort
# expect: writing-blueprints/SKILL.md, brainstorming/SKILL.md,
#         _shared/planning/rigor-tiers.md, _shared/rules/sdd-structure.md
#         (and the rule file itself)

# 3. Phase-name consistency:
grep -rn "goldfish protocol + user" .agents/skills/ | sort

# 4. ADR + index row present:
test -f .skillgrid/decisions/04-adr-0050-egm-goldfish-protocol.md && echo "PASS: ADR exists"
grep -n "0050" .skillgrid/ASSUMPTIONS.md

# 5. Build still green (no code changed, but the floor is the floor):
go build ./... && go test ./...
```

**Step 2: Record evidence.**

Append the outputs to the change's execution ledger (`.skillgrid/sdd/2026-10-25-egm-adoption/progress.md` if execution is ledgered, otherwise note them in the commit message of Task 8's follow-up fix commit, if any fixes were needed). Any failure → fix inline and re-run; this task is green only when every check prints PASS.

**Step 3: Commit (only if Step 1 forced edits).**

If fixes were needed, commit them with:

```bash
git add <fixed files>
git commit -m "fix(skills): close convention-drift gaps from goldfish rename

[skillgrid-context]
Change: 2026-10-25-egm-adoption
Decisions: drift-check failures fixed (list in message body)"
```

If nothing failed, this task commits nothing — its output is the PASS evidence.

---

## Execution notes

- **Ordering:** Task 1 → 2 are coupled (ship together — a rule with no dispatcher is dead). Tasks 3–7 are independent of each other and of 1–2; they can run in any order or parallel worktrees (per `AGENTS.md § Rules`: parallel subagents use isolated worktrees). Task 8 is last before Task 9 (the sweep must see all final text). Task 9 is the whole-change gate.
- **Verification floor:** L2 per the `standard` classification (full `go build ./...` + `go test ./...` in Task 9) — no code changes, so the suite is the guard against accidental non-markdown edits.
- **First-run note:** this change is the protocol's first exercise. When it itself goes through `writing-blueprints` validation (it already has this blueprint), the goldfish run on *this* blueprint is the smoke test — its findings are the hypothesis's first data point (Right condition: ≥1 actionable finding the old critic would have missed).
