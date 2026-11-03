# OpenCode Tool Mapping

Skills speak in actions ("dispatch a subagent", "create a todo", "read a file"). On OpenCode these resolve to the tools below.

| Action skills request | OpenCode equivalent |
| --- | --- |
| Read a file | `read` |
| Write a file | `write` |
| Edit a file | `edit` |
| Run a shell command | `bash` |
| Search file contents | `grep` |
| Find files by name | `glob` |
| Fetch a URL | `webfetch` |
| Invoke a skill | `skill` |
| Dispatch a subagent (`Subagent (general-purpose):` template) | `task` with `subagent_type: "general"` (or `"explore"` for read-only investigation) |
| Resume a subagent (fix rounds) | `task` with the `task_id` from the previous dispatch |
| Ask the user a question | `question` |
| Task tracking ("create a todo", "mark complete") | `todowrite` — replaces the WHOLE list each call with statuses `pending`, `in_progress`, `completed`, `cancelled`; exactly ONE item `in_progress` at a time |

## Task tracking

`todowrite` takes the full todo array on every call — there is no
incremental "mark item X" call. After each task transition
(dispatch → `in_progress`, completion → `completed`), send the updated
full list. Keep exactly one `in_progress` at a time; opencode rejects
the call otherwise. The progress ledger remains the source of truth —
the todo list is its visible projection.

## Instructions file

When a skill mentions "your instructions file", on OpenCode this is
**`AGENTS.md`**, loaded from the workspace root and `~/.config/opencode/`.
