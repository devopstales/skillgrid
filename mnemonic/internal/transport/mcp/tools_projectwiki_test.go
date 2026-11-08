// tools_projectwiki_test.go — the projectwiki_* surface tests (008-transport §3): the D1
// contract (the 12 tool names, the required-arg pins, the two-profile
// split) + the dispatch over a real fixture (a bootstrapped vault with a
// grounded page, the read tools' results, the write queue routing, the
// ownership assertion).

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/projectwiki"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/handler"
)

// projectwikiDetStub resolves explicit → explicit, else testProject (no git).
type projectwikiDetStub struct{}

func (projectwikiDetStub) Resolve(_ string, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return testProject, nil
}

// newProjectWikiFixture builds a server + ProjectWiki handler over a real SQLite store and a
// bootstrapped vault (a grounded decision page with a token anchor against
// a code file — the pin-check's fresh path). Returns the server, the
// store, and the vault dir.
func newProjectWikiFixture(t *testing.T, profile Profile) (*Server, store.Store, string) {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	h := newTestHandlerWithStore(t, st)
	vault := t.TempDir()
	ctx := context.Background()
	if _, err := projectwiki.Bootstrap(ctx, st, vault, projectwiki.Options{Project: testProject}); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vault, "auth.go"), []byte("const AuthModelV1 = 1\n"), 0o644); err != nil {
		t.Fatalf("write code file: %v", err)
	}
	page := `---
title: Auth model
type: decision
project: p
anchors:
  - path: auth.go
    all_of: [AuthModelV1]
---

# Auth model

Decide to use OAuth2 for the auth model.
`
	if err := os.MkdirAll(filepath.Join(vault, "decisions"), 0o755); err != nil {
		t.Fatalf("mkdir decisions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(vault, "decisions", "auth-model.md"), []byte(page), 0o644); err != nil {
		t.Fatalf("write page: %v", err)
	}
	kh := handler.NewProjectWikiHandler(st, config.Config{}, projectwikiDetStub{})
	kh.SetRoots(vault, vault)
	s := NewServer(h, profile, FamilyProjectWiki)
	s.RegisterProjectWikiTools(kh)
	t.Cleanup(s.Close)
	return s, st, vault
}

// initializeInstructions parses the initialize response frame (the
// HandshakeAndCall's frames[0]) and returns the instructions text mcp-go
// SERVES over stdio — the surface the agent actually sees. The Server's
// in-process field alone does not prove delivery (mcp-go captures the
// instructions at construction).
func initializeInstructions(t *testing.T, frames [][]byte) string {
	t.Helper()
	if len(frames) == 0 {
		t.Fatal("no frames from the handshake")
	}
	var rpc struct {
		Result struct {
			Instructions string `json:"instructions"`
		} `json:"result"`
	}
	if err := json.Unmarshal(frames[0], &rpc); err != nil {
		t.Fatalf("initialize frame is not JSON: %v (%s)", err, frames[0])
	}
	return rpc.Result.Instructions
}

// TestProjectWikiInstructionsServedOverStdio proves the projectwiki
// baseline block reaches the wire: the projectwiki server's stdio
// initialize response carries the retrieval discipline and the page-job
// lifecycle (generate/index/validate/lint), and the memory server does not
// carry the block.
func TestProjectWikiInstructionsServedOverStdio(t *testing.T) {
	s, _, _ := newProjectWikiFixture(t, All)
	frames, err := s.HandshakeAndCall(context.Background(), "projectwiki_status")
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	served := initializeInstructions(t, frames)
	if !strings.Contains(served, "Do not enumerate, preload, or search the vault at task start") {
		t.Fatalf("the projectwiki initialize block must carry the retrieval discipline, got: %s", served)
	}
	if !strings.Contains(served, "projectwiki_query") {
		t.Fatalf("the block must route projectwiki_query (the stage-1 entry), got: %s", served)
	}
	for _, name := range []string{"projectwiki_generate", "projectwiki_validate", "projectwiki_lint"} {
		if !strings.Contains(served, name) {
			t.Fatalf("the block must name the lifecycle tool %s, got: %s", name, served)
		}
	}
	// The memory server does not serve the projectwiki block.
	memSrv := NewServer(newTestHandler(t), Agent, FamilyMemory)
	defer memSrv.Close()
	frames2, err := memSrv.HandshakeAndCall(context.Background(), "mem_current_project")
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	if served := initializeInstructions(t, frames2); strings.Contains(served, "Do not enumerate, preload, or search the vault") {
		t.Fatalf("the memory server must not serve the projectwiki block, got: %s", served)
	}
}

// kbAgentTools / kbAdminTools are the 008-transport §2 profile split
// (agent = 8 working-set tools, all = +4 admin tools).
var (
	kbAgentTools = []string{
		"projectwiki_query", "projectwiki_read", "projectwiki_search", "projectwiki_generate",
		"projectwiki_validate", "projectwiki_lint", "projectwiki_status", "projectwiki_cite_source",
	}
	kbAdminTools = []string{
		"projectwiki_reindex", "projectwiki_index", "projectwiki_bootstrap", "projectwiki_rebuild_meta",
	}
)

