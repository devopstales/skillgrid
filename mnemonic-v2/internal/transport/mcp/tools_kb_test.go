// tools_kb_test.go — the kb_* surface tests (008-transport §3): the D1
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

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/kb"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// kbDetStub resolves explicit → explicit, else testProject (no git).
type kbDetStub struct{}

func (kbDetStub) Resolve(_ string, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return testProject, nil
}

// newKbFixture builds a server + KB handler over a real SQLite store and a
// bootstrapped vault (a grounded decision page with a token anchor against
// a code file — the pin-check's fresh path). Returns the server, the
// store, and the vault dir.
func newKbFixture(t *testing.T, profile Profile) (*Server, store.Store, string) {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	h := newTestHandlerWithStore(t, st)
	vault := t.TempDir()
	ctx := context.Background()
	if _, err := kb.Bootstrap(ctx, st, vault, kb.Options{Project: testProject}); err != nil {
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
	kh := handler.NewKbHandler(st, config.Config{}, kbDetStub{})
	kh.SetRoots(vault, vault)
	s := NewServer(h, profile)
	s.RegisterKbTools(kh)
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

// TestKbInstructionsServedOverStdio proves the KB instructions reach the
// wire: the stdio initialize response carries the "### KnowledgeBase"
// section after RegisterKbTools — and does not before. This is the
// regression net for the capture-timing flaw (the mcp-go server captured
// the instructions at construction, so a field rewrite was never served;
// setInstructions rebuilds the server to deliver the updated text).
func TestKbInstructionsServedOverStdio(t *testing.T) {
	s, _, _ := newKbFixture(t, All)
	frames, err := s.HandshakeAndCall(context.Background(), "kb_status")
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	served := initializeInstructions(t, frames)
	if !strings.Contains(served, "### KnowledgeBase") {
		t.Fatalf("the stdio initialize response must carry the KnowledgeBase section, got: %s", served)
	}
	if !strings.Contains(served, "kb_query") {
		t.Fatalf("the KB section must route kb_query (the stage-1 entry), got: %s", served)
	}
	// A server without KB registration does not carry the section.
	memOnly := NewServer(newTestHandler(t), Agent)
	defer memOnly.Close()
	frames2, err := memOnly.HandshakeAndCall(context.Background(), "mem_current_project")
	if err != nil {
		t.Fatalf("handshake: %v", err)
	}
	if served := initializeInstructions(t, frames2); strings.Contains(served, "KnowledgeBase") {
		t.Fatalf("a server without KB tools must not carry the KB section, got: %s", served)
	}
}

// kbAgentTools / kbAdminTools are the 008-transport §2 profile split
// (agent = 8 working-set tools, all = +4 admin tools).
var (
	kbAgentTools = []string{
		"kb_query", "kb_read", "kb_search", "kb_generate",
		"kb_validate", "kb_lint", "kb_status", "kb_cite_source",
	}
	kbAdminTools = []string{
		"kb_reindex", "kb_index", "kb_bootstrap", "kb_rebuild_meta",
	}
)

// TestKbToolContract pins the D1 surface: the 12 tool names (All), the
// 8-tool agent profile (admin dropped), and the required-arg pins
// (kb_query's query, kb_read's path, the write tools' expected_project).
func TestKbToolContract(t *testing.T) {
	all, _, _ := newKbFixture(t, All)
	names := map[string]bool{}
	for _, n := range all.ToolNames() {
		names[n] = true
	}
	for _, w := range append(append([]string{}, kbAgentTools...), kbAdminTools...) {
		if !names[w] {
			t.Errorf("All profile missing tool %s", w)
		}
	}
	if !all.ToolSchema("kb_query").Required["query"] {
		t.Error("kb_query: query must be a required arg")
	}
	if !all.ToolSchema("kb_read").Required["path"] {
		t.Error("kb_read: path must be a required arg")
	}
	if !all.ToolSchema("kb_generate").Required["expected_project"] {
		t.Error("kb_generate: expected_project must be a required arg")
	}
	for _, r := range []string{"page", "name", "expected_project"} {
		if !all.ToolSchema("kb_cite_source").Required[r] {
			t.Errorf("kb_cite_source: %s must be a required arg", r)
		}
	}

	agent, _, _ := newKbFixture(t, Agent)
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

// TestKbDispatchReads drives the read path over the fixture: index →
// query → read (with the pin-check) → search → validate → lint → status.
func TestKbDispatchReads(t *testing.T) {
	s, _, _ := newKbFixture(t, All)
	ctx := context.Background()

	if out := s.CallTool(ctx, "kb_index", map[string]any{"project": testProject}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("kb_index: %s", out)
	}
	if out := s.CallTool(ctx, "kb_query", map[string]any{"query": "auth model"}); !strings.Contains(string(out), "decisions/auth-model.md") {
		t.Fatalf("kb_query must return the grounded page, got: %s", out)
	}
	read := s.CallTool(ctx, "kb_read", map[string]any{"path": "decisions/auth-model.md"})
	if !strings.Contains(string(read), "OAuth2") || !strings.Contains(string(read), "fresh") {
		t.Fatalf("kb_read must return the body + the fresh pin-check, got: %s", read)
	}
	if out := s.CallTool(ctx, "kb_search", map[string]any{"query": "OAuth2"}); !strings.Contains(string(out), "decisions/auth-model.md") {
		t.Fatalf("kb_search must return the page, got: %s", out)
	}
	if out := s.CallTool(ctx, "kb_validate", map[string]any{}); !strings.Contains(string(out), `"ok":true`) {
		t.Fatalf("kb_validate must report ok (the anchor is live), got: %s", out)
	}
	if out := s.CallTool(ctx, "kb_lint", map[string]any{}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("kb_lint: %s", out)
	}
	if out := s.CallTool(ctx, "kb_status", map[string]any{"project": testProject}); !strings.Contains(string(out), "indexed") {
		t.Fatalf("kb_status must report the indexed count, got: %s", out)
	}
}

// TestKbCiteSourceIdempotent pins kb_cite_source's record semantics: the
// first call records the external citation (the build fingerprint), the
// repeat is a no-op (already_cited).
func TestKbCiteSourceIdempotent(t *testing.T) {
	s, _, vault := newKbFixture(t, All)
	ctx := context.Background()
	args := map[string]any{
		"page": "decisions/auth-model.md", "name": "rfc6749", "version": "2012",
		"source_url":       "https://datatracker.ietf.org/doc/html/rfc6749",
		"expected_project": testProject,
	}
	first := s.CallTool(ctx, "kb_cite_source", args)
	if !strings.Contains(string(first), "external:rfc6749") || !strings.Contains(string(first), "build_fingerprint") {
		t.Fatalf("kb_cite_source must record the citation, got: %s", first)
	}
	again := s.CallTool(ctx, "kb_cite_source", args)
	if !strings.Contains(string(again), "already_cited") {
		t.Fatalf("repeat kb_cite_source must report already_cited, got: %s", again)
	}
	// The citation is in the page file (KB-INV-1: files are truth).
	b, err := os.ReadFile(filepath.Join(vault, "decisions", "auth-model.md"))
	if err != nil {
		t.Fatalf("read page: %v", err)
	}
	if !strings.Contains(string(b), "external: rfc6749") {
		t.Fatalf("the page must carry the external citation, got: %s", b)
	}
}

// TestKbGenerateQueueAndOwnership pins kb_generate's write semantics: the
// write runs through the KB write queue (distinct from the memory queue)
// and the expected_project mismatch is an error, not a silent generate.
func TestKbGenerateQueueAndOwnership(t *testing.T) {
	s, _, _ := newKbFixture(t, All)
	ctx := context.Background()
	// A memory write on the memory queue (the KB queue must stay distinct).
	if out := s.CallTool(ctx, "mem_save", saveArgs(1)); strings.Contains(string(out), `"error"`) {
		t.Fatalf("mem_save: %s", out)
	}
	if !s.WritesSerialized() {
		t.Fatal("mem_save must run through the memory write queue")
	}
	if out := s.CallTool(ctx, "kb_generate", map[string]any{"expected_project": testProject}); strings.Contains(string(out), `"error"`) {
		t.Fatalf("kb_generate: %s", out)
	}
	if !s.KbWritesSerialized() {
		t.Fatal("kb_generate must run through the KB write queue")
	}
	if out := s.CallTool(ctx, "kb_generate", map[string]any{"expected_project": "other-project"}); !strings.Contains(string(out), `"error"`) {
		t.Fatalf("kb_generate with a mismatched expected_project must error, got: %s", out)
	}
}
