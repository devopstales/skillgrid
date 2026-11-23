---
title: Recovery for removed worktrees without replay
type: feature
author: devopstales
created: 2026-11-22T00:00:00.000Z
---

Recovery from a removed working directory moves the session to a surviving
worktree in the same repository — only worktrees recorded before the removal
are eligible, and the repository identity is re-verified. The repo's default
branch (`origin/HEAD` when set, falling back to `main` then `master`) is
preferred. A command requested after the directory became unusable is NOT run;
the recovery report says so, and the command is re-evaluated in the new
worktree before retrying.

*By @devopstales.*
