# Orchestrator

The main agent's job: turn one idea into a complete program by running the eight phases in order, one active prompt at a time, with the programmer agreeing at each checkpoint.

When to load: at the start of every session, after `references/pipeline/host-contract.md`. Role: orchestrator.

## The loop

1. Apply `references/pipeline/host-contract.md` (start, resume, reconcile).
2. Read the change entry in `.skillgrid/state.yaml`. `state.yaml` means `.skillgrid/state.yaml`; resolve every `references/...` path from this skill's folder, never from `.skillgrid/`.
3. Derive the phase from the deepest artifact present. Artifacts beat state: if the entry disagrees, trust the artifact and patch the entry.
4. Load that phase's prompt and the references listed for it in `references/pipeline/manifest.yaml`, and nothing else.
5. Produce exactly the outputs the manifest lists for the phase. Save, read back, then update `phase`, `gate` and `changes.<slug>.genesis` in the same edit.
6. Check in with the programmer (host contract), unless the mode is autonomous.
7. Move `phase` to the next label and repeat.

## Non-negotiable rules

- Phase order is fixed. Never skip Phase 4, and never score before it.
- The producer never sees the rubric, targets or verdicts. Critics never see targets or earlier scores. You keep both apart (`references/scoring/evaluator-protocol.md`).
- Every important decision goes to a file: `ASSUMPTIONS.md` for inferences, an ADR in `projectwiki/decisions/` for locked choices, a `projectwiki/concepts/` page when the idea justifies one, and `report.md` for what happened.
- Every new or revised work unit is written to `.skillgrid/work/<YYYY-MM-DD-topic>/` first; the working tree changes only after the new text is read back and accepted.
- Revision is bounded: three passes per gate (`references/prompts/revision-loop.md`).
- Write the program, its artifacts and every screen in the programmer's language. Keep file names, state keys and gate ids in English.
- Specialist names are roles, not proof of separate processes. State what actually ran where, and at what independence grade.
- This skill writes `.skillgrid/` only in the programmer's project root. It never creates `.skillgrid/` inside its own skill folder.

## Phases

| Phase | Prompt | Main outputs |
|---|---|---|
| Phase 0: Intake | `references/prompts/intake.md` | brief, prior-art map, user personas, independence setup |
| Phase 1: Foundation | `references/prompts/foundation.md` | glossary, ADRs, invariants, interfaces, threat matrix |
| Phase 2: Architecture | `references/prompts/architecture.md` | validated blueprint with tracer, waves, doors, rollback |
| Phase 3: Drafting | `references/prompts/drafting.md` | acceptance scenarios, tasks, work units, tracer and wave checks |
| Phase 4: Adversarial Audit | `references/prompts/adversarial-audit.md` | audit with tickets and `audit_status` |
| Phase 5: Revision Loop | `references/prompts/revision-loop.md` | review gate and score gate, at most three passes each |
| Phase 6: Final Score | `references/scoring/program-score.md` | final report with completeness, gates and independence |
| Phase 7: Editorial Package | `references/prompts/editorial-package.md` | README, PR body, changelog, archive move |

## Starting from the middle

When the programmer brings existing material, start at the phase that material needs: a finished program starts at Phase 4; a program missing its package starts at Phase 7; a partial program starts at Phase 3 after you rebuild the Phase 1 and 2 artifacts it implies and the programmer agrees with them.
