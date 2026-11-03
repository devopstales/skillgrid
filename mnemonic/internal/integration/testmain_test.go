package integration

import (
	"os"
	"testing"
)

// TestMain forces the embedder seam off so the seed/index integration tests
// never reach a live Ollama server. RunCodeIndex → embedPass issues sequential
// /api/embeddings calls that would contend across parallel packages and trip
// the per-package timeout. The seed tests assert on indexed rows (files/
// symbols/observations), not vectors, so the Null embedder is a no-op for them.
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
