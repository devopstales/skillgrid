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
