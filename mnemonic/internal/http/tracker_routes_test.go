package http

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 02.9 [AFK] /tracker/* + /backlog/* alias mounted on the existing mux.
// Phase 2: the Backlog.md provider is file-based, so reads return 200 even
// without a CLI (no 503). The mux must mount every route (no unmounted 404).
func TestStep02_Routes(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLIs: file-based backlog still serves
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(".backlog/tasks", 0o755); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{
		"/tracker/config", "/tracker/tasks",
		"/backlog/config", "/backlog/tasks",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("%s must be 200 (file-based), got %d", target, rr.Code)
		}
	}
	// Status routes are write-gated and mounted (not an unmounted 404).
	req := httptest.NewRequest(http.MethodPost, "/tracker/tasks/TASK-001/status",
		strings.NewReader(`{"status":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound && !strings.Contains(rr.Body.String(), `"error"`) {
		t.Error("POST /tracker/tasks/{id}/status must be mounted (got unmounted 404)")
	}
}

// ?provider= overrides the server default per request; unknown → 501.
func TestStep02_ProviderOverride(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLIs: override switches adapter, still 503s
	req := httptest.NewRequest(http.MethodGet, "/tracker/config?provider=github", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("github override must be 200, got %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(body, `"provider":"github"`) {
		t.Errorf("override must switch adapter, got %s", body)
	}
	// ...but task reads still need the CLI: empty PATH → 503 naming gh.
	req = httptest.NewRequest(http.MethodGet, "/tracker/tasks?provider=github", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("github tasks without CLI must be 503, got %d", rr.Code)
	}
	if body := rr.Body.String(); !strings.Contains(body, "gh") {
		t.Errorf("503 must name the gh CLI, got %s", body)
	}
	req = httptest.NewRequest(http.MethodGet, "/tracker/config?provider=nope", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotImplemented {
		t.Errorf("unknown provider must be 501, got %d", rr.Code)
	}
}

// 02.10 [RED] GET /tracker/milestones round-trip: reads .backlog/milestones/*.md
// frontmatter, returns the spec'd {"milestones":[...],"provider":"backlogmd"}
// shape. Missing dir → empty (non-null) array, still 200 (degrade, not error).
func TestStep02_MilestonesEndpoint(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLIs: file-based backlog still serves
	root := t.TempDir()
	t.Chdir(root)

	// Two milestone files: id + title + description frontmatter.
	msDir := filepath.Join(root, ".backlog", "milestones")
	if err := os.MkdirAll(msDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"m-1 - alpha.md": "---\nid: m-1\ntitle: Alpha milestone\ndescription: First milestone\n---\nBody.\n",
		"m-2 - beta.md":  "---\nid: m-2\ntitle: Beta milestone\n---\nBody.\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(msDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/tracker/milestones", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /tracker/milestones must be 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, `"provider":"backlogmd"`) {
		t.Errorf("must report provider backlogmd, got %s", body)
	}
	for _, want := range []string{`"id":"m-1"`, `"title":"Alpha milestone"`, `"id":"m-2"`, `"title":"Beta milestone"`} {
		if !strings.Contains(body, want) {
			t.Errorf("milestone payload missing %s, got %s", want, body)
		}
	}
	// Sorted by id: m-1 must appear before m-2.
	if i, j := strings.Index(body, `"id":"m-1"`), strings.Index(body, `"id":"m-2"`); i == -1 || j == -1 || i > j {
		t.Errorf("milestones must be sorted by id, got %s", body)
	}
}

// 02.10 [RED] GET /tracker/milestones with no milestones dir → 200 with an
// empty (non-null) array, never an error.
func TestStep02_MilestonesEndpointMissingDir(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir())
	t.Chdir(t.TempDir()) // no .backlog at all

	req := httptest.NewRequest(http.MethodGet, "/tracker/milestones", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("missing dir must be 200 (degrade), got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"milestones":[]`) {
		t.Errorf("missing dir must return empty array, got %s", rr.Body.String())
	}
}
