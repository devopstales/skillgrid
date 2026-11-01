-- 001_schema.sql — the full spec 002-schema Postgres render (fresh schema
-- applies this once; later migrations are additive). One schema per project
-- (schema_<Canonicalize(project)>); every statement runs inside it.

CREATE TABLE IF NOT EXISTS index_meta (
	key TEXT PRIMARY KEY,
	schema_version INTEGER NOT NULL DEFAULT 0
);

-- ── Memory domain (spec 002-schema §2) ──────────────────────────────────────

CREATE TABLE IF NOT EXISTS sessions (
	id TEXT PRIMARY KEY,
	sync_id TEXT,
	project TEXT NOT NULL DEFAULT '',
	ownership_mode TEXT NOT NULL DEFAULT 'shared',
	directory TEXT NOT NULL DEFAULT '',
	title TEXT,
	started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	ended_at TIMESTAMPTZ,
	summary TEXT,
	runtime_lease_expires_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ,
	local_creation_project TEXT
);

CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project);

-- Local-only idempotent finalize retry journal; never synced or exported.
CREATE TABLE IF NOT EXISTS session_finalizations (
	session_id TEXT PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
	summary_id INTEGER NOT NULL,
	summary_sync_id TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS observations (
	id BIGSERIAL PRIMARY KEY,
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
	last_seen_at TIMESTAMPTZ,
	pinned BOOLEAN NOT NULL DEFAULT FALSE,
	review_after TIMESTAMPTZ,
	superseded_by_sync_id TEXT,
	expires_at TIMESTAMPTZ,
	embedding vector(768),
	embedding_model TEXT,
	embedding_created_at TIMESTAMPTZ,
	importance DOUBLE PRECISION NOT NULL DEFAULT 0.5,
	retrieval_usage INTEGER NOT NULL DEFAULT 0,
	provenance JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	deleted_at TIMESTAMPTZ
);

-- spec 002-schema §2.2: the two dedupe/topic indexes.
CREATE INDEX IF NOT EXISTS idx_obs_dedupe ON observations(normalized_hash, project, scope, type, title);
CREATE INDEX IF NOT EXISTS idx_obs_topic ON observations(project, scope, topic_key);

-- Vector index parity (S-1): pgvector HNSW with cosine ops — the PG leg of
-- the "HNSW/IVFFlat index" (spec 001-storage §4; the SQLite leg's vec0 is
-- the equivalent). A sequential scan would be a silent query-surface divergence.
CREATE INDEX IF NOT EXISTS idx_obs_embedding_hnsw ON observations USING hnsw (embedding vector_cosine_ops);

-- FTS (spec 001-storage §3): a tsvector column maintained by triggers. The
-- weight classes mirror the FTS5 column weights (title 5.0 / content 1.0 /
-- topic_key 3.0): title → A (heaviest), topic_key → B, content → C, the
-- remaining FTS5 columns (tool_name, type, project) are not indexed —
-- they are filter fields, not recall fields.
ALTER TABLE observations ADD COLUMN IF NOT EXISTS tsv tsvector;

CREATE INDEX IF NOT EXISTS idx_obs_tsv_gin ON observations USING gin (tsv);

CREATE OR REPLACE FUNCTION set_obs_tsv() RETURNS trigger AS $$
BEGIN
	NEW.tsv := setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A')
		|| setweight(to_tsvector('simple', COALESCE(NEW.topic_key, '')), 'B')
		|| setweight(to_tsvector('simple', COALESCE(NEW.content, '')), 'C');
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS obs_tsv_maintain ON observations;
CREATE TRIGGER obs_tsv_maintain BEFORE INSERT OR UPDATE ON observations
FOR EACH ROW EXECUTE FUNCTION set_obs_tsv();

CREATE TABLE IF NOT EXISTS observation_versions (
	id BIGSERIAL PRIMARY KEY,
	observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
	version INTEGER NOT NULL,
	title TEXT,
	content TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	UNIQUE(observation_id, version)
);

CREATE TABLE IF NOT EXISTS user_prompts (
	id BIGSERIAL PRIMARY KEY,
	sync_id TEXT UNIQUE,
	session_id TEXT NOT NULL REFERENCES sessions(id),
	content TEXT NOT NULL,
	project TEXT,
	source_inbox_id TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	tsv tsvector
);

CREATE INDEX IF NOT EXISTS idx_prompts_session ON user_prompts(session_id);
CREATE INDEX IF NOT EXISTS idx_prompts_project ON user_prompts(project);
CREATE INDEX IF NOT EXISTS idx_prompts_tsv_gin ON user_prompts USING gin (tsv);

CREATE OR REPLACE FUNCTION set_prompt_tsv() RETURNS trigger AS $$
BEGIN
	NEW.tsv := to_tsvector('simple', COALESCE(NEW.content, ''))
		|| to_tsvector('simple', COALESCE(NEW.project, ''));
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS prompt_tsv_maintain ON user_prompts;
CREATE TRIGGER prompt_tsv_maintain BEFORE INSERT OR UPDATE ON user_prompts
FOR EACH ROW EXECUTE FUNCTION set_prompt_tsv();

-- Deleted prompts (local); sync_id is the identity key.
CREATE TABLE IF NOT EXISTS prompt_tombstones (
	sync_id TEXT PRIMARY KEY,
	session_id TEXT,
	project TEXT,
	deleted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- spec 002-schema §2.5: keys are sync_id (TEXT), cross-machine portable.
CREATE TABLE IF NOT EXISTS memory_relations (
	id BIGSERIAL PRIMARY KEY,
	sync_id TEXT UNIQUE,
	source_id TEXT NOT NULL,
	target_id TEXT NOT NULL,
	relation TEXT NOT NULL DEFAULT 'pending',
	reason TEXT,
	evidence TEXT,
	confidence DOUBLE PRECISION,
	judgment_status TEXT NOT NULL DEFAULT 'pending',
	marked_by_actor TEXT,
	marked_by_kind TEXT,
	marked_by_model TEXT,
	session_id TEXT,
	superseded_at TIMESTAMPTZ,
	superseded_by_relation_id INTEGER REFERENCES memory_relations(id) ON DELETE SET NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_memrel_source ON memory_relations(source_id, judgment_status);
CREATE INDEX IF NOT EXISTS idx_memrel_target ON memory_relations(target_id, judgment_status);
CREATE INDEX IF NOT EXISTS idx_memrel_supersede ON memory_relations(superseded_by_relation_id);

-- ── Sync domain (spec 002-schema §3) ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sync_mutations (
	seq BIGSERIAL PRIMARY KEY,
	target_key TEXT NOT NULL,
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	op TEXT NOT NULL,
	payload TEXT NOT NULL,
	source TEXT NOT NULL DEFAULT 'local',
	project TEXT,
	occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	acked_at TIMESTAMPTZ,
	disposition TEXT
);

CREATE INDEX IF NOT EXISTS idx_sync_mutations_project ON sync_mutations(project);

CREATE TABLE IF NOT EXISTS sync_state (
	target_key TEXT PRIMARY KEY,
	lifecycle TEXT NOT NULL DEFAULT 'idle',
	last_enqueued_seq BIGINT NOT NULL DEFAULT 0,
	last_acked_seq BIGINT NOT NULL DEFAULT 0,
	last_pulled_seq BIGINT NOT NULL DEFAULT 0,
	backoff_until TIMESTAMPTZ,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sync_chunks (
	target_key TEXT NOT NULL DEFAULT 'local',
	chunk_id TEXT NOT NULL,
	imported_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (target_key, chunk_id)
);

CREATE TABLE IF NOT EXISTS sync_delete_tombstones (
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	session_id TEXT,
	project TEXT NOT NULL DEFAULT '',
	deleted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	PRIMARY KEY (entity, entity_key)
);

CREATE TABLE IF NOT EXISTS sync_delete_tombstones_remote_floors (
	target_key TEXT NOT NULL,
	entity TEXT NOT NULL,
	entity_key TEXT NOT NULL,
	last_mutation_seq BIGINT NOT NULL,
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
	first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sad_status_seen ON sync_apply_deferred(apply_status, first_seen_at);

CREATE TABLE IF NOT EXISTS sync_enrolled_projects (
	project TEXT PRIMARY KEY,
	enrolled_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Replay-safe save idempotency ledger (spec 002-schema §3.3). The FK is
-- ON DELETE SET NULL: a hard-deleted observation leaves its ledger row as a
-- tombstone (Task 6's 410 path relies on it).
CREATE TABLE IF NOT EXISTS observation_save_operations (
	operation_id TEXT NOT NULL PRIMARY KEY,
	fingerprint TEXT NOT NULL,
	observation_id INTEGER,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	FOREIGN KEY (observation_id) REFERENCES observations(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_obs_save_op_created ON observation_save_operations(created_at);
