---
id: TASK-107
title: 'TICKET-01: Migration 072 + Global Store Bootstrap'
status: ready-for-agent
assignee: []
created_date: '2026-10-24 13:18'
updated_date: '2026-10-24 13:19'
labels:
  - mnemonic
  - global-memory
dependencies: []
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
**Scope:** Create `facts.category` column (migration 072), `Service.OpenGlobal()`, `__global__` skip in `ListProjects`/`RunTTLExpiry`. The door check: can `store.Open(dataDir, "__global__")` create `__global__.sqlite` and can `ListProjects` skip it?

**Acceptance:** `__global__.sqlite` created in data dir; `category` column exists with default `'memory'`; `ListProjects` excludes `__global__`; `RunTTLExpiry` skips `__global__`; file name is exactly `__global__.sqlite` (not mangled by `NormalizeID`).

**SATISFIES:** `happy path — global store is created and opened`, `edge — global store is excluded from project listing`, `edge — global store is excluded from TTL expiry`, `failure — global store file name is not mangled by normalization`

**Files:** `mnemonic/internal/store/migrations/072_facts_category.sql` (create), `mnemonic/internal/service/global.go` (create), `mnemonic/internal/service/global_test.go` (create), `mnemonic/internal/service/service.go` (modify: ListProjects + RunTTLExpiry skip)

**Size:** ~300 (S)
**One-way door:** Yes (migration 072 adds NOT NULL DEFAULT column)
**Fails-when:** `go test` exits non-zero, or `__global__.sqlite` file not found in data dir after `OpenGlobal`
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 __global__.sqlite created in data dir
- [ ] #2 category column exists with default 'memory'
- [ ] #3 ListProjects excludes __global__
- [ ] #4 RunTTLExpiry skips __global__
- [ ] #5 file name is exactly __global__.sqlite (not mangled by NormalizeID)
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
