# Issue tracker: Hermes Kanban

Tickets for this project live on a Hermes Kanban board — a durable, SQLite-backed task board shared across Hermes profiles. Tasks are claimed atomically, can depend on other tasks, and are executed by a named profile in an isolated workspace.

The board is the **source of truth** for status. A local Backlog.md entry (if one exists for the ticket) is a human-readable **mirror**: it gets the Kanban task ID as a reference, and its status is updated from the board, never the other way around.

## CLI surface

Use `hermes kanban <command> --help` before running unfamiliar commands. The commands this skill uses:

| Purpose | Command |
|---|---|
| List boards | `hermes kanban boards list` |
| Create board | `hermes kanban boards create <slug> [--name <display>]` (create fails when the board exists — non-fatal) |
| List tasks | `hermes kanban --board <slug> list [--status <status>]` |
| Show a task | `hermes kanban --board <slug> show <task-id>` (includes parents + comments) |
| Create a task | `hermes kanban --board <slug> create <title> [options]` |
| Link a dependency | `hermes kanban --board <slug> link <parent-id> <child-id>` |
| Assign / reassign | `hermes kanban --board <slug> assign <task-id> <profile>` |
| Comment | `hermes kanban --board <slug> comment <task-id> <text>` |
| Block (park for ops) | `hermes kanban --board <slug> block <task-id> <reason>` |
| Unblock | `hermes kanban --board <slug> unblock <task-id> [--reason <note>]` |
| Request review | `hermes kanban --board <slug> request-review <task-id> [--summary <text>]` |
| Complete | `hermes kanban --board <slug> complete <task-id> [--result <text>]` |

Board selection: `--board <slug>` is a **top-level** flag on `hermes kanban` — it goes after `kanban` and before the subcommand (`hermes kanban --board <slug> list`). It is NOT a flag on the subcommands (`create` has no `--board`). Alternatives: scope the process with `hermes kanban boards switch <slug>`, or set `HERMES_KANBAN_BOARD=<slug>`. `hermes kanban list` itself accepts no `--board` flag — use the top-level form. Always operate against `ticketing.kanban.board`.

## Discovery (Step 2a for this tracker)

1. **Auth / binary**: `hermes --version` (or `command -v hermes`). If the binary is missing, stop and ask — do not publish against a guessed CLI.
2. **Board**: read `ticketing.kanban.board` from `.skillgrid/config.yaml`. Check `hermes kanban boards list`.
   - Board missing → create it: `hermes kanban boards create <slug>`.
   - Set the process scope once before any task command: `hermes kanban boards switch <slug>` (or export `HERMES_KANBAN_BOARD=<slug>`), so every later `hermes kanban <subcommand>` targets the right board. Alternatively prefix each command with the top-level flag: `hermes kanban --board <slug> <subcommand> ...`.
   - **Board creation failure is non-fatal**: log a warning, then try to create the task on the `default` board and record the fallback in the task body. A missing board must not block publication.
3. **Assignee**: read `ticketing.kanban.assignee`. If the profile does not resolve, publish anyway (unassigned) and note it in the report.
4. **Duplicate check** (Step 2b): `hermes kanban --board <slug> list --status todo`, repeat per status (`--status` is single-value, not repeatable) — search titles for the ticket's title. If found, skip creation and use the existing task ID.

## Publication mapping (Step 2e)

One Kanban task per ticket in `tasks.md`. The epic becomes the board's top-level card:

| Skillgrid field | Kanban field |
|---|---|
| Ticket title | `create <title>` (positional) |
| Ticket type (coding / research) | `--workspace worktree` (coding) / `--workspace scratch` (research) |
| Dependencies (parent tickets) | `--parent <id>` (repeatable) at create, or `hermes kanban --board <slug> link <parent-id> <child-id>` afterwards |
| Assigned profile | `--assignee <profile>` (from `ticketing.kanban.assignee`) |
| Board slug | top-level `--board <slug>` on `hermes kanban` (from `ticketing.kanban.board`) — not a flag on `create` |
| Skill reference | `--skill <skill>` (optional, repeatable — e.g. the execution skill the ticket runs under) |

Body: the ticket's scope + acceptance + fails-when from `tasks.md`, plus a `Blocked by:` line naming the parent Kanban task IDs (redundant with the native link; it keeps the body self-documenting).

Publish in wave order: parents before children, so child tickets can reference their parents' IDs at create time.

