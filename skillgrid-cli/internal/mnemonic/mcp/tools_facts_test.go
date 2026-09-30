package mcp

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// TestFactAddRegistered covers @step-02: fact_add is listed alongside the
// unchanged mem_* tools.
func TestFactAddRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["fact_add"]; !ok {
		t.Fatal("fact_add not registered")
	}
	if _, ok := tools["mem_save"]; !ok {
		t.Fatal("mem_save missing after fact tools registered")
	}
}

// TestFactAddMCP covers @step-02: the tool inserts a fact, returns its id, and
// records a session_events row with action_type=fact_add and the fact id.
func TestFactAddMCP(t *testing.T) {
	dataDir := t.TempDir()
	project := "factmcp"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })

	sid := "sess-fact-mcp"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	req := mcplib.CallToolRequest{}
	req.Params.Name = "fact_add"
	req.Params.Arguments = map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    "MCP fact: session events trail must carry the fact id",
	}
	res, err := handleFactAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add error: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		t.Fatalf("parse result %q: %v", text, err)
	}
	factID, ok := out["fact_id"].(float64)
	if !ok || factID <= 0 {
		t.Fatalf("no positive fact_id in %s", text)
	}

	// Session trail: action_type=fact_add with the fact id in the payload.
	var actionType, payload string
	if err := st.DB.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_add session event missing: %v", err)
	}
	if !strings.Contains(payload, `"fact_id":`) {
		t.Errorf("payload %q does not carry the fact id", payload)
	}
	st.Close()
}

// TestFactSearchRegistered covers @step-02: the three new fact tools are
// listed alongside the unchanged mem_* tools.
func TestFactSearchRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	for _, name := range []string{"fact_search", "fact_forget", "fact_decay"} {
		if _, ok := tools[name]; !ok {
			t.Fatalf("%s not registered", name)
		}
	}
	if _, ok := tools["mem_save"]; !ok {
		t.Fatal("mem_save missing after fact tools registered")
	}
}

func openFactMCPStore(t *testing.T) (dataDir, project string, st *store.Store) {
	t.Helper()
	dataDir = t.TempDir()
	project = "factmcp3"
	t.Setenv("MNEMONIC_PROJECT", project)
	var err error
	st, err = store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { SetService(nil); st.Close() })
	SetService(service.New(dataDir))
	return dataDir, project, st
}

func seedFactMCPSession(t *testing.T, st *store.Store, project, dataDir, sid string) {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`, sid, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

// TestFactSearchMCP covers @step-02 (Fact tools add search and record a
// session event): the tool returns matching facts and records a
// session_events row with action_type=fact_search and the matched fact ids.
func TestFactSearchMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-fact-mcp-search"
	seedFactMCPSession(t, st, project, dataDir, sid)

	id := factAddID(t, project, sid, "searchable mcp fact about sqlite")

	res, err := handleFactSearch(context.Background(), callReq("fact_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "sqlite",
	}))
	if err != nil {
		t.Fatalf("handleFactSearch: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_search error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out["count"] != float64(1) {
		t.Fatalf("count = %v, want 1", out["count"])
	}

	// Session trail: action_type=fact_search with the matched fact id.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).
		Scan(&payload); err != nil {
		t.Fatalf("fact_search session event missing: %v", err)
	}
	if !strings.Contains(payload, `"fact_ids":[`) || !strings.Contains(payload, strconv.FormatInt(id, 10)) {
		t.Errorf("payload %q does not carry the matched fact id", payload)
	}
}

// TestFactForgetMCP covers @step-02 (Soft-deleted fact absent from default
// search, tool portion): forget soft-deletes, and a following default search
// no longer returns the fact.
func TestFactForgetMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-fact-mcp-forget"
	seedFactMCPSession(t, st, project, dataDir, sid)

	id := factAddID(t, project, sid, "mcp fact that will be forgotten")

	res, err := handleFactForget(context.Background(), callReq("fact_forget", map[string]any{
		"project":    project,
		"session_id": sid,
		"fact_id":    int(id),
	}))
	if err != nil {
		t.Fatalf("handleFactForget: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_forget error: %s", callResultText(t, res))
	}

	// The default search is now clean of the forgotten fact.
	res, err = handleFactSearch(context.Background(), callReq("fact_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "forgotten",
	}))
	if err != nil {
		t.Fatalf("handleFactSearch: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out["count"] != float64(0) {
		t.Fatalf("count = %v, want 0 (soft-deleted absent from default search)", out["count"])
	}
}

// TestFactDecayMCP covers @step-02 (Decay lowers importance and logs events,
// tool portion): the tool applies the AKL decay and records the event trail.
func TestFactDecayMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-fact-mcp-decay"
	seedFactMCPSession(t, st, project, dataDir, sid)

	id := factAddID(t, project, sid, "mcp fact that will decay")

	res, err := handleFactDecay(context.Background(), callReq("fact_decay", map[string]any{
		"project":    project,
		"session_id": sid,
		"fact_id":    int(id),
	}))
	if err != nil {
		t.Fatalf("handleFactDecay: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_decay error: %s", callResultText(t, res))
	}

	// Session trail: action_type=fact_decay with the fact id.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay'`, sid).
		Scan(&payload); err != nil {
		t.Fatalf("fact_decay session event missing: %v", err)
	}
	if !strings.Contains(payload, strconv.FormatInt(id, 10)) {
		t.Errorf("payload %q does not carry the fact id", payload)
	}
}

