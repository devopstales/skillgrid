package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"github.com/devopstales/skillgrid/mnemonic/internal/facts"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

const (
	serverName    = "skillgrid-mnemonic"
	serverVersion = "0.1.0"
)

// svc is the injected service used by all tool handlers. It defaults to the
// data directory from SKILLGRID_MNEMONIC_DATA_DIR / ~/.skillgrid/mnemonic.
var svc *service.Service

// pool is the lazy per-project store pool shared by the MCP server. It is
// initialized on the first query; tests may replace it.
var pool *StorePool

// excludeEnvVar is the Hermes integration seam (2026-10-24-hermes-
// integration, TICKET-02): a comma-separated list of tool name prefixes.
// The Hermes MCP registration sets it to "mem" because the MemoryProvider
// handles the mem_* tools natively; the MCP server hides them so no tool
// name is exposed by both paths.
const excludeEnvVar = "MNEMONIC_MCP_EXCLUDE"

// serverOptions returns the MCP server options: the initialize-injected
// guidance (steer to the composite primary tool, don't re-grep), the
// tool filter that hides the narrow code_* menu tools by default, and
// the unknown-tool suggestion hook (did-you-mean for hallucinated names).
func serverOptions() []server.ServerOption {
	return []server.ServerOption{
		server.WithInstructions(exploreInitializeGuidance()),
		server.WithToolFilter(applyCodeToolFilter()),
		server.WithToolFilter(applyExcludeToolFilter()),
	}
}

// applyExcludeToolFilter returns a ToolFilterFunc that hides every tool
// whose name starts with a prefix listed in MNEMONIC_MCP_EXCLUDE. Empty or
// unset (the default) means no exclusion — the filter passes tools through
// unchanged. Never returns nil (the mcp-go library does not tolerate nil
// filters in its chain).
func applyExcludeToolFilter() server.ToolFilterFunc {
	return func(_ context.Context, tools []mcplib.Tool) []mcplib.Tool {
		prefixes := excludeToolPrefixes()
		if len(prefixes) == 0 {
			return tools
		}
		out := make([]mcplib.Tool, 0, len(tools))
		for _, t := range tools {
			if excludedTool(t.Name, prefixes) {
				continue
			}
			out = append(out, t)
		}
		return out
	}
}

