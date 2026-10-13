package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestCTXMetrics_FailOpenDefault verifies GET /metrics answers 200 with valid
// Prometheus text and zero-valued counters when no proxy sessions exist.
func TestCTXMetrics_FailOpenDefault(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	req := httptest.NewRequest(http.MethodGet, "/metrics?project="+proj, nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	for _, line := range []string{
		"# HELP ctx_requests_total",
		"# TYPE ctx_requests_total counter",
		"ctx_requests_total 0",
		"# HELP ctx_tokens_saved_total",
		"# TYPE ctx_tokens_saved_total counter",
		"ctx_tokens_saved_total 0",
	} {
		if !strings.Contains(body, line) {
			t.Errorf("metrics body missing %q\nbody:\n%s", line, body)
		}
	}
}

// TestCTXMetrics_WithSeededSessions verifies GET /metrics emits non-zero
// counters matching the seeded ctx_proxy_sessions rows.
func TestCTXMetrics_WithSeededSessions(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	seedCtxProxySessions(t, dataDir)

	req := httptest.NewRequest(http.MethodGet, "/metrics?project="+proj, nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	// Seed: sess-ctx-1 (requests=5, saved=3000, hits=3, inv=1) +
	// sess-ctx-2 (requests=3, saved=4000, hits=0, inv=2) → totals
	// requests=8, saved=7000, hits=3, inv=3, sessions=2.
	for _, line := range []string{
		"ctx_requests_total 8",
		"ctx_tokens_saved_total 7000",
		"ctx_cache_hits_total 3",
		"ctx_cache_invalidations_total 3",
		"ctx_sessions_total 2",
	} {
		if !strings.Contains(body, line) {
			t.Errorf("metrics body missing %q\nbody:\n%s", line, body)
		}
	}
}

// TestCTXSessionSummaries_FailOpenDefault verifies GET /api/session-summaries
// answers 200 with an empty items list when no summaries are stored.
func TestCTXSessionSummaries_FailOpenDefault(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	req := httptest.NewRequest(http.MethodGet, "/api/session-summaries?project="+proj, nil)
	req.RemoteAddr = "127.0.0.1:54321"
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := decodeBody(rr, &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	items, _ := out["items"].([]any)
	if len(items) != 0 {
		t.Errorf("expected empty items, got %d", len(items))
	}
	if out["next"] != "" {
		t.Errorf("expected empty next cursor, got %q", out["next"])
	}
}

// TestCTXSessionSummaries_Pagination verifies the offset pagination caps
// items at the limit, emits a next cursor, and the next page returns the
// remainder.
func TestCTXSessionSummaries_Pagination(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	seedSessionSummaries(t, dataDir, 5)

	// Page 1: limit 3 → 3 items + non-empty next.
	rr, out := doGetCtx(t, s, "/api/session-summaries?project="+proj+"&limit=3", "127.0.0.1:1")
	if rr.Code != http.StatusOK {
		t.Fatalf("page1 expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	items, _ := out["items"].([]any)
	if len(items) != 3 {
		t.Fatalf("page1 expected 3 items, got %d", len(items))
	}
	next, _ := out["next"].(string)
	if next == "" {
		t.Fatalf("page1 expected non-empty next cursor, got %q", out["next"])
	}

	// Page 2: continue with cursor → the remaining 2.
	rr2, out2 := doGetCtx(t, s, "/api/session-summaries?project="+proj+"&limit=3&cursor="+next, "127.0.0.1:2")
	if rr2.Code != http.StatusOK {
		t.Fatalf("page2 expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	items2, _ := out2["items"].([]any)
	if len(items2) != 2 {
		t.Fatalf("page2 expected 2 items, got %d", len(items2))
	}
	if out2["next"] != "" {
		t.Errorf("page2 expected empty next cursor, got %q", out2["next"])
	}
}

// TestCTXSessionSummaries_LimitCap verifies the limit query param is capped at
// 100 (brief: default 50, max 100) regardless of how large a value is
// requested.
func TestCTXSessionSummaries_LimitCap(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	// Seed 120 rows: more than the 100 cap so a request for ?limit=100000
	// must return exactly 100 (the cap), not all 120.
	seedSessionSummaries(t, dataDir, 120)

	rr, out := doGetCtx(t, s, "/api/session-summaries?project="+proj+"&limit=100000", "127.0.0.1:1")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	items, _ := out["items"].([]any)
	if len(items) != 100 {
		t.Fatalf("limit=100000 expected 100 items (the cap), got %d", len(items))
	}
	// A cursor is emitted because 20 rows remain past the capped page.
	if out["next"] == "" {
		t.Errorf("expected non-empty next cursor (120 rows, 100 returned), got %q", out["next"])
	}

	// The lower bound is already the 50 default: an unparseable limit falls
	// back to 50 (and the cap never lowers a valid value below it).
	rr2, out2 := doGetCtx(t, s, "/api/session-summaries?project="+proj+"&limit=0", "127.0.0.1:2")
	if rr2.Code != http.StatusOK {
		t.Fatalf("limit=0 expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	items2, _ := out2["items"].([]any)
	if len(items2) != 50 {
		t.Fatalf("limit=0 expected the 50 default, got %d", len(items2))
	}
}

// TestCTXLoopbackOnly verifies all four ctx routes reject a non-loopback
// origin with 403 + a loopback-only JSON body (no data leak).
func TestCTXLoopbackOnly(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	seedCtxProxySessions(t, dataDir)

	for _, path := range []string{"/stats", "/stats-history", "/metrics", "/api/session-summaries"} {
		req := httptest.NewRequest(http.MethodGet, path+"?project="+proj, nil)
		req.RemoteAddr = "192.168.1.10:12345"
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("%s: expected 403 for non-loopback, got %d: %s", path, rr.Code, rr.Body.String())
		}
		body := rr.Body.String()
		if !strings.Contains(body, "loopback only") {
			t.Errorf("%s: 403 body should mention loopback only, got %q", path, body)
		}
	}
}

// TestCTXLoopbackOnly_AllFour200OnLoopback seeds data, then asserts each of
// the four routes answers 200 for a loopback origin.
func TestCTXLoopbackOnly_AllFour200OnLoopback(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	seedCtxProxySessions(t, dataDir)
	seedSessionSummaries(t, dataDir, 2)

	for _, path := range []string{"/stats", "/stats-history", "/metrics", "/api/session-summaries"} {
		req := httptest.NewRequest(http.MethodGet, path+"?project="+proj, nil)
		req.RemoteAddr = "127.0.0.1:9999"
		rr := httptest.NewRecorder()
		s.Handler().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("%s: expected 200 for loopback, got %d: %s", path, rr.Code, rr.Body.String())
		}
	}
}

// seedCtxProxySessions writes two proxy sessions for the metrics assertions.
func seedCtxProxySessions(t *testing.T, dataDir string) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	_, err = st.DB.Exec(`
		INSERT OR REPLACE INTO ctx_proxy_sessions
			(session_id, agent, mode, started_at, ended_at, requests,
			 tokens_before, tokens_after, cache_hits, cache_invalidations)
		VALUES
			('sess-ctx-1', 'opencode', 'cache', 1700000000, 1700000100, 5, 10000, 7000, 3, 1),
			('sess-ctx-2', 'claude', 'token', 1700000200, 1700000300, 3, 9000, 5000, 0, 2)`)
	if err != nil {
		t.Fatalf("seed proxy sessions: %v", err)
	}
}

// seedSessionSummaries writes n session-summary rows with distinct created_at.
func seedSessionSummaries(t *testing.T, dataDir string, n int) {
	t.Helper()
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	for i := 0; i < n; i++ {
		_, err = st.DB.Exec(`
			INSERT OR REPLACE INTO ctx_session_summaries
				(session_id, agent, duration_ms, turns, tokens_saved,
				 judgment_keeps, judgment_truncs, prefilter_pins, prefilter_deads,
				 cache_invalidations, shaping_applied, effort_routed,
				 traffic_memories, summary_json, created_at)
			VALUES (?, 'agent', 1000, 2, 500, 1, 0, 0, 0, 0, 1, 0, 0, '{}', ?)`,
			fmt.Sprintf("sum-%d", i), 1700000000+int64(i))
		if err != nil {
			t.Fatalf("seed session summary %d: %v", i, err)
		}
	}
}

// doGetCtx issues a GET with a forced RemoteAddr and decodes the JSON body.
func doGetCtx(t *testing.T, s *Server, target, remoteAddr string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	var out map[string]any
	if err := decodeBody(rr, &out); err != nil {
		t.Fatalf("decode body: %v (body: %s)", err, rr.Body.String())
	}
	return rr, out
}

// decodeBody decodes a JSON body into dst, tolerating an empty body.
func decodeBody(rr *httptest.ResponseRecorder, dst any) error {
	if rr.Body.Len() == 0 {
		return nil
	}
	return json.Unmarshal(rr.Body.Bytes(), dst)
}
