// Scope + target params on the MCP fact_* tools (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-02).
//
// RED notes: these tests call the handlers with scope/target arguments the
// registered tools do not declare yet, and assert the C3 trail split
// (fact in __global__.sqlite, session_events trail in the project store).
// Before the change, scope/target are silently ignored by the handlers, so
// the global-scoped assertions fail: the fact lands in the project store,
// not the global store.
package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// factToolInputNames returns the input property names declared by a fact_*
// tool's JSON schema (via the real tools/list handler).
func factToolInputNames(t *testing.T, name string) map[string]bool {
	t.Helper()
	msg, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": map[string]any{},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp := NewServer().HandleMessage(context.Background(), msg)
	if resp == nil {
		t.Fatal("nil response to tools/list")
	}
	var out struct {
		Result struct {
			Tools []struct {
				Name        string `json:"name"`
				InputSchema struct {
					Properties map[string]any `json:"properties"`
				} `json:"inputSchema"`
			} `json:"tools"`
		} `json:"result"`
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("parse tools/list: %v", err)
	}
	for _, tool := range out.Result.Tools {
		if tool.Name == name {
			names := make(map[string]bool, len(tool.InputSchema.Properties))
			for p := range tool.InputSchema.Properties {
				names[p] = true
			}
			return names
		}
	}
	t.Fatalf("tool %s not listed", name)
	return nil
}

// TestFactToolsExposeScopeParam (G6): all five fact_* tools declare the
// scope param; the write ops that carry a target (fact_add) declare it.
// ponytail: fact_search/fact_forget/fact_decay/fact_decay_all do not take a
// target (category is a write-time attribute of the fact row; the read/decay
// ops operate on the whole store regardless of category — the fused search
// in a later change adds category as a read filter if needed).
func TestFactToolsExposeScopeParam(t *testing.T) {
	for _, name := range []string{"fact_add", "fact_search", "fact_forget", "fact_decay", "fact_decay_all"} {
		props := factToolInputNames(t, name)
		if !props["scope"] {
			t.Errorf("%s: scope param not declared", name)
		}
	}
	props := factToolInputNames(t, "fact_add")
	if !props["target"] {
		t.Errorf("fact_add: target param not declared")
	}
}

// TestFactAddGlobalScopeMCP (BDD: "happy path — global fact add writes to
// global store with project trail", MCP layer): fact_add(scope=global,
// target=memory) writes the fact to __global__.sqlite and trails to the
// caller's project store.
func TestFactAddGlobalScopeMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-global-add"
	seedFactMCPSession(t, st, project, dataDir, sid)

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    "mcp global note: deploy via flux",
		"scope":      "global",
		"target":     "memory",
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	factID, ok := out["fact_id"].(float64)
	if !ok || factID <= 0 {
		t.Fatalf("no positive fact_id in %s", callResultText(t, res))
	}

	// The fact row is in the GLOBAL store, tagged with the target category.
	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	var category string
	if err := gst.DB.QueryRow(`
		SELECT category FROM facts WHERE id = ?`, int64(factID)).Scan(&category); err != nil {
		t.Fatalf("fact %d not in the global store: %v", int64(factID), err)
	}
	if category != "memory" {
		t.Errorf("global fact category = %q, want 'memory'", category)
	}

	// The trail row is in the PROJECT store (C3), not the global store.
	// The cross-store trail payload carries the target category (the fact id
	// is allocated by the global store and not observable in the trail row —
	// see the facts.AddWith cross-store contract).
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).Scan(&payload); err != nil {
		t.Fatalf("fact_add trail missing from the project store: %v", err)
	}
	if !strings.Contains(payload, `"category":"memory"`) {
		t.Errorf("trail payload %q does not carry the target category", payload)
	}
	var globalTrails int
	if err := gst.DB.QueryRow(`
		SELECT COUNT(*) FROM session_events WHERE action_type = 'fact_add'`).Scan(&globalTrails); err != nil {
		t.Fatalf("count global trails: %v", err)
	}
	if globalTrails != 0 {
		t.Errorf("global store holds %d fact_add trails, want 0", globalTrails)
	}
}

