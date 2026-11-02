-- 100_code_index.sql — the code-index table set (ADR-0046, CI-1): the
-- 7 content tables + the 2 FTS tables, additive on the memory change's
-- 001-004 range. D3: the number (100) is decided here; TICKET-08c mirrors
-- the same number for the PG file.
--
-- Single node table (CI-2): every graph node — code symbol, route, doc,
-- config, SQL table — is a `symbols` row with a kind. No temporal window
-- (CI-4). No valid_from/valid_to anywhere.
--
-- The FTS legs (chunks_fts, symbol_fts) are external-content FTS5 trigram
-- (the memory change's observations_fts pattern — the same DDL + the same
-- four-trigger sync set; TICKET-02 extracts it into the fts.go unit).

-- ── files: the indexing unit (spec 001 §2) ─────────────────────────────────

CREATE TABLE IF NOT EXISTS files (
	id           INTEGER PRIMARY KEY,
	sync_id      TEXT NOT NULL,
	path         TEXT NOT NULL UNIQUE,
	mtime_ns     INTEGER NOT NULL,
	size         INTEGER NOT NULL,
	content_hash TEXT NOT NULL,
	language     TEXT NOT NULL,
	ast_hash     TEXT NOT NULL DEFAULT '',
	indexed_at   TEXT NOT NULL
);

-- ── chunks: the FTS + semantic leg text (spec 001 §3) ──────────────────────

CREATE TABLE IF NOT EXISTS chunks (
	id           INTEGER PRIMARY KEY,
	file_id      INTEGER NOT NULL,
	kind         TEXT NOT NULL DEFAULT 'ast',
	start_line   INTEGER NOT NULL,
	end_line     INTEGER NOT NULL,
	text         TEXT NOT NULL,
	content_hash TEXT NOT NULL,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file_id);

-- ── symbols: the single node table (spec 002 §2) ───────────────────────────

CREATE TABLE IF NOT EXISTS symbols (
	id             INTEGER PRIMARY KEY,
	sync_id        TEXT NOT NULL,
	file_id        INTEGER NOT NULL,
	name           TEXT NOT NULL,
	qualified_name TEXT,
	kind           TEXT NOT NULL,
	language       TEXT,
	signature      TEXT,
	start_line     INTEGER NOT NULL,
	end_line       INTEGER NOT NULL,
	visibility     TEXT,
	is_exported    INTEGER NOT NULL DEFAULT 0,
	return_type    TEXT,
	param_types    TEXT,
	content_hash   TEXT NOT NULL,
	uid            TEXT NOT NULL UNIQUE,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_qualified ON symbols(qualified_name);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);

-- ── edges: the relationship table (spec 002 §4) ────────────────────────────
-- to_id NULL + to_name is the name-only (unresolved) state (CI-INV-3).

CREATE TABLE IF NOT EXISTS edges (
	id         INTEGER PRIMARY KEY,
	kind       TEXT NOT NULL,
	from_id    INTEGER NOT NULL,
	to_id      INTEGER,
	to_name    TEXT,
	file_id    INTEGER NOT NULL,
	confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
	line       INTEGER,
	context    TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (from_id) REFERENCES symbols(id) ON DELETE CASCADE,
	FOREIGN KEY (to_id) REFERENCES symbols(id) ON DELETE CASCADE,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
	UNIQUE(kind, from_id, file_id, to_id, to_name, line)
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id) WHERE to_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_edges_to_name ON edges(to_name) WHERE to_id IS NULL;
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);

-- ── unresolved_edges: the drop-not-guess ledger (spec 001 §5) ──────────────

