package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// seedGraphStore opens the store for proj and inserts a small code graph:
// files + symbols + directed 'calls' edges forming a chain
//  (root → mid1 → mid2 → leaf) plus a sibling branch (root → sib).
// The chain is 3 hops deep so depth=2 from root must truncate.
func seedGraphStore(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("seed graph store: %v", err)
	}
	defer st.Close()
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('/x/root.go', 1, 1, 'a', 'now'),
			('/x/mid1.go', 2, 2, 'b', 'now'),
			('/x/mid2.go', 3, 3, 'c', 'now'),
			('/x/leaf.go', 4, 4, 'd', 'now'),
			('/x/sib.go', 5, 5, 'e', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'root', 'root', 'function', 'go', 'func root()', 1, 5, 'h', 'uid-root' FROM files WHERE path='/x/root.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'mid1', 'mid1', 'function', 'go', 'func mid1()', 1, 5, 'h', 'uid-mid1' FROM files WHERE path='/x/mid1.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'mid2', 'mid2', 'function', 'go', 'func mid2()', 1, 5, 'h', 'uid-mid2' FROM files WHERE path='/x/mid2.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'leaf', 'leaf', 'function', 'go', 'func leaf()', 1, 5, 'h', 'uid-leaf' FROM files WHERE path='/x/leaf.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'sib', 'sib', 'function', 'go', 'func sib()', 1, 5, 'h', 'uid-sib' FROM files WHERE path='/x/sib.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls', (SELECT id FROM symbols WHERE uid='uid-root'), (SELECT file_id FROM symbols WHERE uid='uid-root'), (SELECT id FROM symbols WHERE uid='uid-mid1'), 'mid1', 'EXTRACTED', 10;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls', (SELECT id FROM symbols WHERE uid='uid-mid1'), (SELECT file_id FROM symbols WHERE uid='uid-mid1'), (SELECT id FROM symbols WHERE uid='uid-mid2'), 'mid2', 'EXTRACTED', 11;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls', (SELECT id FROM symbols WHERE uid='uid-mid2'), (SELECT file_id FROM symbols WHERE uid='uid-mid2'), (SELECT id FROM symbols WHERE uid='uid-leaf'), 'leaf', 'EXTRACTED', 12;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls', (SELECT id FROM symbols WHERE uid='uid-root'), (SELECT file_id FROM symbols WHERE uid='uid-root'), (SELECT id FROM symbols WHERE uid='uid-sib'), 'sib', 'EXTRACTED', 13;
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed graph: %v", err)
	}
}

// getJSON is a small helper to GET and decode a JSON body.
func getJSON(t *testing.T, h http.Handler, target string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET %s: expected 200, got %d (%s)", target, rr.Code, rr.Body.String())
	}
	var m map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &m); err != nil {
		t.Fatalf("GET %s: decode: %v", target, err)
	}
	return m
}

// nodeNames returns the set of node labels in a graph response.
func nodeNames(t *testing.T, m map[string]any) map[string]bool {
	t.Helper()
	nodes, ok := m["nodes"].([]any)
	if !ok {
		t.Fatalf("nodes not an array: %T", m["nodes"])
	}
	out := map[string]bool{}
	for _, n := range nodes {
		nm, _ := n.(map[string]any)
		if lbl, ok := nm["label"].(string); ok {
			out[lbl] = true
		}
	}
	return out
}

func edgeCount(m map[string]any) int {
	edges, _ := m["edges"].([]any)
	return len(edges)
}

// 4.1 [RED] Threat: Graph scale — depth filter + node cap + truncated flag.
func TestPhase4_GraphScale(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedGraphStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	// the chain root→mid1→mid2→leaf is 3 hops; depth=2 from root must NOT
	// include 'leaf' and must set truncated=true. node_id is an int symbol id,
	// so resolve root's id via /nodes first.
	nodes := getJSON(t, h, "/mnemonic/graph/nodes?project="+proj)
	rootID := nodeIDByLabel(t, nodes, "root")

	m := getJSON(t, h, "/mnemonic/graph?project="+proj+"&node_id="+itoaStr(rootID)+"&depth=2")
	names := nodeNames(t, m)
	if names["leaf"] {
		t.Errorf("depth=2 from root should exclude 'leaf' (3 hops away), got %v", names)
	}
	if !names["mid1"] || !names["mid2"] {
		t.Errorf("depth=2 from root should include mid1 and mid2, got %v", names)
	}
	if b, _ := m["truncated"].(bool); !b {
		t.Errorf("depth=2 from root should be truncated (leaf dropped), got truncated=%v", m["truncated"])
	}

	// /nodes?limit=2 caps at 2 nodes and sets truncated=true
	capped := getJSON(t, h, "/mnemonic/graph/nodes?project="+proj+"&limit=2")
	cnodes, _ := capped["nodes"].([]any)
	if len(cnodes) != 2 {
		t.Errorf("limit=2 should return exactly 2 nodes, got %d", len(cnodes))
	}
	if b, _ := capped["truncated"].(bool); !b {
		t.Errorf("limit=2 should set truncated=true (5 nodes exist), got %v", capped["truncated"])
	}
}

