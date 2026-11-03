package process

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

// openStore opens a scratch SQLite store with the 005 symbols/edges tables,
// the 012 communities table, and the 014 process tables — the full surface the
// process pass reads from / writes to.
func openStore(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	stmts := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY AUTOINCREMENT, path TEXT NOT NULL, mtime_ns INTEGER, size INTEGER, content_hash TEXT, indexed_at TEXT)`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_id INTEGER NOT NULL REFERENCES files(id) ON DELETE CASCADE, name TEXT NOT NULL, qualified_name TEXT, kind TEXT NOT NULL, language TEXT, signature TEXT, start_line INTEGER NOT NULL, end_line INTEGER NOT NULL, content_hash TEXT NOT NULL, uid TEXT NOT NULL UNIQUE)`,
		`CREATE TABLE edges (id INTEGER PRIMARY KEY AUTOINCREMENT, kind TEXT NOT NULL, from_id INTEGER NOT NULL REFERENCES symbols(id) ON DELETE CASCADE, file_id INTEGER REFERENCES files(id) ON DELETE CASCADE, to_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE, to_name TEXT, target_path TEXT, confidence TEXT NOT NULL DEFAULT 'EXTRACTED', line INTEGER, UNIQUE(kind, from_id, file_id, to_id, to_name, target_path, line))`,
		`CREATE TABLE communities (id INTEGER, symbol_id INTEGER NOT NULL UNIQUE REFERENCES symbols(id) ON DELETE CASCADE)`,
		`CREATE TABLE processes (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, entry_symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE, entry_kind TEXT, cross_community INTEGER NOT NULL DEFAULT 0, content_hash TEXT NOT NULL, label TEXT NOT NULL DEFAULT '', label_status TEXT NOT NULL DEFAULT 'unlabeled', stop_note TEXT NOT NULL DEFAULT '', updated_at TEXT NOT NULL)`,
		`CREATE TABLE process_steps (process_id INTEGER NOT NULL REFERENCES processes(id) ON DELETE CASCADE, step INTEGER NOT NULL, symbol_id INTEGER REFERENCES symbols(id) ON DELETE CASCADE, confidence TEXT, kind TEXT, PRIMARY KEY (process_id, step))`,
		`CREATE TABLE process_meta_cache (key TEXT PRIMARY KEY, value TEXT)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	return db
}

func seedFile(t *testing.T, db *sql.DB, path string) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES (?, 1, 1, 'h', 'now')`, path)
	if err != nil {
		t.Fatalf("seed file %s: %v", path, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedSymbol(t *testing.T, db *sql.DB, fileID int64, name, kind string, line int) int64 {
	t.Helper()
	res, err := db.Exec(`INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid) VALUES (?, ?, ?, ?, ?, 'h', ?)`,
		fileID, name, kind, line, line, name+kind)
	if err != nil {
		t.Fatalf("seed symbol %s: %v", name, err)
	}
	id, _ := res.LastInsertId()
	return id
}

func seedCall(t *testing.T, db *sql.DB, fromID, toID int64, line int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, to_id, confidence, line) VALUES ('calls', ?, ?, 'EXTRACTED', ?)`, fromID, toID, line); err != nil {
		t.Fatalf("seed edge %d->%d: %v", fromID, toID, err)
	}
}

// seedReference adds a references edge (route -> handler), the 010 entry
// link that the process pass follows.
func seedReference(t *testing.T, db *sql.DB, fromID, toID int64, line int) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO edges (kind, from_id, to_id, confidence, line) VALUES ('references', ?, ?, 'EXTRACTED', ?)`, fromID, toID, line); err != nil {
		t.Fatalf("seed ref %d->%d: %v", fromID, toID, err)
	}
}

// traceFixture builds a chain: entry -> a -> b (all functions). The entry has
// a traceable call chain, so the pass should produce a multi-step flow.
func traceFixture(t *testing.T) *sql.DB {
	t.Helper()
	db := openStore(t)
	f := seedFile(t, db, "app/server.go")
	entry := seedSymbol(t, db, f, "handleListUsers", "function", 1)
	a := seedSymbol(t, db, f, "loadUsers", "function", 10)
	b := seedSymbol(t, db, f, "queryDB", "function", 20)
	seedCall(t, db, entry, a, 2)
	seedCall(t, db, a, b, 11)
	_ = entry
	return db
}

