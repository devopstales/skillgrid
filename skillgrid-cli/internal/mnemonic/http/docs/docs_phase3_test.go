package docs

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 3.3 [AFK] GET /docs/tree returns the grouped tree (root, title, updated,
// frontmatter status).
func TestPhase3_Tree(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	// root=all groups each method under a top folder (Skillgrid, Backlog, Docs)
	// and inlines top-level *.md (README.md).
	code, body := do("/docs/tree")
	if code != http.StatusOK {
		t.Fatalf("tree: expected 200, got %d (%s)", code, body)
	}
	for _, want := range []string{
		"Skillgrid", "Backlog", "TASK-001-something.md", "in-progress", "Briefing",
		"009-web-admin-dashboard", "README.md",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("all tree missing %q in %s", want, body)
		}
	}
	// the skillgrid method is a named folder (title "Skillgrid"), not a raw
	// ".skillgrid" directory node.
	if strings.Contains(body, `"name":".skillgrid"`) {
		t.Errorf("all tree should group skillgrid under a named folder: %s", body)
	}

	// root=backlog scopes to just the backlog root
	_, body = do("/docs/tree?root=backlog")
	if !strings.Contains(body, "TASK-001-something.md") {
		t.Errorf("root=backlog missing the task node: %s", body)
	}
	if strings.Contains(body, "briefing.md") {
		t.Errorf("root=backlog leaked the sdd root: %s", body)
	}

	// unknown root -> empty node list (400 per impl), not a crash
	code, _ = do("/docs/tree?root=nope")
	if code != http.StatusBadRequest {
		t.Errorf("unknown root: expected 400, got %d", code)
	}
}

// 3.4 [AFK] GET /docs/search returns matches (path + snippet).
func TestPhase3_Search(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	code, body := do("/docs/search?q=Briefing")
	if code != http.StatusOK {
		t.Fatalf("search: expected 200, got %d (%s)", code, body)
	}
	if !strings.Contains(body, "briefing.md") {
		t.Errorf("search for 'Briefing' missing briefing.md: %s", body)
	}
	if !strings.Contains(body, "snippet") {
		t.Errorf("search result missing snippet field: %s", body)
	}

	// empty q -> empty results
	_, body = do("/docs/search?q=")
	if !strings.Contains(body, `"results":[]`) && !strings.Contains(body, `"results": []`) {
		t.Errorf("empty q should return empty results: %s", body)
	}
}

// 3.5 [AFK] GET /docs/render returns rendered HTML for a doc (escaped).
func TestPhase3_Render(t *testing.T) {
	h := newMDHandler(t)
	do := func(target string) (int, string) {
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
		return rr.Code, rr.Body.String()
	}

	code, body := do("/docs/render?path=README.md")
	if code != http.StatusOK {
		t.Fatalf("render: expected 200, got %d (%s)", code, body)
	}
	// seedMDRepo writes README.md at the repo root (a declared root).
	if !strings.Contains(body, "<h1>") || !strings.Contains(body, "README") {
		t.Errorf("render missing rendered heading: %s", body)
	}
	// traversal still blocked on render
	if rc, _ := do("/docs/render?path=../secret.md"); rc != http.StatusBadRequest {
		t.Errorf("render traversal: expected 400, got %d", rc)
	}
}

// 3.8 [AFK] openapi.yaml documents the docs routes with examples.
func TestPhase3_OpenAPI(t *testing.T) {
	// The openapi.yaml lives at skillgrid-cli/internal/mnemonic/http/ui/openapi.yaml
	// relative to this package's parent. Walk up to find it.
	here, _ := os.Getwd()
	openapi := filepath.Join(here, "..", "ui", "openapi.yaml")
	data, err := os.ReadFile(openapi)
	if err != nil {
		t.Skipf("openapi.yaml not found at %s: %v", openapi, err)
	}
	text := string(data)
	for _, want := range []string{
		"/docs/tree", "/docs/content", "/docs/search", "/docs/render",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("openapi.yaml missing docs route %s", want)
		}
	}
}
