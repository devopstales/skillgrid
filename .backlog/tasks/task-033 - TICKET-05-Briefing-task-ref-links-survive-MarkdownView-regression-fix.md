---
id: TASK-033
title: 'TICKET-05: Briefing task-ref links survive MarkdownView (regression fix)'
status: needs-triage
assignee: []
created_date: '2026-10-02 13:13'
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
aa0aa791 moved plan briefings into MarkdownView, whose sanitiser strips the raw <a> from linkifyTaskRefs; emit markdown links instead via linkifyTaskRefsMarkdown. SATISFIES briefing-task-refs-linkified.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 RED at HEAD: PlansPage.linkedTasks.test.tsx misses /tracker?task=012
- [ ] #2 after fix: npx vitest run src/features/plans → 11 passed
- [ ] #3 taskLinks.test.ts covers linkifyTaskRefsMarkdown
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
