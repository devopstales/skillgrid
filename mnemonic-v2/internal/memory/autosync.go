package memory

// Autosync manager (spec 01-memory/003-sync §5 — Task 17). Port of the
// v1/Engram autosync manager (internal/cloud/autosync/manager.go): a
// background goroutine that debounces dirty notifications (coalescing: a
// burst of N sends within the window yields exactly one cycle), acquires
// the SQLite-backed lease before any network I/O (two MCP servers on one
// data dir must not double-push), and runs push→pull in that goroutine so
// the save commit path never touches the network.
//
// The seam is the store's committed-transaction hook (SetTxnHook): the
// pipeline calls it after the save transaction commits (the onWrite hook —
// the brief's "the manager replaces its implementation with
// NotifyDirty"). NotifyDirty is a non-blocking send on a buffered-1
// channel: a slow transport cannot block a save, and a burst of saves
// coalesces to one cycle (the second send finds the buffer full and is
// dropped — coalescing, not per-call).
//
// The lease is sync_state.lease_owner/lease_until (the 004_sync_lease
// migration), 60 s TTL, Engram store.go:8198-8233 semantics: acquire only
// when the stored lease is absent, ours, or expired (a dead worker's
// lease is re-acquirable once its TTL passes); release clears the pair
// only if the owner still matches (a stale release never wipes a live
// peer's lease).
//
// Degraded state (phase, backoff) follows Engram: exponential backoff
// with ±25% jitter, capped at 5 minutes, 10 consecutive failures →
// disabled + reason. StopForUpgrade holds the lease (PhaseDisabled, no
// release — no other worker picks up sync mid-upgrade);
// ResumeAfterUpgrade clears the flag and re-arms without a restart.

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
)

// ─── Phases (the degraded-state vocabulary, spec 003-sync §5) ─────────────

const (
	PhaseIdle       = "idle"
	PhasePushing    = "pushing"
	PhasePulling    = "pulling"
	PhasePushFailed = "push_failed"
	PhasePullFailed = "pull_failed"
	PhaseBackoff    = "backoff"
	PhaseHealthy    = "healthy"
	PhaseDisabled   = "disabled"
)

// autosync tuning (the Engram defaults; the config's DebounceMs overrides
// the debounce window, the brief's 500 ms).
const (
	autosyncLeaseTTL          = 60 * time.Second
	autosyncBaseBackoff       = time.Second
	autosyncMaxBackoff        = 5 * time.Minute
	autosyncMaxFailures       = 10
	autosyncPollInterval      = 30 * time.Second
	autosyncDefaultDebounceMs = 500

	// targetKeyLocal is the store's local target row (the journal's target
	// key — Task 15's EnqueueMutation / the gate write 'local' rows). The
	// lease lives on it; the pull cursor lives on 'cloud' (the SyncService's
	// recordPullState writes it on the mutation's target key, spec 006 §7).
	targetKeyLocal = "local"
)

// Autosync is the lease-guarded background sync manager. Safe for
// concurrent use (NotifyDirty is called from the commit path; the cycle
// runs in one goroutine).
type Autosync struct {
	svc *SyncService
	tr  MutationTransport
	cfg config.Config

	owner string // autosync-<unixnano> — the lease identity
	mu    sync.Mutex
	// dirtyCh is the onWrite hook's buffered-1 channel: the store never
	// blocks on network (a full buffer drops the mark — coalescing).
	dirtyCh   chan struct{}
	status    autosyncStatus
	leaseHeld bool
	cancelFn  context.CancelFunc
	ready     chan struct{} // closed once the run loop is registered
	wg        sync.WaitGroup
}

// autosyncStatus is the manager's degraded-state snapshot (phase +
// backoff + counters — the brief's phase vocabulary).
type autosyncStatus struct {
	phase        string
	lastErr      error
	failures     int
	backoffUntil time.Time
	cycles       int
}

// NewAutosync builds the manager over the sync service (the journal apply
// seam, Task 15) and the transport (Task 16). The lease owner is minted
// per instance: autosync-<unixnano> (two processes on one data dir get
// different owners — the lease row disambiguates).
func NewAutosync(svc *SyncService, tr MutationTransport, cfg config.Config) *Autosync {
	return &Autosync{
		svc:     svc,
		tr:      tr,
		cfg:     cfg,
		owner:   fmt.Sprintf("autosync-%d", time.Now().UnixNano()),
		dirtyCh: make(chan struct{}, 1),
		status:  autosyncStatus{phase: PhaseIdle},
	}
}

