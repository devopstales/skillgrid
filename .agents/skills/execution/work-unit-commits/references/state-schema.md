# Commit context + session events schema

Model B: the **durable record lives in the commit** (its `[skillgrid-context]`
body block). Each work-unit commit feeds a **`commit` event** on the session
event stream carrying the parsed block, keyed by commit id. A fresh session
resumes from `git log` plus the session's events — there is no separate resume
file to maintain.

## [skillgrid-context] commit body block

Every work-unit commit carries this in its message body. It is the durable
record; the commit event parses it. Format:

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
- `Task:` is the anchor the resume uses for the current task.
- `Remaining:` empty means the unit is complete; non-empty means resume starts
  there.
- One-way-door decisions still get an ADR (via
  `skillgrid:architectural-decision-records`); the `Decisions:` line is the
  per-task record, not a replacement for the ADR.
- A commit with no block still records an event, with empty parsed detail.

## Commit event detail

The event holds the commit id plus the parsed `Task` / `Decisions` /
`Remaining` / `Tried` fields. Query it per session:

```bash
skillgrid session <session-id>
```

or the `session_changes` MCP tool with `session_id`. Events return in sequence
order with the net commit range (`from_commit` at start, `to_commit` at end);
`to_commit` is the resume position.

## Resume decision tree

A fresh session starts from git state plus the session's events:

1. **No commits yet** → fresh start. Nothing to resume.
2. **Latest commit's `Remaining:` names work** → resume there:
   - `git status --short` shows uncommitted remainder to finish first.
   - Say "Resuming from `<task>` — last commit `<short>` `<subject>`."
3. **Resuming another session** → read its events (`skillgrid session
   <session-id>` or `session_changes`): its `to_commit` is where that session
   left off; its event list shows the tool calls that led there.
4. **`Remaining:` is empty and tests are green** → the unit is done; proceed to
   the next task or `skillgrid:ship`.

## When to commit

- After each work-unit commit (the skill commits; it is idempotent to re-read).
- Before a long-running install/build/test command (so an interruption lands on
  a committed unit, not mid-command).
- Before spawning parallel work or crossing a review gate.
- On a PreCompact / session-end hook — commit first, so the event stream holds
  the position.
