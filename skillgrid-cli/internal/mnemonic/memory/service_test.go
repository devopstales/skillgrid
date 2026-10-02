package memory

import (
	"context"
	"database/sql"
	"fmt"
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
	id, err := svc.SessionStart(context.Background(), "/tmp/any-dir", "any-dir session")
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
	// One title-only session (no summary), like the dashboard list expects.
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, title, started_at, status)
		VALUES ('s3', 'recent', '/tmp', 'Skillgrid CLI dashboard status card updates', '2026-01-02T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert titled: %v", err)
	}
	// One session with neither title nor summary must be excluded.
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s4', 'recent', '/tmp', '2026-01-03T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert bare: %v", err)
	}
	sessions, err := svc.RecentContext(ctx, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions (with title or summary), got %d", len(sessions))
	}
	var titled *Session
	for i := range sessions {
		if sessions[i].ID == "s3" {
			titled = &sessions[i]
		}
	}
	if titled == nil {
		t.Fatalf("title-only session missing from recent context")
	}
	if titled.Title != "Skillgrid CLI dashboard status card updates" {
		t.Errorf("expected seed title, got %q", titled.Title)
	}
}

// TestSessionTitleRoundTrip verifies the title column round-trips: an empty
// title stores NULL (unnamed session) and SessionSetTitle renames by id.
func TestSessionTitleRoundTrip(t *testing.T) {
	st, err := openStoreFor(t, "titled")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()
	svc := New(st, "titled")
	ctx := context.Background()
	// Unnamed session -> title must be NULL.
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, title, started_at, status)
		VALUES ('unnamed', 'titled', '/tmp', NULL, '2026-01-01T00:00:00Z', 'active')`); err != nil {
		t.Fatalf("insert unnamed: %v", err)
	}
	var nilTitle sql.NullString
	if err := st.DB.QueryRow(`SELECT title FROM sessions WHERE id = 'unnamed'`).Scan(&nilTitle); err != nil {
		t.Fatalf("read nil title: %v", err)
	}
	if nilTitle.Valid {
		t.Errorf("expected NULL title, got %q", nilTitle.String)
	}
	// SessionSetTitle renames it.
	if err := svc.SessionSetTitle(ctx, "unnamed", "Skillgrid CLI dashboard status card updates"); err != nil {
		t.Fatalf("set title: %v", err)
	}
	var got string
	if err := st.DB.QueryRow(`SELECT title FROM sessions WHERE id = 'unnamed'`).Scan(&got); err != nil {
		t.Fatalf("read title: %v", err)
	}
	if got != "Skillgrid CLI dashboard status card updates" {
		t.Errorf("title = %q, want seed title", got)
	}
	// Unknown session id surfaces as an error.
	if err := svc.SessionSetTitle(ctx, "missing", "x"); err == nil {
		t.Errorf("expected error for unknown session id")
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

func TestBuildFTSQueryTrigramMode(t *testing.T) {
	got := buildFTSQuery("hello", "trigram")
	for _, tg := range []string{`"hel"`, `"ell"`, `"llo"`} {
		if !strings.Contains(got, tg) {
			t.Errorf("trigram mode missing trigram %s in %q", tg, got)
		}
	}
	if !strings.Contains(got, " OR ") {
		t.Errorf("trigram fragments must be OR-joined, got %q", got)
	}
	// Default mode is unchanged: a single term stays a single quoted phrase.
	if got := buildFTSQuery("hello", ""); got != `"hello"` {
		t.Errorf("default mode = %q, want %q", got, `"hello"`)
	}
	// A term shorter than 3 chars uses the whole term as a single "trigram".
	if got := buildFTSQuery("hi", "trigram"); got != `"hi"` {
		t.Errorf("short-term trigram = %q, want %q", got, `"hi"`)
	}
	// Empty query falls back to the default behavior (empty query string).
	if got := buildFTSQuery("", "trigram"); got != "" {
		t.Errorf("empty-query trigram fallback = %q, want empty", got)
	}
	// Prefix mode appends * to the quoted term.
	if got := buildFTSQuery("fun", "prefix"); got != `"fun*"` {
		t.Errorf("prefix mode = %q, want %q", got, `"fun*"`)
	}
	// Multi-term prefix: each term gets the wildcard, OR-joined.
	if got := buildFTSQuery("fun bar", "prefix"); got != `"fun*" OR "bar*"` {
		t.Errorf("multi-term prefix = %q, want %q", got, `"fun*" OR "bar*"`)
	}
}

func TestFTSPhraseModeUnchanged(t *testing.T) {
	// Empty mode (default) is byte-identical to the pre-change behavior:
	// OR-joined double-quoted terms, no trigram/prefix transformation.
	if got := buildFTSQuery("exact phrase match", ""); got != `"exact" OR "phrase" OR "match"` {
		t.Errorf("default mode = %q, want OR-joined quoted terms", got)
	}
	// "all" mode keeps AND-joined quoted terms.
	if got := buildFTSQuery("exact phrase match", "all"); got != `"exact" AND "phrase" AND "match"` {
		t.Errorf("all mode = %q, want AND-joined quoted terms", got)
	}
	// Quoting/escaping is preserved.
	if got := buildFTSQuery(`say "hi"`, ""); got != `"say" OR """hi"""` {
		t.Errorf("escaped quotes = %q", got)
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

func TestDeriveSessionTitle(t *testing.T) {
	cases := []struct {
		name    string
		summary string
		want    string
	}{
		{
			name:    "goal line",
			summary: "## Goal\nRewrite engram-memory skills to target Mnemonic.\n\n## Instructions\n- surgical\n\n## Accomplished\n- done",
			want:    "Rewrite engram-memory skills to target Mnemonic.",
		},
		{
			name:    "goal with trailing blank then next section",
			summary: "## Goal\nCreate a SEPARATE openspec change for a full Cypher query engine.\n\n## Instructions\n- Approach A",
			want:    "Create a SEPARATE openspec change for a full Cypher query engine.",
		},
		{
			name:    "no goal heading falls back to first content line",
			summary: "Some free-form note.\nMore text.",
			want:    "Some free-form note.",
		},
		{
			name:    "goal heading only, no content -> first content line later",
			summary: "## Goal\n\n## Accomplished\n- shipped it",
			want:    "- shipped it",
		},
		{"empty summary", "", ""},
	}
	for _, c := range cases {
		if got := deriveSessionTitle(c.summary); got != c.want {
			t.Errorf("%s: deriveSessionTitle = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestDisplayTitleFallback(t *testing.T) {
	const sid = "dccb5c02-c96c-4930-842b-1e41de6d4261"
	if got := displayTitle(sid, "explicit name", "## Goal\nignored"); got != "explicit name" {
		t.Errorf("explicit should win, got %q", got)
	}
	if got := displayTitle(sid, "", "## Goal\nRewrite engram-memory skills to target Mnemonic.\n\n## Accomplished\n- done"); got != "Rewrite engram-memory skills to target Mnemonic." {
		t.Errorf("goal line fallback, got %q", got)
	}
	// No title, no summary -> fall back to the session id (no vague placeholder).
	if got := displayTitle(sid, "", ""); got != "dccb5c02" {
		t.Errorf("id fallback, got %q", got)
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

// saveResultObservationID is a test helper that pulls the row count out of the
// observations table for the active project, used to assert whether SaveWithAction
// wrote a new row or not.
func obsCount(t *testing.T, st *store.Store, project string) int {
	t.Helper()
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE project = ?`, project).Scan(&n); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	return n
}

