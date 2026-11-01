package http

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func postUsage(t *testing.T, s *Server, sid string, body map[string]any) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/sessions/"+sid+"/usage?project="+proj, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST usage %v: status %d: %s", body, w.Code, w.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func near(v any, want float64) bool {
	f, ok := v.(float64)
	return ok && math.Abs(f-want) < 1e-9
}

func TestSessionUsage_AddsUpAndPrices(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	s, _ := newToolCallsServer(t)

	// claude-sonnet-4-5 → claude-sonnet-4 price: $3 in, $15 out, $0.30 cache per 1M.
	u := postUsage(t, s, "oc-1", map[string]any{"model": "claude-sonnet-4-5", "input": 1_000_000, "output": 100_000, "cache": 0, "agent": "opencode"})
	if !near(u["cost_usd"], 4.5) || u["input_tokens"] != float64(1_000_000) {
		t.Fatalf("first report = %v, want cost 4.5", u)
	}
	u = postUsage(t, s, "oc-1", map[string]any{"input": 0, "output": 100_000, "cache": 1_000_000})
	if !near(u["cost_usd"], 4.5+1.5+0.3) || u["output_tokens"] != float64(200_000) || u["model"] != "claude-sonnet-4-5" {
		t.Fatalf("delta report = %v, want summed totals and kept model", u)
	}

	// total=true replaces (idempotent re-reads of a harness transcript).
	for i := 0; i < 2; i++ {
		u = postUsage(t, s, "oc-2", map[string]any{"model": "gpt-5", "input": 2_000_000, "output": 0, "total": true})
	}
	if !near(u["cost_usd"], 2.5) || u["input_tokens"] != float64(2_000_000) {
		t.Fatalf("total report = %v, want 2M input once, cost 2.5", u)
	}

	// Unknown model: tokens recorded, cost stays n/a (null).
	u = postUsage(t, s, "cur-1", map[string]any{"model": "cursor-secret-model", "input": 500, "output": 50, "agent": "cursor"})
	if u["cost_usd"] != nil || u["input_tokens"] != float64(500) {
		t.Fatalf("unknown model = %v, want null cost", u)
	}
	// Model-only report (Cursor stop hook): model set, no cost invented.
	u = postUsage(t, s, "cur-2", map[string]any{"model": "claude-sonnet-4-5", "agent": "cursor"})
	if u["cost_usd"] != nil || u["model"] != "claude-sonnet-4-5" {
		t.Fatalf("model-only = %v, want model and null cost", u)
	}

	st := getToolJSON(t, s, "/events/stats?project="+proj)
	models, _ := st["byModel"].([]any)
	if len(models) != 3 {
		t.Fatalf("byModel = %v, want sonnet (2 sessions), gpt-5, unknown", models)
	}
	top := models[0].(map[string]any)
	if top["model"] != "claude-sonnet-4-5" || !near(top["costUsd"], 6.3) || top["sessions"] != float64(2) {
		t.Errorf("most expensive model = %v", top)
	}
	sessions, _ := getToolJSON(t, s, "/mnemonic/sessions?project="+proj)["sessions"].([]any)
	found := false
	for _, raw := range sessions {
		row := raw.(map[string]any)
		if row["id"] == "oc-1" {
			found = true
			if row["model"] != "claude-sonnet-4-5" || !near(row["cost_usd"], 6.3) || row["agent"] != "opencode" {
				t.Errorf("oc-1 row = %v", row)
			}
		}
	}
	if !found {
		t.Error("oc-1 missing from /mnemonic/sessions")
	}
}

func TestSessionUsage_RejectsNegative(t *testing.T) {
	s, _ := newToolCallsServer(t)
	raw, _ := json.Marshal(map[string]any{"input": -1})
	req := httptest.NewRequest(http.MethodPost, "/sessions/x/usage?project="+proj, bytes.NewReader(raw))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}
