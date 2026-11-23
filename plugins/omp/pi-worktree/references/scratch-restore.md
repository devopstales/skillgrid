# Scratch checkpoint restore protocol

`scratch-restore {id}` (tool) or `/checkpoint <id>` (command) restores the
worktree, git index, and HEAD to a captured checkpoint. Restores are always a
decision: in TUI/RPC mode a dialog shows a one-line diff preview and a
`files-only` / `cancel` choice (print/json modes with an explicit id proceed
files-only).

## The branch-keyed check

Before any restore, `restoreCheckpoint` reads the current branch
(`git rev-parse --abbrev-ref HEAD` in the repo) and compares it with the
`branch` recorded in the checkpoint's metadata. They must be **identical**.

Why branch-only: a branch name exists in exactly one live worktree, so a
matching branch implies the recorded worktree. The original design's
worktree-root half of the check was retired (Goldfish Stage 5 R1-F001) — git
makes a root mismatch with a matching branch unreachable.

A detached capture records the literal branch `DETACHED`; a restore from a
detached HEAD (current branch `HEAD`) never matches, so a detached capture is
never a safe restore target.

## Refusal reasons

| reason | detail format | cause |
|---|---|---|
| `branch` | `checkpoint on '<recorded>', current '<actual>'` | current branch ≠ recorded branch (including detached HEAD) |
| `missing` | `no checkpoint <id>` | the id is not a safe ref id (`^[\w-]+$`), or the ref is absent or its metadata block is unparseable |

On refusal nothing is written: no before-restore checkpoint, no reset, no
clean.

## Before-restore checkpoint

On a matching branch, the restore **first** takes a scratch checkpoint labeled
`before-restore <id>` (best-effort — a failure there does not block the
restore). So a botched restore can itself be restored: the pre-restore state is
a normal `refs/skillgrid/` ref of the same session.

## Restore mechanics

1. `git reset --hard <headSha>` (skipped when the capture was on an unborn
   HEAD — zero SHA)
2. `git read-tree --reset -u <worktreeTreeSha>` — worktree + index to the
   captured worktree tree
3. `safeClean` (below)
4. `git read-tree --reset <indexTreeSha>` — the **real index** restored
   exactly as captured (staged/unstaged split preserved)

## safeClean — what is deleted vs preserved

`safeClean` removes files that must not survive a restore, in two sets:

1. files present in the captured tree but absent in the restored HEAD
   (`git diff --diff-filter=AD HEAD <captured tree>`)
2. untracked files that appeared **after** the capture and are not in the
   snapshot

**Never deleted:**

- files under ignored directories (the 13-dir filter: `node_modules`, `.venv`,
  `venv`, `env`, `.env`, `dist`, `build`, `.pytest_cache`, `.mypy_cache`,
  `.cache`, `.tox`, `__pycache__`, `.worktree`)
- untracked files that already existed when the checkpoint was captured
  (the `preexisting-untracked` metadata list)
- untracked files skipped at capture for size (the `largeFiles` metadata list:
  >10MiB files, >200-file directories)
- anything under a skipped large directory (the `largeDirs` metadata list)

The net effect: a deleted tracked file comes back, a post-checkpoint untracked
file is removed, and everything the capture intentionally ignored survives.

## Bounds

- 50 checkpoints max per session — the 51st capture prunes the oldest
  (`prune`, `MAX_PER_SESSION`).
- At session start, checkpoints of **other** sessions older than 1 hour
  (`PRUNE_MIN_AGE_MS`) are pruned in the background; a live sibling session's
  fresh refs survive.
