---
id: TASK-018
title: '[FEATURE] SDD spec: 2026-09-17-compact-search-output (context/unfold params)'
status: ready-for-agent
assignee: []
created_date: '2026-09-21 07:15'
updated_date: '2026-09-21 07:15'
labels: []
dependencies:
  - TASK-016
references:
  - .skillgrid/specs/2026-09-17-compact-search-output/tasks.md
  - .skillgrid/specs/2026-09-17-compact-search-output/briefing.md
  - .skillgrid/specs/2026-09-17-compact-search-output/acceptance.feature
  - skillgrid-cli/internal/mnemonic/hybrid/
  - skillgrid-cli/internal/mnemonic/mcp/tools_code_hybrid.go
  - skillgrid-cli/internal/mnemonic/mcp/tools_code_search.go
documentation:
  - .skillgrid/specs/2026-09-17-compact-search-output/blueprint.md
  - .skillgrid/specs/2026-09-17-sqlite-ai-code-indexing/findings.md
priority: medium
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track the SDD spec **2026-09-17-compact-search-output**: token-efficient search output for the agent — a `context` boolean + `unfold` parameter on `code_hybrid_search` / `code_search` that returns name + file + lines + signature without source bodies (target ≤250 tokens for 10 hits vs ~2,000+ with bodies).

**Current State:**
- STATUS `stalled` (2026-09-21): `FormatCompact` absent from the hybrid layer; no `context`/`unfold` params on the MCP tools (verified in code)
- Pattern validated by 5 production tools (mnemo `--context`, context-mode, headroom, CTX, mcp-injector) — see sqlite-ai-code-indexing findings §7
- Sliced into 2 tickets: TICKET-01 `FormatCompact` (hybrid/rank.go), TICKET-02 param wiring (tools_code_hybrid.go + tools_code_search.go)

**Expected State:**
- Both tickets implemented acceptance-first (RED scenario before impl); single PR.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 `FormatCompact` produces `name (file:line_start-line_end) — signature` for symbol hits and `file:line_start-line_end — first 80 chars` for chunk hits, no source bodies
- [ ] #2 `FormatCompact` is deterministic (byte-identical across calls)
- [ ] #3 Compact output for 10 hits is ≤250 tokens (len/4 heuristic)
- [ ] #4 `code_hybrid_search` + `code_search` accept `context` (bool) and `unfold` (paths/globs); `context: true` returns compact, omitted/false returns the unchanged full format
- [ ] #5 `unfold` with exact paths and globs replaces matched hits with full source while others stay compact
<!-- AC:END -->

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
created: 2026-09-21 07:15
---
2026-09-21: spec STATUS is `stalled` — work paused, not dependency-blocked. Re-triage to ready-for-agent when picking it back up. Depends on TASK-016 (done: index-status plumbing) per the slicing notes.
---
<!-- COMMENTS:END -->
