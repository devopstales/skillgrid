---
id: TASK-034
title: 'TICKET-02: D3 force code graph with inspector (ADR-0017) — door check'
status: done
assignee: []
created_date: '2026-10-02 13:13'
updated_date: '2026-10-06 08:35'
labels:
  - >-
    webui-rewrite --priority high --type feature -p TASK-031 --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md --ref
    .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/acceptance.feature
    --plain
milestone: m-2
dependencies:
  - TASK-032
references:
  - .skillgrid/artifacts/04-adr-0017-d3-force-graph.md
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Confirm GraphPage is the mockup D3 force layout (limit=500, Node Inspector) and the Sigma stack is gone; bundle under budget with d3 split out. SATISFIES d3-code-graph.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 G4 three matching lines in GraphPage.tsx
- [x] #2 G5 0 Sigma deps / 1 d3
- [x] #3 npm run build:check prints Bundle-size budget OK with vendor-d3 chunk
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
Verified at d5e01985: G4 3 lines, G5 0/1, build:check 298.2 kB/400 kB with vendor-d3 chunk. Door check held.
<!-- SECTION:NOTES:END -->
