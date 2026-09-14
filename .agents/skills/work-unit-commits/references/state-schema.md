# Checkpoint state schema

Model B: the **durable record lives in the commit** (its `[skillgrid-context]`
body block). The **resume handle** is `.skillgrid/sdd/checkpoint.json`, which is
*derived* from `git log -1` + the last `[skillgrid-context]` block by
`checkpoint-state.sh snapshot` — it is never hand-maintained, so it cannot drift
from history. A fresh session runs `checkpoint-state.sh restore` to pick up.

## checkpoint.json

`checkpoint-state.sh snapshot` writes this. Schema `skillgrid/checkpoint/v1`.

```json
{
  "schema": "skillgrid/checkpoint/v1",
  "updated": "2026-09-14T12:00:00Z",
  "branch": "agent-42",
  "last_commit": "9f1c2ab3c0d1e2f4a5b6c7d8e9f0a1b2c3d4e5f6",
  "last_commit_short": "9f1c2ab",
  "last_commit_subject": "feat(auth): add session refresh",
  "current_task": "auth/session-refresh",
  "completed_tasks": "auth/session-refresh",
  "decisions": "refresh via sliding window, not absolute expiry",
  "remaining": "wire refresh into the token middleware",
  "tried": "none"
}
```

Field notes:
- `branch`, `last_commit*` — straight from `git rev-parse` / `git log -1`.
- `current_task`, `completed_tasks` — derived from the `[skillgrid-context]`
  `Task:` line. The skill refines `completed_tasks` from its todo list before it
  trusts this for resume; the script's value is the last commit's task label.
- `decisions`, `remaining`, `tried` — copied verbatim from the block.
- `updated` — UTC timestamp of the snapshot.

`restore` prints this file **plus** live git truth (`branch`, `HEAD`,
`git status --short`) so a resume is grounded against the file, which may predate
the latest work.

## [skillgrid-context] commit body block

Every work-unit commit carries this in its message body. It is the durable
record; `snapshot` reads it. Format:

```
type(scope): concise description

- key change 1
- key change 2

[skillgrid-context]
Task: <task or ticket id>
Decisions: <key choices this unit made, and why>
Remaining: <what is left in this logical unit — empty if done>
Tried: <failed approaches worth recording — omit line if none>
[/skillgrid-context]
```

Rules:
- `Task:` is the anchor `snapshot` uses for `current_task`.
- `Remaining:` empty means the unit is complete; non-empty means resume starts
  there.
- One-way-door decisions still get an ADR (via
  `skillgrid:architectural-decision-records`); the `Decisions:` line is the
  per-task record, not a replacement for the ADR.

## Resume decision tree

A fresh session starts with `checkpoint-state.sh restore`:

1. **No file** → fresh start. Nothing to resume.
2. **File present, `branch` matches current HEAD branch** → resume:
   - `current_task` / `remaining` tell you where the unit left off.
   - `git status --short` shows uncommitted remainder to finish first.
   - Say "Resuming from `<task>` — last commit `<short>` `<subject>`."
3. **File present, `branch` differs** → ask: "Found a checkpoint from
   `<branch>` (last commit `<short>`). Start fresh on `<current branch>` or
   switch back to `<branch>`?" Do not guess.
4. **`remaining` is empty and tests are green** → the unit is done; proceed to
   the next task or `skillgrid:finishing-a-development-branch`.

## When to snapshot

- After each work-unit commit (the skill runs `snapshot`; it is idempotent).
- Before a long-running install/build/test command (so an interruption lands on
  a checkpoint, not mid-command).
- Before spawning parallel work or crossing a review gate.
- On a PreCompact / session-end hook (CLI-installed later) — `snapshot` is
  already the command it powers.