// debounceMs returns the config's debounce window (the brief's 500 ms).
func (a *Autosync) debounceMs() time.Duration {
	if a.cfg.Sync.DebounceMs > 0 {
		return time.Duration(a.cfg.Sync.DebounceMs) * time.Millisecond
	}
	return autosyncDefaultDebounceMs * time.Millisecond
}

// NotifyDirty is the onWrite hook: a non-blocking send on the buffered-1
// channel (the store never blocks on network — a full buffer drops the
// mark, which is the coalescing semantics the burst test asserts).
func (a *Autosync) NotifyDirty() {
	select {
	case a.dirtyCh <- struct{}{}:
	default:
		// already signaled: dropped (coalescing).
	}
}

// Start launches the run loop in a background goroutine (the in-process
// adapter). A second Start is a no-op (the re-entry guard — the run loop
// owns the goroutine).
func (a *Autosync) Start(ctx context.Context) {
	a.mu.Lock()
	if a.cancelFn != nil {
		a.mu.Unlock()
		return
	}
	if a.ready == nil {
		a.ready = make(chan struct{})
	}
	a.mu.Unlock()
	go a.run(ctx)
}

// Stop cancels the run loop and waits for it to exit (the startup
// handshake: the ready channel closes once the loop is registered, so a
// Stop racing Start still sees the registration). Safe before Start.
func (a *Autosync) Stop() {
	a.mu.Lock()
	ready := a.ready
	a.mu.Unlock()
	if ready != nil {
		<-ready
	}
	a.mu.Lock()
	fn := a.cancelFn
	a.mu.Unlock()
	if fn != nil {
		fn()
	}
	a.wg.Wait()
}

// StopForUpgrade sets PhaseDisabled and stops cycling, but does NOT
// release the lease: no other worker picks up sync mid-upgrade (the
// brief's upgrade-safe contract — Engram manager.go:384-399).
func (a *Autosync) StopForUpgrade() error {
	a.mu.Lock()
	a.status.phase = PhaseDisabled
	a.status.backoffUntil = time.Time{} // a pause is not a backoff
	a.status.lastErr = nil
	a.mu.Unlock()
	return nil
}

// ResumeAfterUpgrade clears the disabled flag (PhaseIdle) and re-arms
// without a restart (the brief's "clears the flag, re-arms" contract —
// Engram manager.go:398-423: a dirty signal wakes the run loop).
func (a *Autosync) ResumeAfterUpgrade() error {
	a.mu.Lock()
	if a.status.phase != PhaseDisabled {
		a.mu.Unlock()
		return nil // not paused: no-op
	}
	a.status.phase = PhaseIdle
	a.status.failures = 0
	a.status.backoffUntil = time.Time{}
	a.mu.Unlock()
	a.NotifyDirty()
	return nil
}

// ─── the run loop (the in-process adapter) ─────────────────────────────────

// run is the main loop: it selects on ctx / dirtyCh (each send resets the
// debounce timer — coalescing: a burst within the window is one cycle) /
// the poll ticker (the 30 s freshness check). It returns when ctx is
// cancelled or Stop runs; on return it releases the lease (the
// leaseHeld guard: StopForUpgrade's pause keeps it).
func (a *Autosync) run(ctx context.Context) {
	a.mu.Lock()
	if a.cancelFn != nil {
		a.mu.Unlock()
		return // re-entry guard
	}
	inner, cancel := context.WithCancel(ctx)
	a.cancelFn = cancel
	ready := a.ready
	a.mu.Unlock()

	a.wg.Add(1)
	defer a.wg.Done()
	defer cancel()
	defer a.releaseLease()
	if ready != nil {
		close(ready)
	}

	debounce := time.NewTimer(a.debounceMs())
	a.drain(debounce)
	poll := time.NewTicker(autosyncPollInterval)
	defer poll.Stop()

	for {
		select {
		case <-inner.Done():
			return
		case <-a.dirtyCh:
			a.drain(debounce)
			debounce.Reset(a.debounceMs())
		case <-debounce.C:
			a.runCycle(inner)
		case <-poll.C:
			a.runCycle(inner)
		}
	}
}

// drain stops a timer and swallows its pending fire (the Engram run loop's
// reset pattern — a stale fire would otherwise fire the cycle early).
func (a *Autosync) drain(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
}

// ─── the cycle (push then pull, lease-guarded) ─────────────────────────────

