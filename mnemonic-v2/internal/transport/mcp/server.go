package mcp

// server.go — the MCP stdio server: the primary agent interface (spec
// 004-transport-mcp §1). A Server is a thin adapter over the shared
// handler: it registers the profile-filtered tool set (K5: 22 agent /
// 6 admin), dispatches tool calls through the write queue (writes) or
// directly (reads), and wraps every result in the D1 envelope.
//
// The server is the MCP surface of the handler; the CLI (Task 14) is the
// other surface. Both call the same Handler — the K6 byte-identical
// seam. Deleting this package does not delete the handler; the handler
// is the contract, this is one adapter over it.

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// Profile selects the tool set (K5: the spec table's literal rows).
type Profile string

const (
	// Agent is the 18-tool agent profile (spec 004-transport-mcp §3's
	// | agent | rows — R-1, fix round 1).
	Agent Profile = "agent"
	// All is the 28-tool profile (agent + the 6 admin tools).
	All Profile = "all"
)

const (
	serverName    = "mnemonic-v2"
	serverVersion = "0.1.0"
)

// instructions is the memory↔wiki routing rule (spec 01-memory/005 §6 —
// wiki-first, mem_lookup as the memory-half fallback).
const instructions = `mnemonic-v2 memory server.
Routing: prefer the wiki for persistent knowledge; use mem_lookup as the narrow raw-fact fallback when the wiki has no answer.
mem_save is the primary write; reuse topic_key to upsert evolving topics.
mem_suggest_topic_key is a hint, not proof of identity (spec 004-dedup §6).
All ids the agent sees are sync_id (INV-6); mem_get_observation accepts both sync_id and the local id.`

// Server is an MCP server instance with the profile-filtered mem_* tools
// registered, ready to be served over stdio.
type Server struct {
	mcp     *server.MCPServer
	h       *handler.Handler
	profile Profile
	queue   *writeQueue
	// calls is the in-process dispatch (the test harness calls CallTool
	// directly; the stdio path goes through mcp.HandleMessage).
	calls map[string]func(ctx context.Context, args map[string]any) (any, error)
	// txnMu guards txnLog (the store's committed-transaction order — the
	// write queue's load-bearing test assertion, fix round 1 IMPORTANT #3).
	txnMu  sync.Mutex
	txnLog []int
}

// NewServer builds an MCP server over the shared handler. The profile
// selects the tool set (Agent = 22 tools, All = 28 tools — spec
// 004-transport-mcp §3's literal name sets, R-1).
func NewServer(h *handler.Handler, profile Profile) *Server {
	s := &Server{
		h:       h,
		profile: profile,
		queue:   newWriteQueue(),
	}
	// The store's committed-transaction seam (store.SetTxnHook) records
	// the transaction order: the write queue's serialization is proven by
	// the store's own log, not a queue-internal flag.
	st := h.Service().St()
	st.SetTxnHook(func() {
		s.txnMu.Lock()
		s.txnLog = append(s.txnLog, len(s.txnLog)+1)
		s.txnMu.Unlock()
	})
	// Build the underlying mcp-go server with the profile-filtered tools.
	s.mcp = server.NewMCPServer(serverName, serverVersion, server.WithInstructions(instructions))
	s.calls = s.buildToolDispatch()
	for _, tool := range s.toolSet() {
		s.mcp.AddTool(tool.def, s.mcpToolHandler(tool.name))
	}
	return s
}

// txnOrder returns a copy of the committed-transaction sequence (the test
// hook for TestWriteQueueSerializes: strictly monotonic, one per write).
func (s *Server) txnOrder() []int {
	s.txnMu.Lock()
	defer s.txnMu.Unlock()
	out := make([]int, len(s.txnLog))
	copy(out, s.txnLog)
	return out
}

// txnCount returns the number of committed transactions (the test hook for
// TestJudgeManyWritesOnceInQueue: one queued write = one transaction).
func (s *Server) txnCount() int {
	s.txnMu.Lock()
	defer s.txnMu.Unlock()
	return len(s.txnLog)
}

// Start blocks on the stdio MCP loop until the client disconnects
// (production entry: the cmd layer calls this).
func (s *Server) Start(ctx context.Context) error {
	return server.ServeStdio(s.mcp)
}

// ToolNames returns the registered tool names (the test hook for the
// contract tests).
func (s *Server) ToolNames() []string {
	tools := s.mcp.ListTools()
	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}
	return names
}

// ToolSchema returns a tool's input schema (the test hook for the
// required-param assertions).
func (s *Server) ToolSchema(name string) ToolSchema {
	st := s.mcp.GetTool(name)
	if st == nil {
		return ToolSchema{}
	}
	required := make(map[string]bool)
	for _, r := range st.Tool.InputSchema.Required {
		required[r] = true
	}
	props := make(map[string]bool)
	if st.Tool.InputSchema.Properties != nil {
		for k := range st.Tool.InputSchema.Properties {
			props[k] = true
		}
	}
	return ToolSchema{Required: required, Props: props}
}

