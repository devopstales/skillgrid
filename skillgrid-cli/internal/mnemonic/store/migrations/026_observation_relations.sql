-- 026: typed @relation edges between observations (change 014, step 14).
--
-- Additive: a NEW table in the same per-project store. It is distinct from
-- memory_relations (006, mem_judge verdicts) and from the codeindex edges
-- table (011/021, symbol-level graph edges): observation_relations is a
-- typed, confidence-weighted relation between OBSERVATION ids.
--
--   source_id      — the observation that asserts the relation.
--   target_id      — the observation the relation points at.
--   relation_type  — one of the canonical set: mentions, depends_on,
--                    contradicts, supports, references.
--   confidence     — 0.0..1.0 weight on the assertion.
--
-- The composite primary key makes each (source, target, type) triple unique,
-- so re-asserting the same triple upserts confidence instead of duplicating.

CREATE TABLE IF NOT EXISTS observation_relations (
    source_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    relation_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type)
);

-- Bidirectional lookup support (mem relations <id> queries source OR target).
CREATE INDEX IF NOT EXISTS idx_obs_rel_source
    ON observation_relations (source_id);
CREATE INDEX IF NOT EXISTS idx_obs_rel_target
    ON observation_relations (target_id);
