# Session/Checkpoint Layer Consolidation

## Briefing

### Problem
`checkpoint.json` + the Handoff Hub are two overlapping, engine-heavy
resume mechanisms:

- `checkpoint.json` (`.agents/hooks/checkpoint-state.sh`, Model B) is a
  **derived** per-change resume handle written after each work-unit commit.
- The Handoff Hub (`skillgrid handoff` CLI over SQLite: `change_snapshots`,
  `checkpoints`, `handoff_refs`) is a queryable change log + named-checkpoint
  store, plus the cleave relay (`session_handoffs`/`session_archives` over
  `.skillgrid/.cleave/`).

Both are derived from git, both are best-effort, and both drift from the
`sessions` table that already anchors every session (`001_initial.sql`). There
is no single place that answers "what changed in session N."

### Goal
Replace both with a single gryph-inspired session→events layer: one `sessions`
table (extended) plus a per-tool-call `session_events` stream. Per-tool-call
fidelity, gryph's `is_sensitive` + counter set, and a committed capture bridge
for opencode, Kilo Code, and Cursor (observe-only).

### Intent
- The session's changes **are** its events, ordered by `sequence`.
- Git is the durable record; the DB is the index (gryph's model).
- "What changed in this session" = `SELECT ... WHERE session_id=? ORDER BY
  sequence` + `git diff from_commit..to_commit` (derived, on demand).
- A crash/timeout in any capture hook never blocks the agent (fail-open).

## Locked decisions
1. Superpowers per-plan layout for `.skillgrid/sdd/`.
2. One table = `sessions` (extended) + `session_events` (per-tool-call stream).
3. Per-tool-call fidelity; adopt gryph `is_sensitive` + counter set.
4. Drop `.skillgrid/.cleave/` (bundles, relay, `session_handoff/resume/status`
   MCP tools).
5. Drop `handoff_refs` (+ writer `relay.RecordTeamRef`, `envelope.go:165`
   field).
6. Capture bridge committed to repo as adapters (no `skillgrid hook install`).
7. Observe-only (no pre-tool deny-gates this consolidation).
8. Repo hook layout: `$REPO/hooks/` (implementations), `$REPO/git-hooks/`
   (shims), `$REPO/plugins/` (capture bridge). Install syncs the repo to
   `~/.skillgrid/` and consumes from there: plugins from
   `~/.skillgrid/plugins`, git hooks from `~/.skillgrid/git-hooks`, impl via
   copy from `~/.skillgrid/hooks`.

## Data model

### Migration `040_session_events.sql` (additive)
```sql
ALTER TABLE sessions ADD COLUMN agent_session_id TEXT;
ALTER TABLE sessions ADD COLUMN from_commit TEXT;   -- git HEAD at session_start
ALTER TABLE sessions ADD COLUMN to_commit   TEXT;   -- git HEAD at session_end
ALTER TABLE sessions ADD COLUMN files_read        INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN files_written     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN commands_exec     INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN errors            INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN sensitive_actions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN blocked_actions   INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS session_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  project TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  action_type TEXT NOT NULL,        -- session_start|session_end|file_read|file_write|command_exec|tool_use|commit|subagent_start|subagent_stop|prompt
  result_status TEXT NOT NULL DEFAULT 'success',  -- success|error|blocked|rejected
  is_sensitive INTEGER NOT NULL DEFAULT 0,
  tool_name TEXT,
  path TEXT,
  command TEXT,
  commit TEXT,
  payload TEXT,                     -- JSON: diff stat, [skillgrid-context], exit code, tokens
  timestamp TEXT NOT NULL,
  UNIQUE (session_id, sequence)
);
CREATE INDEX IF NOT EXISTS idx_events_session_seq ON session_events (session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_events_project_time ON session_events (project, timestamp DESC);
```
`sequence` = per-session monotonic (max+1 in the same transaction as the
counter bump).

### Migration `041_drop_handoff_tables.sql` (after all readers gone)
```sql
DROP TABLE IF EXISTS change_snapshots;
DROP TABLE IF EXISTS checkpoints;
DROP TABLE IF EXISTS handoff_refs;
DROP TABLE IF EXISTS session_handoffs;
DROP TABLE IF EXISTS session_archives;
```
(+ their indexes `idx_snapshots_project_time`, `idx_checkpoints_project_status`,
`idx_handoff_refs_project`, `idx_handoffs_source`, `idx_handoffs_status`,
`idx_archives_session`.)

## Capture bridge (gryph-modeled)

### Canonical event (trimmed from gryph `event.schema.json`)
```json
{ "agent": "opencode|kilocode|cursor", "agent_session_id": "raw id",
  "hook_event": "sessionStart|postToolUse|preToolUse|subagentStart|subagentStop|stop|sessionEnd|prompt",
  "tool_name": "Write|Shell|Read|MCP:<name>", "tool_input": {}, "tool_output": {},
  "cwd": "/abs/path", "conversation_id": "cursor", "model": "...", "duration_ms": 12 }
```

### Single ingest endpoint
`skillgrid mem hook post_tool_use --json <canonical>` (CLI seam at
`mem.go:1035` → `RunHook`, `memory/skills.go:384`; extend `HookPayload`
`skills.go:255` with `action_type`, `tool_name`, `command`, `result_status`).
Steps:
1. Resolve session via `uuid.NewSHA1(oid, agent_session_id)`, upsert `sessions`.
2. Map `hook_event`+`tool_name` → `action_type`.
3. Gryph sensitive path/credential matcher → `is_sensitive`.
4. Insert `session_events` row + bump counter in one transaction.
`from_commit`/`to_commit` via `git rev-parse HEAD` at session_start/end
(best-effort, `cwd` from payload). Fail-open: every adapter wraps the call
(try/finally or `|| true`, short timeout); Cursor `failClosed` stays false;
content stored as hash/preview only (gryph redaction rule).

### Per-agent adapters (committed to repo under `plugins/`)
| Agent | File(s) | Source |
|-------|---------|--------|
| opencode | `plugins/opencode/mnemonic.ts` | `tool.execute.after` `{tool,sessionID,callID,args}`+output; `event` → `session.created`/`session.idle`; `tool.execute.before` (observe) |
| Kilo Code | `plugins/kilo/mnemonic.ts` | `tool.execute.after`, `chat.message`, `event` → `session.created`/`session.idle` (shares ~90% with opencode emitter) |
| Cursor | `plugins/cursor/mnemonic.mdc` | `postToolUse`, `sessionStart`, `subagentStart/Stop`, `stop`, shell hooks |

Session-lifecycle map:
- start = `session.created`/`sessionStart` (+`from_commit`)
- tool = `tool.execute.after`/`postToolUse`
- pre-tool = `tool.execute.before`/`preToolUse` (observe-only)
- subagent = `event` subagent / `subagentStart`/`subagentStop`
- end = `session.idle`/`stop` (+`to_commit`)
- prompt = `session.next.prompted`/`chat.message`/`beforeSubmitPrompt`
- commit = existing post-commit flow calling the same endpoint

### Read path
`SessionChanges(ctx, sessionID)` → `SELECT * FROM session_events WHERE
session_id=? ORDER BY sequence` + `git diff --stat from..to`. Backs
`skillgrid session <id> [--show-diff]` CLI + MCP tool (gryph `gryph session`
equivalent).

## Go removals (verified blast radius)
- `internal/mnemonic/handoff/` (snapshot, checkpoint, refs, rollup, context,
  handoff + tests)
- `cmd/skillgrid/handoff.go` (+tests) + `main.go` dispatch strip;
  `cmd/skillgrid/session.go` (handoff/resume/status) + `session_test.go`
- `mcp/tools_handoff.go` (5 tools) + `registerHandoffTools` in `server.go:66,99`;
  `mcp/tools_session_handoff.go` (session_handoff/resume);
  `mcp/tools_session_status.go` (session_status/knowledge_compact)
- `internal/mnemonic/http/handoff.go` + test + strip `activity.go`
- `internal/mnemonic/relay/` (relay, cleave, status, compact, watchdog + tests)
- `memory/envelope.go:165` HandoffRefs field; `checkpoint.go:278 detectHandoffFile`
- UI: `skillgrid-ui/.../sessions/HandoffsPane.tsx`, `ChangesPane.tsx`, `api.ts`

### Bash
`hooks/checkpoint-state.sh`: drop `cmd_snapshot`/`cmd_restore` + `skillgrid
handoff record` mirror; **keep** `guard`/`guard-msg`/`post-check`. Repoint any
stop/compact hook to SessionEnd capture.

### Skills (14 files, repoint to events model)
work-unit-commits (SKILL + state-schema + checkpoint.md), resume (resume =
latest session events + `to_commit`), using-skillgrid, subagent-execution
(SKILL + setup + implementer-prompt), simple-execution, ship (drop `handoff
archive`), slicing, onboarding (SKILL + config.yaml `state_file`),
_shared/conventions/sdd-structure.md, mnemonic/references/memory.md.

## Hook/plugin install (part of `skillgrid install`)
Repo source: `hooks/` (implementations), `git-hooks/` (shims), `plugins/`
(capture bridge). Install mirrors the whole repo tree (`$REPO/*`, recursively)
to `~/.skillgrid/` (remove-then-copy per top-level entry, dry-run aware),
excluding `.git`/`node_modules` (any depth) and top-level `.skillgrid`/`dist`/
`out`, and never touching live `mnemonic/`/`repos/`/`backup/`/`bin/`/`tmp/`/
`logs`/`config.d`. Then:
- installs agent plugins from `~/.skillgrid/plugins` (mirror-first reads,
  repo fallback),
- wires git `core.hooksPath` → `~/.skillgrid/git-hooks`,
- shims resolve impl via `../hooks/` relative (works in repo and mirror).

## `.skillgrid/sdd/` (superpowers layout)
`.skillgrid/sdd/<YYYY-MM-DD-topic>/` = `progress.md` (line 1 names the plan
file), `task-N-brief.md`, `task-N-report.md`, `review-<base7>..<head7>.diff`;
self-ignoring `.gitignore` (content `*`); `checkpoint.json` no longer written;
repath `sdd-workspace`/`task-brief`/`review-package` scripts.

## Execution order & gates
1. A (additive schema `040`) → 2. B (wire capture + `SessionChanges`) →
   3. C (removals) → 4. D (skills) → 5. E (sdd layout) → 6. drop migration `041`
   → 7. hook/plugin install wiring.

Strict TDD: each removal's `*_test.go` goes RED in the same change as its
code; new `session_events` write + `SessionChanges` read get GREEN tests
first. Backfill: replay `git log` per branch for open sessions; existing
`change_snapshots` rows are a derived index (dead until drop migration).

## Risks
- Capture bridge is the critical path — per-tool-call fidelity only as good as
  the agent→CLI hook; degrade gracefully to session_start/end + commits if an
  adapter isn't wired.
- `sequence` atomicity in-transaction with counter bump (parallel subagents).
- Kilo is opencode-based; shared emitter, two thin shells.

## Notes for user (pre-execution)
- `.cursor/hooks.json` lives at repo root (project-level, committed) — deferred
  with the rest of the cursor bridge (adapter files exist under `plugins/`).
- Keep the opencode/Kilo plugins dependency-free (shell out to CLI) so no
  `package.json` deps are needed.
- Observe-only: `preToolUse`/`tool.execute.before` adapters log but return no
  deny/permission shape this consolidation.
