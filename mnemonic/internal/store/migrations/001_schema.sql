
-- ===== 001_initial.sql =====
-- Skillgrid Mnemonic v1 initial schema

PRAGMA foreign_keys = ON;

-- Memory (Engram-aligned)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    directory TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    summary TEXT,
    status TEXT NOT NULL DEFAULT 'active'
);

CREATE TABLE IF NOT EXISTS observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    project TEXT,
    scope TEXT,
    topic_key TEXT,
    normalized_hash TEXT,
    revision_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE VIRTUAL TABLE IF NOT EXISTS observations_fts USING fts5(
    title,
    content,
    type,
    project,
    content='observations',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS observations_fts_insert AFTER INSERT ON observations BEGIN
    INSERT INTO observations_fts(rowid, title, content, type, project)
    VALUES (new.id, new.title, new.content, new.type, new.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_delete AFTER DELETE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
    VALUES ('delete', old.id, old.title, old.content, old.type, old.project);
END;

CREATE TRIGGER IF NOT EXISTS observations_fts_update AFTER UPDATE ON observations BEGIN
    INSERT INTO observations_fts(observations_fts, rowid, title, content, type, project)
    VALUES ('delete', old.id, old.title, old.content, old.type, old.project);
    INSERT INTO observations_fts(rowid, title, content, type, project)
    VALUES (new.id, new.title, new.content, new.type, new.project);
END;

-- Code index (OpenClaw-aligned)
CREATE TABLE IF NOT EXISTS files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    mtime_ns INTEGER NOT NULL,
    size INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    indexed_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chunks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    text TEXT NOT NULL,
    content_hash TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
    text,
    path UNINDEXED,
    content='chunks',
    content_rowid='id',
    tokenize='trigram'
);

CREATE TRIGGER IF NOT EXISTS chunks_fts_insert AFTER INSERT ON chunks BEGIN
    INSERT INTO chunks_fts(rowid, text, path)
    SELECT new.id, new.text, (SELECT path FROM files WHERE id = new.file_id);
END;

CREATE TRIGGER IF NOT EXISTS chunks_fts_delete AFTER DELETE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, text, path)
    VALUES (
        'delete',
        old.id,
        old.text,
        (SELECT path FROM files WHERE id = old.file_id)
    );
END;

CREATE TRIGGER IF NOT EXISTS chunks_fts_update AFTER UPDATE ON chunks BEGIN
    INSERT INTO chunks_fts(chunks_fts, rowid, text, path)
    VALUES (
        'delete',
        old.id,
        old.text,
        (SELECT path FROM files WHERE id = old.file_id)
    );
    INSERT INTO chunks_fts(rowid, text, path)
    SELECT new.id, new.text, (SELECT path FROM files WHERE id = new.file_id);
END;

-- Web cache (Neuledge-aligned)
CREATE TABLE IF NOT EXISTS web_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT,
    source TEXT NOT NULL,
    cache_key TEXT NOT NULL,
    url TEXT,
    title TEXT,
    query TEXT,
    library_id TEXT,
    version_tag TEXT,
    content TEXT NOT NULL,
    metadata_json TEXT,
    content_hash TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    expires_at TEXT,
    session_id TEXT,
    created_at TEXT,
    UNIQUE(project, source, cache_key)
);