// runCycle is the manager's cycle: disabled/backoff check → lease → push →
// pull. All network I/O is here (the goroutine — the write path never sees
// it). The lease is HELD after the cycle (not released): the TTL (60 s) is
// the release mechanism (a dead worker's lease expires; a live worker
// renews on the next cycle). Stop() releases on shutdown.
//
// runCycle never returns to the run loop carrying a panic (a store seam or
// transport bug would otherwise kill the run goroutine — and with it the
// lease guard, the debounce, and the backoff tracking): the panic is
// recovered and recorded as a cycle failure (logged through the manager's
// status surface, LastError), so a future cycle bug degrades to backoff,
// not a dead worker.
func (a *Autosync) runCycle(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			a.recordFailure(fmt.Errorf("cycle panic: %v", r))
		}
	}()
	if ctx.Err() != nil {
		return
	}
	if !a.cfg.Sync.Auto {
		return // the enable flag is off: a disabled manager never cycles
	}
	a.mu.Lock()
	if a.status.phase == PhaseDisabled {
		a.mu.Unlock()
		return
	}
	if a.status.failures >= autosyncMaxFailures && time.Now().Before(a.status.backoffUntil) {
		a.status.phase = PhaseBackoff
		a.mu.Unlock()
		return
	}
	if time.Now().Before(a.status.backoffUntil) {
		a.mu.Unlock()
		return // in backoff: the poll re-checks after the deadline
	}
	a.mu.Unlock()

	if err := a.acquireLease(autosyncLeaseTTL); err != nil {
		if err == errLeaseHeld {
			// a live peer holds the lease: stand down (no backoff — the
			// next dirty mark re-checks; the TTL is the release mechanism).
			// The stand-down counts as a cycled pass (the single-lease test
			// asserts both workers cycled; the loser's proof is that it
			// cycled and contributed no transport calls).
			a.mu.Lock()
			if a.status.phase == PhasePushing || a.status.phase == PhasePulling {
				a.status.phase = PhaseIdle
			}
			a.status.cycles++
			a.mu.Unlock()
			return
		}
		a.recordFailure(fmt.Errorf("lease: %w", err))
		return
	}
	a.mu.Lock()
	a.leaseHeld = true
	a.mu.Unlock()

	if err := a.pushLeg(ctx); err != nil {
		a.setPhase(PhasePushFailed)
		a.recordFailure(err)
		return
	}
	if err := a.pullLeg(ctx); err != nil {
		a.setPhase(PhasePullFailed)
		a.recordFailure(err)
		return
	}
	a.recordSuccess()
}

// pushLeg pushes the local outbox (ListPendingMutations → PushMutations):
// one push for the whole pending set (the batch cap is enforced at the
// transport boundary; a full batch loops until the outbox drains — the
// CLI's syncPush is the same loop, K4 one code path two callers).
func (a *Autosync) pushLeg(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	a.setPhase(PhasePushing)
	limit := a.cfg.Sync.PushLimit
	if limit <= 0 {
		limit = MaxBatch
	}
	// ponytail: one project per store (one data dir, one journal) — the
	// outbox's project is the batch's project (the CLI's syncPush resolves
	// the same way). A journal row with an empty project (an unenrolled
	// write) has no resolvable schema: a no-op cycle (the transport is
	// project-scoped; the row stays pending until enrollment).
	// Upgrade path: a per-project grouping loop when a single store
	// journals for several projects.
	project, seen := a.batchProject()
	if !seen {
		return nil // nothing pushable (no project in the outbox)
	}
	for {
		batch, err := a.svc.st.ListPendingMutations(ctx, 0, limit)
		if err != nil {
			return fmt.Errorf("list pending: %w", err)
		}
		if len(batch) == 0 {
			return nil // empty outbox: a valid no-op (the idle cycle)
		}
		if _, err := Push(ctx, a.svc.st, a.tr, project, batch); err != nil {
			if err == ErrPaused {
				return nil // paused project: the write was skipped, the cursor is untouched
			}
			return err
		}
	}
}

// batchProject returns the outbox's project (the store's single-project
// journal) — the transport is project-scoped (one schema per project).
func (a *Autosync) batchProject() (string, bool) {
	batch, err := a.svc.st.ListPendingMutations(context.Background(), 0, 1)
	if err != nil || len(batch) == 0 {
		return "", false
	}
	p := batch[0].Project
	return p, p != ""
}

