-- Migration 055: repoint observation FKs from the compat VIEW to observations_core.
--
-- 001_schema.sql declared observation_versions.observation_id and
-- acl_grants.observation_id as `REFERENCES observations(id)`. Before 054,
-- `observations` was a TABLE, so the FK resolved. 054 renamed the table to
-- observations_legacy and recreated `observations` as a VIEW; the FK column
-- still names `observations`, but a foreign key's referenced table must be a
-- table, not a view — so the FK lookup finds no matching table and every
-- insert into the child fails with 787 (FOREIGN KEY constraint failed).
--
-- `PRAGMA foreign_keys=OFF` cannot work around this: it only takes effect
-- when run outside any transaction, and these child inserts happen inside
-- the caller's transaction.
--
-- Fix: repoint both FKs at the real table, observations_core. SQLite has no
-- in-place "alter FK target" statement, so the two child tables are rebuilt
-- with the corrected FK clause. They are small (one row per observation
-- revision / grant), carry no other inbound FK edges, and the rebuild runs in
-- a single explicit transaction, so it is atomic. Idempotent on a fresh DB.
--
-- This is an additive schema repair for TICKET-03 (owned write path); it
-- touches no column layout and preserves all existing rows.

BEGIN;
CREATE TABLE observation_versions_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations_core(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE acl_grants_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations_core(id) ON DELETE CASCADE,
    grantee TEXT NOT NULL,
    grant_type TEXT NOT NULL DEFAULT 'agent',
    created_at TEXT NOT NULL
);
INSERT INTO observation_versions_new SELECT * FROM observation_versions;
INSERT INTO acl_grants_new SELECT * FROM acl_grants;
DROP TABLE observation_versions;
DROP TABLE acl_grants;
ALTER TABLE observation_versions_new RENAME TO observation_versions;
ALTER TABLE acl_grants_new RENAME TO acl_grants;
CREATE INDEX IF NOT EXISTS idx_obs_versions
    ON observation_versions (observation_id, revision);
CREATE INDEX IF NOT EXISTS idx_obs_grants
    ON acl_grants (observation_id, grantee);
COMMIT;
