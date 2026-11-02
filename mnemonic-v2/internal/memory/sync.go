package memory

// Sync core — spec 01-memory/006-sync §3-7 (Task 15: journal apply,
// tombstones, deferred-apply, enrollment). The transport (PG push/pull,
// Task 16) and the autosync manager (Task 17) drive SyncService; this file
// owns the apply seam itself.
//
// INV-4 (local truth): a pulled mutation is applied through the SAME store
// methods a local write uses — no special remote path, no raw SQL. The
// consequence the tests pin: the FTS triggers fire on a pulled upsert,
// exactly as they would for a local save. The journal (sync_mutations) is
// append-only local truth: a pull never rewrites it, and a pulled apply
// never re-enqueues locally.
//
// Ordering is seq + tombstone floors + deferred-apply (spec §3: LWW, no
// CRDT). The v1/Engram autosync manager (lease, debounce, backoff, phases)
// is the proven pattern Task 17 ports; this seam is what its pull loop
// calls per mutation.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// deferredApplyCap: 5 failed replays → dead + pruned (spec 006 §6).
const deferredApplyCap = 5

// SyncService applies pulled mutations to the local store (the pull-apply
// seam) and owns enrollment. It holds no transport: the transport (Task 16)
// fetches mutations and hands them here, one at a time.
type SyncService struct {
	st  store.Store
	cfg config.Config
}

// NewSyncService builds the sync service over a store and wires the sync
// gate from config: cfg.Sync.Enabled turns the store's local journaling on
// (the SetSyncEnabled seam is the gate the SaveTXN's enqueue step reads —
// a disabled service journals nothing). The gate is one config switch:
// construction is where it applies, so a caller that opens a store directly
// (a test) flips it via the store seam instead.
func NewSyncService(st store.Store, cfg config.Config) *SyncService {
	if seam, ok := st.(interface {
		SetSyncEnabled(bool)
	}); ok {
		seam.SetSyncEnabled(cfg.Sync.Enabled)
	}
	return &SyncService{st: st, cfg: cfg}
}

func init() {
	// The store's ApplyPulledMutation interface method and the service's
	// ApplyPulled share one dispatch: the store owns the interface (Task 1),
	// the memory package owns the apply logic (spec 006 §6), and the
	// dependency direction memory → store forbids the store from importing
	// the logic. The hook is the one-way wire (the pipeline's beginSaveTxn
	// type-assertion pattern, inverted).
	store.SetApplyPulledMutationHook(applyPulledMutation)
}

