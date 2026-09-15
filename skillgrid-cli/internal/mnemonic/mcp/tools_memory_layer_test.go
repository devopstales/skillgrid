package mcp

import (
	"context"
	"sort"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// memLayerFixture pins the project to a stable bucket so handlers that open the
// CWD project resolve to one store (mirrors memGovernanceFixture).
func memLayerFixture(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	t.Setenv("MNEMONIC_PROJECT", "laymcp-probe")
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
}

// TestLayerTools covers 02.2 (Scenarios: mem-layers-registered-005-stable,
// bad-layer-args-rejected). mem_layers is registered, the tool surface grows
// additively 77→78, every 005 mem_* tool keeps its name + required params
// unchanged, and bad layer args are rejected with a clear error.
func TestLayerTools(t *testing.T) {
	memLayerFixture(t)

	tools := NewServer().ListTools()

	// The new layers tool is registered.
	if _, ok := tools["mem_layers"]; !ok {
		t.Errorf("expected tool %q to be registered", "mem_layers")
	}

	// Tool surface grows additively: 77 (75 baseline + 2 governance) + 1 layers + 2 session + 2 status/compact = 82.
	if len(tools) != 83 {
		t.Errorf("expected 83 tools (77 + 1 mem_layers + 2 session + 2 status/compact), got %d", len(tools))
	}

	// Existing 005 mem_* tools keep their names + required params unchanged.
	for name, wantRequired := range expectedMemToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 mem tool %q is no longer registered", name)
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

	// Bad layer args rejected clearly:
	//  - mem_layers with no argument → validation error (no layers invented).
	res, err := handleMemLayers(context.Background(), newCallTool("mem_layers", map[string]any{}))
	if err != nil {
		t.Fatalf("handleMemLayers dispatch: %v", err)
	}
	if !res.IsError {
		t.Errorf("mem_layers without an argument should be a validation error, got: %s", callResultText(t, res))
	}
	//  - mem_layers with an unknown argument → validation error.
	res, err = handleMemLayers(context.Background(), newCallTool("mem_layers", map[string]any{
		"wrong_arg": "value",
	}))
	if err != nil {
		t.Fatalf("handleMemLayers dispatch (bad arg): %v", err)
	}
	if !res.IsError {
		t.Errorf("mem_layers with an unknown argument should be a validation error, got: %s", callResultText(t, res))
	}
}
