package http

import (
	"net/http"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/observability"
	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// TestCTXSessionSummaries_ReadThroughT11Writer verifies the end-to-end write
// then read path for the T11 session-summary rollup: seed two proxy sessions,
// run observability.SummarizeSessions to derive the summary rows, then read
// them back over loopback through the existing T10 reader
// (GET /api/session-summaries). This proves the writer feeds the reader — the
// reader itself is unchanged (T10).
func TestCTXSessionSummaries_ReadThroughT11Writer(t *testing.T) {
	s, dataDir := newToolCallsServer(t)
	defer closeStore(t, dataDir)

	// Seed two proxy sessions, then derive the summary rows in-process (the
	// same call the proxy's rollup loop and shutdown final-rollup make).
	st, err := store.Open(dataDir, proj)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	// Seed through the raw DB, then roll. The accessor wraps this store's DB.
	if _, err := st.DB.Exec(`
		INSERT OR REPLACE INTO ctx_proxy_sessions
			(session_id, agent, mode, started_at, ended_at, requests,
			 tokens_before, tokens_after, cache_hits, cache_invalidations)
		VALUES
			('sess-t11-1', 'opencode', 'cache', 1700000000, 1700000120, 5, 10000, 7000, 3, 1),
			('sess-t11-2', 'claude', 'token', 1700000200, 1700000300, 3, 9000, 12000, 0, 2)`); err != nil {
		t.Fatalf("seed proxy sessions: %v", err)
	}
	cs := ctxstore.New(st)
	written, err := observability.SummarizeSessions(cs, 1800000000)
	if err != nil {
		t.Fatalf("SummarizeSessions: %v", err)
	}
	if written != 2 {
		t.Fatalf("SummarizeSessions: expected 2 rows written, got %d", written)
	}
	st.Close()

	// Read back over loopback through the T10 reader.
	rr, out := doGetCtx(t, s, "/api/session-summaries?project="+proj, "127.0.0.1:7777")
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	items, _ := out["items"].([]any)
	if len(items) != 2 {
		t.Fatalf("expected 2 summary items, got %d", len(items))
	}

	// Index by session_id.
	byID := make(map[string]map[string]any, len(items))
	for _, it := range items {
		m, _ := it.(map[string]any)
		if sid, _ := m["session_id"].(string); sid != "" {
			byID[sid] = m
		}
	}
	s1, ok := byID["sess-t11-1"]
	if !ok {
		t.Fatalf("missing summary item for sess-t11-1: %+v", items)
	}
	if got := int64(asInt(s1["duration_ms"])); got != 120 {
		t.Errorf("sess-t11-1 duration_ms = %d, want 120", got)
	}
	if got := int64(asInt(s1["turns"])); got != 5 {
		t.Errorf("sess-t11-1 turns = %d, want 5", got)
	}
	if got := int64(asInt(s1["tokens_saved"])); got != 3000 {
		t.Errorf("sess-t11-1 tokens_saved = %d, want 3000 (10000-7000)", got)
	}
	if got := int64(asInt(s1["cache_invalidations"])); got != 1 {
		t.Errorf("sess-t11-1 cache_invalidations = %d, want 1", got)
	}
	if got := s1["agent"].(string); got != "opencode" {
		t.Errorf("sess-t11-1 agent = %q, want opencode", got)
	}
	// summary_json is parsed to a map by the reader (valid JSON).
	if sj, ok := s1["summary_json"].(map[string]any); !ok {
		t.Errorf("sess-t11-1 summary_json not an object: %T %v", s1["summary_json"], s1["summary_json"])
	} else {
		if got := int64(asInt(sj["tokens_saved"])); got != 3000 {
			t.Errorf("sess-t11-1 summary_json.tokens_saved = %d, want 3000", got)
		}
		if got := sj["mode"].(string); got != "cache" {
			t.Errorf("sess-t11-1 summary_json.mode = %q, want cache", got)
		}
	}

	s2, ok := byID["sess-t11-2"]
	if !ok {
		t.Fatalf("missing summary item for sess-t11-2: %+v", items)
	}
	// sess-t11-2 has after > before: tokens_saved clamps to 0.
	if got := int64(asInt(s2["tokens_saved"])); got != 0 {
		t.Errorf("sess-t11-2 tokens_saved = %d, want 0 (negative clamped)", got)
	}
	if got := int64(asInt(s2["turns"])); got != 3 {
		t.Errorf("sess-t11-2 turns = %d, want 3", got)
	}
}

// asInt converts a decoded JSON number (float64) to int, tolerating an absent
// value (returns 0).
func asInt(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	default:
		return 0
	}
}
