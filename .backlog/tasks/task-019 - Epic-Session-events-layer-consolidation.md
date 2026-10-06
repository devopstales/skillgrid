---
id: TASK-019
title: 'Epic: Session events layer consolidation'
status: done
assignee: []
created_date: '2026-09-21 15:09'
updated_date: '2026-10-06 08:35'
labels: []
milestone: m-0
dependencies: []
references:
  - .skillgrid/specs/2026-09-21-session-events-layer/briefing.md
  - .skillgrid/specs/2026-09-21-session-events-layer/blueprint.md
  - .skillgrid/specs/2026-09-21-session-events-layer/tasks.md
  - .skillgrid/specs/2026-09-21-session-events-layer/acceptance.feature
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Replace checkpoint.json plus the Handoff Hub with a single session-to-events layer.

Current: checkpoint.json (derived resume handle) plus Handoff Hub SQLite (change_snapshots, checkpoints, handoff_refs) plus cleave relay (session_handoffs, session_archives).

Expected: sessions table extended with commit range plus counters; append-only session_events stream ordered by (session_id, sequence); capture via RunHook; SessionChanges read path; old layer deleted; skills repointed; per-plan sdd workspaces.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 SessionChanges returns every tool call in order plus net diff
- [ ] #2 Old surfaces absent (handoff CLI, hub/relay MCP tools, snapshot/restore, checkpoint.json)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Tracer (TICKET-01): 040 migration plus session start/end events plus SessionChanges
2. Capture (TICKET-02/03/04): tool events plus redaction, commit events, CLI plus MCP read
3. Removals (TICKET-05/06/07/07b): hub pkg plus CLI, MCP tools, relay plus HTTP plus UI plus bash
4. Repoint (TICKET-08/09): skills to events model, per-plan sdd scripts
5. Close (TICKET-10): 041 drops old tables
<!-- SECTION:PLAN:END -->
