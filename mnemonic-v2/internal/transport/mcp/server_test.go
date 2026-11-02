package mcp

// server_test.go — the MCP stdio server: the write queue (spec
// 004-transport-mcp §4), the degradation envelope (INV-3), and the stdio
// JSON-RPC round trip (the harness handshake the plugins speak).

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

func TestWriteQueueSerializes(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			s.CallTool(ctx, "mem_save", saveArgs(i))
		}(i)
	}
	wg.Wait()
	if !s.WritesSerialized() {
		t.Fatalf("concurrent writes must be serialized by the queue")
	}
	// The load-bearing assertion (fix round 1, IMPORTANT #3): the store's
	// transaction seam shows exactly 8 transactions in a strictly
	// monotonic order — one per write, none interleaved. A bool flag
	// proves a write ran, not that writes serialized.
	st := s.h.Service().St()
	st.SetTxnHook(func() {}) // detach the hook (the writes already ran)
	txns := s.txnOrder()
	if len(txns) != 8 {
		t.Fatalf("8 concurrent saves must commit exactly 8 transactions, got %d", len(txns))
	}
	for i, n := range txns {
		if n != i+1 {
			t.Fatalf("transaction %d is %d — the store must serialize writes (monotonic order)", i, n)
		}
	}
}

func TestEnvelopeCarriesDegraded(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent) // nil seams
	out := s.CallTool(ctx, "mem_save", saveArgs(1))
	var env struct {
		Degraded       bool   `json:"degraded"`
		DegradedReason string `json:"degraded_reason"`
	}
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if !env.Degraded || env.DegradedReason == "" {
		t.Fatalf("degraded envelope missing: %s", out)
	}
}

// TestEnvelopeCarriesDegradedWhenStoreFails (fix round 1, IMPORTANT #6)
// adds the missing half of the envelope-degraded contract: a failing
// store (not an absent LLM seam) must also surface on the wire as an
// error envelope (the envelope's not-healthy half), never a bare crash or
// an empty result.
func TestEnvelopeCarriesDegradedWhenStoreFails(t *testing.T) {
	st := openTestStore(t, t.TempDir())
	_ = st.Close() // a closed store: every read/write errors
	s := NewServer(newTestHandlerWithStore(t, st), Agent)
	out := s.CallTool(ctx, "mem_search", map[string]any{"query": "x", "project": testProject})
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["error"] != true {
		t.Fatalf("a store failure must be an error envelope on the wire, got %s", out)
	}
	code, _ := env["code"].(string)
	msg, _ := env["message"].(string)
	if code == "" || msg == "" {
		t.Fatalf("error envelope must carry code + message: %s", out)
	}
	if !strings.Contains(msg, "closed") && !strings.Contains(strings.ToLower(msg), "sql") {
		t.Fatalf("the store error must be visible in the message: %s", out)
	}
}

func TestStdioRoundTrip(t *testing.T) {
	// in-process: pipe the server's stdio, speak JSON-RPC, assert a valid tool result frame
	s := NewServer(newTestHandler(t), Agent)
	frames, err := s.HandshakeAndCall(ctx, "mem_current_project") // test harness: initialize → tools/call
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) == 0 || !json.Valid(frames[0]) {
		t.Fatalf("no valid JSON-RPC result frame")
	}
	// Fix round 1, IMPORTANT #1: the returned project must be the
	// fixture's real resolved project (the detector's answer), not just
	// valid JSON.
	rpc := parseToolCallFrame(t, frames[len(frames)-1])
	content, _ := rpc["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("tools/call frame carries no content: %s", frames[len(frames)-1])
	}
	text, _ := content[0].(map[string]any)["text"].(string)
	var env map[string]any
	if err := json.Unmarshal([]byte(text), &env); err != nil {
		t.Fatalf("tool result is not JSON: %v (%s)", err, text)
	}
	if env["project"] != testProject {
		t.Fatalf("stdio mem_current_project = %v, want the fixture's resolved %q", env["project"], testProject)
	}
}

// parseToolCallFrame unmarshals the last handshake frame and returns its
// tools/call result object (the mcp-go CallToolResult shape).
func parseToolCallFrame(t *testing.T, frame []byte) map[string]any {
	t.Helper()
	var rpc struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(frame, &rpc); err != nil {
		t.Fatalf("frame is not JSON: %v (%s)", err, frame)
	}
	if rpc.Result == nil {
		t.Fatalf("frame carries no result: %s", frame)
	}
	return rpc.Result
}

