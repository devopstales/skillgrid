package memory

import (
	"os"
	"testing"
)

// TestMain makes the memory test binary hermetic: it forces the embedder seam
// off so any code-index path in these tests never reaches a live Ollama server.
//
// The sequential /api/embeddings calls are the load that, under the full
// parallel suite, stretched a detached async goroutine past a t.TempDir cleanup
// and produced a rare "directory not empty" RemoveAll race (a late write into a
// temp dir the cleanup was already removing). Removing the Ollama load makes
// the async work finish promptly, so the race window closes.
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
