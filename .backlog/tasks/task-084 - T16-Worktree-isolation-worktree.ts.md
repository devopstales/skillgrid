---
id: TASK-084
title: T16 Worktree isolation (worktree.ts)
status: needs-triage
assignee: []
created_date: '2026-10-09 13:13'
labels:
  - skill-pi
  - swarm
dependencies:
  - TASK-082
references:
  - .skillgrid/specs/2026-10-09-skill-pi/tasks.md
  - .skillgrid/specs/2026-10-09-skill-pi/acceptance.feature
modified_files:
  - skill-pi/extensions/worktree.ts
  - skill-pi/extensions/skillgrid-squad.test.ts
  - skill-pi/extensions/skillgrid-squad.ts
priority: high
type: feature
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Implement createWorktree(repoRoot, taskID), commitWorktree(path, branch, taskID), removeWorktree(path, branch). Branch = worktree-agent-swarm-<taskID> (matches precommit-guard regex). Reject task_id not matching [A-Za-z0-9-]+ before any git call. Wire into skillgrid-squad.ts spawn path. Tracer B, Wave 11.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 In a temp git repo: createWorktree produces a path under .worktrees/ and a branch matching ^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$
- [ ] #2 A task_id with a space is rejected (no git exec)
- [ ] #3 commitWorktree on a repo with changes produces a commit on the branch
- [ ] #4 A git failure (non-zero exit) leaves the worktree path on disk and returns stderr
<!-- AC:END -->

## Definition of Done
<!-- DOD:BEGIN -->
- [ ] #1 Tests pass (`go test ./...` for touched packages)
- [ ] #2 Lint and formatting pass
- [ ] #3 Edge cases covered
- [ ] #4 No new warnings introduced
- [ ] #5 Spec/docs updated if behavior changes
<!-- DOD:END -->
