-- 017: layered memory governance (change 013, step 01).
--
-- Additive governance on top of 005's observation store: every observation
-- becomes a governed asset with an owner, a version history (append, not
-- overwrite), a lifecycle status, a retrieval-usage counter, and a visibility
-- (private by default). All statements are CREATE TABLE IF NOT EXISTS /
-- ALTER TABLE ADD COLUMN so an existing database upgrades without a rewrite;
-- the 005 observations table is never dropped or rebuilt.
--
--   owner             — the creating user/agent identity. Set at save time;
--                       a new observation always has an owner.
--   status            — active | superseded | archived. Active by default;
--                       never inferred from content (explicit mutation only).
--   visibility        — private | team | restricted | agent. Private by
--                       default: a new observation is `private` until an
--                       explicit mem_share. `private` is invisible to other
--                       owners (the creating owner always sees its own).
--   retrieval_usage   — times the observation was returned by a search.
--                       Distinct from duplicate_count (re-saves).
--
-- observation_versions — append-only version history. A mem_update appends a
-- row (observation_id, revision, content, created) instead of overwriting;
-- the latest revision is the read path and revision_count advances.

ALTER TABLE observations ADD COLUMN owner TEXT;
ALTER TABLE observations ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE observations ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private';
ALTER TABLE observations ADD COLUMN retrieval_usage INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS observation_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- A restricted/agent observation is readable by owner + its explicit grants.
CREATE TABLE IF NOT EXISTS acl_grants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    grantee TEXT NOT NULL,
    grant_type TEXT NOT NULL DEFAULT 'agent',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_obs_versions
    ON observation_versions (observation_id, revision);
CREATE INDEX IF NOT EXISTS idx_obs_grants
    ON acl_grants (observation_id, grantee);
