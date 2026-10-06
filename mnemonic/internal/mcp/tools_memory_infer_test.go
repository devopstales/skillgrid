package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/service"
	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// inferSaveFixture pins a project + service over a temp data dir and seeds an
// active session, returning the session id for a mem_save call. It mirrors the
// pinProjectCwd pattern every project-scoped handler test uses.
func inferSaveFixture(t *testing.T) (string, string) {
	t.Helper()
	dataDir := t.TempDir()
	dir := t.TempDir()
	proj := pinProjectCwd(t, dataDir, dir, "infer-probe")
	sid, _, err := service.New(dataDir).SessionStart(context.Background(), dir, "infer")
	if err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return sid, proj
}

// savedTypeTopic reads the type/topic_key a mem_save round-trip persisted from
// the saved id, by reading the observation back through mem_get_observation as
// the saving session's owner (private-by-default per-owner gating, 013 step 01).
func savedTypeTopic(t *testing.T, res *mcplib.CallToolResult, owner string) (string, string) {
	t.Helper()
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("unmarshal mem_save result: %v (text %s)", err, callResultText(t, res))
	}
	get := newCallTool("mem_get_observation", map[string]any{"id": float64(out.ID), "reader_owner": owner})
	gres, err := handleMemGetObservation(context.Background(), get)
	if err != nil {
		t.Fatalf("mem_get_observation: %v", err)
	}
	if gres.IsError {
		t.Fatalf("mem_get_observation errored: %s", callResultText(t, gres))
	}
	var obs struct {
		Type     string `json:"type"`
		TopicKey string `json:"topic_key"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, gres)), &obs); err != nil {
		t.Fatalf("unmarshal observation: %v (text %s)", err, callResultText(t, gres))
	}
	return obs.Type, obs.TopicKey
}

// TestMemSaveInferDefaultOff is the "mem-save-infer-off-by-default" scenario:
// with infer unset the heuristic does NOT run. A non-empty type is preserved
// verbatim and no topic_key is invented (the agent-provided metadata is stored
// byte-identical to today's behaviour).
func TestMemSaveInferDefaultOff(t *testing.T) {
	sid, _ := inferSaveFixture(t)
	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "default off save",
		"type":       "decision",
		"content":    "infer unset must not run the heuristic",
		"session_id": sid,
	}))
	if err != nil {
		t.Fatalf("handleMemSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, res))
	}
	typ, topic := savedTypeTopic(t, res, sid)
	if typ != "decision" {
		t.Errorf("type changed with infer unset: %q", typ)
	}
	if topic != "" {
		t.Errorf("topic_key invented with infer unset: %q", topic)
	}
}

// TestMemSaveInferFillsTopicKey is the "mem-save-infer-fills-topic-key"
// scenario: with infer=true an empty topic_key is filled through the same
// SuggestTopicKey seam the mem_suggest_topic_key tool uses (byte-identical),
// and the provided type is preserved.
func TestMemSaveInferFillsTopicKey(t *testing.T) {
	sid, _ := inferSaveFixture(t)
	res, err := handleMemSave(context.Background(), newCallTool("mem_save", map[string]any{
		"title":      "use sqlite for the store",
		"type":       "decision",
		"content":    "we decided to use SQLite for the store",
		"session_id": sid,
		"infer":      true,
	}))
	if err != nil {
		t.Fatalf("handleMemSave: %v", err)
	}
	if res.IsError {
		t.Fatalf("mem_save errored: %s", callResultText(t, res))
	}
	typ, topic := savedTypeTopic(t, res, sid)
	if typ != "decision" {
		t.Errorf("provided type overwritten: %q", typ)
	}
	want := SuggestTopicKey("decision", "use sqlite for the store", "we decided to use SQLite for the store")
	if topic != want {
		t.Errorf("topic_key = %q, want %q (the SuggestTopicKey seam)", topic, want)
	}
	if !strings.HasPrefix(topic, "decision/") {
		t.Errorf("topic_key %q missing family prefix", topic)
	}
}
