-- 100_code_index.sql — the code-index table set, Postgres render (ADR-0046,
-- CI-1). M4: the SAME number as the SQLite file (migrations/100_code_index.sql)
-- — D3 is decided in TICKET-01; this file only mirrors the number.
--
-- The 7 content tables with PG types (BIGSERIAL for rowids, TIMESTAMPTZ for
-- timestamps, JSONB for the JSON columns) + the 2 FTS legs (tsvector + GIN,
-- the dialect-specific rendering — NOT FTS5) + the 2 BLOB vector tables
-- (pgvector, the extension the memory change already creates) + embed_meta.
-- No ANN tables here: the PG floor is brute-force over the pgvector BLOB
-- (TICKET-08a's degraded: brute_force); the HNSW index is 08a's scope.
--
-- The FTS legs mirror the SQLite FTS5 external-content trigram legs: a
-- generated tsvector column + a GIN index + the tsvector trigger set
-- (BEFORE INSERT OR UPDATE). The 'simple' config is the trigram analog —
-- it tokenizes without stemming, so the identifier-first search (the
-- code-index's recall mode) keeps the tokens intact.

-- ── files: the indexing unit (spec 001 §2) ─────────────────────────────────

CREATE TABLE IF NOT EXISTS files (
	id           BIGSERIAL PRIMARY KEY,
	sync_id      TEXT NOT NULL,
	path         TEXT NOT NULL UNIQUE,
	mtime_ns     BIGINT NOT NULL,
	size         BIGINT NOT NULL,
	content_hash TEXT NOT NULL,
	language     TEXT NOT NULL,
	ast_hash     TEXT NOT NULL DEFAULT '',
	indexed_at   TIMESTAMPTZ NOT NULL
);

-- ── chunks: the FTS + semantic leg text (spec 001 §3) ──────────────────────

CREATE TABLE IF NOT EXISTS chunks (
	id           BIGSERIAL PRIMARY KEY,
	file_id      INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
	kind         TEXT NOT NULL DEFAULT 'ast',
	start_line   INTEGER NOT NULL,
	end_line     INTEGER NOT NULL,
	text         TEXT NOT NULL,
	content_hash TEXT NOT NULL,
	tsv          tsvector
);

CREATE INDEX IF NOT EXISTS idx_chunks_file ON chunks(file_id);

CREATE OR REPLACE FUNCTION set_chunks_tsv() RETURNS trigger AS $body$
BEGIN
	NEW.tsv := to_tsvector('simple', COALESCE(NEW.text, ''));
	RETURN NEW;
END;
$body$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS chunks_fts_maintain ON chunks;
CREATE TRIGGER chunks_fts_maintain BEFORE INSERT OR UPDATE ON chunks
FOR EACH ROW EXECUTE FUNCTION set_chunks_tsv();

CREATE INDEX IF NOT EXISTS idx_chunks_tsv_gin ON chunks USING gin (tsv);

-- ── symbols: the single node table (spec 002 §2) ───────────────────────────

CREATE TABLE IF NOT EXISTS symbols (
	id             BIGSERIAL PRIMARY KEY,
	sync_id        TEXT NOT NULL,
	file_id        INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
	name           TEXT NOT NULL,
	qualified_name TEXT,
	kind           TEXT NOT NULL,
	language       TEXT,
	signature      TEXT,
	start_line     INTEGER NOT NULL,
	end_line       INTEGER NOT NULL,
	visibility     TEXT,
	is_exported    BOOLEAN NOT NULL DEFAULT FALSE,
	return_type    TEXT,
	param_types    TEXT,
	content_hash   TEXT NOT NULL,
	uid            TEXT NOT NULL UNIQUE,
	tsv            tsvector
);

CREATE INDEX IF NOT EXISTS idx_symbols_name ON symbols(name);
CREATE INDEX IF NOT EXISTS idx_symbols_qualified ON symbols(qualified_name);
CREATE INDEX IF NOT EXISTS idx_symbols_file ON symbols(file_id);

CREATE OR REPLACE FUNCTION set_symbol_tsv() RETURNS trigger AS $body$
BEGIN
	NEW.tsv := setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'A')
		|| setweight(to_tsvector('simple', COALESCE(NEW.qualified_name, '')), 'B')
		|| setweight(to_tsvector('simple', COALESCE(NEW.signature, '')), 'C')
		|| setweight(to_tsvector('simple', COALESCE(NEW.kind, '')), 'D')
		|| setweight(to_tsvector('simple', COALESCE(NEW.language, '')), 'D');
	RETURN NEW;
END;
$body$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS symbol_fts_maintain ON symbols;
CREATE TRIGGER symbol_fts_maintain BEFORE INSERT OR UPDATE ON symbols
FOR EACH ROW EXECUTE FUNCTION set_symbol_tsv();

CREATE INDEX IF NOT EXISTS idx_symbol_tsv_gin ON symbols USING gin (tsv);

-- ── edges: the relationship table (spec 002 §4) ────────────────────────────
-- to_id NULL + to_name is the name-only (unresolved) state (CI-INV-3).

CREATE TABLE IF NOT EXISTS edges (
	id         BIGSERIAL PRIMARY KEY,
	kind       TEXT NOT NULL,
	from_id    INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE,
	to_id      INTEGER REFERENCES symbols(id) ON DELETE CASCADE,
	to_name    TEXT,
	file_id    INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
	confidence TEXT NOT NULL DEFAULT 'EXTRACTED',
	line       INTEGER,
	context    TEXT NOT NULL DEFAULT '',
	UNIQUE(kind, from_id, file_id, to_id, to_name, line)
);

CREATE INDEX IF NOT EXISTS idx_edges_from ON edges(from_id);
CREATE INDEX IF NOT EXISTS idx_edges_to ON edges(to_id);
CREATE INDEX IF NOT EXISTS idx_edges_to_name ON edges(to_name);
CREATE INDEX IF NOT EXISTS idx_edges_file ON edges(file_id);

-- ── unresolved_edges: the drop-not-guess ledger (spec 001 §5) ──────────────

CREATE TABLE IF NOT EXISTS unresolved_edges (
	id         BIGSERIAL PRIMARY KEY,
	file_id    INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE,
	reference  TEXT NOT NULL,
	kind       TEXT NOT NULL,
	line       INTEGER,
	candidates TEXT,
	status     TEXT NOT NULL DEFAULT 'open',
	first_seen TIMESTAMPTZ NOT NULL,
	last_seen  TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_unresolved_file ON unresolved_edges(file_id);

-- ── index_freshness: the query-path staleness record (spec 001 §4) ─────────
-- The D-TIER column (extractor_tiers) is added here: the spec 001 §4 DDL
-- predates the D-TIER fold. capabilities / extractor_tiers are JSONB (the
-- dialect-specific rendering of the SQLite TEXT JSON columns).

CREATE TABLE IF NOT EXISTS index_freshness (
	project            TEXT PRIMARY KEY,
	extractor_stamp    TEXT NOT NULL,
	schema_fingerprint TEXT NOT NULL,
	last_indexed_at    TIMESTAMPTZ,
	files_indexed      INTEGER NOT NULL DEFAULT 0,
	files_deleted      INTEGER NOT NULL DEFAULT 0,
	files_oversized    INTEGER NOT NULL DEFAULT 0,
	symbols_count      INTEGER NOT NULL DEFAULT 0,
	edges_count        INTEGER NOT NULL DEFAULT 0,
	chunks_count       INTEGER NOT NULL DEFAULT 0,
	unresolved_count   INTEGER NOT NULL DEFAULT 0,
	coverage_fair      DOUBLE PRECISION,
	capabilities       JSONB NOT NULL DEFAULT '{}'::jsonb,
	extractor_tiers    JSONB,
	updated_at         TIMESTAMPTZ NOT NULL
);

-- ── dependencies + dep_edges: the SBOM graph (spec 002 §7, CI-O6) ──────────
-- The pair counts as ONE of the 7 content tables (B1 counting rule).

CREATE TABLE IF NOT EXISTS dependencies (
	purl      TEXT PRIMARY KEY,
	name      TEXT NOT NULL,
	version   TEXT NOT NULL,
	ecosystem TEXT NOT NULL,
	manifest  TEXT,
	retired   BOOLEAN NOT NULL DEFAULT FALSE,
	last_seen TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS dep_edges (
	from_purl     TEXT NOT NULL REFERENCES dependencies(purl) ON DELETE CASCADE,
	to_purl       TEXT NOT NULL REFERENCES dependencies(purl) ON DELETE CASCADE,
	version_range TEXT,
	PRIMARY KEY (from_purl, to_purl)
);

-- ── embeddings + chunk_embeddings: the BLOB vector store (spec 006 §4.1) ──
-- The BLOB tables are the source of truth for the semantic leg; the HNSW
-- ANN index is TICKET-08a's scope (D-CI8). No ANN tables here — the PG
-- floor is brute-force over the pgvector BLOB.

-- The vector columns are unbounded `vector` (not vector(N)): the BLOB is the
-- source of truth for the semantic leg (spec 006 §4.1) and the model's
-- dimension is the `dim` column (a model swap changes it — the guard). An
-- ANN index (TICKET-08a) would pin the dimension via the ops class, but the
-- BLOB floor is brute-force, so the column is dimension-agnostic.
CREATE TABLE IF NOT EXISTS embeddings (
	symbol_id  INTEGER PRIMARY KEY REFERENCES symbols(id) ON DELETE CASCADE,
	model      TEXT NOT NULL,
	dim        INTEGER NOT NULL,
	vector     vector NOT NULL,
	language   TEXT,
	updated_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS chunk_embeddings (
	chunk_id   INTEGER PRIMARY KEY REFERENCES chunks(id) ON DELETE CASCADE,
	model      TEXT NOT NULL,
	dim        INTEGER NOT NULL,
	vector     vector NOT NULL,
	language   TEXT,
	updated_at TIMESTAMPTZ NOT NULL
);

-- the model-swap guard (spec 006 §4.1 — embed_meta.embedding_model): a
-- model change makes every stored vector non-comparable; the guard (the
-- vectorstore.GuardModelSwap, TICKET-08a) reads this key before a run.
CREATE TABLE IF NOT EXISTS embed_meta (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
