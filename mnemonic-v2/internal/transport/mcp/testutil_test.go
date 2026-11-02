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
// The store's transaction hook is wired here (the write-queue test's
// load-bearing assertion — the committed-transaction order, fix round 1
// IMPORTANT #3): every committed store transaction appends its sequence to
// the Server's txn log.
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

// newTestHandlerNoDetector builds a Handler with NO detector (the fixture
// for mem_current_project's no-detector error path: the dispatch must
// return an envelope error, never a placeholder project name).
func newTestHandlerNoDetector(t *testing.T) *handler.Handler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStore(t, t.TempDir())
	return handler.NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, nil)
}

// distinctProject is the detector fixture's resolution for
// TestCurrentProjectReturnsDetectorResolution — deliberately NOT testProject
// ("p", the pre-fix hardcoded literal) so a regression to the literal fails
// the test (fix round 3: the round-1 fixture resolved to testProject, the
// same value the broken code returned, so the test passed against both).
const distinctProject = "fixture-project"

// newTestHandlerDistinctProject builds a Handler whose detector resolves to
// distinctProject (only this test's fixture uses it — the shared
// testProject is untouched, the other tests depend on it).
func newTestHandlerDistinctProject(t *testing.T) *handler.Handler {
	t.Helper()
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	st := openTestStore(t, t.TempDir())
	det := &detectorStub{resolve: func(cwd, explicit string) (string, error) {
		if explicit != "" {
			return explicit, nil
		}
		return distinctProject, nil
	}}
	return handler.NewHandler(memory.NewService(st, c, memory.NewEventBus(), nil, nil), c, det)
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

// rawQuery runs a SELECT through the store's RawQuery seam (the test
// fixture for row-level assertions).
func rawQuery(st store.Store, query string, args ...any) ([][]any, error) {
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("store %T has no RawQuery seam", st)
	}
	return rq.RawQuery(query, args...)
}

// rawExec runs a write through the store's RawExec seam (the test fixture
// for seed/patch row-level assertions).
func rawExec(st store.Store, query string, args ...any) error {
	re, ok := st.(interface {
		RawExec(query string, args ...any) error
	})
	if !ok {
		return fmt.Errorf("store %T has no RawExec seam", st)
	}
	return re.RawExec(query, args...)
}

// firstInt reads a single-integer probe row (COUNT(*) results).
func firstInt(rows [][]any) int64 {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return 0
	}
	switch n := rows[0][0].(type) {
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}

// firstString reads one column of a probe row.
func firstString(rows [][]any, i int) string {
	if len(rows) == 0 || len(rows[0]) <= i {
		return ""
	}
	s, _ := rows[0][i].(string)
	return s
}
