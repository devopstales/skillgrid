-- 006: semantic relations between observations (mem_judge / mem_compare) and
-- project-name aliasing (mem_merge_projects).

-- A directed, typed link between two observations in the SAME project store.
-- relation vocabulary (mem_judge verdicts):
--   conflicts_with  — the two observations contradict
--   supersedes      — src supersedes dest (dest is stale)
--   related         — topical neighbours
--   compatible      — both can be true simultaneously
--   scoped          — one is a narrowing of the other
--   not_conflict / compatible / scoped are all stored; a not_conflict VERDICT
--   REMOVES a previously-recorded conflicts_with edge instead of adding one.
CREATE TABLE IF NOT EXISTS memory_relations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    src_obs_id INTEGER NOT NULL,
    dst_obs_id INTEGER NOT NULL,
    relation TEXT NOT NULL,
    confidence REAL,
    reason TEXT,
    project TEXT NOT NULL,
    created_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_memory_relations_src
    ON memory_relations (src_obs_id, relation) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_memory_relations_dst
    ON memory_relations (dst_obs_id, relation) WHERE deleted_at IS NULL;

-- Project-name aliases. When the user merges project variants, the old name is
-- recorded here as an alias of the canonical one. Resolution consults the
-- alias table first, so future writes to the old name land in the canonical
-- store.
CREATE TABLE IF NOT EXISTS project_aliases (
    alias TEXT PRIMARY KEY,
    canonical TEXT NOT NULL,
    merged_at TEXT NOT NULL
);
