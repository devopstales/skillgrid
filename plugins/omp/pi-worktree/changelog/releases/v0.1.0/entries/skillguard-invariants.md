---
title: Skillgrid invariants in the worktrunk wrapper
type: feature
author: devopstales
created: 2026-11-22T00:00:00.000Z
---

The runner enforces the skillgrid worktree standard instead of passing
Worktrunk calls through unchanged:

- In a worktree context, `switch --create` with a branch outside
  `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$` is rejected with
  the nearest allowed form suggested; the primary checkout is unconstrained.
- `merge` calls execute with `--no-squash --no-commit` prepended unless the
  caller explicitly passed either flag — the one place the plugin mutates the
  command.
- The first `wt` use observes the actual worktree root; when worktrees land
  outside `<projectroot>/.worktree/` the report carries a one-line "observed
  root differs — noted" (the path template is an owner-set Worktrunk config
  value, outside the plugin's trust boundary).

*By @devopstales.*
