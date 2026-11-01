package memory

// Task 15 — sync journal core (mutation journal + pull-apply seam).
//
// The four acceptance scenarios (acceptance.feature, G13):
//
//	[pull-applies-through-store]    — a pulled upsert lands on B through the
//	                                  same store path a local write uses: B's
//	                                  FTS index serves it (a trigger fired).
//	[tombstone-floor-no-resurrect] — an applied upsert + applied delete, then
//	                                  a LATE upsert for the same entity: the
//	                                  row does not come back.
//	[deferred-apply-5-cap]         — a pulled relation whose target has not
//	                                  been pulled defers; 5 failed replays →
//	                                  the row dies and is pruned.
//	[enrollment-gates-journal]     — an unenrolled project journals nothing;
//	                                  after Enroll, the next save journals one.
//
// The apply seam is SyncService (internal/memory/sync.go); the store seam is
// store.Store (Task 1's interface) + the SyncSeam test hooks.

import (
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// syncStore narrows a store to the sync test seam (store-only assertions; a
// store without the seam is a test bug — the SQLite adapter is the only one
// Task 15 targets).
func syncStore(t *testing.T, st store.Store) store.SyncSeam {
	t.Helper()
	s, ok := st.(store.SyncSeam)
	if !ok {
		t.Fatal("store does not expose the SyncSeam test hooks")
	}
	return s
}

// journalCount counts un-acked (pending) sync_mutations rows.
func journalCount(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM sync_mutations WHERE acked_at IS NULL`)
	if err != nil {
		t.Fatalf("journal count: %v", err)
	}
	return n
}

// pulledObsPayload builds a canonical observation upsert payload (spec 006
// §4 — the project-canonicalized entity) for syncID under sessionID.
func pulledObsPayload(syncID, sessionID, title, content string) string {
	return `{"sync_id":"` + syncID + `","session_id":"` + sessionID + `","type":"decision","title":"` + title +
		`","content":"` + content + `","project":"p","scope":"project"}`
}

// seedRelationMutationPayload builds a relation-mutation payload whose
// endpoints resolve to the given observations' sync ids (obsA → source,
// obsB → target), with relKey as the relation's sync id.
func relationMutationPayload(t *testing.T, relKey string, obsA, obsB store.Observation) string {
	t.Helper()
	return `{"sync_id":"` + relKey + `","source_id":"` + obsA.SyncID + `","target_id":"` + obsB.SyncID + `","relation":"extends"}`
}

// ────────────────────────────────────────────────────────────────────────────
// [pull-applies-through-store]
// ────────────────────────────────────────────────────────────────────────────

// TestPullAppliesThroughStore: machine A saves a session + observation and
// journals them; machine B pulls A's journal (direct ListPendingMutations —
// the transport is Task 16) and applies each mutation. B's store contains
// the observation AND its full-text index serves it (the FTS insert trigger
// fired — the apply went through the store, not raw SQL).
func TestPullAppliesThroughStore(t *testing.T) {
	local := openTestStoreIn(t, t.TempDir(), "p")  // machine A
	remote := openTestStoreIn(t, t.TempDir(), "p") // machine B
	syncStore(t, local).SetSyncEnabled(true)
	syncStore(t, local).SetEnrolledForTest("p")
	svcA := newTestServiceWithStore(t, local)

	// A saves a session + observation (the session auto-registers on save;
	// the journal carries the observation upsert — spec 006 §4).
	res, err := svcA.Save(ctx, SaveRequest{SessionID: "sa", Type: "decision", Title: "Alpha decision",
		Content: "The zebra payload", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("A save: %v", err)
	}
	if !isLive(t, local, res.SyncID) {
		t.Fatalf("A's save must be live on A: %s", res.SyncID)
	}

	// B pulls A's journal (the transport is Task 16 — here: direct).
	muts, err := local.ListPendingMutations(ctx, 0, 100)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(muts) == 0 {
		t.Fatalf("A must have journaled the save")
	}
	svcB := NewSyncService(remote, syncCfg(true))
	for _, m := range muts {
		if err := svcB.ApplyPulled(ctx, m); err != nil {
			t.Fatalf("B apply pulled %s: %v", m.EntityKey, err)
		}
	}

	// B contains the observation by its sync id.
	id := findIDBySyncID(t, remote, res.SyncID)
	if id == 0 {
		t.Fatalf("pulled observation not present on B: %s", res.SyncID)
	}
	got, err := remote.GetObservation(ctx, id, false)
	if err != nil {
		t.Fatalf("B get observation: %v", err)
	}
	if got.Title != "Alpha decision" || got.Content != "The zebra payload" {
		t.Fatalf("pulled observation wrong on B: %+v", got)
	}

	// INV-4 proof: B's FTS index saw it — the insert trigger fired because
	// the apply went through the store's write path, not raw SQL.
	r, err := remote.Search(ctx, store.SearchQuery{Query: "zebra", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("B search: %v", err)
	}
	if len(r.Observations) != 1 {
		t.Fatalf("FTS trigger did not fire on pulled apply: got %d hits, want 1", len(r.Observations))
	}
}

// TestPullApplyIdempotent (TRIANGULATE [pull-idempotent]): applying the same
// remote mutation twice has exactly one effect — one live row on B, no
// second row, no error.
func TestPullApplyIdempotent(t *testing.T) {
	remote := openTestStoreIn(t, t.TempDir(), "p")
	svc := NewSyncService(remote, syncCfg(true))

	m := store.Mutation{TargetKey: "local", Entity: "observation", EntityKey: "obs-idem-1", Op: "upsert",
		Payload: pulledObsPayload("obs-idem-1", "s", "T idem", "content idem"),
		Source:  "remote", Project: "p"}
	if err := svc.ApplyPulled(ctx, m); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := svc.ApplyPulled(ctx, m); err != nil {
		t.Fatalf("second apply must be a no-op, not an error: %v", err)
	}
	if n := liveObsCount(t, remote); n != 1 {
		t.Fatalf("double apply must leave exactly one row; live count = %d", n)
	}
	if n := journalCount(t, remote); n != 0 {
		t.Fatalf("a pulled apply must not re-enqueue locally (append-only outbox); journal = %d", n)
	}
}

// TestPullOutboxAppendOnly (TRIANGULATE [journal-append-only]): B's journal
// is a pure outbox of B's own writes. Pulling + applying A's mutations
// neither appends to B's journal nor rewrites what is already there.
func TestPullOutboxAppendOnly(t *testing.T) {
	local := openTestStoreIn(t, t.TempDir(), "p")  // A
	remote := openTestStoreIn(t, t.TempDir(), "p") // B
	syncStore(t, local).SetSyncEnabled(true)
	syncStore(t, remote).SetSyncEnabled(true)
	syncStore(t, remote).SetEnrolledForTest("p")

	svcB := newTestServiceWithStore(t, remote)
	svcA := NewSyncService(local, syncCfg(true))
	_ = svcA

	// B journals its own save first (a row already in the outbox).
	resB, err := svcB.Save(ctx, SaveRequest{SessionID: "sb", Type: "decision", Title: "Beta own",
		Content: "owned by beta", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatalf("B save: %v", err)
	}
	before, err := local.ListPendingMutations(ctx, 0, 100)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	_ = before
	// A journals a save too.
	svcA2 := newTestServiceWithStore(t, local)
	if _, err := svcA2.Save(ctx, SaveRequest{SessionID: "sa", Type: "decision", Title: "Alpha own",
		Content: "owned by alpha", Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("A save: %v", err)
	}

	// B pulls A's journal and applies it.
	muts, err := local.ListPendingMutations(ctx, 0, 100)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	svcB2 := NewSyncService(remote, syncCfg(true))
	for _, m := range muts {
		if m.Source == "local" && m.TargetKey != "local" {
			continue
		}
		if err := svcB2.ApplyPulled(ctx, m); err != nil {
			t.Fatalf("B apply %s: %v", m.EntityKey, err)
		}
	}

	// B's journal: exactly its own save — one row, untouched by the pull.
	if n := journalCount(t, remote); n != 1 {
		t.Fatalf("pull must not touch B's outbox; B's journal = %d, want 1", n)
	}
	if !isLive(t, remote, resB.SyncID) {
		t.Fatalf("B's own row must still be live: %s", resB.SyncID)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [tombstone-floor-no-resurrect]
// ────────────────────────────────────────────────────────────────────────────

// TestTombstoneFloorPreventsResurrect: upsert then delete, in order; a LATE
// upsert for the same entity key arrives — the row does not come back
// (spec 006 §6: the tombstone floor is recorded BEFORE the delete, and the
// late upsert for the same key is dropped).
func TestTombstoneFloorPreventsResurrect(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "sess", "session row", "") // auto-registers session s
	svc := NewSyncService(st, syncCfg(true))

	u := upsertMutation("obs-1")
	d := deleteMutation("obs-1")
	if err := svc.ApplyPulled(ctx, u); err != nil {
		t.Fatalf("apply upsert: %v", err)
	}
	if !isLive(t, st, "obs-1") {
		t.Fatalf("upsert must create a live row")
	}
	if err := svc.ApplyDelete(ctx, d); err != nil {
		t.Fatalf("apply delete: %v", err)
	}
	if isLive(t, st, "obs-1") {
		t.Fatalf("delete must remove the row")
	}
	// the late upsert (same key, as if the delete raced it on the wire).
	if err := svc.ApplyPulled(ctx, u); err != nil {
		t.Fatalf("apply late upsert: %v", err)
	}
	if isLive(t, st, "obs-1") {
		t.Fatalf("late upsert resurrected a tombstoned row")
	}
}

// TestTombstoneFloorAllowsNewKey (TRIANGULATE, different inputs): the floor
// is per entity key — a delete on obs-1 must not block a NEW observation
// that happens to arrive after it, and a fresh upsert for a key that was
// never tombstoned lands normally.
func TestTombstoneFloorAllowsNewKey(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "sess", "session row", "") // auto-registers session s
	svc := NewSyncService(st, syncCfg(true))

	if err := svc.ApplyPulled(ctx, upsertMutation("obs-a")); err != nil {
		t.Fatalf("apply upsert obs-a: %v", err)
	}
	if err := svc.ApplyDelete(ctx, deleteMutation("obs-a")); err != nil {
		t.Fatalf("apply delete obs-a: %v", err)
	}
	// a DIFFERENT key applies normally after the floor exists.
	if err := svc.ApplyPulled(ctx, upsertMutation("obs-b")); err != nil {
		t.Fatalf("apply upsert obs-b: %v", err)
	}
	if !isLive(t, st, "obs-b") {
		t.Fatalf("a different key must not be blocked by obs-a's floor")
	}
	if isLive(t, st, "obs-a") {
		t.Fatalf("obs-a must stay deleted")
	}
}

// TestPullDeleteIdempotent (TRIANGULATE [pull-idempotent], delete path):
// applying the same remote delete twice has one effect — the tombstone is
// the no-op marker, and a later upsert of the same id stays dropped.
func TestPullDeleteIdempotent(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "sess", "session row", "") // auto-registers session s
	svc := NewSyncService(st, syncCfg(true))

	u := upsertMutation("obs-del-1")
	d := deleteMutation("obs-del-1")
	for _, m := range []store.Mutation{u, d, d} {
		if err := svc.ApplyPulled(ctx, m); err != nil {
			t.Fatalf("apply %s: %v", m.Op, err)
		}
	}
	if isLive(t, st, "obs-del-1") {
		t.Fatalf("double delete must leave the row deleted")
	}
	// a later upsert of the same id is still a no-op.
	if err := svc.ApplyPulled(ctx, u); err != nil {
		t.Fatalf("apply late upsert: %v", err)
	}
	if isLive(t, st, "obs-del-1") {
		t.Fatalf("upsert after double delete must not resurrect")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [deferred-apply-5-cap]
// ────────────────────────────────────────────────────────────────────────────

// TestDeferredApplyRetriesThenDies: a pulled relation whose target
// observation has not been pulled yet goes to sync_apply_deferred; after
// 5 failed replays the row dies and is pruned (spec 006 §6).
func TestDeferredApplyRetriesThenDies(t *testing.T) {
	st := openTestStore(t)
	svc := NewSyncService(st, syncCfg(true))

	// the source observation is present; the target is not.
	src := seedObservation(t, st, "p", "decision", "src", "source content", "")
	targetMutation := store.Mutation{TargetKey: "local", Entity: "relation", EntityKey: "rel-defer-1", Op: "upsert",
		Payload: relationMutationPayload(t, "rel-defer-1", src, store.Observation{SyncID: "obs-new"}),
		Source:  "remote", Project: "p"}
	if err := svc.ApplyPulled(ctx, targetMutation); err != nil {
		t.Fatalf("apply relation: %v", err)
	}
	if n := deferredCount(t, st); n != 1 {
		t.Fatalf("missing-FK mutation must be deferred, got %d", n)
	}
	// 5 failed replays → dead + pruned.
	for i := 0; i < 5; i++ {
		svc.ReplayDeferred(ctx)
	}
	if n := deferredCount(t, st); n != 0 {
		t.Fatalf("dead deferred rows must be pruned after 5 retries, still %d", n)
	}
}

// TestDeferredApplyReplaysWhenTargetArrives (TRIANGULATE, different inputs):
// the SAME deferred relation lands — through the deferred-replay path, not
// a fresh apply — once the missing target observation is pulled. The
// relation is created exactly once, and the deferred queue drains.
func TestDeferredApplyReplaysWhenTargetArrives(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "sess", "session row", "") // auto-registers session s
	svc := NewSyncService(st, syncCfg(true))

	src := seedObservation(t, st, "p", "decision", "src2", "source content two", "")
	m := store.Mutation{TargetKey: "local", Entity: "relation", EntityKey: "rel-replay-1", Op: "upsert",
		Payload: relationMutationPayload(t, "rel-replay-1", src, store.Observation{SyncID: "obs-late"}),
		Source:  "remote", Project: "p"}
	if err := svc.ApplyPulled(ctx, m); err != nil {
		t.Fatalf("apply relation: %v", err)
	}
	if n := deferredCount(t, st); n != 1 {
		t.Fatalf("must defer, got %d", n)
	}
	// one failed replay (target still missing).
	svc.ReplayDeferred(ctx)
	if n := deferredCount(t, st); n != 1 {
		t.Fatalf("one failed replay must keep the row deferred, got %d", n)
	}
	// the target arrives via its own pulled upsert.
	if err := svc.ApplyPulled(ctx, upsertMutation("obs-late")); err != nil {
		t.Fatalf("apply target upsert: %v", err)
	}
	// the next replay lands the relation (idempotent: one row).
	svc.ReplayDeferred(ctx)
	svc.ReplayDeferred(ctx)
	if n := deferredCount(t, st); n != 0 {
		t.Fatalf("replayed relation must drain the deferred queue, got %d", n)
	}
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM memory_relations WHERE sync_id = 'rel-replay-1'`)
	if err != nil {
		t.Fatalf("relation probe: %v", err)
	}
	if n != 1 {
		t.Fatalf("replayed relation must land exactly once, got %d rows", n)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// [enrollment-gates-journal]
// ────────────────────────────────────────────────────────────────────────────

// TestEnrollmentGatesJournal: an unenrolled project's save journals nothing;
// after Enroll, the next save journals exactly one remote mutation
// (spec 006 §4: only enrolled projects generate remote journal entries).
func TestEnrollmentGatesJournal(t *testing.T) {
	st := openTestStore(t)
	syncStore(t, st).SetSyncEnabled(true)
	svc := newTestServiceWithStore(t, st)

	// unenrolled: no journal row.
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save unenrolled: %v", err)
	}
	if n := journalCount(t, st); n != 0 {
		t.Fatalf("unenrolled project must not enqueue remote mutations, got %d", n)
	}

	// enrolled: exactly one.
	syncSvc := NewSyncService(st, syncCfg(true))
	if err := syncSvc.Enroll(ctx, "p", true); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T2", Content: "C2",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save enrolled: %v", err)
	}
	if n := journalCount(t, st); n != 1 {
		t.Fatalf("enrolled project must enqueue exactly one, got %d", n)
	}
}

// TestEnrollmentReversible (TRIANGULATE, different inputs): Enroll(on=false)
// stops the journal again — the gate is a live lookup, not a latch.
func TestEnrollmentReversible(t *testing.T) {
	st := openTestStore(t)
	syncStore(t, st).SetSyncEnabled(true)
	svc := newTestServiceWithStore(t, st)
	syncSvc := NewSyncService(st, syncCfg(true))

	if err := syncSvc.Enroll(ctx, "p", true); err != nil {
		t.Fatalf("enroll on: %v", err)
	}
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := syncSvc.Enroll(ctx, "p", false); err != nil {
		t.Fatalf("enroll off: %v", err)
	}
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T3", Content: "C3",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save off: %v", err)
	}
	if n := journalCount(t, st); n != 1 {
		t.Fatalf("unenroll must stop the journal: got %d rows, want 1 (only the first save)", n)
	}
}

// ────────────────────────────────────────────────────────────────────────────
// Status (spec 006 §7 — per-target lifecycle + seqs + backoff)
// ────────────────────────────────────────────────────────────────────────────

// TestStoreApplyPulledMutationReal: the store's ApplyPulledMutation seam
// (Task 1's interface) is a real method — not the ErrNotImplemented stub —
// and routes to the same behavior the service exposes: an observation
// upsert lands (the FTS trigger fired), a delete tombstones + removes the
// row, and a second identical apply is a no-op (idempotent).
func TestStoreApplyPulledMutationReal(t *testing.T) {
	st := openTestStore(t)
	seedObservation(t, st, "p", "decision", "sess", "session row", "") // auto-registers session s
	m := store.Mutation{TargetKey: "local", Entity: "observation", EntityKey: "obs-store-1", Op: "upsert",
		Payload: pulledObsPayload("obs-store-1", "s", "T store", "content store"),
		Source:  "remote", Project: "p"}
	if err := st.ApplyPulledMutation(ctx, m); err != nil {
		t.Fatalf("store apply upsert: %v", err)
	}
	if !isLive(t, st, "obs-store-1") {
		t.Fatalf("store-applied upsert must be live")
	}
	r, err := st.Search(ctx, store.SearchQuery{Query: "store", Project: "p", Limit: 5})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(r.Observations) != 1 {
		t.Fatalf("FTS trigger must fire on store apply, got %d", len(r.Observations))
	}
	d := deleteMutation("obs-store-1")
	if err := st.ApplyPulledMutation(ctx, d); err != nil {
		t.Fatalf("store apply delete: %v", err)
	}
	if isLive(t, st, "obs-store-1") {
		t.Fatalf("store-applied delete must remove the row")
	}
	// idempotent: a second identical apply of either op is a no-op.
	if err := st.ApplyPulledMutation(ctx, m); err != nil {
		t.Fatalf("second upsert apply must be a no-op: %v", err)
	}
	if isLive(t, st, "obs-store-1") {
		t.Fatalf("second upsert after delete must not resurrect (tombstone floor)")
	}
	if err := st.ApplyPulledMutation(ctx, d); err != nil {
		t.Fatalf("second delete apply must be a no-op: %v", err)
	}
}

// TestStatusReportsTargetLifecycle: after a pull cycle, sync_state carries
// the target's lifecycle and last_pulled_seq (the `mnemonic sync status`
// read surface).
func TestStatusReportsTargetLifecycle(t *testing.T) {
	st := openTestStore(t)
	svc := NewSyncService(st, syncCfg(true))

	m := store.Mutation{TargetKey: "cloud", Entity: "observation", EntityKey: "obs-status-1", Op: "upsert",
		Payload: pulledObsPayload("obs-status-1", "s", "T status", "content status"),
		Source:  "remote", Project: "p", Seq: 41}
	if err := svc.ApplyPulled(ctx, m); err != nil {
		t.Fatalf("apply: %v", err)
	}
	status, err := svc.Status(ctx)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	target, ok := status["cloud"]
	if !ok {
		t.Fatalf("status must report the pulled target, got %v", status)
	}
	if target.LastPulled != 41 {
		t.Fatalf("last_pulled_seq = %d, want 41", target.LastPulled)
	}
	if target.Lifecycle == "" {
		t.Fatalf("lifecycle must be non-empty: %+v", target)
	}
}

// TestNewSyncServiceGatesJournal (TRIANGULATE [enrollment-gates-journal],
// different inputs): NewSyncService wires the sync gate from config — with
// Sync.Enabled=true the store's journaling is on (a save journals one);
// with Sync.Enabled=false a save journals nothing even though the project
// is enrolled (the gate is the config switch, not a latch the caller must
// flip by hand).
func TestNewSyncServiceGatesJournal(t *testing.T) {
	on := openTestStore(t)
	_ = NewSyncService(on, syncCfg(true))
	syncStore(t, on).SetEnrolledForTest("p")
	svcOn := newTestServiceWithStore(t, on)
	if _, err := svcOn.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save enabled: %v", err)
	}
	if n := journalCount(t, on); n != 1 {
		t.Fatalf("enabled sync must journal the save, got %d", n)
	}

	off := openTestStore(t)
	_ = NewSyncService(off, syncCfg(false))
	syncStore(t, off).SetEnrolledForTest("p")
	svcOff := newTestServiceWithStore(t, off)
	if _, err := svcOff.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save disabled: %v", err)
	}
	if n := journalCount(t, off); n != 0 {
		t.Fatalf("disabled sync must not journal, got %d", n)
	}
}
