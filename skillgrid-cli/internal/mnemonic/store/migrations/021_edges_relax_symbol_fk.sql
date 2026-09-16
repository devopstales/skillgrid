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
