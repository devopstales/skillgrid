package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 03.2/03.3 [AFK] The /docs/changes routes are mounted on the real handler
// tree and coexist with the GET /docs shell page route.
func TestStep03_RoutesMounted(t *testing.T) {
	h := newHandler(t)

	// GET /docs still serves the dashboard shell page (exact match wins over
	// the /docs/ subtree handler).
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /docs: expected 200 shell, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `id="menu"`) {
		t.Error("GET /docs must serve the dashboard shell")
	}

	// GET /docs/changes serves the JSON list (empty in the test env, not an
	// error).
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/changes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /docs/changes: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"changes"`) {
		t.Errorf("GET /docs/changes body must carry the changes key: %s", rr.Body.String())
	}
}
