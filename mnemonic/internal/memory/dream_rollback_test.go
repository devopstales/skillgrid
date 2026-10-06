package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"
)

// failingSynthLLM is a DreamLLM whose SynthesizePrompt returns a HARD error (a
// distinct sentinel), standing in for a distillation that fails mid-synthesize
// (the failure point 12.5 guards). With the executor in hard-fail mode the
// synthesize phase returns that error (no deterministic fallback), so the
// dream aborts after consolidate has already mutated state. ConsolidatePrompt
// succeeds so the pre-failure mutation (sources marked consolidated + a new
// observation created) actually happens.
type failingSynthLLM struct {
	consolidate string
}

func (s *failingSynthLLM) ConsolidatePrompt(_ context.Context, _ string, _ []string) (string, error) {
	return s.consolidate, nil
}
func (s *failingSynthLLM) SynthesizePrompt(_ context.Context, _ []string) (string, error) {
	return "", errSynthFail
}

// errSynthFail is the sentinel the failing synth LLM returns.
var errSynthFail = errors.New("synthesis backend unavailable")

// TestDreamRollbackOnFailure (014 step 12.5): start a dream (consolidate),
// capture the pre-distill state, simulate a failure during synthesize, then
// DreamRollback — every observation is restored to its pre-distill state and
// the lock is released.
func TestDreamRollbackOnFailure(t *testing.T) {
	_, svc := newTestStore(t, "rollbackproj")
	sid := newSession(t, svc)
	ctx := context.Background()
	de := NewDreamExecutor(svc)
	de.SetLLM(&failingSynthLLM{consolidate: "Merged auth facts."})
	// Hard-fail mode: a seam error in synthesize aborts the dream (no fallback),
	// which is the failure 12.5's rollback must recover from.
	de.failOnLLMError = true
	locks := NewDreamLockService(svc.DB())

	// Seed 5 DISTINCT same-topic observations as live rows. They are inserted
	// directly (not via Save) so Save's topic_key upsert does not collapse them
	// to one row: all 5 must be present so consolidate merges a 5-member group.
	facts := []string{
		"Fact A: auth token in a signed cookie.",
		"Fact B: cookie expiry 24 hours.",
		"Fact C: refresh tokens rotated on use.",
		"Fact D: session store is SQLite.",
		"Fact E: logout revokes the token.",
	}
	now := time.Now().UTC().Format(time.RFC3339)
	seed := func(content string) int64 {
		t.Helper()
		sum := sha256.Sum256([]byte(content))
		hash := hex.EncodeToString(sum[:])
		res, err := svc.DB().ExecContext(ctx, `
			INSERT INTO observations (
				session_id, type, title, content, project, scope, topic_key,
				normalized_hash, revision_count, created_at, updated_at, source,
				owner, visibility, status, retrieval_usage, expires_at
			) VALUES (?, 'decision', ?, ?, 'rollbackproj', 'project', 'topic/auth',
				?, 0, ?, ?, 'agent', 'rollbackproj', 'private', 'active', 0, ?)`,
			sid, content, content, hash, now, now, now)
		if err != nil {
			t.Fatalf("seed insert: %v", err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatalf("seed lastid: %v", err)
		}
		return id
	}
	for _, f := range facts {
		seed(f)
	}

	var obs []Observation
	var err error
	obs, err = liveRows(t, svc)
	if err != nil {
		t.Fatalf("read live rows: %v", err)
	}
	if len(obs) != len(facts) {
		t.Fatalf("seeded %d live rows, want %d", len(obs), len(facts))
	}

	// Capture the pre-distill state (the snapshot DreamRollback restores to).
	pre, err := de.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if len(pre) != len(obs) {
		t.Fatalf("snapshot captured %d rows, want %d", len(pre), len(obs))
	}

	// Acquire the dream lock (a real dream holds it for its whole run).
	if err := locks.Acquire(ctx, svc.ProjectID(), "dream-worker"); err != nil {
		t.Fatalf("acquire: %v", err)
	}

	// Run the dream: consolidate mutates (marks sources consolidated, creates a
	// merged observation), then synthesize fails.
	if _, err := de.consolidate(ctx, obs); err != nil {
		t.Fatalf("consolidate (should succeed): %v", err)
	}
	if _, err := de.synthesize(ctx, []Memory{{SessionID: sid, Content: "Session: auth work."}}); !errors.Is(err, errSynthFail) {
		t.Fatalf("synthesize should fail with errSynthFail, got: %v", err)
	}

	// Prove the failure left a mutated state: the sources are now consolidated
	// (not "active") and a merged observation was created.
	var consolidatedCount, mergedCount int
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND status = ? AND deleted_at IS NULL`, svc.ProjectID(), dreamStatus).Scan(&consolidatedCount); err != nil {
		t.Fatalf("count consolidated: %v", err)
	}
	if consolidatedCount != len(obs) {
		t.Fatalf("after consolidate, %d/%d sources consolidated, want all", consolidatedCount, len(obs))
	}
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND source = 'dream' AND deleted_at IS NULL`, svc.ProjectID()).Scan(&mergedCount); err != nil {
		t.Fatalf("count merged: %v", err)
	}
	if mergedCount != 1 {
		t.Fatalf("after consolidate, merged obs = %d, want 1", mergedCount)
	}

	// Roll back: restore every observation to its pre-distill state and release
	// the lock.
	if err := de.DreamRollback(ctx, svc.ProjectID(), pre, locks); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// Sources restored: status back to "active" and the row count back to the
	// pre-distill count (the merged observation is gone).
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`, svc.ProjectID()).Scan(&mergedCount); err != nil {
		t.Fatalf("count live after rollback: %v", err)
	}
	if mergedCount != len(obs) {
		t.Fatalf("live rows after rollback = %d, want %d (pre-distill count)", mergedCount, len(obs))
	}
	if err := svc.DB().QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ? AND status = ? AND deleted_at IS NULL`, svc.ProjectID(), dreamStatus).Scan(&consolidatedCount); err != nil {
		t.Fatalf("count consolidated after rollback: %v", err)
	}
	if consolidatedCount != 0 {
		t.Fatalf("consolidated rows after rollback = %d, want 0", consolidatedCount)
	}
	// Content round-tripped (every fact is present again).
	after, err := liveRows(t, svc)
	if err != nil {
		t.Fatalf("read live rows after rollback: %v", err)
	}
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
	if locked, err := locks.IsLocked(ctx, svc.ProjectID()); err != nil {
		t.Fatalf("is locked: %v", err)
	} else if locked {
		t.Fatalf("lock should be released after rollback")
	}
}

// liveRows reads the project's live (non-deleted) observations in id order.
func liveRows(t *testing.T, svc *Service) ([]Observation, error) {
	t.Helper()
	rows, err := svc.DB().QueryContext(context.Background(), `
		SELECT id, session_id, type, title, content, project, scope, topic_key,
		       source, normalized_hash, revision_count, created_at, updated_at,
		       COALESCE(visibility, 'private'), COALESCE(status, 'active'),
		       COALESCE(retrieval_usage, 0)
		FROM observations WHERE project = ? AND deleted_at IS NULL ORDER BY id`,
		svc.ProjectID())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Observation
	for rows.Next() {
		var o Observation
		if err := rows.Scan(&o.ID, &o.SessionID, &o.Type, &o.Title, &o.Content, &o.Project, &o.Scope,
			&o.TopicKey, &o.Source, &o.NormalizedHash, &o.RevisionCount, &o.CreatedAt, &o.UpdatedAt,
			&o.Visibility, &o.Status, &o.RetrievalUsage); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}