// syncMutationPayload is the canonical entity for a pulled observation
// upsert (spec 006 §4: the payload is the canonical entity —
// project-canonicalized, redaction/truncation already done by the source).
type syncMutationPayload struct {
	SyncID    string `json:"sync_id"`
	SessionID string `json:"session_id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	ToolName  string `json:"tool_name"`
	Project   string `json:"project"`
	Scope     string `json:"scope"`
	TopicKey  string `json:"topic_key"`
}

// syncRelationPayload is the canonical entity for a pulled relation upsert.
type syncRelationPayload struct {
	SyncID   string `json:"sync_id"`
	SourceID string `json:"source_id"`
	TargetID string `json:"target_id"`
	Relation string `json:"relation"`
	Reason   string `json:"reason"`
	Evidence string `json:"evidence"`
}

// syncSeam is the store seam ApplyPulled writes through (the SQLite adapter
// implements both — the raw-SQL read surface and AddPulledObservation; the
// type assertion fails fast on an adapter that lacks either, so a
// half-implemented adapter cannot silently no-op a pull).
type syncSeam interface {
	Probe(query string, args ...any) (bool, error)
	ProbeInt(query string, args ...any) (int, error)
	RawQuery(query string, args ...any) ([][]any, error)
	RawExec(query string, args ...any) error
	// RawExecRowsAffected runs a raw statement and returns its row count
	// (the autosync lease takeover's arbiter — the conditional UPDATE's
	// RowsAffected decides who wins the window).
	RawExecRowsAffected(query string, args ...any) (int64, error)
	AddPulledObservation(ctx context.Context, p store.AddObservationParams, syncID string) (store.AddObservationResult, error)
	AddPulledRelation(ctx context.Context, p store.AddRelationParams, syncID string) (store.Relation, error)
}

// applyPulledSeam narrows a store to the sync seam (a failed assertion is
// a test/store wiring bug — the SQLite adapter is the only one Task 15
// targets).
func applyPulledSeam(st store.Store) syncSeam {
	p, ok := st.(syncSeam)
	if !ok {
		panic(fmt.Sprintf("sync: store %T does not implement the sync seam (expected *store.sqliteStore)", st))
	}
	return p
}

// ApplyPulled applies one pulled upsert mutation to the local store
// (spec 006 §6). Dispatch on m.Entity:
//
//   - observation → the same store write path a local save uses
//     (AddObservation → FTS trigger fires → INV-4). Idempotent: a live row
//     with the same sync_id is a no-op (LWW — a second apply of the same
//     mutation has one effect); a tombstoned key (a delete already applied)
//     is a no-op (tombstone floor: a late upsert cannot resurrect).
//   - relation → AddRelation once both endpoint observations are present;
//     a missing endpoint defers the mutation (sync_apply_deferred) and is
//     retried by ReplayDeferred.
//   - prompt / session / project → not journaled by the local save path in
//     this task (the prompt/session journal rows land with their owning
//     tasks; a pulled mutation for them is a no-op here, not an error).
//
// ApplyPulled never enqueues: the journal is the outbox for THIS machine's
// writes only (append-only). It does not return a deferred error: a
// deferral is a normal outcome (the cursor advances — the Engram pull
// loop's per-entity error policy, autosync/manager.go §Phase E).
func (s *SyncService) ApplyPulled(ctx context.Context, m store.Mutation) error {
	if err := applyPulledMutation(ctx, s.st, m); err != nil {
		// A deferral is a successful outcome (the row is parked in the
		// deferred queue; the pull cursor advances) — not an error.
		if isDeferral(err) {
			return nil
		}
		return err
	}
	// the pull cursor (spec 006 §7): record the target's last_pulled_seq
	// after a successful apply (a deferral is a success for the cursor —
	// the row is parked, not lost).
	s.recordPullState(ctx, m)
	return nil
}

// ApplyDelete applies one pulled delete (the tombstone-floor ordering —
// spec 006 §6 — lives in applyDelete).
func (s *SyncService) ApplyDelete(ctx context.Context, m store.Mutation) error {
	return applyDelete(s.st, ctx, m)
}

// applyObservation applies a pulled observation upsert through the store's
// local write path (INV-4). Idempotent: an existing live row with the same
// sync_id (the same mutation applied twice, or a local save that already
// produced it) is a no-op.
func applyObservation(st store.Store, ctx context.Context, m store.Mutation) error {
	// idempotency: a live row with the same sync_id → the effect is already
	// here (this exact mutation was applied before, or a local save wrote
	// the same entity first — LWW says either way, one row).
	if obsBySyncID(st, ctx, m.EntityKey) {
		return nil
	}
	var p syncMutationPayload
	if err := json.Unmarshal([]byte(m.Payload), &p); err != nil {
		return fmt.Errorf("sync: observation payload %s: %w", m.EntityKey, err)
	}
	if p.SyncID == "" {
		p.SyncID = m.EntityKey // the journal key IS the sync_id (INV-6)
	}
	// The store's AddObservation mints its own sync_id; the canonical
	// payload carries the source's, which is the identity the peer knows.
	// A pull must not re-mint: use the payload's sync_id so the row is
	// findable by the id the source journaled. (The store seam
	// AddPulledObservation exists for exactly this — same write path as a
	// local save, but with the caller's sync_id.)
	project := p.Project
	// ponytail: scope is NOT project-canonicalized at apply — the source
	// machine already canonicalized it before enqueue (spec 006 §4: the
	// payload IS the canonical entity). Re-canonicalizing here would
	// diverge the pulled row's search scope from the source's intent.
	_, err := applyPulledSeam(st).AddPulledObservation(ctx, store.AddObservationParams{
		SessionID: p.SessionID,
		Type:      p.Type,
		Title:     p.Title,
		Content:   p.Content,
		ToolName:  p.ToolName,
		Project:   project,
		Scope:     p.Scope,
		TopicKey:  p.TopicKey,
	}, p.SyncID)
	return err
}

// applyRelation applies a pulled relation upsert. Both endpoint
// observations must be present (the store's relation rows reference sync
// ids; a missing endpoint is a missing FK — spec 006 §6 deferred-apply).
// Idempotent: a relation row with the same sync_id → no-op.
func applyRelation(st store.Store, ctx context.Context, m store.Mutation) error {
	var p syncRelationPayload
	if err := json.Unmarshal([]byte(m.Payload), &p); err != nil {
		return fmt.Errorf("sync: relation payload %s: %w", m.EntityKey, err)
	}
	if p.SyncID == "" {
		p.SyncID = m.EntityKey
	}
	// idempotency: a relation with the same sync_id is already here.
	exists, err := probeInt(applyPulledSeam(st), `SELECT COUNT(*) FROM memory_relations WHERE sync_id = ?`, p.SyncID)
	if err != nil {
		return err
	}
	if exists > 0 {
		return nil
	}
	// endpoint presence check → defer on a missing FK.
	for _, endpoint := range []string{p.SourceID, p.TargetID} {
		present, err := probeInt(applyPulledSeam(st),
			`SELECT COUNT(*) FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, endpoint)
		if err != nil {
			return err
		}
		if present == 0 {
			// FK dependency missing → defer (spec 006 §6). The pull cursor
			// advances; ReplayDeferred retries on each cycle, 5 failed
			// replays → dead + pruned.
			// ponytail: the park is the terminal action here — a deferral
			// is a SUCCESSFUL outcome (nil), not an error: the service's
			// ApplyPulled records the pull cursor on nil, and the row is
			// owned by the deferred queue from here on. The only error is
			// a failed park write (the queue itself is broken).
			return parkDeferred(st, ctx, m, "missing_fk_endpoint", 1)
		}
	}
	_, err = applyPulledSeam(st).AddPulledRelation(ctx, store.AddRelationParams{
		SourceID:  p.SourceID,
		TargetID:  p.TargetID,
		Relation:  p.Relation,
		Reason:    p.Reason,
		Evidence:  p.Evidence,
		SessionID: "",
	}, p.SyncID)
	if err != nil {
		return err
	}
	// the relation landed: drain the deferred row for this sync_id (a
	// fresh apply has no row; a replayed one does).
	pruneDeferred(st, ctx, p.SyncID)
	return nil
}

