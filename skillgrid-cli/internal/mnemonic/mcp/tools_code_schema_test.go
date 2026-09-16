package mcp

import (
	"sort"
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// expected005ToolSurface pins the 005 code_* tool contract: each tool's name
// and its required parameter list. Later changes (community tools, hybrid
// tools, ...) must never rename a 005 tool or change its required params —
// this map is the baseline lock that fails when they do.
var expected005ToolSurface = map[string][]string{
	"code_status":           {},
	"code_index":            {},
	"code_search":           {"query"},
	"code_read":             {"path"},
	"code_orient":           {"symbol"},
	"code_signature":        {"symbol"},
	"code_file_toc":         {"symbol"},
	"code_rationale":        {"symbol"},
	"code_grep":             {"pattern"},
	"code_get_callers":      {"symbol"},
	"code_get_callees":      {"symbol"},
	"code_get_dependents":   {"symbol"},
	"code_get_implementors": {"symbol"},
	"code_get_hierarchy":    {"symbol"},
	"code_get_tests_for":    {"symbol"},
	"code_path":             {"from", "to"},
	"code_explain":          {"symbol"},
	"code_impact":           {"symbol"},
	"code_explore":          {"symbol"},
}

// TestCodeSearchSchemaStable locks the code_search tool's name and required
// `query` param schema so it cannot silently regress when new tools are added.
// Threat: Mnemonic tool surface. The existing four code_* tools must keep
// their name and required-param schema (additive fields only).
func TestCodeSearchSchemaStable(t *testing.T) {
	tool := codeSearchTool()
	if tool.Name != "code_search" {
		t.Fatalf("code_search tool name changed to %q", tool.Name)
	}
	schema := tool.InputSchema
	// `query` must be a required string parameter.
	props, ok := schema.Properties["query"]
	if !ok {
		t.Fatalf("code_search input schema is missing the required 'query' property: %+v", schema.Properties)
	}
	if _, isMap := props.(map[string]any); !isMap {
		t.Fatalf("code_search 'query' property is not a JSON schema object: %T", props)
	}
	if !containsString(schema.Required, "query") {
		t.Fatalf("code_search 'query' is not a required parameter; required=%v", schema.Required)
	}
	// The other three existing tools keep their names + required params.
	assertCodeToolStable(t, codeIndexTool(), "code_index", nil)
	assertCodeToolStable(t, codeStatusTool(), "code_status", nil)
	assertCodeToolStable(t, codeReadTool(), "code_read", []string{"path"})
}

// TestToolSurfaceBaseline covers the 005 tool-surface lock (Scenario:
// code_communities returns labeled subsystems and 005 tools stay stable):
// every 005 code_* tool keeps its exact name and required-param contract on
// the live MCP server, not just its individual Tool constructor.
func TestToolSurfaceBaseline(t *testing.T) {
	tools := NewServer().ListTools()
	for name, wantRequired := range expected005ToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 tool %q is no longer registered", name)
			continue
		}
		got := append([]string(nil), st.Tool.InputSchema.Required...)
		want := append([]string(nil), wantRequired...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("%q: required params changed: got %v, want %v", name, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%q: required param %d changed: got %q, want %q", name, i, got[i], want[i])
			}
		}
	}
}

// assertCodeToolStable locks a tool's name and its required parameters.
func assertCodeToolStable(t *testing.T, tool mcplib.Tool, name string, required []string) {
	t.Helper()
	if tool.Name != name {
		t.Errorf("tool name changed: got %q, want %q", tool.Name, name)
	}
	got := append([]string(nil), tool.InputSchema.Required...)
	want := append([]string(nil), required...)
	sort.Strings(got)
	sort.Strings(want)
	if len(got) != len(want) {
		t.Errorf("%s required params = %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s required params = %v, want %v", name, got, want)
			break
		}
	}
}

func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
