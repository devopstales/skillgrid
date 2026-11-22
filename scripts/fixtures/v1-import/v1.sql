CREATE TABLE sessions (
  id TEXT PRIMARY KEY, project TEXT NOT NULL, directory TEXT NOT NULL,
  started_at TEXT NOT NULL, ended_at TEXT, summary TEXT, status TEXT NOT NULL DEFAULT 'active',
  title TEXT
);
CREATE TABLE observations_core (
  id INTEGER PRIMARY KEY AUTOINCREMENT, session_id TEXT NOT NULL REFERENCES sessions(id),
  type TEXT NOT NULL, title TEXT NOT NULL, content TEXT NOT NULL, project TEXT, scope TEXT,
  topic_key TEXT, normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT, knowledge_class TEXT
);
CREATE TABLE observation_state (
  observation_id INTEGER PRIMARY KEY REFERENCES observations_core(id) ON DELETE CASCADE,
  pinned INTEGER NOT NULL DEFAULT 0, review_after TEXT, status TEXT, visibility TEXT
);
INSERT INTO sessions VALUES
  ('sess-v1-1', 'fixturetest', '/repo/fixturetest', '2026-01-01 10:00:00', '2026-01-01 11:00:00', 'fixed the n+1', 'active', 'N+1 fix session'),
  ('sess-v1-2', 'fixturetest', '/repo/fixturetest', '2026-01-02 09:00:00', NULL, NULL, 'active', NULL);
INSERT INTO observations_core (session_id, type, title, content, project, scope, created_at, updated_at) VALUES
  ('sess-v1-1', 'bugfix', 'Fixed N+1 query', 'batched fetch in UserList', 'fixturetest', 'project', '2026-01-01 10:05:00', '2026-01-01 10:05:00'),
  ('sess-v1-1', 'decision', 'Chose postgres for kb', 'scale reasons', 'fixturetest', NULL, '2026-01-01 10:20:00', '2026-01-01 10:20:00'),
  ('sess-v1-2', 'discovery', 'Third observation', 'third content', 'fixturetest', 'project', '2026-01-02 09:30:00', '2026-01-02 09:30:00');
-- the orphan: its session is missing from the v1 store, so the import skips it
INSERT INTO observations_core (session_id, type, title, content, project, scope, created_at, updated_at) VALUES
  ('sess-orphan', 'discovery', 'Orphan observation', 'session missing from v1 store', 'fixturetest', 'project', '2026-01-03 08:00:00', '2026-01-03 08:00:00');
INSERT INTO observation_state (observation_id, pinned) VALUES (1, 1);
