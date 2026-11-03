package store

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// legacySchema mirrors the 001 schema as shipped before migration 002 (named
// sessions). It lacks the sessions.title column, matching a database opened by
// an older binary.
const legacySchema = `
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    project TEXT NOT NULL,
    directory TEXT NOT NULL,
    started_at TEXT NOT NULL,
    ended_at TEXT,
    summary TEXT,
    status TEXT NOT NULL DEFAULT 'active'
);
CREATE TABLE observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT, type TEXT, title TEXT, content TEXT,
    project TEXT, scope TEXT, topic_key TEXT,
    normalized_hash TEXT, revision_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT, updated_at TEXT, deleted_at TEXT
);
CREATE VIRTUAL TABLE observations_fts USING fts5(title, content, type, project, content='observations', content_rowid='id', tokenize='porter');
CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT);
CREATE TABLE chunks (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER, start_line INTEGER, end_line INTEGER, text TEXT, content_hash TEXT);
CREATE VIRTUAL TABLE chunks_fts USING fts5(text, path UNINDEXED, content='chunks', content_rowid='id', tokenize='trigram');
CREATE TABLE web_cache (id INTEGER PRIMARY KEY AUTOINCREMENT, project TEXT, source TEXT, cache_key TEXT, url TEXT, title TEXT, query TEXT, library_id TEXT, version_tag TEXT, content TEXT, metadata_json TEXT, content_hash TEXT, fetched_at TEXT, expires_at TEXT, session_id TEXT, created_at TEXT, UNIQUE(project, source, cache_key));
CREATE VIRTUAL TABLE web_cache_fts USING fts5(title, content, query, url, source, library_id, content='web_cache', content_rowid='id', tokenize='porter');
`

func rawSQLDB(t *testing.T, dataDir, project string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(dataDir, project+".sqlite"))
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	db.SetMaxOpenConns(1)
	return db
}

// TestUpgradeFromLegacySchema verifies a database created before migration 002
// gains sessions.title exactly once on re-open (the real-world upgrade path
// when an old .sqlite file meets the new binary).
func TestUpgradeFromLegacySchema(t *testing.T) {
	dataDir := t.TempDir()
	db := rawSQLDB(t, dataDir, "legacy")
	if _, err := db.Exec(legacySchema); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	var pre int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'title'`).Scan(&pre); err != nil {
		t.Fatalf("check pre-upgrade schema: %v", err)
	}
	if pre != 0 {
		t.Fatalf("precondition: title column already present in legacy schema")
	}
	// Release the raw handle so Open can apply migrations cleanly.
	db.Close()

	st, err := Open(dataDir, "legacy")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st.Close()
	var post int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('sessions') WHERE name = 'title'`).Scan(&post); err != nil {
		t.Fatalf("check post-upgrade schema: %v", err)
	}
	if post != 1 {
		t.Errorf("expected sessions.title to exist after upgrade, got %d", post)
	}

	// Re-open again: migration 002 must not re-apply (non-idempotent ALTER).
	st2, err := Open(dataDir, "legacy")
	if err != nil {
		t.Fatalf("second re-open: %v", err)
	}
	st2.Close()
}
