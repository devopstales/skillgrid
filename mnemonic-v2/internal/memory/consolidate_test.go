package memory

// consolidate_test.go — [consolidate-preview-confirm],
// [consolidate-fingerprint-mismatch], [scan-resurfaces]
// (acceptance.feature), plus triangulation inputs (idempotence, order,
// refusals beyond the brief's set).

import (
	"errors"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// [consolidate-preview-confirm] preview writes nothing and returns the plan
// (both sources + a fingerprint); confirm with the fingerprint merges: the
// survivor carries the merged content + a consolidated_from footer, and
// both sources are soft-deleted with structured supersession provenance.
func TestConsolidatePreviewThenConfirm(t *testing.T) {
	svc := newTestService(t)
	a := seedObservation(t, svc.st, "p", "decision", "auth-model one", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model two", "B", "")
	prev, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "merged", NewContent: "C", ExpectedProject: "p", Confirm: false})
	if err != nil {
		t.Fatal(err)
	}
	if prev.Plan == nil || len(prev.Plan.Sources) != 2 || prev.Plan.Fingerprint == "" {
		t.Fatalf("plan wrong: %+v", prev)
	}
	if prev.Plan.Project != "p" {
		t.Fatalf("plan project wrong: %+v", prev.Plan)
	}
	if n := liveObsCount(t, svc.st); n != 2 {
		t.Fatalf("preview must write nothing: %d live", n)
	}
	conf, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "merged", NewContent: "C", ExpectedProject: "p", Fingerprint: prev.Plan.Fingerprint, Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if conf.Plan == nil || conf.Survivor == nil || conf.Survivor.Title != "merged" {
		t.Fatalf("survivor wrong: %+v", conf)
	}
	if !strings.Contains(conf.Survivor.Content, a.SyncID) || !strings.Contains(conf.Survivor.Content, b.SyncID) {
		t.Fatalf("survivor must carry a consolidated_from footer with the source sync ids: %q", conf.Survivor.Content)
	}
	for _, src := range []int64{a.ID, b.ID} {
		// the soft-deleted row is invisible to GetObservation (it filters
		// deleted_at IS NULL) — probe the row directly for its
		// supersession provenance.
		deleted, err := probeStore(t, svc.st).Probe(`
			SELECT deleted_at IS NOT NULL FROM observations WHERE id = ?`, src)
		if err != nil {
			t.Fatal(err)
		}
		if !deleted {
			t.Fatalf("source %d must be soft-deleted", src)
		}
		by, err := probeStore(t, svc.st).Probe(`
			SELECT ifnull(superseded_by_sync_id,'') = ? FROM observations WHERE id = ?`,
			conf.Survivor.SyncID, src)
		if err != nil {
			t.Fatal(err)
		}
		if !by {
			t.Fatalf("source %d superseded_by_sync_id must point at the survivor", src)
		}
	}
	if n := liveObsCount(t, svc.st); n != 1 {
		t.Fatalf("only the survivor must stay live: %d live", n)
	}
}

// (TRIANGULATE, different inputs: survivor inheritance) the survivor
// inherits the newest source's session/type/scope; an explicit type
// overrides it; topic_key is explicit-only (never inherited from a source
// that has one).
func TestConsolidateSurvivorInheritance(t *testing.T) {
	svc := newTestService(t)
	a := seedObservation(t, svc.st, "p", "decision", "auth-model one", "A", "auth/model")
	b := seedObservationInScope(t, svc.st, "p", "personal", "auth-model two", "B", "")
	conf, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "merged", NewContent: "C", ExpectedProject: "p",
		Fingerprint: fingerprintOf(t, svc, a, b, "merged", "C", "p", ""), Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	s := conf.Survivor
	if s.Scope != "personal" || s.SessionID != testSession {
		t.Fatalf("survivor must inherit the newest source's scope/session: %+v", s)
	}
	if s.Type != "decision" {
		t.Fatalf("survivor must inherit the newest source's type: %+v", s)
	}
	if s.TopicKey != nil {
		t.Fatalf("topic_key is explicit-only: %+v", s.TopicKey)
	}
	// explicit type override
	c2 := seedObservation(t, svc.st, "p", "decision", "auth-model three", "C2", "")
	c3 := seedObservation(t, svc.st, "p", "decision", "auth-model four", "D", "")
	typ := "pattern"
	conf2, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{c2.ID, c3.ID},
		NewTitle: "m2", NewContent: "C2", ExpectedProject: "p", Type: &typ,
		Fingerprint: fingerprintOf(t, svc, c2, c3, "m2", "C2", "p", typ), Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if conf2.Survivor.Type != "pattern" {
		t.Fatalf("explicit type must override inheritance: %+v", conf2.Survivor)
	}
}