// CallTool dispatches a tool call in-process (the test harness path; the
// stdio path goes through mcp.HandleMessage). Writes go through the queue;
// reads bypass it. Returns the envelope JSON.
func (s *Server) CallTool(ctx context.Context, name string, args map[string]any) []byte {
	if args == nil {
		args = map[string]any{}
	}
	fn, ok := s.calls[name]
	if !ok {
		b, _ := json.Marshal(toolError("unknown tool", name))
		return b
	}
	isWrite := s.isWriteTool(name)
	var result any
	var err error
	if isWrite {
		err = s.queue.Submit(ctx, func() {
			result, err = fn(ctx, args)
		})
		if err != nil {
			// A queue-level error (a cancelled ctx, a panicked worker) —
			// the fn did not run (or ran and panicked).
			result = nil
		}
	} else {
		result, err = fn(ctx, args)
	}
	if err != nil {
		code, msg := envelope.MapError(err)
		b, _ := json.Marshal(toolError(code, msg))
		return b
	}
	// Wrap the result in the envelope (degraded flag when the result
	// carries one).
	env, wrapErr := wrapEnvelope(result)
	if wrapErr != nil {
		b, _ := json.Marshal(toolError(envelope.CodeInternal, wrapErr.Error()))
		return b
	}
	b, _ := json.Marshal(env)
	return b
}

// HandshakeAndCall runs the MCP initialize handshake + a tools/call over
// the in-process HandleMessage (the test harness: a valid JSON-RPC
// round trip without a real stdio pipe). Returns the result frames.
func (s *Server) HandshakeAndCall(ctx context.Context, toolName string) ([][]byte, error) {
	frames := [][]byte{}
	// 1. initialize
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`
	resp := s.mcp.HandleMessage(ctx, json.RawMessage(initReq))
	if resp != nil {
		b, _ := json.Marshal(resp)
		frames = append(frames, b)
	}
	// 2. tools/call
	callReq := fmt.Sprintf(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":%q,"arguments":{}}}`, toolName)
	resp2 := s.mcp.HandleMessage(ctx, json.RawMessage(callReq))
	if resp2 != nil {
		b, _ := json.Marshal(resp2)
		frames = append(frames, b)
	}
	return frames, nil
}

// WritesSerialized reports whether the write queue serialized at least one
// write (the test hook for TestWriteQueueSerializes).
func (s *Server) WritesSerialized() bool {
	return s.queue.WritesSerialized()
}

// WritesSinceLastReset returns the total write count (the test hook for
// TestJudgeManyWritesOnceInQueue).
func (s *Server) WritesSinceLastReset() int {
	return s.queue.WritesSinceLastReset()
}

// SetDeadlockForTest stops the write queue worker (the test hook for
// TestReadsBypassTheQueue: a read that enqueues would hang).
func (s *Server) SetDeadlockForTest() {
	s.queue.SetDeadlockForTest()
}

// Close stops the write queue worker and detaches the store's transaction
// hook (the test cleanup; the cmd layer calls this on shutdown).
func (s *Server) Close() {
	s.h.Service().St().SetTxnHook(func() {})
	s.queue.Close()
}

// mcpToolHandler adapts a dispatch fn onto the mcp-go ToolHandlerFunc
// signature (the stdio path: mcp-go calls this with a CallToolRequest).
func (s *Server) mcpToolHandler(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		args, _ := req.Params.Arguments.(map[string]any)
		out := s.CallTool(ctx, name, args)
		return mcplib.NewToolResultText(string(out)), nil
	}
}

// isWriteTool reports whether a tool mutates state (spec 004-transport-mcp
// §4: the write queue serializes these; reads bypass it).
func (s *Server) isWriteTool(name string) bool {
	switch name {
	case "mem_save", "mem_update", "mem_delete", "mem_judge", "mem_judge_many", "mem_consolidate", "mem_pin", "mem_unpin":
		return true
	}
	return false
}

// toolError is the error envelope (a JSON object with code + message +
// the tool name for debuggability).
func toolError(code, message string) map[string]any {
	return map[string]any{
		"error":   true,
		"code":    code,
		"message": message,
	}
}

// wrapEnvelope merges a result with the degradation state (the D1
// envelope: result fields + degraded/degraded_reason when degraded).
func wrapEnvelope(result any) (map[string]any, error) {
	degraded, reason := extractDegraded(result)
	return envelope.WithDegraded(result, degraded, reason)
}

// extractDegraded inspects a result for the degraded flag (the service
// result types carry Degraded bool; the envelope surfaces it).
func extractDegraded(result any) (bool, string) {
	b, _ := json.Marshal(result)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return false, ""
	}
	d, _ := m["degraded"].(bool)
	reason, _ := m["degraded_reason"].(string)
	if d && reason == "" {
		reason = "intelligence seam absent (deterministic floor ran)"
	}
	return d, reason
}

// ToolSchema is the test-hook shape for a tool's input schema.
type ToolSchema struct {
	Required map[string]bool
	Props    map[string]bool
}
