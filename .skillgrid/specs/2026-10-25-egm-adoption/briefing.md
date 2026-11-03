# EGM Adoption — Design Briefing (requirements & intent)

> **STATUS:** `draft` (2026-10-25)

**Tier:** T2
**Classification:** standard (12 files touched, 3 of which are `_shared/` convention files; no behavior change to code)
**Change class (fast-track check):** not fast-track — it touches `_shared/rules/` (the shared-convention drift boundary is Applicable, which blocks fast-track per `rules.fast_track.blocked_when` in `.skillgrid/config.yaml`).

## Problem

AI-assisted coding speed has outpaced verification. The pipeline's only pre-implementation critic is the "critic subagent" inside `skillgrid:writing-blueprints` — a single, unnamed, context-bearing reviewer. It shares the designer session's blind spots: the same model that wrote the blueprint is the one checking it, with the conversation history that shaped it. This is exactly the failure mode Google's Elephant-Goldfish Model (EGM, introduced 2026-10-22 in the Google Open Source Blog post, repo `google/elephant-goldfish-model`) names "Hallucination Cascades & Context Pollution": the model validates its own faulty assumptions instead of detecting errors.

Google's answer is a **Goldfish Protocol**: stateless, zero-context validator subagents that share no conversation history with the designer, run in a fixed multi-stage sequence (Zoom Out → Comprehension → Critic → Implementation Readiness), each against a written rubric, with a synthesizer that dedupes findings before the human sees them. A community adaptation (`vshvedov/elephant-goldfish`) adds two cheap, high-value moves: a **hidden-hypothesis diagnosis** in debugging (the fresh subagent sees only the symptom + repro, not the session's hypothesis), and a **fix-or-rebut-verbatim + hard-round-cap + user-escalation** shape for review loops.

## Goal

Adopt the EGM *method* — not the plugin — into skillgrid-v2's SDD pipeline, so that every blueprint is validated by fresh, context-free goldfish validators against written rubrics before slicing, and the two supporting disciplines (hidden-hypothesis debugging, capped rebuttal review loops) land where they already have a home.

## Success Criteria (measurable)

1. The goldfish protocol is the *only* pre-implementation design gate: `writing-blueprints` no longer names a bare "critic subagent"; the 4-stage protocol with a synthesizer is the named mechanism, and it is tier-gated per the rigor tiers.
2. The readiness gate is mechanical: a blueprint fails readiness unless it satisfies the **Zero-Decision Rule** (a fresh implementer can execute it with zero design choices), stated in the shared rule.
3. Every goldfish dispatch is fresh-context: the rule states that goldfish subagents receive only the named artifact(s) + the codebase — never the designer conversation.
4. Debugging gains the asymmetry step: a fresh subagent diagnoses from symptom + repro only, before it sees the session's hypothesis; convergence/divergence is recorded in the debug state file.
5. Review loops gain an explicit cap and escalation: review fix loops are capped at 3 rounds (aligned with the existing fix-loop cap in `_shared/planning/rigor-tiers.md`), every finding is fixed *or* rebutted verbatim (no silent dismissals), and hitting the cap escalates to the human with the open findings listed.
6. Shared-convention drift is zero: no skill references the old "critic subagent" mechanism after this change (grep-verifiable), and the new shared rule is referenced by every skill that now runs the protocol.

## Falsifiable Requirements

- **R1 (goldfish protocol rule).** The repo contains `.agents/skills/_shared/rules/goldfish-protocol.md` defining: (a) four stages in order — Stage 0 Zoom Out, Stage 1 Comprehension, Stage 2 Design Critic, Stage 3 Implementation Readiness; (b) the four goldfish personas with their audit vectors; (c) the common mandates (fresh subagent, read-only, no chitchat, structured output with severity Blocker/Major/Nit + affected scope + rationale); (d) the stage gating table (Stage 0/1/3 always on, Stage 2 mandatory at T2+ and skippable at T0/T1 with a recorded reason); (e) the Zero-Decision Rule as the Stage 3 pass bar; (f) the synthesizer step (dedupe + severity merge before human triage); (g) the loop: failure → route findings back to the designer (Elephant) → fix → re-validate, max 2 rounds, then escalate to the user with the open findings.
  - *True when:* all seven elements are present and each is stated as a rule a subagent can follow mechanically.
  - *False when:* any element is missing, or a stage's pass criteria are left to reviewer judgment without a written rubric.
