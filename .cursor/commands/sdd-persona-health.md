---
name: sdd-persona-health
id: sdd-persona-health
category: Workflow
description: Validate Norse persona prompt packs, model routing, and surface readiness
agent: board
subtask: true
---

You are `board`. Run a persona-health readiness check.

SOURCE OF TRUTH:
- `docs/09-subagent-personas.md`
- `.agents/workflows/sdd-persona-route.md`
- `.configs/ide-model-mapping.json`
- `.configs/ide-persona-capabilities.json`

TASK:
1. Confirm required persona definitions exist.
2. Confirm model tier mapping exists for every configured surface.
3. Confirm each surface has an explicit capability tier and fallback policy.
4. Report blockers for missing prompt packs, missing routing entries, or unsupported mandatory features.

REQUIRED RETURN FORMAT:
- `status`: `completed | blocked | failed`
- `executive_summary`: `overview`, `used_tokens`
- `artifacts`: health report path(s) or `none`
- `next_recommended`
- `risks`
- `personas_invoked`
- `conflicts`
- `hitl_required`
- `accepted_decision`
