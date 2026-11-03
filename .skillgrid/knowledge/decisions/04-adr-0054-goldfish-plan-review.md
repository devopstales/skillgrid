# Goldfish plan review: before-code document review is a fresh-context subflow of writing-blueprints, not a new phase and not part of qa

---
status: "accepted"
supersedes: none
date: 2026-11-01
---

## Context and Problem Statement

skillgrid's before-code review of a blueprint was a same-context Self-Review plus a single generic "Critic subagent" pass plus a structural Plan Review. All three run with the author's assumptions already in context, so they rubber-stamp the blind spots they exist to catch. The Elephant-Goldfish Model (EGM, `github.com/google/elephant-goldfish-model`) shows that a review performed in a *fresh, zero-context* subagent — given only the document and the files it references — exposes exactly the unstated assumptions a same-context reviewer has already internalized. The question was where to put that review.

Two failure classes exist at two different pipeline points: (a) a wrong *design* (caught before code, on the document) and (b) a wrong *build* (caught after code, on the diff). skillgrid's `qa` gate owns (b) strongly. (a) was under-served. The risk was conflating them — adding EGM's document review to `qa` would merge the before-code and after-code gates and move the design check to the expensive point.

## Decision Outcome

The fresh-context Goldfish review is a **subflow of `writing-blueprints`'s existing validation seam** (between Self-Review and the Execution Handoff), not a new pipeline phase and not part of `qa`. It is a 3-stage review — zoom-out, comprehension, and design-critic + implementation-readiness — where each stage dispatches a freshly spawned `reviewer` (fallback `general`) subagent given only the target file and its referenced files, with no conversation history. EGM's named Goldfish subagents are realized as persona brief files under `.agents/skills/_shared/craft/goldfish/` that the subagent reads first — EGM's own documented fallback for harnesses that do not register named subagents. The `qa` gate, `requesting-code-review`, `parallel-code-review`, and `review-ledger` are unchanged; the only cross-seam link is that a parked CRITICAL blueprint-review finding surfaces as a WARNING through `qa`'s existing decision-debt path.

### Consequences

- Good, because the design check happens at the cheap point (before code) with fresh context, and the two failure classes stay separate. skillgrid's file boundary (`briefing.md`/`blueprint.md`) is a cleaner split than EGM's header scan, so different rubrics apply per file without parsing.
- Good, because no new phase, resume marker, or `sdd-structure.md` change is required — the review is a validation subflow, and its report artifacts land in the spec zone and are committed (ADR-0013: repo is source of truth).
- Bad, because the before-code review is advisory-to-the-gate: it catches design gaps but does not itself render a PASS/FAIL. Its force comes from committing the findings and feeding parked CRITICALs forward as decision-debt, not from blocking execution.

## Revisit

When a drift-report (doc vs. merged code) change lands and the after-code half needs the same persona briefs, or when the persona briefs prove useful for `briefing.md` review and the what-side gets its own stage.
