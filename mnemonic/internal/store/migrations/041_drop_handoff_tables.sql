-- 041: drop the Hub/relay tables (ONE-WAY).
--
-- Drops the five Hub/relay tables created by 019 (session_handoffs,
-- session_archives) and 039 (change_snapshots, checkpoints, handoff_refs)
-- plus their six indexes. The session-events layer from 040
-- (sessions columns + session_events) is now the index, and the git log
-- (with its [skillgrid-context] blocks) is the durable record — Hub UI
-- history is reconstructable from `git log` only.
--
-- ONE-WAY: re-applying earlier migrations cannot restore dropped rows.
-- DROP TABLE IF EXISTS / DROP INDEX IF EXISTS so the migration is
-- idempotent and safe on databases that never had these tables.

DROP TABLE IF EXISTS change_snapshots;
DROP TABLE IF EXISTS checkpoints;
DROP TABLE IF EXISTS handoff_refs;
DROP TABLE IF EXISTS session_handoffs;
DROP TABLE IF EXISTS session_archives;

DROP INDEX IF EXISTS idx_snapshots_project_time;
DROP INDEX IF EXISTS idx_checkpoints_project_status;
DROP INDEX IF EXISTS idx_handoff_refs_project;
DROP INDEX IF EXISTS idx_handoffs_source;
DROP INDEX IF EXISTS idx_handoffs_status;
DROP INDEX IF EXISTS idx_archives_session;
