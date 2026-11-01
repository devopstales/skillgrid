-- 040: session events layer (replaces checkpoint file + handoff hub + cleave relay).
--
-- Additive only. sessions gains the commit range (from_commit at session
-- start, to_commit at session end) plus the agent session identity and the
-- per-session activity counters; session_events is the append-only
-- per-tool-call stream ordered by (session_id, sequence).
--
--   agent_session_id — the caller's session UUID (opencode/kilocode/cursor
--                      raw id); the sessions.id row key stays authoritative.
--   from_commit      — `git rev-parse HEAD` in the session directory at
--                      session start (empty outside a repo, never an error).
--   to_commit        — HEAD at session end.
--   *_read/_written/commands_exec/errors/sensitive_actions/blocked_actions —
--                      counters bumped in the same transaction as each event
--                      insert (see memory.nextSequence).
ALTER TABLE sessions ADD COLUMN agent_session_id TEXT;
ALTER TABLE sessions ADD COLUMN from_commit TEXT;
ALTER TABLE sessions ADD COLUMN to_commit TEXT;
ALTER TABLE sessions ADD COLUMN files_read INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN files_written INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN commands_exec INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN errors INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN sensitive_actions INTEGER NOT NULL DEFAULT 0;
ALTER TABLE sessions ADD COLUMN blocked_actions INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    result_status TEXT NOT NULL DEFAULT 'success',
    is_sensitive INTEGER NOT NULL DEFAULT 0,
    tool_name TEXT,
    path TEXT,
    command TEXT,
    "commit" TEXT,
    payload TEXT,
    timestamp TEXT NOT NULL,
    UNIQUE (session_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_events_session_seq ON session_events (session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_events_project_time ON session_events (project, timestamp DESC);