CREATE VIRTUAL TABLE IF NOT EXISTS web_cache_fts USING fts5(
    title,
    content,
    query,
    url,
    source,
    library_id,
    content='web_cache',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS web_cache_fts_insert AFTER INSERT ON web_cache BEGIN
    INSERT INTO web_cache_fts(rowid, title, content, query, url, source, library_id)
    VALUES (new.id, new.title, new.content, new.query, new.url, new.source, new.library_id);
END;

CREATE TRIGGER IF NOT EXISTS web_cache_fts_delete AFTER DELETE ON web_cache BEGIN
    INSERT INTO web_cache_fts(web_cache_fts, rowid, title, content, query, url, source, library_id)
    VALUES ('delete', old.id, old.title, old.content, old.query, old.url, old.source, old.library_id);
END;

CREATE TRIGGER IF NOT EXISTS web_cache_fts_update AFTER UPDATE ON web_cache BEGIN
    INSERT INTO web_cache_fts(web_cache_fts, rowid, title, content, query, url, source, library_id)
    VALUES ('delete', old.id, old.title, old.content, old.query, old.url, old.source, old.library_id);
    INSERT INTO web_cache_fts(rowid, title, content, query, url, source, library_id)
    VALUES (new.id, new.title, new.content, new.query, new.url, new.source, new.library_id);
END;

-- Meta (migration bookkeeping is managed by store.migrate)

-- ===== 002_session_title.sql =====
-- 002: named sessions. The `title` column shows the session name in the web
-- dashboard session list (mem-sessions). Fresh databases get the column here;
-- the updated 001 omits it so this is the single source of truth.
ALTER TABLE sessions ADD COLUMN title TEXT;

-- ===== 003_backfill_session_titles.sql =====
-- 003: give sessions that have NEITHER a title NOR a summary a placeholder
-- title so they render in the web dashboard session list (mem-sessions)
-- instead of being dropped by the (title OR summary) filter in RecentContext.
-- Sessions that DO have a summary get their title derived from the "## Goal"
-- line by the memory service (deriveSessionTitle) at end/summary/read time —
-- kept in Go because modernc.org/sqlite lacks the regexp/char functions needed
-- for SQL-level extraction.
UPDATE sessions
SET title = 'Untitled session'
WHERE (title IS NULL OR TRIM(title) = '')
  AND (summary IS NULL OR TRIM(summary) = '');

-- ===== 004_prompts_passive.sql =====
-- Skillgrid Mnemonic v1.2 — prompts capture + passive observation provenance
--
-- Prompts: captured user prompts (plugin side strips <private> tags, truncates
-- to 2000 chars). Kept for searchability across sessions and as a source of
-- context for "what were we working on" recall.
CREATE TABLE IF NOT EXISTS prompts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    project TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_prompts_session ON prompts(session_id);
CREATE INDEX IF NOT EXISTS idx_prompts_project_time ON prompts(project, created_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS prompts_fts USING fts5(
    content,
    content='prompts',
    content_rowid='id',
    tokenize='porter'
);

CREATE TRIGGER IF NOT EXISTS prompts_fts_insert AFTER INSERT ON prompts BEGIN
    INSERT INTO prompts_fts(rowid, content) VALUES (new.id, new.content);
END;

CREATE TRIGGER IF NOT EXISTS prompts_fts_delete AFTER DELETE ON prompts BEGIN
    INSERT INTO prompts_fts(prompts_fts, rowid, content)
    VALUES ('delete', old.id, old.content);
END;

-- Provenance for observations. Values: "agent" (default, via mem_save),
-- "passive" (server-extracted from assistant/tool output), "prompt" (rare,
-- used if a prompt is later promoted to an observation), or "compact"
-- (saved during compaction).
ALTER TABLE observations ADD COLUMN source TEXT NOT NULL DEFAULT 'agent';

-- Per-project migration bookkeeping so `skillgrid setup` can detect when a
-- project id changed (e.g. rename, git remote change) and roll the data over.
-- The migration is one-time and driven from HTTP POST /projects/migrate.
CREATE TABLE IF NOT EXISTS project_migrations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    old_project TEXT NOT NULL,
    new_project TEXT NOT NULL,
    migrated_at TEXT NOT NULL
);

-- ===== 005_review_cycle.sql =====
-- Mnemonic: per-observation review cycle (mem_review).
-- review_after is intentionally NOT copied on project migration (local-only
-- metadata). It is NULL until a review cycle is set.

ALTER TABLE observations ADD COLUMN review_after TEXT;

-- ===== 006_relations_aliases.sql =====
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

-- ===== 007_prompt_link.sql =====
-- 007: prompt provenance link on observations.
--
-- When mem_save is called with capture_prompt=true (the default), the server
-- best-effort attaches the most recent user prompt recorded for the same
-- session (via mem_save_prompt / the prompts table) to the observation. This
-- closes the Engram "capture_prompt" gap: the observation row then carries a
-- prompt_id so a reader can recall *what was asked* alongside *what we
-- concluded*, without reconstructing it from the session transcript.
--
-- prompt_id is intentionally NULLable and denormalised (a plain FK column, no
-- ON DELETE) so that deleting a prompt never cascades to observations and so
-- that older observations (saved before this migration) are simply unaffected.
ALTER TABLE observations ADD COLUMN prompt_id INTEGER;

-- ===== 008_obs_lifecycle.sql =====
-- 008: observation lifecycle + embedding columns.
--
-- Parity with Engram's observation lifecycle. Every column is additive
-- (ADD COLUMN with a safe default) so existing databases upgrade without a
-- rewrite. The FTS index stays untouched — search still runs on the same
-- title/content fields; the new fields change ordering and ranking, not
-- matching.
--
--   pinned             — first-class pin (mem_pin / mem_unpin). Pinned rows
--                        sort ahead of everything else in mem_context and
--                        boost mem_search ordering.
--   expires_at         — RFC3339 timestamp; when in the past the row is
--                        soft-excluded from search (mirrors Engram's TTL).
--   duplicate_count    — incremented on repeated Save() of the same hash;
--                        powers Engram-style "seen N times" recency
--                        weighting without storing a full counter of hits.
--   last_seen_at       — updated whenever a duplicate save happens. Gives
--                        search a recency signal independent of revision
--                        count.
--   embedding          — optional BLOB payload of floating32 vector. Only
--                        populated when MNEMONIC_EMBED=1 and an embedder is
--                        available; otherwise NULL, and search falls back
--                        to FTS5-only (reciprocal rank fusion only when a
--                        non-NULL embedding set is present).
--   embedding_model    — model name / version that produced the embedding,
--                        for provenance and to gate re-embedding on model
--                        swap.
--   embedding_created_at — when the embedding was written.

-- SQLite has no DROP COLUMN and ADD COLUMN must have a constant default,
-- so the pinned and lifecycle columns are added individually with their
-- defaults set at the column level.

ALTER TABLE observations ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;

-- expires_at is nullable — most rows never expire.
ALTER TABLE observations ADD COLUMN expires_at TEXT;

-- duplicate_count starts at 0 (NOT Engram's 1) so the first Save() is
-- unambiguous; we only count observed duplicates past the first write.
-- SQLite does not support default expressions, so the initial value is 0
-- for ALL rows (including pre-existing ones).
ALTER TABLE observations ADD COLUMN duplicate_count INTEGER NOT NULL DEFAULT 0;

-- last_seen_at is the timestamp of the most recent duplicate save, in UTC
-- RFC3339 format, same as created_at.
ALTER TABLE observations ADD COLUMN last_seen_at TEXT;

-- P4 — embedding triplet. Stored as BLOB because SQLite has no native
-- vector type; we own the byte layout (little-endian float32, contiguous)
-- and the dimension is inferred from len/4 when needed. The model string
-- is free-form so we can embed with a local ONNX endpoint or a remote URL
-- depending on MNEMONIC_EMBED_ENDPOINT.
ALTER TABLE observations ADD COLUMN embedding BLOB;
ALTER TABLE observations ADD COLUMN embedding_model TEXT;
ALTER TABLE observations ADD COLUMN embedding_created_at TEXT;

-- Ordering index so mem_context and mem_search can sort by pinned DESC,
-- recency DESC with a single seek. SQLite ignores leading columns with
-- NULL values, so this is cheap to maintain.
CREATE INDEX IF NOT EXISTS idx_obs_pinned
    ON observations (pinned DESC, last_seen_at DESC, created_at DESC)
    WHERE deleted_at IS NULL;

-- TTL sweep: expires_at is the only TTL source (no separate "expiring"
-- marker).
CREATE INDEX IF NOT EXISTS idx_obs_expires
    ON observations (expires_at)
    WHERE expires_at IS NOT NULL AND deleted_at IS NULL;

-- Dedup helper: identical to Engram's idx_obs_dedupe. Used by Save() to
-- locate existing rows for duplicate_count / last_seen_at updates without
-- a full table scan.
CREATE INDEX IF NOT EXISTS idx_obs_dedupe
    ON observations (normalized_hash, project, scope, type, title, created_at DESC);

-- Embedding lookup: when enabled, we fetch embeddings by project so a
-- subsequent search can compute cosine over the right candidate set. The
-- index only exists to prune non-embedded rows; the BLOB itself is not
-- searchable.
CREATE INDEX IF NOT EXISTS idx_obs_embedding
    ON observations (project, embedding_created_at DESC)
    WHERE embedding IS NOT NULL;

-- ===== 009_teams_schema.sql =====
-- 009: Hybrid agent teams control-plane tables (metadata + file paths only).
-- Content lives under {project}/.skillgrid/files/; SQLite stores paths/status.
-- Additive only — does not rewrite observations or existing mnemonic tables.

CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    role TEXT NOT NULL,
    system_prompt_path TEXT,
    model TEXT,
    status TEXT NOT NULL DEFAULT 'idle',
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    parent_task_id TEXT,
    title TEXT NOT NULL,
    brief_path TEXT NOT NULL,
    output_path TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    assigned_to TEXT,
    created_by TEXT,
    priority INTEGER NOT NULL DEFAULT 0,
    memory_context_path TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    completed_at TEXT,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_task_id) REFERENCES tasks(id),
    FOREIGN KEY (assigned_to) REFERENCES team_members(id)
);

CREATE INDEX IF NOT EXISTS idx_tasks_status_priority
    ON tasks (status, priority DESC, created_at ASC);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    subject TEXT,
    body_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unread',
    in_reply_to TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    read_at TEXT,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (from_agent) REFERENCES team_members(id),
    FOREIGN KEY (to_agent) REFERENCES team_members(id),
    FOREIGN KEY (in_reply_to) REFERENCES messages(id)
);

