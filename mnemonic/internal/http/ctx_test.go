package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// doLoopback is a loopback-address variant of do (server_test.go): the ctx
// observability routes are loopback-only (TICKET-10), and do's
// httptest.NewRequest defaults RemoteAddr to 192.0.2.1 (a TEST-NET address,
// not loopback). These tests exercise the route from the local operator
// perspective, so they force a loopback origin.
func doLoopback(t *testing.T, h http.Handler, method, target string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, reader)
	req.RemoteAddr = "127.0.0.1:54321"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(rr.Body.Bytes()))
	_ = dec.Decode(&out)
	return rr, out
}

// TestCTXStats_FailOpenDefault verifies GET /stats responds 200 with
// zero-valued JSON when no proxy sessions exist (the common case before any
// proxy traffic has been recorded).
func TestCTXStats_FailOpenDefault(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	rr, out := doLoopback(t, s.Handler(), http.MethodGet, "/stats?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["sessions"] != float64(0) {
		t.Errorf("expected sessions=0, got %v", out["sessions"])
	}
	if out["tokens_saved"] != float64(0) {
		t.Errorf("expected tokens_saved=0, got %v", out["tokens_saved"])
	}
	if out["cache_hits"] != float64(0) {
		t.Errorf("expected cache_hits=0, got %v", out["cache_hits"])
	}
	if out["requests"] != float64(0) {
		t.Errorf("expected requests=0, got %v", out["requests"])
	}
	// Mode should be present (empty string when no sessions).
	if _, present := out["mode"]; !present {
		t.Errorf("expected mode key in response, got %v", out)
	}
}

// TestCTXStats_WithSeededSessions verifies GET /stats returns non-zero
// metrics when proxy sessions have been written to the store.
func TestCTXStats_WithSeededSessions(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Seed a proxy session via raw SQL.
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = st.DB.Exec(`
		INSERT OR REPLACE INTO ctx_proxy_sessions
			(session_id, agent, mode, started_at, ended_at, requests,
			 tokens_before, tokens_after, cache_hits, cache_invalidations)
		VALUES ('sess-ctx-1', 'opencode', 'cache', 1700000000, 1700000100, 5, 10000, 7000, 3, 1)`)
	if err != nil {
		t.Fatalf("seed proxy session: %v", err)
	}
	st.Close()

	rr, out := doLoopback(t, s.Handler(), http.MethodGet, "/stats?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if out["sessions"] != float64(1) {
		t.Errorf("expected sessions=1, got %v", out["sessions"])
	}
	if out["tokens_saved"] != float64(3000) {
		t.Errorf("expected tokens_saved=3000, got %v", out["tokens_saved"])
	}
	if out["cache_hits"] != float64(3) {
		t.Errorf("expected cache_hits=3, got %v", out["cache_hits"])
	}
	if out["requests"] != float64(5) {
		t.Errorf("expected requests=5, got %v", out["requests"])
	}
	if out["mode"] != "cache" {
		t.Errorf("expected mode=cache, got %v", out["mode"])
	}
}

// TestCTXStatsHistory_FailOpenDefault verifies GET /stats-history responds
// 200 with an empty per-granularity bucket list when no history rows exist.
func TestCTXStatsHistory_FailOpenDefault(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	rr, out := doLoopback(t, s.Handler(), http.MethodGet, "/stats-history?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	buckets, ok := out["buckets"].(map[string]any)
	if !ok {
		t.Fatalf("expected buckets object, got %T: %v", out["buckets"], out)
	}
	// The four standard granularities must all be present.
	for _, g := range []string{"hourly", "daily", "weekly", "monthly"} {
		if _, present := buckets[g]; !present {
			t.Errorf("expected granularity %q in buckets, got keys: %v", g, keys(buckets))
		}
	}
}

// TestCTXStatsHistory_WithSeededBuckets verifies GET /stats-history returns
// non-empty bucket lists when history rows have been written.
func TestCTXStatsHistory_WithSeededBuckets(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer func() {
		if st, err := store.Open(dataDir, proj); err == nil {
			st.Close()
		}
	}()

	// Seed a stats history bucket via raw SQL.
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = st.DB.Exec(`
		INSERT OR REPLACE INTO ctx_stats_history
			(bucket_id, granularity, bucket_start, tokens_saved,
			 requests, sessions, cache_hits, created_at)
		VALUES ('hourly-1700000000', 'hourly', 1700000000, 5000, 10, 2, 4, 1700000000)`)
	if err != nil {
		t.Fatalf("seed stats bucket: %v", err)
	}
	st.Close()

	rr, out := doLoopback(t, s.Handler(), http.MethodGet, "/stats-history?project="+proj, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	buckets, ok := out["buckets"].(map[string]any)
	if !ok {
		t.Fatalf("expected buckets object, got %T", out["buckets"])
	}
	hourly, ok := buckets["hourly"].([]any)
	if !ok {
		t.Fatalf("expected hourly to be an array, got %T", buckets["hourly"])
	}
	if len(hourly) != 1 {
		t.Fatalf("expected 1 hourly bucket, got %d", len(hourly))
	}
	b0, _ := hourly[0].(map[string]any)
	if b0["tokens_saved"] != float64(5000) {
		t.Errorf("expected tokens_saved=5000, got %v", b0["tokens_saved"])
	}
	if b0["requests"] != float64(10) {
		t.Errorf("expected requests=10, got %v", b0["requests"])
	}
}

// keys returns the sorted keys of a map for error messages.
func keys(m map[string]any) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
