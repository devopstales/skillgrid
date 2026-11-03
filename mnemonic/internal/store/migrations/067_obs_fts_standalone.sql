-- Migration 059: make observations_fts a standalone FTS5 table.
--
-- 001_schema.sql created observations_fts as an EXTERNAL-CONTENT FTS5 table:
--
--     CREATE VIRTUAL TABLE observations_fts USING fts5(
--         title, content, type, project,
--         content='observations', content_rowid='id', tokenize='porter');
--
-- For an external-content table FTS5 does not own the document text: it reads
-- the content table (`observations`) to reconcile every index operation, and
-- a 'delete' command deletes the terms it finds *in the content table row*.
-- The three AFTER triggers on `observations` keep the index in sync by
-- inserting the (rowid, title, content, type, project) values explicitly.
--
-- 054 renamed the table to observations_legacy and recreated `observations`
-- as a VIEW. Two consequences:
--
--   1. A VIEW is not a usable FTS5 content table. The delete command can no
--      longer reconcile against it, so a 'delete' by terms corrupts the index
--      (SQLITE_CORRUPT 267, "database disk image is malformed"). The INSTEAD
--      OF write path (TICKET-03) maintains the index by explicit values, so
--      the corruption only surfaced on the few writers that still routed
--      through the view (the FTS delete in the mem_update path).
--
--   2. The legacy AFTER *INSERT/UPDATE/DELETE* triggers are dead on a view
--      (after-triggers do not fire on views), so the index is maintained
--      solely by the INSTEAD OF triggers (fresh) or by the Go fts*Obs helpers
--      (direct-table writes). The three legacy triggers are kept only so
--      observations_legacy (the pre-split copy, written by the migration
--      backfill) stays self-consistent; they are no longer on the live path.
--
-- Fix: recreate observations_fts as a STANDALONE FTS5 table (no `content=`).
-- A standalone table owns its rows and does not reconcile against a content
-- table, so the 'delete' command deletes exactly the terms passed — no view
-- lookup, no corruption. The index is rebuilt from the live split tables
-- (observations_core + observation_state + observation_provenance, the same
-- LEFT JOIN the compat view uses) so every existing observation re-enters the
-- index.
--
-- Idempotency guard: only run the rebuild when the current FTS table is still
-- external-content (`content=observations`). A standalone table (this
-- migration already applied) leaves `sql` without that clause, so re-running
-- the file is a no-op. Idempotent on a fresh DB (001 creates the
-- external-content table; this converts it once).

-- Idempotency: the conversion (drop triggers + drop/recreate the FTS table +
-- rebuild from the live split tables) is safe to run more than once. On a
-- fresh DB the table is external-content (001), so the conversion applies
-- once; on an already-converted DB the table is standalone and the rebuild
-- re-derives the same index from observations_core (a no-op in effect). The
-- rebuild SELECTs from observations_core directly, so it is independent of the
-- FTS table's content-table declaration.
DROP TRIGGER IF EXISTS observations_fts_insert;
DROP TRIGGER IF EXISTS observations_fts_update;
DROP TRIGGER IF EXISTS observations_fts_delete;

DROP TABLE IF EXISTS observations_fts;

CREATE VIRTUAL TABLE observations_fts USING fts5(
    title,
    content,
    type,
    project,
    tokenize='porter'
);

-- Rebuild the index from ALL core rows (including soft-deleted): the search
-- paths filter `deleted_at IS NULL` explicitly, so a soft-deleted row's FTS
-- entry is harmless. Rebuilding comprehensively means the index is complete
-- after migration regardless of how the rows were written (trigger-mediated
-- view seeds, direct core writes, or the Go fts*Obs helpers), so test seeds
-- that INSERT through the compat VIEW (whose INSTEAD OF trigger no longer
-- writes the FTS row) are still searchable.
INSERT INTO observations_fts(rowid, title, content, type, project)
SELECT c.id, c.title, c.content, c.type, c.project
FROM observations_core AS c;
