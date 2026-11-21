# Phases at a glance

A one-screen map of the eight phases, what each produces, and where the programmer is asked to agree.

When to load: when you need an overview without loading every prompt. Role: orchestrator.

| Phase | Produces | Check-in |
|---|---|---|
| Phase 0: Intake | assumptions, brief with user personas, prior-art map, risk class and Definition of Done, independence setup | yes |
| Phase 1: Foundation | glossary, ADRs, invariants, interfaces and error model, threat matrix | yes |
| Phase 2: Architecture | validated `blueprint.md`: tracer, waves, one-way doors, rollback, scope contract | yes |
| Phase 3: Drafting | `acceptance.feature`, `tasks.md`, work units; tracer and wave checkpoints | after the first work unit, after every wave, and at the end |
| Phase 4: Adversarial Audit | read-only audit with quoted tickets and `audit_status` | yes; always on `major_rewrite` |
| Phase 5: Revision Loop | review gate and score gate, at most three passes each, draft comparison | yes; always when a gate fails its third pass in check-in mode |
| Phase 6: Final Score | completeness, review, score, gates and independence in one report | yes |
| Phase 7: Editorial Package | README, PR body, changelog, release note, archive move | final inventory |

Order is fixed. Phase 4 is never skipped. Scores come after the audit. In autonomous mode the check-ins are written to `report.md` instead of stopping, except `major_rewrite`.
