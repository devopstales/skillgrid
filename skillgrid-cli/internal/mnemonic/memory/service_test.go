package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

type fixture struct {
	svc    *Service
	st     *store.Store
	clean  func()
	sessID string
}

func newFixture(t *testing.T, project string) *fixture {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir, project)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	clean := func() { st.Close() }
	t.Cleanup(clean)
	svc := New(st, project)
	var sessID string
	res, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1', ?, '/tmp', '2026-01-01T00:00:00Z', 'active')`, project)
	if err != nil {
		t.Fatalf("insert session: %v", err)
	}
	_ = res
	sessID = "s1"
	return &fixture{svc: svc, st: st, clean: clean, sessID: sessID}
}

const session1 = "s1"

func TestSaveAndSearch(t *testing.T) {
	fx := newFixture(t, "mem-test")
	ctx := context.Background()
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: session1,
		Type:      "decision",
		Title:     "Chose SQLite for local store",
		Content:   "Why: single binary. Where: internal/mnemonic.",
	}); err != nil {
		t.Fatalf("save: %v", err)
	}
	hits, err := fx.svc.Search(ctx, "sqlite", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Title != "Chose SQLite for local store" {
		t.Errorf("unexpected title: %q", hits[0].Title)
	}

	// Get by id should return the same observation.
	got, err := fx.svc.Get(ctx, hits[0].ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != hits[0].ID {
		t.Errorf("get/id mismatch: %d != %d", got.ID, hits[0].ID)
	}
	if got.Content != "Why: single binary. Where: internal/mnemonic." {
		t.Errorf("get/content mismatch: %q", got.Content)
	}
}

func TestSaveRejectsInvalidType(t *testing.T) {
	fx := newFixture(t, "mem-test")
	_, err := fx.svc.Save(context.Background(), SaveInput{
		SessionID: session1,
		Type:      "nonexistent-type",
		Title:     "T",
		Content:   "C",
	})
	if err == nil {
		t.Fatalf("expected error for invalid type")
	}
	if !strings.Contains(err.Error(), "nonexistent-type") {
		t.Errorf("error should name the invalid type: %v", err)
	}
	// And no observation row should have been inserted.
	var n int
	if err := fx.st.DB.QueryRow(`SELECT COUNT(*) FROM observations`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 observation rows, got %d", n)
	}
}

func TestSaveDedupWithin24h(t *testing.T) {
	fx := newFixture(t, "mem-test")
	ctx := context.Background()
	in := SaveInput{
		SessionID: session1,
		Type:      "discovery",
		Title:     "Found gotcha",
		Content:   "body",
	}
	id1, err := fx.svc.Save(ctx, in)
	if err != nil {
		t.Fatalf("save1: %v", err)
	}
	id2, err := fx.svc.Save(ctx, in)
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same id for duplicate, got %d and %d", id1, id2)
	}
}

func TestTopicKeyUpsert(t *testing.T) {
	fx := newFixture(t, "mem-test")
	ctx := context.Background()
	in1 := SaveInput{
		SessionID: session1,
		Type:      "decision",
		Title:     "auth-model",
		Content:   "v1",
		TopicKey:  "architecture/auth-model",
	}
	id1, err := fx.svc.Save(ctx, in1)
	if err != nil {
		t.Fatalf("save1: %v", err)
	}
	in2 := SaveInput{
		SessionID: session1,
		Type:      "decision",
		Title:     "auth-model updated",
		Content:   "v2",
		TopicKey:  "architecture/auth-model",
	}
	id2, err := fx.svc.Save(ctx, in2)
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	if id1 != id2 {
		t.Errorf("expected same id for topic-key upsert, got %d and %d", id1, id2)
	}
	got, err := fx.svc.Get(ctx, id1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Content != "v2" {
		t.Errorf("expected upserted content v2, got %q", got.Content)
	}
	if got.RevisionCount != 1 {
		t.Errorf("expected revision_count 1, got %d", got.RevisionCount)
	}
}

func TestSessionLifecycle(t *testing.T) {
	// Create a session row directly so the test focuses on summary/end
	// behavior. (SessionStart re-resolves the project from the directory,
	// which won't match "lifecycle"; the summary/end paths are the
	// spec-relevant units under test here.)
	st, err := openStoreFor(t, "lifecycle")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "lifecycle")
	id := "sess-lifecycle-1"
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'lifecycle', '/tmp', '2026-01-01T00:00:00Z', 'active')`, id); err != nil {
		t.Fatalf("insert session: %v", err)
	}
	// Summary before end is allowed by the service API.
	if err := svc.SessionSummary(context.Background(), id, "## Goal\ndone"); err != nil {
		t.Fatalf("summary: %v", err)
	}
	// End should succeed and record the summary.
	if err := svc.SessionEnd(context.Background(), id, "## Goal\ndone (final)"); err != nil {
		t.Fatalf("end: %v", err)
	}
	var summary, status string
	if err := st.DB.QueryRow(`SELECT summary, status FROM sessions WHERE id = ?`, id).Scan(&summary, &status); err != nil {
		t.Fatalf("read session: %v", err)
	}
	if status != "ended" {
		t.Errorf("expected status 'ended', got %q", status)
	}
	if summary != "## Goal\ndone (final)" {
		t.Errorf("expected final summary, got %q", summary)
	}
}

func TestSessionStartCreatesRow(t *testing.T) {
	st, err := openStoreFor(t, "start")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "start")
	id, err := svc.SessionStart(context.Background(), "/tmp/any-dir")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if id == "" {
		t.Fatalf("expected non-empty session id")
	}
	// The row should exist, scoped to whatever project was resolved for the
	// directory. The test asserts presence + non-empty id.
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM sessions WHERE id = ?`, id).Scan(&n); err != nil {
		t.Fatalf("read: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 row, got %d", n)
	}
}

