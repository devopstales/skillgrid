package memory

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// seedTrigram seeds an observation of content "trigram <i>" (content that
// distinguishes each row for the pre-filter order assertions) and returns
// the row.
func seedTrigram(t *testing.T, st store.Store, project, title string, i int) store.Observation {
	t.Helper()
	return seedObservation(t, st, project, "decision", title, fmt.Sprintf("trigram %d", i), "")
}

// ── the brief's three scenarios (acceptance.feature LLM-first dedup block) ──

// [dedup-llm-primary] an LLM seam that classifies the new save as an update
// of a live observation → the save merges into the target, the verdict is
// labeled llm, and the result is not degraded.
func TestDedupLLMPrimaryWhenSeamPresent(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	live := seedTrigram(t, svc.st, "p", "old title", 1) // the only row → head of pre-filter
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictUpdate, CandidateID: live.ID, Reason: "llm"}, nil
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "new", Content: "same thing reworded", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "update" || r.Dedup.Reason != "llm" || r.ID != live.ID {
		t.Fatalf("LLM verdict must win and merge: %+v", r)
	}
	if r.Degraded {
		t.Fatalf("seam present → not degraded")
	}
	if r.RevisionCount != 1 {
		t.Fatalf("update must be an in-place revision (revision_count++): got %d", r.RevisionCount)
	}
	if r.Dedup.MergedIntoSyncID == nil || *r.Dedup.MergedIntoSyncID != live.SyncID {
		t.Fatalf("MergedIntoSyncID must reference the target by sync_id: %+v", r.Dedup.MergedIntoSyncID)
	}
}

// [dedup-trust-but-verify] an LLM seam that points at a non-live candidate
// id → the merge falls back to the most-recent live candidate (the head of
// the pre-filter). The verdict is trusted; the pointer is verified.
func TestDedupTrustButVerifyCandidateID(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	seedTrigram(t, svc.st, "p", "older row", 1)
	head := seedTrigram(t, svc.st, "p", "head", 2) // most recent = head of pre-filter
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictUpdate, CandidateID: 999, Reason: "llm"}, nil
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != head.ID {
		t.Fatalf("stale CandidateID must fall back to most-recent live candidate: got id %d want %d", r.ID, head.ID)
	}
	if r.Dedup.Verdict != "update" || r.Dedup.Reason != "llm" {
		t.Fatalf("the LLM verdict is trusted even when its pointer is repaired: %+v", r.Dedup)
	}
}

// [dedup-noop-first-class] an LLM seam that classifies the new save as a
// no-op → no observation is written (a first-class outcome, not a fallback).
func TestDedupNoopIsFirstClass(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	live := seedTrigram(t, svc.st, "p", "known row", 1)
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictNoop, CandidateID: live.ID, Reason: "llm"}, nil
	})
	before := countObs(t, svc.st)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "noop" || r.Dedup.Reason != "llm" {
		t.Fatalf("noop verdict expected, got %+v", r.Dedup)
	}
	if after := countObs(t, svc.st); after != before {
		t.Fatalf("noop must write nothing: before %d after %d", before, after)
	}
	if r.ID != live.ID {
		t.Fatalf("noop carries the matched row's identity: got %d want %d", r.ID, live.ID)
	}
}

// [dedup-bounded-retry] (the LLM leg of degraded-bounded-retry) a seam that
// fails on every call → exactly one bounded retry, then the hash floor, and
// the result is flagged degraded.
func TestDedupFailingSeamRetriesOnceThenFloor(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	seedTrigram(t, svc.st, "p", "existing", 1)
	calls := 0
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		calls++
		return llm.DedupDecision{}, fmt.Errorf("llm down")
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("a failing seam gets exactly one bounded retry: %d classify calls", calls)
	}
	if r.Dedup.Verdict != "add" || r.Dedup.Reason != "hash" {
		t.Fatalf("after the retry the hash floor runs: %+v", r.Dedup)
	}
	if !r.Degraded {
		t.Fatal("a degraded call is always flagged")
	}
}

// ── triangulation: the remaining 4-way arm + the pre-filter contract ───────

