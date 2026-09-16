-- 033: hub file scoring + observation risk (change 014, step 23).
--
-- Additive: two new nullable columns; no existing schema is touched.
--
--   symbols.hub_score — the file-level hub ratio stamped by
--                       memory.IdentifyHubFiles (014 step 23.1):
--                       importers / total_files, in [0, 1]. NULL until the
--                       first hub pass; 0 for non-hub files. The value is
--                       file-level (a hub is a property of a file's import
--                       fan-in, not of a single symbol) but stored on each
--                       symbol row of the file so it can be joined without a
--                       second hop.
--
--   observations.risk_score — the hub-involvement risk stamped by
--                       memory.RecomputeRiskScores (014 step 23.3): the
--                       hub_score of the file the observation references
--                       (via graph_ref → symbols.file_id → files), 0 for
--                       observations whose referenced file is not a hub (or
--                       has no hub_score stamped yet). NULL until the first
--                       risk pass (pre-033 rows).
--
-- Both columns are advisory: they power the `mem graph --risk` view and the
-- change-impact analysis; a stamp failure must never fail the save, and a
-- missing value degrades to 0 at read time.

ALTER TABLE symbols ADD COLUMN hub_score REAL;
ALTER TABLE observations ADD COLUMN risk_score REAL;

-- Risk-view support: risk-desc lookups within a project for `mem graph
-- --risk` (the query reads stored values in-memory, but the column is
-- indexed so the threshold filter pre-filters without a full scan).
CREATE INDEX IF NOT EXISTS idx_obs_risk
    ON observations (project, risk_score DESC)
    WHERE deleted_at IS NULL AND risk_score IS NOT NULL;
