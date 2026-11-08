-- 001_schema.sql — the full spec 002-schema SQLite render (fresh DB applies
-- this once; later migrations are additive).

CREATE TABLE IF NOT EXISTS index_meta (
	key TEXT PRIMARY KEY,
	schema_version INTEGER NOT NULL DEFAULT 0
);

-- ── Memory domain (spec 002-schema §2) ──────────────────────────────────────

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	sync_id TEXT,
	project TEXT NOT NULL,
	ownership_mode TEXT NOT NULL DEFAULT 'shared',
	directory TEXT NOT NULL,
	title TEXT,
	started_at TEXT NOT NULL DEFAULT (datetime('now')),
	ended_at TEXT,
	summary TEXT,
	runtime_lease_expires_at TEXT,
	created_at TEXT,
	local_creation_project TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);

-- Local-only idempotent finalize retry journal; never synced or exported.
CREATE TABLE IF NOT EXISTS session_finalizations (
	session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
	summary_id INTEGER NOT NULL,
	summary_sync_id TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS observations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sync_id TEXT UNIQUE,
	session_id TEXT NOT NULL REFERENCES sessions(id),
	type TEXT NOT NULL,
	title TEXT NOT NULL,
	content TEXT NOT NULL,
	tool_name TEXT,
	project TEXT,
	scope TEXT NOT NULL DEFAULT 'project',
	topic_key TEXT,
	normalized_hash TEXT,
	revision_count INTEGER NOT NULL DEFAULT 0,
	duplicate_count INTEGER NOT NULL DEFAULT 0,
	last_seen_at TEXT,
	pinned BOOLEAN NOT NULL DEFAULT 0,
	review_after TEXT,
	superseded_by_sync_id TEXT,
	expires_at TEXT,
	embedding BLOB,
	embedding_model TEXT,
	embedding_created_at TEXT,
	importance REAL NOT NULL DEFAULT 0.5,
	retrieval_usage INTEGER NOT NULL DEFAULT 0,
	provenance TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now')),
	deleted_at TEXT
);

-- spec 002-schema §2.2: the two dedupe/topic indexes.
CREATE INDEX IF NOT EXISTS idx_obs_dedupe ON observations(normalized_hash, project, scope, type, title);
CREATE INDEX IF NOT EXISTS idx_obs_topic ON observations(project, scope, topic_key);

CREATE TABLE IF NOT EXISTS observation_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
	version INTEGER NOT NULL,
	title TEXT,
	content TEXT NOT NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	UNIQUE(observation_id, version)
);

CREATE TABLE IF NOT EXISTS user_prompts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sync_id TEXT UNIQUE,
	session_id TEXT NOT NULL REFERENCES sessions(id),
	content TEXT NOT NULL,
	project TEXT,
	source_inbox_id TEXT,
	created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_prompts_session ON user_prompts(session_id);
CREATE INDEX IF NOT EXISTS idx_prompts_project ON user_prompts(project);