// [consolidate-fingerprint-mismatch] a source changed between preview and
// confirm → the stale fingerprint is refused (ErrConsolidateFingerprintMismatch
// + a fresh plan in the result, no writes); the selection's structural
// refusals (fewer than 2 ids, duplicate ids, cross-project) fire before any
// write as well.
func TestConsolidateFingerprintMismatch(t *testing.T) {
	svc := newTestService(t)
	a := seedObservation(t, svc.st, "p", "decision", "auth-model one", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model two", "B", "")
	prev, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "m", NewContent: "C", ExpectedProject: "p"})
	if err != nil {
		t.Fatal(err)
	}
	// mutate a source between preview and confirm
	if err := svc.st.UpdateObservationForProject(ctx, a.ID, "p", store.UpdateParams{Content: ptr("A changed")}); err != nil {
		t.Fatal(err)
	}
	res, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "m", NewContent: "C", ExpectedProject: "p", Fingerprint: prev.Plan.Fingerprint, Confirm: true})
	if !errors.Is(err, ErrConsolidateFingerprintMismatch) {
		t.Fatalf("want fingerprint mismatch, got %v", err)
	}
	if res.Plan == nil || res.Plan.Fingerprint == "" || res.Plan.Fingerprint == prev.Plan.Fingerprint {
		t.Fatalf("mismatch must carry a fresh plan: %+v", res)
	}
	if n := liveObsCount(t, svc.st); n != 2 {
		t.Fatalf("mismatch must write nothing: %d live", n)
	}
	// a confirm with the fresh fingerprint now succeeds
	if _, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "m", NewContent: "C", ExpectedProject: "p", Fingerprint: res.Plan.Fingerprint, Confirm: true}); err != nil {
		t.Fatalf("fresh fingerprint must confirm: %v", err)
	}
	// refusal cases
	other := seedObservation(t, svc.st, "p", "decision", "auth-model other", "O", "")
	for i, bad := range []ConsolidateRequest{
		{IDsToDelete: []int64{other.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"},           // <2 ids
		{IDsToDelete: []int64{other.ID, other.ID}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"}, // duplicate ids
		{IDsToDelete: []int64{999998, 999999}, NewTitle: "m", NewContent: "C", ExpectedProject: "p"},     // missing sources
		{NewTitle: "m", NewContent: "C", ExpectedProject: "p"},                                           // no ids at all
		{NewTitle: "m", NewContent: "C"}, // unresolvable project
	} {
		if _, err := svc.Consolidate(ctx, bad); err == nil {
			t.Fatalf("case %d: must refuse %+v", i, bad)
		}
	}
	// cross-project: an expected_project the sources do not belong to is a
	// structural refusal (the ownership assertion — spec 01-memory/001 §6,
	// the mirror of UpdateObservationForProject's in-transaction guard) —
	// refused before any write.
	if _, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, other.ID},
		NewTitle: "m", NewContent: "C", ExpectedProject: "other"}); !errors.Is(err, ErrConsolidateRefused) {
		t.Fatalf("cross-project selection must be refused, got %v", err)
	}
}

// (TRIANGULATE, different inputs: confirm without a fingerprint) the MCP
// contract allows confirm without a fingerprint — it re-resolves, refuses if
// the selection no longer holds, and executes otherwise.
func TestConsolidateConfirmWithoutFingerprint(t *testing.T) {
	svc := newTestService(t)
	a := seedObservation(t, svc.st, "p", "decision", "auth-model one", "A", "")
	b := seedObservation(t, svc.st, "p", "decision", "auth-model two", "B", "")
	conf, err := svc.Consolidate(ctx, ConsolidateRequest{IDsToDelete: []int64{a.ID, b.ID},
		NewTitle: "merged", NewContent: "C", ExpectedProject: "p", Confirm: true})
	if err != nil {
		t.Fatal(err)
	}
	if conf.Survivor == nil || liveObsCount(t, svc.st) != 1 {
		t.Fatalf("direct confirm must execute: survivor=%v live=%d", conf.Survivor != nil, liveObsCount(t, svc.st))
	}
}

// [scan-resurfaces] a scan re-runs candidate detection over the project and
// surfaces NEW pending relations without duplicating already-seen pairs
// (a pair with any existing relation row — pending or judged — is skipped);
// it returns the count of newly materialized pending rows.
func TestScanSurfacesNewPending(t *testing.T) {
	svc := newTestService(t)
	// one conflicting pair; the save pipeline materializes the pending row
	seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "use bearer tokens", "auth/model")
	r, err := svc.Save(ctx, SaveRequest{SessionID: testSession, Type: "decision", Title: "auth-model v2",
		Content: "switch to mTLS", Project: "p", Scope: "project", TopicKey: "auth/model"})
	if err != nil {
		t.Fatal(err)
	}
	if n := pendingCount(t, svc.st); n != 1 {
		t.Fatalf("setup: expected 1 pending, got %d", n)
	}
	// the save's own materialization already covered the pair: the scan
	// finds nothing new for it (both the save's source row and the seed
	// row have a relation row — the seen-check skips both).
	n, err := svc.Scan(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("scan must not duplicate an already-seen pair: %d new", n)
	}
	// a judged pair is equally already-seen
	rel, err := svc.Judge(ctx, JudgeRequest{JudgmentID: r.Candidates[0].ID, Relation: "not_conflict", Reason: "fine"})
	if err != nil {
		t.Fatal(err)
	}
	seedObservation(t, svc.st, "p", "decision", "auth-model third", "t3", "auth/model")
	if _, err := svc.Save(ctx, SaveRequest{SessionID: testSession, Type: "decision", Title: "auth-model v3",
		Content: "another", Project: "p", Scope: "project", TopicKey: "auth/model"}); err != nil {
		t.Fatal(err)
	}
	n, err = svc.Scan(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("scan must not duplicate a judged pair: %d new", n)
	}
	_ = rel
}

