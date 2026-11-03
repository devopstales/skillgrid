package store

import (
	"database/sql"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// clm-renumber: the CLM change landed its three migrations at 050/051/052.
// CTX (ADR-0033) places its ctx_* tables on the SAME shared store starting at
// 056, so the CLM files are shifted up by 3: 050→053, 051→054, 052→055. The
// rename is one-way (a wrong rename re-applies the wrong DDL), so this test
// pins the resulting numbers, the application order, and that re-applying the
// set leaves existing CLM rows intact.

// TestMigrationCLMRenumberNaming asserts each CLM table is created by exactly
// one migration, that those migrations carry the renumbered 053/054/055
// filenames, and that no file in the embedded set still carries the pre-CTX
// CLM numbers 051/052 (those were CLM-only and are now free for ctx_* to start
// at 056+). The 050 number is shared with the non-CLM scan_findings migration,
// so the absence of 051/052 is the meaningful CLM assertion.
func TestMigrationCLMRenumberNaming(t *testing.T) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read migrations dir: %v", err)
	}
	wantTable := map[string]string{
		"053_tool_outputs.sql":      "tool_outputs",
		"054_indexed_files.sql":     "indexed_files",
		"055_context_revisions.sql": "context_revisions",
	}
	for name, table := range wantTable {
		b, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if !containsCreateTable(string(b), table) {
			t.Errorf("migration %s does not create table %s", name, table)
		}
	}

	// No file may carry the pre-CTX CLM numbers 051/052.
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	for _, n := range names {
		if len(n) >= 3 && (n[:3] == "051" || n[:3] == "052") {
			t.Errorf("pre-CTX CLM number still present: %s (expected shifted to 053/054/055)", n)
		}
	}
}

// TestMigrationCLMRenumberOrderAndRowsIntact applies the set on a fresh store,
// verifies the three CLM migrations are recorded once each in ascending
// (053 → 054 → 055) order, seeds one row per CLM table, and re-opens to prove
// the migrations skip on re-apply (row counts unchanged, each recorded once).
func TestMigrationCLMRenumberOrderAndRowsIntact(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(dir, "clmrenum")
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	clm := []string{"053_tool_outputs.sql", "054_indexed_files.sql", "055_context_revisions.sql"}
	for _, name := range clm {
		if countMigration(t, st.DB, name) != 1 {
			t.Fatalf("expected %s recorded once, got %d", name, countMigration(t, st.DB, name))
		}
	}
	if err := assertCLMOrder(t, st.DB); err != nil {
		t.Fatalf("application order: %v", err)
	}

	// Seed one row per CLM table to measure row-count integrity.
	if _, err := st.DB.Exec(`INSERT INTO tool_outputs (session_id, project_id, tool_name, output, size_bytes, created_at)
		VALUES ('s1','p1','grep','hello',5,datetime('now'))`); err != nil {
		t.Fatalf("seed tool_outputs: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO indexed_files (project, name, kind, language, content)
		VALUES ('p1','a.go','code','go','func main(){}')`); err != nil {
		t.Fatalf("seed indexed_files: %v", err)
	}
	if _, err := st.DB.Exec(`INSERT INTO context_revisions (session_id, project_id, revision, anchor_count, anchor_digest, messages, size_estimate, calibration_factor, created_at)
		VALUES ('s1','p1',1,10,'deadbeef','[]',100,4.0,datetime('now'))`); err != nil {
		t.Fatalf("seed context_revisions: %v", err)
	}
	pre := clmRowCounts(t, st.DB)

	// Re-open: the runner re-walks the set but must skip every applied
	// migration, so CLM rows survive untouched and each stays recorded once.
	st.Close()
	st2, err := Open(dir, "clmrenum")
	if err != nil {
		t.Fatalf("re-open: %v", err)
	}
	defer st2.Close()
	for _, name := range clm {
		if countMigration(t, st2.DB, name) != 1 {
			t.Errorf("%s re-applied on re-open (count %d)", name, countMigration(t, st2.DB, name))
		}
	}
	post := clmRowCounts(t, st2.DB)
	for table, want := range pre {
		if got := post[table]; got != want {
			t.Errorf("row count for %s changed after re-apply: before=%d after=%d", table, want, got)
		}
	}
}

// assertCLMOrder fails if the three CLM migration keys are not present in
// index_meta in ascending filename order (the order the runner applies them).
func assertCLMOrder(t *testing.T, db *sql.DB) error {
	t.Helper()
	rows, err := db.Query(
		`SELECT key FROM index_meta
		 WHERE key IN ('migration:053_tool_outputs.sql','migration:054_indexed_files.sql','migration:055_context_revisions.sql')
		 ORDER BY key`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return err
		}
		got = append(got, k)
	}
	if len(got) != 3 {
		return errExpectedThreeCLM(got)
	}
	for i := 1; i < len(got); i++ {
		if got[i-1] >= got[i] {
			return errCLMOrder(got)
		}
	}
	return nil
}

func errExpectedThreeCLM(got []string) error {
	return &clmOrderError{msg: "expected 3 CLM migration rows, got " + strconv.Itoa(len(got))}
}
func errCLMOrder(got []string) error {
	return &clmOrderError{msg: "CLM migrations not applied in ascending order: " + strings.Join(got, " < ")}
}

type clmOrderError struct{ msg string }

func (e *clmOrderError) Error() string { return e.msg }

func clmRowCounts(t *testing.T, db *sql.DB) map[string]int {
	t.Helper()
	out := map[string]int{}
	for _, table := range []string{"tool_outputs", "indexed_files", "context_revisions"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		out[table] = n
	}
	return out
}

// containsCreateTable reports whether sqlText has a CREATE [VIRTUAL] TABLE
// statement whose table name is exactly table (matching on the leading
// statement, not a substring, so e.g. "tool_outputs" does not match the
// "tool_outputs_fts" vtab).
func containsCreateTable(sqlText, table string) bool {
	upper := strings.ToUpper(sqlText)
	open := strings.ToUpper(table) + " ("
	for _, prefix := range []string{"CREATE TABLE IF NOT EXISTS ", "CREATE TABLE ", "CREATE VIRTUAL TABLE IF NOT EXISTS ", "CREATE VIRTUAL TABLE "} {
		if strings.Contains(upper, prefix+open) {
			return true
		}
	}
	return false
}
