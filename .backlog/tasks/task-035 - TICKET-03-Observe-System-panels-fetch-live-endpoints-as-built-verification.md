---
id: TASK-035
title: >-
  TICKET-03: Observe + System panels fetch live endpoints (as-built
  verification)
status: done
assignee: []
created_date: '2026-10-02 13:13'
updated_date: '2026-10-02 13:33'
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
Confirm Compaction, Web Cache, Telemetry, Settings, Swagger pages exist and their fetch paths are registered on the Go mux (G6 = 4). SATISFIES panels-fetch-live-endpoints (Observe).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 G6 = 5 registered routes in server.go
- [x] #2 settings + swagger vitest PASS
- [x] #3 SwaggerPage iframe src is apiUrl('/swagger/')
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Verified: G6=4, settings+swagger 9 passed. Trivy route gap moved to TASK-036.
<!-- SECTION:NOTES:END -->
