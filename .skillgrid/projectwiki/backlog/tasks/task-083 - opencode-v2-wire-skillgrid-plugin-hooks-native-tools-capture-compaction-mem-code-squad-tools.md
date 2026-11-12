---
id: TASK-083
title: >-
  opencode v2: wire skillgrid plugin hooks + native tools (capture, compaction,
  mem/code/squad tools)
status: done
assignee: []
created_date: '2026-10-18 10:35'
updated_date: '2026-10-18 11:08'
labels: []
dependencies: []
priority: medium
type: feature
---

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Comments

<!-- COMMENTS:BEGIN -->
created: 2026-10-18 11:08
---
RESOLVED (06bfe639): wired v2 hooks natively. skillgrid-compaction setup now subscribes to SDK SSE (session.created/idle/compacted + session.next.compaction.started) and posts/gets the same session + compaction endpoints the v1 event hook hit. skillgrid-events setup subscribes to session.next.tool.called/success/failed and captures tool calls to /memory/session-events (mirroring v1 tool.execute.after); the blocking tool.execute.before policy gate has no v2 equivalent and stays v1-only (best-effort in v2). Added shared skillgrid-v2sse.ts (base-URL resolution + reconnecting SSE async generator, fail-open). Split the 3 tool plugins' 21 tool bodies into plugins/opencode/tools/<name>.ts (one file per tool) for the OpenCode v2 tools/ loader (filename = tool name; no in-plugin tool registration in v2). Verified: 5/5 plugins load in live v2.0.28 (zero 'failed to load plugin'), 21/21 tool files parse with a valid tool() default export. tools/ scanned at server boot -> full restart needed for the live session (MCP path serves the tools meanwhile).
---
<!-- COMMENTS:END -->
