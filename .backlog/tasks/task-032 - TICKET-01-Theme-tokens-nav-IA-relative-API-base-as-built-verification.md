---
id: TASK-032
title: 'TICKET-01: Theme tokens, nav IA, relative API base (as-built verification)'
status: needs-triage
assignee: []
created_date: '2026-10-02 13:12'
labels:
  - >-
    webui-rewrite --priority high --type feature -p TASK-031 --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/acceptance.feature
    --plain
dependencies: []
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Confirm the shell matches the mockup: indigo/surface tokens, six nav groups in order, demoted routes reachable, apiOrigin() relative in production. SATISFIES nav-matches-mockup, theme-tokens, relative-api-urls.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 G1 AppLayout.test.tsx PASS
- [ ] #2 G2 route grep = 5
- [ ] #3 G3 indigo/surface tokens present
- [ ] #4 G8 no fetch() site hardcodes 127.0.0.1:7438 outside apiBase.ts
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
