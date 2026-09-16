-- 012 community knowledge graph schema (additive; 005 symbols/edges untouched)
-- Change 008, step 01.

CREATE TABLE IF NOT EXISTS communities (
    id INTEGER PRIMARY KEY,
    symbol_id INTEGER NOT NULL UNIQUE REFERENCES symbols(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_communities_symbol ON communities(symbol_id);

CREATE TABLE IF NOT EXISTS community_meta (
    id INTEGER PRIMARY KEY,
    label TEXT NOT NULL,
    symbol_count INTEGER NOT NULL,
    god_nodes TEXT NOT NULL DEFAULT '',
    cache_key TEXT NOT NULL
);
