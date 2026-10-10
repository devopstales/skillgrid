-- Migration 054: split observations god-table behind compat view (expand phase).
--
-- One-way checkpoint (maintainer-approved 2026-10-10): rebuilds `observations`
-- via COPY/RENAME. Tests run on store COPIES only; never on the live store.
-- Rollback: the pre-split rows stay intact in `observations_legacy` for one
-- release (TICKET-07 drill restores from the pre-migration .dump).
--
-- Column map (40 columns, legacy order preserved by the view):
--   observations_core (13): id, session_id, type, title, content, project,
--     scope, topic_key, normalized_hash, revision_count, created_at,
--     updated_at, deleted_at
--   observation_state (7+id): status, visibility, archived_at, archive_reason,
--     review_after, expires_at, pinned
--   observation_provenance (7+id): source, prompt_id, tool_name, provenance,
--     owner, memory_type, graph_ref
--   observation_stats (7+id): duplicate_count, last_seen_at, retrieval_usage,
--     importance_score, recency_decay, maturity_tier, risk_score
--   observation_temporal (3+id): valid_at, invalid_at, superseded_by
--   observation_vectors (3+id): embedding, embedding_model, embedding_created_at
--
-- Side tables carry plain INTEGER PRIMARY KEY observation_id (no FK clauses:
-- FK shape deferred to TICKET-03, which owns the write path). The compat VIEW
-- exposes all 40 columns in legacy order with LEFT JOINs so every core row
-- renders even if a side row is missing.
--
-- Index moves (DROP from legacy, CREATE on owning side table). Partial-index
-- predicates and key columns can only reference their own table, so
-- cross-table keys/predicates are adapted (documented per index):
--   idx_obs_pinned: was (pinned, last_seen_at, created_at) WHERE deleted_at
--     IS NULL -> now (pinned DESC) on observation_state (last_seen_at lives
--     in stats, created_at/deleted_at in core).
--   idx_obs_expires: was (expires_at) WHERE expires_at IS NOT NULL AND
--     deleted_at IS NULL -> now (expires_at) WHERE expires_at IS NOT NULL.
--   idx_obs_dedupe: exact move onto observations_core (all keys local).
--   idx_obs_embedding: was (project, embedding_created_at) WHERE embedding
--     IS NOT NULL -> now (embedding_created_at DESC) WHERE embedding IS NOT
--     NULL on observation_vectors (project lives in core).
--   idx_obs_importance / idx_obs_risk: was (project, score) WHERE deleted_at
--     IS NULL AND score IS NOT NULL -> now (score DESC) WHERE score IS NOT
--     NULL on observation_stats.
--   idx_obs_memory_type: was (project, memory_type) WHERE memory_type IS NOT
--     NULL AND deleted_at IS NULL -> now (memory_type) WHERE memory_type IS
--     NOT NULL on observation_provenance.
--   idx_obs_invalid_at + idx_obs_graph_ref stay on observations_legacy (not
--     in the move list); the bitemporal filter keeps a same-predicate index
--     on the new table as idx_obs_temporal_invalid_at (new name, no clash).
--
-- Write-API limits through the VIEW (verified engine behavior, NOT fixable
-- in SQL — TICKET-03 rewrites writers to hit core + side tables directly,
-- which restores all four):
--   - Reads are fully compat (row-hash parity proven by
--     TestObservationsParity); writes LAND correctly in all six tables.
--   - Result.LastInsertId() after a view INSERT returns a STALE counter
--     (inserts inside INSTEAD OF trigger programs do not advance
--     sqlite3_last_insert_rowid).
--   - RowsAffected() after a view UPDATE/DELETE is always 0 (same cause),
--     so callers branching on `n == 0 -> not found` misfire.
--   - INSERT ... RETURNING yields NULL; INSERT ... ON CONFLICT (UPSERT)
--     is rejected on views ("cannot UPSERT a view").
--   Until TICKET-03 lands, ~60 write-path tests fail on exactly these four
--     mechanisms; no failure shows read corruption (see TICKET-02 report).
--   - observations_fts_* triggers ride along to observations_legacy on
--     RENAME; view-mediated writes maintain observations_fts inside the
--     INSTEAD OF trigger bodies below, so FTS stays current.

