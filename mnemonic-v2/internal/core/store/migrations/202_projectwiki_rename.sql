-- 202: full rebrand kb → projectwiki (2026-11-07). Copies the 200/201 kb_* DDL
-- (kb_run_state incl. the 201 mode column) to projectwiki_* with the data, then
-- drops the legacy tables. No IF NOT EXISTS: fail loud on a hand-crafted store.
-- Applies only after 200+201, so kb_* always exists when this runs.

CREATE TABLE projectwiki_pages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	sync_id TEXT NOT NULL,
	project TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	title TEXT NOT NULL,
	type TEXT NOT NULL,
	status TEXT NOT NULL DEFAULT 'draft',
	aliases TEXT NOT NULL DEFAULT '[]',
	in_degree INTEGER NOT NULL DEFAULT 0,
	body TEXT NOT NULL,
	body_hash TEXT NOT NULL,
	anchor_count INTEGER NOT NULL DEFAULT 0,
	claim_count INTEGER NOT NULL DEFAULT 0,
	source_count INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL,
	deleted_at TEXT,
	UNIQUE (project, rel_path)
);

CREATE TABLE projectwiki_claims (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	page_id INTEGER NOT NULL,
	subject TEXT NOT NULL,
	literal TEXT NOT NULL,
	anchor_path TEXT NOT NULL,
	anchor_selector TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL
);

CREATE TABLE projectwiki_run_state (
	project TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	phase TEXT NOT NULL DEFAULT 'idle',
	source_fingerprint TEXT NOT NULL DEFAULT '',
	processed TEXT NOT NULL DEFAULT '{}',
	plan TEXT NOT NULL DEFAULT '{}',
	last_full_run_at TEXT,
	updated_at TEXT NOT NULL,
	mode TEXT NOT NULL DEFAULT ''
);

CREATE TABLE projectwiki_page_baselines (
	project TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	source_fingerprint TEXT NOT NULL,
	page_version TEXT NOT NULL,
	completed_by TEXT NOT NULL,
	completed_run_id TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (project, rel_path)
);

CREATE TABLE projectwiki_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project TEXT NOT NULL,
	event_type TEXT NOT NULL,
	subject_kind TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	event_key TEXT NOT NULL,
	occurred_at TEXT NOT NULL,
	UNIQUE (project, event_key)
);

INSERT INTO projectwiki_pages (id, sync_id, project, rel_path, title, type, status, aliases, in_degree, body, body_hash, anchor_count, claim_count, source_count, updated_at, deleted_at)
SELECT id, sync_id, project, rel_path, title, type, status, aliases, in_degree, body, body_hash, anchor_count, claim_count, source_count, updated_at, deleted_at FROM kb_pages;

INSERT INTO projectwiki_claims (id, page_id, subject, literal, anchor_path, anchor_selector, created_at)
SELECT id, page_id, subject, literal, anchor_path, anchor_selector, created_at FROM kb_claims;

INSERT INTO projectwiki_run_state (project, run_id, phase, source_fingerprint, processed, plan, last_full_run_at, updated_at, mode)
SELECT project, run_id, phase, source_fingerprint, processed, plan, last_full_run_at, updated_at, mode FROM kb_run_state;

INSERT INTO projectwiki_page_baselines (project, rel_path, source_fingerprint, page_version, completed_by, completed_run_id, updated_at)
SELECT project, rel_path, source_fingerprint, page_version, completed_by, completed_run_id, updated_at FROM kb_page_baselines;

INSERT INTO projectwiki_events (id, project, event_type, subject_kind, rel_path, event_key, occurred_at)
SELECT id, project, event_type, subject_kind, rel_path, event_key, occurred_at FROM kb_events;

DROP TABLE kb_pages;
DROP TABLE kb_claims;
DROP TABLE kb_run_state;
DROP TABLE kb_page_baselines;
DROP TABLE kb_events;
DROP TABLE kb_pages_fts;

CREATE VIRTUAL TABLE projectwiki_pages_fts USING fts5(
	title, body, type, rel_path,
	tokenize='trigram',
	content='projectwiki_pages',
	content_rowid='id'
);

CREATE TRIGGER projectwiki_pages_fts_insert AFTER INSERT ON projectwiki_pages BEGIN
	INSERT INTO projectwiki_pages_fts(rowid, title, body, type, rel_path)
	VALUES (new.id, new.title, new.body, new.type, new.rel_path);
END;

CREATE TRIGGER projectwiki_pages_fts_delete BEFORE DELETE ON projectwiki_pages BEGIN
	DELETE FROM projectwiki_pages_fts WHERE rowid = old.id;
END;

CREATE TRIGGER projectwiki_pages_fts_update BEFORE UPDATE ON projectwiki_pages BEGIN
	DELETE FROM projectwiki_pages_fts WHERE rowid = old.id;
END;

CREATE TRIGGER projectwiki_pages_fts_update_insert AFTER UPDATE ON projectwiki_pages BEGIN
	INSERT INTO projectwiki_pages_fts(rowid, title, body, type, rel_path)
	VALUES (new.id, new.title, new.body, new.type, new.rel_path);
END;

INSERT INTO projectwiki_pages_fts(projectwiki_pages_fts) VALUES('rebuild');
