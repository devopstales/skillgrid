package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// seedPrototypes writes two prototype directories under .skillgrid/prototypes/: one with a
// full prototype.md (date + hypothesis + extra file) and one with no prototype.md
// (WIP) to confirm it is still listed.
func seedPrototypes(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, ".skillgrid", "prototypes")

	a := filepath.Join(base, "001-cgo-free-vector-db")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	contentA := `# Prototype: 001-cgo-free-vector-db

**Date:** 2026-09-24
**Type:** comparison (G vs viant)
**Hypothesis:** Given a Go module, when we adopt either, then both work.

## How to run
`
	if err := os.WriteFile(filepath.Join(a, "prototype.md"), []byte(contentA), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a, "g.go"), []byte("package g\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	b := filepath.Join(base, "002-dashboard-variants")
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "index.html"), []byte("<html></html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// GET /prototypes lists prototype directories and parses date/type/hypothesis from
// each prototype.md; a WIP prototype without prototype.md is still listed.
func TestListPrototypes(t *testing.T) {
	cwd := seedPrototypes(t)
	h := NewPrototypes(cwd)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/prototypes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Source     string      `json:"source"`
		Count      int         `json:"count"`
		Prototypes []Prototype `json:"prototypes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Source != ".skillgrid/prototypes" {
		t.Errorf("source = %q", resp.Source)
	}
	if resp.Count != 2 || len(resp.Prototypes) != 2 {
		t.Fatalf("expected 2 prototypes, got %d (%+v)", resp.Count, resp.Prototypes)
	}
	// sorted by name: 001 < 002.
	if resp.Prototypes[0].Name != "001-cgo-free-vector-db" {
		t.Errorf("prototypes[0].name = %q", resp.Prototypes[0].Name)
	}
	s0 := resp.Prototypes[0]
	if s0.Date != "2026-09-24" {
		t.Errorf("prototype[0].date = %q, want 2026-09-24", s0.Date)
	}
	if s0.Venue != "comparison (G vs viant)" {
		t.Errorf("prototype[0].venue = %q", s0.Venue)
	}
	if s0.Hypothesis != "Given a Go module, when we adopt either, then both work." {
		t.Errorf("prototype[0].hypothesis = %q", s0.Hypothesis)
	}
	if len(s0.Files) != 2 {
		t.Errorf("prototype[0].files = %v, want 2 entries", s0.Files)
	}
	// WIP prototype (no prototype.md) is still listed with its files.
	s1 := resp.Prototypes[1]
	if s1.Name != "002-dashboard-variants" {
		t.Errorf("prototypes[1].name = %q", s1.Name)
	}
	if s1.Date != "" || s1.Hypothesis != "" {
		t.Errorf("prototype[1] should have no parsed meta, got %+v", s1)
	}
	if len(s1.Files) != 1 || s1.Files[0] != "index.html" {
		t.Errorf("prototype[1].files = %v", s1.Files)
	}
}

// GET /prototypes/{name}/index.html serves the preview; traversal and a
// missing file do not.
func TestPrototypeFile(t *testing.T) {
	cwd := seedPrototypes(t)
	h := NewPrototypeFile(cwd)

	serve := func(name, file string) *httptest.ResponseRecorder {
		t.Helper()
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/prototypes/"+name+"/"+file, nil)
		req.SetPathValue("name", name)
		req.SetPathValue("file", file)
		h.ServeHTTP(rr, req)
		return rr
	}

	ok := serve("002-dashboard-variants", "index.html")
	if ok.Code != http.StatusOK {
		t.Fatalf("index.html: expected 200, got %d (%s)", ok.Code, ok.Body.String())
	}
	if ok.Body.String() != "<html></html>" {
		t.Errorf("index.html body = %q", ok.Body.String())
	}
	if ok.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("missing nosniff")
	}

	for _, tc := range []struct {
		name, file string
		want       int
	}{
		{"002-dashboard-variants", "missing.html", http.StatusNotFound},
		{"002-dashboard-variants", "../secret", http.StatusBadRequest},
		{"..", "index.html", http.StatusBadRequest},
		{"002-dashboard-variants", "", http.StatusBadRequest},
	} {
		rr := serve(tc.name, tc.file)
		if rr.Code != tc.want {
			t.Errorf("%s/%s: expected %d, got %d (%s)", tc.name, tc.file, tc.want, rr.Code, rr.Body.String())
		}
	}
}

// A **Topic:** header links the prototype to that change when the directory
// exists; otherwise a spec that cites the prototype directory does.
func TestPrototypeChangeLinks(t *testing.T) {
	root := t.TempDir()
	proto := filepath.Join(root, ".skillgrid", "prototypes")
	explicit := filepath.Join(proto, "002-dashboard-variants")
	cited := filepath.Join(proto, "001-mockup")
	if err := os.MkdirAll(explicit, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(cited, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "archive", "2026-09-08-web-admin-dashboard"), 0o755); err != nil {
		t.Fatal(err)
	}
	spec := filepath.Join(root, ".skillgrid", "specs", "2026-10-02-mnemonic-webui-rewrite")
	if err := os.MkdirAll(spec, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(explicit, "prototype.md"), []byte("**Topic:** 2026-09-08-web-admin-dashboard\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cited, "prototype.md"), []byte("# Prototype\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(spec, "briefing.md"), []byte("Evidence: `.skillgrid/prototypes/001-mockup/`\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewPrototypes(root)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/prototypes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	var resp struct {
		Prototypes []Prototype `json:"prototypes"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, p := range resp.Prototypes {
		got[p.Name] = p.Change
	}
	if got["002-dashboard-variants"] != "2026-09-08-web-admin-dashboard" {
		t.Errorf("explicit topic change = %q", got["002-dashboard-variants"])
	}
	if got["001-mockup"] != "2026-10-02-mnemonic-webui-rewrite" {
		t.Errorf("cited change = %q", got["001-mockup"])
	}
}

// Missing .skillgrid/prototypes root → 404.
func TestListPrototypesMissing(t *testing.T) {
	h := NewPrototypes(t.TempDir())
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/prototypes", nil))
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing root: expected 404, got %d (%s)", rr.Code, rr.Body.String())
	}
}
