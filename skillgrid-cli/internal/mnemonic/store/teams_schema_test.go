package store

import (
	"testing"
)

// TestTeamsSchemaMigrationAddsTablesSafely — @step-01 Scenario: Store open adds teams tables safely
func TestTeamsSchemaMigrationAddsTablesSafely(t *testing.T) {
	dataDir := t.TempDir()
	st, err := Open(dataDir, "teams-schema")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer st.Close()

	// Seed an observation so we can assert migrations do not rewrite memory.
	if _, err := st.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1','teams-schema','/tmp', ?, 'active')`, testNow); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := st.DB.Exec(`
		INSERT INTO observations
			(session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at)
		VALUES ('s1','decision','keep-me','body','teams-schema','project','h-keep',0,?,?)`, testNow, testNow); err != nil {
		t.Fatalf("seed observation: %v", err)
	}

	wantTables := []string{
		"teams",
		"team_members",
		"tasks",
		"messages",
		"task_results",
		"reviews",
	}
	for _, name := range wantTables {
		var n int
		if err := st.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_list WHERE name = ?`, name).Scan(&n); err != nil {
			t.Fatalf("pragma_table_list %q: %v", name, err)
		}
		if n == 0 {
			t.Errorf("expected table %q to exist after 009 migration", name)
		}
	}

	var obsCount int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE title = 'keep-me'`).Scan(&obsCount); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	if obsCount != 1 {
		t.Errorf("observations rewritten; want 1 keep-me row, got %d", obsCount)
	}

	var migRecorded int
	if err := st.DB.QueryRow(`SELECT COUNT(*) FROM index_meta WHERE key = 'migration:009_teams_schema.sql'`).Scan(&migRecorded); err != nil {
		t.Fatalf("index_meta: %v", err)
	}
	if migRecorded != 1 {
		t.Errorf("expected migration:009_teams_schema.sql recorded, got count %d", migRecorded)
	}
}

// TestTeamsSchemaMigrationIdempotentOnReopen — @step-01 Scenario: Migration remains idempotent on reopen
func TestTeamsSchemaMigrationIdempotentOnReopen(t *testing.T) {
	dataDir := t.TempDir()
	st1, err := Open(dataDir, "teams-idem")
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if _, err := st1.DB.Exec(`
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES ('s1','teams-idem','/tmp', ?, 'active')`, testNow); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if _, err := st1.DB.Exec(`
		INSERT INTO observations
			(session_id, type, title, content, project, scope, normalized_hash, revision_count, created_at, updated_at)
		VALUES ('s1','decision','keep-me','body','teams-idem','project','h-keep',0,?,?)`, testNow, testNow); err != nil {
		t.Fatalf("seed observation: %v", err)
	}
	st1.Close()

	st2, err := Open(dataDir, "teams-idem")
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer st2.Close()

	for _, name := range []string{"teams", "team_members", "tasks", "messages", "task_results", "reviews"} {
		var n int
		if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM pragma_table_list WHERE name = ?`, name).Scan(&n); err != nil {
			t.Fatalf("pragma_table_list %q: %v", name, err)
		}
		if n == 0 {
			t.Errorf("expected table %q after reopen", name)
		}
	}
	var obsCount int
	if err := st2.DB.QueryRow(`SELECT COUNT(*) FROM observations WHERE title = 'keep-me'`).Scan(&obsCount); err != nil {
		t.Fatalf("count observations: %v", err)
	}
	if obsCount != 1 {
		t.Errorf("observations changed on reopen; want 1, got %d", obsCount)
	}
}
