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
	// codeTools is the code-index handler (B3 — the registration seam):
	// nil until RegisterCodeTools is called (the memory-only build still
	// serves without it). The dispatch consults it for the code_* tools.
	codeTools *handler.CodeHandler
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

// RegisterCodeTools adds the code_* tool family to the server (B3 — the
// registration seam, spec 008-transport §1): it extends the dispatch with
// the profile-filtered code_* tools + the isWriteTool set (code_index is a
// queued write) + the combined instructions. It is ADDITIVE — the memory-only
// build still serves without it (codeTools is nil until this is called). The
// profile (Agent/All) selects which code_* tools register (the two-profile
// mechanism, D-PROFILE — not v1's SKILLGRID_MCP_CODE_TOOLS env filter).
func (s *Server) RegisterCodeTools(ch *handler.CodeHandler) {
	s.codeTools = ch
	// Extend the in-process dispatch with the code_* tools (the profile
	// filters the set — the agent profile drops the admin tools).
	for name, fn := range s.codeToolDispatch() {
		s.calls[name] = fn
	}
	// Register the profile-filtered code_* tool definitions with mcp-go
	// (the stdio path: the tools/list response carries them).
	for _, tool := range s.codeToolSet() {
		s.mcp.AddTool(tool.def, s.mcpToolHandler(tool.name))
	}
}

// codeToolDispatch returns the code_* tool name → dispatch fn map (the
// profile filters the set — the agent profile drops the admin tools). The
// dispatch fns are the MCP adapter's half of the K6 seam: the CLI adapter
// (cmd/mnemonic/index.go) has its own, over the same CodeHandler.
func (s *Server) codeToolDispatch() map[string]func(ctx context.Context, args map[string]any) (any, error) {
	ch := s.codeTools
	if ch == nil {
		return nil
	}
	d := map[string]func(ctx context.Context, args map[string]any) (any, error){
		"code_status":           dispatchCodeStatus(ch),
		"code_index":            dispatchCodeIndex(ch),
		"code_search":           dispatchCodeSearch(ch),
		"code_read":             dispatchCodeRead(ch),
		"code_orient":           dispatchCodeOrient(ch),
		"code_explain":          dispatchCodeExplain(ch),
		"code_path":             dispatchCodePath(ch),
		"code_impact":           dispatchCodeImpact(ch),
		"code_affected":         dispatchCodeAffected(ch),
		"code_grep":             dispatchCodeGrep(ch),
		"code_semantic":         dispatchCodeSemantic(ch),
		"code_get_callers":      dispatchCodeNeighbors(ch, "callers"),
		"code_get_callees":      dispatchCodeNeighbors(ch, "callees"),
		"code_get_dependents":   dispatchCodeNeighbors(ch, "dependents"),
		"code_get_implementors": dispatchCodeNeighbors(ch, "implementors"),
		"code_get_hierarchy":    dispatchCodeNeighbors(ch, "hierarchy"),
		"code_get_tests_for":    dispatchCodeNeighbors(ch, "tests_for"),
		"code_resolve":          dispatchCodeResolve(ch),
		"code_packages":         dispatchCodePackages(ch),
		// admin (All profile only)
		"code_rename":           dispatchCodeRename(ch),
		"code_unresolved":       dispatchCodeUnresolved(ch),
		"code_circles":          dispatchCodeCircles(ch),
		"code_deps":             dispatchCodeDeps(ch),
		"code_dep_impact":       dispatchCodeDepImpact(ch),
		"code_embedding_status": dispatchCodeEmbeddingStatus(ch),
		"code_watch":            dispatchCodeWatch(ch),
		// v2.1-deferred (declared, reporting "run code_index --flag")
		"code_pdg_query":         dispatchCodeDeferred(ch, "code_pdg_query", "pdg"),
		"code_taint":             dispatchCodeDeferred(ch, "code_taint", "pdg"),
		"code_communities":       dispatchCodeDeferred(ch, "code_communities", "communities"),
		"code_god_nodes":         dispatchCodeDeferred(ch, "code_god_nodes", "communities"),
		"code_explain_community": dispatchCodeDeferred(ch, "code_explain_community", "communities"),
		"code_processes":         dispatchCodeDeferred(ch, "code_processes", "process"),
		"code_process":           dispatchCodeDeferred(ch, "code_process", "process"),
		// knowledge-graph reads (the knowledge nodes in the core graph)
		"code_route":      dispatchCodeRoute(ch),
		"code_navigates":  dispatchCodeNavigates(ch),
		"code_docs":       dispatchCodeDocs(ch),
		"code_configs":    dispatchCodeConfigs(ch),
		"code_sql_schema": dispatchCodeSQLSchema(ch),
	}
	return d
}

