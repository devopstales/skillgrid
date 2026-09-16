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
