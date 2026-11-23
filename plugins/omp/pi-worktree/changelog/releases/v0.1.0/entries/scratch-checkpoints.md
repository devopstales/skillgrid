---
title: Turn-level scratch checkpoints of the worktree
type: feature
author: devopstales
created: 2026-11-22T00:00:00.000Z
---

Each `turn_end` captures the worktree as a git ref under `refs/skillgrid/` —
HEAD, index tree, and worktree tree — without touching the real index and
without creating a commit. A turn that left the worktree tree unchanged creates
no new ref (dedup on the worktree tree SHA).

The capture skips the standard noise set (13 directory patterns including
`.worktree`, untracked files over 10MiB, untracked directories with more than
200 files) and records what it skips. Restores are always a decision — the
`scratch-restore` tool or `/checkpoint` command shows a one-line diff preview
and a `files-only` / `cancel` dialog; the restore takes a before-restore
checkpoint first, and is refused when the current branch differs from the
branch recorded at capture.

Refs are bounded: 50 max per session (oldest pruned), and at session start the
refs of other sessions older than 1 hour are pruned in the background.

*By @devopstales.*
