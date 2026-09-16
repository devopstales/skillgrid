package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 02.9 [AFK] /tracker/* + /backlog/* alias mounted on the existing mux.
func TestStep02_Routes(t *testing.T) {
	h := newHandler(t)
	t.Setenv("SKILLGRID_TRACKER", "backlogmd")
	t.Setenv("PATH", t.TempDir()) // no CLIs: every tracker route degrades, none 404s
	for _, target := range []string{
		"/tracker/config", "/tracker/tasks", "/tracker/tasks/TASK-001",
		"/backlog/config", "/backlog/tasks", "/backlog/tasks/TASK-001",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, req)
		if rr.Code == http.StatusNotFound {
			t.Errorf("%s must be mounted (got 404)", target)
		}
		if rr.Code != http.StatusServiceUnavailable {
			t.Errorf("%s without CLI must be 503, got %d", target, rr.Code)
		}
	}
	// Status routes are write-gated and mounted.
	req := httptest.NewRequest(http.MethodPost, "/tracker/tasks/TASK-001/status",
		strings.NewReader(`{"status":"done"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code == http.StatusNotFound {
		t.Error("POST /tracker/tasks/{id}/status must be mounted (got 404)")
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
