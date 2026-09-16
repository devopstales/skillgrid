-- 028: fine-grained typed memory categories (change 014, step 18).
--
-- Additive: one new nullable TEXT column on observations. No existing schema
-- is touched; the existing `type` column (coarse taxonomy: learning, decision,
-- bug, session_log, ...) is left untouched. pre-028 rows read back with
-- memory_type NULL (not typed), so untyped saves keep working unchanged.
--
--   memory_type — one of the 9 typed categories (step 18):
--                  profile, preferences, entities, events, identity, soul,
--                  cases, trajectories, experiences.
--                NULL = the observation was not given a fine-grained type.
--                Validation (rejecting unknown values) happens in Go, not the
--                schema, so the column stays nullable and pre-existing rows
--                are never invalid.
--
-- The partial index lets `mem list --type <cat>` (the RecentWithType read
-- path) seek straight to typed rows without scanning the whole table. NULL
-- memory_type rows are excluded, matching "only typed observations".

ALTER TABLE observations ADD COLUMN memory_type TEXT;

CREATE INDEX IF NOT EXISTS idx_obs_memory_type
    ON observations (project, memory_type)
    WHERE memory_type IS NOT NULL AND deleted_at IS NULL;

-- Async two-phase session commit (step 18.3): compression_index is the per-
-- session "commit" counter the SYNC phase of SessionCommit increments. It is
-- additive (NOT NULL DEFAULT 0) so pre-028 sessions read back with 0 and the
-- commit path can advance it without a backfill.
ALTER TABLE sessions ADD COLUMN compression_index INTEGER NOT NULL DEFAULT 0;
