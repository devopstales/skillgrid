package memory

import (
	"context"
	"testing"
)

// TestTripleStoreOrphanDetection covers @step-07: CheckCrossLinkIntegrity
// reports observations whose graph_ref points at a symbol that no longer
// exists, and lets valid references pass.
func TestTripleStoreOrphanDetection(t *testing.T) {
	fx := newFixture(t, "orphancheck")
	ctx := context.Background()
	db := fx.st.DB

	symbolID := seedSymbolFile(t, db, "/tmp/orphan.go", "orphanFunc", "uid-orphan-1")

	// A valid reference (graph_ref resolves to a live symbol).
	obsValid, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "valid ref note",
		Content:   "observation bound to a live symbol",
		Source:    "/tmp/orphan.go",
	})
	if err != nil {
		t.Fatalf("save valid: %v", err)
	}

	// An orphan reference: point graph_ref at a symbol id that does not exist.
	_, err = db.Exec(`UPDATE observations SET graph_ref = 987654 WHERE id = ?`, obsValid)
	if err != nil {
		t.Fatalf("set orphan graph_ref: %v", err)
	}
	// And a second observation still holding the valid reference.
	obsGood, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "good ref note",
		Content:   "observation bound to the same live symbol",
		Source:    "/tmp/orphan.go",
	})
	if err != nil {
		t.Fatalf("save good: %v", err)
	}

	orphans, err := fx.svc.CheckCrossLinkIntegrity(ctx)
	if err != nil {
		t.Fatalf("integrity check: %v", err)
	}
	if len(orphans) != 1 || orphans[0].ID != obsValid || orphans[0].GraphRef != 987654 {
		t.Fatalf("orphans=%+v want exactly the 987654 reference (obs %d)", orphans, obsValid)
	}
	for _, o := range orphans {
		if o.ID == obsGood {
			t.Fatalf("valid reference (obs %d, symbol %d) reported as orphan", obsGood, symbolID)
		}
	}
}
