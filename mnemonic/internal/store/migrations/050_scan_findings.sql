-- 050_scan_findings: structured scanner results + dependency graph (additive; T2, no new deps)
CREATE TABLE IF NOT EXISTS scans (
	id TEXT PRIMARY KEY,                         -- UUIDv7
	tool TEXT NOT NULL,                          -- trivy | wapiti | nuclei | semgrep
	target TEXT NOT NULL,
	scanners TEXT NOT NULL DEFAULT '',           -- e.g. vuln,sbom
	status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','ok','error','partial')),
	started_at TEXT NOT NULL,
	finished_at TEXT,
	finding_count INTEGER NOT NULL DEFAULT 0,
	raw_path TEXT,
	error TEXT
);
CREATE INDEX IF NOT EXISTS idx_scans_started ON scans (started_at);
CREATE INDEX IF NOT EXISTS idx_scans_tool ON scans (tool, started_at);

CREATE TABLE IF NOT EXISTS findings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	scan_id TEXT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
	tool TEXT NOT NULL,
	dedup_hash TEXT NOT NULL,                    -- sha256(tool, cve|rule_id, pkg, version, file, line)
	severity TEXT NOT NULL,                      -- normalized: CRITICAL|HIGH|MEDIUM|LOW|INFO
	title TEXT NOT NULL,
	cve_id TEXT,
	rule_id TEXT,
	package TEXT,
	version TEXT,
	fixed_version TEXT,
	file TEXT,
	line INTEGER,
	message TEXT,
	links TEXT NOT NULL DEFAULT '[]',            -- JSON array of URLs
	UNIQUE (scan_id, dedup_hash)
);
CREATE INDEX IF NOT EXISTS idx_findings_scan ON findings (scan_id);
CREATE INDEX IF NOT EXISTS idx_findings_hash ON findings (dedup_hash);
CREATE INDEX IF NOT EXISTS idx_findings_sev ON findings (severity);

CREATE VIRTUAL TABLE IF NOT EXISTS findings_fts USING fts5(
	title, cve_id, rule_id, package, message, file,
	content='findings', content_rowid='id'
);
CREATE TRIGGER IF NOT EXISTS findings_ai AFTER INSERT ON findings BEGIN
	INSERT INTO findings_fts(rowid, title, cve_id, rule_id, package, message, file)
	VALUES (new.id, new.title, new.cve_id, new.rule_id, new.package, new.message, new.file);
END;
CREATE TRIGGER IF NOT EXISTS findings_ad AFTER DELETE ON findings BEGIN
	INSERT INTO findings_fts(findings_fts, rowid, title, cve_id, rule_id, package, message, file)
	VALUES ('delete', old.id, old.title, old.cve_id, old.rule_id, old.package, old.message, old.file);
END;
CREATE TRIGGER IF NOT EXISTS findings_au AFTER UPDATE ON findings BEGIN
	INSERT INTO findings_fts(findings_fts, rowid, title, cve_id, rule_id, package, message, file)
	VALUES ('delete', old.id, old.title, old.cve_id, old.rule_id, old.package, old.message, old.file);
	INSERT INTO findings_fts(rowid, title, cve_id, rule_id, package, message, file)
	VALUES (new.id, new.title, new.cve_id, new.rule_id, new.package, new.message, new.file);
END;

CREATE TABLE IF NOT EXISTS dependencies (
	purl TEXT PRIMARY KEY,                       -- pkg:pypi/flask@3.0.2
	name TEXT NOT NULL,
	version TEXT NOT NULL,
	ecosystem TEXT,
	manifest TEXT,                                -- provenance: which manifest produced it
	retired INTEGER NOT NULL DEFAULT 0,           -- soft-retire (never deleted)
	last_seen TEXT                                 -- last scan that saw this purl
);
CREATE INDEX IF NOT EXISTS idx_deps_retired ON dependencies (retired);

CREATE TABLE IF NOT EXISTS dep_edges (
	from_purl TEXT NOT NULL,
	to_purl TEXT NOT NULL,
	version_range TEXT,
	UNIQUE (from_purl, to_purl)
);
CREATE INDEX IF NOT EXISTS idx_dep_edges_from ON dep_edges (from_purl);
CREATE INDEX IF NOT EXISTS idx_dep_edges_to ON dep_edges (to_purl);