CREATE TABLE IF NOT EXISTS task_results (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    output_path TEXT NOT NULL,
    summary TEXT,
    token_usage INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS reviews (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    reviewer_id TEXT NOT NULL,
    review_type TEXT NOT NULL,
    passed INTEGER NOT NULL,
    comments_path TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (reviewer_id) REFERENCES team_members(id)
);

-- ===== 009_tool_name.sql =====
-- 009: observation tool_name provenance.
-- Additive column so mem_save can record which tool produced the row
-- (Engram-parity tool provenance). Nullable — most rows leave it unset.

ALTER TABLE observations ADD COLUMN tool_name TEXT;

-- ===== 010_tiered_context.sql =====
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

-- ===== 011_hybrid_code_intel.sql =====
-- 011 hybrid code intelligence schema (additive; do not rewrite files/chunks)
-- Change 005, step 01.

CREATE TABLE IF NOT EXISTS symbols (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    qualified_name TEXT,
    kind TEXT NOT NULL,
    language TEXT,
    signature TEXT,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    content_hash TEXT NOT NULL,
    uid TEXT NOT NULL UNIQUE
);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);
CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);

CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    from_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    to_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    to_name TEXT,
    target_path TEXT,
    confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
    line INTEGER,
    UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line)
);
CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_kind ON edges(kind);

CREATE TABLE IF NOT EXISTS rationale (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'note',
    line INTEGER
);
CREATE INDEX IF NOT EXISTS idx_rationale_symbol ON rationale(symbol_id);

CREATE VIRTUAL TABLE IF NOT EXISTS symbol_fts USING fts5(
    name,
    qualified_name,
    signature,
    kind,
    language,
    content='symbols',
    content_rowid='id',
    tokenize='unicode61'
);

CREATE TRIGGER IF NOT EXISTS symbol_fts_insert AFTER INSERT ON symbols BEGIN
    INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;

CREATE TRIGGER IF NOT EXISTS symbol_fts_delete AFTER DELETE ON symbols BEGIN
    INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name, signature, kind, language)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.kind, old.language);
END;

CREATE TRIGGER IF NOT EXISTS symbol_fts_update AFTER UPDATE ON symbols BEGIN
    INSERT INTO symbol_fts(symbol_fts, rowid, name, qualified_name, signature, kind, language)
    VALUES ('delete', old.id, old.name, old.qualified_name, old.signature, old.kind, old.language);
    INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
    VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;

CREATE TABLE IF NOT EXISTS embeddings (
    symbol_id INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    vector BLOB NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS embed_meta (
    key TEXT PRIMARY KEY,
    value TEXT
);

CREATE TABLE IF NOT EXISTS lsh_buckets (
    bucket INTEGER NOT NULL,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    PRIMARY KEY (bucket, symbol_id)
);
CREATE INDEX IF NOT EXISTS idx_lsh_bucket ON lsh_buckets(bucket);

CREATE TABLE IF NOT EXISTS index_freshness (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- ===== 011b_edges_file_id.sql =====
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

-- ===== 012_community_knowledge_graph.sql =====
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

-- ===== 013_framework_routes.sql =====
-- 013 framework routes schema (additive; 005 symbols/edges untouched).
-- Change 010, step 01.
--
-- Route and navigation metadata lives in a small additive side table that
-- mirrors the 005 symbols table. Route nodes are `kind='route'` rows in
-- symbols; `references` (route -> handler) and `navigates` (function ->
-- screen) edges are rows in the existing 005 edges table. This table only
-- carries the framework-specific payload (framework, method, path pattern,
-- screen, and the per-file drop-not-guess drop count) so the 005 node/edge
-- contract stays byte-for-byte unchanged.
CREATE TABLE IF NOT EXISTS route_meta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    framework TEXT NOT NULL,
    method TEXT,
    path_pattern TEXT,
    screen TEXT
);
CREATE INDEX IF NOT EXISTS idx_route_meta_symbol ON route_meta(symbol_id);
CREATE INDEX IF NOT EXISTS idx_route_meta_file ON route_meta(file_id);

-- Per-file extraction diagnostics: the count of references/navigates that were
-- dropped at extraction by the drop-not-guess policy (ambiguous, no same-file
-- match, no explicit specifier, no unique global/owner-qualified match). A
-- drop is reported as a warning (count + sample), never a silent discard, and
-- dropped refs are excluded from blast-radius math.
CREATE TABLE IF NOT EXISTS route_drops (
    file_id INTEGER PRIMARY KEY REFERENCES files(id) ON DELETE CASCADE,
    dropped INTEGER NOT NULL DEFAULT 0
);

-- ===== 014_process_flows.sql =====
-- 014 precomputed process flows schema (additive; 005 symbols/edges + 012/013
-- untouched). Change 008, step 02.
--
-- A process is a precomputed execution flow traced from one of 010's entry
-- points (a kind='route' symbol, a route's resolved handler, or a CLI main)
-- through the 005 call edges. The flow is advisory, never load-bearing — it
-- reads the graph but adds no symbols/edges of its own, so 005 search is
-- unchanged. The trace is deterministic (depth-capped BFS, stable ordering)
-- and cached by the process's content-hash (process_meta_cache) so an
-- unchanged re-index does not re-trace or re-label.
CREATE TABLE IF NOT EXISTS processes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    entry_symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    entry_kind TEXT,
    cross_community INTEGER NOT NULL DEFAULT 0,
    content_hash TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    label_status TEXT NOT NULL DEFAULT 'unlabeled',
    stop_note TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_processes_name ON processes(name);
CREATE UNIQUE INDEX IF NOT EXISTS idx_processes_content ON processes(content_hash);

CREATE TABLE IF NOT EXISTS process_steps (
    process_id INTEGER NOT NULL REFERENCES processes(id) ON DELETE CASCADE,
    step INTEGER NOT NULL,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    confidence TEXT,
    kind TEXT,
    PRIMARY KEY (process_id, step)
);
CREATE INDEX IF NOT EXISTS idx_process_steps_symbol ON process_steps(symbol_id);

-- Content-hash cache for the process pass (mirrors the 012 community
-- community_meta_cache pattern): the process-pass content key that the current
-- stored partition corresponds to, so an unchanged re-index is a cache hit.
CREATE TABLE IF NOT EXISTS process_meta_cache (
    key TEXT PRIMARY KEY,
    value TEXT
);

-- ===== 015_knowledge_graph.sql =====
-- 015 knowledge graph schema (additive; 005 symbols/edges + 012/013/014
-- untouched). Change 008, step 03.
--
-- The knowledge pass indexes the code's surrounding knowledge — markdown
-- docs, config files, and SQL DDL — as first-class graph nodes alongside the
-- 005 symbols:
--
--   doc_nodes          a markdown doc (kind=doc node in this table)
--   config_nodes       a yaml/toml/json config file (kind=config node)
--   sql_schema_nodes   a SQL DDL table or column (kind=table|column)
--
-- The knowledge edges live in the existing 005 edges table (so the graph
-- spans code→doc→config→table in one traversal):
--
--   references  doc node -> doc node        (markdown link / wikilink)
--   configures  config node -> code symbol  (a config key names the code)
--   reads       code symbol -> table node   (SQL DML reads the table)
--   writes      code symbol -> table node   (SQL DML writes the table)
--
-- Every knowledge edge carries a confidence label (EXTRACTED for explicit
-- syntax, INFERRED for convention-derived, AMBIGUOUS for a resolved-but-
-- guessed or unresolvable reference). Unresolvable CONFIG references are
-- kept AMBIGUOUS (not dropped, per 03.5); doc/SQL references follow the
-- drop-or-AMBIGUOUS rule per scenario. The knowledge nodes reference their
-- source file (file_id cascade) so a deleted file prunes its footprint.
CREATE TABLE IF NOT EXISTS doc_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    path TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_doc_nodes_file ON doc_nodes(file_id);

CREATE TABLE IF NOT EXISTS config_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    path TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_config_nodes_file ON config_nodes(file_id);

CREATE TABLE IF NOT EXISTS sql_schema_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    column_name TEXT,
    kind TEXT NOT NULL,
    path TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sql_schema_nodes_table ON sql_schema_nodes(table_name);

-- ===== 016_pdg_taint.sql =====
-- 016 per-function CFG + PDG + taint schema (additive; 005 symbols/edges +
-- 012/013/014/015 untouched). Change 011, step 01.
--
-- The brief named this migration 014_pdg_taint.sql, but 014 is already taken
-- by 008's process flows; this is the next free number (016). It is additive:
-- the four tables are created here and populated ONLY by the opt-in --pdg pass
-- (a non---pdg index leaves them empty, and 005/008/010 are byte-for-byte
-- unchanged). No new grammar, no CGo — the CFG is built over the existing
-- gotreesitter AST, and the PDG + taint solver are pure Go.
--
-- cfg_blocks: basic blocks of a per-function control-flow graph. A block is
-- keyed stably by (symbol_id, start_line, end_line) so repeated builds of the
-- same function are reproducible. symbol_id references the 005 function/method
-- symbol whose body was CFG'd.
CREATE TABLE IF NOT EXISTS cfg_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    block_no INTEGER NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    kind TEXT NOT NULL DEFAULT 'normal',
    UNIQUE(symbol_id, block_no)
);
CREATE INDEX IF NOT EXISTS idx_cfg_blocks_symbol ON cfg_blocks(symbol_id);

-- cfg_edges: control-flow edges between basic blocks of the SAME function
-- (the CFG is intraprocedural; crossing a call boundary is not a CFG edge).
-- condition labels the branch condition ('true'/'false'/'loop'/'post'/'call'
-- etc.) that justified the edge.
CREATE TABLE IF NOT EXISTS cfg_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    from_block INTEGER NOT NULL,
    to_block INTEGER NOT NULL,
    condition TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, from_block, to_block, condition)
);
CREATE INDEX IF NOT EXISTS idx_cfg_edges_symbol ON cfg_edges(symbol_id);

