package relay

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// openStore opens a fresh project store (applies all migrations incl. 019) in
// a temp data dir and returns its *sql.DB plus a temp project root for the
// cleave bundle.
func openStore(t *testing.T, projectID string) (*store.Store, string) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, projectID)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	root := t.TempDir()
	return st, root
}

// seedSession inserts a minimal sessions row so a handoff's source_session
// satisfies the FK (session_handoffs.source_session REFERENCES sessions(id)
// ON DELETE SET NULL).
func seedSession(t *testing.T, st *store.Store, id string) {
	t.Helper()
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'relayproj', '/tmp', '2026-01-01T00:00:00Z', 'active')`, id); err != nil {
		t.Fatalf("seed session: %v", err)
	}
}

func handoffRowCount(t *testing.T, st *store.Store, projectID string) int {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_handoffs WHERE project = ?`, projectID).Scan(&n); err != nil {
		t.Fatalf("count handoffs: %v", err)
	}
	return n
}

// TestHandoff covers the happy path (Scenario: Handoff writes cleave bundle and
// row): a successful handoff writes the three .cleave/ files AND a
// session_handoffs row, and ReadBundle round-trips the content.
func TestHandoff(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s1")

	hid, paths, err := Handoff(ctx, st.DB, "relayproj", "", root, Bundle{
		Progress:      "implemented relay",
		Knowledge:     "fail closed first",
		NextPrompt:    "resume the verify phase",
		SourceSession: "s1",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	if hid == "" {
		t.Fatalf("handoff must return a non-empty handoff_id")
	}
	if len(paths) != 3 {
		t.Fatalf("handoff must return 3 paths, got %d", len(paths))
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected cleave file on disk: %v", err)
		}
	}
	// The row is written (fail-closed: files present → row present).
	if n := handoffRowCount(t, st, "relayproj"); n != 1 {
		t.Fatalf("expected 1 session_handoffs row, got %d", n)
	}
	// Round-trip the bundle.
	b, err := ReadBundle(root)
	if err != nil {
		t.Fatalf("read bundle: %v", err)
	}
	if b.NextPrompt != "resume the verify phase" {
		t.Errorf("NextPrompt = %q, want the handoff value", b.NextPrompt)
	}
}

// TestHandoffFailClosedNoOrphan asserts the load-bearing invariant: if the
// cleave file write fails, NO session_handoffs row is written (no orphan row
// without files). A read-only cleave dir forces the file write to fail.
func TestHandoffFailClosedNoOrphan(t *testing.T) {
	st, root := openStore(t, "relayproj")
	// Make the .skillgrid dir read-only so MkdirAll/.cleave fails.
	sg := filepath.Join(root, ".skillgrid")
	if err := os.MkdirAll(sg, 0o555); err != nil {
		t.Fatalf("mkdir .skillgrid: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(sg, 0o755) })

	_, _, err := Handoff(context.Background(), st.DB, "relayproj", "", root, Bundle{
		Progress:   "p",
		NextPrompt: "np",
	})
	if err == nil {
		t.Fatalf("expected handoff to fail on read-only cleave dir")
	}
	// Fail closed: no orphan row.
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 session_handoffs rows (fail closed), got %d", n)
	}
}

// TestHandoffRequiresNextPrompt asserts a handoff with no NEXT_PROMPT fails
// before any file or row is written (resume must not invent a prompt).
func TestHandoffRequiresNextPrompt(t *testing.T) {
	st, root := openStore(t, "relayproj")
	_, _, err := Handoff(context.Background(), st.DB, "relayproj", "", root, Bundle{
		Progress: "only progress, no prompt",
	})
	if err == nil {
		t.Fatalf("expected handoff to fail with empty next_prompt")
	}
	if n := handoffRowCount(t, st, "relayproj"); n != 0 {
		t.Fatalf("expected 0 rows when next_prompt is empty, got %d", n)
	}
}

// TestResume covers the happy path (Scenario: Handoff writes cleave bundle and
// row → resume returns the NEXT_PROMPT): a resume for a known handoff returns
// the stored NEXT_PROMPT. With archive=true it records a session_archives row
// and flips the handoff to archived.
func TestResume(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s1")

	hid, _, err := Handoff(ctx, st.DB, "relayproj", "ho-1", root, Bundle{
		Progress:      "p",
		NextPrompt:    "the resume prompt text",
		SourceSession: "s1",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}

	prompt, id, archiveID, err := Resume(ctx, st.DB, "relayproj", hid, root, true)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if prompt != "the resume prompt text" {
		t.Errorf("prompt = %q, want the stored NEXT_PROMPT", prompt)
	}
	if id != "ho-1" {
		t.Errorf("resume handoff_id = %q, want ho-1", id)
	}
	if archiveID <= 0 {
		t.Errorf("archive=true must return an archive_id, got %d", archiveID)
	}
	// Archive row recorded.
	var arc int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_archives WHERE handoff_id = ?`, hid).Scan(&arc); err != nil {
		t.Fatalf("count archive: %v", err)
	}
	if arc != 1 {
		t.Errorf("expected 1 session_archives row, got %d", arc)
	}
	// Handoff flipped to archived.
	var status string
	if err := st.DB.QueryRow(`SELECT status FROM session_handoffs WHERE handoff_id = ?`, hid).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "archived" {
		t.Errorf("handoff status = %q, want archived", status)
	}
}

// TestResumeMissing covers the edge (Scenario: Missing cleave or unknown
// handoff id): a known handoff id whose .cleave/ bundle has been removed
// errors clearly and does NOT invent a prompt.
func TestResumeMissing(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s-missing")

	hid, _, err := Handoff(ctx, st.DB, "relayproj", "ho-missing", root, Bundle{
		Progress:      "p",
		NextPrompt:    "will be deleted",
		SourceSession: "s-missing",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}
	// Delete the cleave bundle so resume must fail closed.
	if err := os.RemoveAll(filepath.Join(root, ".skillgrid", ".cleave")); err != nil {
		t.Fatalf("remove cleave: %v", err)
	}

	prompt, _, _, err := Resume(ctx, st.DB, "relayproj", hid, root, false)
	if err == nil {
		t.Fatalf("expected resume to fail on missing .cleave/ bundle")
	}
	if prompt != "" {
		t.Errorf("resume must not invent a prompt, got %q", prompt)
	}
	if !strings.Contains(err.Error(), "cleave") {
		t.Errorf("error should mention the missing cleave bundle, got: %v", err)
	}
}

// TestResumeArchiveStatusFlipFails asserts the fail-closed status-update fix:
// if the `UPDATE session_handoffs SET status='archived'` fails after the
// session_archives row is written, Resume surfaces an error (no swallowed
// `_, _ =`), so an archive row is never left with a still-pending handoff
// silently.
func TestResumeArchiveStatusFlipFails(t *testing.T) {
	st, root := openStore(t, "relayproj")
	ctx := context.Background()
	seedSession(t, st, "s-flip")

	hid, _, err := Handoff(ctx, st.DB, "relayproj", "ho-flip", root, Bundle{
		Progress:      "p",
		NextPrompt:    "prompt",
		SourceSession: "s-flip",
	})
	if err != nil {
		t.Fatalf("handoff: %v", err)
	}

	// Wrap the store so the UPDATE that flips status fails (the INSERT for the
	// archive row still succeeds).
	db := &failOnUpdate{DB: st.DB, failOn: "UPDATE session_handoffs SET status"}
	_, _, _, rerr := Resume(ctx, db, "relayproj", hid, root, true)
	if rerr == nil {
		t.Fatalf("expected resume(archive=true) to error when the status flip fails")
	}
	if !strings.Contains(rerr.Error(), "failed to mark handoff archived") {
		t.Errorf("error should say the status flip failed, got: %v", rerr)
	}
	// The archive row was written (before the flip), but the handoff is still
	// pending → the inconsistency is surfaced, not hidden.
	var arc int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_archives WHERE handoff_id = ?`, hid).Scan(&arc); err != nil {
		t.Fatalf("count archive: %v", err)
	}
	if arc != 1 {
		t.Errorf("expected the archive row to be written before the flip, got %d", arc)
	}
	var status string
	if err := st.DB.QueryRow(`SELECT status FROM session_handoffs WHERE handoff_id = ?`, hid).Scan(&status); err != nil {
		t.Fatalf("read status: %v", err)
	}
	if status != "pending" {
		t.Errorf("after a failed flip the handoff should still be pending, got %q", status)
	}
}

// failOnUpdate wraps *sql.DB and returns an error from ExecContext for queries
// containing failOn (used to simulate a transient status-flip failure).
type failOnUpdate struct {
	*sql.DB
	failOn string
}

func (f *failOnUpdate) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.Contains(query, f.failOn) {
		return nil, fmt.Errorf("injected status-flip failure")
	}
	return f.DB.ExecContext(ctx, query, args...)
}

// TestResumeUnknown covers the edge (Scenario: Missing cleave or unknown
// handoff id): a resume for a handoff id that was never recorded errors
// clearly (fail closed).
func TestResumeUnknown(t *testing.T) {
	st, root := openStore(t, "relayproj")
	_, _, _, err := Resume(context.Background(), st.DB, "relayproj", "never-existed", root, false)
	if err == nil {
		t.Fatalf("expected resume to fail on unknown handoff id")
	}
	if !strings.Contains(err.Error(), "unknown handoff id") {
		t.Errorf("error should say unknown handoff id, got: %v", err)
	}
}

// TestReadBundleMissing asserts ReadBundle fails closed on an absent bundle
// (the unit-level mirror of the Resume missing-cleave edge).
func TestReadBundleMissing(t *testing.T) {
	root := t.TempDir()
	_, err := ReadBundle(root)
	if err == nil {
		t.Fatalf("expected ReadBundle to fail on absent .cleave/ dir")
	}
}

// TestSoftOptionalL0 asserts the soft-optional 003 L0 path: a handoff works
// with no L0 tree present (degrades gracefully, no error), and mirrors the
// bundle into .skillgrid/workspace/sessions/{id}/ when that tree exists.
func TestSoftOptionalL0(t *testing.T) {
	// No L0 tree: handoff still succeeds.
	st, root := openStore(t, "relayproj")
	if _, err := st.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-l0', 'relayproj', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, _, err := Handoff(context.Background(), st.DB, "relayproj", "l0-none", root, Bundle{
		Progress: "p", NextPrompt: "np", SourceSession: "sess-l0",
	}); err != nil {
		t.Fatalf("handoff without L0 tree must succeed: %v", err)
	}

	// L0 tree present: handoff mirrors the bundle into it.
	st2, root2 := openStore(t, "relayproj2")
	if _, err := st2.DB.Exec(`INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('sess-l0', 'relayproj2', '/tmp', '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	l0Dir := filepath.Join(root2, L0Dir, "sess-l0")
	if err := os.MkdirAll(l0Dir, 0o755); err != nil {
		t.Fatalf("mkdir L0: %v", err)
	}
	if _, _, err := Handoff(context.Background(), st2.DB, "relayproj2", "l0-mirror", root2, Bundle{
		Progress: "p2", NextPrompt: "np2", SourceSession: "sess-l0",
	}); err != nil {
		t.Fatalf("handoff with L0 tree: %v", err)
	}
	for _, name := range []string{FileProgress, FileKnowledge, FileNextPrompt} {
		if _, err := os.Stat(filepath.Join(l0Dir, name)); err != nil {
			t.Errorf("expected L0 mirror of %s: %v", name, err)
		}
	}
}
