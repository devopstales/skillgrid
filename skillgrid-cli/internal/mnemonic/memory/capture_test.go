package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSavePromptTruncatesAndRejectsTiny(t *testing.T) {
	fx := newFixture(t, "prompt-test")
	ctx := context.Background()

	if _, err := fx.svc.SavePrompt(ctx, PromptInput{SessionID: session1, Content: "ok"}); err == nil {
		t.Fatal("expected too-small prompt to be rejected")
	}

	long := strings.Repeat("word ", 900) // ~4500 chars
	if len(long) <= MaxPromptLength {
		t.Fatalf("test premise: len=%d", len(long))
	}
	id, err := fx.svc.SavePrompt(ctx, PromptInput{SessionID: session1, Content: long})
	if err != nil {
		t.Fatalf("save prompt: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero prompt id")
	}
}

func TestCapturePassiveExtractsKeyLearnings(t *testing.T) {
	fx := newFixture(t, "capture-test")
	ctx := context.Background()

	text := `Some narrative output.

## Key Learnings:

1. bcrypt cost=12 is the right balance for our server
2. JWT refresh tokens need atomic rotation to avoid races
3. FTS5 queries must be sanitized before MATCH`

	res, err := fx.svc.CapturePassive(ctx, PassiveInput{
		Content:   text,
		SessionID: session1,
		Source:    "task-complete",
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if res.Saved == 0 {
		t.Fatalf("expected some captures, got %d (skipped %d)", res.Saved, res.Skipped)
	}

	before, ok := fx.obsCount(t)
	if !ok {
		t.Fatal("count before")
	}
	// Idempotency: re-running on the same text must not add new rows.
	res2, err := fx.svc.CapturePassive(ctx, PassiveInput{
		Content:   text,
		SessionID: session1,
	})
	if err != nil {
		t.Fatalf("re-capture: %v", err)
	}
	after, ok := fx.obsCount(t)
	if !ok {
		t.Fatal("count after")
	}
	if after != before {
		t.Fatalf("idempotency: rows grew from %d to %d (saved=%d)", before, after, res2.Saved)
	}
}

func TestCapturePassiveNoLearnings(t *testing.T) {
	fx := newFixture(t, "capture-empty")
	ctx := context.Background()
	res, err := fx.svc.CapturePassive(ctx, PassiveInput{
		Content:   "Just a paragraph with no structured learnings at all here today, friend.",
		SessionID: session1,
	})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	if res.Saved != 0 {
		t.Fatalf("expected 0 saved, got %d", res.Saved)
	}
}

func TestSessionStartByClientIDIdempotent(t *testing.T) {
	fx := newFixture(t, "clientid-test")
	ctx := context.Background()

	// Point the directory's project resolution at the fixture project so the
	// newly-registered session shares the fixture's store/project scope.
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.json"),
		[]byte(`{"project":"clientid-test"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	id, proj, existed, err := fx.svc.SessionStartByClientID(ctx, "abc-123", dir, "First")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	if id != "abc-123" || proj != "clientid-test" || existed {
		t.Fatalf("first: id=%q proj=%q existed=%v", id, proj, existed)
	}
	id2, _, existed2, err := fx.svc.SessionStartByClientID(ctx, "abc-123", dir, "Second")
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if id2 != "abc-123" || !existed2 {
		t.Fatalf("second: id=%q existed=%v", id2, existed2)
	}
}

func TestProjectMigrationSameIDIsNoop(t *testing.T) {
	fx := newFixture(t, "migrate-src")
	ctx := context.Background()
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1, Type: "decision", Title: "X", Content: "Y",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	moved, err := fx.svc.MigrateProjects(ctx, "migrate-src", "migrate-src")
	if err != nil || moved != 0 {
		t.Fatalf("same-id migrate: moved=%d err=%v", moved, err)
	}
}

func TestLastObservationAt(t *testing.T) {
	fx := newFixture(t, "lastat-test")
	ctx := context.Background()
	if before, err := fx.svc.LastObservationAt(ctx); err != nil || !before.IsZero() {
		t.Fatalf("before: %v %v", before, err)
	}
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1, Type: "learning", Title: "Z", Content: "W",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	after, err := fx.svc.LastObservationAt(ctx)
	if err != nil || after.IsZero() {
		t.Fatalf("after: %v %v", after, err)
	}
	if after.After(time.Now().UTC().Add(time.Minute)) {
		t.Fatalf("after in the future: %v", after)
	}
}

func TestCompactionContextIncludesRecentObs(t *testing.T) {
	fx := newFixture(t, "compact-test")
	ctx := context.Background()
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1, Type: "bugfix", Title: "Fixed N+1 in UserList", Content: "**What**: indexed query",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	c, err := fx.svc.CompactionContext(ctx, session1, 5)
	if err != nil {
		t.Fatalf("compact: %v", err)
	}
	if len(c.Observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(c.Observations))
	}
}

// obsCount returns the number of live observations for the fixture project.
func (fx *fixture) obsCount(t *testing.T) (int, bool) {
	t.Helper()
	var n int
	err := fx.st.DB.QueryRow(
		`SELECT COUNT(*) FROM observations WHERE project = ? AND deleted_at IS NULL`,
		fx.svc.projectID,
	).Scan(&n)
	if err != nil {
		return 0, false
	}
	return n, true
}
