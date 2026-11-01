package memory

// judge_test.go — [judge-sets-provenance], [judge-many-one-write],
// [compare-idempotent] (acceptance.feature), plus the brief's
// triangulation inputs (already-judged idempotence, batch bounds).

import (
	"fmt"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// [judge-sets-provenance] a pending candidate resolved through Judge comes
// back judged and carries the judging actor + model provenance (INV-1).
func TestJudgeSetsProvenance(t *testing.T) {
	svc := newTestService(t)
	cand := seedPendingRelation(t, svc.st, "p", testSession)
	rel, err := svc.Judge(ctx, JudgeRequest{JudgmentID: cand.ID, Relation: "conflicts_with",
		Reason: "mutually exclusive", Evidence: "see diff", Confidence: 0.9})
	if err != nil {
		t.Fatal(err)
	}
	if rel.JudgmentStatus != "judged" || rel.Relation != "conflicts_with" {
		t.Fatalf("not judged: %+v", rel)
	}
	if rel.Reason != "mutually exclusive" || rel.Evidence != "see diff" || rel.Confidence != 0.9 {
		t.Fatalf("verdict fields not stored: %+v", rel)
	}
	// INV-1 provenance: the judged row carries the judging actor (agent)
	// and kind (mcp); marked_by_model is "" when the agent sends no model
	// name (the brief's JudgeRequest carries no model field — the
	// transport layer, Task 13, maps the agent's model name onto the
	// JudgeParams when it is present).
	if rel.MarkedByActor != "agent" || rel.MarkedByKind != "mcp" {
		t.Fatalf("provenance not set: %+v", rel)
	}
}

// (TRIANGULATE, different inputs: idempotence) judging the same candidate a
// second time is a no-op: it returns the existing judged row untouched
// (no second write, no field churn).
func TestJudgeIdempotentWhenAlreadyJudged(t *testing.T) {
	svc := newTestService(t)
	cand := seedPendingRelation(t, svc.st, "p", testSession)
	_, err := svc.Judge(ctx, JudgeRequest{JudgmentID: cand.ID, Relation: "conflicts_with",
		Reason: "first", Evidence: "e1", Confidence: 0.9})
	if err != nil {
		t.Fatal(err)
	}
	counts := txnCount(t, svc.st)
	rel, err := svc.Judge(ctx, JudgeRequest{JudgmentID: cand.ID, Relation: "compatible",
		Reason: "second verdict", Evidence: "e2", Confidence: 0.1})
	if err != nil {
		t.Fatal(err)
	}
	if *counts != 0 {
		t.Fatalf("re-judge must be a no-op write, txns %d", *counts)
	}
	if rel.Relation != "conflicts_with" || rel.Reason != "first" {
		t.Fatalf("re-judge must return the existing row untouched: %+v", rel)
	}
}

// [judge-many-one-write] five judgments commit as ONE write (txn hook);
// the 1–32 bounds refuse empty and 33-item batches.
func TestJudgeManyOneTransaction(t *testing.T) {
	svc := newTestService(t)
	ids := seedPendingRelations(t, svc.st, "p", testSession, 5)
	counts := txnCount(t, svc.st)
	rels, err := svc.JudgeMany(ctx, makeJudgeRequests(ids, "not_conflict"))
	if err != nil {
		t.Fatal(err)
	}
	if *counts != 1 {
		t.Fatalf("judge_many must be ONE write, txns %d", *counts)
	}
	for _, r := range rels {
		if r.JudgmentStatus != "judged" || r.Relation != "not_conflict" {
			t.Fatalf("batch member not judged: %+v", r)
		}
	}
	if pendingCount(t, svc.st) != 0 {
		t.Fatalf("batch must clear the pending rows: %d left", pendingCount(t, svc.st))
	}
	if _, err := svc.JudgeMany(ctx, []JudgeRequest{}); err == nil {
		t.Fatalf("0 judgments must fail (1–32)")
	}
	if _, err := svc.JudgeMany(ctx, makeJudgeRequests(ids33(t, svc.st), "not_conflict")); err == nil {
		t.Fatalf("33 judgments must fail (1–32)")
	}
}

// [compare-idempotent] an LLM compare creates a judged relation (system
// provenance) and a second compare of the same pair returns that same row —
// idempotent per unordered pair, one row total.
func TestCompareCreatesJudgedRelationIdempotently(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	svc.stub().setCompare(llm.Judgment{Relation: "compatible", Confidence: 0.8, Evidence: "e", Model: "stub"})
	a := seedObservation(t, svc.st, "p", "decision", "auth-model a", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model b", "B", "")
	r1, err := svc.Compare(ctx, a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r1.JudgmentStatus != "judged" || r1.Relation != "compatible" {
		t.Fatalf("compare must create a judged relation: %+v", r1)
	}
	if r1.MarkedByKind != "system" {
		t.Fatalf("compare provenance must be system: %+v", r1)
	}
	r2, err := svc.Compare(ctx, a.ID, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if r2.ID != r1.ID {
		t.Fatalf("compare must be idempotent per pair: %d vs %d", r2.ID, r1.ID)
	}
	if n := judgedPairCount(t, svc.st, a.SyncID, b.SyncID); n != 1 {
		t.Fatalf("one judged row per unordered pair: %d", n)
	}
}

// (TRIANGULATE, different inputs: degraded compare) with no LLM seam the
// floor runs: no relation is created, a clean sentinel is returned, and the
// second call agrees (idempotence holds in the degraded path too).
func TestCompareDegradedCreatesNothing(t *testing.T) {
	svc := newTestService(t)
	a := seedObservation(t, svc.st, "p", "decision", "auth-model a", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model b", "B", "")
	for i := 0; i < 2; i++ {
		rel, err := svc.Compare(ctx, a.ID, b.ID)
		if err == nil {
			t.Fatalf("degraded compare must signal the floor, got %+v", rel)
		}
		if !strings.Contains(err.Error(), "floor") {
			t.Fatalf("degraded compare sentinel: %v", err)
		}
	}
	if n := judgedPairCount(t, svc.st, a.SyncID, b.SyncID); n != 0 {
		t.Fatalf("degraded compare must create no relation: %d", n)
	}
	if pendingCount(t, svc.st) != 0 {
		t.Fatalf("degraded compare must not leave a pending candidate: %d", pendingCount(t, svc.st))
	}
}

// (TRIANGULATE, different inputs: compare validation) Compare rejects a
// pair of ids that are not two distinct live observations (missing row,
// deleted row, or a==b) before any seam call or write.
func TestCompareRefusesInvalidPair(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	svc.stub().setCompare(llm.Judgment{Relation: "compatible", Confidence: 0.8})
	a := seedObservation(t, svc.st, "p", "decision", "auth-model a", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model b", "B", "")
	if _, err := svc.Compare(ctx, a.ID, a.ID); err == nil {
		t.Fatalf("a pair with itself must be refused")
	}
	if _, err := svc.Compare(ctx, a.ID, 999999); err == nil {
		t.Fatalf("a missing second observation must be refused")
	}
	if err := svc.st.DeleteObservationForProject(ctx, b.ID, "p", false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Compare(ctx, a.ID, b.ID); err == nil {
		t.Fatalf("a deleted second observation must be refused")
	}
}

// ── test helpers (this file only) ──────────────────────────────────────────

// seedPendingRelation seeds two live observations plus ONE pending
// memory_relations row (source = first, target = second) and returns the
// stored pending row (the shape Task 9's materialization produces).
func seedPendingRelation(t *testing.T, st store.Store, project, sessionID string) store.Relation {
	t.Helper()
	a := seedObservation(t, st, project, "decision", "auth-model design", "use bearer tokens", "auth/model")
	b := seedObservation(t, st, project, "decision", "auth-model redesign", "switch to mTLS", "auth/model")
	return addPendingRelation(t, st, a.SyncID, b.SyncID, sessionID)
}

// seedPendingRelations seeds k distinct pending relations (fresh pair per
// relation) and returns their ids in order.
func seedPendingRelations(t *testing.T, st store.Store, project, sessionID string, k int) []int64 {
	t.Helper()
	ids := make([]int64, 0, k)
	for i := 0; i < k; i++ {
		a := seedObservation(t, st, project, "decision",
			fmt.Sprintf("auth-model design %d", i), "content A"+string(rune('0'+i)), "auth/model")
		b := seedObservation(t, st, project, "decision",
			fmt.Sprintf("auth-model redesign %d", i), "content B"+string(rune('0'+i)), "auth/model")
		ids = append(ids, addPendingRelation(t, st, a.SyncID, b.SyncID, sessionID).ID)
	}
	return ids
}

// addPendingRelation inserts one pending relation row and returns it.
func addPendingRelation(t *testing.T, st store.Store, source, target, sessionID string) store.Relation {
	t.Helper()
	rel, err := st.AddRelation(ctx, store.AddRelationParams{
		SourceID: source, TargetID: target, Relation: "pending",
		Reason: "test candidate", SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("add pending relation: %v", err)
	}
	return rel
}

// makeJudgeRequests builds one not_conflict verdict per id.
func makeJudgeRequests(ids []int64, relation string) []JudgeRequest {
	out := make([]JudgeRequest, len(ids))
	for i, id := range ids {
		out[i] = JudgeRequest{JudgmentID: id, Relation: relation, Reason: "batch", Confidence: 0.5}
	}
	return out
}

// ids33 seeds 33 fresh pending relations (a dedicated service store so the
// default test store is not polluted with 66 observations) and returns their
// ids.
func ids33(t *testing.T, st store.Store) []int64 {
	t.Helper()
	return seedPendingRelations(t, st, "p", testSession, 33)
}

// judgedPairCount counts judged memory_relations rows for an unordered
// sync-id pair (either direction).
func judgedPairCount(t *testing.T, st store.Store, a, b string) int {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`
		SELECT COUNT(*) FROM memory_relations
		 WHERE judgment_status = 'judged'
		   AND ((source_id = ? AND target_id = ?) OR (source_id = ? AND target_id = ?))`,
		a, b, b, a)
	if err != nil {
		t.Fatalf("judged pair count: %v", err)
	}
	return n
}
