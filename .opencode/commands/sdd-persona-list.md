---
name: sdd-persona-list
id: sdd-persona-list
category: Workflow
description: List Norse personas, mapped roles, and runtime availability
agent: board
subtask: true
---

You are `board`. Enumerate the Norse persona registry and readiness.

INPUT:
- Optional filter: $ARGUMENTS

SOURCE OF TRUTH:
- `docs/09-subagent-personas.md`
- `.agents/workflows/sdd-persona-route.md`

TASK:
1. List personas: primary **`odin`**; board chair **`board`**; specialists **`kvasir`**, `thor`, `tyr`, `heimdall`, `frigg`, `loki`, `mimir`, `bragi`, `vidar`.
2. Show role intent and replaced legacy persona mapping.
3. Report availability/readiness by surface when possible.
4. Highlight missing persona prompt packs or unsupported surfaces.

REQUIRED RETURN FORMAT:
- `status`: `completed | blocked | failed`
- `executive_summary`: `overview`, `used_tokens`
- `artifacts`: supporting paths or `none`
- `next_recommended`
- `risks`
- `personas_invoked`
- `conflicts`
- `hitl_required`
- `accepted_decision`