- **R2 (writing-blueprints rewires the gate).** `skillgrid:writing-blueprints` (`.agents/skills/planning/writing-blueprints/SKILL.md`) runs the goldfish protocol in place of the old "critic subagent (conditional)" step: dispatches Stage 0 + Stage 1 + Stage 3 always, Stage 2 per the tier gate, feeds all findings to the synthesizer, and then runs the existing user-decide loop on the synthesized findings. The old step's skip conditions (T0/T1 or fast-track) now apply only to Stage 2, and the blueprint header line records per-stage results (`Goldfish: zoom-out ✓ comprehension ✓ critic skipped (T1) readiness ✓` or `…critic ✓…`).
  - *True when:* the old "Critic subagent (conditional)" block no longer exists, the four stages are dispatched as named subagents, and the header recording format is stated.
  - *False when:* the old block survives, or the user-decide loop runs on raw per-goldfish output instead of synthesized findings.
- **R3 (brainstorming no-code rule).** `skillgrid:brainstorming` (`.agents/skills/planning/brainstorming/SKILL.md`) carries an explicit **No-Code Rule** section: the Elephant session produces no code, no scaffolding, no file-layout decisions beyond what the spec requires; if the session catches itself sketching code, it stops and defers to the blueprint. It also cross-references the goldfish protocol ("the blueprint is validated by the goldfish protocol per `_shared/rules/goldfish-protocol.md` before it is committed").
  - *True when:* both the rule and the cross-reference are present in the skill body.
  - *False when:* either is missing.
