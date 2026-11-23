# scratch extension smoke (`src/scratch.ts`)

`pi` 1.4.0 is installed (checked 2026-11-22), so the steps below can be run
manually. The vllm model endpoint in `~/.pi/agent/models.json` is used.

Setup (once, per smoke run):

```sh
export PATH="$HOME/.bun/bin:$PATH"
cd <this repo>
SMOKE=$(mktemp -d /tmp/pi-smoke-XXXX)
cd $SMOKE
git init -q -b main
git config core.hooksPath /dev/null
echo "one" > a.txt
git add a.txt
git -c user.name=paladm -c user.email=paladm@localhost commit -qm "chore: init"
```

Key invocation fact (verified): `pi` runs with cwd = the directory it is
invoked from, and the extension captures at `ctx.cwd` (the repo root via
`repoRoot`), so run pi **from the temp repo**. Print mode gates tools
behind an approval dialog that cannot be shown; with `-ne` (no other
extensions) and `--no-mcp` the built-in `bash` approval is satisfied and the
extension tool runs unblocked. The commands below use that form:

```sh
pi --print --provider vllm --model qwen3.8-uncensored \
  -e <this repo>/src/scratch.ts -ne --no-mcp \
  --session-dir $SMOKE/.sessions -c "<prompt>"
```

`--session-dir` is a fresh dir, so each `pi` call below continues one
persistent Pi session (session file + session id carry across calls; the
checkpoints' `session` metadata must match across all steps).

## 1. Mutating turn → checkpoint + footer

1. From `$SMOKE`:
   `pi --print … -c "Use the bash tool to run: echo two >> a.txt"`
2. Expected: the agent appends the line; `git for-each-ref refs/skillgrid/`
   in `$SMOKE` shows exactly **1** new ref
   `refs/skillgrid/<session-id>-t1-<epoch>`.
3. In TUI mode (`pi -e <this repo>/src/scratch.ts` from `$SMOKE`, no
   `--print`): the footer shows `◆ 1 checkpoint` (singular at 1) with a
   brief `✓ checkpoint —` flash right after the capture.

**Verified (print mode, 2026-11-22):** steps 1+2 observed — 1 ref created,
metadata block complete (`branch main`, `head`, `index-tree`, `worktree-tree`,
`session <id>`, `turn 1`, `label` = first 60 chars of the prompt, `created`,
`preexisting-untracked` listing). The footer string is verified by the
`getCache`-driven rendering path; the TUI component itself is manual (TUI
only).

## 2. Second mutating turn → `◆ 2 checkpoints`

1. Same session dir, second call:
   `… -c "Use the bash tool to run: echo new > after.txt"`
2. Expected: a **second** ref `-t2-<epoch>`; TUI footer `◆ 2 checkpoints`.

**Verified:** 2nd ref created (turn 2, `worktree-tree` differs from turn 1's).

## 3. Read-only turn → dedup, no new ref

1. `… -c "Read the file a.txt and tell me its contents. Do not modify anything."`
2. Expected: the tool runs, no new `refs/skillgrid/` ref for this turn
   (dedup on worktree tree SHA).

**Verified with caveat:** the read-only turn itself did not add a ref, but
the resumed session's first `turn_end` (before the read) re-captured the
unchanged tree as a new ref — `lastTree` is in-memory per process and the
blueprint's "seed lastTree from the newest ref" (Stage 9 F1) is not wired in
core (src/core/ is frozen for this task). Core dedup within a process is
unit-tested (`tests/scratch.test.ts` "read-only turn is skipped"); the
cross-process first-turn re-capture is a known limitation recorded here.

## 4. /checkpoint lists with label

1. TUI: type `/checkpoint` (or `scratch-restore` with no args via the agent).
2. Expected: the list shows, per checkpoint: `[turn N] <created>  <label>`
   + `    id: <checkpoint id>` — the label is the captured prompt prefix.

**Verified (via the tool's list path in print mode):** the agent reported
the checkpoint list with turn, created, label (60-char prompt prefix) and id.

## 5. Restore after the second edit → file back, dialog confirmed

1. Delete/modify state past the target checkpoint, e.g. from a new shell:
   `rm $SMOKE/after.txt` is NOT the test — instead mutate again:
   `… -c "Use the bash tool to run: echo three >> a.txt"`.
2. Then, same session:
   `… -c "Call the scratch-restore tool with {\"id\": \"<turn-2 id from step 2>\"}"`.
3. Expected: in TUI/RPC a `ctx.ui.select` dialog shows
   `Restore checkpoint <id>?` + turn/created/branch + label + the one-line
   diff preview (`N file(s) vs current: <first 5 paths>`); choosing
   `files-only` → the worktree matches the turn-2 tree exactly:
   `a.txt` is back to `one`, `after.txt` removed (it was created after the
   target checkpoint), pre-existing untracked files survive, and a
   **before-restore** checkpoint ref exists (label `before-restore <id>`).
   Tool result text:
   `Restored to <id> (turn 2, <label>). A before-restore checkpoint was
   taken first.`
4. In print mode (no dialog) the restore proceeds without a prompt
   (headless explicit id).

**Verified (print mode):** restoring to the first turn's checkpoint from a
later state returned the exact tool text, added a `before-restore` ref, and
`a.txt` returned to its captured content. The dialog itself (label/turn/
created/branch + preview line + files-only/cancel) is manual — TUI only.

## 6. Refusals

1. `scratch-restore {"id": "no-such-id"}` → tool text
   `No checkpoint no-such-id for this session.` (details `refused: "missing"`).
2. Cross-branch: `git checkout -b feature-x` in `$SMOKE`, then restore a
   `main` checkpoint → `Restore refused: checkpoint on 'main', current
   'feature-x'`.

**Verified:** missing-id path (the agent reported the exact refusal text).
Cross-branch refusal is unit-tested in `tests/scratch.test.ts`.
