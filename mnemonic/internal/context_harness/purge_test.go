package context_harness

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
)

// purgeCount returns the tool_outputs row count for a session/project.
func purgeCount(t *testing.T, st *store.Store, sessionID, projectID string) int {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(
		`SELECT count(*) FROM tool_outputs WHERE session_id = ? AND project_id = ?`,
		sessionID, projectID,
	).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

// purgeSeed inserts one sandbox row for a session/project/tool.
func purgeSeed(t *testing.T, st *store.Store, sessionID, projectID, toolName string) {
	t.Helper()
	if _, err := st.DB.Exec(
		`INSERT INTO tool_outputs (session_id, project_id, tool_name, output, size_bytes, created_at)
		 VALUES (?, ?, ?, 'payload', 7, datetime('now'))`,
		sessionID, projectID, toolName,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}
}

// TestPurgeSession_ClearsToolOutputs purges the target session's sandbox rows
// and returns the deleted count, leaving other sessions untouched.
func TestPurgeSession_ClearsToolOutputs(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	purgeSeed(t, st, "s1", "ctxproj", "bash")
	purgeSeed(t, st, "s1", "ctxproj", "web_search_exa")
	purgeSeed(t, st, "s2", "ctxproj", "bash")

	n, err := PurgeSession(st.DB, "ctxproj", "s1")
	if err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 deleted, got %d", n)
	}
	if got := purgeCount(t, st, "s1", "ctxproj"); got != 0 {
		t.Fatalf("expected 0 rows for s1, got %d", got)
	}
	if got := purgeCount(t, st, "s2", "ctxproj"); got != 1 {
		t.Fatalf("expected 1 row for s2 to remain, got %d", got)
	}
}

// TestPurgeSession_FailOpenNoRevisions confirms purge does not error when the
// context_revisions table (migration 055, a later PR) is absent, and still
// purges tool_outputs.
func TestPurgeSession_FailOpenNoRevisions(t *testing.T) {
	st, err := store.Open(t.TempDir(), "ctxproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	// Fail-open precondition: context_revisions must NOT exist yet in PR 2.
	var present int
	if err := st.DB.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name = 'context_revisions'`,
	).Scan(&present); err != nil {
		t.Fatalf("sqlite_master: %v", err)
	}
	if present != 0 {
		t.Fatalf("precondition failed: context_revisions exists in PR 2 store")
	}

	purgeSeed(t, st, "s1", "ctxproj", "bash")

	n, err := PurgeSession(st.DB, "ctxproj", "s1")
	if err != nil {
		t.Fatalf("purge should be fail-open with no context_revisions, got: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 deleted, got %d", n)
	}
	if got := purgeCount(t, st, "s1", "ctxproj"); got != 0 {
		t.Fatalf("expected 0 rows for s1, got %d", got)
	}
}