- **R4 (debugging asymmetry).** `skillgrid:structured-debugging` (`.agents/skills/verification/structured-debugging/SKILL.md`) adds a **Goldfish Diagnosis** step in Phase 1: before the session states its own hypothesis as the working one, it dispatches a fresh read-only subagent that receives *only* the symptom + the repro steps (never the session's hypothesized root cause), and reports its top 1–3 candidate root causes with file:line + falsifying evidence. The session then compares: convergence (goldfish matches session hypothesis) → proceed with confidence; divergence → the goldfish's candidates enter the debug state file's hypothesis list first, and the user is told the goldfish flagged a different root cause. The outcome (converged/diverged) is appended to the state file entry.
  - *True when:* the step names the input restriction (symptom + repro only), the comparison, and the state-file record.
  - *False when:* the goldfish receives the session's hypothesis, or the divergence outcome is not recorded.
- **R5 (review loop cap + verbatim rebuttal).** `skillgrid:receiving-code-review` (`.agents/skills/verification/receiving-code-review/SKILL.md`) states the loop discipline: (a) review fix loops are capped at **3 rounds** (citing `_shared/planning/rigor-tiers.md` § Fix loop cap as the single source — not a new number); (b) every finding is resolved by *fix* or by *rebuttal recorded verbatim* in `review.md` — a finding may not disappear without one of the two; (c) hitting the cap or a repeated finding at the same file:method escalates to the user with the open findings listed verbatim.
  - *True when:* all three are present and the cap cites the shared source rather than restating a number.
  - *False when:* the cap is a new independent number, silent dismissal is still possible, or there is no escalation shape.
- **R6 (rigor tiers alignment).** `_shared/planning/rigor-tiers.md` is updated so that: T2's review column and the "What each skill does with the tier" entry for `writing-blueprints` name the goldfish protocol (Stage 2 mandatory); T3's fresh-model rule is extended to apply to the Stage 2 goldfish (the critic goldfish runs on a model different from the designer's when one is available; the human may decline, the decline is recorded — same posture as the existing T3 reviewer rule).
  - *True when:* both edits are present and the T3 extension reuses the existing "recommendation the human can decline" posture.
  - *False when:* the tier table still names only "two-axis"/"critic subagent", or the T3 rule for the goldfish critic is a new hard requirement.
- **R7 (structure + router rename).** `_shared/rules/sdd-structure.md` renames the phase `blueprint-validation (critic subagent + user)` to `blueprint-validation (goldfish protocol + user)` in: the Phase Order chain (line 10), the skill table row for `writing-blueprints`, and the Resume markers table. `lifecycle/using-skillgrid/SKILL.md`'s artifact-creation-order list (item 3) makes the same rename.
  - *True when:* a `grep -rn "critic subagent" .agents/skills/` returns zero hits in any pipeline file (the phrase may survive only inside `goldfish-protocol.md` itself as the description of what was replaced, if at all).
  - *False when:* any pipeline file still names "critic subagent" as the mechanism.
- **R8 (ADR + index).** `.skillgrid/knowledge/decisions/04-adr-0050-egm-goldfish-protocol.md` records the adoption: Context (the EGM post + the two repos, the four failure modes Google names, the gap analysis), Decision (4-stage protocol + Zero-Decision Rule + fresh-context mandate + the two supporting disciplines; the plugin itself NOT adopted, the single-doc design NOT adopted, `egm-drift-report` and `egm-bootstrap` parked as follow-ups), Consequences. A pointer row for 0050 is added to `ASSUMPTIONS.md` § LOCKED (in force: yes).
  - *True when:* the file exists with the frontmatter (`type: TechnicalDecision`, `status: accepted`, `date: 2026-10-25`) and the index row exists.
  - *False when:* either is missing, or the ADR restates operational rules instead of citing them.
- **R9 (config + AGENTS pointer).** `.skillgrid/config.yaml` gains a comment block under `rules:` pointing at `_shared/rules/goldfish-protocol.md` (no schema change, no new key). `AGENTS.md`'s `<!-- skillgrid:start -->` workflow line is updated: `writing-blueprints → blueprint-validation (goldfish protocol) → …`.
  - *True when:* both are present and no new config key was invented.
  - *False when:* a new YAML key was added, or the workflow line still names the critic subagent.

## Trust Boundaries

None new. All goldfish subagents are read-only, spawned in-process by the pipeline skill, and receive only named artifact paths + repo access — the same trust posture as the existing critic subagent and the parallel-code-review reviewers. No new network surface, no new filesystem write path, no Mnemonic tool-contract change.

## Scope (files)

- Create: `.agents/skills/_shared/rules/goldfish-protocol.md`
- Create: `.skillgrid/knowledge/decisions/04-adr-0050-egm-goldfish-protocol.md`
- Modify: `.agents/skills/planning/writing-blueprints/SKILL.md`
- Modify: `.agents/skills/planning/brainstorming/SKILL.md`
- Modify: `.agents/skills/verification/structured-debugging/SKILL.md`
- Modify: `.agents/skills/verification/receiving-code-review/SKILL.md`
- Modify: `.agents/skills/_shared/planning/rigor-tiers.md`
- Modify: `.agents/skills/_shared/rules/sdd-structure.md`
- Modify: `.agents/skills/lifecycle/using-skillgrid/SKILL.md`
- Modify: `.skillgrid/config.yaml` (comment only)
- Modify: `AGENTS.md` (workflow line only)
- Modify: `.skillgrid/knowledge/ASSUMPTIONS.md` (one index row)

## Open Questions

None blocking. Two items deliberately parked (recorded in the ADR, not decided here):
- `egm-drift-report` (spec-vs-code drift scanner) — a new tool, not a method; revisit when the docs surface grows.
- Divergent-lens brainstorming (vshvedov's inverted pattern for `brainstorming`) — a larger rework of the interview flow; revisit after one full pipeline run on the new protocol.
