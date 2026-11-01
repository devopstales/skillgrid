---
id: TASK-080
title: >-
  mcp: mem_save should EnsureSession before insert (FK:
  observations_core.session_id)
status: done
assignee: []
created_date: '2026-10-18 08:30'
updated_date: '2026-10-18 08:55'
labels: []
dependencies: []
references:
  - mnemonic/internal/mcp/tools_memory.go
  - mnemonic/internal/memory/obs_split.go
  - mnemonic/internal/memory/session_agent.go
  - mnemonic/internal/mcp/tools_memory_test.go
priority: medium
type: bug
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Current: MCP mem_save with a named session_id that has no sessions row fails at the core INSERT with FOREIGN KEY constraint failed (observations_core.session_id -> sessions.id, FK ON via mnemonic/internal/store/store.go:186, first touch at obs_split.go:162) and rolls back the whole save. Repro observed live on 2026-10-18: session_id 'ship-2026-10-18-swagger-ui-air-gap' (never registered; all sessions are UUIDs from mem_session_start).

Expected: the save succeeds and the session row self-heals. handleMemSave (mnemonic/internal/mcp/tools_memory.go:186) never calls EnsureSession before SaveWithAction, while every other session-aware handler does (http/toolcalls.go:50, http/policy.go:72, http/server.go:378) — that is the gap.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 handleMemSave with an unregistered named session_id returns the observation id (action add) and a sessions row for that id exists afterward
- [ ] #2 An already-registered session_id behaves exactly as before (idempotent EnsureSession, no duplicate sessions row)
- [ ] #3 cd mnemonic && go build ./... && go test ./internal/mcp/... green; gofmt clean
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Regression test added and red-then-green (TDD); commit footer carries Refs: TASK-080
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. RED: in mnemonic/internal/mcp mirror tools_memory_test.go — seed a store, call handleMemSave with a brand-new named session_id (no sessions row, no prior mem_session_start); assert it returns the observation id and a sessions row for that id now exists.
2. GREEN: in handleMemSave (tools_memory.go:186), after openNamed/retargetAlias resolve (h + projectID known) and before SaveWithAction (line 280), call h.Memory().EnsureSession(ctx, sessionID, projectID, h.Root(), "", harnessFromContext(ctx)) and ignore the created bool — mirror http/toolcalls.go:50, http/policy.go:72, http/server.go:378.
3. Keep the MCP tool schema and mem_save argument shape unchanged (plugin ~/.config/opencode/plugins/mnemonic-memory.ts passes args through).
4. Verify: cd mnemonic && go build ./... && go test ./internal/mcp/... (new test + existing suite green) && gofmt -w on touched files.
5. Commit: fix(mcp): EnsureSession in handleMemSave before SaveWithAction — footer "Refs: TASK-080".
<!-- SECTION:PLAN:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-18 08:55
---
Implemented c0c1673e (fix(mcp): EnsureSession in handleMemSave before SaveWithAction). TDD: RED FK (787) repro + GREEN + triangulate idempotency. Full MCP suite green (258s), gofmt clean, go build ./... clean.
---
<!-- COMMENTS:END -->