// [dedup-delete-supersedes] the delete arm: the target is superseded by the
// new save (soft-deleted, superseded_by_sync_id → the new row's sync_id,
// a supersedes relation recorded) and the new save is stored.
func TestDedupDeleteSupersedesTarget(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	syncStore(t, svc.st).SetSyncEnabled(true) // the delete journal is enrollment-gated
	syncStore(t, svc.st).SetEnrolledForTest("p")
	target := seedTrigram(t, svc.st, "p", "stale rule", 1)
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictDelete, CandidateID: target.ID, Reason: "llm"}, nil
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "current rule", Content: "the replacement", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "delete" || r.Dedup.Reason != "llm" {
		t.Fatalf("delete verdict expected: %+v", r.Dedup)
	}
	if r.ID == target.ID {
		t.Fatal("delete must store the new observation (the target is gone, not revised)")
	}
	if r.Dedup.MergedIntoSyncID == nil || *r.Dedup.MergedIntoSyncID != target.SyncID {
		t.Fatalf("MergedIntoSyncID must reference the superseded target: %+v", r.Dedup)
	}
	if isLive(t, svc.st, target.SyncID) {
		t.Fatal("the target must be soft-deleted")
	}
	if ok, err := probeStore(t, svc.st).Probe(`SELECT superseded_by_sync_id = ? FROM observations WHERE id = ?`, r.SyncID, target.ID); err != nil || !ok {
		t.Fatalf("the target's superseded_by_sync_id must point at the new sync_id: ok=%v err=%v", ok, err)
	}
	if n, err := probeStore(t, svc.st).ProbeInt(`SELECT COUNT(*) FROM memory_relations WHERE source_id = ? AND target_id = ? AND relation = 'supersedes'`, r.SyncID, target.SyncID); err != nil || n != 1 {
		t.Fatalf("a supersedes edge (new → old) must be recorded: n=%d err=%v", n, err)
	}
	if after := countObs(t, svc.st); after != 1 {
		t.Fatalf("the live count must hold at 1 (target deleted, new stored): %d", after)
	}
	// the delete journals (enrollment-gated) with the target's sync_id as
	// the entity key — the peer's apply path resolves id ← sync_id.
	if n, err := probeStore(t, svc.st).ProbeInt(`SELECT COUNT(*) FROM sync_mutations WHERE entity = 'observation' AND op = 'delete' AND entity_key = ?`, target.SyncID); err != nil || n != 1 {
		t.Fatalf("the superseded target must journal a delete: n=%d err=%v", n, err)
	}
}

// [dedup-add-new-row] the add arm: the LLM says genuinely new → a fresh row
// with the llm reason (the 4th verdict; the pipeline's insert path is
// unchanged, the seam's verdict is what is under test).
func TestDedupAddStoresFreshRow(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	seedTrigram(t, svc.st, "p", "unrelated", 1)
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "llm"}, nil
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "brand new", Content: "a fresh fact", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "add" || r.Dedup.Reason != "llm" {
		t.Fatalf("add verdict with llm reason expected: %+v", r.Dedup)
	}
	if r.Dedup.MergedIntoSyncID != nil {
		t.Fatalf("add merges into nothing: %+v", r.Dedup.MergedIntoSyncID)
	}
	if after := countObs(t, svc.st); after != 2 {
		t.Fatalf("add must insert a new row: %d live rows", after)
	}
}

// [dedup-noop-without-candidate] a noop verdict carrying no candidate id is
// still a no-op (spec 004-dedup §2: the id is only meaningful for
// update/delete) — the pipeline's CandidateID>0 guard must not misread it
// as an insert.
func TestDedupNoopWithoutCandidateWritesNothing(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	seedTrigram(t, svc.st, "p", "some row", 1)
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictNoop, Reason: "llm"}, nil
	})
	before := countObs(t, svc.st)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "noop" || r.Dedup.Reason != "llm" {
		t.Fatalf("noop verdict expected: %+v", r.Dedup)
	}
	if after := countObs(t, svc.st); after != before {
		t.Fatalf("a noop without a candidate must write nothing: before %d after %d", before, after)
	}
}