// ── triangulation (beyond the brief's set) ─────────────────────────────────

// TestUnknownToolIsEnvelopeError proves the error shape on the wire: an
// unknown tool is a JSON-RPC error frame (never a crash, never a nil frame),
// and the server keeps serving after it (the queue and the registry survive).
func TestUnknownToolIsEnvelopeError(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	frames, err := s.HandshakeAndCall(ctx, "mem_not_a_tool")
	if err != nil {
		t.Fatal(err)
	}
	// HandshakeAndCall returns the initialize frame + the tools/call frame
	// (2 frames). The tools/call frame (the last one) must be the error.
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames (initialize + tools/call), got %d", len(frames))
	}
	var rpc struct {
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(frames[len(frames)-1], &rpc); err != nil {
		t.Fatalf("frame is not JSON: %v (%s)", err, frames[len(frames)-1])
	}
	if rpc.Error == nil || rpc.Error.Code == 0 || rpc.Error.Message == "" {
		t.Fatalf("unknown tool must be a JSON-RPC error frame with a code and message, got %s", frames[len(frames)-1])
	}
	// The server is still healthy: a follow-up call on the same instance works.
	out := s.CallTool(ctx, "mem_current_project", map[string]any{})
	if !json.Valid(out) {
		t.Fatalf("server unhealthy after unknown-tool call: %s", out)
	}
}

// TestReadsBypassTheQueue proves the queue covers writes only: a read tool
// completes without touching the write queue (a dead worker goroutine
// cannot stall reads). A test-only deadlock switch stands in for a crashed
// worker: if the read path enqueued, this would hang (the test times out).
func TestReadsBypassTheQueue(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	s.SetDeadlockForTest()
	out := s.CallTool(ctx, "mem_current_project", map[string]any{})
	if !json.Valid(out) {
		t.Fatalf("read tool did not bypass the write queue: %s", out)
	}
}

// TestWriteQueueDrainsAfterError proves a failed write does not wedge the
// queue: a save that errors (missing required arg) is answered with the
// error envelope, and the next save on the same server still succeeds.
func TestWriteQueueDrainsAfterError(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	bad := s.CallTool(ctx, "mem_save", map[string]any{"title": "no content", "type": "decision"})
	if !json.Valid(bad) {
		t.Fatalf("failed write must still be valid JSON: %s", bad)
	}
	good := s.CallTool(ctx, "mem_save", saveArgs(99))
	var env struct {
		SyncID   string `json:"sync_id"`
		Degraded bool   `json:"degraded"`
	}
	if err := json.Unmarshal(good, &env); err != nil {
		t.Fatalf("second save envelope is not JSON: %v (%s)", err, good)
	}
	if env.SyncID == "" {
		t.Fatalf("queue wedged after a failed write: %s", good)
	}
}

// TestJudgeManyWritesOnceInQueue proves the mem_judge_many contract (spec
// 004-transport-mcp §4: "mem_judge_many is one queued write"): the batch
// occupies the write queue exactly once, not once per judgment.
//
// Fix round 1 (MINOR #3): the test previously passed `judgments` as a
// native []map[string]any, but dispatchJudgeMany reads it as a JSON array
// string — so the call returned the "must be a JSON array string" error
// and `got < 3` passed vacuously. Now the JSON string is passed (the D1
// contract) and the store's transaction seam counts the committed
// transactions: one queued write = one transaction.
func TestJudgeManyWritesOnceInQueue(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	obs1 := s.CallTool(ctx, "mem_save", saveArgs(1))
	obs2 := s.CallTool(ctx, "mem_save", saveArgs(2))
	if !json.Valid(obs1) || !json.Valid(obs2) {
		t.Fatalf("seeding saves failed: %s / %s", obs1, obs2)
	}
	// Seed real pending relations for the judgment_ids (the store seam —
	// the same fixture pattern the handler's Judge test uses).
	st := s.h.Service().St()
	rel1, err := st.AddRelation(ctx, store.AddRelationParams{SourceID: "obs-a", TargetID: "obs-b", Relation: "pending", SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("seed rel1: %v", err)
	}
	rel2, err := st.AddRelation(ctx, store.AddRelationParams{SourceID: "obs-c", TargetID: "obs-d", Relation: "pending", SessionID: "sess-1"})
	if err != nil {
		t.Fatalf("seed rel2: %v", err)
	}
	baseTxns := s.txnCount()
	judgments := fmt.Sprintf(`[{"judgment_id":%d,"relation":"conflicts_with","reason":"r1"},{"judgment_id":%d,"relation":"refines","reason":"r2"}]`, rel1.ID, rel2.ID)
	out := s.CallTool(ctx, "mem_judge_many", map[string]any{
		"judgments": judgments,
		"model":     "test-model",
	})
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("judge_many envelope is not JSON: %v (%s)", err, out)
	}
	if env["error"] == true {
		t.Fatalf("judge_many returned an error envelope (the test must pass the D1 JSON string, not a native array): %s", out)
	}
	// The batch is ONE queued write = ONE transaction (the store seam
	// counts it — a per-judgment enqueue would commit more).
	if got := s.txnCount() - baseTxns; got != 1 {
		t.Fatalf("judge_many must commit exactly one transaction (one queued write), got %d", got)
	}
}