func TestRecentContext(t *testing.T) {
	st, err := openStoreFor(t, "recent")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "recent")
	ctx := context.Background()
	// Insert two sessions with summaries.
	ids := []string{"s1", "s2"}
	for _, id := range ids {
		if _, err := st.DB.Exec(`
			INSERT INTO sessions (id, project, directory, started_at, summary, status)
			VALUES (?, 'recent', '/tmp', '2026-01-01T00:00:00Z', '## Goal\nwork', 'ended')`, id); err != nil {
			t.Fatalf("insert %s: %v", id, err)
		}
	}
	sessions, err := svc.RecentContext(ctx, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
}

func TestSaveRequiresSessionID(t *testing.T) {
	fx := newFixture(t, "mem-test")
	_, err := fx.svc.Save(context.Background(), SaveInput{
		Type:    "decision",
		Title:   "T",
		Content: "C",
	})
	if err == nil {
		t.Errorf("expected error for missing session_id")
	}
	if !strings.Contains(err.Error(), "session_id") {
		t.Errorf("error should mention session_id: %v", err)
	}
}

func TestSaveRequiresTitle(t *testing.T) {
	fx := newFixture(t, "mem-test")
	_, err := fx.svc.Save(context.Background(), SaveInput{
		SessionID: session1,
		Type:      "decision",
		Title:     "",
		Content:   "C",
	})
	if err == nil {
		t.Errorf("expected error for empty title")
	}
	if !strings.Contains(err.Error(), "title") {
		t.Errorf("error should mention title: %v", err)
	}
}

func TestSaveRequiresContent(t *testing.T) {
	fix := fx(t, "mem-test")
	_, err := fix.svc.Save(context.Background(), SaveInput{SessionID: session1, Type: "decision", Title: "T"})
	if err == nil {
		t.Errorf("expected error for empty content")
	}
	if err != nil && !strings.Contains(err.Error(), "content") {
		t.Errorf("error should mention content: %v", err)
	}
}

func TestSearchMatchMode(t *testing.T) {
	fx := newFixture(t, "mem-test")
	ctx := context.Background()
	// Two observations, one with "authentication", one with "authorization".
	if _, err := fx.svc.Save(ctx, SaveInput{SessionID: session1, Type: "decision", Title: "auth A", Content: "authentication flow"}); err != nil {
		t.Fatalf("save1: %v", err)
	}
	if _, err := fx.svc.Save(ctx, SaveInput{SessionID: session1, Type: "decision", Title: "auth B", Content: "authorization policy"}); err != nil {
		t.Fatalf("save2: %v", err)
	}
	any, err := fx.svc.Search(ctx, "authentication authorization", "any", 10)
	if err != nil {
		t.Fatalf("search any: %v", err)
	}
	all, err := fx.svc.Search(ctx, "authentication authorization", "all", 10)
	if err != nil {
		t.Fatalf("search all: %v", err)
	}
	if len(any) < 1 {
		t.Errorf("expected at least 1 any hit, got %d", len(any))
	}
	if len(all) > len(any) {
		t.Errorf("expected all <= any, got %d > %d", len(all), len(any))
	}
}

func TestIsValidType(t *testing.T) {
	cases := map[string]bool{
		"decision":     true,
		"architecture": true,
		"bugfix":       true,
		"pattern":      true,
		"config":       true,
		"correction":   true,
		"discovery":    true,
		"learning":     true,
		"lesson":       true,
		"preference":   true,
		"convention":   true,
		"standing":     true,
		"session_log":  true,
		"nonsense":     false,
		"":             false,
		"DECISION":     true, // case-insensitive
	}
	for in, want := range cases {
		if got := IsValidType(in); got != want {
			t.Errorf("IsValidType(%q) = %v, want %v", in, got, want)
		}
	}
}

// fx is a sugar helper for tests that only need the service+fixture.
func fx(t *testing.T, project string) *fixture {
	return newFixture(t, project)
}

// openStoreFor opens a fresh store (no pre-created session) under a temp dir.
func openStoreFor(t *testing.T, project string) (*store.Store, error) {
	t.Helper()
	dataDir := t.TempDir()
	return store.Open(dataDir, project)
}