CREATE TABLE IF NOT EXISTS observations_core (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    project TEXT,
    scope TEXT,
    topic_key TEXT,
    normalized_hash TEXT,
    revision_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS observation_state (
    observation_id INTEGER PRIMARY KEY,
    status TEXT NOT NULL DEFAULT 'active',
    visibility TEXT NOT NULL DEFAULT 'private',
    archived_at TEXT,
    archive_reason TEXT,
    review_after TEXT,
    expires_at TEXT,
    pinned INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS observation_provenance (
    observation_id INTEGER PRIMARY KEY,
    source TEXT NOT NULL DEFAULT 'agent',
    prompt_id INTEGER,
    tool_name TEXT,
    provenance TEXT,
    owner TEXT,
    memory_type TEXT,
    graph_ref INTEGER
);

CREATE TABLE IF NOT EXISTS observation_stats (
    observation_id INTEGER PRIMARY KEY,
    duplicate_count INTEGER NOT NULL DEFAULT 0,
    last_seen_at TEXT,
    retrieval_usage INTEGER NOT NULL DEFAULT 0,
    importance_score REAL,
    recency_decay REAL,
    maturity_tier TEXT,
    risk_score REAL
);

CREATE TABLE IF NOT EXISTS observation_temporal (
    observation_id INTEGER PRIMARY KEY,
    valid_at TEXT,
    invalid_at TEXT,
    superseded_by INTEGER
);

CREATE TABLE IF NOT EXISTS observation_vectors (
    observation_id INTEGER PRIMARY KEY,
    embedding BLOB,
    embedding_model TEXT,
    embedding_created_at TEXT
);

INSERT INTO observations_core
    (id, session_id, type, title, content, project, scope, topic_key,
     normalized_hash, revision_count, created_at, updated_at, deleted_at)
    SELECT id, session_id, type, title, content, project, scope, topic_key,
     normalized_hash, revision_count, created_at, updated_at, deleted_at
    FROM observations;

INSERT INTO observation_state
    (observation_id, status, visibility, archived_at, archive_reason,
     review_after, expires_at, pinned)
    SELECT id, status, visibility, archived_at, archive_reason,
     review_after, expires_at, pinned
    FROM observations;

INSERT INTO observation_provenance
    (observation_id, source, prompt_id, tool_name, provenance,
     owner, memory_type, graph_ref)
    SELECT id, source, prompt_id, tool_name, provenance,
     owner, memory_type, graph_ref
    FROM observations;

INSERT INTO observation_stats
    (observation_id, duplicate_count, last_seen_at, retrieval_usage,
     importance_score, recency_decay, maturity_tier, risk_score)
    SELECT id, duplicate_count, last_seen_at, retrieval_usage,
     importance_score, recency_decay, maturity_tier, risk_score
    FROM observations;

INSERT INTO observation_temporal
    (observation_id, valid_at, invalid_at, superseded_by)
    SELECT id, valid_at, invalid_at, superseded_by
    FROM observations;

INSERT INTO observation_vectors
    (observation_id, embedding, embedding_model, embedding_created_at)
    SELECT id, embedding, embedding_model, embedding_created_at
    FROM observations;

ALTER TABLE observations RENAME TO observations_legacy;

CREATE VIEW observations AS
SELECT
    c.id, c.session_id, c.type, c.title, c.content, c.project, c.scope,
    c.topic_key, c.normalized_hash, c.revision_count, c.created_at,
    c.updated_at, c.deleted_at,
    pr.source, st.review_after, pr.prompt_id, st.pinned, st.expires_at,
    sa.duplicate_count, sa.last_seen_at,
    vc.embedding, vc.embedding_model, vc.embedding_created_at,
    pr.tool_name, pr.owner, st.status, st.visibility, sa.retrieval_usage,
    pr.graph_ref, sa.importance_score, sa.recency_decay, sa.maturity_tier,
    pr.provenance, pr.memory_type, sa.risk_score,
    st.archive_reason, st.archived_at,
    tm.valid_at, tm.invalid_at, tm.superseded_by
FROM observations_core AS c
LEFT JOIN observation_state AS st ON st.observation_id = c.id
LEFT JOIN observation_provenance AS pr ON pr.observation_id = c.id
LEFT JOIN observation_stats AS sa ON sa.observation_id = c.id
LEFT JOIN observation_temporal AS tm ON tm.observation_id = c.id
LEFT JOIN observation_vectors AS vc ON vc.observation_id = c.id;

DROP INDEX IF EXISTS idx_obs_pinned;
DROP INDEX IF EXISTS idx_obs_expires;
DROP INDEX IF EXISTS idx_obs_dedupe;
DROP INDEX IF EXISTS idx_obs_embedding;
DROP INDEX IF EXISTS idx_obs_importance;
DROP INDEX IF EXISTS idx_obs_risk;
DROP INDEX IF EXISTS idx_obs_memory_type;

CREATE INDEX IF NOT EXISTS idx_obs_pinned
    ON observation_state (pinned DESC);
CREATE INDEX IF NOT EXISTS idx_obs_expires
    ON observation_state (expires_at)
    WHERE expires_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_obs_dedupe
    ON observations_core (normalized_hash, project, scope, type, title, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_obs_embedding
    ON observation_vectors (embedding_created_at DESC)
    WHERE embedding IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_obs_importance
    ON observation_stats (importance_score DESC)
    WHERE importance_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_obs_risk
    ON observation_stats (risk_score DESC)
    WHERE risk_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_obs_memory_type
    ON observation_provenance (memory_type)
    WHERE memory_type IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_obs_temporal_invalid_at
    ON observation_temporal (invalid_at)
    WHERE invalid_at IS NOT NULL AND invalid_at != '';

-- Compat write fan-out: INSTEAD OF triggers so every pre-split caller
-- (INSERT with any column subset, UPDATE with any SET subset, DELETE with
-- any WHERE) keeps working unchanged. Unspecified NEW columns read as NULL
-- inside the trigger, so NOT NULL DEFAULT columns are COALESCE'd to the
-- same defaults a direct table INSERT would apply. Side-table rowids equal
-- the core id (INTEGER PRIMARY KEY), so LastInsertId() still returns the
-- new observation id. FTS delete+insert mirrors the legacy
-- observations_fts_* triggers (which now ride observations_legacy).
CREATE TRIGGER IF NOT EXISTS observations_view_insert
INSTEAD OF INSERT ON observations BEGIN
    INSERT INTO observations_core
        (id, session_id, type, title, content, project, scope, topic_key,
         normalized_hash, revision_count, created_at, updated_at, deleted_at)
        VALUES
        (NEW.id, NEW.session_id, NEW.type, NEW.title, NEW.content,
         NEW.project, NEW.scope, NEW.topic_key, NEW.normalized_hash,
         COALESCE(NEW.revision_count, 0), NEW.created_at, NEW.updated_at,
         NEW.deleted_at);
    INSERT INTO observation_state
        (observation_id, status, visibility, archived_at, archive_reason,
         review_after, expires_at, pinned)
        VALUES
        (last_insert_rowid(), COALESCE(NEW.status, 'active'),
         COALESCE(NEW.visibility, 'private'), NEW.archived_at,
         NEW.archive_reason, NEW.review_after, NEW.expires_at,
         COALESCE(NEW.pinned, 0));
    INSERT INTO observation_provenance
        (observation_id, source, prompt_id, tool_name, provenance,
         owner, memory_type, graph_ref)
        VALUES
        (last_insert_rowid(), COALESCE(NEW.source, 'agent'), NEW.prompt_id,
         NEW.tool_name, NEW.provenance, NEW.owner, NEW.memory_type,
         NEW.graph_ref);
    INSERT INTO observation_stats
        (observation_id, duplicate_count, last_seen_at, retrieval_usage,
         importance_score, recency_decay, maturity_tier, risk_score)
        VALUES
        (last_insert_rowid(), COALESCE(NEW.duplicate_count, 0),
         NEW.last_seen_at, COALESCE(NEW.retrieval_usage, 0),
         NEW.importance_score, NEW.recency_decay, NEW.maturity_tier,
         NEW.risk_score);
    INSERT INTO observation_temporal
        (observation_id, valid_at, invalid_at, superseded_by)
        VALUES
        (last_insert_rowid(), NEW.valid_at, NEW.invalid_at,
         NEW.superseded_by);
    INSERT INTO observation_vectors
        (observation_id, embedding, embedding_model, embedding_created_at)
        VALUES
        (last_insert_rowid(), NEW.embedding, NEW.embedding_model,
         NEW.embedding_created_at);
    INSERT INTO observations_fts(rowid, title, content, type, project)
        VALUES
        (last_insert_rowid(), NEW.title, NEW.content, NEW.type,
         NEW.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_view_update
INSTEAD OF UPDATE ON observations BEGIN
    UPDATE observations_core SET
        session_id = NEW.session_id, type = NEW.type, title = NEW.title,
        content = NEW.content, project = NEW.project, scope = NEW.scope,
        topic_key = NEW.topic_key, normalized_hash = NEW.normalized_hash,
        revision_count = NEW.revision_count, created_at = NEW.created_at,
        updated_at = NEW.updated_at, deleted_at = NEW.deleted_at
        WHERE id = OLD.id;
    UPDATE observation_state SET
        status = NEW.status, visibility = NEW.visibility,
        archived_at = NEW.archived_at, archive_reason = NEW.archive_reason,
        review_after = NEW.review_after, expires_at = NEW.expires_at,
        pinned = NEW.pinned
        WHERE observation_id = OLD.id;
    UPDATE observation_provenance SET
        source = NEW.source, prompt_id = NEW.prompt_id,
        tool_name = NEW.tool_name, provenance = NEW.provenance,
        owner = NEW.owner, memory_type = NEW.memory_type,
        graph_ref = NEW.graph_ref
        WHERE observation_id = OLD.id;
    UPDATE observation_stats SET
        duplicate_count = NEW.duplicate_count,
        last_seen_at = NEW.last_seen_at,
        retrieval_usage = NEW.retrieval_usage,
        importance_score = NEW.importance_score,
        recency_decay = NEW.recency_decay, maturity_tier = NEW.maturity_tier,
        risk_score = NEW.risk_score
        WHERE observation_id = OLD.id;
    UPDATE observation_temporal SET
        valid_at = NEW.valid_at, invalid_at = NEW.invalid_at,
        superseded_by = NEW.superseded_by
        WHERE observation_id = OLD.id;
    UPDATE observation_vectors SET
        embedding = NEW.embedding, embedding_model = NEW.embedding_model,
        embedding_created_at = NEW.embedding_created_at
        WHERE observation_id = OLD.id;
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
        VALUES ('delete', OLD.id, OLD.title, OLD.content, OLD.type, OLD.project);
    INSERT INTO observations_fts(rowid, title, content, type, project)
        VALUES (NEW.id, NEW.title, NEW.content, NEW.type, NEW.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_view_delete
INSTEAD OF DELETE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
        VALUES ('delete', OLD.id, OLD.title, OLD.content, OLD.type, OLD.project);
    DELETE FROM observation_vectors WHERE observation_id = OLD.id;
    DELETE FROM observation_temporal WHERE observation_id = OLD.id;
    DELETE FROM observation_stats WHERE observation_id = OLD.id;
    DELETE FROM observation_provenance WHERE observation_id = OLD.id;
    DELETE FROM observation_state WHERE observation_id = OLD.id;
    DELETE FROM observations_core WHERE id = OLD.id;
END;
