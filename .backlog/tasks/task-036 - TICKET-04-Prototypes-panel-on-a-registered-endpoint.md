---
id: TASK-036
title: 'TICKET-04: Prototypes panel on a registered endpoint'
status: needs-triage
assignee: []
created_date: '2026-10-02 13:13'
labels:
  - >-
    webui-rewrite --priority high --type feature -p TASK-031 --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/acceptance.feature
    --plain
dependencies:
  - TASK-032
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Project → Prototypes must fetch a path the Go mux registers as JSON and list .skillgrid/prototypes/NNN-name/ entries. BLOCKED: depends on the uncommitted repo-wide spikes→prototypes rename in the main working tree (staged renames + docs_prototypes.go + PrototypesPage.tsx). Precondition: git status --short | rg -c '^R  .skillgrid/spikes/' prints 0. SATISFIES panels-fetch-live-endpoints (Prototypes).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 G7 = 1 (fetch path registered in server.go)
- [ ] #2 full npm test PASS
- [ ] #3 nav label Prototypes at /project/prototypes
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