// [dedup-pre-filter-recent-window] the pre-filter contract (spec 004-dedup
// §3): ≤dedupWindowSize rows, most-recent first (id DESC — created_at is
// second-granularity), non-deleted only.
func TestDedupPreFilterRecentWindowCapAndOrder(t *testing.T) {
	st := openTestStore(t)
	want := dedupWindowSize + 3
	ids := make([]int64, 0, want)
	for i := 1; i <= want; i++ {
		ids = append(ids, seedTrigram(t, st, "p", fmt.Sprintf("t%d", i), i).ID)
	}
	head, contents := preFilter(ctx, st, "p")
	if len(contents) != dedupWindowSize {
		t.Fatalf("the pre-filter is capped at %d: got %d", dedupWindowSize, len(contents))
	}
	// contents are most-recent first: ids[0] is the OLDEST row, so position
	// i is row ids[want-1-i]'s content ("trigram <i>").
	for i, c := range contents {
		want := fmt.Sprintf("trigram %d", want-i)
		if c != want {
			t.Fatalf("pre-filter must be most-recent first: position %d got %q want %q", i, c, want)
		}
	}
	if head.ID != ids[want-1] || head.SyncID != syncIDOf(t, st, ids[want-1]) {
		t.Fatalf("the head is the most-recent live candidate: head=%+v", head)
	}
	// non-deleted only: soft-delete the newest row → it drops off the head.
	if err := st.DeleteObservationForProject(ctx, ids[want-1], "p", false); err != nil {
		t.Fatal(err)
	}
	head, contents = preFilter(ctx, st, "p")
	if len(contents) != dedupWindowSize || head.ID != ids[want-2] {
		t.Fatalf("soft-deleted rows are not candidates: head=%d len=%d", head.ID, len(contents))
	}
}

// The pre-filter is project-scoped: candidates from another project never
// cross in (spec 004-dedup §6: dedup is scoped to project+scope).
func TestDedupPreFilterDoesNotCrossProjects(t *testing.T) {
	st := openTestStore(t)
	pRow := seedTrigram(t, st, "p", "p row", 1)
	seedTrigram(t, st, "q", "q row", 1)
	head, contents := preFilter(ctx, st, "p")
	if len(contents) != 1 || head.ID != pRow.ID {
		t.Fatalf("project scope must hold: head=%+v len=%d", head, len(contents))
	}
}

// syncIDOf returns a live row's sync id (a test assertion helper).
func syncIDOf(t *testing.T, st store.Store, id int64) string {
	t.Helper()
	o, err := st.GetObservation(ctx, id, false)
	if err != nil {
		t.Fatalf("sync id of %d: %v", id, err)
	}
	return o.SyncID
}

// Trust-but-verify on an EMPTY pre-filter: a positive CandidateID with no
// live candidate to fall back to stays 0 — the pipeline's CandidateID>0
// guard makes update/noop no-ops, so a stale pointer degrades to "nothing
// merged," never to a wrong row.
func TestDedupTrustButVerifyEmptyPreFilter(t *testing.T) {
	svc := newTestServiceWithLLM(t, &stubLLM{})
	svc.stub().setClassify(func(ctx context.Context, newContent string, candidates []string) (llm.DedupDecision, error) {
		return llm.DedupDecision{Verdict: llm.VerdictUpdate, CandidateID: 42, Reason: "llm"}, nil
	})
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "x", Content: "y", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "update" {
		t.Fatalf("the verdict is still trusted: %+v", r.Dedup)
	}
	if r.Dedup.MergedIntoSyncID != nil {
		t.Fatalf("no live candidate → nothing merged: %+v", r.Dedup.MergedIntoSyncID)
	}
	if n, err := probeStore(t, svc.st).ProbeInt(`SELECT COUNT(*) FROM observations`); err != nil || n != 0 {
		t.Fatalf("an update with no verified target writes no row: n=%d err=%v", n, err)
	}
}

// The prod adapter (Task 3's OpenAI-compat client) through the full LLM
// path: the seam's 0-based candidate index lands as a 1-based CandidateID
// (candidate 0 → the pre-filter head) and the merge executes — the other
// "2 real" in the task's adapter list (stub + floor).
func TestDedupLLMPrimaryWithRealClient(t *testing.T) {
	st := openTestStore(t)
	svc := newTestServiceWithStore(t, st)
	// Re-point the service's seam at the real client (the constructor took
	// a nil seam; this test exercises the client directly, not the stub).
	svc.llm = testLLMClient(t, func(w http.ResponseWriter) {
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"verdict\":\"update\",\"candidate\":0}"}}]}`)
	})
	live := seedTrigram(t, svc.st, "p", "old", 1)
	r, err := svc.Save(ctx, SaveRequest{SessionID: "s", Type: "decision", Title: "new", Content: "reworded", Project: "p", Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Dedup.Verdict != "update" || r.Dedup.Reason != "llm" || r.ID != live.ID {
		t.Fatalf("real-client classify must merge into the first candidate: %+v", r)
	}
}

// testLLMClient points Task 3's OpenAI-compat client at a per-test handler.
func testLLMClient(t *testing.T, handler func(w http.ResponseWriter)) llm.LLM {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handler(w)
	}))
	t.Cleanup(srv.Close)
	c, err := llm.NewClient(srv.URL+"/v1", "test-model", "k", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
