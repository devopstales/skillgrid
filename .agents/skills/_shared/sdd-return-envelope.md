# SDD Return Envelope Contract

All `sdd-*` phase skills, slash commands, and workflows must return this envelope when the phase ends (success, block, or fail). Return it as **JSON or YAML** — not prose-only summaries.

Canonical path: `.agents/skills/_shared/sdd-return-envelope.md`

## Required Fields

| Field | Type | Notes |
|-------|------|-------|
| `status` | `completed \| blocked \| failed` | Phase outcome |
| `executive_summary` | object | See below |
| `detailed_report` | string or object | Optional verbose report; phase-specific keys allowed inside |
| `artifacts` | string[] | Paths or artifact keys written or validated |
| `next_recommended` | string | Deterministic next safe action or command |
| `risks` | string or string[] | Open/accepted risks, or explicit `none` |
| `skill_resolution` | enum | `injected \| fallback-registry \| fallback-path \| none` |

### `executive_summary` (required)

- `overview`: 1–3 sentence summary
- `used_tokens`:
  - `input`, `output`, `total`, **or**
  - `not_available` with reason

## Failure / Block Requirements

When `status` is `blocked` or `failed`, also include (top-level or inside `detailed_report`):

- `stop_condition`: exact stop condition
- `failing_gate`: gate id if any (e.g. `sdd-gate apply`)
- `missing_artifacts` or `missing_evidence`: list if any
- explicit remediation in `next_recommended`

## Canonical Example

```json
{
  "status": "completed",
  "executive_summary": {
    "overview": "Wrote delta spec for auth slice; three requirements with scenarios.",
    "used_tokens": { "input": 12000, "output": 3400, "total": 15400 }
  },
  "detailed_report": "Optional verbose narrative or structured sub-report.",
  "artifacts": [
    "openspec/changes/add-auth/specs/auth/spec.md"
  ],
  "next_recommended": "/sdd-design add-auth",
  "risks": "none",
  "skill_resolution": "injected"
}
```

```yaml
status: completed
executive_summary:
  overview: "..."
  used_tokens:
    input: 0
    output: 0
    total: 0
detailed_report: null
artifacts:
  - path/to/artifact.md
next_recommended: "/sdd-apply <change>"
risks: none
skill_resolution: fallback-registry
```

## Phase Extensions (add to the same envelope)

Do not omit required base fields. Add only what applies:

| Phase / command | Extra fields or notes |
|-----------------|----------------------|
| `sdd-verify` | `converged`, `hallucination_ratio`, `legitimate_flaws` (VDD converge); verification matrix in `detailed_report` |
| `sdd-loop` | When all AFK tasks done, include `<promise>COMPLETE</promise>` in `executive_summary.overview` |
| `sdd-design-ui` | `detailed_report` may include `loaded_required_skills`, `loaded_optional_skills`, `missing_skills` |
| Norse persona subagent | `persona`, `capability`, `findings_severity`, `hitl_required` — see `docs/09-subagent-personas.md` |

## Standard Instruction (every phase)

When the phase completes, return **only** the canonical envelope (JSON or YAML). Load executor protocol from `skills/_shared/sdd-phase-common.md`. Persona commands also load the extended fields above.

**One-liner for skills/commands:**

> Return the standard SDD envelope per `skills/_shared/sdd-return-envelope.md` (or `.agents/skills/_shared/sdd-return-envelope.md` from repo root).