// [scan-resurfaces] (the post-pull leg, the acceptance scenario's shape)
// observations that were pulled in AFTER the save pipeline ran are surfaced
// by the scan: a directly-seeded conflicting row has no save-pipeline
// materialization, so the scan is the only pass that catches it. The scan
// returns the new pending count and the pending rows are queryable.
func TestScanResurfacesPulledObservations(t *testing.T) {
	svc := newTestService(t)
	seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "use bearer tokens", "auth/model")
	// the "pulled" conflicting observation (added directly, not via Save)
	pulled := seedObservation(t, svc.st, "p", "decision", "auth-model v2", "switch to mTLS", "auth/model")
	n, err := svc.Scan(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("scan must surface the pulled conflict: %d new", n)
	}
	if n := pendingCount(t, svc.st); n != 1 {
		t.Fatalf("pending row not materialized: %d", n)
	}
	// re-scan: idempotent (the pair is now already-seen)
	if n, _ := svc.Scan(ctx, "p"); n != 0 {
		t.Fatalf("re-scan must not duplicate: %d new", n)
	}
	_ = pulled
}

// (TRIANGULATE, different inputs: empty project) a scan of a project with
// nothing to detect returns 0, not an error.
func TestScanEmptyProject(t *testing.T) {
	svc := newTestService(t)
	n, err := svc.Scan(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("empty scan must return 0: %d", n)
	}
}

// (TRIANGULATE, different inputs: scan order) the scan re-runs the gate in
// the project's id order, so the pending rows it materializes are in the
// same order as the live rows (deterministic output for the agent).
func TestScanOrderDeterministic(t *testing.T) {
	svc := newTestService(t)
	base := seedObservation(t, svc.st, "p", "decision", "auth-model redesign", "base", "auth/model")
	scanOnce := func() int {
		n, err := svc.Scan(ctx, "p")
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	v2 := seedObservation(t, svc.st, "p", "decision", "auth-model v2", "one", "auth/model")
	v3 := seedObservation(t, svc.st, "p", "decision", "auth-model v3", "two", "auth/model")
	if n := scanOnce(); n != 2 {
		t.Fatalf("expected 2 new pending: %d", n)
	}
	// the scan walks rows in id order and materializes source=scanned row:
	// the first row walked (base, the smallest id) is the first pending row's
	// source, with its candidates (v2, v3) as targets, in pre-filter order.
	rows := scanPendingRows(t, svc.st)
	if len(rows) != 2 {
		t.Fatalf("expected 2 pending rows: %d", len(rows))
	}
	if strVal(rows[0], 0) != base.SyncID {
		t.Fatalf("scan must run in row id order: first row's source is %v, want %v",
			strVal(rows[0], 0), base.SyncID)
	}
	if strVal(rows[1], 0) != base.SyncID {
		t.Fatalf("second row must be the same scanned source: %v", strVal(rows[1], 0))
	}
	gotTargets := map[string]bool{strVal(rows[0], 1): true, strVal(rows[1], 1): true}
	if !gotTargets[v2.SyncID] || !gotTargets[v3.SyncID] {
		t.Fatalf("targets must be v2 and v3: %v", gotTargets)
	}
}

// ── test helpers (this file only) ──────────────────────────────────────────

// fingerprintOf previews a consolidation and returns its plan fingerprint
// (the confirm-with-fingerprint tests need the preview's value).
func fingerprintOf(t *testing.T, svc *Service, a, b store.Observation, title, content, project, typ string) string {
	t.Helper()
	var typPtr *string
	if typ != "" {
		typPtr = &typ
	}
	prev, err := svc.Consolidate(ctx, ConsolidateRequest{
		IDsToDelete: []int64{a.ID, b.ID}, NewTitle: title, NewContent: content,
		ExpectedProject: project, Type: typPtr,
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	return prev.Plan.Fingerprint
}

// idOf returns the local id of a live observation by sync id (0 = absent).
func idOf(t *testing.T, st store.Store, syncID string) int64 {
	t.Helper()
	n, err := probeStore(t, st).ProbeInt(`SELECT id FROM observations WHERE sync_id = ? AND deleted_at IS NULL`, syncID)
	if err != nil {
		t.Fatalf("id of: %v", err)
	}
	return int64(n)
}

// scanPendingRows lists the scan's pending rows (source, target) in id
// order — the assertion surface for the scan's output.
func scanPendingRows(t *testing.T, st store.Store) [][]any {
	t.Helper()
	rows, err := rawQueryRows(t, st, `
		SELECT source_id, target_id FROM memory_relations
		 WHERE judgment_status = 'pending' ORDER BY id`, nil)
	if err != nil {
		t.Fatalf("scan pending rows: %v", err)
	}
	return rows
}
