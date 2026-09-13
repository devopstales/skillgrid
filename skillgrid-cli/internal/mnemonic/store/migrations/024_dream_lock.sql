-- 024: per-project distillation (dream) lock (change 014, step 12).
--
-- The DreamExecutor's consolidate/synthesize/prune phases mutate observations
-- (they mark sources consolidated, create derived summaries, soft-delete low-
-- importance rows). Two concurrent distillations on the same project would
-- double-merge or double-prune, so the dream takes a per-project lock for its
-- whole run. This table is that lock: one row per project, stamped with the
-- moment it was taken and by whom.
--
-- Additive only: a fresh CREATE TABLE IF NOT EXISTS; no existing table is
-- dropped or rebuilt, and no observations column is touched. Importance and
-- maturity tier are computed on-the-fly from existing columns (retrieval_usage,
-- created_at, type) in the dream code, so this step adds NO observations
-- columns.
--
--   project_id  — the store bucket the distillation is running against (the
--                 lock is per-project, not per-observation).
--   locked_at   — RFC3339 UTC timestamp of the Acquire. A lock older than the
--                 5-minute TTL is treated as stale and auto-released (the
--                 holder crashed or the process died mid-dream).
--   locked_by   — advisory label for the holder (process/agent identity).
--                 Cosmetic only; the lock itself is the row's existence.

CREATE TABLE IF NOT EXISTS distill_lock (
    project_id TEXT PRIMARY KEY,
    locked_at  TEXT NOT NULL,
    locked_by  TEXT NOT NULL DEFAULT ''
);
