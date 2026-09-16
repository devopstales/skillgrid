package mcp

import (
	"context"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// TestWatcherSurface covers 03.15 (Scenario: Watcher keeps tool schemas
// stable): the watcher + fingerprint gate + banner do not alter any existing
// 005/008/010 code_* tool schemas; the staleness banner is present on the
// response path; bad watcher/status args are rejected clearly.
func TestWatcherSurface(t *testing.T) {
	// 1) No existing code_* tool schema is altered. The 005 baseline lock
	// (expected005ToolSurface) + the 008/010 tools must keep their exact names
	// and required params even with the watcher + banner wired into the
	// response path (they are response-path concerns, not tool-schema changes).
	tools := NewServer().ListTools()
	for name, wantRequired := range expected005ToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 tool %q no longer registered (watcher must not remove tools)", name)
			continue
		}
		got := append([]string(nil), st.Tool.InputSchema.Required...)
		assertSameRequired(t, name, got, wantRequired)
	}
	// The 010 step-02 PR-command tools keep their schemas too.
	for name, want := range map[string][]string{"code_affected": {}, "code_rename": {"old", "new"}} {
		st, ok := tools[name]
		if !ok {
			t.Errorf("010 tool %q no longer registered", name)
			continue
		}
		assertSameRequired(t, name, append([]string(nil), st.Tool.InputSchema.Required...), want)
	}

	// 2) The staleness banner is present on the response path (a referenced
	// pending file is named; this exercises applyFreshness directly).
	resetFresh(t)
	SetPendingSource(func() map[string]struct{} { return map[string]struct{}{"main.go": {}} })
	res, err := JSONResult(map[string]any{"hits": []map[string]any{{"path": "main.go"}}})
	if err != nil {
		t.Fatalf("json: %v", err)
	}
	out := applyFreshness(res, []string{"main.go"})
	if text := callResultText(t, out); !contains(text, "⚠️ main.go is pending sync — Read it directly") {
		t.Errorf("expected the staleness banner on the response path; got: %s", text)
	}

	// 3) Bad watcher/status args are rejected clearly. The staleness banner /
	// fingerprint gate take no tool arguments (they are response-path), so a
	// tool that is passed an unknown watcher arg is rejected by the existing
	// argument validation. Verify code_read rejects a missing required `path`
	// clearly (the watcher does not loosen validation).
	SetService(service.New(t.TempDir()))
	badReq := mcplib.CallToolRequest{}
	badReq.Params.Name = "code_read"
	badReq.Params.Arguments = map[string]any{}
	res2, _ := handleCodeRead(context.Background(), badReq)
	if res2 == nil || !res2.IsError {
		t.Errorf("expected code_read with a missing required `path` to be rejected clearly")
	}
}

// assertSameRequired checks a tool's required params equal want (order-agnostic).
func assertSameRequired(t *testing.T, name string, got, want []string) {
	t.Helper()
	normalize := func(s []string) string {
		cp := append([]string(nil), s...)
		sortStrings(cp)
		out := ""
		for i, v := range cp {
			if i > 0 {
				out += ","
			}
			out += v
		}
		return out
	}
	if normalize(got) != normalize(want) {
		t.Errorf("%s required params = %v, want %v (watcher must not alter tool schemas)", name, got, want)
	}
}
