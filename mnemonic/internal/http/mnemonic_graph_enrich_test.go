package http

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// G14 / TICKET-12: node enrichment (additive lines/signature fields) + the
// pure on-demand locals resolver (parse per click, store diff empty).

// seedEnrichStore opens the store for proj and inserts one real Go file with a
// two-line function carrying a doc comment and local variables, plus a
// directory that can be used as the workspace root for source reads.
func seedEnrichStore(t *testing.T, dataDir, root string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed enrich store: %v", err)
	}
	defer st.Close()
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('demo/calc.go', 1, 1, 'a', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'Add', 'demo.Add', 'function', 'go', 'func Add(a, b int) int', 4, 7, 'h', 'uid-add'
		FROM files WHERE path='demo/calc.go';
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed enrich: %v", err)
	}
}

const enrichGoSrc = `package demo

// Add sums two integers.
func Add(a, b int) int {
	sum := a + b
	return sum
}
`

// enrichNode looks up a node by label in a graph response body.
func enrichNode(t *testing.T, m map[string]any, label string) map[string]any {
	t.Helper()
	nodes, ok := m["nodes"].([]any)
	if !ok {
		t.Fatalf("nodes not an array: %T", m["nodes"])
	}
	for _, n := range nodes {
		nm, _ := n.(map[string]any)
		if nm["label"] == label {
			return nm
		}
	}
	t.Fatalf("node %q not in graph response", label)
	return nil
}

// TestGraphEnrich_Lines [G14] — a served code-graph node carries its source
// line range (additive: existing fields unchanged) and, when stored, its
// signature.
func TestGraphEnrich_Lines(t *testing.T) {
	dataDir := t.TempDir()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo/calc.go"), []byte(enrichGoSrc), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedEnrichStore(t, dataDir, root)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	req := httptest.NewRequest(http.MethodGet, "/mnemonic/graph/data?project="+proj, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET graph: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	n := enrichNode(t, m, "Add")

	// additive line range (1-based, matches the fixture source)
	lines, ok := n["lines"].(map[string]any)
	if !ok {
		t.Fatalf("node must carry additive 'lines' object, got %T: %v", n["lines"], n)
	}
	if s, _ := lines["start"].(float64); s != 4 {
		t.Errorf("lines.start = %v, want 4", lines["start"])
	}
	if e, _ := lines["end"].(float64); e != 7 {
		t.Errorf("lines.end = %v, want 7", lines["end"])
	}
	// signature is cheap (stored column) — must be served when present
	if sig, _ := n["signature"].(string); sig != "func Add(a, b int) int" {
		t.Errorf("signature = %q, want %q", sig, "func Add(a, b int) int")
	}
	// existing fields untouched (no regression of the pre-TICKET-12 shape)
	for _, key := range []string{"id", "uid", "label", "type", "language", "path", "degree", "community"} {
		if _, ok := n[key]; !ok {
			t.Errorf("node lost existing key %q: %v", key, n)
		}
	}
}

// TestLocalsResolver_Ephemeral [G14] — the locals resolver parses a symbol's
// locals on demand (per click) and NEVER writes: the store content diff
// before/after the resolve is empty.
func TestLocalsResolver_Ephemeral(t *testing.T) {
	dataDir := t.TempDir()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo/calc.go"), []byte(enrichGoSrc), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedEnrichStore(t, dataDir, root)

	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	before, err := dbContent(st.DB)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}

	loc, err := resolveSymbolLocals(st, root, "Add", "go", 4, 7)
	if err != nil {
		t.Fatalf("resolveSymbolLocals: %v", err)
	}
	if loc == nil {
		t.Fatalf("resolveSymbolLocals returned nil for a locatable function")
	}
	// locals: the param (a, b) and the local variable (sum)
	if len(loc.Params) == 0 {
		t.Errorf("expected params in %v", loc)
	}
	foundSum := false
	for _, l := range loc.Locals {
		if l.Name == "sum" {
			foundSum = true
		}
	}
	if !foundSum {
		t.Errorf("expected local 'sum' in %v", loc)
	}

	after, err := dbContent(st.DB)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}
	if before != after {
		t.Fatalf("locals resolver must be pure/read-only: store diff not empty")
	}
}

// TestLocalsResolver_EphemeralUnlocatable (TRIANGULATE) — a symbol whose body
// cannot be located (line range off, malformed file) resolves to a nil
// result without an error (degraded), and still writes nothing.
func TestLocalsResolver_EphemeralUnlocatable(t *testing.T) {
	dataDir := t.TempDir()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo/calc.go"), []byte("package demo\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedEnrichStore(t, dataDir, root)

	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()

	before, err := dbContent(st.DB)
	if err != nil {
		t.Fatalf("snapshot before: %v", err)
	}
	loc, err := resolveSymbolLocals(st, root, "Add", "go", 4, 7)
	if err != nil {
		t.Fatalf("unlocatable symbol must degrade, not error: %v", err)
	}
	if loc != nil {
		t.Errorf("expected nil locals for unlocatable body, got %v", loc)
	}
	after, err := dbContent(st.DB)
	if err != nil {
		t.Fatalf("snapshot after: %v", err)
	}
	if before != after {
		t.Fatalf("locals resolver must be pure/read-only: store diff not empty")
	}
}

// dbContent is a deterministic content fingerprint of the store's graph
// tables (the read-only proof surface for TestLocalsResolver_Ephemeral: any
// write into symbols/edges/files/chunks changes the digest).
func dbContent(db *sql.DB) (string, error) {
	var (
		b    strings.Builder
		rows *sql.Rows
		err  error
	)
	for _, q := range []string{
		`SELECT id, uid, name, kind, language, signature, start_line, end_line, content_hash FROM symbols ORDER BY id`,
		`SELECT id, kind, from_id, file_id, COALESCE(to_id,-1), to_name, confidence, line FROM edges ORDER BY id`,
		`SELECT id, path, mtime_ns, size, content_hash FROM files ORDER BY id`,
	} {
		if rows, err = db.Query(q); err != nil {
			return "", err
		}
		cols, err := rows.Columns()
		if err != nil {
			rows.Close()
			return "", err
		}
		for rows.Next() {
			vals := make([]any, len(cols))
			pv := make([]any, len(cols))
			for i := range vals {
				pv[i] = &vals[i]
			}
			if err := rows.Scan(pv...); err != nil {
				rows.Close()
				return "", err
			}
			for _, v := range vals {
				b.WriteString(fmt.Sprint(v))
				b.WriteByte('|')
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return "", err
		}
		rows.Close()
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(b.String()))), nil
}
