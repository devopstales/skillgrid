package memory

import (
	"errors"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// ledgerCount counts observation_save_operations rows (probe-seam assertion
// on the ledger table the gate inserts into).
func ledgerCount(t *testing.T, st store.Store) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT COUNT(*) FROM observation_save_operations`)
	if err != nil {
		t.Fatalf("ledger count: %v", err)
	}
	return n
}

// [replay-identical-returns-committed-id] a replay of an identical save bound
// to an operation id returns the committed id and nothing is rewritten: the
// replay path returns early — no second upsert, no second mutation, no
// second ledger row.
func TestReplayIdenticalReturnsCommittedID(t *testing.T) {
	svc := newTestService(t)
	req := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project", OperationID: "op-1"}
	r1, err := svc.Save(ctx, req)
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	r2, err := svc.Save(ctx, req)
	if err != nil {
		t.Fatalf("save 2: %v", err)
	}
	if !r2.Replayed {
		t.Fatalf("identical replay must set Replayed: %+v", r2)
	}
	if r2.ID != r1.ID || r2.SyncID != r1.SyncID {
		t.Fatalf("identical replay must return committed id, not rewrite: %+v vs %+v", r1, r2)
	}
	// nothing is rewritten: one observation row, one ledger row.
	if n := liveObsCount(t, svc.st); n != 1 {
		t.Fatalf("replay must not write a new observation row; live count = %d", n)
	}
	if n := ledgerCount(t, svc.st); n != 1 {
		t.Fatalf("replay must not write a second ledger row; ledger count = %d", n)
	}
}

// [replay-changed-payload-409] the same operation id with a changed payload
// is refused with ErrOperationConflict (409) and nothing is written.
func TestReplayChangedPayloadConflicts(t *testing.T) {
	svc := newTestService(t)
	base := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project", OperationID: "op-2"}
	r1, err := svc.Save(ctx, base)
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	changed := base
	changed.Content = "C2"
	_, err = svc.Save(ctx, changed)
	if !errors.Is(err, ErrOperationConflict) {
		t.Fatalf("want 409 conflict, got %v", err)
	}
	// the conflict must not have touched the committed row or the ledger.
	if n := liveObsCount(t, svc.st); n != 1 {
		t.Fatalf("conflict must not write a new row; live count = %d", n)
	}
	if n := ledgerCount(t, svc.st); n != 1 {
		t.Fatalf("conflict must not write a second ledger row; ledger count = %d", n)
	}
	if got, _ := svc.st.GetObservation(ctx, r1.ID, false); got.Content != "C" {
		t.Fatalf("conflict must not rewrite the committed row: %q", got.Content)
	}
}

// [replay-deleted-target-410] a hard-deleted target leaves its ledger row as
// a tombstone (observation_id SET NULL); replaying the operation id is
// refused with ErrOperationExpired (410).
func TestReplayDeletedTargetExpired(t *testing.T) {
	svc := newTestService(t)
	req := SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C", Project: "p", Scope: "project", OperationID: "op-3"}
	r1, err := svc.Save(ctx, req)
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	// hard delete → the ledger FK is ON DELETE SET NULL: the row survives
	// with observation_id NULL (Task 1's DDL).
	if err := svc.st.DeleteObservationForProject(ctx, r1.ID, "p", true); err != nil {
		t.Fatalf("hard delete: %v", err)
	}
	_, err = svc.Save(ctx, req)
	if !errors.Is(err, ErrOperationExpired) {
		t.Fatalf("want 410 expired, got %v", err)
	}
	// the tombstone is intact and nothing was re-inserted.
	if n := liveObsCount(t, svc.st); n != 0 {
		t.Fatalf("expired replay must not re-insert; live count = %d", n)
	}
	if n := ledgerCount(t, svc.st); n != 1 {
		t.Fatalf("expired replay must not write a second ledger row; ledger count = %d", n)
	}
}

// Triangulation 1 — a non-replayed (first) save with an operation id writes
// exactly one ledger row and returns Replayed:false; a save with no
// operation id writes none (different inputs: no-op-id branch).
func TestSaveOperationIDWritesOneLedgerRow(t *testing.T) {
	svc := newTestService(t)
	// first save with an operation id: one row, not a replay.
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T", Content: "C",
		Project: "p", Scope: "project", OperationID: "op-ledger"})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if r.Replayed {
		t.Fatalf("first save must not be a replay: %+v", r)
	}
	if n := ledgerCount(t, svc.st); n != 1 {
		t.Fatalf("first save with operation id must write exactly one ledger row; got %d", n)
	}
	// no operation id → no ledger row.
	if _, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "T2", Content: "C2",
		Project: "p", Scope: "project"}); err != nil {
		t.Fatalf("save without op id: %v", err)
	}
	if n := ledgerCount(t, svc.st); n != 1 {
		t.Fatalf("save without operation id must not write a ledger row; got %d", n)
	}
}

// Triangulation 2 — a replay differing only inside a <private> span is an
// identical replay, not a conflict: the fingerprint covers the REDACTED+
// truncated content (spec 01-memory/001 §4). The committed fingerprint is
// pinned against the spec's "v1:" + length-prefixed normalization
// (different inputs than TestReplayIdenticalReturnsCommittedID: redaction
// active, a second operation id).
func TestReplayIdenticalAfterRedactionIsNotConflict(t *testing.T) {
	svc := newTestServiceWith(t, config.MemoryCfg{PrivateTag: "<private>"})
	base := SaveRequest{SessionID: "s", Type: "decision", Title: "T",
		Content: "alpha <private>secret</private> beta", Project: "p", Scope: "project", OperationID: "op-redact"}
	r1, err := svc.Save(ctx, base)
	if err != nil {
		t.Fatalf("save 1: %v", err)
	}
	replay := base
	replay.Content = "alpha <private>different-secret</private> beta"
	r2, err := svc.Save(ctx, replay)
	if err != nil {
		t.Fatalf("redaction-identical replay must not conflict: %v", err)
	}
	if !r2.Replayed || r2.ID != r1.ID {
		t.Fatalf("redaction-identical replay must return the committed id: %+v vs %+v", r1, r2)
	}
	// pin the stored fingerprint: the gate stores "v1:" + the normalized
	// content hash (storeNormalizedHash over the redacted+truncated
	// content). A normalization drift breaks replays loudly here.
	storedFingerprintMustEqual(t, svc.st, "op-redact",
		"v1:"+storeNormalizedHash("alpha beta"))
}

// storedFingerprintMustEqual asserts the ledger row's stored fingerprint
// equals want (the store's test seams return bool/int only, so the read is
// an equality probe).
func storedFingerprintMustEqual(t *testing.T, st store.Store, operationID, want string) {
	t.Helper()
	ok, err := probeStore(t, st).Probe(
		`SELECT fingerprint = ? FROM observation_save_operations WHERE operation_id = ?`,
		want, operationID)
	if err != nil {
		t.Fatalf("fingerprint probe: %v", err)
	}
	if !ok {
		t.Fatalf("stored fingerprint for %q != the recomputed %q", operationID, want)
	}
}
