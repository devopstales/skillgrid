# Phase 3: Drafting

Write the program work unit by work unit, in the program's own conventions, saving each unit before starting the next.

When to load: Phase 3. Roles: implementer builds; orchestrator runs the checks and the panel.

## Before each work unit: the unit brief

Assemble `.skillgrid/work/<YYYY-MM-DD-topic>/briefs/<unit>.md` for the implementer from files, never from memory:

- the unit's entry in `tasks.md`, including its Files column (the edit scope), its test, and its rollback boundary;
- the unit's row in the blueprint's wave plan: entry and exit state, the seam it proves, the evidence required;
- the terms and invariants it touches, from `projectwiki/glossary/` and `invariants/`;
- the interface and error contract from `blueprint.md`;
- the ledger entries the unit touches, from the consistency ledger;
- the last accepted diff or the neighbouring module's interface.

The brief never contains the rubric, the quality target, panel verdicts or audit findings. The implementer writes for the next maintainer, not for a score.

## Writing

Follow `references/specialists/code-craft.md` for entry points, seams, error messages and tests, and `references/patterns/code-patterns.md` for the measured ranges. While writing:

- Hold the conventions: naming, layering, error model, test style.
- Give every unit at least one thing a maintainer could still recall tomorrow: a named seam, an actionable error, a test that names the behavior.
- Trust the domain. Do not explain the design inside comments; let names and tests carry it.
- Do not run the machine-code checks while writing. Write first; the tells are for revision.

## After each work unit

1. Save the attempt to `.skillgrid/work/<YYYY-MM-DD-topic>/attempts/<unit>/attempt-1.md`, read it back, then apply it to the working tree.
2. Run the focused test and the acceptance scenario; record the command and its output.
3. Check: the unit's job is clear, names and contracts match the ledger, the conventions are recognizable, the exit state pulls forward, and the unit adds something the previous one did not.
4. Update `tasks.md` `[x]`, `progress.md`, and `state.yaml` (`genesis` unchanged).

## First-unit checkpoint (tracer)

Run the user panel on the tracer unit (`references/roles/panel.md`, unit read). Save the verdict. If the panel does not reach a majority to keep using it, revise the unit once against the two-vote defects, run the panel again in compare mode, keep the preferred version, and report both results. Then check in.

## Every wave

After each wave:

1. Update the ledger and run the consistency check (`references/specialists/consistency.md`). Send any finding to the fix editor as a targeted ticket and fix it before continuing.
2. Run the integration test and record the command and output.
3. Run the panel on the latest user-visible surface.
4. Compare the running line count with the scope contract. If the change is heading more than 15 percent over the plan, propose where to cut before writing on.
5. Check in.

## Finishing the draft

The phase ends when every planned work unit exists and has been read back. Set the scope gate to `PASS` (at or under the plan), `FLAG` (within 10 percent over) or `BLOCK` (further over). A change over its budget goes to scope review before Phase 6 can call it complete, unless the intake declared a larger exception.
