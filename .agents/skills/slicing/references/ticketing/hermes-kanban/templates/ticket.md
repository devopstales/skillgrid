# Hermes Kanban task body template

Fill with the ticket's fields from `tasks.md`. Keep it self-documenting — the
native `--parent` links already carry the dependency; the `Blocked by:` line is
redundant on purpose so the body stands alone.

```
TICKET-<NN>: <title>

Scope: <one line: what this ticket builds>

Acceptance:
- <falsifiable criterion 1>
- <falsifiable criterion 2>

Fails-when: <output/exit-code that means the verify command failed>

Blocked by: <parent kanban task id> (or "none")
```

Notes:

- Coding tickets: `--workspace worktree`. Research tickets: `--workspace scratch`.
- Long bodies with newlines: use `--body-file` (mutually exclusive with `--body`).
- The epic is a top-level card: title `Epic: <epic summary from tasks.md>`,
  `--workspace scratch`, every ticket created with `--parent <epic-task-id>`.