CREATE TABLE IF NOT EXISTS unresolved_edges (
	id         INTEGER PRIMARY KEY,
	file_id    INTEGER NOT NULL,
	reference  TEXT NOT NULL,
	kind       TEXT NOT NULL,
	line       INTEGER,
	candidates TEXT,
	status     TEXT NOT NULL DEFAULT 'open',
	first_seen TEXT NOT NULL,
	last_seen  TEXT NOT NULL,
	FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_unresolved_file ON unresolved_edges(file_id);

-- ── index_freshness: the query-path staleness record (spec 001 §4) ─────────
-- The D-TIER column (extractor_tiers) is added here: the spec 001 §4 DDL
-- predates the D-TIER fold (drift noted in the coordinator's spec set).

CREATE TABLE IF NOT EXISTS index_freshness (
	project            TEXT PRIMARY KEY,
	extractor_stamp    TEXT NOT NULL,
	schema_fingerprint TEXT NOT NULL,
	last_indexed_at    TEXT,
	files_indexed      INTEGER NOT NULL DEFAULT 0,
	files_deleted      INTEGER NOT NULL DEFAULT 0,
	files_oversized    INTEGER NOT NULL DEFAULT 0,
	symbols_count      INTEGER NOT NULL DEFAULT 0,
	edges_count        INTEGER NOT NULL DEFAULT 0,
	chunks_count       INTEGER NOT NULL DEFAULT 0,
	unresolved_count   INTEGER NOT NULL DEFAULT 0,
	coverage_fair      REAL,
	capabilities       TEXT NOT NULL,
	extractor_tiers    TEXT,
	updated_at         TEXT NOT NULL
);

-- ── dependencies + dep_edges: the SBOM graph (spec 002 §7, CI-O6) ──────────
-- The pair counts as ONE of the 7 content tables (B1 counting rule).

CREATE TABLE IF NOT EXISTS dependencies (
	purl      TEXT PRIMARY KEY,
	name      TEXT NOT NULL,
	version   TEXT NOT NULL,
	ecosystem TEXT NOT NULL,
	manifest  TEXT,
	retired   INTEGER NOT NULL DEFAULT 0,
	last_seen TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS dep_edges (
	from_purl     TEXT NOT NULL,
	to_purl       TEXT NOT NULL,
	version_range TEXT,
	PRIMARY KEY (from_purl, to_purl),
	FOREIGN KEY (from_purl) REFERENCES dependencies(purl) ON DELETE CASCADE,
	FOREIGN KEY (to_purl) REFERENCES dependencies(purl) ON DELETE CASCADE
);

-- ── embeddings + chunk_embeddings: the BLOB vector store (spec 006 §4.1) ──
-- The BLOB tables are the source of truth for the semantic leg; the vec0 /
-- HNSW ANN tables are TICKET-08a's scope (D-CI8) and land in their own
-- migration. Created here because the vector Store methods (ADR-0046 M2)
-- need a table to render against from day one.

CREATE TABLE IF NOT EXISTS embeddings (
	symbol_id  INTEGER PRIMARY KEY,
	model      TEXT NOT NULL,
	dim        INTEGER NOT NULL,
	vector     BLOB NOT NULL,
	language   TEXT,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (symbol_id) REFERENCES symbols(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS chunk_embeddings (
	chunk_id   INTEGER PRIMARY KEY,
	model      TEXT NOT NULL,
	dim        INTEGER NOT NULL,
	vector     BLOB NOT NULL,
	language   TEXT,
	updated_at TEXT NOT NULL,
	FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

-- the model-swap guard (spec 006 §4.1 — embed_meta.embedding_model): a
-- model change makes every stored vector non-comparable; the guard (the
-- vectorstore.GuardModelSwap, TICKET-08a) reads this key before a run.
CREATE TABLE IF NOT EXISTS embed_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

-- ── chunks_fts: the chunk FTS leg (spec 001 §3, CI-INV-1 derived) ──────────

CREATE VIRTUAL TABLE chunks_fts USING fts5(
text,
tokenize='trigram',
content='chunks',
content_rowid='id'
);

CREATE TRIGGER chunks_fts_insert AFTER INSERT ON chunks BEGIN
	INSERT INTO chunks_fts(rowid, text)
	VALUES (new.id, new.text);
END;

CREATE TRIGGER chunks_fts_delete BEFORE DELETE ON chunks BEGIN
	DELETE FROM chunks_fts WHERE rowid = old.id;
END;

CREATE TRIGGER chunks_fts_update BEFORE UPDATE ON chunks BEGIN
	DELETE FROM chunks_fts WHERE rowid = old.id;
END;

CREATE TRIGGER chunks_fts_update_insert AFTER UPDATE ON chunks BEGIN
	INSERT INTO chunks_fts(rowid, text)
	VALUES (new.id, new.text);
END;

-- ── symbol_fts: the symbol FTS leg (spec 002 §8, CI-INV-1 derived) ─────────

CREATE VIRTUAL TABLE symbol_fts USING fts5(
name,
qualified_name,
signature,
kind,
language,
tokenize='trigram',
content='symbols',
content_rowid='id'
);

CREATE TRIGGER symbol_fts_insert AFTER INSERT ON symbols BEGIN
	INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
	VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;

CREATE TRIGGER symbol_fts_delete BEFORE DELETE ON symbols BEGIN
	DELETE FROM symbol_fts WHERE rowid = old.id;
END;

CREATE TRIGGER symbol_fts_update BEFORE UPDATE ON symbols BEGIN
	DELETE FROM symbol_fts WHERE rowid = old.id;
END;

CREATE TRIGGER symbol_fts_update_insert AFTER UPDATE ON symbols BEGIN
	INSERT INTO symbol_fts(rowid, name, qualified_name, signature, kind, language)
	VALUES (new.id, new.name, new.qualified_name, new.signature, new.kind, new.language);
END;
