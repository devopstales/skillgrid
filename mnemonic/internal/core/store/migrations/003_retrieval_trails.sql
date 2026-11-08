-- 003_retrieval_trails.sql — the retrieval trail (Task 11): the search /
-- lookup / VectorSearch audit log (spec 01-memory/005 §7, INV-5). One row per
-- retrieval: the route (fts/rrf/lookup), the query, the filters, the returned
-- ids, and the degraded status. The "why did the agent recall this" audit
-- record — `mnemonic trail recent|show` inspects it.
--
-- Additive-only: a fresh DB applies 001 + 002 + this; an existing DB (schema
-- version 2) applies this on its next Migrate. Never synced or exported
-- (local-only audit, like session_trails). The project+timestamp index is the
-- `trail recent` read (the newest retrievals for a project first).

CREATE TABLE IF NOT EXISTS retrieval_trails (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	route TEXT NOT NULL,
	query TEXT NOT NULL,
	project TEXT,
	filters TEXT,
	ids TEXT,
	degraded INTEGER NOT NULL DEFAULT 0,
	timestamp TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_retrieval_trails_project ON retrieval_trails(project, timestamp DESC);
