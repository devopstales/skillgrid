package store

import (
	"testing"
	"time"
)

// TestMigration011FactMemoryTables covers @step-01 happy path: opening a
// project store after the 040/042 migrations creates the Fact Memory and
// Skills tables (facts, facts_fts, skills, skills_fts, skill_usage) with the
// 014 importance columns on facts, while prior observations stay untouched.
func TestMigration011FactMemoryTables(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "factproj")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer st.Close()

	// Prerequisite migrations from the acceptance precondition are applied.
	if countMigration(t, st.DB, "040_session_events.sql") != 1 {
		t.Fatalf("expected 040 applied before 011")
	}
	if countMigration(t, st.DB, "042_vec0_tables.sql") != 1 {
		t.Fatalf("expected 042 applied before 011")
	}
	if countMigration(t, st.DB, "011_facts_skills.sql") != 1 {
		t.Fatalf("expected 011_facts_skills.sql recorded once")
	}

	// Fact Memory + Skills tables exist (FTS5 virtual tables show up in
	// sqlite_master as tables too).
	for _, tbl := range []string{"facts", "facts_fts", "skills", "skills_fts", "skill_usage"} {
		if !tableExists(t, st.DB, tbl) {
			t.Fatalf("missing table %s after store open", tbl)
		}
	}

	// 014 importance columns are present on facts with NOT NULL defaults.
	for _, col := range []string{
		"importance_score", "recency_decay", "maturity_tier", "retrieval_usage",
		"deleted_at", "created_at", "updated_at", "content",
	} {
		if !columnExists(t, st.DB, "facts", col) {
			t.Errorf("facts missing column %s", col)
		}
	}
	var score float64
	var decay float64
	var tier string
	var usage int
	if err := st.DB.QueryRow(
		`SELECT importance_score, recency_decay, maturity_tier, retrieval_usage
		 FROM facts WHERE id = 1`).Scan(&score, &decay, &tier, &usage); err == nil {
		t.Errorf("unexpected pre-existing fact row id=1")
	}

	// skills carries the spec columns.
	for _, col := range []string{"name", "language", "description", "code_path",
		"deleted_at", "created_at", "updated_at"} {
		if !columnExists(t, st.DB, "skills", col) {
			t.Errorf("skills missing column %s", col)
		}
	}
	// skill_usage carries the spec columns.
	for _, col := range []string{"skill_id", "session_id", "timestamp", "stdout", "stderr"} {
		if !columnExists(t, st.DB, "skill_usage", col) {
			t.Errorf("skill_usage missing column %s", col)
		}
	}

	// Prior observations are unchanged: seed one, re-open, read it back.
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s011', 'factproj', '/tmp', ?, 'active')`, now); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observations (session_id, type, title, content, project, scope, created_at, updated_at)
		VALUES ('s011', 'discovery', 'pre-fact', 'body', 'factproj', 'project', ?, ?)`, now, now); err != nil {
		t.Fatalf("seed observation: %v", err)
	}

	st.Close()
	st2, err := Open(dir, "factproj")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	if countMigration(t, st2.DB, "011_facts_skills.sql") != 1 {
		t.Fatalf("011 applied more than once on re-open")
	}
	var title string
	if err := st2.DB.QueryRow(`SELECT title FROM observations WHERE title='pre-fact'`).Scan(&title); err != nil {
		t.Fatalf("prior observation unreadable: %v", err)
	}
	if title != "pre-fact" {
		t.Errorf("prior observation rewritten: %q", title)
	}
	var fts int
	if err := st2.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE name='facts_fts'`).Scan(&fts); err != nil {
		t.Fatalf("count facts_fts: %v", err)
	}
	if fts != 1 {
		t.Errorf("facts_fts count after re-open = %d, want 1", fts)
	}
}

// TestMigration011ReopenIdempotent covers @step-01 edge: a second open of a
// store that already applied 011 creates no duplicate tables.
func TestMigration011ReopenIdempotent(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "factidem")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	st.Close()
	st2, err := Open(dir, "factidem")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()

	var n int
	if err := st2.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE name='facts'`).Scan(&n); err != nil {
		t.Fatalf("count facts: %v", err)
	}
	if n != 1 {
		t.Errorf("duplicate facts tables after re-open: got %d, want 1", n)
	}
	if err := st2.DB.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE name='skill_usage'`).Scan(&n); err != nil {
		t.Fatalf("count skill_usage: %v", err)
	}
	if n != 1 {
		t.Errorf("duplicate skill_usage tables after re-open: got %d, want 1", n)
	}
}
