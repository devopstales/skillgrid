package main

import (
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedGraphProject plants one project store with a file, two symbols, and three
// edges in distinct temporal states (active, expired, pending) so `mem graph`
// has something to report.
func seedGraphProject(t *testing.T, dataDir, project string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer st.Close()
	db := st.DB

	var fileID int64
	if err := db.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('graph.go', 1, 100, 'h', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var fromID, toID int64
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, 'alpha', 'function', 1, 5, 'c1', 'uid-alpha')
		RETURNING id`, fileID).Scan(&fromID); err != nil {
		t.Fatalf("insert from: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, kind, start_line, end_line, content_hash, uid)
		VALUES (?, 'beta', 'function', 6, 10, 'c2', 'uid-beta')
		RETURNING id`, fileID).Scan(&toID); err != nil {
		t.Fatalf("insert to: %v", err)
	}
	now := time.Now().Unix()
	ins := func(kind string, validFrom, validTo int64) {
		t.Helper()
		var vt interface{}
		if validTo != 0 {
			vt = validTo
		}
		if _, err := db.Exec(`
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
			VALUES (?, ?, ?, ?, 'beta', NULL, 'EXTRACTED', 1, ?, ?)`,
			kind, fromID, fileID, toID, validFrom, vt); err != nil {
			t.Fatalf("insert edge %s: %v", kind, err)
		}
	}
	ins("calls", now-10000, 0)          // active
	ins("imports", now-10000, now-100)  // expired
	ins("references", now+10000, 0)     // pending
}

// TestMemGraphTemporalStatus is 10.3 [AFK] — `skillgrid mem graph` lists the
// project's codeindex edges with their temporal status (active/expired/
// pending) and the raw valid_from / valid_to fields.
func TestMemGraphTemporalStatus(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-graph"
	seedGraphProject(t, dataDir, project)

	out := runMemCLI(t, dataDir, "graph", "--project", project, "--dir", dataDir)

	// The CLI reports all three edges (history view) with a per-status count.
	if !strings.Contains(out, `"count": 3`) {
		t.Fatalf("mem graph must report count 3: %s", out)
	}
	for _, st := range []string{`"active": 1`, `"expired": 1`, `"pending": 1`} {
		if !strings.Contains(out, st) {
			t.Fatalf("mem graph missing status count %s: %s", st, out)
		}
	}
	// The raw temporal fields are present (Go field names ValidFrom/ValidTo).
	if !strings.Contains(out, `"ValidFrom"`) || !strings.Contains(out, `"ValidTo"`) {
		t.Fatalf("mem graph must include ValidFrom and ValidTo: %s", out)
	}
	// Each temporal state is labeled on its edge.
	for _, st := range []string{`"active"`, `"expired"`, `"pending"`} {
		if !strings.Contains(out, st) {
			t.Fatalf("mem graph must label edges with %s: %s", st, out)
		}
	}
}
