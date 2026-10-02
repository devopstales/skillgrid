---
id: TASK-036
title: 'TICKET-04: Security + Prototypes panels on registered endpoints'
status: needs-triage
assignee: []
created_date: '2026-10-02 13:13'
updated_date: '2026-10-02 13:22'
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
System → Security must fetch GET /security/trivy and Project → Prototypes must fetch a registered JSON path listing .skillgrid/prototypes/NNN-name/. At HEAD both pages exist but neither endpoint is registered (panels render ErrorState). BLOCKED: both halves exist uncommitted in the main working tree (docs.NewTrivy shells out to the trivy CLI — needs its own review; docs.NewPrototypes is part of the spikes→prototypes rename) in a server.go that also carries unrelated routes. Precondition: rename committed and server.go clean. SATISFIES panels-fetch-live-endpoints (Security and Prototypes).
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
