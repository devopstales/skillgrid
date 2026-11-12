---
id: TASK-111
title: 'TICKET-05: Prefetch Injection (Plugin + MCP Resource + CLI)'
status: needs-triage
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies:
  - TASK-110
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** `facts.RenderFrozenSnapshot()` — renders all facts sorted by importance, two labeled sections (GLOBAL MEMORY + USER PROFILE), `\n§\n` separators, usage headers. OpenCode V2 plugin (`memory-prefetch.ts`) — idempotent per session (memoize by session ID), fail-open. MCP resource `memory/global`. CLI `skillgrid memory global prefetch`.

**Acceptance:** Output contains both section headers with usage % + char counts. Entries separated by `\n§\n`. Sorted by importance DESC. Plugin: second call with same session ID returns cached result. CLI error → empty string (fail-open). Empty store → empty sections, no error.

**SATISFIES:** `happy path — prefetch renders two labeled sections`, `edge — prefetch is frozen mid-session`, `failure — prefetch with empty store renders empty sections`

**Files:** `mnemonic/internal/facts/prefetch.go` (create), `mnemonic/internal/facts/prefetch_test.go` (create), `plugins/opencode/src/memory-prefetch.ts` (create), `mnemonic/cmd/mnemonic/memory.go` (modify), `mnemonic/internal/mcp/server.go` (modify)

**Size:** ~500 (M)
**Blocked by:** TICKET-04
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 output contains both section headers with usage % + char counts
- [ ] #2 entries separated by \n§\n
- [ ] #3 sorted by importance DESC
- [ ] #4 plugin: second call with same session ID returns cached result
- [ ] #5 CLI error → empty string (fail-open)
- [ ] #6 empty store → empty sections, no error
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
