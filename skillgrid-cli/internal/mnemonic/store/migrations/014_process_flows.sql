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