-- pdg_edges: control- and data-dependence edges derived from the CFG. Every
-- edge carries a Confidence Label in EXTRACTED | INFERRED | AMBIGUOUS |
-- LSP_RESOLVED (the fourth value joins 005's three). A data-dependence is
-- EXTRACTED only where fully resolved; an unresolved boundary is
-- INFERRED/AMBIGUOUS (never a fabricated EXTRACTED); an LSP-resolved boundary
-- is LSP_RESOLVED, not AMBIGUOUS. kind is 'control' or 'data'.
CREATE TABLE IF NOT EXISTS pdg_edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    from_line INTEGER NOT NULL,
    to_line INTEGER NOT NULL,
    from_name TEXT NOT NULL DEFAULT '',
    to_name TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT 'AMBIGUOUS',
    note TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, kind, from_line, to_line, from_name, to_name, confidence)
);
CREATE INDEX IF NOT EXISTS idx_pdg_edges_symbol ON pdg_edges(symbol_id);

-- taint_findings: the step 02 taint-solver output (tainted source -> sink
-- through a data flow). Created here so the opt-in ---pdg schema is complete;
-- step 01 leaves it empty (no taint pass yet).
CREATE TABLE IF NOT EXISTS taint_findings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
    source_line INTEGER NOT NULL,
    sink_line INTEGER NOT NULL,
    source_name TEXT NOT NULL DEFAULT '',
    sink_name TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT 'AMBIGUOUS',
    note TEXT NOT NULL DEFAULT '',
    UNIQUE(symbol_id, source_line, sink_line, source_name, sink_name, confidence)
);
CREATE INDEX IF NOT EXISTS idx_taint_findings_symbol ON taint_findings(symbol_id);

-- ===== 017_layered_memory_governance.sql =====
-- 017: layered memory governance (change 013, step 01).
--
-- Additive governance on top of 005's observation store: every observation
-- becomes a governed asset with an owner, a version history (append, not
-- overwrite), a lifecycle status, a retrieval-usage counter, and a visibility
-- (private by default). All statements are CREATE TABLE IF NOT EXISTS /
-- ALTER TABLE ADD COLUMN so an existing database upgrades without a rewrite;
-- the 005 observations table is never dropped or rebuilt.
--
--   owner             — the creating user/agent identity. Set at save time;
--                       a new observation always has an owner.
--   status            — active | superseded | archived. Active by default;
--                       never inferred from content (explicit mutation only).
--   visibility        — private | team | restricted | agent. Private by
--                       default: a new observation is `private` until an
--                       explicit mem_share. `private` is invisible to other
--                       owners (the creating owner always sees its own).
--   retrieval_usage   — times the observation was returned by a search.
--                       Distinct from duplicate_count (re-saves).
--
-- observation_versions — append-only version history. A mem_update appends a
-- row (observation_id, revision, content, created) instead of overwriting;
-- the latest revision is the read path and revision_count advances.

ALTER TABLE observations ADD COLUMN owner TEXT;
ALTER TABLE observations ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE observations ADD COLUMN visibility TEXT NOT NULL DEFAULT 'private';
ALTER TABLE observations ADD COLUMN retrieval_usage INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS observation_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    revision INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- A restricted/agent observation is readable by owner + its explicit grants.
CREATE TABLE IF NOT EXISTS acl_grants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    observation_id INTEGER NOT NULL REFERENCES observations(id) ON DELETE CASCADE,
    grantee TEXT NOT NULL,
    grant_type TEXT NOT NULL DEFAULT 'agent',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_obs_versions
    ON observation_versions (observation_id, revision);
CREATE INDEX IF NOT EXISTS idx_obs_grants
    ON acl_grants (observation_id, grantee);

