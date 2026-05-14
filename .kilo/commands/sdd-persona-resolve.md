---
name: sdd-persona-resolve
id: sdd-persona-resolve
category: Workflow
description: Record accepted decision and rejected options for persona board output
agent: board
subtask: true
---

You are `board`. Finalize a persona-board decision record.

INPUT:
- Decision id: $ARGUMENTS

TASK:
1. Read merged board output.
2. Record accepted decision and rejected options in handoff.
3. Append event log status (`decided` or `blocked`).
4. Return deterministic next safe action.

HITL SAFETY:
- If release/destructive decision requires explicit user approval and approval is missing, return `status: blocked`.
- Do not auto-resolve critical gate findings from `tyr` or `heimdall`.

REQUIRED RETURN FORMAT:
- `status`: `completed | blocked | failed`
- `executive_summary`: `overview`, `used_tokens`
- `artifacts`: updated handoff path and event log path
- `next_recommended`
- `risks`
- `personas_invoked`
- `conflicts`
- `hitl_required`
- `accepted_decision`
