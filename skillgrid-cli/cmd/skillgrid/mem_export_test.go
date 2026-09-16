package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedExportProject plants a project store with one observation, one edge, and
// one embedding so `mem export` has all three sources to report.
func seedExportProject(t *testing.T, dataDir, project string) {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open %s: %v", project, err)
	}
	defer st.Close()
	db := st.DB

	if _, err := db.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-exp', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	var fileID int64
	if err := db.QueryRow(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('exp.go', 1, 100, 'h', '2026-01-01T00:00:00Z')
		RETURNING id`).Scan(&fileID); err != nil {
		t.Fatalf("insert file: %v", err)
	}
	var symID int64
	if err := db.QueryRow(`
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		VALUES (?, 'alpha', 'alpha', 'function', 'go', 'func alpha()', 1, 5, 'c1', 'uid-alpha-exp')
		RETURNING id`, fileID).Scan(&symID); err != nil {
		t.Fatalf("insert symbol: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line, valid_from, valid_to)
		VALUES ('calls', ?, ?, ?, 'beta', 'exp.go', 'EXTRACTED', 1, ?, NULL)`,
		symID, fileID, symID, time.Now().Unix()); err != nil {
		t.Fatalf("insert edge: %v", err)
	}
	// 4-dim little-endian float32 vector (1.0, 2.0, 3.0, 4.0).
	vec := []byte{0, 0, 0, 128, 0, 0, 0, 0, 0, 0, 0, 160, 0, 0, 0, 192}
	if _, err := db.Exec(`
		INSERT INTO embeddings (symbol_id, model, dim, vector, updated_at)
		VALUES (?, 'exp-model', 4, ?, '2026-01-02T00:00:00Z')`,
		symID, vec); err != nil {
		t.Fatalf("insert embedding: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO observations (
			session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, created_at, updated_at, source, tool_name,
			owner, visibility, status, retrieval_usage, expires_at, graph_ref
		) VALUES ('sess-exp', 'decision', 'export probe', 'exported observation body', ?,
			'project', NULL, 'h-export', 0, '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z',
			'agent', NULL, 'owner-1', 'private', 'active', 0, NULL, ?)`,
		project, symID); err != nil {
		t.Fatalf("insert observation: %v", err)
	}
}

// TestMemExportCLI is 11.3 [AFK]: `mem export` streams the project bundle to
// stdout as JSON, writes to a file with --file, and omits all embedding data
// with --skip-embeddings.
func TestMemExportCLI(t *testing.T) {
	dataDir := t.TempDir()
	project := "memcli-export"
	seedExportProject(t, dataDir, project)

	// 1) stdout: JSON with all three sources.
	out := runMemCLI(t, dataDir, "export", "--project", project, "--dir", dataDir)
	if !strings.Contains(out, `"format"`) || !strings.Contains(out, "cogx-v1") {
		t.Fatalf("mem export must print the COGX bundle, got: %s", out)
	}
	for _, want := range []string{`"observations"`, `"graph_edges"`, `"embeddings"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("mem export missing %s: %s", want, out)
		}
	}
	if !strings.Contains(out, `"exported observation body"`) {
		t.Fatalf("mem export missing the observation content: %s", out)
	}
	if !strings.Contains(out, `"calls"`) {
		t.Fatalf("mem export missing the edge kind: %s", out)
	}

	// 2) --file: the bundle lands in the file (not stdout).
	outFile := filepath.Join(t.TempDir(), "out.json")
	out = runMemCLI(t, dataDir, "export", "--project", project, "--dir", dataDir, "--file", outFile)
	if strings.Contains(out, `"observations"`) {
		t.Fatalf("mem export --file must not print the bundle to stdout, got: %s", out)
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read export file: %v", err)
	}
	if !strings.Contains(string(data), `"exported observation body"`) {
		t.Fatalf("export file missing the observation: %s", data)
	}
	if !strings.Contains(string(data), `"embeddings"`) {
		t.Fatalf("export file missing embeddings: %s", data)
	}

	// 3) --skip-embeddings: no embedding data anywhere in the payload.
	out = runMemCLI(t, dataDir, "export", "--project", project, "--dir", dataDir, "--skip-embeddings")
	if strings.Contains(out, `"embeddings": [`) && !strings.Contains(out, `"embeddings": []`) {
		// Allow only an empty array; any populated embedding fails this.
		t.Fatalf("mem export --skip-embeddings must not include embedding data: %s", out)
	}
	if strings.Contains(out, `"vector"`) {
		t.Fatalf("mem export --skip-embeddings must omit per-observation vectors: %s", out)
	}
	if !strings.Contains(out, `"exported observation body"`) {
		t.Fatalf("mem export --skip-embeddings still exports observations: %s", out)
	}
}
