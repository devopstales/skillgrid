package http

import (
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
)

// firstDistJS sorts all .js assets in the embedded dist and returns the first,
// so the choice is deterministic and survives content-hash rebuilds.
func firstDistJS(t *testing.T) string {
	t.Helper()
	sub, err := fs.Sub(uiDistFS, "ui/dist/assets")
	if err != nil {
		t.Fatalf("fs.Sub ui/dist/assets: %v", err)
	}
	var names []string
	err = fs.WalkDir(sub, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".js") {
			names = append(names, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk dist/assets: %v", err)
	}
	if len(names) == 0 {
		t.Fatalf("no .js assets found in ui/dist/assets")
	}
	sort.Strings(names)
	return names[0]
}

// TestPhase1_Embed covers @phase-1: GET / serves the embedded SPA shell,
// hashed assets serve their real content, and API-prefix 404s stay JSON.
func TestPhase1_Embed(t *testing.T) {
	h := newHandler(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `<div id="root">`) || !strings.Contains(strings.ToLower(rr.Body.String()), `<!doctype html>`) {
		t.Errorf("GET / must return the SPA index.html, got: %.200s", rr.Body.String())
	}

	asset := firstDistJS(t)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/assets/"+asset, nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /%s: expected 200, got %d", asset, rr.Code)
	}
	f, err := uiDistFS.Open("ui/dist/assets/" + asset)
	if err != nil {
		t.Fatalf("open %s: %v", asset, err)
	}
	want, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		t.Fatalf("read %s: %v", asset, err)
	}
	if got := rr.Body.String(); got != string(want) {
		t.Errorf("GET /%s content differs from the embedded asset (got %d bytes, want %d)", asset, len(got), len(want))
	}

	// /memory/{rest...} is an API prefix → unknown tails stay 404 JSON (not
	// the SPA shell), while /memory/status (registered above) still resolves.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/memory/nope", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /memory/nope: expected 404, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"error"`) {
		t.Errorf("GET /memory/nope must be a JSON error, got: %.200s", rr.Body.String())
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/memory/status?project=p1", nil))
	if rr.Code != http.StatusOK {
		t.Errorf("GET /memory/status: expected 200, got %d", rr.Code)
	}
}

// TestPhase1_SPAFallback covers @phase-1: openapi/swagger preserved, SPA
// fallback for non-API paths, API routes not shadowed.
func TestPhase1_SPAFallback(t *testing.T) {
	h := newHandler(t)

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /openapi.yaml: expected 200, got %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "yaml") {
		t.Errorf("GET /openapi.yaml content-type: expected yaml, got %q", ct)
	}
	if !strings.Contains(rr.Body.String(), "openapi:") {
		t.Errorf("GET /openapi.yaml body must be the spec, got: %.200s", rr.Body.String())
	}

	for path, marker := range map[string]string{
		"/swagger/":            "<!DOCTYPE html>",
		"/swagger/swagger-ui.css": "",
	} {
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("GET %s: expected 200, got %d (%s)", path, rr.Code, rr.Body.String())
		}
		if marker != "" && !strings.Contains(strings.ToLower(rr.Body.String()), strings.ToLower(marker)) {
			t.Errorf("GET %s body missing %q", path, marker)
		}
	}

	// SPA fallback: non-API, non-asset path → index.html.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/tracker", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /tracker: expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `<div id="root">`) {
		t.Errorf("GET /tracker must serve the SPA index.html, got: %.200s", rr.Body.String())
	}

	// API route with unknown id → 404 JSON, not index.html.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/projects/xyz", nil))
	if rr.Code != http.StatusNotFound {
		t.Fatalf("GET /projects/xyz: expected 404, got %d", rr.Code)
	}
	if strings.Contains(rr.Body.String(), `<div id="root">`) {
		t.Errorf("GET /projects/xyz must be JSON, not the SPA shell: %.200s", rr.Body.String())
	}

	// Existing API routes must not be shadowed.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/projects", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /projects: expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"projects"`) {
		t.Errorf("GET /projects must return the projects JSON, got: %.200s", rr.Body.String())
	}

	// /docs shell page (SPA fallback) vs /docs/changes API route.
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `<div id="root">`) {
		t.Errorf("GET /docs must serve the SPA shell, got %d: %.200s", rr.Code, rr.Body.String())
	}
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/changes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /docs/changes: expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"changes"`) {
		t.Errorf("GET /docs/changes must return the changes JSON, got: %.200s", rr.Body.String())
	}
}