// TestCurrentProjectReturnsDetectorResolution (fix round 1, IMPORTANT #1;
// round 3: distinct fixture) pins that mem_current_project returns the
// detector's resolution, not a hardcoded literal: this test's detector
// resolves to distinctProject ("fixture-project" — NOT the pre-fix "p"
// literal, and NOT the shared testProject), so a regression to the
// literal "p" fails the assertion. An explicit selector wins.
func TestCurrentProjectReturnsDetectorResolution(t *testing.T) {
	s := NewServer(newTestHandlerDistinctProject(t), Agent)
	// No explicit → the detector's chain (resolves to distinctProject, a
	// value distinct from the pre-fix "p" literal).
	out := s.CallTool(ctx, "mem_current_project", map[string]any{})
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["project"] != distinctProject {
		t.Fatalf("mem_current_project = %v, want the detector's %q (not the pre-fix literal)", env["project"], distinctProject)
	}
	// An explicit selector wins over the detector.
	out2 := s.CallTool(ctx, "mem_current_project", map[string]any{"project": "explicit-wins"})
	var env2 map[string]any
	if err := json.Unmarshal(out2, &env2); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out2)
	}
	if env2["project"] != "explicit-wins" {
		t.Fatalf("explicit project must win, got %v", env2["project"])
	}
}

// TestCurrentProjectNoDetectorIsError (fix round 1, IMPORTANT #1) pins the
// no-detector path: a Handler with no detector returns an envelope error,
// never a placeholder project name.
func TestCurrentProjectNoDetectorIsError(t *testing.T) {
	s := NewServer(newTestHandlerNoDetector(t), Agent)
	out := s.CallTool(ctx, "mem_current_project", map[string]any{})
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["error"] != true {
		t.Fatalf("no-detector mem_current_project must be an error envelope, got %s", out)
	}
	if _, ok := env["project"]; ok {
		t.Fatalf("no-detector result must not carry a project name: %s", out)
	}
	if env["code"] == "" || env["message"] == "" {
		t.Fatalf("error envelope must carry a code + message: %s", out)
	}
}

// TestCapturePassiveIsHonestNoOp (fix round 1, IMPORTANT #2) pins that
// mem_capture_passive is an honest no-op on the wire when the service is
// not wired: it carries degraded:true + a reason (the agent knows it is
// not a success), not a bare captured:0. The pinned required args are
// validated before the no-op — absent → invalid_arguments.
func TestCapturePassiveIsHonestNoOp(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	out := s.CallTool(ctx, "mem_capture_passive", map[string]any{
		"text":       "## Key Learnings:\n- x",
		"session_id": "s1",
	})
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["captured"] != float64(0) {
		t.Fatalf("capture_passive must report captured:0 (no-op), got %v", env["captured"])
	}
	if env["degraded"] != true {
		t.Fatalf("capture_passive no-op must carry degraded:true, got %v", env["degraded"])
	}
	if env["degraded_reason"] == "" {
		t.Fatalf("capture_passive no-op must carry a non-empty degraded_reason: %s", out)
	}
	// A pinned required arg absent → invalid_arguments (the no-op must
	// not swallow the call's own shape errors).
	bad := s.CallTool(ctx, "mem_capture_passive", map[string]any{"text": "## Key Learnings:\n- x"})
	var badEnv map[string]any
	if err := json.Unmarshal(bad, &badEnv); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, bad)
	}
	if badEnv["error"] != true || badEnv["code"] != "invalid_arguments" {
		t.Fatalf("capture_passive without session_id must be invalid_arguments, got %s", bad)
	}
}

