-- 070_add_knowledge_class: nullable durability dimension on observations
-- (ADR-0038). One-way: the runner applies it once (tracked in index_meta);
-- no down-migration. Existing rows stay NULL until the re-runnable
-- BackfillKnowledgeClass (internal/memory) labels them from the type column.
--
-- The dimension lives on observations_core (the live table behind the
-- observations compat view after 062_obs_split); the view exposes it in
-- legacy read order so every unqualified `SELECT ... FROM observations`
-- (obsSelectCols) keeps resolving.
--
-- Index: the brief asks for (project, knowledge_class). The split index
-- shapes (cf. idx_obs_memory_type in 062) key on the owning table's own
-- columns, and the class predicate pairs with a project filter in the same
-- query, so the class-leading key serves that access path.

ALTER TABLE observations_core ADD COLUMN knowledge_class TEXT;

-- Additive view rebuild: the same 40 columns as the 062 view, with
-- knowledge_class appended (the hot-read scan path, obsSplitSelectCols,
-- keeps its 31-column contract and labels the class separately). The runner
-- executes per-statement (execMigrationFile), so the drop+create window is
-- one statement inside a single migration run — safe for this local-first
-- store.
DROP VIEW observations;

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
    tm.valid_at, tm.invalid_at, tm.superseded_by,
    c.knowledge_class
FROM observations_core AS c
LEFT JOIN observation_state AS st ON st.observation_id = c.id
LEFT JOIN observation_provenance AS pr ON pr.observation_id = c.id
LEFT JOIN observation_stats AS sa ON sa.observation_id = c.id
LEFT JOIN observation_temporal AS tm ON tm.observation_id = c.id
LEFT JOIN observation_vectors AS vc ON vc.observation_id = c.id;

CREATE INDEX IF NOT EXISTS idx_obs_knowledge_class
    ON observations_core (knowledge_class, project)
    WHERE knowledge_class IS NOT NULL;

-- Compat write fan-out: the 062 INSTEAD OF triggers reference NEW.<col> for
-- every view column, so the rebuilt 41-column view breaks them until they
-- name knowledge_class too (a trigger that omits a view column still fails
-- when the view is queried for that column's NEW value). Rebuild all three
-- with the new column wired into core; the FTS half is untouched (the class
-- is not an FTS column).
DROP TRIGGER IF EXISTS observations_view_insert;
DROP TRIGGER IF EXISTS observations_view_update;
DROP TRIGGER IF EXISTS observations_view_delete;

CREATE TRIGGER IF NOT EXISTS observations_view_insert
INSTEAD OF INSERT ON observations BEGIN
    INSERT INTO observations_core
        (id, session_id, type, title, content, project, scope, topic_key,
         normalized_hash, revision_count, created_at, updated_at, deleted_at,
         knowledge_class)
        VALUES
        (NEW.id, NEW.session_id, NEW.type, NEW.title, NEW.content,
         NEW.project, NEW.scope, NEW.topic_key, NEW.normalized_hash,
         COALESCE(NEW.revision_count, 0), NEW.created_at, NEW.updated_at,
         NEW.deleted_at, NEW.knowledge_class);
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
        updated_at = NEW.updated_at, deleted_at = NEW.deleted_at,
        knowledge_class = NEW.knowledge_class
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
    UPDATE observations_fts SET
        title = NEW.title, content = NEW.content, type = NEW.type,
        project = NEW.project
        WHERE rowid = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS observations_view_delete
INSTEAD OF DELETE ON observations BEGIN
    DELETE FROM observation_vectors WHERE observation_id = OLD.id;
    DELETE FROM observation_temporal WHERE observation_id = OLD.id;
    DELETE FROM observation_stats WHERE observation_id = OLD.id;
    DELETE FROM observation_provenance WHERE observation_id = OLD.id;
    DELETE FROM observation_state WHERE observation_id = OLD.id;
    DELETE FROM observations_core WHERE id = OLD.id;
    DELETE FROM observations_fts WHERE rowid = OLD.id;
    INSERT INTO observations_fts(rowid, title, content, type, project)
        SELECT c.id, c.title, c.content, c.type, c.project
        FROM observations_core AS c
        WHERE c.id != OLD.id
          AND NOT EXISTS (SELECT 1 FROM observations_fts f WHERE f.rowid = c.id);
END;
