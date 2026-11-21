# Phase 2: Architecture

Turn the foundation into a blueprint with momentum: a tracer bullet, ordered waves, explicit doors, and a plan a reviewer can attack.

When to load: Phase 2. Role: orchestrator as architect, with the consistency specialist.

## Outputs

- `.skillgrid/change-specs/<topic>/blueprint.md` — the technical plan, validated. It carries:
  - the tracer bullet (the thinnest end-to-end thread that proves the seam);
  - ordered waves with per-wave integration evidence;
  - one-way doors, each with a reversibility line and a door check;
  - the threat matrix (Applicable / Not applicable, with reasons);
  - Global Constraints, including the anticipated file set for `ship-drift-check`;
  - the scope contract: planned files, lines and waves, with the reason for any overage;
  - the rollback boundary.
- `state.yaml` — `phase: blueprint`, then `blueprint-validated` after the gate.

## Rules

- Every work unit has a job no other unit does. If two share one, merge them now; it is cheaper than cutting later.
- Vary the shape of consecutive slices so a reviewer cannot predict the next one's template.
- The tracer proves the riskiest seam first, not the easiest file.
- The blueprint promises only what the acceptance scenarios will test.
- The planned file/line budget must add up to the scope contract. If it does not, fix the plan before coding.
- Delegate to `skillgrid:blueprint` for the format and the critic subagent; this prompt only fixes the genesis-specific content.

## Check-in

Show the tracer, the wave count, the doors, the threat rows that apply, and the one assumption most likely to be wrong. Gate: `blueprint-validation` (critic subagent + programmer).