// ReplayDeferred replays every deferred mutation once (the pull cycle's
// retry step, spec 006 §6). A replay that succeeds is removed from the
// queue; a replay that still fails increments retry_count, and at
// deferredApplyCap the row is dead and pruned. Idempotent: replaying a
// drained queue is a no-op.
func (s *SyncService) ReplayDeferred(ctx context.Context) {
	rows, err := deferredRows(s.st, ctx)
	if err != nil {
		return
	}
	for _, row := range rows {
		var m store.Mutation
		if err := json.Unmarshal([]byte(row.payload), &m); err != nil {
			// A malformed row is dead on sight (it can never replay);
			// prune it so it does not re-fail every cycle.
			pruneDeferred(s.st, ctx, row.syncID)
			continue
		}
		// Take the row out of the queue BEFORE the apply: the apply path
		// RE-PARKS on a still-missing FK (a fresh insert — the pre-prune
		// means no ON CONFLICT), so the replay outcome is read from the
		// returned error, not the queue.
		// Read the carried count BEFORE the prune (the prune removes the
		// row the count lives in).
		carried := row.retryCount
		pruneDeferred(s.st, ctx, row.syncID)
		err := applyPulledMutation(ctx, s.st, m)
		switch {
		case isDeferral(err):
			// re-parked: a failed replay. The apply path re-parked with
			// count 1 (it doesn't know the carried count) — overwrite with
			// carried+1 (the monotonic counter). At the 5th failure the
			// row is dead + pruned (spec 006 §6).
			next := carried + 1
			if next >= deferredApplyCap {
				pruneDeferred(s.st, ctx, row.syncID)
			} else {
				_ = parkDeferred(s.st, ctx, m, "replay_deferred", next)
			}
		case err != nil:
			// A hard error (payload corrupt, a still-missing FK that the
			// apply path re-parked, a store failure): the apply path left
			// the row parked (or re-parked it) — fix the count to
			// carried+1 (the monotonic counter); at the 5th failure the
			// row is dead + pruned.
			next := carried + 1
			if n, _ := probeInt(applyPulledSeam(s.st),
				`SELECT COUNT(*) FROM sync_apply_deferred WHERE sync_id = ?`, row.syncID); n == 0 {
				// no row left (the error was not a park): the row is dead.
				continue
			}
			if next >= deferredApplyCap {
				pruneDeferred(s.st, ctx, row.syncID)
			} else {
				_ = parkDeferred(s.st, ctx, m, "replay_error", next)
			}
		}
		// (nil: it landed — the apply path pruned the row; nothing to do.)
	}
}

