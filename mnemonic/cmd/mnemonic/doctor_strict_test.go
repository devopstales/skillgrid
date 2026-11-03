package main

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// openStrictStore opens an in-memory store with the chunks schema (the 005
// tables the doctor --strict redaction check reads) for doctor strict checks.
func openStrictStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`); err != nil {
		t.Fatalf("create files: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE chunks (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL, start_line INTEGER, end_line INTEGER, text TEXT, content_hash TEXT)`); err != nil {
		t.Fatalf("create chunks: %v", err)
	}
	return db
}

// TestDoctorStrictRedactionViolation covers @step-01 (Scenario: Doctor strict
// exits non-zero on redaction violation): an indexed chunk containing a
// secret-like pattern is a redaction violation — the strict report flags the
// file and the overall check is not clean.
func TestDoctorStrictRedactionViolation(t *testing.T) {
	db := openStrictStore(t)
	if _, err := db.Exec(`INSERT INTO files (path, indexed_at) VALUES ('auth.go', ?)`, "now"); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fileID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path='auth.go'`).Scan(&fileID); err != nil {
		t.Fatalf("file id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text) VALUES (?, 1, 3, ?)`,
		fileID, "const k = \"AKIAIOSFODNN7EXAMPLE\"\nfunc Load() string { return k }\n"); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	rep := runDoctorStrictChecks(db, time.Hour)
	if !rep.Redaction.Violation {
		t.Errorf("an indexed secret must be a redaction violation, got %+v", rep.Redaction)
	}
	if len(rep.Redaction.Files) == 0 {
		t.Errorf("redaction violation must name the offending file, got %+v", rep.Redaction)
	}
}

// TestDoctorStrictClean covers the clean side: no secrets + a fresh index → the
// strict report is clean (no redaction or freshness violation), so a non-strict
// run's exit is 0.
func TestDoctorStrictClean(t *testing.T) {
	db := openStrictStore(t)
	if _, err := db.Exec(`INSERT INTO files (path, indexed_at) VALUES ('client.go', ?)`, time.Now().UTC().Format(time.RFC3339)); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fileID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path='client.go'`).Scan(&fileID); err != nil {
		t.Fatalf("file id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text) VALUES (?, 1, 3, ?)`,
		fileID, "func New() *C { return nil }\nfunc (c *C) Get(u string) error { return nil }\n"); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	rep := runDoctorStrictChecks(db, time.Hour)
	if rep.Redaction.Violation {
		t.Errorf("a clean index must not be a redaction violation, got %+v", rep.Redaction)
	}
	if rep.Freshness.Violation {
		t.Errorf("a fresh index must not be a freshness violation, got %+v", rep.Freshness)
	}
	if !rep.Clean() {
		t.Errorf("a clean index must report clean, got %+v", rep)
	}
}

// TestDoctorStrictFreshnessViolation covers @step-01 (Scenario: Strict doctor
// reports redaction and freshness state): an index older than the max age is a
// freshness violation.
func TestDoctorStrictFreshnessViolation(t *testing.T) {
	db := openStrictStore(t)
	old := time.Now().UTC().Add(-48 * time.Hour).Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO files (path, indexed_at) VALUES ('a.go', ?)`, old); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fileID int64
	if err := db.QueryRow(`SELECT id FROM files WHERE path='a.go'`).Scan(&fileID); err != nil {
		t.Fatalf("file id: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text) VALUES (?, 1, 1, 'func a() {}\n')`, fileID); err != nil {
		t.Fatalf("insert chunk: %v", err)
	}

	rep := runDoctorStrictChecks(db, time.Hour)
	if !rep.Freshness.Violation {
		t.Errorf("an index older than the max age must be a freshness violation, got %+v", rep.Freshness)
	}
}

// TestDoctorStrictEmptyIndex covers the edge: an empty index (no files) is not
// a redaction violation (no secrets) and not a freshness violation (nothing to
// be stale about) — it is clean.
func TestDoctorStrictEmptyIndex(t *testing.T) {
	db := openStrictStore(t)
	rep := runDoctorStrictChecks(db, time.Hour)
	if !rep.Clean() {
		t.Errorf("an empty index must be clean, got %+v", rep)
	}
}