// TestSessionLifecycleWired (fix round 1, IMPORTANT #2) pins the session
// tools that the Service already exposes (T07): mem_session_start
// registers the row, mem_session_end ends it (+ summary),
// mem_session_summary persists the six sections, and
// mem_session_finalize is the atomic close. None of them may be a healthy
// empty envelope.
func TestSessionLifecycleWired(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	st := s.h.Service().St()
	// start: the session row must exist after the call.
	out := s.CallTool(ctx, "mem_session_start", map[string]any{
		"id": "sess-lc", "project": testProject, "directory": "/tmp/w", "resume": true,
	})
	assertToolOK(t, out)
	rows, err := rawQuery(st, `SELECT COUNT(*) FROM sessions WHERE id = ?`, "sess-lc")
	if err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if n := firstInt(rows); n != 1 {
		t.Fatalf("mem_session_start must register the session row, got %d rows", n)
	}
	// summary: the six sections persist as a session_summary observation
	// (the spec row: "Persist the six-section structured summary"; the
	// summary must precede the end — the save pipeline 409s an ended
	// session).
	sum := s.CallTool(ctx, "mem_session_summary", map[string]any{
		"session_id": "sess-lc", "goal": "g", "instructions": "i", "discoveries": "d",
		"accomplished": "a", "next_steps": "n", "relevant_files": "f", "project": testProject,
	})
	assertToolOK(t, sum)
	rows, err = rawQuery(st, `SELECT ifnull(content, '') FROM observations WHERE session_id = ? AND type = 'session_summary'`, "sess-lc")
	if err != nil {
		t.Fatalf("read summary obs: %v", err)
	}
	countRows, err := rawQuery(st, `SELECT COUNT(*) FROM observations WHERE session_id = ? AND type = 'session_summary'`, "sess-lc")
	if err != nil {
		t.Fatalf("count summary obs: %v", err)
	}
	if n := firstInt(countRows); n != 1 {
		t.Fatalf("mem_session_summary must persist exactly one session_summary observation, got %d", n)
	}
	if s := firstString(rows, 0); s == "" || !strings.Contains(s, "## Goal") || !strings.Contains(s, "## Relevant Files") {
		t.Fatalf("mem_session_summary must carry the six sections, got %q", s)
	}
	// end: ended_at set; the free-text summary replaces the sections
	// (the end's summary is the row's final summary field).
	end := s.CallTool(ctx, "mem_session_end", map[string]any{"session_id": "sess-lc", "summary": "done for now"})
	assertToolOK(t, end)
	rows, err = rawQuery(st, `SELECT ifnull(ended_at,''), ifnull(summary,'') FROM sessions WHERE id = ?`, "sess-lc")
	if err != nil {
		t.Fatalf("read session: %v", err)
	}
	if len(rows) == 0 || firstString(rows, 0) == "" || firstString(rows, 1) != "done for now" {
		t.Fatalf("mem_session_end must set ended_at + summary, got %v", rows)
	}
	// finalize: the idempotent close (a re-finalize on an ended session is
	// a no-op success, never a 409 — the spec row's retry semantics).
	fin := s.CallTool(ctx, "mem_session_finalize", map[string]any{"session_id": "sess-lc", "project": testProject})
	assertToolOK(t, fin)
	fin2 := s.CallTool(ctx, "mem_session_finalize", map[string]any{"session_id": "sess-lc", "project": testProject})
	assertToolOK(t, fin2)
}

