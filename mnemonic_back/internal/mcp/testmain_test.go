package mcp

import (
	"os"
	"testing"
)

// TestMain forces the embedder seam off so the code-index tests in this package
// (which call RunCodeIndex → embedPass) never reach a live Ollama server. The
// sequential /api/embeddings calls would otherwise contend across the parallel
// test packages and trip the per-package timeout. The code_* MCP tests assert on
// structural graph data (symbols/edges/processes), not vectors, so the Null
// embedder is a no-op for them.
func TestMain(m *testing.M) {
	prev, had := os.LookupEnv("SKILLGRID_MNEMONIC_EMBEDDER")
	os.Setenv("SKILLGRID_MNEMONIC_EMBEDDER", "off")
	code := m.Run()
	if had {
		os.Setenv("SKILLGRID_MNEMONIC_EMBEDDER", prev)
	} else {
		os.Unsetenv("SKILLGRID_MNEMONIC_EMBEDDER")
	}
	os.Exit(code)
}
