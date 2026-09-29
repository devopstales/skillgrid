package facts

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "factstest")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

// TestAddCreatesFactRowAndEvent covers @step-02 (add portion): Add inserts one
// facts row with 014 importance defaults, an FTS-visible row, and a
// session_events trail with action_type=fact_add and the fact id in the
// payload JSON.
func TestAddCreatesFactRowAndEvent(t *testing.T) {
	st := openTestStore(t)
	db := st.DB

	sid := "sess-facts-test"
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`
		INSERT OR IGNORE INTO sessions (id, project, directory, started_at, status)
		VALUES (?, 'factstest', '/tmp', ?, 'active')`, sid, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	s := New(db, "factstest")
	id, err := s.Add(context.Background(), sid, "GDE pattern: cache the registry once per session")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if id <= 0 {
		t.Fatalf("Add returned id %d, want > 0", id)
	}

	// The facts row exists with the content and 014 defaults.
	var content string
	var score float64
	var decay float64
	var tier string
	var usage int
	var deletedAt sql.NullString
	if err := db.QueryRow(`
		SELECT content, importance_score, recency_decay, maturity_tier, retrieval_usage, deleted_at
		FROM facts WHERE id = ?`, id).
		Scan(&content, &score, &decay, &tier, &usage, &deletedAt); err != nil {
		t.Fatalf("read fact: %v", err)
	}
	if content != "GDE pattern: cache the registry once per session" {
		t.Errorf("content = %q", content)
	}
	if score != 1.0 || decay != 0.0 || tier != "new" || usage != 0 {
		t.Errorf("014 defaults wrong: score=%v decay=%v tier=%q usage=%d", score, decay, tier, usage)
	}
	if deletedAt.Valid {
		t.Errorf("deleted_at should be NULL for a fresh fact")
	}

	// The FTS index mirrors the row (content='facts' external table).
	var ftsCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM facts_fts WHERE facts_fts MATCH 'GDE'`).Scan(&ftsCount); err != nil {
		t.Fatalf("facts_fts match: %v", err)
	}
	if ftsCount != 1 {
		t.Errorf("facts_fts matches = %d, want 1", ftsCount)
	}

	// The session_events trail records action_type=fact_add with the id.
	var actionType, payload string
	if err := db.QueryRow(`
		SELECT action_type, payload FROM session_events
		WHERE session_id = ? AND action_type = 'fact_add'`, sid).
		Scan(&actionType, &payload); err != nil {
		t.Fatalf("fact_add session event missing: %v", err)
	}
	if !strings.Contains(payload, `"fact_id":`) || !strings.Contains(payload, strconv.FormatInt(id, 10)) {
		t.Errorf("payload %q does not carry the fact id", payload)
	}
}

// TestAddEmptyContentFails guards the required-content boundary.
func TestAddEmptyContentFails(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB, "factstest")
	if _, err := s.Add(context.Background(), "sess-empty", ""); err == nil {
		t.Fatal("Add with blank content should fail")
	}
	var n int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&n); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if n != 0 {
		t.Errorf("facts rows = %d, want 0", n)
	}
}

// TestAddMissingSessionFails: session_events.session_id has an FK to
// sessions(id); an unknown session must not insert a dangling fact or event.
func TestAddMissingSessionFails(t *testing.T) {
	st := openTestStore(t)
	s := New(st.DB, "factstest")
	if _, err := s.Add(context.Background(), "no-such-session", "fact body"); err == nil {
		t.Fatal("Add with unknown session should fail (FK)")
	}
	var factsN, eventsN int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&factsN); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM session_events WHERE action_type='fact_add'`).Scan(&eventsN); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if factsN != 0 || eventsN != 0 {
		t.Errorf("rollback failed: facts=%d events=%d, want 0/0", factsN, eventsN)
	}
}
