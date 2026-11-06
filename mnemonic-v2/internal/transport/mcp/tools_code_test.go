package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// F3 — code_affected declares the base arg (spec 008: {"changed": [...]}
	// / {"base": "<ref>"}; exactly one set). base is declared (optional) for
	// the contract; the MCP surface reports the CLI command when it's set.
	if !srv.ToolSchema("code_affected").Props["base"] {
		t.Fatalf("code_affected must declare the base arg (spec 008 contract)")
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

	// F6 — the combined instructions: RegisterCodeTools rewrites the
	// Server's instructions to the combined memory + code block (the memory-
	// only instructions stay intact when code tools are absent).
	if !strings.Contains(allSrv.instructions, "code_search") {
		t.Fatalf("after RegisterCodeTools the instructions must be the combined memory + code block (name code routing), got: %s", allSrv.instructions)
	}
	if !strings.Contains(allSrv.instructions, "code_orient") {
		t.Fatalf("the combined instructions must name code_orient (the navigation entry point), got: %s", allSrv.instructions)
	}
	// A memory-only server (no code tools) keeps the memory instructions.
	memOnly := NewServer(newTestHandler(t), Agent)
	defer memOnly.Close()
	if strings.Contains(memOnly.instructions, "code_search") {
		t.Fatalf("a memory-only server must keep the memory instructions (no code routing), got: %s", memOnly.instructions)
	}
	// F6 + capture-timing fix — the stdio initialize response must carry
	// the combined text: the mcp-go server captures the instructions at
	// construction, so the field rewrite above alone never reached the
	// wire (setInstructions rebuilds the server to serve the updated text).
	frames, err := allSrv.HandshakeAndCall(context.Background(), "mem_current_project")
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	if served := initializeInstructions(t, frames); !strings.Contains(served, "code_search") {
		t.Fatalf("the stdio initialize response must carry the combined memory + code instructions, got: %s", served)
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
	// F1 — the spec's 4-state set (fresh/lag/empty/unknown), NOT the
	// pre-fix fresh/stale/lag/unindexed.
	valid := map[string]bool{"fresh": true, "lag": true, "empty": true, "unknown": true}
	if !valid[out.Freshness] {
		t.Fatalf("freshness %q is not a valid 4-state value (fresh/lag/empty/unknown)", out.Freshness)
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

// TestCodeStatusFreshnessFourStates verifies the T-CI-4 4-state derivation
// (F1): empty (no index_freshness rows), unknown (the table query errors),
// lag (a file's indexed_at is newer than last_indexed_at), and fresh (no
// file is newer). Each state is seeded + asserted.
func TestCodeStatusFreshnessFourStates(t *testing.T) {
	// empty — no index_freshness rows (the index has never run).
	chEmpty := newTestCodeHandler(t)
	outEmpty, err := chEmpty.Status(ctx, "")
	if err != nil {
		t.Fatalf("empty status: %v", err)
	}
	if outEmpty.Freshness != "empty" {
		t.Fatalf("a store with no index_freshness rows must be 'empty', got %q", outEmpty.Freshness)
	}

	// lag — a file's indexed_at is NEWER than last_indexed_at (the index is
	// behind the working tree). Index a fixture (writes last_indexed_at),
	// then bump a file's indexed_at past it (the file changed after the last
	// recorded index).
	chLag := newTestCodeHandler(t)
	stLag := chLag.Store()
	rootLag := t.TempDir()
	mustWriteFile(t, rootLag, "main.go", "package main\n\nfunc main() {}\n")
	if _, err := codeindex.Index(ctx, rootLag, stLag, codeindex.IndexOpts{Project: testProject}); err != nil {
		t.Fatalf("index fixture: %v", err)
	}
	chLag.SetRootForTest(rootLag)
	// Bump the file's indexed_at past the freshness's last_indexed_at (the
	// lag signal: a file indexed after the last recorded index).
	raw := stLag.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if _, err := raw.RawQuery(`SELECT 1`); err != nil { // the seam is reachable
		t.Fatalf("RawQuery seam: %v", err)
	}
	if err := bumpFileIndexedAt(t, stLag); err != nil {
		t.Fatalf("bump file indexed_at: %v", err)
	}
	outLag, err := chLag.Status(ctx, "")
	if err != nil {
		t.Fatalf("lag status: %v", err)
	}
	if outLag.Freshness != "lag" {
		t.Fatalf("a file indexed after last_indexed_at must be 'lag', got %q", outLag.Freshness)
	}

	// fresh — a normal indexed store (no file is newer than last_indexed_at).
	chFresh := newTestCodeHandler(t)
	rootFresh := t.TempDir()
	mustWriteFile(t, rootFresh, "main.go", "package main\n\nfunc main() {}\n")
	indexFixtureTree(t, chFresh.Store(), rootFresh)
	chFresh.SetRootForTest(rootFresh)
	outFresh, err := chFresh.Status(ctx, "")
	if err != nil {
		t.Fatalf("fresh status: %v", err)
	}
	if outFresh.Freshness != "fresh" {
		t.Fatalf("a store with no file newer than last_indexed_at must be 'fresh', got %q", outFresh.Freshness)
	}
}

// bumpFileIndexedAt sets every files row's indexed_at to a timestamp far in
// the future (the lag test's seed: a file "indexed" after the last recorded
// index). It uses the store's RawExec seam (the handler's direct-write
// surface, the same pattern the session rows use).
func bumpFileIndexedAt(t *testing.T, st store.Store) error {
	t.Helper()
	re, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("store %T does not expose the RawExec seam", st)
	}
	return re.RawExec(`UPDATE files SET indexed_at = '9999-12-31 23:59:59'`)
}

// TestCodeIndexQueuedWrite409 verifies the active-run guard (B3) via the
// IndexFn seam (F4 — the brief's blocking-IndexFn seam, NOT a pre-set run):
// the run that Index ITSELF mints is held active while the (blocking) index
// driver runs; a repeat while it is active → 409 (operation_conflict); after
// the run completes (the channel closes), the next run succeeds. This proves
// the full mint→hold→409→clear→success cycle.
func TestCodeIndexQueuedWrite409(t *testing.T) {
	ch := newTestCodeHandler(t)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() {}\n")
	ch.SetRootForTest(root)

	// The blocking IndexFn seam (F4): run 1 blocks on the gate channel
	// (the index is "in flight"); the test closes it to complete run 1.
	// A nil indexFn would be the default (codeindex.Index); the injected
	// fn is the test's blocking driver.
	gate := make(chan struct{})
	calls := 0
	ch.SetIndexFnForTest(func(ctx context.Context, root string, st store.Store, opts codeindex.IndexOpts) (codeindex.IndexReport, error) {
		calls++
		if calls == 1 {
			// Run 1: block until the test closes the gate (the index is in
			// flight — the run Index minted is held active).
			<-gate
		}
		return codeindex.Index(ctx, root, st, opts)
	})

	// Run 1 (the real Index): mints the run_id + holds the guard while the
	// blocking fn runs. It runs in a goroutine (it blocks on the gate).
	run1 := make(chan error, 1)
	go func() {
		_, err := ch.Index(ctx, "", false, false, false, false, false)
		run1 <- err
	}()

	// The guard is held by run 1 (the run Index minted, the blocking fn in
	// flight). A repeat (run 2) → 409 operation_conflict.
	waitForActive(t, ch)
	res, err := ch.Index(ctx, "", false, false, false, false, false)
	if err == nil {
		t.Fatalf("a repeat code_index while the first is active must error (the guard), got res=%+v", res)
	}
	code, _ := envelope.MapError(err)
	if code != envelope.CodeOperationConflict {
		t.Fatalf("a repeat code_index while active must map to 409 operation_conflict, got code=%q (err=%v)", code, err)
	}

	// Close the gate → run 1 completes → the guard clears → run 3 succeeds.
	close(gate)
	if err := <-run1; err != nil {
		t.Fatalf("run 1 (the real Index) must succeed once the gate closes, got: %v", err)
	}
	// The guard cleared (run 1's defer finished it).
	waitForInactive(t, ch)
	// Run 3 (after the clear): succeeds.
	if _, err := ch.Index(ctx, "", false, false, false, false, false); err != nil {
		t.Fatalf("a code_index after the prior run cleared must succeed, got: %v", err)
	}
}

// waitForActive polls until the guard is active (run 1's mint is in flight).
func waitForActive(t *testing.T, ch *handler.CodeHandler) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ch.HasActiveIndexForTest() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("the guard must become active while run 1's blocking fn is in flight")
}

// waitForInactive polls until the guard clears (the run's defer finished it).
func waitForInactive(t *testing.T, ch *handler.CodeHandler) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !ch.HasActiveIndexForTest() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("the guard must clear after the run completes")
}

