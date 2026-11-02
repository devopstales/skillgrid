package mcp

// testutil_test.go — shared fixtures for the transport tests: a real
// SQLite-backed handler (the same wiring production uses, minus git) and
// the save-arg factory the queue tests drive.

import (
	"fmt"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// testProject is the project every fixture store opens for (fixed so the
// detector never has to guess — the tests pass it explicitly).
const testProject = "p"

// openTestStore opens a real SQLite store for testProject in dir.
func openTestStore(t *testing.T, dir string) store.Store {
	t.Helper()
	st, err := store.Open(ctx, dir, testProject, store.SQLite, "")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// newTestHandler builds a Handler over a real SQLite store with nil
// intelligence seams (absent seams → the INV-3 deterministic floors). The
// detector is a fixed stub: no git shell-out in tests.
func newTestHandler(t *testing.T) *handler.Handler {
	return newTestHandlerWithStore(t, openTestStore(t, t.TempDir()))
}

// newTestHandlerWithStore builds a Handler over a caller-provided store.
func newTestHandlerWithStore(t *testing.T, st store.Store) *handler.Handler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	det := &detectorStub{resolve: func(cwd, explicit string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		return testProject, nil
	}}
	return handler.NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, det)
}

// newTestHandlerWithNoLLM is newTestHandler with the seams explicitly nil
// (the envelope-degraded fixture: absent seams, never a failing one). The
// service's runDedup returns degraded=true when the LLM is nil (the
// INV-3 floor), so a save is always (at least) degraded.
func newTestHandlerWithNoLLM(t *testing.T) *handler.Handler {
	return newTestHandlerWithStore(t, openTestStore(t, t.TempDir()))
}

// saveArgs builds one valid mem_save arg set (the queue tests' input).
func saveArgs(i int) map[string]any {
	return map[string]any{
		"title":      fmt.Sprintf("decide thing %d", i),
		"type":       "decision",
		"content":    fmt.Sprintf("**What**: thing %d was decided.", i),
		"session_id": "sess-1",
		"project":    testProject,
	}
}
