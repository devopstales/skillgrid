-- 025: AKL importance scoring columns (change 014, step 13).
--
-- Additive: three new nullable columns on observations; no existing schema
-- is touched. The values are populated on save (memory.Service.Save) and on
-- an explicit recompute (memory.Service.RecomputeImportance):
--
--   importance_score — retrieval_usage * exp(-decay_rate * age_days), the
--                      continuous AKL importance value (014 step 13.1).
--                      NULL until first stamped (pre-025 rows).
--   recency_decay    — exp(-decay_rate * age_days), the recency factor
--                      alone, stored for provenance.
--   maturity_tier    — 'fresh' | 'mature' | 'archival' (014 step 13.2).
--                      Transitions are monotonic (never downgraded); the
--                      stored value is max(previous, computed).

ALTER TABLE observations ADD COLUMN importance_score REAL;
ALTER TABLE observations ADD COLUMN recency_decay REAL;
ALTER TABLE observations ADD COLUMN maturity_tier TEXT;

-- Ranking/prune support: importance-desc lookups within a project (the
-- query-time boost reads stored values in-memory, but the column is indexed
-- so prune/dream phases can pre-filter without a full scan).
CREATE INDEX IF NOT EXISTS idx_obs_importance
    ON observations (project, importance_score DESC)
    WHERE deleted_at IS NULL AND importance_score IS NOT NULL;
