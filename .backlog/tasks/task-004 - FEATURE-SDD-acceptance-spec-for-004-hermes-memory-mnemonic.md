---
id: TASK-004
title: '[FEATURE] SDD acceptance spec for 004-hermes-memory (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
updated_date: '2026-09-05'
labels: []
dependencies:
  - TASK-001
priority: medium
type: feature
references:
  - docs/skillgrid/changes/004-hermes-memory/acceptance.feature
  - docs/skillgrid/changes/004-hermes-memory/change.md
  - docs/skillgrid/changes/004-hermes-memory/tasks.md
  - docs/skillgrid/changes/004-hermes-memory/steps/
documentation:
  - docs/skillgrid/changes/004-hermes-memory/change.md
  - .agents/skills/_shared/conventions/sdd-structure.md
---

## Description

Track the SDD **acceptance.feature** specs for change **004-hermes-memory** (Hermes Fact Memory & Agent Skills, steps 01–05). Tasks punch-lists exist; implementation waits on human choice between `sdd-apply` and `sdd-propose`.

**Current State:**
- Plan tracked by TASK-001; step folders and `tasks.md` exist under `docs/skillgrid/changes/004-hermes-memory/steps/`
- Per-step `acceptance.feature` written (spec phase); no production Fact Memory / Agent Skill code yet

**Expected State:**
- Steps 01–05 task checkboxes completed during `sdd-apply` against these acceptance scenarios
- Threat RED scenarios covered: path escape / unknown language (04); Mnemonic tool surface (02, 03, 04)
- `go test ./...` passes for touched packages

## Acceptance Criteria

<!-- AC:BEGIN -->
- [ ] #1 Acceptance features exist for steps 01–05 (filesystem + Engram `sdd/004-hermes-memory/spec`)
- [ ] #2 Each step has @happy / @edge / @failure; threat-matrix scenarios in 02, 03, 04
- [ ] #3 Apply marks tasks `[x]` in dependency order (01 → 02|03 → 04 → 05)
- [ ] #4 Soft-after 003 `010_*`; `go test ./...` passes for touched packages
<!-- AC:END -->

## Definition of Done

<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
- [ ] #6 Every `@step-NN` Feature has passing @happy / @edge / @failure
- [ ] #7 Threat-matrix RED coverage for steps 02, 03, 04 passed
<!-- DOD:END -->

## Implementation Plan

<!-- SECTION:PLAN:BEGIN -->
1. Gate on TASK-001 plan readiness and human Implement vs Revise decision.
2. On Implement: run `sdd-apply` against step acceptance features in order 01 → 02|03 → 04 → 05.
3. Keep per-step `acceptance.feature` and Engram `sdd/004-hermes-memory/spec` aligned as scenarios pass.
4. Prioritize threat RED: path escape / unknown language (04); Mnemonic tool surface (02, 03, 04).
5. Close when all AC scenarios + change DoD are green; archive with TASK-001.
<!-- SECTION:PLAN:END -->

## Technical Notes

- Steps: `01-facts-schema`, `02-fact-tools`, `03-skills-registry`, `04-skill-execute-hybrid`, `05-commit-hooks-cli`
- Paths: `docs/skillgrid/changes/004-hermes-memory/steps/*/acceptance.feature`
- Spec: Engram `sdd/004-hermes-memory/spec`; carry-through plan ticket TASK-001
- Open: sqlite-vec on modernc (extension vs CGO) decided in step 01

## Priority

Medium — builds on 003; product urgency after 003 lands.

## Comments

<!-- Conversation appends here. -->

- 2026-09-04: Created for `force_ticket_creation` on sdd-spec (acceptance/tasks artifact). `backlog` CLI SIGILL/SIGTRAP on this host (Bun arch mismatch); ticket written as filesystem task file matching TASK-001 frontmatter. Duplicate-search: TASK-001 covers plan only; no prior acceptance-spec ticket for 004.
- 2026-09-05: Filled missing `type`, `references`, Definition of Done, and Implementation Plan via filesystem fallback (CLI still SIGILL on edit).
