# The chain

One screen. The canonical phase order lives in `skillgrid:_shared/rules/sdd-structure.md` § Phase Order; this file restates only this skill's position in it and never re-specifies the whole chain.

**The strict sequence is the only order.** No divergence, no parallelism, no
out-of-order revisits. A skill MUST NOT run unless every preceding skill has
completed.

```text
skillgrid → brainstorm → questioning → research → prototype → blueprint → slicing → execute → test → adversarial-authoring → drift → ship
```

(`questioning`, `research`, `prototype` run inline in brainstorm/blueprint at their canonical position. `drift` is the single exception — may run on demand after `execute`.)

The eight delivery phases map onto this chain:

```text
Phase 0  Intake           onboarding (this skill) -> brainstorm (questioning via skillgrid:questioning)
Phase 1  Foundation       ADR records (_shared/adr-records) + skillgrid:research|skillgrid:prototype|sketch (research and prototype delegated to standalone skills; sketch still inline in blueprint)
Phase 2  Architecture     blueprint (5-phase plan) -> blueprint-validation (goldfish + user)
Phase 3  Drafting         blueprint Phase 5 (acceptance spec) -> slicing -> ticketing -> execute
Phase 4  Adversarial Audit qa + code-review (fresh context, read-only)
Phase 5  Revision Loop    receiving-code-review -> fix -> re-run (<=3 passes)
Phase 6  Final Score      qa gate + calibration + floor
Phase 7  Editorial Package document -> ship -> reflect
```

Rules:

- **Strict sequence.** The chain above is the only order. A skill that starts before its predecessor has completed is a phase-order violation.
- Artifact order is permanent: `briefing.md` → `blueprint.md` → `blueprint-validation` → `acceptance.feature` → `tasks.md` → code → `report.md` → `review.md`.
- The user gate after `slicing` is mandatory (`user-gate-slice`).
- Phase 4 is never skipped; no score before it.
- Fast-track may skip earlier planning only as `skillgrid:_shared/planning/fast-track.md` allows; it never skips the tail (`test` → `ship`).
- Phase 0's onboarding is part of this skill (`references/pipeline/onboarding.md`); Phase 3 and Phase 7 delegate to model-invoked skills by name.
