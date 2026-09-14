---
status: in-progress          # in-progress | done | blocked
branch: {branch}
last_commit: {short}         # e.g. 9f1c2ab
last_commit_subject: {subject}
timestamp: {iso}             # UTC
task: {task}
---

# Checkpoint: {title}

> Derived from `.skillgrid/sdd/checkpoint.json` + `git log -1`. This file is a
> human-readable *view*; the JSON is the source of truth. Re-run
> `checkpoint-state.sh snapshot` to refresh — do not hand-edit.

## Working on: {title}

### Summary
{1-3 sentences: the goal and where the unit currently stands.}

### Decisions
- {key choice and why — from the [skillgrid-context] block}

### Remaining
1. {concrete next step, in priority order}

### Notes
- {gotchas, blocked items, things tried that didn't work — from `Tried:`}

---
Restore: run `checkpoint-state.sh restore` in a fresh session.