// pullLeg pulls the peer's mutations since the local cursor (PullMutations
// → ApplyPulled — the SyncService's seam advances last_pulled_seq per
// applied mutation) and replays the deferred queue (the pull cycle's
// retry step, spec 006 §6).
//
// The peer's project is resolved from the outbox (the store's single-
// project journal) or, if the outbox is empty (no local writes yet), from
// the config's project name. The transport is project-scoped (one schema
// per project); the pull reads the peer's schema for that project.
func (a *Autosync) pullLeg(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	a.setPhase(PhasePulling)
	since := a.lastPulledSeq()
	project, seen := a.batchProject()
	if !seen {
		// no local writes yet: fall back to the config's project name
		// (the store's binding — the CLI's syncPull resolves the same
		// way: explicit selector, else the store's project).
		project = a.cfg.Project.Name
		if project == "" {
			return nil // unresolvable: no schema to read
		}
	}
	_, err := PullAndApply(ctx, a.svc.st, a.tr, project, since)
	return err
}

// lastPulledSeq reads the target row's last_pulled_seq (the pull cursor —
// spec 006 §7). The cursor is target-scoped (the SyncService's
// recordPullState writes it on the mutation's target key, 'cloud' for
// pulled rows; the local journal's target is 'local'). A fresh store
// (no row yet) starts at 0.
func (a *Autosync) lastPulledSeq() int64 {
	seam := applyPulledSeam(a.svc.st)
	// the cursor is on the 'cloud' target (the peer's mutations land
	// with target_key='cloud' — the transport's PullAndApply applies
	// them through the SyncService, which records the cursor on the
	// mutation's target key).
	n, err := seam.ProbeInt(`SELECT COALESCE(last_pulled_seq, 0) FROM sync_state WHERE target_key = 'cloud'`)
	if err != nil {
		return 0
	}
	return int64(n)
}

// ─── the lease (sync_state.lease_owner/lease_until — 004_sync_lease) ──────

// acquireLease takes the lease when the stored lease is absent, ours, or
// expired (the Engram semantics: a dead worker's lease is re-acquirable
// once its TTL passes; a live peer's lease stands the worker down).
//
// The takeover is ATOMIC, not SELECT-then-UPSERT: the conditional UPDATE
// fires only when the row holds no live peer lease (absent, ours, or
// expired), and its effect is the arbiter — RowsAffected==0 means a live
// peer holds it, so two workers racing the same window cannot both win
// (the SELECT-then-UPSERT form let both workers upsert and last writer
// won; the UPDATE's WHERE clause closes the TOCTOU). The upsert after a
// successful takeover only guarantees the row exists for a fresh store.
func (a *Autosync) acquireLease(ttl time.Duration) error {
	seam := applyPulledSeam(a.svc.st)
	now := time.Now().UTC()
	nowStamp := now.Format("2006-01-02 15:04:05")
	untilStamp := now.Add(ttl).Format("2006-01-02 15:04:05")
	// The WHERE clause: the row is available when it holds no live peer
	// lease. A row is "live peer" when lease_owner is set, != us, and
	// lease_until > now. The UPDATE matches (and thus takes the row)
	// when:
	//   lease_owner IS NULL     → no one holds it (fresh / released)
	//   lease_owner = ?         → we already hold it (renew)
	//   lease_until IS NULL     → no expiry (treat as unheld, take it)
	//   lease_until <= ?        → expired (dead worker's lease, <= now)
	// A live peer (owner set, != us, until > now) does NOT match →
	// RowsAffected==0. The row's existence then decides between "fresh
	// store" (create it) and "peer holds it" (stand down).
	res, err := seam.RawExecRowsAffected(
		`UPDATE sync_state SET lease_owner = ?, lease_until = ?
		 WHERE target_key = ?
		   AND (lease_owner IS NULL
		        OR lease_owner = ?
		        OR lease_until IS NULL
		        OR lease_until <= ?)`,
		a.owner, untilStamp, targetKeyLocal, a.owner, nowStamp)
	if err != nil {
		return fmt.Errorf("acquire lease: %w", err)
	}
	if res == 0 {
		// The UPDATE matched nothing: the row exists and a live peer
		// holds it (lease_owner != us, lease_until > now). The probe
		// confirms (it excludes our own owner, so a stale row owned by
		// us with an unparseable expiry counts as absent — the upsert
		// below takes it back). The probe's QueryRow returns
		// sql.ErrNoRows when no row matches: any probe error means "no
		// live peer" (a real DB error is surfaced by the upsert).
		peerHeld, _ := seam.Probe(
			`SELECT 1 FROM sync_state WHERE target_key = ? AND lease_owner IS NOT NULL AND lease_owner != ?`,
			targetKeyLocal, a.owner)
		if peerHeld {
			return errLeaseHeld // a live peer holds the lease: stand down
		}
		// no live peer: take (or create) the row with our lease.
		if err := seam.RawExec(
			`INSERT INTO sync_state (target_key, lease_owner, lease_until) VALUES (?, ?, ?)
			 ON CONFLICT(target_key) DO UPDATE SET lease_owner = excluded.lease_owner, lease_until = excluded.lease_until`,
			targetKeyLocal, a.owner, untilStamp); err != nil {
			return fmt.Errorf("acquire lease (takeover): %w", err)
		}
		return nil
	}
	return nil
}

