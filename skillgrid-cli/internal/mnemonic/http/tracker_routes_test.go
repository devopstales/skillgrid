package http

import (
	"net/http"
	"net/http/httptest"
	"os"
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