// TestProjectWikiToolContract pins the D1 surface: the 12 tool names (All), the
// 8-tool agent profile (admin dropped), and the required-arg pins
// (projectwiki_query's query, projectwiki_read's path, the write tools' expected_project).
func TestProjectWikiToolContract(t *testing.T) {
	all, _, _ := newProjectWikiFixture(t, All)
	names := map[string]bool{}
	for _, n := range all.ToolNames() {
		names[n] = true
	}
	for _, w := range append(append([]string{}, kbAgentTools...), kbAdminTools...) {
		if !names[w] {
			t.Errorf("All profile missing tool %s", w)
		}
	}
	if !all.ToolSchema("projectwiki_query").Required["query"] {
		t.Error("projectwiki_query: query must be a required arg")
	}
	if !all.ToolSchema("projectwiki_read").Required["path"] {
		t.Error("projectwiki_read: path must be a required arg")
	}
	if !all.ToolSchema("projectwiki_generate").Required["expected_project"] {
		t.Error("projectwiki_generate: expected_project must be a required arg")
	}
	for _, r := range []string{"page", "name", "expected_project"} {
		if !all.ToolSchema("projectwiki_cite_source").Required[r] {
			t.Errorf("projectwiki_cite_source: %s must be a required arg", r)
		}
	}

	agent, _, _ := newProjectWikiFixture(t, Agent)
	anames := map[string]bool{}
	for _, n := range agent.ToolNames() {
		anames[n] = true
	}
	for _, w := range kbAgentTools {
		if !anames[w] {
			t.Errorf("agent profile missing tool %s", w)
		}
	}
	for _, a := range kbAdminTools {
		if anames[a] {
			t.Errorf("agent profile must not expose admin tool %s", a)
		}
	}
}

// TestProjectWikiDispatchReads drives the read path over the fixture: index →
// query → read (with the pin-check) → search → validate → lint → status.
func TestProjectWikiDispatchReads(t *testing.T) {
	s, _, _ := newProjectWikiFixture(t, All)
	ctx := context.Background()

	if out := s.CallTool(ctx, "projectwiki_index", map[string]any{"project": testProject}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("projectwiki_index: %s", out)
	}
	if out := s.CallTool(ctx, "projectwiki_query", map[string]any{"query": "auth model"}); !strings.Contains(string(out), "decisions/auth-model.md") {
		t.Fatalf("projectwiki_query must return the grounded page, got: %s", out)
	}
	read := s.CallTool(ctx, "projectwiki_read", map[string]any{"path": "decisions/auth-model.md"})
	if !strings.Contains(string(read), "OAuth2") || !strings.Contains(string(read), "fresh") {
		t.Fatalf("projectwiki_read must return the body + the fresh pin-check, got: %s", read)
	}
	if out := s.CallTool(ctx, "projectwiki_search", map[string]any{"query": "OAuth2"}); !strings.Contains(string(out), "decisions/auth-model.md") {
		t.Fatalf("projectwiki_search must return the page, got: %s", out)
	}
	if out := s.CallTool(ctx, "projectwiki_validate", map[string]any{}); !strings.Contains(string(out), `"ok":true`) {
		t.Fatalf("projectwiki_validate must report ok (the anchor is live), got: %s", out)
	}
	if out := s.CallTool(ctx, "projectwiki_lint", map[string]any{}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("projectwiki_lint: %s", out)
	}
	if out := s.CallTool(ctx, "projectwiki_status", map[string]any{"project": testProject}); !strings.Contains(string(out), "indexed") {
		t.Fatalf("projectwiki_status must report the indexed count, got: %s", out)
	}
}

// TestProjectWikiCiteSourceIdempotent pins projectwiki_cite_source's record semantics: the
// first call records the external citation (the build fingerprint), the
// repeat is a no-op (already_cited).
func TestProjectWikiCiteSourceIdempotent(t *testing.T) {
	s, _, vault := newProjectWikiFixture(t, All)
	ctx := context.Background()
	args := map[string]any{
		"page": "decisions/auth-model.md", "name": "rfc6749", "version": "2012",
		"source_url":       "https://datatracker.ietf.org/doc/html/rfc6749",
		"expected_project": testProject,
	}
	first := s.CallTool(ctx, "projectwiki_cite_source", args)
	if !strings.Contains(string(first), "external:rfc6749") || !strings.Contains(string(first), "build_fingerprint") {
		t.Fatalf("projectwiki_cite_source must record the citation, got: %s", first)
	}
	again := s.CallTool(ctx, "projectwiki_cite_source", args)
	if !strings.Contains(string(again), "already_cited") {
		t.Fatalf("repeat projectwiki_cite_source must report already_cited, got: %s", again)
	}
	// The citation is in the page file (PW-INV-1: files are truth).
	b, err := os.ReadFile(filepath.Join(vault, "decisions", "auth-model.md"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !strings.Contains(string(b), "external: rfc6749") {
		t.Fatalf("the page must carry the external citation, got: %s", b)
	}
}

// TestProjectWikiGenerateQueueAndOwnership pins projectwiki_generate's write
// semantics: the write runs through the ProjectWiki write queue (the family
// split means the wiki server never touches the memory queue — it carries no
// mem tools) and the expected_project mismatch is an error, not a silent
// generate.
func TestProjectWikiGenerateQueueAndOwnership(t *testing.T) {
	s, _, _ := newProjectWikiFixture(t, All)
	ctx := context.Background()
	if out := s.CallTool(ctx, "projectwiki_generate", map[string]any{"expected_project": testProject}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("projectwiki_generate: %s", out)
	}
	if !s.ProjectWikiWritesSerialized() {
		t.Fatal("projectwiki_generate must run through the ProjectWiki write queue")
	}
	if out := s.CallTool(ctx, "projectwiki_generate", map[string]any{"expected_project": "other-project"}); !strings.Contains(string(out), `"error"`) {
		t.Fatalf("projectwiki_generate with a mismatched expected_project must error, got: %s", out)
	}
	// The family split: the wiki server carries no memory tools.
	if _, ok := s.toolDefs["mem_save"]; ok {
		t.Fatal("the wiki server must not register mem_save (family split)")
	}
}