-- ===== 018_layered_distill.sql =====
-- 018: layered distillation (change 013, step 02).
--
-- Additive on top of 017_layered_memory_governance.sql (step 01) and 005's
-- observation/session store. Every statement is CREATE TABLE IF NOT EXISTS so
-- an existing database upgrades without a rewrite; no 005 / 017 table is
-- dropped or rebuilt.
--
-- The layering is PROVENANCE-LINKED: a distilled record (L1 atom, L2 scenario,
-- L3 persona delta) is never created unless its L0 source (session and/or
-- topic) resolves to a live row. A distilled record whose source cannot be
-- resolved is simply not written — a layer is never orphaned from its
-- provenance.
--
--   observation_layers — the L0→L1→L2→L3 link table. Each row links a derived
--   layer record to its resolvable L0 source. The L0 source is a session_id
--   (the raw session record) and/or a topic (the session summary). An L1 atom
--   is an existing observation (observation_id, its session is the L0); an
--   L2/L3 record lives in personas (persona_id). The same derived record may
--   be re-linked to a changed L0 source on re-distill (dedup by
--   project+layer+target+source+content_hash).
--
--   personas — the L2 scenario + L3 persona-delta records. A scenario is a
--   working-context block distilled from a session; a persona delta is a
--   long-term profile increment. Both carry their content hash so a re-distill
--   on an unchanged L0 source is a cache hit (no re-distillation).

CREATE TABLE IF NOT EXISTS observation_layers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    layer TEXT NOT NULL,
    target_kind TEXT NOT NULL,
    target_id INTEGER,
    source_session TEXT NOT NULL,
    source_topic TEXT,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, layer, target_kind, target_id, source_session, content_hash)
);

CREATE TABLE IF NOT EXISTS personas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    kind TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL
);

-- distill_llm_cache — the content-hash cache for the optional LLM pass. A row
-- per (project, source_hash) holds the L2 scenario + L3 persona delta the LLM
-- produced for that L0 source. Re-running the pass on an unchanged source is a
-- cache hit (no re-distillation, no second LLM call); a changed source misses.
CREATE TABLE IF NOT EXISTS distill_llm_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    source_hash TEXT NOT NULL,
    scenario TEXT NOT NULL,
    delta TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, source_hash)
);

CREATE INDEX IF NOT EXISTS idx_layers_session
    ON observation_layers (project, source_session, layer);
CREATE INDEX IF NOT EXISTS idx_layers_target
    ON observation_layers (project, layer, target_kind, target_id);
CREATE INDEX IF NOT EXISTS idx_personas_kind
    ON personas (project, kind);

-- ===== 019_session_relay.sql =====
-- 019: structured session handoff (change 006, step 01).
--
-- Additive relay schema on top of 001's session/observation store. Every
-- statement is CREATE TABLE IF NOT EXISTS / CREATE INDEX IF NOT EXISTS so an
-- existing database upgrades without a rewrite; the 001 sessions and
-- observations tables are never dropped, altered, or rebuilt.
--
-- The relay is the continuity unit for a session: a handoff row points at the
-- on-disk cleave bundle (PROGRESS.md / KNOWLEDGE.md / NEXT_PROMPT.md under
-- .skillgrid/.cleave/) and an archive row records a handoff that was folded
-- into a resumable archive. Steps 02–05 (relay FS, MCP tools, CLI, watchdog)
-- build on these tables; no row here is ever written without its cleave
-- bundle on disk (fail closed).
--
--   handoff_id       — the operator-facing identifier (also the cleave
--                      bundle directory name); unique, resume target.
--   source_session   — the sessions.id this handoff was taken from.
--   status           — pending | archived. Pending by default; flipped
--                      explicitly when the handoff is archived.
--   cleave_path      — repo-relative path of the cleave bundle directory.
--   context_summary  — optional short context note written at handoff time.

CREATE TABLE IF NOT EXISTS session_handoffs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    handoff_id TEXT NOT NULL,
    source_session TEXT REFERENCES sessions(id) ON DELETE SET NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    cleave_path TEXT NOT NULL,
    context_summary TEXT,
    created_at TEXT NOT NULL,
    archived_at TEXT,
    UNIQUE (project, handoff_id)
);

-- session_archives — one row per handoff that was archived. path is the
-- on-disk archive (tarball or directory) the next session resumes from.
CREATE TABLE IF NOT EXISTS session_archives (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE SET NULL,
    -- No FK to session_handoffs: SQLite FKs must target a PRIMARY KEY or
    -- UNIQUE index, and handoff_id's uniqueness is scoped (project,
    -- handoff_id), so the link is enforced by the application (step 02).
    handoff_id TEXT NOT NULL,
    path TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, handoff_id)
);

CREATE INDEX IF NOT EXISTS idx_handoffs_source
    ON session_handoffs (project, source_session);
CREATE INDEX IF NOT EXISTS idx_handoffs_status
    ON session_handoffs (project, status);
CREATE INDEX IF NOT EXISTS idx_archives_session
    ON session_archives (project, session_id);

-- ===== 020_ttl_extraction.sql =====
-- 020: TTL config + extraction metadata (change 014, step 01).
-- Change-014's migration, sequenced as 020 (the brief's 014_ prefix collides
-- with the existing 014_process_flows.sql); sorts after 019_session_relay.sql
-- so it applies last. Additive: new tables only; no existing schema is touched.

CREATE TABLE IF NOT EXISTS ttl_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS extraction_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT,
    content_hash TEXT,
    extracted_at TIMESTAMP,
    model TEXT
);

-- ===== 021_edges_relax_symbol_fk.sql =====
-- 021: relax the symbols foreign keys on edges.from_id / edges.to_id.
--
-- 011 declared `from_id` and `to_id` as REFERENCES symbols(id). That is wrong
-- for knowledge-graph edges (015): doc->doc `references` edges (and config->
-- doc edges) use doc_nodes.id / config_nodes.id as their endpoints, which do
-- not exist in symbols. With foreign_keys=ON (always set at open), upserting a
-- doc reference whose from_id is a doc_nodes PK failed with
-- `FOREIGN KEY constraint failed (787)`, so the knowledge pass warned and
-- dropped doc references on existing stores.
--
-- The codebase already treats edge endpoints as polymorphic node ids
-- (doc_nodes.id, config_nodes.id, sql_schema_nodes.id, or symbols.id):
--   - knowledge/store.go SaveDoc:        from_id = doc_nodes.id
--   - knowledge_test.go:                 JOIN doc_nodes dn ON dn.id = e.from_id
--   - codeindex/indexer_hook_test:       from_id IN (SELECT id FROM doc_nodes)
-- so the FK to symbols is both wrong and unenforceable (a doc_nodes id is
-- never a symbols id). Drop the two symbols FKs; keep the files FK (a valid
-- edge always references a real file) and the unique constraint.
--
-- SQLite cannot drop a single FK without rebuilding the table. This is a
-- simple, idempotent rebuild: recreate `edges` without the symbols FKs, copy
-- the rows (preserving PKs and any doc/config-node endpoints), drop the old
-- table, and recreate the original indexes (which DROP TABLE removed). A
-- second run rebuilds the already-fixed table — a data-preserving no-op.

