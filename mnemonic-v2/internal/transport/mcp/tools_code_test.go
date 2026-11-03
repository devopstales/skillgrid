package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/extract"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/graph"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/watch"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// newTestCodeHandler builds a CodeHandler over a real SQLite store with the
// production wiring (the same store the test fixture opens).
func newTestCodeHandler(t *testing.T) *handler.CodeHandler {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	return handler.NewCodeHandler(st, nil, c, "", &detectorStub{resolve: func(cwd, explicit string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		return testProject, nil
	}})
}

// indexFixtureTree indexes a small Go tree into the store (the fixture the
// code_* read tests query against).
func indexFixtureTree(t *testing.T, st store.Store, root string) {
	t.Helper()
	if _, err := codeindex.Index(ctx, root, st, codeindex.IndexOpts{Project: testProject}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
}

// codeAgentTools is the agent-profile code_* tool names (spec 008 §3 — the
// 18 distinct tool names covering the spec's 16 logical tools; the six
// neighbor views are listed individually).
var codeAgentTools = []string{
	"code_status", "code_index", "code_search", "code_read",
	"code_orient", "code_explain", "code_path", "code_impact",
	"code_affected", "code_grep", "code_semantic",
	"code_get_callers", "code_get_callees", "code_get_dependents",
	"code_get_implementors", "code_get_hierarchy", "code_get_tests_for",
	"code_resolve", "code_packages",
}

// codeAdminTools is the admin-only code_* tool names (spec 008 §3).
var codeAdminTools = []string{
	"code_rename", "code_unresolved", "code_circles", "code_deps",
	"code_dep_impact", "code_embedding_status", "code_watch",
	"code_pdg_query", "code_taint", "code_communities", "code_god_nodes",
	"code_explain_community", "code_processes", "code_process",
	"code_route", "code_navigates", "code_docs", "code_configs",
	"code_sql_schema",
}

// TestCodeToolContract verifies the D1 contract: every agent tool name is
// present with its required params; code_resolve returns the pin-check shape;
// code_packages returns the package map.
func TestCodeToolContract(t *testing.T) {
	srv := NewServer(newTestHandler(t), Agent)
	defer srv.Close()
	srv.RegisterCodeTools(newTestCodeHandler(t))

	names := srv.ToolNames()
	for _, want := range codeAgentTools {
		if !hasTool(srv, want) {
			t.Fatalf("agent tool %q missing from registered tools: %v", want, names)
		}
	}

	// code_search requires query.
	if !srv.ToolSchema("code_search").Required["query"] {
		t.Fatalf("code_search must require query")
	}
	// code_read requires path.
	if !srv.ToolSchema("code_read").Required["path"] {
		t.Fatalf("code_read must require path")
	}

	// code_resolve returns the pin-check shape (exists, called_from, candidates).
	ch := newTestCodeHandler(t)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	indexFixtureTree(t, ch.Store(), root)
	ch.SetRootForTest(root)
	out := srv.CallTool(ctx, "code_resolve", map[string]any{"symbol": "main"})
	res := mustMap(t, out)
	for _, key := range []string{"exists", "called_from", "candidates"} {
		if _, ok := res[key]; !ok {
			t.Fatalf("code_resolve must return %q, got keys: %v", key, keys(res))
		}
	}

	// code_packages returns the package map (path, files, symbols).
	out = srv.CallTool(ctx, "code_packages", map[string]any{})
	res = mustMap(t, out)
	if _, ok := res["packages"]; !ok {
		t.Fatalf("code_packages must return the package map under 'packages', got keys: %v", keys(res))
	}
}

// TestCodeProfileMembership verifies the two-profile mechanism: all is a
// superset of agent; admin-only tools like code_rename appear in all but not
// in agent.
func TestCodeProfileMembership(t *testing.T) {
	agentSrv := NewServer(newTestHandler(t), Agent)
	defer agentSrv.Close()
	agentSrv.RegisterCodeTools(newTestCodeHandler(t))

	allSrv := NewServer(newTestHandler(t), All)
	defer allSrv.Close()
	allSrv.RegisterCodeTools(newTestCodeHandler(t))

	agentNames := nameSet(agentSrv)
	allNames := nameSet(allSrv)

	// all is a superset of agent (every agent tool is in all).
	for n := range agentNames {
		if !allNames[n] {
			t.Fatalf("agent tool %q must be in the all profile (superset)", n)
		}
	}

	// code_rename (admin-only) is in all but not in agent.
	if !allNames["code_rename"] {
		t.Fatalf("code_rename must be in the all profile")
	}
	if agentNames["code_rename"] {
		t.Fatalf("code_rename must NOT be in the agent profile")
	}
	// Every declared admin tool is in all.
	for _, want := range codeAdminTools {
		if !allNames[want] {
			t.Fatalf("admin tool %q must be in the all profile", want)
		}
	}
}

// TestCodeStatusReportsVersionTierCoverage verifies the T-CI-4 fix:
// code_status reports the 4-state freshness (not a boolean), the per-language
// extractor tier, the schema version, and fair coverage.
func TestCodeStatusReportsVersionTierCoverage(t *testing.T) {
	ch := newTestCodeHandler(t)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	indexFixtureTree(t, ch.Store(), root)
	ch.SetRootForTest(root)

	out, err := ch.Status(ctx, "")
	if err != nil {
		t.Fatalf("code_status: %v", err)
	}
	if out.Freshness == "" {
		t.Fatalf("code_status must report a non-empty freshness (4-state value, not a boolean)")
	}
	valid := map[string]bool{"fresh": true, "stale": true, "lag": true, "unindexed": true}
	if !valid[out.Freshness] {
		t.Fatalf("freshness %q is not a valid 4-state value (fresh/stale/lag/unindexed)", out.Freshness)
	}
	if out.SchemaVersion <= 0 {
		t.Fatalf("code_status must report the schema version, got %d", out.SchemaVersion)
	}
	// Per-language tier: the Go file was indexed, so "go" must have a tier.
	if _, ok := out.Tiers["go"]; !ok {
		t.Fatalf("code_status must report per-language extractor tier for go, got: %v", out.Tiers)
	}
	_ = out.CoverageFair // present (may be 0 for a tiny fixture)
}

// TestCodeIndexQueuedWrite409 verifies the active-run guard (B3): a repeat
// code_index while the first is active returns 409 (operation_conflict). The
// guard is the CodeHandler's runRegistry — the one shared across the MCP +
// CLI surfaces (both call the same Index method). The 409 contract is the
// handler's: the errOperationConflict sentinel → envelope.CodeOperationConflict.
func TestCodeIndexQueuedWrite409(t *testing.T) {
	ch := newTestCodeHandler(t)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	ch.SetRootForTest(root)

	// A first code_index run completes (the run registers + clears itself).
	if _, err := ch.Index(ctx, "", false, false, false, false, false); err != nil {
		t.Fatalf("a first code_index run must succeed, got: %v", err)
	}
	if ch.HasActiveIndexForTest() {
		t.Fatalf("a completed code_index run must clear the active-run guard")
	}

	// Start an active run (the runRegistry's guard — the re-index is in
	// flight, e.g. a concurrent surface or the test seam).
	ch.StartIndexForTest()
	defer ch.FinishIndexForTest()

	// A repeat code_index while active → 409 operation_conflict (the guard,
	// B3). The handler returns the errOperationConflict sentinel; the
	// envelope maps it to CodeOperationConflict (the 409 the agent
	// recognizes — do not double-start).
	res, err := ch.Index(ctx, "", false, false, false, false, false)
	if err == nil {
		t.Fatalf("a repeat code_index while active must error (the guard), got res=%+v", res)
	}
	code, _ := envelope.MapError(err)
	if code != envelope.CodeOperationConflict {
		t.Fatalf("a repeat code_index while active must map to 409 operation_conflict, got code=%q (err=%v)", code, err)
	}
	// The guard is still active (the repeat did not clear it — the state is
	// the caller's, not this call's).
	if !ch.HasActiveIndexForTest() {
		t.Fatalf("a repeat code_index must not clear the active-run guard")
	}
}

// TestV21DeferredReportRunFlag verifies the v2.1-deferred tools are declared
// in the surface and report "run code_index --flag" (deferred, not missing).
func TestV21DeferredReportRunFlag(t *testing.T) {
	srv := NewServer(newTestHandler(t), All)
	defer srv.Close()
	srv.RegisterCodeTools(newTestCodeHandler(t))

	// code_pdg_query is v2.1-deferred: declared + reports run code_index --pdg.
	if !hasTool(srv, "code_pdg_query") {
		t.Fatalf("code_pdg_query must be declared in the surface (deferred, not missing)")
	}
	res := mustMap(t, srv.CallTool(ctx, "code_pdg_query", map[string]any{}))
	if !resIsError(res) {
		t.Fatalf("a v2.1-deferred tool must report its deferral, got: %v", res)
	}
	if !strings.Contains(str(res["message"]), "code_index --pdg") {
		t.Fatalf("code_pdg_query must report 'run code_index --pdg', got message=%v", res["message"])
	}

	// code_communities is v2.1-deferred too.
	if !hasTool(srv, "code_communities") {
		t.Fatalf("code_communities must be declared in the surface (deferred, not missing)")
	}
	res = mustMap(t, srv.CallTool(ctx, "code_communities", map[string]any{}))
	if !strings.Contains(str(res["message"]), "code_index --communities") {
		t.Fatalf("code_communities must report 'run code_index --communities', got message=%v", res["message"])
	}
}

// TestMCPCLIParity verifies the K5/K6 seam: the same search query via the
// CLI (CodeHandler.Search) and via the MCP tool (code_search) produces
// byte-identical JSON (the envelope marshal of the same CodeHandler return).
func TestMCPCLIParity(t *testing.T) {
	srv := NewServer(newTestHandler(t), Agent)
	defer srv.Close()
	ch := newTestCodeHandler(t)
	srv.RegisterCodeTools(ch)

	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc greet() string { return \"hello\" }\n\nfunc main() { _ = greet() }\n")
	indexFixtureTree(t, ch.Store(), root)
	ch.SetRootForTest(root)

	query := "greet"
	// CLI path: the same CodeHandler method the CLI calls, envelope-marshaled.
	cliResult, err := ch.Search(ctx, query, search.SearchQuery{Project: testProject, Limit: 10})
	if err != nil {
		t.Fatalf("CLI CodeHandler.Search: %v", err)
	}
	cliEnv, err := envelope.WithDegraded(cliResult, false, "")
	if err != nil {
		t.Fatalf("CLI envelope: %v", err)
	}
	cliBytes, _ := json.Marshal(cliEnv)

	// MCP path: the code_search tool call (the server's wrapEnvelope of the
	// same CodeHandler result).
	mcpBytes := srv.CallTool(ctx, "code_search", map[string]any{"query": query, "limit": 10})

	if string(cliBytes) != string(mcpBytes) {
		t.Fatalf("K5 parity broken:\nCLI: %s\nMCP: %s", cliBytes, mcpBytes)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────

func nameSet(srv *Server) map[string]bool {
	out := map[string]bool{}
	for _, n := range srv.ToolNames() {
		out[n] = true
	}
	return out
}

func hasTool(srv *Server, name string) bool {
	for _, n := range srv.ToolNames() {
		if n == name {
			return true
		}
	}
	return false
}

func mustMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return m
}

func keys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func resIsError(res map[string]any) bool {
	b, _ := res["error"].(bool)
	return b
}

// isJSON reports whether b is valid JSON (the CallTool's envelope check).
func isJSON(b []byte) bool {
	var v any
	return json.Unmarshal(b, &v) == nil
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func mustWriteFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for fixture %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", rel, err)
	}
}

// unused imports guard (the test file imports the packages it exercises).
var _ = graph.Impact
var _ = watch.NewWatcher
var _ = extract.Stamp
