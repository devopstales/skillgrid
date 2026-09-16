-- 029: directory-retrieval trajectory columns on retrieval_trails (change 014,
-- step 19).
--
-- The retrieval_trails table ALREADY EXISTS from migration 010 (tiered
-- context): it is the per-search audit table with the 010 columns
-- (project, query, directories_json, files_json, result_path, corpus,
-- created_at). Step 19 ADDS the directory-drill-down trajectory to the SAME
-- table, purely additively — no existing column is touched, and the 010
-- code path (insertTrail) keeps working unchanged on the pre-029 rows
-- (these columns read back NULL for legacy rows).
--
--   path     — the complete drill-down path as a JSON array of trajectory
--              steps (each step: path, score, depth, path_prefix).
--   scores   — the per-visit scores, in order (a JSON array of floats).
--   depth    — the final depth reached (the deepest directory visited).

ALTER TABLE retrieval_trails ADD COLUMN path TEXT;
ALTER TABLE retrieval_trails ADD COLUMN scores TEXT;
ALTER TABLE retrieval_trails ADD COLUMN depth INTEGER;
