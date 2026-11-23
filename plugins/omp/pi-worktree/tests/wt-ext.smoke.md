# wt extension smoke (`src/wt.ts`)

`pi` 1.4.0 is installed at `~/.local/bin/pi` (checked 2026-11-22), so the
steps below can be run manually. wt 0.50.0 is on PATH.

Setup (once):

```sh
export PATH="$HOME/.bun/bin:$PATH"
cd <this repo>
pi install .          # or: pi -e ./src/wt.ts
```

## 1. Session follows a worktree (TUI)

1. `pi -e ./src/wt.ts` in a temp repo with wt initialized and a first commit.
2. Ask the agent (or call the tool) to run
   `wt switch --create agent-smoke`.
3. Expected: the session's cwd becomes the new worktree
   (e.g. `<main>.agent-smoke`); the tool result says
   `Moved the session to <path>`; the source session remains available via
   `/resume`.

## 2. Print mode stops instead of moving

1. `pi -e ./src/wt.ts --print "run wt switch --create agent-print"`.
2. Expected: the tool result reports
   `stopped` + the restart path; the process exits without the session
   having moved (Pi must be restarted in the reported directory).

## 3. Unapproved project command → approval dialog

1. In the repo, create a project hook:
   `wt config create --project`, then set
   `pre-start = "echo PRE-START-RAN > /tmp/pre-start-ran.txt"` in
   `.config/wt.toml`.
2. From TUI, run `wt switch --create agent-approve`.
3. Expected: wt stops with `needs approval`; Pi shows a confirmation dialog
   listing the unapproved command. Approve → `wt config approvals add --yes`
   runs for exactly that command and the wt call is retried once; the hook
   runs (`/tmp/pre-start-ran.txt` exists) and the session moves.
4. Decline instead → the command is not run and the pending command is
   listed. In print mode (no dialog), the tool fails and lists the commands
   to approve in a terminal with `wt config approvals add`.
5. `wt config show` (or `~/.config/worktrunk/approvals.toml`) shows the
   saved approval.

## 4. No-replay recovery

1. From TUI in a worktree (e.g. the one from step 1), delete that worktree
   directory from another shell (simulating "another process" removing it).
2. Run any wt command via the tool, e.g. `wt list`.
3. Expected: the tool result says the session's cwd no longer exists, that
   it recovered to a surviving worktree of the same repository (preferring
   the default branch), and that the requested command was NOT run
   (no-replay — re-evaluate before retrying). A linked "Session recovered"
   session is created; the source session stays `/resume`-able.
4. No surviving recorded worktree → the result names the original path to
   restart Pi in; no move, no replay.

## 5. `-y` is never accepted

`wt list -y` via the tool → error: the tool does not accept `-y`/`--yes`.
