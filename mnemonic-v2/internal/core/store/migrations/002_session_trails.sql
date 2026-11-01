-- 002_session_trails.sql — the session trail (Task 7): the append-only,
-- per-session, sequence-ordered audit log of committed saves. One row per
-- committed save (the SaveEvent consumer, trails.go, appends it). The
-- v1 session_events pattern, ported: per-session monotonically increasing
-- sequence, timestamped, FK to the sessions row (the trail's session is
-- auto-registered by the save's own transaction, so the FK always resolves).
--
-- Additive-only: a fresh DB applies 001 + this; an existing DB (schema
-- version 1) applies this on its next Migrate. Never synced or exported
-- (local-only audit, like session_finalizations).

CREATE TABLE IF NOT EXISTS session_trails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	project TEXT,
	sequence INTEGER NOT NULL,
	action_type TEXT NOT NULL DEFAULT 'save',
	sync_id TEXT,
	timestamp TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_session_trails_session ON session_trails(session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_session_trails_project ON session_trails(project, timestamp DESC);
