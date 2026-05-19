# Skillgrid scripts

Executable helpers installed under `.skillgrid/scripts/` in each target repo. Run from the **repository root** unless noted.

The Web UI is **not** in this folder — use `skillgrid serve` from [skillgrid-cli](../../skillgrid-cli/) (see [docs/15-webui.md](../../docs/15-webui.md)).

## SDD gates and apply

| Script | Purpose | Called by |
|--------|---------|-----------|
| `sdd-gate.sh` | Phase gates (labels, artifacts, routing) | `/sdd-apply`, `/sdd-verify`, `/sdd-loop`, git hooks via `install.sh` |
| `validate-task-labels.sh` | `[Label: AFK\|HITL]` and `[Budget: …]` on `tasks.md` | `sdd-gate.sh`, `sdd-apply` / `sdd-verify` skills |
| `checkpoint-record.sh` | Tier 1 checkpoint (log + handoff + event) | SDD workflows, `skillgrid checkpoint` |
| `sdd-ralph-loop.sh` | AFK multi-iteration `/sdd-loop` driver | [docs/10-sdd-ralph-loop.md](../../docs/10-sdd-ralph-loop.md) |

Docs: [18-checkpoints.md](../../docs/18-checkpoints.md), [07-hooks-and-automation.md](../../docs/07-hooks-and-automation.md).

## UI previews

| Script | Purpose |
|--------|---------|
| `preview.sh` | Scaffold HTML or markdown under `.skillgrid/preview/` for `/sdd-design-ui` |

## Change-scoped handoff (primary SDD)

Active change state lives in `.skillgrid/tasks/context_<change-id>.md` (template: `template-handoff-context.md`). Agents maintain it per `skillgrid-handoff.md`; checkpoints update **Last checkpoint** via `checkpoint-record.sh`.

| Script | Purpose |
|--------|---------|
| `handoff-registry-init.sh` | Create `registry_<change-id>.md` from template (compact dispatch index) |

## Session handoffs (optional)

Timestamped snapshots under `.skillgrid/handoffs/` for **session-to-session** transfer (not a substitute for `context_<change-id>.md`).

| Script | Purpose |
|--------|---------|
| `handoff-create.sh` | `full` or `quick` session handoff; optional 4th arg `change-id` runs checkpoint |
| `handoff-resume.sh` | Resume latest or given file; `list` lists files; runs validate + staleness |
| `handoff-validate.sh` | Section/placeholder checks on a session handoff file |
| `handoff-check-staleness.sh` | Age, commits since file, working-tree drift |

Examples: [docs/02-workflow-usage.md](../../docs/02-workflow-usage.md#session-handoff-helpers).
