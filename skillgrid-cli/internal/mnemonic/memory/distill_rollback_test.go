package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"
)

// distillLLM is a DreamLLM whose ConsolidatePrompt succeeds (so the pre-failure
// mutation — sources marked consolidated + a merged row — happens) and whose
// SynthesizePrompt returns the hard sentinel errDistillFail. It stands in for a
// distillation that fails mid-synthesize, the failure point 17.2's rollback must
// recover from.
type distillLLM struct{ consolidate string }

func (d *distillLLM) ConsolidatePrompt(_ context.Context, _ string, _ []string) (string, error) {
	return d.consolidate, nil
}
func (d *distillLLM) SynthesizePrompt(_ context.Context, _ []string) (string, error) {
	return "", errDistillFail
}

// errDistillFail is the sentinel the distill LLM returns in synthesize.
var errDistillFail = errors.New("distill synthesis backend unavailable")

// TestDistillRollbackOnFailure (014 step 17.2): capture the pre-distill state,
// mutate it the way a failed distillation would (consolidate marks sources
// consolidated and rewrites a merged row; synthesize fails), then DistillRollback
// — every observation round-trips to its exact pre-distill state, the lock is
// released, and no partial change (consolidated status, rewritten merged row)
// remains. The restore is atomic (step 12 finding F1): it runs in a single
// transaction so a crash mid-rollback cannot leave a half-restored project.
func TestDistillRollbackOnFailure(t *testing.T) {
	_, svc := newTestStore(t, "distillrollbackproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	de := NewDreamExecutor(svc)
	de.SetLLM(&distillLLM{consolidate: "Merged auth facts."})
	de.failOnLLMError = true
	locks := NewDistillLockService(svc.DB())
	pid := svc.ProjectID()

	// Seed 5 distinct same-topic observations as live rows, inserted directly
	// (not via Save) so Save's topic_key upsert does not collapse them.
	facts := []string{
		"Fact A: auth token in a signed cookie.",
		"Fact B: cookie expiry 24 hours.",
		"Fact C: refresh tokens rotated on use.",
		"Fact D: session store is SQLite.",
		"Fact E: logout revokes the token.",
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, f := range facts {
		sum := sha256.Sum256([]byte(f))
		if _, err := svc.DB().ExecContext(ctx, `
			INSERT INTO observations (
				session_id, type, title, content, project, scope, topic_key,
				normalized_hash, revision_count, created_at, updated_at, source,
				owner, visibility, status, retrieval_usage, expires_at
			) VALUES (?, 'decision', ?, ?, ?, 'project', 'topic/auth',
				?, 0, ?, ?, 'agent', ?, 'private', 'active', 0, ?)`,
			sid, f, f, pid, hex.EncodeToString(sum[:]), now, now, pid, now); err != nil {
			t.Fatalf("seed insert: %v", err)
		}
	}

	// Capture the pre-distill state (the snapshot DistillRollback restores to).
	pre, err := de.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(pre) != len(facts) {
		t.Fatalf("snapshot captured %d rows, want %d", len(pre), len(facts))
	}
	preContents := make(map[string]bool, len(pre))
	for _, s := range pre {
		preContents[s.Content] = true
	}

	// Acquire the distill lock (a real distillation holds it for its whole run).
	if err := locks.Acquire(ctx, pid, "distill-worker"); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	// Run the distillation: consolidate mutates (sources consolidated + a row
	// rewritten to the merged body), then synthesize fails.
	obs, err := liveRows(t, svc)
	if err != nil {
		t.Fatalf("read live rows: %v", err)
	}
	if _, err := de.consolidate(ctx, obs); err != nil {
		t.Fatalf("consolidate (should succeed): %v", err)
	}
	if _, err := de.synthesize(ctx, []Memory{{SessionID: sid, Content: "Session: auth work."}}); !errors.Is(err, errDistillFail) {
		t.Fatalf("synthesize should fail with errDistillFail, got: %v", err)
	}

	// Prove the failure left a mutated state: the sources are now consolidated
	// and at least one row's content no longer matches a pre-distill fact (the
	// merged body replaced it).
	var consolidated int
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND status = ? AND deleted_at IS NULL`, pid, dreamStatus).Scan(&consolidated); err != nil {
		t.Fatalf("count consolidated: %v", err)
	}
	if consolidated != len(facts) {
		t.Fatalf("after consolidate, %d/%d consolidated, want all", consolidated, len(facts))
	}
	// At least one row's content no longer matches a pre-distill fact (the
	// merged body replaced it) — the pre-failure mutation the rollback reverts.
	mutated := 0
	rows, err := liveRows(t, svc)
	if err != nil {
		t.Fatalf("read live rows after consolidate: %v", err)
	}
	for _, o := range rows {
		if !preContents[o.Content] {
			mutated++
		}
	}
	if mutated == 0 {
		t.Fatalf("consolidate did not mutate any row's content (no pre-failure mutation to roll back)")
	}

	// Roll back to the pre-distill state.
	if err := de.DreamRollback(ctx, pid, pre, locks.AsDreamLock()); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Every observation round-trips to its EXACT pre-distill column values.
	after, err := liveRows(t, svc)
	if err != nil {
		t.Fatalf("read live rows after rollback: %v", err)
	}
	if len(after) != len(pre) {
		t.Fatalf("live rows after rollback = %d, want %d (pre-distill count)", len(after), len(pre))
	}
	preByID := make(map[int64]ObservationSnapshot, len(pre))
	for _, s := range pre {
		preByID[s.ID] = s
	}
	for _, o := range after {
		s, ok := preByID[o.ID]
		if !ok {
			t.Fatalf("row %d is not in the pre-distill snapshot (partial change)", o.ID)
		}
		if o.Content != s.Content {
			t.Errorf("row %d content = %q, want pre-distill %q", o.ID, o.Content, s.Content)
		}
		if o.Status != s.Status {
			t.Errorf("row %d status = %q, want pre-distill %q (consolidation not reverted)", o.ID, o.Status, s.Status)
		}
		if o.Project != s.Project {
			t.Errorf("row %d project = %q, want %q", o.ID, o.Project, s.Project)
		}
	}

	// No partial change remains: no consolidated status, and every live row's
	// content is a pre-distill fact (the merged body is gone).
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND status = ? AND deleted_at IS NULL`, pid, dreamStatus).Scan(&consolidated); err != nil {
		t.Fatalf("count consolidated after rollback: %v", err)
	}
	if consolidated != 0 {
		t.Fatalf("consolidated rows after rollback = %d, want 0", consolidated)
	}
	for _, o := range after {
		if !preContents[o.Content] {
			t.Fatalf("row %d content %q is not a pre-distill fact (partial change)", o.ID, o.Content)
		}
	}
	// Content round-tripped (every fact is present again).
	afterSet := make(map[string]bool, len(after))
	for _, o := range after {
		afterSet[o.Content] = true
	}
	for _, f := range facts {
		if !afterSet[f] {
			t.Errorf("fact %q not restored after rollback", f)
		}
	}

	// Lock released after rollback.
	if locked, err := locks.IsLocked(ctx, pid); err != nil {
		t.Fatalf("is locked: %v", err)
	} else if locked {
		t.Fatalf("lock should be released after rollback")
	}
}

// TestDistillRollbackAtomic (014 step 17.2, step 12 finding F1): the rollback's
// restore + orphan sweep run in a SINGLE transaction. A failure mid-restore
// rolls the whole thing back — no half-restored project is left. This is what
// distinguishes the atomic rollback from step 12's best-effort, non-transactional
// three-step rollback (restore, then sweep, then release).
func TestDistillRollbackAtomic(t *testing.T) {
	_, svc := newTestStore(t, "distillatomicproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	de := NewDreamExecutor(svc)
	pid := svc.ProjectID()

	// Seed 3 live rows.
	now := time.Now().UTC().Format(time.RFC3339)
	for i, f := range []string{"Atomic F1", "Atomic F2", "Atomic F3"} {
		sum := sha256.Sum256([]byte(f))
		if _, err := svc.DB().ExecContext(ctx, `
			INSERT INTO observations (
				session_id, type, title, content, project, scope, topic_key,
				normalized_hash, revision_count, created_at, updated_at, source,
				owner, visibility, status, retrieval_usage
			) VALUES (?, 'decision', ?, ?, ?, 'project', ?,
				?, 0, ?, ?, 'agent', ?, 'private', 'active', 0)`,
			sid, f, f, pid, fmt.Sprintf("topic/a%d", i), hex.EncodeToString(sum[:]), now, now, pid); err != nil {
			t.Fatalf("seed insert: %v", err)
		}
	}
	pre, err := de.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	// Mutate: mark the rows consolidated (the pre-failure change) and add a
	// synthetic orphan row (id beyond the snapshot) that a failed dream created.
	for _, s := range pre {
		if _, err := svc.DB().ExecContext(ctx, `UPDATE observations SET status = ? WHERE id = ?`, dreamStatus, s.ID); err != nil {
			t.Fatalf("mutate status: %v", err)
		}
	}
	// Find the next id and insert an orphan the rollback sweep must delete.
	if _, err := svc.DB().ExecContext(ctx, `
		INSERT INTO observations (session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at, source, owner, visibility, status)
		VALUES (?, 'learning', 'orphan', 'synthesized orphan', ?, 'project', 'orphanhash', 0, ?, ?, 'dream', ?, 'private', 'active')`,
		sid, pid, now, now, pid); err != nil {
		t.Fatalf("insert orphan: %v", err)
	}

	// Force a failure MID-RESTORE via the injected executor: the first
	// statement succeeds, the second errors (a simulated crash after partial
	// writes). The atomic rollback must leave the table EXACTLY as it was
	// pre-failure (all consolidated + orphan present), not half-restored.
	locks := NewDistillLockService(svc.DB())
	if err := locks.Acquire(ctx, pid, "atomic-worker"); err != nil {
		t.Fatalf("acquire: %v", err)
	}
	var call int
	failingExec := func(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
		call++
		if call == 2 {
			return errors.New("simulated crash mid-restore")
		}
		if _, err := tx.ExecContext(ctx, query, args...); err != nil {
			return err
		}
		return nil
	}
	err = de.DreamRollbackWith(ctx, pid, pre, locks.AsDreamLock(), failingExec)
	if err == nil {
		t.Fatalf("rollback with a mid-restore failure should return an error")
	}

	// The failure left the project UNCHANGED from its pre-failure state: the rows
	// are still consolidated (the first restore was rolled back) and the orphan
	// still exists (the sweep did not run). This is the F1 guarantee — an atomic
	// rollback never leaves a half-restored project.
	var consolidated int
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND status = ? AND deleted_at IS NULL`, pid, dreamStatus).Scan(&consolidated); err != nil {
		t.Fatalf("count consolidated: %v", err)
	}
	if consolidated != len(pre) {
		t.Fatalf("after a failed atomic rollback, %d/%d rows consolidated, want all (restore must roll back)", consolidated, len(pre))
	}
	var orphan int
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND title = 'orphan' AND deleted_at IS NULL`, pid).Scan(&orphan); err != nil {
		t.Fatalf("count orphan: %v", err)
	}
	if orphan != 1 {
		t.Fatalf("after a failed atomic rollback, orphan rows = %d, want 1 (sweep must not run)", orphan)
	}
	// The lock must STILL be held on a failed rollback (fail loud: do not free a
	// project whose data was not fully restored).
	if locked, err := locks.IsLocked(ctx, pid); err != nil {
		t.Fatalf("is locked: %v", err)
	} else if !locked {
		t.Fatalf("lock should still be held after a FAILED rollback (fail loud)")
	}
}