// excludeToolPrefixes parses MNEMONIC_MCP_EXCLUDE into a trimmed,
// de-duplicated prefix list ("" and whitespace entries dropped).
func excludeToolPrefixes() []string {
	v := strings.TrimSpace(os.Getenv(excludeEnvVar))
	if v == "" {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range strings.Split(v, ",") {
		p = strings.TrimSpace(p)
		if p != "" && !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return out
}

// excludedTool reports whether name starts with any of the prefixes.
func excludedTool(name string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

// applyUnknownToolSuggestion wraps a tool handler so that when the MCP
// library's handleToolCall returns a "tool not found" error for a name
// that is close to a registered tool, the error message is augmented
// with a "did you mean X?" suggestion. This prevents agents from
// hallucinating tool names (e.g. code_scan) and having no recovery
// hint.
func applyUnknownToolSuggestion(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcplib.CallToolRequest) (*mcplib.CallToolResult, error) {
		res, err := next(ctx, req)
		if err == nil {
			return res, nil
		}
		msg := err.Error()
		if !strings.Contains(msg, "not found") {
			return res, err
		}
		// Extract the requested tool name from the error message.
		name := extractToolName(msg)
		if name == "" {
			return res, err
		}
		s := server.ServerFromContext(ctx)
		if s == nil {
			return res, err
		}
		tools := s.ListTools()
		if tools == nil {
			return res, err
		}
		names := make([]string, 0, len(tools))
		for n := range tools {
			names = append(names, n)
		}
		if hint := toolSuggestion(name, names); hint != "" {
			return res, fmt.Errorf("%s%s", msg, hint)
		}
		return res, err
	}
}

// extractToolName pulls the tool name from a "tool 'X' not found" error message.
func extractToolName(msg string) string {
	// Format: "tool 'code_scan' not found: ..."
	start := strings.Index(msg, "'")
	if start < 0 {
		return ""
	}
	rest := msg[start+1:]
	end := strings.Index(rest, "'")
	if end < 0 {
		return ""
	}
	return rest[:end]
}

// ensurePool lazily creates the shared store pool on first use.
func ensurePool() *StorePool {
	if pool == nil {
		d, _ := service.DefaultDataDir()
		pool = NewStorePool(d, 0, 0)
	}
	return pool
}

// Start blocks on the stdio MCP loop until the client disconnects.
func Start() error {
	s := NewServer()
	return server.ServeStdio(s)
}

// Server is an MCP server instance with all mem_*/code_*/web_*/team_* tools
// registered, ready to be served over stdio.
type Server = server.MCPServer

// NewServer returns an MCP server instance with all tools registered.
func NewServer() *Server {
	s := server.NewMCPServer(serverName, serverVersion, serverOptions()...)
	s.Use(applyUnknownToolSuggestion)
	registerMemoryTools(s)
	registerCodeTools(s)
	registerOrientTools(s)
	registerGrepTools(s)
	registerGraphTools(s)
	registerExploreTools(s)
	registerHybridTools(s)
	registerCommunityTools(s)
	registerRouteTools(s)
	registerUnresolvedTools(s)
	registerProcessTools(s)
	registerKnowledgeTools(s)
	registerWebTools(s)
	registerTeamsTools(s)
	registerRetrievalTools(s)
	registerCompactionTools(s)
	registerAffectedTools(s)
	registerPdgTools(s)
	registerTaintTools(s)
	registerMemoryGovernanceTools(s)
	registerMemoryLayerTools(s)
	registerSessionTools(s)
	registerSessionInjectTools(s)
	registerQueryEventTools(s)
	registerFactTools(s)
	registerSkillTools(s)
	registerMemoryHybridTools(s)
	registerSecondBrainTools(s)
	registerScanTools(s)
	registerDepTools(s)
	registerMemoryGlobalResource(s)
	return s
}

// memoryGlobalResourceURI is the MCP resource that serves the frozen global
// memory snapshot (change 2026-10-24-global-llm-wiki, TICKET-05): clients
// read memory/global to get the same block the OpenCode plugin injects.
const memoryGlobalResourceURI = "memory/global"

// registerMemoryGlobalResource registers the memory/global MCP resource.
// The handler opens the global store on each read and renders the
// snapshot at read time (the frozen per-session cache is the OpenCode
// plugin's job, not the resource's). Fail-open: any error returns empty
// text content rather than a protocol error — the injection must never
// break the client.
func registerMemoryGlobalResource(s *server.MCPServer) {
	s.AddResource(mcplib.NewResource(
		memoryGlobalResourceURI,
		"Global memory snapshot",
		mcplib.WithResourceDescription(
			"The frozen snapshot of the global store fact store: GLOBAL MEMORY + USER PROFILE sections, § separators, usage headers, sorted by importance. Fail-open: empty when the store cannot be read."),
		mcplib.WithMIMEType("text/markdown"),
	), func(ctx context.Context, req mcplib.ReadResourceRequest) ([]mcplib.ResourceContents, error) {
		block, err := readMemoryGlobalResource(ctx)
		return []mcplib.ResourceContents{mcplib.TextResourceContents{
			URI:      memoryGlobalResourceURI,
			MIMEType: "text/markdown",
			Text:     block,
		}}, err
	})
}

// readMemoryGlobalResource renders the global snapshot, fail-open: any error
// (store open, query) yields an empty string.
func readMemoryGlobalResource(ctx context.Context) (string, error) {
	s, err := rootService()
	if err != nil {
		return "", nil // fail-open
	}
	gh, cleanup, err := s.OpenGlobal()
	if err != nil {
		return "", nil // fail-open
	}
	defer cleanup()
	st := facts.New(gh.Store().DB, service.GlobalProjectID)
	block, err := facts.RenderFrozenSnapshot(ctx, st)
	if err != nil {
		return "", nil // fail-open
	}
	return block, nil
}
