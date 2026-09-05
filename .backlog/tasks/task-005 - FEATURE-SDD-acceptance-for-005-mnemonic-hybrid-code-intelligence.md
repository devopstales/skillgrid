---
id: TASK-005
title: '[FEATURE] SDD acceptance for 005-mnemonic-hybrid-code-intelligence (mnemonic)'
status: needs-triage
assignee: []
created_date: '2026-09-04 14:23'
updated_date: '2026-09-05'
labels: []
dependencies: []
priority: medium
type: feature
references:
  - docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md
  - docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/tasks.md
  - docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/acceptance.feature
  - docs/plan/07-nmemonic-hybid-search.md
  - skillgrid-cli/internal/mnemonic/
documentation:
  - docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md
  - .agents/skills/_shared/conventions/sdd-structure.md
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Track SDD **acceptance** for change **005-mnemonic-hybrid-code-intelligence** (hybrid/graph code-intelligence foundation slice, steps 01–04). Turns chunk-FTS into Symbols/Edges, Identifier-Aware FTS, Tier-1/2 tools, and offline RRF hybrid search.

**Current State:**
- Intent/plan folded into `docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md`
- Acceptance/tasks artifacts exist; no production hybrid extractor / graph / hybrid-search tools yet

**Expected State:**
- Steps 01–04 implemented against acceptance scenarios (schema+extractor → identifier FTS/orientation → graph tools → hybrid search)
- Existing `code_status` / `code_index` / `code_search` / `code_read` contracts unchanged
- `go test ./...` passes for touched packages
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 Acceptance features exist for steps 01–04 (filesystem + Engram `sdd/005-mnemonic-hybrid-code-intelligence/spec`)
- [ ] #2 Each step has @happy / @edge / @failure; foundation-slice scenarios cover Go/TS/TSX
- [ ] #3 Existing `code_search` name + required `query` schema unchanged
- [ ] #4 Hybrid search works with embeddings off; malformed file does not fail index run
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Change-level Definition of Done in `change.md` fully checked
- [ ] #7 Existing chunk `code_*` contracts remain name- and signature-stable
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Triage to `ready-for-agent` once human confirms Implement vs Revise for 005.
2. Follow `docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md` Step Blueprint (01–04): schema/extractor → identifier FTS/orientation → graph tools → offline hybrid RRF.
3. Drive from `tasks.md` + `acceptance.feature`; keep memory `semantic_search` distinct from hybrid code search.
4. Languages for this slice: Go / TypeScript / TSX only; embeddings off by default (Null Adapter).
5. Verify with `go test ./...` on touched packages; mark AC/DoD and archive when change DoD is green.
<!-- SECTION:PLAN:END -->

## Technical Notes

- Affected paths: Mnemonic codeindex / hybrid modules under `skillgrid-cli/internal/mnemonic/`
- Change: `docs/skillgrid/changes/005-mnemonic-hybrid-code-intelligence/change.md`
- Source proposal: `docs/plan/07-nmemonic-hybid-search.md`
- Soft preference: after 002 identity; orthogonal to 003/004

## Priority

Medium — foundation slice for agent code navigation; orthogonal to 004/006.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created thin acceptance stub for 005.
- 2026-09-05: Enriched description/AC and filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (`backlog` CLI SIGILL on this host).
