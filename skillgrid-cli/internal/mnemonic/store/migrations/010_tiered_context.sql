-- 010: tiered context registry, long-term memories, retrieval trails, path embeddings.
--
-- Additive only — CREATE TABLE IF NOT EXISTS. Existing observations, FTS5,
-- and the code index are untouched. Numbering: 009 is already used
-- (tool_name); this change ships 010_* as planned for tiered context.
--
--   tiered_contents   — L2 full_path plus optional L0/L1 sidecar paths
--   long_term_memories — durable commit targets (optional link to tiered row)
--   retrieval_trails  — per-search query / directories / files / result path
--   path_embeddings   — Pure Go / optional embedding blob keyed by path

CREATE TABLE IF NOT EXISTS tiered_contents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    full_path TEXT NOT NULL,
    abstract_path TEXT,
    overview_path TEXT,
    title TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (project, full_path)
);

CREATE INDEX IF NOT EXISTS idx_tiered_contents_project
    ON tiered_contents (project, updated_at DESC);

CREATE TABLE IF NOT EXISTS long_term_memories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    title TEXT,
    tiered_content_id INTEGER REFERENCES tiered_contents(id),
    source_link TEXT,
    full_path TEXT NOT NULL,
    abstract_path TEXT,
    overview_path TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ltm_project
    ON long_term_memories (project, created_at DESC);

CREATE TABLE IF NOT EXISTS retrieval_trails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    query TEXT NOT NULL,
    directories_json TEXT,
    files_json TEXT,
    result_path TEXT,
    corpus TEXT,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_retrieval_trails_project
    ON retrieval_trails (project, created_at DESC);

CREATE TABLE IF NOT EXISTS path_embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    path TEXT NOT NULL,
    embedding BLOB,
    embedding_model TEXT,
    embedding_created_at TEXT,
    UNIQUE (project, path)
);

CREATE INDEX IF NOT EXISTS idx_path_embeddings_project
    ON path_embeddings (project, embedding_created_at DESC)
    WHERE embedding IS NOT NULL;
