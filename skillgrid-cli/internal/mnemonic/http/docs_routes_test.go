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

	// GET /docs/changes serves the JSON list (empty in the test env, not an
	// error).
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/docs/changes", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("GET /docs/changes: expected 200, got %d (%s)", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"changes"`) {
		t.Errorf("GET /docs/changes body must carry the changes key: %s", rr.Body.String())
	}
}
