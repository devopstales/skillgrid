package store

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestMain handles the re-exec'd worker branch BEFORE any test runs.
// The worker is a second OS process: when the test binary is invoked
// with MNEMONIC_CONCURRENT_WORKER=1, TestMain calls runConcurrentWorker
// and exits — no test body executes in the worker.
func TestMain(m *testing.M) {
	if os.Getenv("MNEMONIC_CONCURRENT_WORKER") != "1" {
		os.Exit(m.Run())
	}
	for _, a := range os.Args {
		if a != os.Args[0] && !strings.HasPrefix(a, "-") {
			runConcurrentWorker(a)
			return
		}
	}
	fmt.Fprintln(os.Stderr, "worker: missing dataDir argument")
	os.Exit(2)
}

// concurrentWorkerSaves is the scenario's save count: the second process
// performs exactly this many saves against the shared project file
// (acceptance.feature, scenario concurrent-process-wal).
const concurrentWorkerSaves = 20

// walTestBusyTimeoutMS overrides the DSN's busy_timeout (5000) for the
// two-process WAL tests. The default absorbs the contention: both writers'
// BEGIN IMMEDIATE waits inside one 5 s budget, so a healthy store never
// reaches the retry wrapper. The override exists so a test can SHORTEN
// that budget (TestConcurrentProcessWALBusyExhaustion) — when
// busy_timeout is too small to absorb the hold, the busy error is the
// store's contract to retry (Task 1's withWriteRetry), not the test's
// problem.
var walTestBusyTimeoutMS = 5000