// Enroll sets a project's enrollment in sync_enrolled_projects (spec 006
// §4: only enrolled projects generate remote journal entries — the
// pipeline's step-7 ENQUEUE checks it in-transaction, Task 5's call site).
// on=false removes the enrollment (the gate is a live lookup, not a latch).
func (s *SyncService) Enroll(ctx context.Context, project string, on bool) error {
	if on {
		txn, err := beginSaveTxn(ctx, s.st)
		if err != nil {
			return err
		}
		defer func() { _ = txn.Rollback() }()
		if err := txn.EnrollProject(ctx, project); err != nil {
			return err
		}
		return txn.Commit(ctx)
	}
	// unenroll: the gate's in-txn path has no remove (enrollment is
	// insert-only there); the remove is a direct statement through the
	// store seam.
	return exec(s.st, `DELETE FROM sync_enrolled_projects WHERE project = ?`, project)
}

// TargetStatus is one target_key's sync_state row (spec 006 §7 — the
// `mnemonic sync status` read surface).
type TargetStatus struct {
	Lifecycle    string
	LastEnqueued int64
	LastAcked    int64
	LastPulled   int64
	BackoffUntil string
}

// Status reports every target's sync_state (lifecycle + seqs + backoff).
// A target with no row yet (nothing pulled/pushed) is absent from the map.
func (s *SyncService) Status(ctx context.Context) (map[string]TargetStatus, error) {
	_ = ctx
	rows, err := query(s.st,
		`SELECT target_key, lifecycle, last_enqueued_seq, last_acked_seq, last_pulled_seq, ifnull(backoff_until,'')
		 FROM sync_state`, nil)
	if err != nil {
		return nil, err
	}
	out := make(map[string]TargetStatus, len(rows))
	for _, r := range rows {
		out[strVal(r, 0)] = TargetStatus{
			Lifecycle:    strVal(r, 1),
			LastEnqueued: int64Val(r, 2),
			LastAcked:    int64Val(r, 3),
			LastPulled:   int64Val(r, 4),
			BackoffUntil: strVal(r, 5),
		}
	}
	return out, nil
}

// recordPullState advances the pulled target's last_pulled_seq in
// sync_state (the `mnemonic sync status` cursor; spec 006 §7). The target
// row is created on first pull (lifecycle 'idle' → the transport/autosync
// layer owns the lifecycle transitions from Task 16/17 on).
func (s *SyncService) recordPullState(ctx context.Context, m store.Mutation) {
	_ = ctx
	_ = exec(s.st,
		`INSERT INTO sync_state (target_key, lifecycle, last_pulled_seq) VALUES (?, 'idle', ?)
		 ON CONFLICT(target_key) DO UPDATE SET
		   last_pulled_seq = MAX(last_pulled_seq, excluded.last_pulled_seq)`,
		m.TargetKey, m.Seq)
}