### Create example

```bash
hermes kanban --board <project-slug> create "TICKET-02: <title>" \
  --assignee <profile> \
  --workspace worktree \
  --parent <parent-task-id> \
  --skill subagent-execution \
  --body "<ticket body with Blocked by: <parent task ids>>"
```

Multiple parents: repeat `--parent` once per parent. Research tickets use `--workspace scratch` instead of `--workspace worktree`. Long bodies with newlines: use `--body-file` (mutually exclusive with `--body`) instead of inline `--body`.

### Post-create verification (mandatory)

After each create, run `hermes kanban --board <slug> show <task-id>` and confirm:

1. The title matches the ticket title
2. The assignee matches `ticketing.kanban.assignee`
3. Every declared parent appears in the task's parent list
4. The workspace matches the ticket type (`worktree` for coding, `scratch` for research)

If any check fails, fix with `hermes kanban --board <slug> assign` / `hermes kanban --board <slug> link` **before** ending the turn. Do not leave broken tickets.

## Backlog.md mirror

When a ticket was also published to Backlog.md (or a Backlog.md entry exists for it), update the local entry with the Kanban task ID:

- Add the Kanban task ID to the entry's **References** (or a `Kanban task:` line near the top) so `backlog task view <ID> --plain` shows the board link.
- Keep the Backlog.md entry's status in sync *from* the board: mirror the board status into the local status line at each transition (see below). The board wins on any conflict.

When the resolved tracker is **only** Hermes Kanban, Backlog.md may hold no entry at all — `tasks.md` (Step 3) then carries the task ID and is the local mirror.

## Ticket lifecycle (mandatory when a task ID is set)

Creation alone is not enough. When `tasks.md` has a real Kanban task ID, every later execution phase owns a board update:

| Phase | Skillgrid status | Hermes Kanban action |
|---|---|---|
Every command below is prefixed `hermes kanban --board <ticketing.kanban.board>` (top-level flag, before the subcommand) — omitted in the table for readability.

| **Wave start** (blockers all done) | `backlog` → `ready` | `hermes kanban --board <slug> unblock <task-id> --reason "wave <n> ready"` (no-op when the task is already ready) |
| **Execution (start)** | `ready` → `in-progress` | `hermes kanban --board <slug> claim <task-id>` (atomic; prints the resolved workspace path — use it as the worktree for the implementer) |
| **Execution (per commit)** | `in-progress` | Commit footer **`Refs: <task-id>`** |
| **Review** | `in-progress` → `review` | `hermes kanban --board <slug> request-review <task-id> --summary "<what was implemented and how it was verified>"` |
| **Done** | `review` → `done` | `hermes kanban --board <slug> complete <task-id> --result "<verdict>"`; the closing commit uses `Closes <task-id>` |

If a worker stalls and the card must be parked for human ops: `hermes kanban --board <slug> block <task-id> <reason>` (the reason is appended as a comment; optional `--kind {capability,dependency,needs_input,transient}`).

If no task ID exists (e.g. `ticketing.enabled: false`), skip all board mutations. Do not invent a task id — always read it from the create output.

## Epic

Publish the epic as one top-level task titled `Epic: <epic summary from tasks.md>` with `--workspace scratch` (no code changes — it is a tracking card), assigned to the same profile. Every ticket is a child of the epic task (`--parent <epic-task-id>` at create, or `hermes kanban --board <slug> link <epic-task-id> <ticket-task-id>` afterwards). The epic's board status mirrors the skillgrid epic: ready when the first wave starts, done when all tickets are done.

## Rationalizations (tracker-specific)

| Excuse | Reality |
|---|---|
| "The board doesn't exist yet, I'll skip the tickets" | Create the board first (non-fatal fallback: publish to `default` and note the warning). Publish. |
| "I'll skip the duplicate check, the board is new" | Re-runs of the pipeline are not rare. Search titles before create. |
| "I'll link the dependency after, the parent doesn't exist yet" | Publish in wave order — parents first — so `--parent` works at create time. |
| "The workspace flag is just a hint" | It decides where the worker executes. Coding without `worktree` runs on a shared checkout and races other workers. Set it. |
| "Backlog.md is easier, I'll just keep using it" | The repository chose its tracker. Read `ticketing.type` first; the board is the source of truth. |
| "I'll claim it when the subagent asks" | Claiming is the execution-start transition. Do it at wave start pickup, so the status machine stays honest. |