// TestConcurrentProcessWAL is the two-process WAL backstop
// (acceptance.feature, scenario concurrent-process-wal; gate G16).
//
// Given one process holding the store open with a live session;
// When a second process performs twenty saves against the same project
// file; Then all twenty saves succeed and the store contains all twenty
// rows.
//
// Two REAL OS processes: the test binary re-execs itself with
// MNEMONIC_CONCURRENT_WORKER=1 (the standard testing re-exec idiom — the
// test binary is already a linked, race-aware executable, so the worker
// is a genuine second process on the same file, no `go run` involved).
// The worker opens the same project file through store.Open (the real
// WAL DSN: _txlock=immediate + busy_timeout(5000) + journal_mode(WAL) +
// the withWriteRetry backoffs) and runs 20 saves through the real save
// path (AddObservation), printing one "ok" line per save to stderr and
// exiting non-zero on the first failure.
//
// The parent (this process) also opens the same file with a live
// session row and writes its own rows interleaved with the worker's
// run. After the worker reports 20/20, the parent asserts the store
// holds every worker row and every parent row, and the file pings
// clean — no lost write, no corruption on either side (the worker's
// exit code is the worker-side proof).
//
// The guarantee here is that two real OS processes share the file;
// interleaving is NOT enforced (the parent can write all 10 rows after
// the worker finished its 20 — still a two-process test, possibly
// serialized), and the assertions are order-independent on counts.
func TestConcurrentProcessWAL(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	const project = "concurrent-wal"
	const parentSession = "ses-parent"
	const workerSession = "ses-worker"

	// The parent opens the store first and registers its live session
	// row ("one process holding the store open with a live session").
	st, err := Open(ctx, dataDir, project, SQLite, "")
	if err != nil {
		t.Fatalf("open parent store: %v", err)
	}
	defer st.Close()
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: parentSession, Type: "decision",
		Title:   "Parent live row 0",
		Content: "parent row zero, holds the store open",
		Project: project,
	}); err != nil {
		t.Fatalf("parent row 0: %v", err)
	}

	// Build the worker from the test binary itself (os.Executable is the
	// compiled test binary; the MNEMONIC_CONCURRENT_WORKER env flag makes
	// the re-exec'd binary run runConcurrentWorker in TestMain instead of
	// any test body — a genuine second OS process, not a goroutine).
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^$", dataDir)
	cmd.Env = append(os.Environ(), "MNEMONIC_CONCURRENT_WORKER=1")
	cmd.Dir = filepath.Dir(exe)
	var stderr strings.Builder
	cmd.Stderr = &stderr // the worker reports on stderr (stdout is the test framework's)

	// Start the worker (process 2) while the parent keeps writing (the
	// scenario's concurrent window: the parent's own rows interleave
	// with the worker's 20).
	if err := cmd.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	// Kill the worker if the parent fails before reaping it (a t.Fatal in
	// the loop below would otherwise leak the child holding the file).
	finished := false
	t.Cleanup(func() {
		if !finished {
			_ = cmd.Process.Kill()
		}
	})
	for i := 1; i <= 10; i++ {
		if _, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: parentSession, Type: "decision",
			Title:   fmt.Sprintf("Parent live row %d", i),
			Content: fmt.Sprintf("parent row %d written while the worker runs", i),
			Project: project,
		}); err != nil {
			t.Fatalf("parent row %d: %v", i, err)
		}
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	// A healthy worker finishes 20 small saves in well under a minute;
	// the 120 s cap is the deadlock detector (a stuck writer surfaces
	// as SQLITE_BUSY exhaustion, not a hang — but a hang is the other
	// failure mode this backstop exists to catch).
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker process failed: %v\nworker stderr:\n%s", err, stderr.String())
		}
		finished = true
	case <-time.After(120 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		finished = true
		t.Fatalf("worker did not finish within 120 s\nworker stderr:\n%s", stderr.String())
	}

	// Worker-side proof: it reported 20/20 successes (each line is
	// "ok <i>/<total>"); any failure made it exit non-zero and the test
	// already failed.
	okLines := 0
	for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		if strings.HasPrefix(line, "ok ") {
			okLines++
		}
	}
	if okLines != concurrentWorkerSaves {
		t.Fatalf("worker reported %d/%d ok saves, want %d\nworker stderr:\n%s",
			okLines, concurrentWorkerSaves, concurrentWorkerSaves, stderr.String())
	}

	// Store-side proof: every worker row (by session — the 20 scenario
	// saves plus the session-registration row) and every parent row are
	// present — no lost write, no corruption on the reader side.
	workerRows := countObservationsBySession(t, st, workerSession)
	if workerRows != concurrentWorkerSaves+1 {
		t.Errorf("store holds %d worker rows, want %d (20 saves + session row)", workerRows, concurrentWorkerSaves+1)
	}
	parentRowsAll := countObservationsBySession(t, st, parentSession)
	if parentRowsAll != 11 { // row 0 + 10 interleaved
		t.Errorf("store holds %d parent rows, want 11", parentRowsAll)
	}

	// Integrity check: the file is a healthy database after both writers
	// are done (corruption that survives the writes shows up here).
	if err := st.Ping(ctx); err != nil {
		t.Fatalf("post-run ping: %v", err)
	}
}

