-- 003_retrieval_trails.sql — the retrieval trail (Task 11) PG render: the
-- search / lookup / VectorSearch audit log (spec 01-memory/005 §7, INV-5).
-- Same semantics as the SQLite render; additive-only.

CREATE TABLE IF NOT EXISTS retrieval_trails (
	id BIGSERIAL PRIMARY KEY,
	route TEXT NOT NULL,
	query TEXT NOT NULL,
	project TEXT,
	filters TEXT,
	ids TEXT,
	degraded BOOLEAN NOT NULL DEFAULT FALSE,
	timestamp TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_retrieval_trails_project ON retrieval_trails(project, timestamp DESC);
