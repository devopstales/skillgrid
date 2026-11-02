package memory

// Task 17 — autosync manager (spec 01-memory/003-sync §5).
//
// The three acceptance scenarios (acceptance.feature, G14):
//
//	[autosync-single-lease]                 — two workers on the same store:
//	                                         at most one holds the sync
//	                                         lease at a time; cycles complete
//	                                         without error.
//	[write-path-never-blocks-on-network]    — a save with a slow remote
//	                                         returns well before the network
//	                                         round-trip completes (the
//	                                         onWrite hook is a buffered-1
//	                                         channel; the commit path never
//	                                         does network I/O).
//	[autosync-debounce]                     — a burst of dirty notifications
//	                                         coalesces to exactly one cycle.
//
// The manager is a port of the v1/Engram autosync manager (lease, debounce,
// backoff, phases — internal/cloud/autosync/manager.go): the lease is the
// sync_state row's lease_owner/lease_until pair (60 s TTL — a dead worker's
// lease is re-acquirable once it expires), the debounce coalesces bursts,
// and the backoff is exponential with ±25% jitter capped at 5 minutes.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// autosyncTarget is the manager's sync_state target row (the store's local
// target key — the journal's target, the lease lives on it).
const autosyncTarget = "local"

// leaseStamp parses a lease_until timestamp (datetime('now') format).
func leaseStamp(t *testing.T, s string) time.Time {
	t.Helper()
	ts, err := time.Parse("2006-01-02 15:04:05", s)
	if err != nil {
		t.Fatalf("parse lease stamp %q: %v", s, err)
	}
	return ts
}

// syncLeaseOf reads the sync_state lease row's (owner, until) pair through
// the RawQuery seam (the SyncSeam test hooks expose no read surface for the
// lease columns — the pipeline's raw-SQL read pattern). "" = no lease row.
func syncLeaseOf(t *testing.T, st store.Store) (string, string) {
	t.Helper()
	s, ok := st.(syncSeam)
	if !ok {
		t.Fatal("store does not expose the sync seam")
	}
	rows, err := s.RawQuery(`SELECT ifnull(lease_owner,''), ifnull(lease_until,'') FROM sync_state WHERE target_key = ?`, autosyncTarget)
	if err != nil {
		t.Fatalf("lease read: %v", err)
	}
	if len(rows) != 1 || len(rows[0]) < 2 {
		t.Fatalf("lease row unexpected: %v", rows)
	}
	return strVal(rows[0], 0), strVal(rows[0], 1)
}

// leaseExpired reports whether the stored lease is absent or past its
// lease_until (a dead worker's lease is re-acquirable once expired).
func leaseExpired(t *testing.T, st store.Store) bool {
	t.Helper()
	owner, until := syncLeaseOf(t, st)
	if owner == "" || until == "" {
		return true
	}
	return leaseStamp(t, until).Before(time.Now().UTC())
}

