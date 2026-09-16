-- 011b: ensure edges carries a file_id for target-state pruning of
-- re-indexed files. Fresh stores already have the column (added in 011);
-- this statement is a guarded no-op for them and an additive upgrade for
-- pre-011b stores.

CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);

-- Backfill file_id for any existing edges that lack it (pre-011b stores).
UPDATE edges
SET file_id = (
    SELECT s.file_id FROM symbols s WHERE s.id = edges.from_id
)
WHERE file_id IS NULL;