// TestAdminToolsReachable (fix round 1, IMPORTANT #2 + R-1) pins that the
// four admin-named tools (timeline, merge_projects, doctor, stats) exist
// in the All profile, validate their pinned required args, and never return
// a healthy empty envelope: a wired tool returns real rows, an unwired one
// returns degraded:true + reason (R-2).
func TestAdminToolsReachable(t *testing.T) {
	s := NewServer(newTestHandler(t), All)
	// stats: wired (per-type counts from the store).
	out := s.CallTool(ctx, "mem_stats", map[string]any{"project": testProject})
	assertToolOK(t, out)
	if !strings.Contains(string(out), "by_type") {
		t.Fatalf("mem_stats must return per-type statistics, got %s", out)
	}
	// timeline: an unwired no-op must be an honest degraded envelope (R-2).
	tl := s.CallTool(ctx, "mem_timeline", map[string]any{"session_id": "s1", "project": testProject})
	assertDegradedNoOp(t, tl, "mem_timeline")
	// doctor: wired (config/FTS/vector/embed checks) — always reachable.
	doc := s.CallTool(ctx, "mem_doctor", map[string]any{})
	assertToolOK(t, doc)
	if !strings.Contains(string(doc), "checks") {
		t.Fatalf("mem_doctor must return the capability checks, got %s", doc)
	}
	// merge_projects: a no-op without a service method (R-2: degraded +
	// reason, never a healthy empty envelope) + required-arg validation.
	mp := s.CallTool(ctx, "mem_merge_projects", map[string]any{"source": "a", "target": "b"})
	assertDegradedNoOp(t, mp, "mem_merge_projects")
	bad := s.CallTool(ctx, "mem_merge_projects", map[string]any{"source": "a"})
	var badEnv map[string]any
	if err := json.Unmarshal(bad, &badEnv); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, bad)
	}
	if badEnv["code"] != "invalid_arguments" {
		t.Fatalf("merge_projects without target must be invalid_arguments, got %s", bad)
	}
}

// TestListProjectsWired (fix round 1, IMPORTANT #2) pins mem_list_projects:
// a wired read (distinct projects from the store), agent-profile (spec §3).
func TestListProjectsWired(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	// Seed a row in a project the store does not open for (list_projects
	// spans the tables, not the open project): the result must reflect the
	// store's actual rows, not a hardcoded fixture.
	st := s.h.Service().St()
	if err := rawExec(st, `INSERT INTO sessions (id, project, directory, ownership_mode, started_at, created_at)
		VALUES ('sess-lp', 'other-project', '/tmp/w', 'shared', datetime('now'), datetime('now'))`); err != nil {
		t.Fatalf("seed other project: %v", err)
	}
	out := s.CallTool(ctx, "mem_list_projects", map[string]any{})
	assertToolOK(t, out)
	if !strings.Contains(string(out), testProject) {
		t.Fatalf("mem_list_projects must list the store's projects (incl. %q), got %s", testProject, out)
	}
	if !strings.Contains(string(out), "other-project") {
		t.Fatalf("mem_list_projects must list every stored project (incl. the seeded %q), got %s", "other-project", out)
	}
}

// TestGetObservationSchemaDropsHistoryCursor (fix round 1, R-4) pins the
// parked cursor: mem_get_observation's schema carries no history_cursor
// until the store's version paging exists (the handler signature keeps it).
func TestGetObservationSchemaDropsHistoryCursor(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	sch := s.ToolSchema("mem_get_observation")
	if sch.Props["history_cursor"] {
		t.Fatalf("mem_get_observation must not expose history_cursor until store version paging exists (R-4)")
	}
	if !sch.Props["include_history"] {
		t.Fatal("mem_get_observation must keep include_history")
	}
	// The dispatch still tolerates an incoming cursor (forward-compat) —
	// it just ignores it.
	got := s.CallTool(ctx, "mem_get_observation", map[string]any{"sync_id": "does-not-exist"})
	var env map[string]any
	if err := json.Unmarshal(got, &env); err != nil {
		t.Fatalf("not JSON: %v (%s)", err, got)
	}
	if env["error"] != true {
		t.Fatalf("an unknown sync_id must be an error envelope: %s", got)
	}
}

// ── helpers (fix round 1) ──────────────────────────────────────────────────

// assertToolOK pins a tool result is not an error envelope.
func assertToolOK(t *testing.T, out []byte) {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["error"] == true {
		t.Fatalf("expected a tool result, got an error envelope: %s", out)
	}
}

// assertDegradedNoOp pins an unwired tool's R-2 envelope: a non-error result
// carrying degraded:true + a non-empty reason (never a healthy empty map).
func assertDegradedNoOp(t *testing.T, out []byte, tool string) {
	t.Helper()
	var env map[string]any
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("envelope is not JSON: %v (%s)", err, out)
	}
	if env["error"] == true {
		t.Fatalf("%s must be a (degraded) result, not an error envelope: %s", tool, out)
	}
	if env["degraded"] != true {
		t.Fatalf("%s no-op must carry degraded:true (R-2), got %s", tool, out)
	}
	if env["degraded_reason"] == "" {
		t.Fatalf("%s no-op must carry a non-empty degraded_reason (R-2): %s", tool, out)
	}
}
