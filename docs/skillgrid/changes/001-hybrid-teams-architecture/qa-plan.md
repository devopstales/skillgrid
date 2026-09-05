# QA plan: 001-hybrid-teams-architecture

Human gate after agent verify. Waive only with explicit note; otherwise exercise below.

## Environments

- Worktree: `/data/git/AI/skillgrid-wt-001-hybrid-teams` branch `feat/001-hybrid-teams-architecture`
- Build: `cd skillgrid-cli && go test ./...`
- Optional: run mnemonic HTTP with `SKILLGRID_HTTP_TOKEN=test` against a temp project dir

## Happy path

1. Spawn a task (service or MCP `team_spawn_task`) with a brief; confirm `pending` id and `files/tasks/{id}/brief.md` on disk.
2. Pull with a member id; confirm highest priority claimed and brief readable.
3. Submit output → status `review_spec`; submit passed review; mark done → `done` + `task_results`.

## Edge / failure

1. Empty pull → clear error, nothing claimed.
2. Bad MCP spawn (missing title/brief) → tool error, no orphan under `.skillgrid/files/tasks/`.
3. HTTP: POST `/teams/tasks` without bearer when token set → **401**; GET `/teams/tasks/{id}` without bearer → **200**; `/memory/reviews` still serves.

## Pass criteria

- Steps above succeed; no panics; no orphan markdown after failed SQL/spawn.
- After apply fixes **02.6** / **02.7**: concurrent pull does not double-claim; second same-type review does not wipe prior comments file.

## Waive

Reply `waive QA` with reason, or `QA accepted` after exercising the plan.

## Result

**Accepted** 2026-09-05 — user message: `QA accepted`
