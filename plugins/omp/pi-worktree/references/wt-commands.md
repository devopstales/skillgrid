# wt commands this plugin uses

One-page pointer. The compact live catalog is `wt --help` (the `worktrunk`
tool accepts `--help` in `args` and help calls return immediately); this doc
lists only the commands the skillgrid invariants touch and their default
rewrites. Hand-written — not generated, so it cannot drift silently from the
installed binary.

**Minimum version: `wt` ≥ 0.50.0** (verified against `wt 0.50.0`, 2026-11-22).
The plugin's runner depends on Worktrunk's `WORKTRUNK_DIRECTIVE_CD_FILE`
directive protocol, which is Worktrunk-internal (not documented public API).
`wt-core.ts` probes protocol support at startup; see `README.md` →
git-fallback behavior for what happens when `wt` is absent.

## Commands

### `wt switch --create <branch>`

Create a worktree for `<branch>` and switch to it. The runner:

- enforces the branch allow-list in a **worktree context**: the branch must
  match `^(agent-|worktree-agent-|worktree-wf_)[A-Za-z0-9._/-]+$`
  (`agent-*`, `worktree-agent-*`, `worktree-wf_*`); a mismatch is rejected
  with the nearest allowed form suggested. The **primary checkout is
  unconstrained**.
- observes the actual worktree root on first use; a root outside
  `<projectroot>/.worktree/` adds a one-line "observed root differs — noted"
  to the report (never a silent follow).
- follows Worktrunk's directory directive: the pi session moves into a linked
  session in the new worktree (source session stays `/resume`-able); in
  print/json mode the result is `stopped` + the restart directory.

### `wt merge <target>`

Merge the current branch into `<target>` locally (Worktrunk never fetches).
The runner rewrites the command — the one place it mutates a call:

- `--no-squash --no-commit` are prepended unless the caller explicitly passed
  either flag (an explicit `--squash` is preserved and not rewritten).
  Worktrunk's default pipeline squashes; the skillgrid standard keeps
  work-unit commits atomic and independently revertable.

### `wt config state marker set waiting|working --cwd <cwd>` / `wt config state marker clear`

The branch status markers (`wt list` shows 💬 waiting / 🤖 working).
`src/markers.ts` is the **sole writer** of this marker state while this plugin
is installed:

- `session_start` → `set waiting` (💬)
- `agent_start` → `set working` (🤖)
- `agent_end` → `set waiting` (💬)
- `session_shutdown` → `clear`

A session killed before shutdown can leave a stale marker; clear it manually
with `wt config state marker clear`.

### `wt config approvals add`

Approve project commands. When a `wt` call needs project-command approval, the
runner lists the unapproved commands in a dialog; approving runs
`wt config approvals add --yes` to save **exactly those commands**, then
retries the original command once. The model can never pass `-y`/`--yes`
itself — the runner rejects it. Non-interactive mode: the command fails and
lists what to approve.

### `wt list`

List worktrees (and their branch markers). Used for inspection; markers
written per the lifecycle above show 💬/🤖 for live pi sessions.
