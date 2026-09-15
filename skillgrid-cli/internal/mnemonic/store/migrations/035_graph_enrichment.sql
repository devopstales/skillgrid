-- 035 graph enrichment (additive; 034's edges UNIQUE / upsert contract untouched).
--
-- edges.context records WHY a reference exists (e.g. 'doc_mention', 'route');
-- edges.confidence_score is the numeric form of the categorical confidence label
-- (EXTRACTED=1.0, INFERRED=0.85, AMBIGUOUS=0.5) for graphify-style edge weighting.
-- Both are NOT NULL with defaults, so every existing INSERT INTO edges keeps
-- working (ADD COLUMN does not touch the unique constraint or ON CONFLICT list).
ALTER TABLE edges ADD COLUMN context TEXT NOT NULL DEFAULT '';
ALTER TABLE edges ADD COLUMN confidence_score REAL NOT NULL DEFAULT 1.0;

-- files.ast_hash is a deterministic hash of the extracted symbol/edge STRUCTURE
-- (not raw content): a comment/format-only edit produces the same hash, so a
-- future pass can skip re-embedding when the structure is unchanged.
ALTER TABLE files ADD COLUMN ast_hash TEXT NOT NULL DEFAULT '';

-- community_meta.hub_label is the top god-node name of the community, stored
-- so a cache hit (which rebuilds communities from community_meta) can report
-- the hub without re-ranking god nodes.
ALTER TABLE community_meta ADD COLUMN hub_label TEXT NOT NULL DEFAULT '';
