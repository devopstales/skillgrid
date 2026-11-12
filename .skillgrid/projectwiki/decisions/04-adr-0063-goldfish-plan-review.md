---
type: TechnicalDecision
status: accepted
accepted: 2026-11-12
supersedes: "0054 (04-adr-0054-goldfish-plan-review.md — the 3-stage same-context-adjacent plan-review subflow)"
date: 2026-11-12
---

# ADR-0063 — Goldfish plan review: before-code plan review is a fresh-context 5-stage multi-persona Goldfish reference set inside writing-blueprints, not a new phase and not a new top-level skill

Change: 2026-11-02-egm-goldfish-brainstorm-shootdown

Supersedes: ADR-0054

## Context

skillgrid's before-code review of a `blueprint.md` was a same-context Self-Review plus a single generic "Critic subagent" pass plus one structural Plan Review. All three run with the author's assumptions already in context, so they rubber-stamp the very blind spots they exist to catch — unstated assumptions, terms a fresh reader cannot resolve, altitude drops, and gaps that make the plan unbuildable by a zero-context implementer. EGM's `egm-review` + `egm-review-loop` + `egm-resolve` show that a review performed by fresh, zero-context subagents — given only the document and the files it references — exposes exactly those blind spots, and that a multi-round sharpening loop with tracked convergence beats a single pass.

ADR-0054 locked the placement: the fresh-context Goldfish review is a subflow of `writing-blueprints`'s validation seam, not a new pipeline phase and not part of `qa`. This ADR keeps that placement but replaces ADR-0054's 3-stage subflow (zoom-out / comprehension / design-critic + implementation-readiness, one pass) with the fuller EGM port. The original draft of this change specified the review as a new top-level skill (`skillgrid:goldfish-review`); Amendment A1 (2026-11-11, owner-approved) retracted that — the owner's constraint is "we have too many skills" — and restored the subflow placement.

## Decision

Before-code plan review is a fresh-context 5-stage multi-persona Goldfish reference set inside writing-blueprints (subflow, not a new top-level skill — A1), with a sharpening loop and Elephant-side resolution recorded in `review-ledger`; the after-code half (qa / code review) is a separate, unchanged gate.

- **Reference set, not a skill** (R6, R14): `writing-blueprints/references/goldfish-review.md` (5-stage pipeline + 4 presets + checkpoint gates + pause/resume + execution constraints), `goldfish-review-loop.md` (sharpening loop), `goldfish-resolve.md` (Elephant-side resolution), `harnesses.md` (harness portability table). `writing-blueprints` invokes the set in its validation seam (Self-Review → Owed-Decision Gate → Goldfish review → User Review Gate → Execution Handoff); `references/plan-review.md` is deleted; the set stays runnable standalone on any design doc (blueprint, briefing, ADR).
- **Five-stage fresh-context pipeline** (R7): Initial Zoom Out → Comprehension Test → Critic Review → Intermediate Zoom Out → Implementation Readiness. Each stage dispatches fresh `reviewer` (fallback `general`) subagents given ONLY the target file path(s) and the files it references — no conversation history, no author narrative. Stage 8 (Intermediate Zoom Out) re-reads the whole doc after Stages 6+7 fixes and catches altitude drift introduced by remediation — the key addition over ADR-0054's 3 stages.
- **Four execution presets** (R8): Interactive (checkpoint after each stage; default for New Project / New Function), Consolidated (no pauses, findings deduped into one report), Feature-spec-only (stages 5–8), Implementation-only (stage 9). Single-select; the topological order 5→6→7→8→9 is preserved.
- **Sharpening loop** (R9): up to 5 rounds per stage; convergence = 2 consecutive clean rounds (one clean round is not enough — a fresh reviewer can find a different gap in the same text); settled findings (contested / accepted-without-fix) are not re-opened; a fix does not settle a finding (re-raised at the same scope → the fix failed); churn (a new finding at the scope of an applied fix) asks the user; turn-taking (the loop idles between rounds, no auto-advance); a frozen-candidate re-hash before each round invalidates the round on drift.
- **Elephant-side resolution** (R10): each round's findings are resolved by the main agent, not a subagent — categorize each finding CONTEST / ESCALATE / PROPOSE-FIX / DECLINE (an adapter over `code-review`'s triage verdicts + `receiving-code-review` buckets, plus the doc-review deltas), apply the rules (a contest must be checkable; don't add context for subagents; don't contest a comprehension gap; don't change a severity), present the interactive gate (auto-accept / continue / interactive breakdown / comment on file / custom instruction). The report is the `review-ledger.md` round entry — per-finding AI suggestion + justification + proposed edit; no separate `goldfish_review_report.md`.
- **Records** (R11–R13, R17): 13 persona briefs (7 review + 6 lens) live in `_shared/elephant-goldfish/`; every brief and the review reference cite `_shared/craft/review-discipline.md`; stage reports commit to the change folder in the spec zone; a parked CRITICAL surfaces at qa time as a WARNING (never CRITICAL) via `qa`'s debt-surfacing step.
- **What changes vs ADR-0054:** the 3-stage one-pass subflow becomes the 5-stage pipeline with the sharpening loop, 4 presets, and ledger-recorded Elephant-side resolution. What does not change: the placement (subflow of the validation seam, not a phase, not qa), the fresh-context isolation mandate, and the after-code half, which remains a separate, unchanged gate.

## Alternatives considered

- **A new top-level `goldfish-review` skill** (the original draft) — rejected, reinstated as Chosen by Amendment A1: it duplicates the seam, grows the skill roster, and deviates from ADR-0054's subflow placement. The review is a subflow of blueprint validation, not a new pipeline phase.
- **Keep ADR-0054's 3-stage one-pass subflow** — superseded: a single pass has no convergence tracking, and no stage re-reads the doc after fixes, so altitude drift introduced by remediation is not caught.
- **A single re-review after the first pass (max 1)** — superseded by the sharpening loop: max-1 has no convergence rule, no settled/fixed finding tracking, and no churn detection.
- **Register new named OpenCode subagents per stage persona** — rejected: the harness does not make that cheap; the persona-brief-file fallback is the faithful, lower-friction port.

## Consequences

- Good, because the design is checked at the cheap point (before code) by five fresh-context stages, and the Intermediate Zoom Out catches the blind spot ADR-0054's 3-stage pass could not: altitude drift introduced by the fixes themselves.
- Good, because the loop is bounded and observable: convergence (2 clean rounds) or cap (5) or churn, with turn-taking — no silent auto-advance and no unbounded re-review.
- Good, because resolution runs in the Elephant context, where the institutional knowledge to write good fixes and defend design choices lives; findings and dispositions are ledger rounds, so parked CRITICALs feed forward to qa as decision-debt.
- Good, because no new pipeline phase, no new top-level skill, and no `sdd-structure.md` change — ADR-0054's placement is preserved.
- Bad, because the review is far more subagent work than the old single Critic pass (5 stages × up to 5 rounds) — time-constrained reviews use the Consolidated preset.
- Bad, because the before-code review remains advisory-to-the-gate: it catches design gaps but does not render PASS/FAIL; its force comes from committed findings and the parked-CRITICAL → qa decision-debt path, not from blocking execution.
- Neutral, because the reference set remains runnable standalone on any design doc; the persona briefs are shared with Track A's shootdown (ADR-0062) and Track D's drift mode (ADR-0064).
- Neutral, because ADR-0054 stays in the index (frozen, In force: no) as the historical record of the 3-stage subflow.
