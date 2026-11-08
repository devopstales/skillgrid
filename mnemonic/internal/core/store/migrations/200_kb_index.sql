CREATE TABLE IF NOT EXISTS kb_pages (
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

CREATE TABLE IF NOT EXISTS kb_claims (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	page_id INTEGER NOT NULL,
	subject TEXT NOT NULL,
	literal TEXT NOT NULL,
	anchor_path TEXT NOT NULL,
	anchor_selector TEXT NOT NULL DEFAULT '{}',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kb_run_state (
	project TEXT PRIMARY KEY,
	run_id TEXT NOT NULL,
	phase TEXT NOT NULL DEFAULT 'idle',
	source_fingerprint TEXT NOT NULL DEFAULT '',
	processed TEXT NOT NULL DEFAULT '{}',
	plan TEXT NOT NULL DEFAULT '{}',
	last_full_run_at TEXT,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kb_page_baselines (
	project TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	source_fingerprint TEXT NOT NULL,
	page_version TEXT NOT NULL,
	completed_by TEXT NOT NULL,
	completed_run_id TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (project, rel_path)
);

CREATE TABLE IF NOT EXISTS kb_events (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	project TEXT NOT NULL,
	event_type TEXT NOT NULL,
	subject_kind TEXT NOT NULL,
	rel_path TEXT NOT NULL,
	event_key TEXT NOT NULL,
	occurred_at TEXT NOT NULL,
	UNIQUE (project, event_key)
);

CREATE VIRTUAL TABLE IF NOT EXISTS kb_pages_fts USING fts5(
	title, body, type, rel_path,
	tokenize='trigram',
	content='kb_pages',
	content_rowid='id'
);

CREATE TRIGGER kb_pages_fts_insert AFTER INSERT ON kb_pages BEGIN
	INSERT INTO kb_pages_fts(rowid, title, body, type, rel_path)
	VALUES (new.id, new.title, new.body, new.type, new.rel_path);
END;

CREATE TRIGGER kb_pages_fts_delete BEFORE DELETE ON kb_pages BEGIN
	DELETE FROM kb_pages_fts WHERE rowid = old.id;
END;

CREATE TRIGGER kb_pages_fts_update BEFORE UPDATE ON kb_pages BEGIN
	DELETE FROM kb_pages_fts WHERE rowid = old.id;
END;

CREATE TRIGGER kb_pages_fts_update_insert AFTER UPDATE ON kb_pages BEGIN
	INSERT INTO kb_pages_fts(rowid, title, body, type, rel_path)
	VALUES (new.id, new.title, new.body, new.type, new.rel_path);
END;
