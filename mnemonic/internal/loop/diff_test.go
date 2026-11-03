package loop

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDiffImpactCallers(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	stmts := []string{
		`CREATE TABLE files (id INTEGER PRIMARY KEY, path TEXT)`,
		`CREATE TABLE symbols (id INTEGER PRIMARY KEY, file_id INTEGER, name TEXT, start_line INTEGER)`,
		`CREATE TABLE edges (id INTEGER PRIMARY KEY, from_id INTEGER, to_id INTEGER, kind TEXT)`,
		`INSERT INTO files (id, path) VALUES (1, 'a.go')`,
		`INSERT INTO symbols (id, file_id, name, start_line) VALUES (10, 1, 'Run', 1)`,
		`INSERT INTO edges (id, from_id, to_id, kind) VALUES (1, 11, 10, 'calls'), (2, 12, 10, 'call')`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	lines, err := DiffImpact(context.Background(), db, []string{"a.go", "missing.go"})
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0].Symbol != "Run" || lines[0].Callers != 2 {
		t.Fatalf("impact = %+v", lines)
	}
	if lines[1].Symbol != "" {
		t.Fatalf("missing file got symbol %q", lines[1].Symbol)
	}
	text := strings.Join(FormatImpact(lines), "\n")
	if !strings.Contains(text, "a.go Run callers: 2") || !strings.Contains(text, "not indexed") {
		t.Fatalf("format = %s", text)
	}
}
