---
name: skillgrid-checkpoints
description: >
  Record Tier 1 operational checkpoints: git fingerprint, change context, handoff sync,
  checkpoints.log index, and JSONL timeline event. Trigger: before apply, after loop reflect,
  verify pass, pre-archive, handoff create; or when user says "checkpoint", "record checkpoint".
license: Apache-2.0
metadata:
  author: devopstales
  version: "1.0"
---

## Purpose

Capture a **safe resume point** for the active OpenSpec change on the **current branch** (single working tree — no git worktrees). Checkpoints are cheap, deterministic, and file-first.

## When to run

| Trigger | `--name` | `--trigger` | Phase |
|---------|----------|-------------|-------|
| Before `/sdd-apply` (after gate passes) | `before-apply` | `before-apply` | `apply` |
| After `/sdd-loop` reflect | `after-loop-<N>` | `after-loop` | `loop` |
| After `/sdd-verify` PASS | `verify-pass` | `verify-pass` | `verify` |
| Before `/sdd-archive` | `pre-archive` | `pre-archive` | `archive` |
| Handoff `create` mode | `handoff-create` | `handoff-create` | `handoff` |

## Command

From repo root (after `sdd-gate` passes when applicable):

```bash
.skillgrid/scripts/checkpoint-record.sh \
  --change <change-id> \
  --name <label> \
  --trigger <trigger-id> \
  --phase <phase> \
  --evidence "<one-line summary>" \
  [--slice "<active task line>"]
```

CLI equivalent:

```bash
skillgrid checkpoint --change <change-id> --name <label> [same options]
```

## What it writes

1. **Append** one line to `.skillgrid/tasks/checkpoints.log` (dashboard index).
2. **Update** `## Last checkpoint` in `.skillgrid/tasks/context_<change-id>.md` (if file exists).
3. **Append** JSONL event with `node: "checkpoint"` to `.skillgrid/tasks/events/<change-id>.jsonl`.

## Coordinator rules

- Run **before apply** immediately after `sdd-gate.sh apply` exits 0 — before reading specs or editing code.
- Run **after loop reflect** once per `/sdd-loop` iteration (include slice/task in `--evidence` or `--slice`).
- Run **verify-pass** only when verification verdict is PASS (or equivalent).
- Run **pre-archive** before moving OpenSpec folders or merging specs.
- Do **not** replace handoff or events with checkpoints; they complement each other.

## Evidence examples

- `apply gate passed; starting slice 2`
- `loop iteration 3 completed; tests green`
- `verify PASS; traceability matrix complete`
- `pre-archive; branch ready for merge`

## Out of scope

- Full session markdown snapshots (Tier 2 — not implemented here).
- Git worktrees or parallel implementation lanes.
- Replacing `mem_save` / Engram session summaries.

## Related

- **`docs/18-checkpoints.md`** — canonical human reference (format, triggers, CLI, dashboard, troubleshooting)
- `.agents/skills/_shared/skillgrid-handoff.md` — handoff + event contract
- `docs/15-webui.md` — where checkpoints appear in the Web UI
- `skillgrid-vertical-slices` — slice-level checkpoint names in tasks
