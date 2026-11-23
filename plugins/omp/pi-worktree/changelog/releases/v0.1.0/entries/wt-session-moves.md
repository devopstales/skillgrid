---
title: Worktrunk runner with linked-session movement
type: feature
author: devopstales
created: 2026-11-22T00:00:00.000Z
---

The `worktrunk` tool (and the `/wt` command) runs `wt` with the
`WORKTRUNK_DIRECTIVE_CD_FILE` directive protocol: when `wt` requests a
directory change, pi moves the session into a linked session in the destination
worktree, and the source session stays available via `/resume`. In print/json
mode a directory request stops continuation and reports where to restart.

When the session's working directory is gone (removed by a `wt` command or
another process), the next `wt` call recovers the session to a surviving
worktree of the same repository — repository identity re-verified, the repo's
default branch preferred — and does NOT run a command requested after the
directory died.

Project commands that need approval are offered in a dialog; approving saves
exactly those commands (as `wt config approvals add`) and retries once. The
model can never pass `-y`/`--yes`.

*By @devopstales.*
