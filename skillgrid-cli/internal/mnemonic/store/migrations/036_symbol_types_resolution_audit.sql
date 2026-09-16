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
