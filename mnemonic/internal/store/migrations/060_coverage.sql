-- 060_coverage: per-file line coverage rows, keyed by (project, file) so a
-- re-scan upserts instead of duplicating (additive; R9 coverage dispatch).
CREATE TABLE IF NOT EXISTS coverage (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project TEXT NOT NULL,
	file TEXT NOT NULL,
	package TEXT NOT NULL DEFAULT '',
	pct_line REAL NOT NULL,
	scanned_at TEXT NOT NULL,
	UNIQUE (project, file)
);
CREATE INDEX IF NOT EXISTS idx_coverage_project ON coverage(project);
