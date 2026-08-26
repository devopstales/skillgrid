package memory_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"skillgrid-cli/internal/mnemonic/memory"
	"skillgrid-cli/internal/mnemonic/store"
)

const testProject = "test-project"

func openTestService(t *testing.T) (*memory.Service, *store.Store) {
	t.Helper()

	dir := t.TempDir()
	st, err := store.Open(dir, testProject)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	svc := memory.New(st, testProject)
	return svc, st
}

func ensureSession(t *testing.T, st *store.Store, sessionID string) {
	t.Helper()

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := st.DB.Exec(
		`INSERT INTO sessions (id, project, directory, started_at, status) VALUES (?, ?, ?, ?, ?)`,
		sessionID, testProject, "/tmp/test", now, "active",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMemorySaveSearch(t *testing.T) {
	svc, st := openTestService(t)
	const sessionID = "sess-1"
	ensureSession(t, st, sessionID)

	ctx := context.Background()
	id, err := svc.Save(ctx, memory.SaveInput{
		Title:     "Chose SQLite",
		Type:      "decision",
		Content:   "Selected SQLite with FTS5 for local-first memory storage.",
		Scope:     "project",
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	hits, err := svc.Search(ctx, "SQLite", "any", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d", len(hits))
	}
	if hits[0].Title != "Chose SQLite" {
		t.Fatalf("title=%q", hits[0].Title)
	}
}

func TestMemoryDedup24h(t *testing.T) {
	svc, st := openTestService(t)
	const sessionID = "sess-dedup"
	ensureSession(t, st, sessionID)

	ctx := context.Background()
	input := memory.SaveInput{
		Title:     "Chose SQLite",
		Type:      "decision",
		Content:   "Same content.",
		Scope:     "project",
		SessionID: sessionID,
	}

	id1, err := svc.Save(ctx, input)
	if err != nil {
		t.Fatal(err)
	}

	id2, err := svc.Save(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("expected dedup id %d, got %d", id1, id2)
	}

	hits, err := svc.Search(ctx, "SQLite", "any", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit after dedup, got %d", len(hits))
	}
}

func TestMemoryTopicKeyUpsert(t *testing.T) {
	svc, st := openTestService(t)
	const sessionID = "sess-topic"
	ensureSession(t, st, sessionID)

	ctx := context.Background()
	const topicKey = "auth-strategy"

	id1, err := svc.Save(ctx, memory.SaveInput{
		Title:     "Auth v1",
		Type:      "decision",
		Content:   "JWT tokens.",
		Scope:     "project",
		TopicKey:  topicKey,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatal(err)
	}

	id2, err := svc.Save(ctx, memory.SaveInput{
		Title:     "Auth v2",
		Type:      "decision",
		Content:   "Session cookies instead.",
		Scope:     "project",
		TopicKey:  topicKey,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("expected topic_key upsert to reuse id %d, got %d", id1, id2)
	}

	obs, err := svc.Get(ctx, id2)
	if err != nil {
		t.Fatal(err)
	}
	if obs.Title != "Auth v2" {
		t.Fatalf("title=%q", obs.Title)
	}
	if obs.RevisionCount != 1 {
		t.Fatalf("revision_count=%d, want 1", obs.RevisionCount)
	}
}

func TestMemorySearchMatchMode(t *testing.T) {
	svc, st := openTestService(t)
	const sessionID = "sess-search"
	ensureSession(t, st, sessionID)

	ctx := context.Background()
	for _, obs := range []memory.SaveInput{
		{Title: "SQLite storage", Type: "decision", Content: "Local database.", Scope: "project", SessionID: sessionID},
		{Title: "Redis cache", Type: "decision", Content: "In-memory cache layer.", Scope: "project", SessionID: sessionID},
	} {
		if _, err := svc.Save(ctx, obs); err != nil {
			t.Fatal(err)
		}
	}

	allHits, err := svc.Search(ctx, "SQLite Redis", "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(allHits) != 0 {
		t.Fatalf("all mode expected 0 hits, got %d", len(allHits))
	}

	anyHits, err := svc.Search(ctx, "SQLite Redis", "any", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(anyHits) != 2 {
		t.Fatalf("any mode expected 2 hits, got %d", len(anyHits))
	}
}

func TestMemoryGet(t *testing.T) {
	svc, st := openTestService(t)
	const sessionID = "sess-get"
	ensureSession(t, st, sessionID)

	ctx := context.Background()
	id, err := svc.Save(ctx, memory.SaveInput{
		Title:     "Test observation",
		Type:      "discovery",
		Content:   "Details here.",
		Scope:     "project",
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatal(err)
	}

	obs, err := svc.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if obs.ID != id || obs.Type != "discovery" {
		t.Fatalf("unexpected observation: %+v", obs)
	}
}

func TestSessionLifecycle(t *testing.T) {
	workspace := t.TempDir()
	cfgDir := filepath.Join(workspace, ".skillgrid")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(cfgDir, "config.json"),
		[]byte(`{"project":"test-project"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	svc, _ := openTestService(t)
	ctx := context.Background()

	sessionID, err := svc.SessionStart(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if sessionID == "" {
		t.Fatal("expected non-empty session id")
	}

	const summary = "Implemented session lifecycle and recent context APIs."
	if err := svc.SessionSummary(ctx, sessionID, summary); err != nil {
		t.Fatal(err)
	}

	sessions, err := svc.RecentContext(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session in recent context, got %d", len(sessions))
	}
	if sessions[0].ID != sessionID {
		t.Fatalf("session id=%q, want %q", sessions[0].ID, sessionID)
	}
	if sessions[0].Summary != summary {
		t.Fatalf("summary=%q, want %q", sessions[0].Summary, summary)
	}
	if sessions[0].Status != "active" {
		t.Fatalf("status=%q, want active", sessions[0].Status)
	}
}

func TestSessionEnd(t *testing.T) {
	workspace := t.TempDir()
	cfgDir := filepath.Join(workspace, ".skillgrid")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(cfgDir, "config.json"),
		[]byte(`{"project":"test-project"}`),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	svc, _ := openTestService(t)
	ctx := context.Background()

	sessionID, err := svc.SessionStart(ctx, workspace)
	if err != nil {
		t.Fatal(err)
	}

	const summary = "Closed session after indexing pass."
	if err := svc.SessionEnd(ctx, sessionID, summary); err != nil {
		t.Fatal(err)
	}

	sessions, err := svc.RecentContext(ctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Status != "ended" {
		t.Fatalf("status=%q, want ended", sessions[0].Status)
	}
	if sessions[0].Summary != summary {
		t.Fatalf("summary=%q, want %q", sessions[0].Summary, summary)
	}
}