// TestSaveWithActionRouting covers the real 4-way AUDN routing table
// (TICKET-04, ADR-0011). With the LLM pass armed, each verdict routes to its
// documented behavior:
//   - noop   → BumpDuplicate(candidateID), no new row
//   - add    → new row (insertObservation)
//   - update → topic-key-style upsert into candidateID (same row, no new row)
//   - delete → new row THEN MarkSuperseded(old=candidate, new=newRow)
func TestSaveWithActionRouting(t *testing.T) {
	ctx := context.Background()
	project := "audn-routing"

	t.Run("noop bumps duplicate, no new row", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		// Seed one observation so the pre-filter returns a candidate.
		seed, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "seed note", Content: "seed content",
		})
		if err != nil {
			t.Fatalf("seed save: %v", err)
		}
		_ = seed
		before := obsCount(t, svc.store, project)

		llm := &dedupLLM{verdict: VerdictNoop, candidate: int(seed)}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "noop note", Content: "noop content",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictNoop {
			t.Errorf("action = %q, want noop", res.Action)
		}
		if res.ObservationID != seed {
			t.Errorf("observationID = %d, want candidate %d", res.ObservationID, seed)
		}
		if res.SupersededID != 0 {
			t.Errorf("supersededID = %d, want 0 for noop", res.SupersededID)
		}
		after := obsCount(t, svc.store, project)
		if after != before {
			t.Errorf("row count changed: %d -> %d, want no new row", before, after)
		}
		// Verify duplicate_count was bumped on the seed row.
		var dup int
		if err := svc.store.DB.QueryRow(
			`SELECT COALESCE(duplicate_count,0) FROM observations WHERE id = ?`, seed,
		).Scan(&dup); err != nil {
			t.Fatalf("read dup count: %v", err)
		}
		if dup < 1 {
			t.Errorf("duplicate_count = %d, want >=1 after noop", dup)
		}
	})

	t.Run("add writes a new row", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		// Seed one observation so the pre-filter returns a candidate (so the
		// LLM pass is actually armed and called).
		seed, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "learning",
			Title: "seed other", Content: "seed other content",
		})
		if err != nil {
			t.Fatalf("seed save: %v", err)
		}
		_ = seed
		before := obsCount(t, svc.store, project)

		llm := &dedupLLM{verdict: VerdictAdd, candidate: 0}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "learning",
			Title: "add note", Content: "add content",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictAdd {
			t.Errorf("action = %q, want add", res.Action)
		}
		if res.ObservationID <= 0 {
			t.Errorf("observationID = %d, want a new positive id", res.ObservationID)
		}
		if res.SupersededID != 0 {
			t.Errorf("supersededID = %d, want 0 for add", res.SupersededID)
		}
		after := obsCount(t, svc.store, project)
		if after != before+1 {
			t.Errorf("row count: %d -> %d, want %d (new row)", before, after, before+1)
		}
	})

	t.Run("update upserts into candidate, no new row", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		// Seed the target observation.
		seed, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "target note", Content: "target content",
		})
		if err != nil {
			t.Fatalf("seed save: %v", err)
		}
		before := obsCount(t, svc.store, project)

		llm := &dedupLLM{verdict: VerdictUpdate, candidate: int(seed)}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "updated title", Content: "updated content",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictUpdate {
			t.Errorf("action = %q, want update", res.Action)
		}
		// Update routes into the candidate row: the returned id IS the
		// candidate (no new row), and the candidate's content/title is
		// rewritten in place.
		if res.ObservationID != seed {
			t.Errorf("observationID = %d, want candidate %d", res.ObservationID, seed)
		}
		if res.SupersededID != 0 {
			t.Errorf("supersededID = %d, want 0 for update", res.SupersededID)
		}
		after := obsCount(t, svc.store, project)
		if after != before {
			t.Errorf("row count changed: %d -> %d, want no new row", before, after)
		}
		// The candidate row now holds the updated content.
		var newContent string
		if err := svc.store.DB.QueryRow(
			`SELECT content FROM observations WHERE id = ?`, seed,
		).Scan(&newContent); err != nil {
			t.Fatalf("read content: %v", err)
		}
		if newContent != "updated content" {
			t.Errorf("candidate content = %q, want %q", newContent, "updated content")
		}
	})

	t.Run("delete inserts new row then supersedes candidate", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		seed, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "superseded note", Content: "superseded content",
		})
		if err != nil {
			t.Fatalf("seed save: %v", err)
		}
		before := obsCount(t, svc.store, project)

		llm := &dedupLLM{verdict: VerdictDelete, candidate: int(seed)}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "newer note", Content: "newer content",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictDelete {
			t.Errorf("action = %q, want delete", res.Action)
		}
		if res.ObservationID <= 0 || res.ObservationID == seed {
			t.Errorf("observationID = %d, want a new positive id distinct from candidate %d", res.ObservationID, seed)
		}
		if res.SupersededID != seed {
			t.Errorf("supersededID = %d, want candidate %d", res.SupersededID, seed)
		}
		after := obsCount(t, svc.store, project)
		if after != before+1 {
			t.Errorf("row count: %d -> %d, want %d (new row)", before, after, before+1)
		}
		// The old row is now marked superseded.
		var oldStatus, oldSupersededBy string
		if err := svc.store.DB.QueryRow(
			`SELECT status, COALESCE(CAST(superseded_by AS TEXT), '') FROM observations WHERE id = ?`, seed,
		).Scan(&oldStatus, &oldSupersededBy); err != nil {
			t.Fatalf("read old row: %v", err)
		}
		if oldStatus != "superseded" {
			t.Errorf("old status = %q, want superseded", oldStatus)
		}
		if oldSupersededBy != fmt.Sprintf("%d", res.ObservationID) {
			t.Errorf("old superseded_by = %q, want %d", oldSupersededBy, res.ObservationID)
		}
		// The supersedes edge (old -> new) exists.
		var edgeN int
		if err := svc.store.DB.QueryRow(
			`SELECT COUNT(*) FROM memory_relations
			 WHERE src_obs_id = ? AND dst_obs_id = ? AND relation = 'supersedes' AND deleted_at IS NULL`,
			seed, res.ObservationID,
		).Scan(&edgeN); err != nil {
			t.Fatalf("read edge: %v", err)
		}
		if edgeN != 1 {
			t.Errorf("supersedes edge count = %d, want 1", edgeN)
		}
	})

	t.Run("hash floor: LLM error is non-fatal, falls to add", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		seed, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "learning",
			Title: "seed x", Content: "seed x content",
		})
		if err != nil {
			t.Fatalf("seed save: %v", err)
		}
		_ = seed
		before := obsCount(t, svc.store, project)

		// LLM errors: runDedupCheck must return the hash floor (Reason="hash",
		// Verdict=""), and SaveWithAction must fall through to add — no panic,
		// no error.
		llm := &dedupLLM{err: fmt.Errorf("llm down")}
		svc.SetDedupLLM(llm)
		svc.EnableDedupLLM(true)

		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "learning",
			Title: "floor note", Content: "floor content",
		})
		if err != nil {
			t.Fatalf("save with action (llm error): %v", err)
		}
		if res.Action != VerdictAdd {
			t.Errorf("action = %q, want add (hash floor)", res.Action)
		}
		if res.ObservationID <= 0 {
			t.Errorf("observationID = %d, want a new positive id", res.ObservationID)
		}
		after := obsCount(t, svc.store, project)
		if after != before+1 {
			t.Errorf("row count: %d -> %d, want %d (new row)", before, after, before+1)
		}
	})

	t.Run("hash floor: exact duplicate is noop+bump", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		first, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "dup note", Content: "dup content",
		})
		if err != nil {
			t.Fatalf("first save: %v", err)
		}
		before := obsCount(t, svc.store, project)

		// No LLM armed (default): the deterministic hash floor fires. An exact
		// duplicate is a noop + BumpDuplicate, returning the existing id.
		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "dup note", Content: "dup content",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictNoop {
			t.Errorf("action = %q, want noop (hash floor)", res.Action)
		}
		if res.ObservationID != first {
			t.Errorf("observationID = %d, want existing %d", res.ObservationID, first)
		}
		after := obsCount(t, svc.store, project)
		if after != before {
			t.Errorf("row count changed: %d -> %d, want no new row", before, after)
		}
	})
}

