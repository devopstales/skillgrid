package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestMemSearchSignalsKeyword(t *testing.T) {
	t.Setenv("MNEMONIC_EMBED", "")
	memRetrievalFixture(t)
	ctx := context.Background()
	sid := startMCPSession(t)
	saveRes, err := handleMemSave(ctx, newCallTool("mem_save", map[string]any{
		"title": "signal banana", "type": "learning", "content": "signal banana", "session_id": sid,
	}))
	if err != nil || saveRes.IsError {
		t.Fatalf("save: %v %s", err, callResultText(t, saveRes))
	}
	res, err := handleMemSearch(ctx, newCallTool("mem_search", map[string]any{
		"query": "banana", "reader_owner": sid,
	}))
	if err != nil || res.IsError {
		t.Fatalf("search: %v %s", err, callResultText(t, res))
	}
	var body struct {
		Observations []struct {
			MatchedVia string             `json:"matched_via"`
			Signals    map[string]float64 `json:"signals"`
		} `json:"observations"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Observations) != 1 {
		t.Fatalf("observations = %d", len(body.Observations))
	}
	o := body.Observations[0]
	if o.MatchedVia != "keyword" {
		t.Fatalf("matched_via = %q", o.MatchedVia)
	}
	if o.Signals["vector"] != 0 || o.Signals["entity"] != 0 {
		t.Fatalf("signals = %+v", o.Signals)
	}
	for _, key := range []string{"keyword", "vector", "recency", "entity", "decay", "importance"} {
		v, ok := o.Signals[key]
		if !ok || v < 0 || v > 1 {
			t.Fatalf("signal %s = %v ok=%v", key, v, ok)
		}
	}
}
