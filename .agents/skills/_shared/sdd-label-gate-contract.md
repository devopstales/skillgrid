# SDD Label Gate Contract

Applies to implementation and verification phases using task labels.

## Validator

All label validation is handled by the unified gate script (do not invoke the label script directly from phase prompts):

```bash
.skillgrid/scripts/sdd-gate.sh <phase> --change <change-name>
```

The gate script calls `validate-task-labels.sh` internally when `tasks.md` exists.

## Rules (now enforced programmatically)

- Validation failure is a hard gate failure (script exits 1)
- Missing or invalid `[Label: AFK|HITL]` metadata is blocking
- Git hooks enforce label validation pre-commit and pre-push
- No manual validation needed in phase prompts — the gate script is the source of truth

## Status Mapping

- Apply phase: gate script exit 1 → phase returns `status: blocked`
- Verify phase: gate script exit 1 → phase returns `status: failed` with CRITICAL gate finding
- Git hooks: commit/push blocked on exit 1

## Output Fields (on gate failure)

- `status`: `blocked` or `failed`
- `executive_summary`: gate error details
- `artifacts`: gate output evidence
- `next_recommended`: explicit remediation
- `risks`: workflow-quality risk
