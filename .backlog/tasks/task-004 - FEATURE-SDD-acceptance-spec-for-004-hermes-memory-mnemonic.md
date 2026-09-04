---
id: TASK-004
title: '[FEATURE] SDD acceptance spec for 004-hermes-memory (mnemonic)'
status: ready-for-agent
assignee: []
created_date: '2026-09-04'
labels: []
dependencies:
  - TASK-001
priority: medium
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
