package http

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// writeKnowledgeBundle writes a minimal concept bundle (two linked concepts
// + index.md/log.md catalogs) under root for the /knowledge/graph route tests.
func writeKnowledgeBundle(t *testing.T, root string) {
	t.Helper()
	files := map[string]string{
		"index.md":         "# Index\n",
		"log.md":           "# Log\n",
		"decisions/d1.md":  "---\ntype: decision\ntitle: D-1\nstatus: active\n---\n# D-1\n\n[see](../invariants/i1.md)\n",
		"invariants/i1.md": "---\ntype: invariant\ntitle: I-1\nstatus: deprecated\n---\n# I-1\n\ntext\n",
	}
	for path, content := range files {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
}

// TestKnowledgeGraphRoute covers @step-10 knowledge-graph-view (G11) at the
// HTTP surface: GET /knowledge/graph is an open read (no auth) that derives
// the concept graph per call from the project's current bundle — nodes carry
// frontmatter + flags, edges only for resolving links, catalogs excluded,
// backlinks present. The same route re-derived after a bundle edit reflects
// the edit (per-call, never stored).
func TestKnowledgeGraphRoute(t *testing.T) {
	workDir := t.TempDir()
	t.Chdir(workDir)
	dataDir := t.TempDir()
	t.Setenv("SKILLGRID_MNEMONIC_DATA_DIR", dataDir)
	seedStore(t, dataDir)
	s := NewServer(service.New(dataDir))
	h := s.Handler()

	// Empty bundle (the cwd has no knowledge zone): 200 with an empty graph,
	// not an error.
	rr, out := do(t, h, http.MethodGet, "/knowledge/graph?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("empty bundle: status %d, want 200: %s", rr.Code, rr.Body.String())
	}
	if n, _ := out["nodes"].([]any); len(n) != 0 {
		t.Errorf("empty bundle: nodes = %d, want 0", len(n))
	}
	if _, ok := out["backlinks"]; !ok {
		t.Errorf("empty bundle: backlinks key missing: %v", out)
	}

	// The route is an OPEN read: a client with no bearer token gets the
	// graph, not 401.
	rr2, _ := do(t, h, http.MethodGet, "/knowledge/graph?project="+proj, nil)
	if rr2.Code == http.StatusUnauthorized {
		t.Errorf("route is open read, got 401")
	}

	// Missing project: 400, matching the ?project= convention of the
	// other reads.
	rr3, _ := do(t, h, http.MethodGet, "/knowledge/graph", nil)
	if rr3.Code != http.StatusBadRequest {
		t.Errorf("missing project: status %d, want 400", rr3.Code)
	}

	// Now the bundle exists: 2 concepts + 1 resolving edge + the flags.
	writeKnowledgeBundle(t, workDir)
	rr4, out4 := do(t, h, http.MethodGet, "/knowledge/graph?project="+proj, nil)
	if rr4.Code != http.StatusOK {
		t.Fatalf("bundle: status %d, want 200: %s", rr4.Code, rr4.Body.String())
	}
	nodes, _ := out4["nodes"].([]any)
	if len(nodes) != 2 {
		t.Fatalf("bundle: nodes = %d, want 2 (index/log excluded): %v", len(nodes), out4)
	}
	byID := map[string]map[string]any{}
	for _, n := range nodes {
		m, _ := n.(map[string]any)
		byID[m["id"].(string)] = m
	}
	d1, ok := byID["decisions/d1"]
	if !ok {
		t.Fatalf("missing node decisions/d1: %v", byID)
	}
	if d1["title"] != "D-1" || d1["type"] != "decision" || d1["status"] != "active" {
		t.Errorf("d1 frontmatter = %v, want D-1/decision/active", d1)
	}
	i1, ok := byID["invariants/i1"]
	if !ok {
		t.Fatalf("missing node invariants/i1: %v", byID)
	}
	if b, _ := i1["deprecated"].(bool); !b {
		t.Errorf("i1 deprecated = %v, want true", i1["deprecated"])
	}
	if b, _ := i1["orphan"].(bool); b {
		t.Errorf("i1 orphan = true, want false (d1 links to it)")
	}
	edges, _ := out4["edges"].([]any)
	if len(edges) != 1 {
		t.Fatalf("bundle: edges = %d, want 1: %v", len(edges), edges)
	}
	edge, _ := edges[0].(map[string]any)
	if edge["source"] != "decisions/d1" || edge["target"] != "invariants/i1" {
		t.Errorf("edge = %v, want decisions/d1 -> invariants/i1", edge)
	}
	backlinks, _ := out4["backlinks"].(map[string]any)
	if ids, _ := backlinks["invariants/i1"].([]any); len(ids) != 1 || ids[0] != "decisions/d1" {
		t.Errorf("backlinks[invariants/i1] = %v, want [decisions/d1]", backlinks["invariants/i1"])
	}

	// Per-call, never stored: edit the bundle (drop the link) and the SAME
	// route reflects the edit on the next request without any write call.
	if err := os.WriteFile(filepath.Join(workDir, "decisions", "d1.md"),
		[]byte("---\ntype: decision\ntitle: D-1\nstatus: active\n---\n# D-1\n\ntext\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rr5, out5 := do(t, h, http.MethodGet, "/knowledge/graph?project="+proj, nil)
	if rr5.Code != http.StatusOK {
		t.Fatalf("re-derive: status %d, want 200: %s", rr5.Code, rr5.Body.String())
	}
	if edges5, _ := out5["edges"].([]any); len(edges5) != 0 {
		t.Errorf("re-derive: edges = %d, want 0 (the link was removed): %v", len(edges5), edges5)
	}
	if nodes5, _ := out5["nodes"].([]any); len(nodes5) != 2 {
		t.Errorf("re-derive: nodes = %d, want 2", len(nodes5))
	}
}