CREATE TABLE IF NOT EXISTS edges (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    from_id INTEGER NOT NULL,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    to_id INTEGER,
    to_name TEXT,
    target_path TEXT,
    confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
    line INTEGER,
    UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line)
);

CREATE TABLE edges_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    kind TEXT NOT NULL,
    from_id INTEGER NOT NULL,
    file_id INTEGER REFERENCES files(id) ON DELETE CASCADE,
    to_id INTEGER,
    to_name TEXT,
    target_path TEXT,
    confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
    line INTEGER,
    UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line)
);

INSERT INTO edges_new (id, kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
    SELECT id, kind, from_id, file_id, to_id, to_name, target_path, confidence, line FROM edges;

DROP TABLE edges;

ALTER TABLE edges_new RENAME TO edges;

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_kind ON edges(kind);

-- ===== 022_triple_store.sql =====
-- 022: triple-store cross-linkage (change 014, step 07).
--
-- Cross-links the relational store (observations, long_term_memories), the
-- vector store (embeddings), and the graph store (symbols) inside the single
-- per-project SQLite database. Additive: every new column is nullable, so
-- existing rows and every existing query path are untouched. The cross-link
-- query (memory.CrossLinkQuery) and the integrity check
-- (memory.CheckCrossLinkIntegrity) are opt-in methods, not part of the
-- default search path.
--
--   observations.graph_ref      — codeindex symbols.id when the observation's
--                                 source file has an indexed symbol; NULL
--                                 otherwise (best-effort, never blocks Save).
--   long_term_memories.embedding_blob — optional precomputed embedding BLOB
--                                 (little-endian float32), mirroring the
--                                 observations.embedding layout.
--
-- NOTE: the `embeddings` table (migration 011) already bridges symbols to
-- vectors and lives in the SAME store as observations, so it is reused as
-- the symbol_embeddings bridge instead of creating a duplicate table. The
-- codeindex package exposes it via the symbol_embeddings.go accessors.

ALTER TABLE observations ADD COLUMN graph_ref INTEGER;

CREATE INDEX IF NOT EXISTS idx_obs_graph_ref
    ON observations (graph_ref)
    WHERE graph_ref IS NOT NULL;

ALTER TABLE long_term_memories ADD COLUMN embedding_blob BLOB;

-- ===== 023_temporal_edges.sql =====
-- 023: temporal bounds on graph edges (change 014, step 10).
--
-- Adds valid_from / valid_to to the `edges` table so a relationship carries the
-- time window in which it was true. Additive only:
--
--   valid_from INTEGER NOT NULL DEFAULT 0   — UNIX seconds when the relationship
--                                             was first observed. 0 is the
--                                             backfill for edges created before
--                                             this migration ("unknown/always
--                                             active": 0 <= now, so the
--                                             current-state filter keeps them).
--   valid_to   INTEGER                      — NULL = the edge is still active;
--                                             a past UNIX second = the edge
--                                             expired at that moment and is
--                                             hidden from current-state queries
--                                             but preserved for history.
--
-- SQLite ADD COLUMN cannot take an expression, so valid_from's default is the
-- constant 0; new edge INSERTs set valid_from = time.Now().Unix() explicitly
-- (codeindex), so the 0 default only applies to rows created before this
-- migration. valid_to has no default (NULL) — a fresh edge is active.
--
-- Current-state edge queries filter on
--     valid_from <= now AND (valid_to IS NULL OR valid_to > now)
-- so expired (valid_to <= now) and not-yet-active (valid_from > now) edges are
-- hidden from graph traversal, while the history read (QueryEdgesWithHistory)
-- applies no filter.

ALTER TABLE edges ADD COLUMN valid_from INTEGER NOT NULL DEFAULT 0;
ALTER TABLE edges ADD COLUMN valid_to INTEGER;

-- Partial index on the active window: the current-state filter scans this
-- instead of the full edges table. valid_to IS NULL rows (the active
-- majority) are included; expired rows (valid_to set) are pruned as they age.
CREATE INDEX IF NOT EXISTS idx_edges_valid_to
    ON edges (valid_from, valid_to)
    WHERE valid_to IS NULL;

-- ===== 024_dream_lock.sql =====
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

-- ===== 025_importance.sql =====
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

-- ===== 026_observation_relations.sql =====
-- 026: typed @relation edges between observations (change 014, step 14).
--
-- Additive: a NEW table in the same per-project store. It is distinct from
-- memory_relations (006, mem_judge verdicts) and from the codeindex edges
-- table (011/021, symbol-level graph edges): observation_relations is a
-- typed, confidence-weighted relation between OBSERVATION ids.
--
--   source_id      — the observation that asserts the relation.
--   target_id      — the observation the relation points at.
--   relation_type  — one of the canonical set: mentions, depends_on,
--                    contradicts, supports, references.
--   confidence     — 0.0..1.0 weight on the assertion.
--
-- The composite primary key makes each (source, target, type) triple unique,
-- so re-asserting the same triple upserts confidence instead of duplicating.

CREATE TABLE IF NOT EXISTS observation_relations (
    source_id INTEGER NOT NULL,
    target_id INTEGER NOT NULL,
    relation_type TEXT NOT NULL,
    confidence REAL NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (source_id, target_id, relation_type)
);

-- Bidirectional lookup support (mem relations <id> queries source OR target).
CREATE INDEX IF NOT EXISTS idx_obs_rel_source
    ON observation_relations (source_id);
CREATE INDEX IF NOT EXISTS idx_obs_rel_target
    ON observation_relations (target_id);

-- ===== 027_provenance.sql =====
-- 027: provenance chain metadata on observations (change 014, step 15).
--
-- Additive: one new nullable TEXT column on observations. No existing schema
-- is touched; pre-027 rows read back with provenance NULL (no-provenance
-- saves keep working unchanged).
--
--   provenance — a JSON object recording the curation chain that produced
--                the observation:
--                  {"session_id": "...", "curate_command": "mem_save ...",
--                   "source_files": ["src/a.go"], "llm_reasoning": "..."}
--                Set ONLY on the initial save; the update path preserves it
--                (immutable once set — only the first write may establish
--                the chain).

ALTER TABLE observations ADD COLUMN provenance TEXT;

-- ===== 028_memory_type.sql =====
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

-- ===== 029_retrieval_trails.sql =====
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

-- ===== 030_snapshots.sql =====
-- 030: multi-version store snapshots (change 014, step 20.1).
--
-- A snapshot is a point-in-time view of a project's observations state:
-- the serialized rows as a BLOB plus a SHA-256 integrity hash. Snapshots are
-- multi-version — every Snapshot() call adds a new row, so a project can roll
-- back to any previous capture. The table is purely additive: it adds no
-- columns to observations and touches no existing schema.
--
--   id          — surrogate primary key (monotonic; newest snapshot = highest id)
--   project_id  — the project bucket this snapshot captures
--   state_hash  — SHA-256 (hex) of the serialized data BLOB (integrity check)
--   data        — the serialized observations state (JSON array of rows)
--   created_at  — RFC3339 UTC timestamp of the capture

CREATE TABLE IF NOT EXISTS snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id TEXT NOT NULL,
    state_hash TEXT NOT NULL,
    data BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL
);

