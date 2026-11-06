-- Migration 068: add human-gate status lifecycle to ctx_traffic_memories.
--
-- T13 adds a human gate to the traffic learner: the learner only proposes
-- candidate memories, and ctx learn apply/revoke promotes or demotes them.
--   status     TEXT NOT NULL DEFAULT 'candidate'
--                 candidate | applied | revoked. A row stays candidate until a
--                 human applies it; revocation moves it to revoked so the
--                 learner stops re-suggesting it.
--   applied_at INTEGER
--                 unix millis of the last apply; NULL until first applied.
--
-- The CREATE is a no-op on a pre-068 DB (061 already created the table) and is
-- the authoritative definition for a fresh DB. The two ALTERs add the columns
-- in place for pre-068 databases; on a fresh DB the CREATE already carried
-- them, so the ALTERs are the no-op half of the pair.
CREATE TABLE IF NOT EXISTS ctx_traffic_memories (
    memory_id    TEXT PRIMARY KEY,
    session_id   TEXT NOT NULL,
    category     TEXT NOT NULL,
    memory_text  TEXT NOT NULL,
    evidence_json TEXT NOT NULL DEFAULT '{}',
    confidence   REAL NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    status       TEXT NOT NULL DEFAULT 'candidate',
    applied_at   INTEGER
);

ALTER TABLE ctx_traffic_memories ADD COLUMN status TEXT NOT NULL DEFAULT 'candidate';
ALTER TABLE ctx_traffic_memories ADD COLUMN applied_at INTEGER;