// TestFactAddGlobalUserTargetMCP (BDD: "edge — global fact add with target
// user uses user category", MCP layer): target=user stamps category='user' in
// the global store.
func TestFactAddGlobalUserTargetMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-global-user"
	seedFactMCPSession(t, st, project, dataDir, sid)

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    "mcp user profile: prefers concise answers",
		"scope":      "global",
		"target":     "user",
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	factID := int64(out["fact_id"].(float64))

	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	var category string
	if err := gst.DB.QueryRow(`
		SELECT category FROM facts WHERE id = ?`, factID).Scan(&category); err != nil {
		t.Fatalf("fact %d not in the global store: %v", factID, err)
	}
	if category != "user" {
		t.Errorf("global user-target fact category = %q, want 'user'", category)
	}
}

// TestFactAddProjectScopeMCP (BDD: "happy path — project fact add writes to
// project store (unchanged)", MCP layer): an explicit scope=project behaves
// exactly like the pre-change default — project store only, no global row.
func TestFactAddProjectScopeMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-proj-add"
	seedFactMCPSession(t, st, project, dataDir, sid)

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    "mcp project note stays local",
		"scope":      "project",
		"target":     "memory",
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_add error: %s", callResultText(t, res))
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	factID := int64(out["fact_id"].(float64))

	var category string
	if err := st.DB.QueryRow(`
		SELECT category FROM facts WHERE id = ?`, factID).Scan(&category); err != nil {
		t.Fatalf("fact %d not in the project store: %v", factID, err)
	}
	if category != "memory" {
		t.Errorf("project fact category = %q, want 'memory'", category)
	}

	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	var globalFacts int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if globalFacts != 0 {
		t.Errorf("global store holds %d facts, want 0 for a project-scoped add", globalFacts)
	}
}

// TestFactAddGlobalUnknownSessionMCP (BDD: "failure — global fact add with
// unknown session fails cleanly", MCP layer): the session must exist in the
// PROJECT store (where the trail is written); an unknown session is a clean
// tool error with no partial write in the global store.
func TestFactAddGlobalUnknownSessionMCP(t *testing.T) {
	dataDir, project, _ := openFactMCPStore(t)

	res, err := handleFactAdd(context.Background(), callReq("fact_add", map[string]any{
		"project":    project,
		"session_id": "sess-mcp-no-such-session",
		"content":    "mcp fact that must not land",
		"scope":      "global",
		"target":     "memory",
	}))
	if err != nil {
		t.Fatalf("handleFactAdd: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected a tool error for an unknown session, got: %s", callResultText(t, res))
	}
	text := callResultText(t, res)
	if !strings.Contains(text, "sess-mcp-no-such-session") {
		t.Errorf("error %q does not name the missing session", text)
	}

	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	var globalFacts int
	if err := gst.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&globalFacts); err != nil {
		t.Fatalf("count global facts: %v", err)
	}
	if globalFacts != 0 {
		t.Errorf("global store holds %d facts after a failed add, want 0", globalFacts)
	}
}

// TestFactSearchGlobalScopeMCP: a fact_search with scope=global reads the
// global store — it finds the global fact, and its trail lands in the
// project store (C3 applies to reads' trails too).
func TestFactSearchGlobalScopeMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-global-search"
	seedFactMCPSession(t, st, project, dataDir, sid)

	// Seed a global fact through the handler.
	factAddIDWith(t, project, sid, "searchable mcp global fact about kubernetes", "global", "memory")

	res, err := handleFactSearch(context.Background(), callReq("fact_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "kubernetes",
		"scope":      "global",
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
		t.Fatalf("count = %v, want 1 (the global fact)", out["count"])
	}

	// The search trail is in the project store.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).Scan(&payload); err != nil {
		t.Fatalf("fact_search trail missing from the project store: %v", err)
	}

	// The fact is NOT in the project store: a project-scope search must not
	// see it (store isolation).
	res, err = handleFactSearch(context.Background(), callReq("fact_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "kubernetes",
		"scope":      "project",
	}))
	if err != nil {
		t.Fatalf("handleFactSearch project: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_search error: %s", callResultText(t, res))
	}
	var out2 map[string]any
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out2); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out2["count"] != float64(0) {
		t.Errorf("project-scope search count = %v, want 0 (the fact lives in the global store)", out2["count"])
	}
}

