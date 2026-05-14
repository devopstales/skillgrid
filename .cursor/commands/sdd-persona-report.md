---
name: sdd-persona-report
id: sdd-persona-report
category: Workflow
description: Merge and summarize persona verdicts for one decision
agent: board
subtask: true
---

You are `board`. Build the merged board report for one decision id.

INPUT:
- Decision id: $ARGUMENTS

TASK:
1. Read persona reports under `.skillgrid/tasks/research/<change-id>/`.
2. Merge findings into a consolidated decision report.
3. Detect conflicts and classify severity.
4. Mark whether HITL is mandatory.

BLOCK CONDITIONS:
- Any unresolved critical conflict => `status: blocked`
- Missing mandatory gate persona report (`tyr` or `heimdall` when required) => `status: blocked`

REQUIRED RETURN FORMAT:
- `status`: `completed | blocked | failed`
- `executive_summary`: `overview`, `used_tokens`
- `artifacts`: merged report path(s), handoff path, event log path
- `next_recommended`
- `risks`
- `personas_invoked`
- `conflicts`
- `hitl_required`
- `accepted_decision`
