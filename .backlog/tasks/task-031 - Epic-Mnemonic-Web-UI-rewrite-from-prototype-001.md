---
id: TASK-031
title: 'Epic: Mnemonic Web UI rewrite from prototype 001'
status: needs-triage
assignee: []
created_date: '2026-10-02 13:11'
updated_date: '2026-10-06 08:35'
labels:
  - webui-rewrite
milestone: m-2
dependencies: []
references:
  - .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/briefing.md
  - .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/blueprint.md
  - .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md
  - .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/acceptance.feature
  - .skillgrid/artifacts/04-adr-0017-d3-force-graph.md
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Bring skillgrid-ui to prototype 001 IA, indigo tokens, D3 force graph (ADR-0017); panels on live Go endpoints. As-built on release/2 (aa0aa791); this epic verifies it and closes gaps. Tickets: .skillgrid/specs/2026-10-02-mnemonic-webui-rewrite/tasks.md
<!-- SECTION:DESCRIPTION:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