// TestFactDecayAllRegistered covers acceptance scenario 5 (tool portion):
// fact_decay_all is listed alongside the other fact tools.
func TestFactDecayAllRegistered(t *testing.T) {
	s := NewServer()
	tools := s.ListTools()
	if _, ok := tools["fact_decay_all"]; !ok {
		t.Fatal("fact_decay_all not registered")
	}
}

// TestFactDecayAllMCP covers acceptance scenario 5 (tool portion): the tool
// decays every live fact in the session, purges the ones below the threshold,
// returns the counts, and records a fact_decay_batch session event.
func TestFactDecayAllMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-fact-mcp-decay-all"
	seedFactMCPSession(t, st, project, dataDir, sid)

	// Two fresh facts that survive the decay pass.
	factAddID(t, project, sid, "mcp decay-all fact one that survives")
	factAddID(t, project, sid, "mcp decay-all fact two that survives")
	// One old fact with a decay rate that drops it below the 0.5 threshold.
	oldID := factAddID(t, project, sid, "mcp decay-all old fact that is purged")

	now := time.Now().UTC()
	backdate := now.Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		UPDATE facts SET created_at = ?, recency_decay = ?
		WHERE id = ?`, backdate, 0.1, oldID); err != nil {
		t.Fatalf("backdate old fact: %v", err)
	}

	res, err := handleFactDecayAll(context.Background(), callReq("fact_decay_all", map[string]any{
		"project":    project,
		"session_id": sid,
		"threshold":  0.5,
	}))
	if err != nil {
		t.Fatalf("handleFactDecayAll: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_decay_all error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out["decayed"] != float64(3) {
		t.Errorf("decayed = %v, want 3", out["decayed"])
	}
	if out["purged"] != float64(1) {
		t.Errorf("purged = %v, want 1", out["purged"])
	}
	if out["threshold"] != float64(0.5) {
		t.Errorf("threshold = %v, want 0.5", out["threshold"])
	}

	// Session trail: action_type=fact_decay_batch with the counts.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay_batch'`, sid).
		Scan(&payload); err != nil {
		t.Fatalf("fact_decay_batch session event missing: %v", err)
	}
	if !strings.Contains(payload, `"decayed":3`) || !strings.Contains(payload, `"purged":1`) {
		t.Errorf("payload %q does not carry the correct counts", payload)
	}
}

// factAddID drives the fact_add handler and returns the new fact id.
func factAddID(t *testing.T, project, sid, content string) int64 {
	t.Helper()
	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    content,
	}))
	if err != nil {
		t.Fatalf("fact_add: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add tool error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse fact_add result: %v", err)
	}
	id, _ := out["fact_id"].(float64)
	if id <= 0 {
		t.Fatalf("no positive fact_id in %v", out)
	}
	return int64(id)
}

// TestFactAddRequiresContent: a missing/blank content is a clean tool error.
func TestFactAddRequiresContent(t *testing.T) {
	dataDir := t.TempDir()
	project := "factmcp2"
	t.Setenv("MNEMONIC_PROJECT", project)
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatal(err)
	}
	SetService(service.New(dataDir))
	t.Cleanup(func() { SetService(nil) })
	defer st.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-nocontent', ?, ?, ?, 'active')`, project, dataDir, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	req := mcplib.CallToolRequest{}
	req.Params.Name = "fact_add"
	req.Params.Arguments = map[string]any{
		"project":    project,
		"session_id": "sess-nocontent",
		"content":    "",
	}
	res, err := handleFactAdd(context.Background(), req)
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for blank content")
	}
}