// TestProcessTrace covers 02.2 (Scenario: Process list returns precomputed
// flows from entry points): seeded from an entry point, the pass traces
// through the 005 call edges into processes + process_steps; each flow has
// named steps and a content hash; and code_processes (List) returns the
// complete flow in one call without per-query traversal.
func TestProcessTrace(t *testing.T) {
	db := traceFixture(t)
	f := db
	var entryID int64
	if err := f.QueryRow(`SELECT id FROM symbols WHERE name = 'handleListUsers'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup entry: %v", err)
	}
	entries := []Entry{{SymbolID: entryID, Kind: "cli-main"}}

	res, err := Run(context.Background(), db, entries, stubLLM{label: "List users flow"}, RunOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(res.Processes) != 1 {
		t.Fatalf("expected 1 process, got %d", len(res.Processes))
	}
	p := res.Processes[0]
	// Named steps: the entry plus the two callees (a depth of 2 is within the
	// default cap).
	if len(p.Steps) != 3 {
		t.Fatalf("expected 3 steps (entry + 2 callees), got %d: %+v", len(p.Steps), p.Steps)
	}
	for i, s := range p.Steps {
		if s.Name == "" {
			t.Errorf("step %d has an empty name (not named): %+v", i, s)
		}
	}
	if p.ContentHash == "" {
		t.Errorf("process has no content hash")
	}

	// Persisted: List (the code_processes backing) returns the flow in one
	// call, with its steps — no per-query traversal.
	listed, err := List(db)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("List returned %d processes, want 1", len(listed))
	}
	// Rebuild the full trace via Get (the code_process <name> backing).
	got, err := Get(db, listed[0].Name)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(got.Steps) != 3 {
		t.Fatalf("Get returned %d steps, want 3: %+v", len(got.Steps), got.Steps)
	}
	// The persisted row is present in processes + process_steps.
	var procCount, stepCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM processes`).Scan(&procCount); err != nil {
		t.Fatalf("count processes: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM process_steps`).Scan(&stepCount); err != nil {
		t.Fatalf("count steps: %v", err)
	}
	if procCount != 1 || stepCount != 3 {
		t.Errorf("persisted processes=%d steps=%d, want 1/3", procCount, stepCount)
	}
}

// TestProcessDeterministic covers the determinism constraint: two Runs on the
// same graph yield identical flow structure (stable ordering, same content
// hash), and the second is served from cache without re-tracing.
func TestProcessDeterministic(t *testing.T) {
	db := traceFixture(t)
	var entryID int64
	if err := db.QueryRow(`SELECT id FROM symbols WHERE name = 'handleListUsers'`).Scan(&entryID); err != nil {
		t.Fatalf("lookup: %v", err)
	}
	entries := []Entry{{SymbolID: entryID, Kind: "cli-main"}}

	first, err := Run(context.Background(), db, entries, stubLLM{label: "L"}, RunOptions{})
	if err != nil {
		t.Fatalf("Run 1: %v", err)
	}
	second, err := Run(context.Background(), db, entries, stubLLM{label: "L2"}, RunOptions{})
	if err != nil {
		t.Fatalf("Run 2: %v", err)
	}
	if !second.FromCache {
		t.Errorf("second Run should be served from cache (content-hash hit)")
	}
	if len(first.Processes) != 1 || len(second.Processes) != 1 {
		t.Fatalf("expected 1 process each, got %d/%d", len(first.Processes), len(second.Processes))
	}
	a, b := first.Processes[0], second.Processes[0]
	if a.ContentHash != b.ContentHash {
		t.Errorf("content hash not stable across runs: %q vs %q", a.ContentHash, b.ContentHash)
	}
	if len(a.Steps) != len(b.Steps) {
		t.Errorf("step count differs: %d vs %d", len(a.Steps), len(b.Steps))
	}
	for i := range a.Steps {
		if a.Steps[i].SymbolID != b.Steps[i].SymbolID || a.Steps[i].Name != b.Steps[i].Name {
			t.Errorf("step %d differs: %+v vs %+v", i, a.Steps[i], b.Steps[i])
		}
	}
}