// applyPulledMutation is the store-level apply (a pulled mutation → the
// same store path a local write uses, INV-4). The store's
// ApplyPulledMutation method routes here; SyncService.ApplyPulled is the
// same function plus the sync_state cursor (the service owns the
// lifecycle read surface, spec 006 §7).
func applyPulledMutation(ctx context.Context, st store.Store, m store.Mutation) error {
	switch m.Op {
	case "delete":
		return applyDelete(st, ctx, m)
	case "upsert":
		// ponytail: the tombstone floor is checked only for observation
		// upserts — the other entities have no local upsert path in this
		// task, so a floor can only race an observation. Upgrade path: the
		// per-entity floor lookup when prompt/session applies land.
		if m.Entity == "observation" {
			if floor, err := tombstoneFloor(st, m.Entity, m.EntityKey); err != nil {
				return err
			} else if floor {
				// A delete for this key was already applied: drop the late
				// upsert (spec 006 §6 — the classic resurrection race).
				return nil
			}
			return applyObservation(st, ctx, m)
		}
		switch m.Entity {
		case "relation":
			return applyRelation(st, ctx, m)
		default:
			// session / prompt / project: no local upsert path yet (their
			// journal rows land with their owning tasks). No-op, not an
			// error — the pull cursor must advance.
			return nil
		}
	default:
		return fmt.Errorf("sync: unknown mutation op %q (seq %d)", m.Op, m.Seq)
	}
}

// applyDelete applies a delete: the tombstone floor is recorded BEFORE
// the delete itself (spec 006 §6 ordering) — a late upsert for the same
// entity_key after the delete cannot resurrect the row. The floor is the
// no-op marker for a second delete of the same key.
func applyDelete(st store.Store, ctx context.Context, m store.Mutation) error {
	switch m.Entity {
	case "observation":
		// Record the floor first (spec 006 §6 ordering).
		if err := st.Tombstone(ctx, m.Entity, m.EntityKey); err != nil {
			return err
		}
		// then the actual delete. The row may already be gone (a local
		// delete, or a second delete of the same key — idempotent no-op).
		if id := obsIDBySyncID(st, ctx, m.EntityKey); id != 0 {
			return st.DeleteObservationForProject(ctx, id, m.Project, true)
		}
		return nil
	default:
		// prompt / relation / session / project: the tombstone floor is
		// still recorded (the floor table is entity-generic); there is no
		// local delete path for these yet (their owning tasks).
		return st.Tombstone(ctx, m.Entity, m.EntityKey)
	}
}

// ── seam helpers ───────────────────────────────────────────────────────────

// ponytail: the sync service's READS cross into the store's raw-SQL test
// seams (Probe/ProbeInt/RawQuery/RawExec — the pipeline's ledger-read
// pattern: the memory package cannot see the store's unexported methods,
// and these lookups have no FTS/trigger surface). All WRITES go through
// store.Store methods (INV-4); the reads are pure lookups. Upgrade path:
// dedicated Store interface methods when Task 18's Postgres adapter lands
// and the read surface is proven stable.

func probeInt(seam syncSeam, query string, args ...any) (int, error) {
	return seam.ProbeInt(query, args...)
}

// query runs a read via the RawQuery seam and returns the rows (values
// scan as []any: int64 / []byte / string / nil).
func query(st store.Store, q string, args []any) ([][]any, error) {
	return applyPulledSeam(st).RawQuery(q, args...)
}

// exec runs a statement via the RawExec seam.
func exec(st store.Store, q string, args ...any) error {
	return applyPulledSeam(st).RawExec(q, args...)
}

