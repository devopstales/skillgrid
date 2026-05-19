# SDD Enforcement Contract

Canonical runtime enforcement source for `sdd-orchestrator` and `sdd-*` phases.

## 0) Executable Gate Script

The `sdd-gate.sh` script at `.skillgrid/scripts/sdd-gate.sh` is the single source of truth for all gate enforcement.

- All gates are **programmatic**, not procedural
- Exit code 0 = gate passed, exit 1 = gate failed
- Run via: `sdd-gate.sh <phase> --change <change-name>`
- Git hooks enforce gates pre-commit (staged openspec changes) and pre-push (only changes in the push range) via `install.sh` / `skillgrid install`

## 1) Phase Routing And Stop Conditions

- Use explicit phase transitions; do not silently skip phases.
- `HITL` unresolved decision => `status: blocked`.
- `AFK` continuation only when scope, acceptance criteria, and checks are explicit.
- Missing mandatory artifact or gate output => fail closed with `status: failed`.

## 2) Mandatory Skill-Gate Matrix

All gates run through `sdd-gate.sh`:

| Phase | Gates Run |
|-------|-----------|
| brainstorm | labels, artifacts, phase_state |
| propose | artifacts, phase_state |
| spec | labels, artifacts, phase_state |
| design | labels, artifacts, phase_state, persona_routing |
| tasks | labels, artifacts, phase_state, persona_routing |
| apply | labels, artifacts, phase_state, persona_routing, slices |
| verify | labels, artifacts, phase_state, two_stage_review, persona_board, slices |
| archive | labels, artifacts, phase_state, two_stage_review, persona_routing, persona_board |

### Artifact requirements (enforced by `gate_artifacts`)

| Phase | Required files |
|-------|----------------|
| propose | `proposal.md` |
| brainstorm | `proposal.md`, `design.md`, `tasks.md`, `specs/**/spec.md`; if `ui_scope: true` in proposal → `ui-wireframes.md`, `ui-decisions.md` |
| spec | `proposal.md`, `specs/**/spec.md` |
| design | `proposal.md`, `design.md`, `specs/**/spec.md` |
| tasks | `proposal.md`, `design.md`, `specs/**/spec.md` |
| apply / verify / archive | `proposal.md`, `design.md`, `tasks.md`, `specs/**/spec.md` |

Labels are enforced only when `tasks.md` exists (skipped for propose/spec/design without tasks).

## 3) Two-Stage Review Gate

All decision-ready reports must include:

1. Stage 1: spec compliance
2. Stage 2: code quality

Any critical finding in either stage blocks progression.

## 4) Gate Failure Handling

Gate failures are **programmatic hard blocks**:

- `sdd-gate.sh exit 1` → phase MUST exit with `status: "blocked"` or `status: "failed"`
- Label validation runs inside `sdd-gate.sh` (do not call `validate-task-labels.sh` separately in phase prompts)
- No manual gate checks needed in prompt instructions — the script is the source of truth

## 5) Standard Return Envelope

Use `skills/_shared/sdd-return-envelope.md` as canonical envelope contract.