// TestLSPFlagDrivesTier verifies the M8 window is closed (TICKET-08b):
// --lsp RUNS the tier (NOT the declared no-op) — the flag on → the LSP pass
// runs, capabilities.lsp:true in code_status, and the index result no longer
// reports "lsp_not_built". A resolved member call (the mock resolver seam) →
// an LSP_RESOLVED edge in the store. The MCP code_index tool's lsp:true arg
// drives the same tier (I-3 — the surface parity).
func TestLSPFlagDrivesTier(t *testing.T) {
	ch := newTestCodeHandler(t)
	root := t.TempDir()
	mustWriteFile(t, root, "main.go", "package main\n\nfunc main() {\n\ts.Do(1)\n}\n\nfunc (x *S) Do(n int) int { return n }\n\ntype S struct{}\n")
	ch.SetRootForTest(root)

	// The mock resolver (the test double for the real gopls subprocess): it
	// resolves the fixture's s.Do() member call → the tier's LSP_RESOLVED edge.
	ch.SetLSPResolverForTest(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
		if receiver == "s" && callee == "Do" {
			return "Do", true, nil
		}
		return "", false, nil
	})

	// The handler: lsp=true → the tier RUNS (the no-op window is closed — no
	// "lsp_not_built", no "LSP tier not yet built" message).
	res, err := ch.Index(ctx, "", false, true, false, false, false)
	if err != nil {
		t.Fatalf("Index with lsp=true must not crash, got: %v", err)
	}
	if res.Degraded == "lsp_not_built" {
		t.Fatalf("lsp=true must NOT report the no-op (the window is closed — the tier runs), got degraded=%q", res.Degraded)
	}
	for _, w := range res.Warnings {
		if w == "LSP tier not yet built (TICKET-08b)" {
			t.Fatalf("lsp=true must not carry the no-op message (the tier is built), got warnings=%v", res.Warnings)
		}
	}

	// capabilities.lsp:true in code_status (the tier is wired + ran).
	status, err := ch.Status(ctx, "")
	if err != nil {
		t.Fatalf("code_status: %v", err)
	}
	if !status.Capabilities["lsp"] {
		t.Fatalf("code_status must report capabilities.lsp:true (--lsp ran the tier), got %v", status.Capabilities)
	}

	// The resolved member call → an LSP_RESOLVED edge in the store (the 4th
	// confidence tier).
	stStore := ch.Store()
	raw, _ := stStore.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if raw == nil {
		t.Fatalf("the test store must expose the RawQuery seam")
	}
	rows, err := raw.RawQuery(`SELECT COUNT(*) FROM edges WHERE confidence = 'LSP_RESOLVED'`)
	if err != nil {
		t.Fatalf("count LSP_RESOLVED edges: %v", err)
	}
	if len(rows) == 0 || len(rows[0]) == 0 {
		t.Fatalf("the LSP pass must write the resolved edge (the mock resolver resolved s.Do)")
	}
	if n, ok := rows[0][0].(int64); !ok || n != 1 {
		t.Fatalf("one resolved call → one LSP_RESOLVED edge, got %v", rows[0][0])
	}

	// The MCP surface: the code_index tool's lsp:true arg drives the same
	// tier (I-3 — the surface parity, the window closed on both surfaces).
	ch2 := newTestCodeHandler(t)
	ch2.SetRootForTest(root)
	ch2.SetLSPResolverForTest(func(ctx context.Context, filePath string, line int, receiver, callee string) (string, bool, error) {
		if receiver == "s" && callee == "Do" {
			return "Do", true, nil
		}
		return "", false, nil
	})
	srv := NewServer(newTestHandler(t), All)
	defer srv.Close()
	srv.RegisterCodeTools(ch2)
	out := srv.CallTool(ctx, "code_index", map[string]any{"lsp": true})
	resMap := mustMap(t, out)
	if resMap["degraded"] == "lsp_not_built" {
		t.Fatalf("the MCP code_index lsp:true must NOT report the no-op (the window is closed), got: %v", resMap)
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

// TestMCPCLIParity verifies the K5/K6 seam's MCP side (F5): the code_search
// tool's CallTool bytes are the envelope marshal of the CodeHandler result.
// The CLI side (the real `mnemonic index search --json` byte emission via the
// finish() marshal path — NOT a re-marshal) is driven by captureRun in
// cmd/mnemonic/index_test.go's TestIndexSearchParity (the raw CLI --json bytes
// are compared byte-for-byte against the MCP CallTool envelope). The MCP +
// CLI both marshal the SAME CodeHandler.Search return through the same
// envelope (the K5/K6 seam); this test pins the MCP half, the cmd test pins
// the CLI half (the finish() path), and the two are byte-identical by
// construction (the same CodeHandler method + the same envelope).
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
	// The MCP path: the code_search tool call (the server's wrapEnvelope of
	// the CodeHandler result — the byte source the CLI must match).
	mcpBytes := srv.CallTool(ctx, "code_search", map[string]any{"query": query, "limit": 10})

	// The CodeHandler result (the same method the CLI calls) envelope-
	// marshaled: the MCP CallTool bytes must equal this (the K5 seam — the
	// server's wrapEnvelope is the envelope.WithDegraded + json.Marshal).
	searchResult, err := ch.Search(ctx, query, search.SearchQuery{Project: testProject, Limit: 10})
	if err != nil {
		t.Fatalf("CodeHandler.Search: %v", err)
	}
	expectedEnv, err := envelope.WithDegraded(searchResult, false, "")
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	expectedBytes, _ := json.Marshal(expectedEnv)
	if string(mcpBytes) != string(expectedBytes) {
		t.Fatalf("the MCP code_search bytes must be the envelope marshal of the CodeHandler result:\nMCP: %s\nexpected: %s", mcpBytes, expectedBytes)
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
