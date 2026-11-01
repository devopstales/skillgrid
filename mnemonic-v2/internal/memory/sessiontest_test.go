package memory

import (
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// timeT is a test-local time value (the tests parse store timestamps without
// importing the store's internal format).
type timeT = time.Time

// parseStoreTime parses the store's "2006-01-02 15:04:05" UTC layout.
func parseStoreTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05", s)
}

// markSessionEnded sets ended_at on a session row (test seam for the
// session-policy tests; Task 7's SessionEnd is the production path).
func markSessionEnded(t *testing.T, st store.Store, sessionID string) {
	t.Helper()
	raw, ok := st.(interface{ rawExec(q string, args ...any) error })
	if !ok {
		t.Fatal("store lacks rawExec test seam")
	}
	if err := raw.rawExec(`UPDATE sessions SET ended_at = datetime('now') WHERE id = ?`, sessionID); err != nil {
		t.Fatalf("mark ended: %v", err)
	}
}

// setSessionOwnedProject switches a session to project_owned mode pinned to
// project (test seam for the ownership check).
func setSessionOwnedProject(t *testing.T, st store.Store, sessionID, project string) {
	t.Helper()
	raw, ok := st.(interface{ rawExec(q string, args ...any) error })
	if !ok {
		t.Fatal("store lacks rawExec test seam")
	}
	if err := raw.rawExec(`UPDATE sessions SET ownership_mode = 'project_owned', project = ? WHERE id = ?`, project, sessionID); err != nil {
		t.Fatalf("set owned project: %v", err)
	}
}