// TestConcurrentProcessWALBusyExhaustion is the retry wrapper's two-process
// backstop (the scenario's RED half: a busy error that busy_timeout alone
// cannot absorb must still be absorbed by Task 1's withWriteRetry). It is
// TestConcurrentProcessWAL's variant with a different input: the DSN's
// busy_timeout is shortened (10 ms via MNEMONIC_WAL_BUSY_TIMEOUT_MS, which
// runConcurrentWorker mirrors into the test package's walTestBusyTimeoutMS)
// so the parent's held write lock makes the worker's BEGIN IMMEDIATE hit
// SQLITE_BUSY inside its budget. The parent's 750 ms hold exceeds the full
// backoff budget (10+25+50+100+200 ms ≈ 385 ms), so a save that starts
// inside the hold window fails when the retry wrapper is absent (the test's
// RED) — with it, the wrapper's retries are what carry the save through.
// Same two real OS processes, same file, different budget.
func TestConcurrentProcessWALBusyExhaustion(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()
	const project = "concurrent-wal"
	const parentSession = "ses-parent"
	const workerSession = "ses-worker"

	// The parent (this process) writes with the shortened budget too —
	// both sides of the scenario share the DSN, and the worker's hold is
	// what must be absorbed, so the parent's budget is not what is
	// exercised here (it does one write and then reads).
	prev := walTestBusyTimeoutMS
	walTestBusyTimeoutMS = 10
	t.Cleanup(func() { walTestBusyTimeoutMS = prev })

	st, err := Open(ctx, dataDir, project, SQLite, "")
	if err != nil {
		t.Fatalf("open parent store: %v", err)
	}
	defer st.Close()
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: parentSession, Type: "decision",
		Title:   "Parent hold row",
		Content: "parent holds the write lock for the worker's window",
		Project: project,
	}); err != nil {
		t.Fatalf("parent row: %v", err)
	}

	// The parent's hold is what the worker must ride out: start it
	// BEFORE the worker so the worker's session-registration save and
	// its first saves all meet the held write lock (a 10 ms
	// busy_timeout cannot wait them out — withWriteRetry's backoffs
	// can). The hold's 750 ms exceeds the full backoff budget
	// (10+25+50+100+200 ms ≈ 385 ms), so a save that starts inside the
	// hold window fails when the retry wrapper is absent (the test's
	// RED) and the wrapper's effect is the variable under test.
	holdDone, err := rawSQLiteHold(ctx, st, 750*time.Millisecond)
	if err != nil {
		t.Fatalf("parent hold: %v", err)
	}
	defer func() { <-holdDone }()

	// Worker: the same 20-save scenario, but its busy_timeout is 10 ms —
	// a hold the 5 s default absorbs is, to it, a retryable busy error.
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	cmd := exec.Command(exe, "-test.run=^$", dataDir)
	cmd.Env = append(os.Environ(), "MNEMONIC_CONCURRENT_WORKER=1", "MNEMONIC_WAL_BUSY_TIMEOUT_MS=10")
	cmd.Dir = filepath.Dir(exe)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	finished := false
	t.Cleanup(func() {
		if !finished {
			_ = cmd.Process.Kill()
		}
	})
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("worker process failed: %v\nworker stderr:\n%s", err, stderr.String())
		}
		finished = true
	case <-time.After(120 * time.Second):
		_ = cmd.Process.Kill()
		<-done
		finished = true
		t.Fatalf("worker did not finish within 120 s\nworker stderr:\n%s", stderr.String())
	}

	okLines := 0
	for _, line := range strings.Split(strings.TrimSpace(stderr.String()), "\n") {
		if strings.HasPrefix(line, "ok ") {
			okLines++
		}
	}
	if okLines != concurrentWorkerSaves {
		t.Fatalf("worker reported %d/%d ok saves, want %d\nworker stderr:\n%s",
			okLines, concurrentWorkerSaves, concurrentWorkerSaves, stderr.String())
	}
	if n := countObservationsBySession(t, st, workerSession); n != concurrentWorkerSaves+1 {
		t.Errorf("store holds %d worker rows, want %d (20 saves + session row)", n, concurrentWorkerSaves+1)
	}
	if err := st.Ping(ctx); err != nil {
		t.Fatalf("post-run ping: %v", err)
	}
}

