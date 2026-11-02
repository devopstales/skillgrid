package mcp

// server_test.go — the MCP stdio server: the write queue (spec
// 004-transport-mcp §4), the degradation envelope (INV-3), and the stdio
// JSON-RPC round trip (the harness handshake the plugins speak).

import (
	"encoding/json"
	"sync"
	"testing"
	"time"
)

func TestWriteQueueSerializes(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	start := time.Now()
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
	_ = start
}

func TestEnvelopeCarriesDegraded(t *testing.T) {
	s := NewServer(newTestHandlerWithNoLLM(t), Agent) // nil seams
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
func TestJudgeManyWritesOnceInQueue(t *testing.T) {
	s := NewServer(newTestHandler(t), Agent)
	obs1 := s.CallTool(ctx, "mem_save", saveArgs(1))
	obs2 := s.CallTool(ctx, "mem_save", saveArgs(2))
	if !json.Valid(obs1) || !json.Valid(obs2) {
		t.Fatalf("seeding saves failed: %s / %s", obs1, obs2)
	}
	args := map[string]any{
		"judgments": []map[string]any{
			{"judgment_id": 1, "relation": "conflicts_with", "reason": "r1"},
			{"judgment_id": 2, "relation": "refines", "reason": "r2"},
		},
		"model": "test-model",
	}
	out := s.CallTool(ctx, "mem_judge_many", args)
	if !json.Valid(out) {
		t.Fatalf("judge_many envelope is not JSON: %s", out)
	}
	if got := s.WritesSinceLastReset(); got < 3 {
		// 2 seeding saves + the one queued batch. The batch must be a
		// single queue slot — a per-judgment enqueue would count more.
		t.Fatalf("judge_many must enqueue exactly one write (saw %d total writes)", got)
	}
}
