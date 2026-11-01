-- 002_session_trails.sql — the session trail (Task 7) PG render: the
-- append-only, per-session, sequence-ordered audit log of committed saves.
-- Same semantics as the SQLite render; additive-only.

CREATE TABLE IF NOT EXISTS session_trails (
	id BIGSERIAL PRIMARY KEY,
	session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
	project TEXT,
	sequence INTEGER NOT NULL,
	action_type TEXT NOT NULL DEFAULT 'save',
	sync_id TEXT,
	timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_trails_session ON session_trails(session_id, sequence);
CREATE INDEX IF NOT EXISTS idx_session_trails_project ON session_trails(project, timestamp DESC);