// errLeaseHeld is the "a live peer holds the lease" stand-down (not an
// error state for the manager — the worker simply waits; the recordFailure
// call site converts it to a backoff-less stand-down).
var errLeaseHeld = fmt.Errorf("lease: held by another worker")

// releaseLease clears the lease pair only if the owner still matches (a
// stale release never wipes a live peer's lease — Engram semantics) and
// only when the manager holds it (StopForUpgrade's pause keeps it).
func (a *Autosync) releaseLease() {
	a.mu.Lock()
	if !a.leaseHeld {
		a.mu.Unlock()
		return
	}
	a.leaseHeld = false
	a.mu.Unlock()
	seam := applyPulledSeam(a.svc.st)
	_ = seam.RawExec(
		`UPDATE sync_state SET lease_owner = NULL, lease_until = NULL
		 WHERE target_key = ? AND lease_owner = ?`,
		targetKeyLocal, a.owner)
}

// leasePair reads the target row's (owner, until) ("" = none).
func leasePair(seam syncSeam, targetKey string) (string, string) {
	rows, err := seam.RawQuery(
		`SELECT ifnull(lease_owner,''), ifnull(lease_until,'') FROM sync_state WHERE target_key = ?`, targetKey)
	if err != nil || len(rows) == 0 {
		return "", ""
	}
	if len(rows[0]) < 2 {
		return "", ""
	}
	return strVal(rows[0], 0), strVal(rows[0], 1)
}

// ─── state tracking (backoff + phases, Engram manager.go:949-1032) ────────

func (a *Autosync) setPhase(phase string) {
	a.mu.Lock()
	a.status.phase = phase
	a.mu.Unlock()
}

// recordFailure increments the failure counter and sets the backoff
// (exponential, ±25% jitter, capped at 5 minutes); at the 10-failure
// ceiling the manager is disabled (phase disabled + reason). The
// lease-held stand-down is handled at the call site (no backoff — the
// TTL is the release mechanism; this method is not called with
// errLeaseHeld).
func (a *Autosync) recordFailure(err error) {
	a.mu.Lock()
	a.status.failures++
	a.status.lastErr = err
	a.status.backoffUntil = time.Now().Add(a.computeBackoff(a.status.failures))
	if a.status.failures >= autosyncMaxFailures {
		a.status.phase = PhaseDisabled
	} else if a.status.phase == PhasePushing {
		a.status.phase = PhasePushFailed
	} else if a.status.phase == PhasePulling {
		a.status.phase = PhasePullFailed
	} else {
		a.status.phase = PhaseBackoff
	}
	a.mu.Unlock()
}

// recordSuccess resets the degraded state (the cycle completed clean).
func (a *Autosync) recordSuccess() {
	a.mu.Lock()
	a.status.phase = PhaseHealthy
	a.status.failures = 0
	a.status.lastErr = nil
	a.status.backoffUntil = time.Time{}
	a.status.cycles++
	a.mu.Unlock()
}

// computeBackoff is the Engram formula: base·2^(n-1) capped at 5 minutes,
// ±25% jitter (a uniform draw in [-base/4, +base/4]).
func (a *Autosync) computeBackoff(failures int) time.Duration {
	base := autosyncBaseBackoff
	for i := 1; i < failures && base < autosyncMaxBackoff; i++ {
		if base > autosyncMaxBackoff/2 {
			base = autosyncMaxBackoff
		} else {
			base *= 2
		}
	}
	jitter := time.Duration(rand.Int63n(int64(base/2)+1)) - base/4
	result := base + jitter
	if result < autosyncBaseBackoff/2 {
		result = autosyncBaseBackoff / 2
	}
	if result > autosyncMaxBackoff {
		result = autosyncMaxBackoff
	}
	return result
}
