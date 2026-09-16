package mcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory/layer"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// distillableSummary is L0 content with deterministic Key Learnings so the
// no-LLM floor produces L1 atoms when the session-close hook fires.
const distillableSummary = `## Goal
Tune the auth token rotation.

## Key Learnings:

1. JWT refresh tokens need atomic rotation to avoid races
2. bcrypt cost=12 is the right balance for our server
3. FTS5 queries must be sanitized before MATCH
`

// newSessionFixture sets up a service + session for the real close-handler
// tests. It returns the dataDir (to reopen the store for assertions), the
// service (to arm distillation / read layers), and the session id (the raw L0).
func newSessionFixture(t *testing.T, project, sessionID string) (string, *service.Service, string) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, '/tmp', ?, 'active')`, sessionID, project, now); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	if _, err := st.DB.Exec(`UPDATE sessions SET summary = ? WHERE id = ? AND project = ?`,
		distillableSummary, sessionID, project); err != nil {
		t.Fatalf("set summary: %v", err)
	}
	st.Close()

	t.Setenv("MNEMONIC_PROJECT", project)
	svc := service.New(dataDir)
	SetService(svc)
	t.Cleanup(func() { SetService(nil) })
	return dataDir, svc, sessionID
}

// countLayers returns the number of layer rows for a session.
func countLayers(t *testing.T, dataDir, project, sessionID string) int {
	t.Helper()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer st.Close()
	var n int
	if err := st.DB.QueryRow(`
		SELECT COUNT(*) FROM observation_layers
		WHERE project = ? AND source_session = ?`, project, sessionID).Scan(&n); err != nil {
		t.Fatalf("count layers: %v", err)
	}
	return n
}

// waitForHook polls until the detached distill goroutine has run (fired > 0 and
// reached a terminal result: either Created or a captured best-effort Err).
func waitForHook(t *testing.T, svc *service.Service) bool {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if svc.DistillHookFired() > 0 {
			last := svc.DistillLastResult()
			if last.Created || last.Err != nil {
				return true
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// TestSessionEndDistillHookWired is the end-to-end wiring assertion the reviewer
// flagged as missing: a session close THROUGH THE REAL CLOSE HANDLER
// (handleMemSessionEnd → memory.SessionEnd) with distillation ENABLED actually
// fires the detached distill goroutine, session close returns success, and the
// hook is best-effort (a distill that errors does not break close).
func TestSessionEndDistillHookWired(t *testing.T) {
	dataDir, svc, sessionID := newSessionFixture(t, "sessend-on", "sess-end-on")
	if err := svc.EnableDistill(); err != nil {
		t.Fatalf("enable distill: %v", err)
	}

	// Close through the REAL handler, carrying the L0 summary (as a session
	// close does). The distill hook receives it as the L0 text.
	res, err := handleMemSessionEnd(context.Background(), newCallTool("mem_session_end", map[string]any{
		"session_id": sessionID,
		"summary":    distillableSummary,
	}))
	if err != nil {
		t.Fatalf("handleMemSessionEnd dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("session close must return success, got error: %s", callResultText(t, res))
	}

	// Wait for the detached goroutine to finish the distill pass.
	if !waitForHook(t, svc) {
		t.Fatalf("distill hook did not fire on session close (hookFired=%d)", svc.DistillHookFired())
	}
	last := svc.DistillLastResult()
	if last.Err != nil {
		t.Fatalf("distill hook errored unexpectedly (best-effort should swallow, but it blocked): %v", last.Err)
	}
	if !last.Created {
		t.Fatalf("distill hook ran but did not create layers: %+v", last)
	}
	// Final confirmation: the L1 atoms (and scenario/persona) are on disk.
	if n := countLayers(t, dataDir, "sessend-on", sessionID); n == 0 {
		t.Fatalf("distill hook created layers (result says so) but none are on disk")
	}
}

// TestSessionEndDistillHookBestEffort covers the finding's best-effort
// requirement: a session close THROUGH THE REAL CLOSE HANDLER with distillation
// ENABLED returns success even when the detached distill goroutine ERRORS. The
// error is captured in DistillResult.Err (surfaced, not propagated) — the hook
// is best-effort and never breaks session close. A test-only hook (via
// SetDistillHookForTest) injects a distill error into the detached goroutine so
// the test can prove the error is surfaced yet never breaks the close.
func TestSessionEndDistillHookBestEffort(t *testing.T) {
	_, svc, sessionID := newSessionFixture(t, "sessend-bbest", "sess-bbest")
	if err := svc.EnableDistill(); err != nil {
		t.Fatalf("enable distill: %v", err)
	}
	// Inject a hook that records the opt-in firing (DistillHookFired) and
	// captures a distill error (DistillResult.Err) — simulating a distill pass
	// that failed while session close must still succeed.
	fired := 0
	svc.SetDistillHookForTest(func(ctx context.Context, projectID, sessionID, summary string) {
		fired++
		svc.RecordDistillTestResult(layer.DistillResult{Err: errors.New("injected best-effort distill error")})
	})

	// Close through the REAL handler. The detached goroutine runs the injected
	// hook (which errors) — but the close itself must still return success.
	res, err := handleMemSessionEnd(context.Background(), newCallTool("mem_session_end", map[string]any{
		"session_id": sessionID,
		"summary":    distillableSummary,
	}))
	if err != nil {
		t.Fatalf("handleMemSessionEnd dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("session close must return success even when distill errors, got: %s", callResultText(t, res))
	}

	// The detached goroutine must have run the injected hook and captured the
	// distill error.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && svc.DistillLastResult().Err == nil {
		time.Sleep(25 * time.Millisecond)
	}
	if fired == 0 {
		t.Fatalf("distill hook did not fire on session close (hookFired=0)")
	}
	last := svc.DistillLastResult()
	if last.Err == nil {
		t.Fatalf("expected the distill to error, but the hook captured no error: %+v", last)
	}
	if last.Created {
		t.Fatalf("a distill error must not be reported as created: %+v", last)
	}
}

// TestSessionEndDistillHookDisabled covers the opt-in default: with
// distillation NOT enabled, a real session close does NOT fire the hook (no
// layers), and close still succeeds.
func TestSessionEndDistillHookDisabled(t *testing.T) {
	dataDir, _, sessionID := newSessionFixture(t, "sessend-off", "sess-end-off")

	res, err := handleMemSessionEnd(context.Background(), newCallTool("mem_session_end", map[string]any{
		"session_id": sessionID,
	}))
	if err != nil {
		t.Fatalf("handleMemSessionEnd dispatch: %v", err)
	}
	if res.IsError {
		t.Fatalf("session close must return success, got error: %s", callResultText(t, res))
	}
	// Give a (disabled) hook any chance to run, then assert nothing was created.
	time.Sleep(150 * time.Millisecond)
	if n := countLayers(t, dataDir, "sessend-off", sessionID); n != 0 {
		t.Fatalf("disabled distillation must not fire the hook, got %d layer rows", n)
	}
}