// 4.2 [AFK] /mnemonic/graph happy path returns nodes (label/type/path/degree/community)
// + edges (source/target/type/weight/directed).
func TestPhase4_GraphHappy(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedGraphStore(t, dataDir)
	svc := service.New(dataDir)
	h := NewServer(svc).Handler()

	m := getJSON(t, h, "/mnemonic/graph?project="+proj)
	names := nodeNames(t, m)
	for _, want := range []string{"root", "mid1", "mid2", "leaf", "sib"} {
		if !names[want] {
			t.Errorf("full graph missing node %q, got %v", want, names)
		}
	}
	// 4 edges seeded
	if n := edgeCount(m); n != 4 {
		t.Errorf("full graph should have 4 edges, got %d", n)
	}
	// node shape: check one node has the expected keys
	nodes := m["nodes"].([]any)
	first, _ := nodes[0].(map[string]any)
	for _, key := range []string{"id", "label", "type", "path", "degree", "community"} {
		if _, ok := first[key]; !ok {
			t.Errorf("node missing key %q: %v", key, first)
		}
	}
	// edge shape
	if n := edgeCount(m); n > 0 {
		edges := m["edges"].([]any)
		e, _ := edges[0].(map[string]any)
		for _, key := range []string{"id", "source", "target", "type", "weight", "directed"} {
			if _, ok := e[key]; !ok {
				t.Errorf("edge missing key %q: %v", key, e)
			}
		}
	}
	// degraded must be false when edges are present
	if b, _ := m["degraded"].(bool); b {
		t.Errorf("graph with edges should not be degraded, got %v", m)
	}
}

// 4.3 [RED] Threat: 005/008 soft dep — graph degrades to node-only + file-list
// fallback when edge/community data is absent.
func TestPhase4_GraphDegraded(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	// seed symbols + files but NO edges
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('/p/a.go', 1, 1, 'a', 'now'),
			('/p/b.go', 2, 2, 'b', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'alpha', 'alpha', 'function', 'go', 'func alpha()', 1, 5, 'h', 'uid-alpha' FROM files WHERE path='/p/a.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT id, 'beta', 'beta', 'function', 'go', 'func beta()', 1, 5, 'h', 'uid-beta' FROM files WHERE path='/p/b.go';
	`
	if _, err := st.DB.Exec(seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	st.Close()

	svc := service.New(dataDir)
	h := NewServer(svc).Handler()
	m := getJSON(t, h, "/mnemonic/graph?project="+proj)
	if b, _ := m["degraded"].(bool); !b {
		t.Errorf("graph with no edges should be degraded, got %v", m["degraded"])
	}
	if n := edgeCount(m); n != 0 {
		t.Errorf("degraded graph should have 0 edges, got %d", n)
	}
	// nodes still present
	if len(nodeNames(t, m)) != 2 {
		t.Errorf("degraded graph should still list its 2 nodes, got %v", nodeNames(t, m))
	}
	// file-list fallback present
	files, ok := m["files"].([]any)
	if !ok || len(files) != 2 {
		t.Errorf("degraded graph should carry a 2-item file list, got %v", m["files"])
	}
}

// 4.8 [AFK] openapi.yaml documents the graph routes.
func TestPhase4_OpenAPI(t *testing.T) {
	here, _ := os.Getwd()
	openapi := filepath.Join(here, "ui", "openapi.yaml")
	data, err := os.ReadFile(openapi)
	if err != nil {
		t.Skipf("openapi.yaml not found at %s: %v", openapi, err)
	}
	text := string(data)
	for _, want := range []string{"/mnemonic/graph", "/mnemonic/graph/nodes"} {
		if !strings.Contains(text, want) {
			t.Errorf("openapi.yaml missing graph route %s", want)
		}
	}
}

// nodeIDByLabel resolves a node id from a /nodes response by label.
func nodeIDByLabel(t *testing.T, nodes map[string]any, label string) int64 {
	t.Helper()
	arr, _ := nodes["nodes"].([]any)
	for _, n := range arr {
		nm, _ := n.(map[string]any)
		if lbl, _ := nm["label"].(string); lbl == label {
			if id, ok := nm["id"].(float64); ok {
				return int64(id)
			}
		}
	}
	t.Fatalf("node %q not found in /nodes", label)
	return 0
}

// itoaStr is a tiny int64→string for the test file.
func itoaStr(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