// TestFactForgetDecayGlobalScopeMCP: fact_forget and fact_decay with
// scope=global operate on the global store, and their trails land in the
// project store.
func TestFactForgetDecayGlobalScopeMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-global-forget-decay"
	seedFactMCPSession(t, st, project, dataDir, sid)

	id := factAddIDWith(t, project, sid, "mcp global fact to forget and decay", "global", "memory")

	// Forget in the global store.
	res, err := handleFactForget(context.Background(), callReq("fact_forget", map[string]any{
		"project":    project,
		"session_id": sid,
		"fact_id":    int(id),
		"scope":      "global",
	}))
	if err != nil {
		t.Fatalf("handleFactForget: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_forget error: %s", callResultText(t, res))
	}
	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	if !sqlNullCheck(t, gst, id) {
		t.Fatalf("global fact %d not soft-deleted by scope=global forget", id)
	}

	// Decay in the global store (a fresh fact decays to ~1.0, but the op must
	// hit the global row and trail to the project store).
	res, err = handleFactDecay(context.Background(), callReq("fact_decay", map[string]any{
		"project":    project,
		"session_id": sid,
		"fact_id":    int(id),
		"scope":      "global",
	}))
	if err != nil {
		t.Fatalf("handleFactDecay: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_decay error: %s", callResultText(t, res))
	}
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay'`, sid).Scan(&payload); err != nil {
		t.Fatalf("fact_decay trail missing from the project store: %v", err)
	}

	// Decay in the PROJECT store must not find the global fact id.
	res, err = handleFactDecay(context.Background(), callReq("fact_decay", map[string]any{
		"project":    project,
		"session_id": sid,
		"fact_id":    int(id),
		"scope":      "project",
	}))
	if err != nil {
		t.Fatalf("handleFactDecay project: %v", err)
	}
	if !res.IsError {
		t.Errorf("project-scope decay of a global fact id should error (unknown fact), got: %s", callResultText(t, res))
	}
}