// TestSaveDelegatesToSaveWithAction asserts that Service.Save is now a thin
// delegate to SaveWithAction: for the common (non-LLM) path it must return the
// same (id, error) it always did, and the underlying row state must be identical
// to a direct SaveWithAction call.
func TestSaveDelegatesToSaveWithAction(t *testing.T) {
	ctx := context.Background()
	project := "audn-delegate"

	t.Run("hash-hit returns existing id (preserves old Save contract)", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		first, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "delegate dup", Content: "delegate content",
		})
		if err != nil {
			t.Fatalf("first save: %v", err)
		}
		// Second identical save: old Save returned the EXISTING row id. The
		// delegate must preserve that exact contract.
		second, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "delegate dup", Content: "delegate content",
		})
		if err != nil {
			t.Fatalf("second save: %v", err)
		}
		if second != first {
			t.Errorf("delegate Save returned %d, want existing %d", second, first)
		}
	})

	t.Run("new row returns new id", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		id, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "learning",
			Title: "delegate new", Content: "delegate new content",
		})
		if err != nil {
			t.Fatalf("save: %v", err)
		}
		if id <= 0 {
			t.Errorf("save returned id %d, want positive", id)
		}
		obs, err := svc.Get(ctx, id)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if obs.Title != "delegate new" {
			t.Errorf("title = %q, want %q", obs.Title, "delegate new")
		}
	})

	t.Run("delegate matches direct SaveWithAction (add path)", func(t *testing.T) {
		_, svc := newTestStore(t, project)
		sid := newSession(t, svc)
		// No LLM armed: both Save and SaveWithAction take the add path.
		id, err := svc.Save(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "parity note", Content: "parity content",
		})
		if err != nil {
			t.Fatalf("save: %v", err)
		}
		res, err := svc.SaveWithAction(ctx, SaveInput{
			SessionID: sid, Type: "decision",
			Title: "parity note 2", Content: "parity content 2",
		})
		if err != nil {
			t.Fatalf("save with action: %v", err)
		}
		if res.Action != VerdictAdd {
			t.Errorf("action = %q, want add", res.Action)
		}
		if res.ObservationID == id {
			t.Errorf("SaveWithAction returned the same id %d as the prior Save; expected a distinct new row", id)
		}
	})
}