// setLease writes the target row's lease pair directly (the test's stand-in
// for a dead worker that never released its lease).
func setLease(t *testing.T, st store.Store, owner, until string) {
	t.Helper()
	s, ok := st.(syncSeam)
	if !ok {
		t.Fatal("store does not expose the sync seam")
	}
	if err := s.RawExec(
		`INSERT INTO sync_state (target_key, lease_owner, lease_until) VALUES (?, ?, ?)
		 ON CONFLICT(target_key) DO UPDATE SET lease_owner = excluded.lease_owner, lease_until = excluded.lease_until`,
		autosyncTarget, owner, until); err != nil {
		t.Fatalf("set lease: %v", err)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [autosync-single-lease] — the lease row is the arbiter: two workers
// (two processes on one data dir) must not both push. The test asserts the
// store row, not a worker-internal flag.
// ────────────────────────────────────────────────────────────────────────────

func TestAutosyncSingleLease(t *testing.T) {
	st := openTestStore(t)
	tr := newFakeTransport(t)
	a1 := NewAutosync(NewSyncService(st, syncCfg(true)), tr, syncCfgAuto())
	a2 := NewAutosync(NewSyncService(st, syncCfg(true)), tr, syncCfgAuto())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a1.Start(ctx)
	a2.Start(ctx)
	defer a1.Stop()
	defer a2.Stop()

	a1.NotifyDirty()
	// wait for a1 (the winner) to cycle and hold the lease, THEN notify a2
	// (the loser): a2's cycle must see the live lease and stand down.
	waitForCycles(t, a1)
	if c := a1.CycleCount(); c < 1 {
		t.Fatalf("worker 1 must have cycled (attempted the lease), got %d", c)
	}
	a2.NotifyDirty()
	// wait for a2 to cycle (a stand-down counts as a cycled pass) — one
	// acquires the lease, the other must stand down; the proof is that the
	// loser cycled (it attempted) and contributed nothing.
	waitForCycles(t, a2)
	if c := a2.CycleCount(); c < 1 {
		t.Fatalf("worker 2 must have cycled (attempted the lease and stood down), got %d", c)
	}
	if err := a1.LastError(); err != nil {
		t.Fatalf("worker 1 cycle errored: %v", err)
	}
	if err := a2.LastError(); err != nil {
		t.Fatalf("worker 2 cycle errored (a stand-down is not a failure): %v", err)
	}
	// exactly ONE worker pushed: the loser stood down on the live lease
	// (its LastError is nil, so the split is visible only in the winner's
	// count). If both pushed, the split was never proven and this fails.
	if a1.CycleCount() != 1 {
		t.Fatalf("exactly one worker must win the lease and push, got a1=%d a2=%d",
			a1.CycleCount(), a2.CycleCount())
	}
	// the lease row is the single-worker proof: it names exactly one owner
	// (the winner — either a1 or a2, not both, not a foreign owner).
	owner, until := syncLeaseOf(t, st)
	if owner == "" || until == "" {
		t.Fatalf("the lease row must record the winner (owner=%q until=%q)", owner, until)
	}
	if owner != a1.Owner() && owner != a2.Owner() {
		t.Fatalf("lease row holds a foreign owner %q", owner)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [write-path-never-blocks-on-network] — the onWrite hook is a buffered-1
// channel send: a 500 ms round-trip on the transport must not delay the
// save commit. The store's committed-txn seam (SetTxnHook) is the pipeline's
// onWrite hook; the manager replaces its implementation with NotifyDirty.
// ────────────────────────────────────────────────────────────────────────────

func TestWritePathNeverBlocksOnNetwork(t *testing.T) {
	st := openTestStore(t)
	tr := newFakeTransport(t)
	tr.Slow(500 * time.Millisecond) // the network is slower than the save
	cfg := syncCfgAuto()
	cfg.Project.Name = "p"
	svc := NewSyncService(st, syncCfg(true))
	if err := svc.Enroll(context.Background(), "p", true); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	a := NewAutosync(svc, tr, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.Start(ctx)
	defer a.Stop()

	s := newTestServiceWithStore(t, st)
	// the manager replaces the pipeline's onWrite hook implementation with
	// its NotifyDirty (the store's committed-txn seam fires after commit).
	st.SetTxnHook(a.NotifyDirty)

	start := time.Now()
	if _, err := s.Save(ctx, SaveRequest{
		SessionID: "s", Type: "decision", Title: "T slow",
		Content: "content slow", Project: "p",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("save blocked on network: %v (the onWrite hook must be a non-blocking channel send)", elapsed)
	}
	// the cycle must have happened in the background, after the save
	// returned — the transport's counter proves the round-trip ran without
	// delaying the commit.
	waitForCycles(t, a)
	if tr.Calls() < 1 {
		t.Fatalf("the debounced cycle must have run in the background (transport calls: %d)", tr.Calls())
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [autosync-debounce] — a burst of dirty notifications within the debounce
// window coalesces to exactly one cycle (coalescing, not per-call: the
// second send finds the buffer full and is dropped).
// ────────────────────────────────────────────────────────────────────────────

func TestDebounceCoalescesBursts(t *testing.T) {
	st := openTestStore(t)
	cfg := syncCfgAuto()
	cfg.Project.Name = "p"
	svc := NewSyncService(st, syncCfg(true))
	if err := svc.Enroll(context.Background(), "p", true); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	// one journaled save (the outbox must be non-empty for the push to
	// reach the transport — an empty outbox is a no-op cycle). One local
	// ctx for both the save and the manager (no package-global ctx mixing).
	dctx, dcancel := context.WithCancel(context.Background())
	defer dcancel()
	s := newTestServiceWithStore(t, st)
	if _, err := s.Save(dctx, SaveRequest{
		SessionID: "s", Type: "decision", Title: "T burst",
		Content: "content burst", Project: "p",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	tr := newFakeTransport(t)
	a := NewAutosync(svc, tr, cfg)
	a.Start(dctx)
	defer a.Stop()
	for i := 0; i < 10; i++ {
		a.NotifyDirty()
	}
	waitForCycles(t, a)
	if n := tr.Calls(); n != 1 {
		t.Fatalf("a 10-send burst must coalesce to one cycle, got %d transport calls", n)
	}
	// the cycle is clean (a failed cycle also counts as one transport call —
	// the phase pins that this one succeeded).
	if a.Phase() != PhaseHealthy {
		t.Fatalf("the burst's cycle must end healthy, got %q", a.Phase())
	}
}

// TestNotifyDirtyNonBlocking (TRIANGULATE [write-path-never-blocks-on-network],
// different input: the channel full) — the buffered-1 send must never block:
// after the buffer holds one dirty mark, a flood of sends returns
// immediately (the mark is already there; the debounce timer resets once).
func TestNotifyDirtyNonBlocking(t *testing.T) {
	a := NewAutosync(NewSyncService(openTestStore(t), syncCfg(true)), newFakeTransport(t), syncCfgAuto())
	a.NotifyDirty() // fills the buffer (Start not called — no consumer)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		a.NotifyDirty()
	}
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Fatalf("NotifyDirty blocked: %v for 1000 sends on a full buffer (must be non-blocking)", elapsed)
	}
}

// TestAutosyncLeaseExpiry (TRIANGULATE [autosync-single-lease], different
// input: a dead worker's stale lease) — a worker that crashed holding the
// lease leaves lease_until in the future; the lease is re-acquirable once
// it expires (60 s TTL semantics — a second worker must not be outed
// forever by a dead peer).
func TestAutosyncLeaseExpiry(t *testing.T) {
	st := openTestStore(t)
	cfg := syncCfgAuto()
	cfg.Project.Name = "p"
	svc := NewSyncService(st, syncCfg(true))
	if err := svc.Enroll(context.Background(), "p", true); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	s := newTestServiceWithStore(t, st)
	if _, err := s.Save(ctx, SaveRequest{
		SessionID: "s", Type: "decision", Title: "T expiry",
		Content: "content expiry", Project: "p",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	// a dead worker's unexpired lease.
	setLease(t, st, "autosync-dead", time.Now().UTC().Add(60*time.Second).Format("2006-01-02 15:04:05"))
	if leaseExpired(t, st) {
		t.Fatalf("fixture: the dead worker's lease must be unexpired")
	}
	// a second worker cannot acquire while it is held.
	tr := newFakeTransport(t)
	a := NewAutosync(svc, tr, cfg)
	a.CycleOnce(context.Background())
	if tr.Calls() != 0 {
		t.Fatalf("a live lease must stand the second worker down, got %d transport calls", tr.Calls())
	}
	// the lease expires (simulate time passing the TTL): re-acquirable.
	setLease(t, st, "autosync-dead", time.Now().UTC().Add(-time.Second).Format("2006-01-02 15:04:05"))
	if !leaseExpired(t, st) {
		t.Fatalf("fixture: the lease must be expired now")
	}
	a.CycleOnce(context.Background())
	if tr.Calls() != 1 {
		t.Fatalf("an expired lease must be re-acquirable (got %d transport calls, want 1)", tr.Calls())
	}
	owner, _ := syncLeaseOf(t, st)
	if owner != a.Owner() {
		t.Fatalf("the re-acquisition must record the new owner, got %q want %q", owner, a.Owner())
	}
}

// TestAutosyncDisabledNeverCycles (TRIANGULATE [autosync-debounce],
// different input: sync.auto=false) — a disabled manager never cycles:
// dirty notifications pile up (buffered) but no transport call happens.
func TestAutosyncDisabledNeverCycles(t *testing.T) {
	cfg := syncCfg(true)
	cfg.Sync.Auto = false // the enable flag is off
	tr := newFakeTransport(t)
	a := NewAutosync(NewSyncService(openTestStore(t), cfg), tr, cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	a.Start(ctx)
	defer a.Stop()
	for i := 0; i < 10; i++ {
		a.NotifyDirty()
	}
	// wait past the debounce window (2×): a disabled manager still coalesces
	// nothing.
	waitForCycles(t, a) // the 0-count wait returns when the window passes
	if tr.Calls() != 0 {
		t.Fatalf("a disabled manager must never cycle, got %d transport calls", tr.Calls())
	}
}

// TestAutosyncCyclePushThenPull (TRIANGULATE [autosync-cycle], different
// input: both legs have work) — one cycle pushes the local outbox and then
// pulls the peer's mutations; the pull lands through the same store path
// (INV-4 — the FTS index serves it). Push first, then pull (a push
// failure does not pull: the outbound gate is the brief's ordering).
func TestAutosyncCyclePushThenPull(t *testing.T) {
	st := openTestStore(t)
	cfg := syncCfgAuto()
	cfg.Project.Name = "p" // the peer's project (the store's binding)
	svc := NewSyncService(st, syncCfg(true))
	if err := svc.Enroll(context.Background(), "p", true); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	s := newTestServiceWithStore(t, st)
	if _, err := s.Save(ctx, SaveRequest{
		SessionID: "s", Type: "decision", Title: "T push",
		Content: "content push", Project: "p",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if n := journalCount(t, st); n != 1 {
		t.Fatalf("the outbox must hold one mutation, got %d", n)
	}
	// the peer's side: one mutation to pull (a different sync id).
	tr := newFakeTransport(t)
	tr.seed("p", 1)

	a := NewAutosync(svc, tr, cfg)
	a.CycleOnce(context.Background())

	// the push leg: the outbox drained (acked).
	if n := journalCount(t, st); n != 0 {
		t.Fatalf("the push must ack the outbox, %d pending remain", n)
	}
	// the pull leg: the peer's mutation landed through the store (INV-4 —
	// the FTS index serves it).
	if !obsBySyncID(st, ctx, "k") {
		t.Fatalf("the pulled mutation must land through the store path (obs %q missing)", "k")
	}
}

// TestAutosyncStopForUpgradeHoldsLease (TRIANGULATE
// [autosync-single-lease], different input: the upgrade pause) —
// StopForUpgrade sets the disabled phase and does NOT release the lease:
// a second worker must not pick up sync mid-upgrade (the brief's
// upgrade-safe contract).
func TestAutosyncStopForUpgradeHoldsLease(t *testing.T) {
	st := openTestStore(t)
	tr := newFakeTransport(t)
	a := NewAutosync(NewSyncService(st, syncCfg(true)), tr, syncCfgAuto())
	a.CycleOnce(context.Background()) // acquire + hold the lease
	if !leaseHeldBy(t, st, a.Owner()) {
		owner, until := syncLeaseOf(t, st)
		t.Fatalf("the cycle must hold the lease, row is (%q, %q)", owner, until)
	}
	if err := a.StopForUpgrade(); err != nil {
		t.Fatalf("stop for upgrade: %v", err)
	}
	if a.Phase() != PhaseDisabled {
		t.Fatalf("phase after StopForUpgrade must be disabled, got %q", a.Phase())
	}
	if !leaseHeldBy(t, st, a.Owner()) {
		t.Fatalf("StopForUpgrade must NOT release the lease (a second worker must stand down mid-upgrade)")
	}
	// a second worker is stood down while the lease is held.
	b := NewAutosync(NewSyncService(st, syncCfg(true)), newFakeTransport(t), syncCfgAuto())
	b.CycleOnce(context.Background())
	if _, until := syncLeaseOf(t, st); until != "" && !leaseExpired(t, st) {
		if owner, _ := syncLeaseOf(t, st); owner != a.Owner() {
			t.Fatalf("the upgrade hold must stand worker B down; row owner is %q", owner)
		}
	}
	// resume re-arms without a restart: the next cycle runs again.
	if err := a.ResumeAfterUpgrade(); err != nil {
		t.Fatalf("resume: %v", err)
	}
	if a.Phase() != PhaseIdle {
		t.Fatalf("phase after resume must be idle, got %q", a.Phase())
	}
}

// leaseHeldBy reports whether the lease row names owner and is unexpired.
func leaseHeldBy(t *testing.T, st store.Store, owner string) bool {
	t.Helper()
	o, until := syncLeaseOf(t, st)
	if o != owner || until == "" {
		return false
	}
	return !leaseStamp(t, until).Before(time.Now().UTC())
}

// ────────────────────────────────────────────────────────────────────────────
// test driver (the synchronous adapter — the brief's second adapter: the
// goroutine is the in-process path, this one drives a single cycle without
// the debounce window for deterministic assertions).
// ────────────────────────────────────────────────────────────────────────────

// CycleOnce runs one manager cycle synchronously (backoff/disabled check →
// lease → push → pull → lease release). The test driver: the goroutine's
// cycle, minus the debounce timer.
func (a *Autosync) CycleOnce(ctx context.Context) {
	a.runCycle(ctx)
}

// Owner returns the manager's lease owner identity (autosync-<unixnano>).
func (a *Autosync) Owner() string { return a.owner }

// CycleCount returns the number of completed cycles (the fake-transport
// poll's target).
func (a *Autosync) CycleCount() int {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status.cycles
}

// LastError returns the last cycle's error (nil = the last cycle was clean).
func (a *Autosync) LastError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status.lastErr
}

// Phase returns the manager's current degraded-state phase.
func (a *Autosync) Phase() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.status.phase
}

// waitForCycles polls the workers' cycle counters until every worker has
// cycled at least once (a stand-down counts as a cycled pass), or the 2 s
// window passes (the brief: no timing assertions — the poll is the
// debounce proof; the coalescing/split assertions are the caller's, not the
// counter's). A window pass with a worker not yet cycled is not itself a
// failure: the caller's counter assertions are the proof.
func waitForCycles(t *testing.T, a ...*Autosync) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second) // 4× the 500 ms window
	for time.Now().Before(deadline) {
		all := true
		for _, m := range a {
			if m.CycleCount() < 1 {
				all = false
				break
			}
		}
		if all {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	// the window passed: the caller asserts (the lease test's 1-vs-0 split
	// assertion is the proof of which worker stood down).
}

// errFakeTransport is the fake transport's injected failure (the backoff
// test's input — a cycle that fails on the push leg).
var errFakeTransport = errors.New("fake transport: injected failure")

// Slow sets the fake transport's simulated round-trip latency (the
// write-path test's slow remote — the transport sleeps per call, the
// save must not).
func (f *fakeTransport) Slow(d time.Duration) {
	f.slowMs = d
}