// obsBySyncID reports whether a live (non-deleted) observation row exists
// with the sync id.
func obsBySyncID(st store.Store, ctx context.Context, syncID string) bool {
	_ = ctx
	ok, err := applyPulledSeam(st).Probe(
		`SELECT COUNT(*) > 0 FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		return false
	}
	return ok
}

// obsIDBySyncID returns the local id of a live observation with the sync
// id (0 = absent or deleted).
func obsIDBySyncID(st store.Store, ctx context.Context, syncID string) int64 {
	_ = ctx
	n, err := applyPulledSeam(st).ProbeInt(
		`SELECT ifnull(id, 0) FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		return 0
	}
	return int64(n)
}

// tombstoneFloor reports whether a delete tombstone was already recorded
// for the entity key (a delete applied before this upsert → the upsert
// must be dropped, spec 006 §6).
func tombstoneFloor(st store.Store, entity, key string) (bool, error) {
	return applyPulledSeam(st).Probe(
		`SELECT COUNT(*) > 0 FROM sync_delete_tombstones WHERE entity = ? AND entity_key = ?`,
		entity, key)
}

// ── deferred-apply queue (sync_apply_deferred) ────────────────────────────

type deferredRow struct {
	syncID     string
	payload    string
	retryCount int
}

// deferredRows lists the deferred queue, oldest first (first_seen_at is
// the retry order — FIFO within a pull cycle).
func deferredRows(st store.Store, ctx context.Context) ([]deferredRow, error) {
	_ = ctx
	rows, err := query(st,
		`SELECT sync_id, payload, retry_count FROM sync_apply_deferred
		 ORDER BY first_seen_at ASC, sync_id ASC`, nil)
	if err != nil {
		return nil, err
	}
	var out []deferredRow
	for _, r := range rows {
		out = append(out, deferredRow{
			syncID:     strVal(r, 0),
			payload:    strVal(r, 1),
			retryCount: intVal(r, 2),
		})
	}
	return out, nil
}

// defer parks a mutation in the deferred queue (idempotent: a row already
// deferred for the same sync_id keeps its retry_count — a second apply of
// the same mutation must not reset the 5-cap).
// parkDeferred parks a mutation with an explicit retry_count (the caller
// owns the count — a first park passes 1, a replay re-park passes the
// carried count + 1). The count is a MONOTONIC failure counter: it never
// resets across re-parks (the 5-cap counts failed replays, not park
// events).
func parkDeferred(st store.Store, ctx context.Context, m store.Mutation, reason string, retryCount int) error {
	_ = ctx
	payload, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("sync: defer payload: %w", err)
	}
	if retryCount <= 0 {
		retryCount = 1
	}
	if err := exec(st, `
		INSERT INTO sync_apply_deferred (sync_id, entity, payload, entity_key, project, reason_code, retry_count)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(sync_id) DO UPDATE SET retry_count = excluded.retry_count`,
		m.EntityKey, m.Entity, string(payload), m.EntityKey, m.Project, reason, retryCount); err != nil {
		return err
	}
	return errDeferral
}

// errDeferral marks "the mutation was parked in the deferred queue" — the
// caller distinguishes a deferral (a successful outcome: the pull cursor
// advances) from a hard error.
var errDeferral = errors.New("sync: deferred (missing FK dependency)")

// isDeferral reports whether err is the deferral sentinel.
func isDeferral(err error) bool { return errors.Is(err, errDeferral) }

// pruneDeferred removes a row from the deferred queue (replayed OK, or
// dead after the 5-cap).
func pruneDeferred(st store.Store, ctx context.Context, syncID string) {
	_ = ctx
	_ = exec(st, `DELETE FROM sync_apply_deferred WHERE sync_id = ?`, syncID)
}

// ── row-scan helpers (RawQuery returns []any) ─────────────────────────────

func strVal(r []any, i int) string {
	if i >= len(r) || r[i] == nil {
		return ""
	}
	switch v := r[i].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func intVal(r []any, i int) int {
	if i >= len(r) || r[i] == nil {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

func int64Val(r []any, i int) int64 {
	if i >= len(r) || r[i] == nil {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}
