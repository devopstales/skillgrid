package search

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestDB builds a minimal store with the files/chunks/chunks_fts tables that
// AdrSearch depends on, and indexes the given ADR bodies by path.
func newTestDB(t *testing.T, files map[string]string) *sql.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL UNIQUE, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`,
		`CREATE TABLE IF NOT EXISTS chunks (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER, start_line INTEGER, end_line INTEGER, text TEXT, content_hash TEXT)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(text, path UNINDEXED, content='chunks', content_rowid='id', tokenize='trigram')`,
		`CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN INSERT INTO chunks_fts(rowid, text) VALUES (new.id, new.text); END;`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
	for p, body := range files {
		if _, err := db.Exec(`INSERT INTO files (path) VALUES (?)`, p); err != nil {
			t.Fatalf("insert file %s: %v", p, err)
		}
		var fileID int64
		if err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, p).Scan(&fileID); err != nil {
			t.Fatalf("select file id: %v", err)
		}
		if _, err := db.Exec(`INSERT INTO chunks (file_id, start_line, end_line, text) VALUES (?,?,?,?)`, fileID, 1, 200, body); err != nil {
			t.Fatalf("insert chunk %s: %v", p, err)
		}
	}
	return db
}

const adr0019 = `# Locked decisions are ADR files; ASSUMPTIONS.md holds the path

---
status: "accepted"
supersedes: none
date: 2026-10-02
---

## Context

One file per locked decision; the index carries none of status/supersedes/date.
`

const adr0020 = `# Superseding decision

---
status: "accepted"
supersedes: 0019
date: 2026-10-05
---

## Context

This supersedes the earlier decision about ADR files.
`

func TestAdrSearchReturnsStructuredHits(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/decisions/04-adr-0019-decisions-are-files.md": adr0019,
	})
	hits, err := AdrSearch(db, "decision", 10)
	if err != nil {
		t.Fatalf("adr search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	h := hits[0]
	if h.Path != ".skillgrid/decisions/04-adr-0019-decisions-are-files.md" {
		t.Errorf("path = %q", h.Path)
	}
	if h.Status != "accepted" {
		t.Errorf("status = %q, want accepted", h.Status)
	}
	if h.Date != "2026-10-02" {
		t.Errorf("date = %q, want 2026-10-02", h.Date)
	}
	if h.Title != "Locked decisions are ADR files; ASSUMPTIONS.md holds the path" {
		t.Errorf("title = %q", h.Title)
	}
	if h.Supersedes != "" {
		t.Errorf("supersedes = %q, want empty (none)", h.Supersedes)
	}
	if h.SupersededBy != "" {
		t.Errorf("supersededBy = %q, want empty", h.SupersededBy)
	}
}

func TestAdrSearchFiltersNonAdrFiles(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/decisions/04-adr-0019-decisions-are-files.md": adr0019,
		".skillgrid/README.md":                                   "# Readme\n\ndecision text here\n",
		"src/main.go":                                            "package main\n\ndecision decision\n",
	})
	hits, err := AdrSearch(db, "decision", 10)
	if err != nil {
		t.Fatalf("adr search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected only the ADR file to match, got %d hits: %v", len(hits), paths(hits))
	}
}

func TestAdrSearchSupersedesChain(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/decisions/04-adr-0019-decisions-are-files.md": adr0019,
		".skillgrid/decisions/04-adr-0020-superseding-decision.md": adr0020,
	})
	hits, err := AdrSearch(db, "decision", 10)
	if err != nil {
		t.Fatalf("adr search: %v", err)
	}
	byNum := map[string]AdrHit{}
	for _, h := range hits {
		base := filepath.Base(h.Path)
		if m := adrNum.FindStringSubmatch(base); m != nil {
			byNum[m[1]] = h
		}
	}
	h19, ok19 := byNum["0019"]
	h20, ok20 := byNum["0020"]
	if !ok19 || !ok20 {
		t.Fatalf("expected both 0019 and 0020, got %v", paths(hits))
	}
	if h20.Supersedes != "0019" {
		t.Errorf("0020 supersedes = %q, want 0019", h20.Supersedes)
	}
	if h19.SupersededBy != ".skillgrid/decisions/04-adr-0020-superseding-decision.md" {
		t.Errorf("0019 supersededBy = %q, want the 0020 path", h19.SupersededBy)
	}
}

func TestAdrSearchEmptyQuery(t *testing.T) {
	db := newTestDB(t, map[string]string{
		".skillgrid/decisions/04-adr-0019-decisions-are-files.md": adr0019,
	})
	hits, err := AdrSearch(db, "   ", 10)
	if err != nil {
		t.Fatalf("adr search: %v", err)
	}
	if len(hits) != 0 {
		t.Errorf("expected no hits for an empty query, got %d", len(hits))
	}
}

func paths(hits []AdrHit) []string {
	var out []string
	for _, h := range hits {
		out = append(out, h.Path)
	}
	return out
}

// keep the os import used by future fixtures if needed (currently unused guard).
var _ = os.PathSeparator
