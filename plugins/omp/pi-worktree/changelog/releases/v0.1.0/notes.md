This release adds turn-level scratch checkpoints of the worktree (`refs/skillgrid/`, safe decision-gated restore) and a Worktrunk runner with linked-session movement, no-replay recovery, skillgrid branch allow-listing, and `--no-squash --no-commit` merge defaults, so pi sessions follow the skillgrid worktree standard instead of restating it.

## 🚀 Features

### Turn-level scratch checkpoints of the worktree

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

### Worktrunk runner with linked-session movement

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

### Worktrunk status markers for pi sessions

The plugin updates Worktrunk branch markers so `wt list` shows whether pi is
actively working or waiting for input: session start sets waiting (💬), agent
start sets working (🤖), agent end sets waiting (💬) again, and session
shutdown clears the marker.

If a session is killed before shutdown, the marker can remain stale; the plugin
reports it and it is removed manually with `wt config state marker clear`.

This plugin is the sole writer of Worktrunk marker state while installed. If
Worktrunk's own pi plugin is also installed, both write the same markers — the
README notes the conflict (no runtime detection).

### Recovery for removed worktrees without replay

Recovery from a removed working directory moves the session to a surviving
worktree in the same repository — only worktrees recorded before the removal
are eligible, and the repository identity is re-verified. The repo's default
branch (`origin/HEAD` when set, falling back to `main` then `master`) is
preferred. A command requested after the directory became unusable is NOT run;
the recovery report says so, and the command is re-evaluated in the new
worktree before retrying.

### Skillgrid invariants in the worktrunk wrapper

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