-- Deleted prompts (local); sync_id is the identity key.
CREATE TABLE IF NOT EXISTS prompt_tombstones (
	sync_id TEXT PRIMARY KEY,
	session_id TEXT,
	project TEXT,
	deleted_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- spec 002-schema §2.5: keys are sync_id (TEXT), cross-machine portable.
CREATE TABLE IF NOT EXISTS memory_relations (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sync_id TEXT UNIQUE,
	source_id TEXT NOT NULL,
	target_id TEXT NOT NULL,
	relation TEXT NOT NULL DEFAULT 'pending',
	reason TEXT,
	evidence TEXT,
	confidence REAL,
	judgment_status TEXT NOT NULL DEFAULT 'pending',
	marked_by_actor TEXT,
	marked_by_kind TEXT,
	marked_by_model TEXT,
	session_id TEXT,
	superseded_at TEXT,
	superseded_by_relation_id INTEGER REFERENCES memory_relations(id) ON DELETE SET NULL,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_memrel_source ON memory_relations(source_id, judgment_status);
CREATE INDEX IF NOT EXISTS idx_memrel_target ON memory_relations(target_id, judgment_status);
CREATE INDEX IF NOT EXISTS idx_memrel_supersede ON memory_relations(superseded_by_relation_id);

-- ── Sync domain (spec 002-schema §3) ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sync_mutations (
	seq INTEGER PRIMARY KEY AUTOINCREMENT,
	target_key TEXT NOT NULL,
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	op TEXT NOT NULL,
	payload TEXT NOT NULL,
	source TEXT NOT NULL DEFAULT 'local',
	project TEXT,
	occurred_at TEXT NOT NULL DEFAULT (datetime('now')),
	acked_at TEXT,
	disposition TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_mutations_project ON sync_mutations(project);

CREATE TABLE IF NOT EXISTS sync_state (
	target_key TEXT PRIMARY KEY,
	lifecycle TEXT NOT NULL DEFAULT 'idle',
	last_enqueued_seq INTEGER NOT NULL DEFAULT 0,
	last_acked_seq INTEGER NOT NULL DEFAULT 0,
	last_pulled_seq INTEGER NOT NULL DEFAULT 0,
	backoff_until TEXT,
	updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS sync_chunks (
	target_key TEXT NOT NULL DEFAULT 'local',
	chunk_id TEXT NOT NULL,
	imported_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (target_key, chunk_id)
);

CREATE TABLE IF NOT EXISTS sync_delete_tombstones (
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	session_id TEXT,
	project TEXT NOT NULL DEFAULT '',
	deleted_at TEXT NOT NULL DEFAULT (datetime('now')),
	PRIMARY KEY (entity, entity_key)
);

CREATE TABLE IF NOT EXISTS sync_delete_tombstones_remote_floors (
	target_key TEXT NOT NULL,
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	last_mutation_seq INTEGER NOT NULL,
	PRIMARY KEY (target_key, entity, entity_key)
);

CREATE TABLE IF NOT EXISTS sync_apply_deferred (
	sync_id TEXT PRIMARY KEY,
	entity TEXT NOT NULL,
	payload TEXT NOT NULL,
	entity_key TEXT NOT NULL DEFAULT '',
	project TEXT NOT NULL DEFAULT '',
	reason_code TEXT NOT NULL DEFAULT '',
	apply_status TEXT NOT NULL DEFAULT 'deferred',
	retry_count INTEGER NOT NULL DEFAULT 0,
	first_seen_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_sad_status_seen ON sync_apply_deferred(apply_status, first_seen_at);

CREATE TABLE IF NOT EXISTS sync_enrolled_projects (
	project TEXT PRIMARY KEY,
	enrolled_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Replay-safe save idempotency ledger (spec 002-schema §3.3). The FK is
-- ON DELETE SET NULL: a hard-deleted observation leaves its ledger row as a
-- tombstone (Task 6's 410 path relies on it).
CREATE TABLE IF NOT EXISTS observation_save_operations (
	operation_id TEXT NOT NULL PRIMARY KEY,
	fingerprint TEXT NOT NULL,
	observation_id INTEGER,
	created_at TEXT NOT NULL DEFAULT (datetime('now')),
	FOREIGN KEY (observation_id) REFERENCES observations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_obs_save_op_created ON observation_save_operations(created_at);

-- ── FTS5 trigram retrieval backbone (spec 001-storage §3) ──────────────────
-- External-content tables so the FTS rowid maps to the base row; the four
-- triggers each keep the index in step with INSERT/UPDATE/DELETE.
-- Verbatim FTS definitions from Engram store.go:3125-3191.

CREATE VIRTUAL TABLE observations_fts USING fts5(
	title,
	content,
	tool_name,
	type,
	project,
	topic_key,
	tokenize='trigram',
	content='observations',
	content_rowid='id'
);

CREATE TRIGGER obs_fts_insert AFTER INSERT ON observations
WHEN new.deleted_at IS NULL BEGIN
	INSERT INTO observations_fts(rowid, title, content, tool_name, type, project, topic_key)
	VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project, new.topic_key);
END;

CREATE TRIGGER obs_fts_delete BEFORE DELETE ON observations
WHEN old.deleted_at IS NULL BEGIN
	DELETE FROM observations_fts WHERE rowid = old.id;
END;

CREATE TRIGGER obs_fts_update BEFORE UPDATE ON observations
WHEN old.deleted_at IS NULL BEGIN
	DELETE FROM observations_fts WHERE rowid = old.id;
END;

CREATE TRIGGER obs_fts_update_insert AFTER UPDATE ON observations
WHEN new.deleted_at IS NULL BEGIN
	INSERT INTO observations_fts(rowid, title, content, tool_name, type, project, topic_key)
	VALUES (new.id, new.title, new.content, new.tool_name, new.type, new.project, new.topic_key);
END;

CREATE VIRTUAL TABLE prompts_fts USING fts5(
	content,
	project,
	tokenize='trigram',
	content='user_prompts',
	content_rowid='id'
);

CREATE TRIGGER prompt_fts_insert AFTER INSERT ON user_prompts BEGIN
	INSERT INTO prompts_fts(rowid, content, project)
	VALUES (new.id, new.content, new.project);
END;

CREATE TRIGGER prompt_fts_delete BEFORE DELETE ON user_prompts BEGIN
	DELETE FROM prompts_fts WHERE rowid = old.id;
END;

CREATE TRIGGER prompt_fts_update BEFORE UPDATE ON user_prompts BEGIN
	DELETE FROM prompts_fts WHERE rowid = old.id;
END;

CREATE TRIGGER prompt_fts_update_insert AFTER UPDATE ON user_prompts BEGIN
	INSERT INTO prompts_fts(rowid, content, project)
	VALUES (new.id, new.content, new.project);
END;
