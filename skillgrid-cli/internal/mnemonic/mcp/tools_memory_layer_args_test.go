package mcp

import (
	"context"
	"sort"
	"strings"
	"testing"
)

// TestBadLayerArgs covers 02.10 (Scenario: bad-layer-args-rejected). mem_layers
// with a missing or invalid argument is rejected with a clear validation error
// and no layers are invented. This is the edge-case companion to TestLayerTools
// (the registration + 005-stability threat test).
func TestBadLayerArgs(t *testing.T) {
	memLayerFixture(t)

	cases := []struct {
		name string
		args map[string]any
	}{
		{"no-arg", map[string]any{}},
		{"unknown-arg", map[string]any{"wrong_arg": "value"}},
		{"empty-both", map[string]any{"session_id": "  ", "topic_key": "  "}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := handleMemLayers(context.Background(), newCallTool("mem_layers", tc.args))
			if err != nil {
				t.Fatalf("handleMemLayers dispatch (%s): %v", tc.name, err)
			}
			if !res.IsError {
				t.Errorf("%s: expected a validation error, got: %s", tc.name, callResultText(t, res))
			}
			// The error must be clear (named) — not an opaque dispatch failure.
			if got := callResultText(t, res); !strings.Contains(got, "mem_layers") && !strings.Contains(got, "session_id") && !strings.Contains(got, "topic_key") {
				t.Errorf("%s: error should name the missing/invalid arg, got: %s", tc.name, got)
			}
		})
	}
}

// TestMemLayersRegisteredAndStable is the @step-02 acceptance assertion that the
// layered surface is live (mem_layers registered) while every 005 mem_* tool
// keeps its name + required params unchanged. It is the 005-stable half of
// 02.10, kept separate from the RED threat test (TestLayerTools) so the
// additive-contract check is explicit and re-runnable.
func TestMemLayersRegisteredAndStable(t *testing.T) {
	memLayerFixture(t)
	tools := NewServer().ListTools()

	if _, ok := tools["mem_layers"]; !ok {
		t.Fatal("mem_layers is not registered")
	}
	// The additive surface count: 82 (78 step-02 baseline + 2 status/compact).
	if len(tools) != 83 {
		t.Errorf("expected 83 tools, got %d", len(tools))
	}
	for name, wantRequired := range expectedMemToolSurface {
		st, ok := tools[name]
		if !ok {
			t.Errorf("005 mem tool %q no longer registered", name)
			continue
		}
		got := append([]string(nil), st.Tool.InputSchema.Required...)
		want := append([]string(nil), wantRequired...)
		sort.Strings(got)
		sort.Strings(want)
		if len(got) != len(want) {
			t.Errorf("%q: required params changed: got %v want %v", name, got, want)
			continue
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("%q: required param %d changed: got %q want %q", name, i, got[i], want[i])
			}
		}
	}
}