// TestFactDecayAllGlobalScopeMCP: fact_decay_all with scope=global decays the
// global store's facts, trails to the project store, and leaves the project
// store's facts untouched.
func TestFactDecayAllGlobalScopeMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-global-decay-all"
	seedFactMCPSession(t, st, project, dataDir, sid)

	factAddIDWith(t, project, sid, "mcp global decay-all one", "global", "memory")
	factAddIDWith(t, project, sid, "mcp global decay-all two", "global", "memory")
	localID := factAddIDWith(t, project, sid, "mcp project decay-all stays", "project", "memory")

	res, err := handleFactDecayAll(context.Background(), callReq("fact_decay_all", map[string]any{
		"project":    project,
		"session_id": sid,
		"scope":      "global",
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
	if out["decayed"] != float64(2) {
		t.Errorf("decayed = %v, want 2 (only the global store's facts)", out["decayed"])
	}

	// Trail in the project store.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_decay_batch'`, sid).Scan(&payload); err != nil {
		t.Fatalf("fact_decay_batch trail missing from the project store: %v", err)
	}

	// The project fact was not touched: its updated_at equals created_at
	// (a no-op decay would still rewrite it if the scope leaked).
	var created, updated string
	if err := st.DB.QueryRow(`
		SELECT created_at, updated_at FROM facts WHERE id = ?`, localID).Scan(&created, &updated); err != nil {
		t.Fatalf("read project fact: %v", err)
	}
	if created != updated {
		t.Errorf("project fact %d was rewritten by a global-scope decay_all (created=%q updated=%q)", localID, created, updated)
	}
}

// factAddIDWith drives the fact_add handler with explicit scope/target and
// returns the new fact id.
func factAddIDWith(t *testing.T, project, sid, content, scope, target string) int64 {
	t.Helper()
	args := map[string]any{
		"project":    project,
		"session_id": sid,
		"content":    content,
	}
	if scope != "" {
		args["scope"] = scope
	}
	if target != "" {
		args["target"] = target
	}
	res, err := handleFactAdd(context.Background(), callReq("fact_add", args))
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

// sqlNullCheck reports whether the global fact's deleted_at is set (the soft
// delete landed in the global store).
func sqlNullCheck(t *testing.T, gst *store.Store, id int64) bool {
	t.Helper()
	var n int
	if err := gst.DB.QueryRow(`
		SELECT COUNT(*) FROM facts WHERE id = ? AND deleted_at IS NOT NULL`, id).Scan(&n); err != nil {
		t.Fatalf("check deleted_at: %v", err)
	}
	return n == 1
}

// TestFactSearchScopeAllMCP (BDD: "happy path — fused search returns hits
// from both stores"): fact_search(scope=all) is the fused-search read path —
// it searches both stores, returns every hit labeled with scope + category,
// and trails to the project store only (never the global store).
func TestFactSearchScopeAllMCP(t *testing.T) {
	dataDir, project, st := openFactMCPStore(t)
	sid := "sess-mcp-scope-all"
	seedFactMCPSession(t, st, project, dataDir, sid)

	factAddIDWith(t, project, sid, "mcp global note about responses", "global", "memory")
	factAddIDWith(t, project, sid, "mcp project note about responses", "project", "memory")

	res, err := handleFactSearch(context.Background(), callReq("fact_search", map[string]any{
		"project":    project,
		"session_id": sid,
		"query":      "responses",
		"scope":      "all",
	}))
	if err != nil {
		t.Fatalf("handleFactSearch scope=all: %v", err)
	}
	if res.IsError {
		t.Fatalf("fact_search scope=all error: %s", callResultText(t, res))
	}
	var out struct {
		Facts []struct {
			Content  string `json:"content"`
			Scope    string `json:"scope"`
			Category string `json:"category"`
		} `json:"facts"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal([]byte(callResultText(t, res)), &out); err != nil {
		t.Fatalf("parse result: %v", err)
	}
	if out.Count != 2 {
		t.Fatalf("count = %d, want 2 (one per store)", out.Count)
	}
	// Every hit is labeled scope + category; the two scopes are both present.
	seen := map[string]bool{}
	for _, f := range out.Facts {
		if f.Scope != "global" && f.Scope != "project" {
			t.Errorf("hit scope = %q, want global|project", f.Scope)
		}
		if f.Category != "memory" {
			t.Errorf("hit category = %q, want memory", f.Category)
		}
		seen[f.Scope] = true
	}
	if !seen["global"] || !seen["project"] {
		t.Errorf("hits do not cover both stores: %v", seen)
	}

	// The trail is in the PROJECT store (mode fused) and nowhere in the
	// global store.
	var payload string
	if err := st.DB.QueryRow(`
		SELECT payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_search'`, sid).Scan(&payload); err != nil {
		t.Fatalf("fact_search trail missing from the project store: %v", err)
	}
	if !strings.Contains(payload, `"mode":"fused"`) {
		t.Errorf("trail payload %q does not carry mode fused", payload)
	}
	gst, err := store.Open(dataDir, "__global__")
	if err != nil {
		t.Fatalf("open global store: %v", err)
	}
	defer gst.Close()
	var globalTrails int
	if err := gst.DB.QueryRow(
		`SELECT COUNT(*) FROM session_events WHERE action_type = 'fact_search'`,
	).Scan(&globalTrails); err != nil {
		t.Fatalf("count global trails: %v", err)
	}
	if globalTrails != 0 {
		t.Errorf("global store holds %d fact_search trail rows, want 0", globalTrails)
	}
}