-- ===== 031_handoff.sql =====
-- 031: handoff cursor (change 014, step 21).
--
-- The handoff artifact (prefix + delta) is generated on demand. The delta
-- ("changed since the last handoff") needs a persistent, cross-process cursor
-- because the memory package cannot depend on the service layer and a fresh CLI
-- process cannot rely on in-process state. This tiny key/value table stores the
-- last-handoff timestamp (RFC3339) under a fixed key. Purely additive: a new
-- table, no changes to any existing schema.
--
--   key   — fixed key (handoff:cursor); one row per project store
--   value — RFC3339 UTC timestamp of the most recent handoff generation

CREATE TABLE IF NOT EXISTS handoff_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- ===== 033_hub_score.sql =====
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

-- ===== 034_unresolved_refs.sql =====
-- 034 unresolved refs + symbol segment vocabulary (additive; 005/013 untouched).
--
-- unresolved_refs logs every route-handler reference that failed to resolve at
-- extraction (drop-not-guess): the reference name, kind, line, the number of
-- global symbols matched by name, and its status. It is target-state per file
-- (a re-index replaces the file's rows) so a ref that later resolves drops
-- out of the table. A UNIQUE identity index carries the upsert.
CREATE TABLE IF NOT EXISTS unresolved_refs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    reference_name TEXT NOT NULL,
    reference_kind TEXT NOT NULL DEFAULT 'route_handler',
    line INTEGER NOT NULL DEFAULT 0,
    candidates INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'failed',
    first_seen_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_unresolved_identity
    ON unresolved_refs(file_id, reference_name, reference_kind, line);
CREATE INDEX IF NOT EXISTS idx_unresolved_file ON unresolved_refs(file_id);
CREATE INDEX IF NOT EXISTS idx_unresolved_name ON unresolved_refs(reference_name);
CREATE INDEX IF NOT EXISTS idx_unresolved_status ON unresolved_refs(status);

-- symbol_segments is a reverse index of name segment -> symbol name (a
-- codegraph-style name_segment_vocab) for cheap partial-reference resolution:
-- a segment of a symbol name (CamelCase / snake_case / digit boundary) maps to
-- the symbol it belongs to. Target-state per file at index time.
CREATE TABLE IF NOT EXISTS symbol_segments (
    segment TEXT NOT NULL,
    symbol_name TEXT NOT NULL,
    PRIMARY KEY (segment, symbol_name)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS idx_symbol_segments_name ON symbol_segments(symbol_name);

-- ===== 035_graph_enrichment.sql =====
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

-- ===== 036_symbol_types_resolution_audit.sql =====
-- 036: per-symbol type enrichment + call-resolution audit ledger.
--
-- Borrowed from the gitnexus graph index's manifest, which quantifies the
-- resolver's own decisions (a guessed/refused ledger, an unresolved-member
-- histogram, declared capabilities) and enriches each symbol with type and
-- visibility data. Mnemonic had the drop-not-guess POLICY but no quantified
-- ledger of how often each branch fired, and symbols carried no type/visibility
-- columns at all.

--
-- 1. Symbol type/visibility enrichment
--
-- The gotreesitter DefinitionSpan carries only name/kind/range, so these are
-- DERIVED from the source span at extraction time (a Go/TS heuristic, not a
-- type inference). Empty string means "not derivable / not a function", never
-- "unknown-but-present". This makes code_rename / code_impact able to reason
-- about signatures, and is the same enrichment the external codegraph and
-- gitnexus indexes both expose.
ALTER TABLE symbols ADD COLUMN return_type  TEXT NOT NULL DEFAULT '';
ALTER TABLE symbols ADD COLUMN param_types  TEXT NOT NULL DEFAULT '';
ALTER TABLE symbols ADD COLUMN visibility   TEXT NOT NULL DEFAULT '';
ALTER TABLE symbols ADD COLUMN is_exported  INTEGER NOT NULL DEFAULT 0;

--
-- 2. Unresolved receiver-member call sites
--
-- A receiver-qualified call (obj.Scan(...), rows.QueryRow(...), x.UTC(...)) is
-- name-only against a receiver the extractor cannot statically bind. gitnexus
-- records these per member name with a histogram plus an internal/external
-- split. The graph step resolves the ones whose receiver type is a known
-- symbol; what remains is the genuine "these call sites are unresolvable and
-- here's the shape of why" backlog. This is the call-layer analogue of the
-- route-layer unresolved_refs table from 034.
CREATE TABLE IF NOT EXISTS unresolved_members (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    file_id    INTEGER REFERENCES files(id) ON DELETE SET NULL,
    file_path  TEXT NOT NULL,
    language   TEXT,
    member     TEXT NOT NULL,
    receiver   TEXT NOT NULL DEFAULT '',
    external   INTEGER NOT NULL DEFAULT 0,
    line       INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_umember_member  ON unresolved_members (member);
CREATE INDEX IF NOT EXISTS idx_umember_file_id ON unresolved_members (file_id);

--
-- 3. Call-resolution audit ledger
--
-- Per run, per language: how many call sites were extracted, and how many
-- receiver-qualified members were left unresolved. This quantifies the
-- drop-not-guess policy ("we refused N, guessed 0, across M Go calls") so an
-- agent can see the resolver's health instead of a bare "index is stale".
CREATE TABLE IF NOT EXISTS resolution_audit (
    language       TEXT PRIMARY KEY,
    call_sites     INTEGER NOT NULL DEFAULT 0,
    unresolved     INTEGER NOT NULL DEFAULT 0
);

--
-- 4. Index schema fingerprint
--
-- A short fingerprint over the structural schema (the tables + columns the
-- graph step writes). If it differs from the one baked into the current
-- binary, the index was built by an older extraction schema and is silently
-- stale even though every file's content hash is clean. index_meta is
-- INTEGER-typed (schema_version) so string values live in a side key/value
-- table, mirroring fingerprint_meta / embed_meta.
CREATE TABLE IF NOT EXISTS index_meta_kv (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- ===== 037_ast_chunking_semantic_partition.sql =====
-- 037: AST-boundary chunking for the semantic tier + language partition on
-- the vector store. Borrowed from cocoindex-code (the 4th external index in
-- docs/NOTES.md), whose whole identity is "AST-based semantic code search":
-- it chunks at Tree-Sitter logical boundaries (~1000 chars) and stores the
-- embedding's language as a partition key for exact index-level filtering.
-- Mnemonic chunked by fixed line windows (ChunkLines) and had no language
-- column on the vector store, so a language-scoped semantic search full-scanned
-- and filtered in Go.

--
-- 1. Chunk lineage
--
-- A chunk is now either a fixed line window (kind='lines', the previous
-- behavior) or an AST symbol boundary (kind='ast', one per function/class/
-- method). The embedding tier prefers ast chunks — a function-level chunk is
-- contextually coherent AND fits a 512-token encoder window — and falls back
-- to line chunks for non-code or when a symbol exceeds the char target.
ALTER TABLE chunks ADD COLUMN kind TEXT NOT NULL DEFAULT 'lines';

--
-- 2. Chunk-level semantic embedding
--
-- The vector store was symbol-only (one row per symbol, PK symbol_id). The
-- embedding TIER now also embeds chunk text (the ~1000-char coherent unit),
-- so semantic recall works for non-symbol code (lambdas, config blocks, doc
-- spans) and for the chunk-level recall cocoindex-code is built around. A
-- symbol still keeps its name+signature vector in `embeddings`; chunk vectors
-- live here, keyed by chunk_id.
CREATE TABLE IF NOT EXISTS chunk_embeddings (
    chunk_id   INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
    model      TEXT NOT NULL,
    dim        INTEGER NOT NULL,
    vector     BLOB NOT NULL,
    updated_at TEXT NOT NULL
);

--
-- 3. Language partition on the vector store
--
-- The cocoindex partition key, expressed relationally: a `language` column on
-- both vector tables so a "search only Go" semantic query is an exact
-- index-level filter (WHERE language = ?) instead of a full-scan + Go-side
-- filter. An index on the column makes the filter cheap.
ALTER TABLE embeddings ADD COLUMN language TEXT NOT NULL DEFAULT '';
ALTER TABLE chunk_embeddings ADD COLUMN language TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_embeddings_language ON embeddings (language);
CREATE INDEX IF NOT EXISTS idx_chunk_embeddings_language ON chunk_embeddings (language);

-- ===== 038_community_analytics.sql =====
-- 038: Community structural analytics + import-cycle detection. Borrowed from
-- graphify (the 2nd external index in docs/NOTES.md), whose GRAPH_REPORT.md
-- surfaces three structural signals mnemonic did not compute:
--   1. per-community COHESION (internal-edge density) — a low number says
--      "this cluster is held together by little" and an agent reading
--      code_communities should see it per community;
--   2. IMPORT CYCLES — dependency loops (A imports B imports A) are a
--      classic structural smell graphify calls out under "Import Cycles";
--   3. the god-node ranking graphify prints — mnemonic already has that
--      (community_meta.god_nodes / code_god_nodes), so it is NOT rebuilt.
-- Mnemonic's community_meta had only label/symbol_count/god_nodes/hub_label:
-- no cohesion score, no cycle table. Both are cheap graph walks over data the
-- index already stores (the 005 symbols/edges), so they land as advisory
-- pass outputs, never load-bearing.

--
-- 1. Cohesion per community
--
-- A REAL column on community_meta (the per-community metadata table, already
-- rewritten target-state by the community pass). NULL means "not yet computed
-- by a 038 pass" (a store built before 038, or the trivial one-community
-- early return). Computed as internal_edges / C(n,2) in Go (no SQL division
-- by zero for the n<2 case).
ALTER TABLE community_meta ADD COLUMN cohesion REAL;

--
-- 2. Import cycles
--
-- A standalone table (NOT on community_meta): a cycle is a property of the
-- import subgraph, not of a single community, and cycles can span community
-- boundaries. One row per distinct cycle discovered; the canonical cycle is
-- stored as a comma-separated list of file paths in traversal order (the
-- path repeats its start to close the loop). Rebuilt target-state each index
-- run (DELETE all, then re-detect) because the import subgraph is recomputed
-- from the live edges table and cycles come and go with the code.
CREATE TABLE IF NOT EXISTS import_cycles (
    id          INTEGER PRIMARY KEY,
    cycle       TEXT NOT NULL,
    node_count  INTEGER NOT NULL,
    discovered  TEXT NOT NULL
);

-- ===== 039_handoff_hub.sql =====
-- 039: handoff hub — change snapshots, named checkpoints, handoff refs
-- (change 015-handoff-hub).
--
-- Additive only. The three tables combine the three existing checkpoint
-- sources into one queryable hub:
--
--   change_snapshots — a git-derived, append-only log of EVERY commit for the
--                      project. Reconstructable by replaying `git log`, so it
--                      can never be the sole source of truth; the commit
--                      (and its [skillgrid-context] block) is the durable
--                      record, this table is the index. context_json is the
--                      parsed [skillgrid-context] block (Task/Decisions/
--                      Remaining/Tried) and is NULL when the commit carries
--                      no block. Idempotent upsert on (project, commit).
--
--   checkpoints      — named, INTENTIONAL markers placed before risky actions
--                      (before-apply-<change>, before long pauses, before
--                      validation). Unlike change_snapshots these are placed
--                      by a human/agent, capture dirty state + links to the
--                      active spec dir / PRD / handoff file, and carry a
--                      lifecycle (open | verified | stale | archived) driven
--                      by drift verification.
--
--   handoff_refs     — the join between a handoff artifact (session cleave
--                      bundle, team task completion, or a checkpoint marker)
--                      and the commit range it covers, plus the spec dir it
--                      hands off. This is what makes "this handoff covers
--                      commits X..Y of change Z" explicit.

CREATE TABLE IF NOT EXISTS change_snapshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    branch TEXT NOT NULL,
    "commit" TEXT NOT NULL,
    commit_short TEXT,
    subject TEXT,
    context_json TEXT,
    changed_files TEXT,
    author TEXT,
    committed_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    UNIQUE (project, "commit")
);

CREATE INDEX IF NOT EXISTS idx_snapshots_project_time
    ON change_snapshots (project, committed_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS checkpoints (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT NOT NULL,
    name TEXT NOT NULL,
    branch TEXT,
    "commit" TEXT,
    dirty INTEGER NOT NULL DEFAULT 0,
    prd_path TEXT,
    spec_dir TEXT,
    handoff_file TEXT,
    evidence TEXT,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TEXT NOT NULL,
    verified_at TEXT,
    UNIQUE (project, name)
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_project_status
    ON checkpoints (project, status, created_at DESC);

CREATE TABLE IF NOT EXISTS handoff_refs (
    handoff_id TEXT NOT NULL,
    handoff_type TEXT NOT NULL,
    project TEXT NOT NULL,
    from_commit TEXT,
    to_commit TEXT,
    spec_dir TEXT,
    team_id TEXT,
    task_id TEXT,
    created_at TEXT NOT NULL,
    UNIQUE (handoff_id, handoff_type, project)
);

CREATE INDEX IF NOT EXISTS idx_handoff_refs_project
    ON handoff_refs (project, created_at DESC);