// rawSQLiteHold holds the store's write lock for d and returns when the
// hold has started (a BEGIN IMMEDIATE transaction committed after a Go
// sleep — same-package access to the test's *sqliteStore; the Store
// interface exposes no raw-transaction surface, and the single-connection
// pool means this IS the store's writer connection). While it runs, a
// cross-process BEGIN IMMEDIATE from another connection gets SQLITE_BUSY
// after its busy_timeout. The sleep is inside the transaction goroutine
// (SQL-side sleep — sqlite3_sleep — is unavailable in the embedded
// build); the returned channel closes when the hold's transaction
// commits, so the test can both overlap the hold with the worker's saves
// and join before asserting.
func rawSQLiteHold(ctx context.Context, st Store, d time.Duration) (<-chan struct{}, error) {
	sq, ok := st.(*sqliteStore)
	if !ok {
		return nil, fmt.Errorf("rawSQLiteHold: store %T is not *sqliteStore", st)
	}
	started := make(chan struct{})
	done := make(chan struct{})
	go func() {
		tx, err := sq.db.BeginTx(ctx, nil) // _txlock=immediate → BEGIN IMMEDIATE
		if err != nil {
			close(started)
			close(done)
			return
		}
		close(started)
		time.Sleep(d)
		_ = tx.Commit()
		close(done)
	}()
	<-started
	return done, nil
}

// countObservationsBySession returns the live (non-deleted) observation
// count for one session (the raw-query test seam — the same seam the
// memory package's fixtures use).
func countObservationsBySession(t *testing.T, st Store, sessionID string) int {
	t.Helper()
	raw, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		t.Fatal("store lacks the raw-query test seam (expected *store.sqliteStore)")
	}
	rows, err := raw.RawQuery(
		`SELECT COUNT(*) FROM observations WHERE session_id = ? AND deleted_at IS NULL`,
		sessionID)
	if err != nil {
		t.Fatalf("count %s rows: %v", sessionID, err)
	}
	switch v := rows[0][0].(type) {
	case int64:
		return int(v)
	case int:
		return v
	default:
		t.Fatalf("count %s rows: unexpected scan type %T", sessionID, rows[0][0])
	}
	return 0
}

// runConcurrentWorker is the re-exec'd branch (the second OS process).
// It runs in the worker, not a test: it must not call t.Fatal or other
// testing helpers (there is no live *testing.T for this process's own
// execution — the parent owns the assertions). It exits non-zero on the
// first save failure and prints one "ok" line per successful save to
// stderr (the worker's stdout is captured by the parent as the test
// framework's PASS/FAIL line; stderr is the worker's own report
// channel).
func runConcurrentWorker(dataDir string) {
	ctx := context.Background()
	const project = "concurrent-wal"
	const session = "ses-worker"

	// The worker reuses the test package's walTestBusyTimeoutMS (the DSN
	// busy_timeout it applies is the parent's current value: the
	// exhaustion variant shortens it so a hold the 5 s default absorbs
	// becomes a busy error — the retry wrapper's two-process job).
	busy := walTestBusyTimeoutMS
	if v := os.Getenv("MNEMONIC_WAL_BUSY_TIMEOUT_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			busy = n
		}
	}
	prev := walTestBusyTimeoutMS
	walTestBusyTimeoutMS = busy
	defer func() { walTestBusyTimeoutMS = prev }()

	st, err := Open(ctx, dataDir, project, SQLite, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "worker open: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()
	// Register the worker's live session row up front (its own process
	// is the "second process" of the scenario; the row's session_id is
	// the worker's live session for the duration of the run).
	if _, err := st.AddObservation(ctx, AddObservationParams{
		SessionID: session, Type: "decision",
		Title: "Worker live row", Content: "worker session registration row",
		Project: project,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "worker session registration: %v\n", err)
		os.Exit(1)
	}
	for i := 1; i <= concurrentWorkerSaves; i++ {
		_, err := st.AddObservation(ctx, AddObservationParams{
			SessionID: session, Type: "decision",
			Title:   fmt.Sprintf("Worker save %d", i),
			Content: fmt.Sprintf("worker save %d of %d", i, concurrentWorkerSaves),
			Project: project,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "worker save %d/%d failed: %v\n", i, concurrentWorkerSaves, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "ok %d/%d\n", i, concurrentWorkerSaves)
	}
	fmt.Fprintln(os.Stderr, "done")
}
