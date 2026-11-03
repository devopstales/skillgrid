package mcp

import "testing"

func TestHarnessFromClientName(t *testing.T) {
	cases := map[string]string{
		"cursor-vscode": "cursor",
		"Cursor":        "cursor",
		"opencode":      "opencode",
		"kilo-code":     "kilo",
		"claude-code":   "claude-code",
		"":              "",
	}
	for in, want := range cases {
		if got := harnessFromClientName(in); got != want {
			t.Errorf("harnessFromClientName(%q) = %q, want %q", in, got, want)
		}
	}
}
