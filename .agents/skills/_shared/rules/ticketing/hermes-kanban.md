# Issue tracker: Hermes Kanban

Tickets for this project live on a Hermes Kanban board — a durable, SQLite-backed task board shared across Hermes profiles. Tasks are claimed atomically, can depend on other tasks, and are executed by a named profile in an isolated workspace.

The board is the **source of truth** for status. A local Backlog.md entry (if one exists for the ticket) is a human-readable **mirror**: it gets the Kanban task ID as a reference, and its status is updated from the board, never the other way around.

## CLI surface

Use `hermes kanban <command> --help` before running unfamiliar commands. The commands this skill uses:

| Purpose | Command |
|---|---|
| List boards | `hermes kanban boards list` |
| Create board | `hermes kanban boards create <slug>` (idempotency: create fails when the board exists — non-fatal) |
| List tasks | `hermes kanban list [--board <slug>] [--status <status>]` |
| Show a task | `hermes kanban show <task-id>` (includes parents + comments) |
| Create a task | `hermes kanban create <title> [options]` |
| Link a dependency | `hermes kanban link <parent-id> <child-id>` |
| Assign / reassign | `hermes kanban assign <task-id> <profile>` |
| Comment | `hermes kanban comment <task-id> <text>` |
| Block (park for ops) | `hermes kanban block <task-id> <reason>` |
| Unblock | `hermes kanban unblock <task-id> [--reason <note>]` |
| Request review | `hermes kanban request-review <task-id> [--summary <text>]` |
| Complete | `hermes kanban complete <task-id> [--result <text>]` |

Board selection for non-board-level commands: pass `--board <slug>` when the command's top-level parser accepts it, or scope the session with `hermes kanban boards switch <slug>` first. Always operate against `ticketing.kanban.board`.

## Discovery (Step 2a for this tracker)

1. **Auth / binary**: `hermes --version` (or `command -v hermes`). If the binary is missing, stop and ask — do not publish against a guessed CLI.
2. **Board**: read `ticketing.kanban.board` from `.skillgrid/config.yaml`. Check `hermes kanban boards list`.
   - Board missing → create it: `hermes kanban boards create <slug>`.
   - **Board creation failure is non-fatal**: log a warning, then try to create the task on the `default` board and record the fallback in the task body. A missing board must not block publication.
3. **Assignee**: read `ticketing.kanban.assignee`. If the profile does not resolve, publish anyway (unassigned) and note it in the report.
4. **Duplicate check** (Step 2b): `hermes kanban list [--status todo --status running --status review --status done]` — search titles for the ticket's title. If found, skip creation and use the existing task ID.

## Publication mapping (Step 2e)

One Kanban task per ticket in `tasks.md`. The epic becomes the board's top-level card:

| Skillgrid field | Kanban field |
|---|---|
| Ticket title | `hermes kanban create <title>` (positional) |
| Ticket type (coding / research) | `--workspace worktree` (coding) / `--workspace scratch` (research) |
| Dependencies (parent tickets) | `--parents <id1,id2>` at create, or `hermes kanban link <parent-id> <child-id>` afterwards |
| Assigned profile | `--assignee <profile>` (from `ticketing.kanban.assignee`) |
| Board slug | `--board <slug>` (from `ticketing.kanban.board`) |
| Skill reference | `--skill <skill>` (optional — e.g. the execution skill the ticket runs under) |

Body: the ticket's scope + acceptance + fails-when from `tasks.md`, plus a `Blocked by:` line naming the parent Kanban task IDs (redundant with the native link; it keeps the body self-documenting).

Publish in wave order: parents before children, so child tickets can reference their parents' IDs at create time.

### Create example

```bash
hermes kanban create "TICKET-02: <title>" \
  --board <project-slug> \
  --assignee <profile> \
  --workspace worktree \
  --parents <parent-task-id> \
  --skill subagent-execution \
  --body "<ticket body with Blocked by: <parent task ids>>"
```

Research tickets use `--workspace scratch` instead of `--workspace worktree`.

### Post-create verification (mandatory)

After each create, run `hermes kanban show <task-id>` and confirm:

1. The title matches the ticket title
2. The assignee matches `ticketing.kanban.assignee`
3. Every declared parent appears in the task's parent list
4. The workspace matches the ticket type (`worktree` for coding, `scratch` for research)

If any check fails, fix with `hermes kanban assign` / `hermes kanban link` **before** ending the turn. Do not leave broken tickets.

## Backlog.md mirror

When a ticket was also published to Backlog.md (or a Backlog.md entry exists for it), update the local entry with the Kanban task ID:

- Add the Kanban task ID to the entry's **References** (or a `Kanban task:` line near the top) so `backlog task view <ID> --plain` shows the board link.
- Keep the Backlog.md entry's status in sync *from* the board: mirror the board status into the local status line at each transition (see below). The board wins on any conflict.

When the resolved tracker is **only** Hermes Kanban, Backlog.md may hold no entry at all — `tasks.md` (Step 3) then carries the task ID and is the local mirror.

## Ticket lifecycle (mandatory when a task ID is set)

Creation alone is not enough. When `tasks.md` has a real Kanban task ID, every later execution phase owns a board update:

| Phase | Skillgrid status | Hermes Kanban action |
|---|---|---|
| **Wave start** (blockers all done) | `backlog` → `ready` | `hermes kanban unblock <task-id> --reason "wave <n> ready"` (no-op when the task is already ready) |
| **Execution (start)** | `ready` → `in-progress` | `hermes kanban claim <task-id>` (atomic; prints the resolved workspace path — use it as the worktree for the implementer) |
| **Execution (per commit)** | `in-progress` | Commit footer **`Refs: <task-id>`** |
| **Review** | `in-progress` → `review` | `hermes kanban request-review <task-id> --summary "<what was implemented and how it was verified>"` |
| **Done** | `review` → `done` | `hermes kanban complete <task-id> --result "<verdict>"`; the closing commit uses `Closes <task-id>` |

If a worker stalls and the card must be parked for human ops: `hermes kanban block <task-id> <reason>` (the reason is appended as a comment).

If no task ID exists (e.g. `ticketing.enabled: false`), skip all board mutations. Do not invent a task id — always read it from the create output.

## Epic

Publish the epic as one top-level task titled `Epic: <epic summary from tasks.md>` with `--workspace scratch` (no code changes — it is a tracking card), assigned to the same profile. Every ticket is a child of the epic task (`--parents <epic-task-id>` or `hermes kanban link <epic-task-id> <ticket-task-id>`). The epic's board status mirrors the skillgrid epic: ready when the first wave starts, done when all tickets are done.

## Rationalizations (tracker-specific)

| Excuse | Reality |
|---|---|
| "The board doesn't exist yet, I'll skip the tickets" | Create the board first (non-fatal fallback: publish to `default` and note the warning). Publish. |
| "I'll skip the duplicate check, the board is new" | Re-runs of the pipeline are not rare. Search titles before create. |
| "I'll link the dependency after, the parent doesn't exist yet" | Publish in wave order — parents first — so `--parents` works at create time. |
| "The workspace flag is just a hint" | It decides where the worker executes. Coding without `worktree` runs on a shared checkout and races other workers. Set it. |
| "Backlog.md is easier, I'll just keep using it" | The repository chose its tracker. Read `ticketing.type` first; the board is the source of truth. |
| "I'll claim it when the subagent asks" | Claiming is the execution-start transition. Do it at wave start pickup, so the status machine stays honest. |