// codeToolSet returns the profile-filtered code_* tool definitions (the
// agent profile drops the admin tools — the two-profile mechanism,
// D-PROFILE). The tool definitions are the D1 contract: the plugins' only
// interface into the code-index.
func (s *Server) codeToolSet() []toolEntry {
	ch := s.codeTools
	if ch == nil {
		return nil
	}
	entries := []toolEntry{
		{name: "code_status", def: codeStatusTool(), isWrite: false},
		{name: "code_index", def: codeIndexTool(), isWrite: true},
		{name: "code_search", def: codeSearchTool(), isWrite: false},
		{name: "code_read", def: codeReadTool(), isWrite: false},
		{name: "code_orient", def: codeOrientTool(), isWrite: false},
		{name: "code_explain", def: codeExplainTool(), isWrite: false},
		{name: "code_path", def: codePathTool(), isWrite: false},
		{name: "code_impact", def: codeImpactTool(), isWrite: false},
		{name: "code_affected", def: codeAffectedTool(), isWrite: false},
		{name: "code_grep", def: codeGrepTool(), isWrite: false},
		{name: "code_semantic", def: codeSemanticTool(), isWrite: false},
		{name: "code_get_callers", def: codeNeighborTool("code_get_callers", "callers"), isWrite: false},
		{name: "code_get_callees", def: codeNeighborTool("code_get_callees", "callees"), isWrite: false},
		{name: "code_get_dependents", def: codeNeighborTool("code_get_dependents", "dependents"), isWrite: false},
		{name: "code_get_implementors", def: codeNeighborTool("code_get_implementors", "implementors"), isWrite: false},
		{name: "code_get_hierarchy", def: codeNeighborTool("code_get_hierarchy", "hierarchy"), isWrite: false},
		{name: "code_get_tests_for", def: codeNeighborTool("code_get_tests_for", "tests_for"), isWrite: false},
		{name: "code_resolve", def: codeResolveTool(), isWrite: false},
		{name: "code_packages", def: codePackagesTool(), isWrite: false},
		// admin (All profile only)
		{name: "code_rename", def: codeRenameTool(), isWrite: true},
		{name: "code_unresolved", def: codeUnresolvedTool(), isWrite: false},
		{name: "code_circles", def: codeCirclesTool(), isWrite: false},
		{name: "code_deps", def: codeDepsTool(), isWrite: false},
		{name: "code_dep_impact", def: codeDepImpactTool(), isWrite: false},
		{name: "code_embedding_status", def: codeEmbeddingStatusTool(), isWrite: false},
		{name: "code_watch", def: codeWatchTool(), isWrite: true},
		// v2.1-deferred (declared in the surface, not missing)
		{name: "code_pdg_query", def: codeDeferredTool("code_pdg_query", "pdg"), isWrite: false},
		{name: "code_taint", def: codeDeferredTool("code_taint", "pdg"), isWrite: false},
		{name: "code_communities", def: codeDeferredTool("code_communities", "communities"), isWrite: false},
		{name: "code_god_nodes", def: codeDeferredTool("code_god_nodes", "communities"), isWrite: false},
		{name: "code_explain_community", def: codeDeferredTool("code_explain_community", "communities"), isWrite: false},
		{name: "code_processes", def: codeDeferredTool("code_processes", "process"), isWrite: false},
		{name: "code_process", def: codeDeferredTool("code_process", "process"), isWrite: false},
		// knowledge-graph reads
		{name: "code_route", def: codeRouteTool(), isWrite: false},
		{name: "code_navigates", def: codeNavigatesTool(), isWrite: false},
		{name: "code_docs", def: codeDocsTool(), isWrite: false},
		{name: "code_configs", def: codeConfigsTool(), isWrite: false},
		{name: "code_sql_schema", def: codeSQLSchemaTool(), isWrite: false},
	}
	if s.profile == Agent {
		// Filter to agent tools (drop the admin + deferred + knowledge reads).
		admin := map[string]bool{}
		for _, n := range codeAdminOnlyTools {
			admin[n] = true
		}
		out := make([]toolEntry, 0, len(entries))
		for _, e := range entries {
			if !admin[e.name] {
				out = append(out, e)
			}
		}
		return out
	}
	return entries
}

// codeAdminOnlyTools is the admin-only code_* tool names (the agent profile
// drops them — the two-profile mechanism, D-PROFILE). The v2.1-deferred +
// the knowledge-graph reads are admin-only too (they are not in the 16
// agent tools).
var codeAdminOnlyTools = []string{
	"code_rename", "code_unresolved", "code_circles", "code_deps",
	"code_dep_impact", "code_embedding_status", "code_watch",
	"code_pdg_query", "code_taint", "code_communities", "code_god_nodes",
	"code_explain_community", "code_processes", "code_process",
	"code_route", "code_navigates", "code_docs", "code_configs",
	"code_sql_schema",
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
// hook (the test cleanup; the cmd layer calls this on shutdown). It also
// stops the code watcher (code_watch's live watcher, if started).
func (s *Server) Close() {
	s.h.Service().St().SetTxnHook(func() {})
	s.queue.Close()
	if s.codeTools != nil {
		s.codeTools.StopWatchForTest()
	}
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
// §4: the write queue serializes these; reads bypass it). The code_* writes
// (code_index — the queued write with the active-run guard, code_rename,
// code_watch) are added by the code-index change (B3 — additive to the
// memory set).
func (s *Server) isWriteTool(name string) bool {
	switch name {
	case "mem_save", "mem_update", "mem_delete", "mem_judge", "mem_judge_many", "mem_consolidate", "mem_pin", "mem_unpin":
		return true
	case "code_index", "code_rename", "code_watch":
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
// envelope: result fields + degraded/degraded_reason when degraded). The
// degradation extraction is envelope.ExtractDegraded (the shared K6 seam —
// the same call the CLI adapter makes, so the flag never diverges).
func wrapEnvelope(result any) (map[string]any, error) {
	degraded, reason := envelope.ExtractDegraded(result)
	return envelope.WithDegraded(result, degraded, reason)
}

// ToolSchema is the test-hook shape for a tool's input schema.
type ToolSchema struct {
	Required map[string]bool
	Props    map[string]bool
}
