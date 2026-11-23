---
type: Gotcha
title: "wt's piped (no-TTY) runtime silently skips unapproved hooks; approvals must be stored under a PTY"
status: verified
sources:
  - ".skillgrid/change-archive/2026-11-22-pi-worktree-plugin/report.md (QA finding E / R10)"
---

# wt's piped (no-TTY) runtime silently skips unapproved hooks

A Pi agent plugin spawns `wt` with piped stdio (no TTY). Two wt 0.50.0 behaviors
bite that runtime:

1. **A hook-triggering command silently skips an unapproved hook.** `wt switch`
   (with a `post-switch` hook) in a fresh repo exits **0 with no marker** when the
   hook is unapproved — it does not error and does not print an approval prompt.
   So "did the command need approval?" cannot be detected from the command's own
   exit code / output. Detect it separately by running `wt hook <phase>` (the
   command's hook phase, e.g. `post-switch`), which exits 1 with
   `needs approval to execute N command` + `Cannot prompt for approval in
   non-interactive environment` when an approval is pending.
2. **`wt config approvals add --yes` does not confirm from a pipe.** The `--yes`
   flag does **not** auto-answer the interactive `Allow and remember? [y/N]`
   prompt when stdin is a pipe (exit 1). The store succeeds only under a PTY:
   `script -qec "wt config approvals add --yes" /dev/null` with the answer
   (`y\n`) fed to the PTY's stdin.

## The check

- An agent runtime that drives `wt` must treat "exit 0 but a pending hook
  approval" as a distinct state: probe `wt hook <phase>` after a hook-capable
  command succeeds, and route a pending approval through the UI, then store it
  under a PTY (`script -qec … /dev/null`), then re-run the command.
- Never rely on the hook command's own exit code to detect a pending approval
  (it is 0), and never rely on `--yes` to confirm from a pipe (it does not).
- `wt switch --create <existing>` re-runs idempotently (exit 0), so re-running
  the original command after storing the approval is safe.
