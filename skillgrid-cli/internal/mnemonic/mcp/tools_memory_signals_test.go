package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
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

// TestMemSearchEmbedderError enters the mem_search branch that drops an
// EmbedQuery error. An external embedder with no base URL fails before any
// network call. The search still returns the keyword floor, and the failed
// call does not write a query_cache row.
func TestMemSearchEmbedderError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("MNEMONIC_EMBED", "1")
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatalf("mkdir .skillgrid/config.d: %v", err)
	}
	yaml := "mnemonic:\n  embedder:\n    provider: external\n    model: fail-model\n"
	if err := os.WriteFile(filepath.Join(root, ".skillgrid", "config.d", "indexing.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("write indexing: %v", err)
	}
	t.Chdir(root)

	dataDir := t.TempDir()
	const projectID = "emb-err-probe"
	t.Setenv("MNEMONIC_PROJECT", projectID)
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })

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
	if o.Signals["vector"] != 0 {
		t.Fatalf("signals.vector = %v", o.Signals["vector"])
	}

	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	var n int
	if err := h.Store().DB.QueryRow(`SELECT COUNT(*) FROM query_cache`).Scan(&n); err != nil {
		t.Fatalf("count cache: %v", err)
	}
	if n != 0 {
		t.Fatalf("query_cache rows = %d, want 0 (embedder error must not be stored)", n)
	}
}

// TestMemSearchAllProjectsSignalsAreKeywordOnly: the all_projects path has no
// ranker, so it reports matched_via=keyword with only the keyword rank proxy
// set. Recency, decay, importance, vector and entity are 0 rather than
// invented values.
func TestMemSearchAllProjectsSignalsAreKeywordOnly(t *testing.T) {
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
		"query": "banana", "all_projects": true, "reader_owner": sid,
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
	if o.Signals["keyword"] != 1 {
		t.Fatalf("signals.keyword = %v, want 1 (rank proxy for first hit)", o.Signals["keyword"])
	}
	for _, key := range []string{"vector", "recency", "entity", "decay", "importance"} {
		v, ok := o.Signals[key]
		if !ok || v != 0 {
			t.Fatalf("signals.%s = %v ok=%v, want 0", key, v, ok)
		}
	}
}
