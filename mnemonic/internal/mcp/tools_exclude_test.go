package mcp

import (
	"testing"

	mcplib "github.com/mark3labs/mcp-go/mcp"
)

// fakeTools returns two throwaway tools for filter assertions.
func fakeTools() []mcplib.Tool {
	return []mcplib.Tool{
		{Name: "mem_save"},
		{Name: "mem_search"},
		{Name: "code_explore"},
		{Name: "web_cache_search"},
	}
}

// TestMCPExcludeFilters is the acceptance scenario "MNEMONIC_MCP_EXCLUDE=mem
// suppresses mem_* tools": the Hermes MCP registration sets the env var so
// the server hides the tools the MemoryProvider handles natively, while
// every other harness (env unset) keeps its full tool list.
func TestMCPExcludeFilters(t *testing.T) {
	tools := fakeTools()

	// Unset: the filter is a pass-through (no exclusion), the list is
	// untouched.
	if f := applyExcludeToolFilter(); f == nil {
		t.Fatal("applyExcludeToolFilter() must never return nil (mcp-go tolerates no nil filters)")
	} else if got := f(nil, tools); len(got) != len(tools) {
		t.Fatalf("pass-through filter changed the list: %v", got)
	}

	// "mem": mem_* suppressed, everything else listed.
	t.Setenv("MNEMONIC_MCP_EXCLUDE", "mem")
	f := applyExcludeToolFilter()
	if f == nil {
		t.Fatal("applyExcludeToolFilter() with env set should be non-nil")
	}
	got := f(nil, tools)
	var names []string
	for _, t := range got {
		names = append(names, t.Name)
	}
	want := map[string]bool{"code_explore": true, "web_cache_search": true}
	if len(got) != len(want) {
		t.Fatalf("filtered tools = %v, want %v", names, want)
	}
	for _, n := range names {
		if !want[n] {
			t.Fatalf("unexpected tool %q after exclude=mem: %v", n, names)
		}
	}

	// Multiple prefixes: a comma-separated list suppresses each family.
	t.Setenv("MNEMONIC_MCP_EXCLUDE", "mem,web")
	got = applyExcludeToolFilter()(nil, tools)
	if len(got) != 1 || got[0].Name != "code_explore" {
		t.Fatalf("filtered tools with mem,web = %v, want [code_explore]", got)
	}

	// Whitespace is tolerated; empty/blank entries are dropped.
	t.Setenv("MNEMONIC_MCP_EXCLUDE", " mem , , web ")
	got = applyExcludeToolFilter()(nil, tools)
	if len(got) != 1 || got[0].Name != "code_explore" {
		t.Fatalf("filtered tools with spaced prefixes = %v, want [code_explore]", got)
	}

	// A bogus prefix excludes nothing.
	t.Setenv("MNEMONIC_MCP_EXCLUDE", "zzz")
	if got := applyExcludeToolFilter()(nil, tools); len(got) != len(tools) {
		t.Fatalf("bogus prefix changed the list: %v", got)
	}
}

// applyServerFilters runs the full filter chain (the same two filters the
// server wires in serverOptions) over a tool list and returns the names.
// This is the wire-level contract a real MCP client sees.
func applyServerFilters(t *testing.T, tools []mcplib.Tool) map[string]bool {
	t.Helper()
	out := tools
	if f := applyCodeToolFilter(); f != nil {
		out = f(nil, out)
	}
	if f := applyExcludeToolFilter(); f != nil {
		out = f(nil, out)
	}
	names := map[string]bool{}
	for _, tool := range out {
		names[tool.Name] = true
	}
	return names
}

// TestNewServer_ExcludesMemTools drives the real filter chain with the env
// var set: the wire tool list must contain the code_* and web_* tools but
// none of the mem_* tools (the Hermes registration contract).
func TestNewServer_ExcludesMemTools(t *testing.T) {
	t.Setenv("MNEMONIC_MCP_EXCLUDE", "mem")
	s := NewServer()
	raw := s.ListTools()
	if raw == nil {
		t.Fatal("ListTools returned nil")
	}
	tools := make([]mcplib.Tool, 0, len(raw))
	for _, st := range raw {
		tools = append(tools, st.Tool)
	}
	names := applyServerFilters(t, tools)
	if !names["code_explore"] {
		t.Fatal("code_explore missing from the filtered tool list")
	}
	if !names["web_cache_search"] {
		t.Fatal("web_cache_search missing from the filtered tool list")
	}
	for name := range names {
		if len(name) >= 4 && name[:4] == "mem_" {
			t.Fatalf("excluded tool %q is still listed (prefix mem_)", name)
		}
	}
}

// TestNewServer_NoExcludeByDefault is the no-regression side of the same
// contract: with the env var unset the full mem_* surface is listed.
func TestNewServer_NoExcludeByDefault(t *testing.T) {
	s := NewServer()
	raw := s.ListTools()
	if raw == nil {
		t.Fatal("ListTools returned nil")
	}
	tools := make([]mcplib.Tool, 0, len(raw))
	for _, st := range raw {
		tools = append(tools, st.Tool)
	}
	names := applyServerFilters(t, tools)
	if !names["mem_save"] {
		t.Fatal("mem_save missing with the env var unset")
	}
	if !names["code_explore"] {
		t.Fatal("code_explore missing with the env var unset")
	}
}
