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
