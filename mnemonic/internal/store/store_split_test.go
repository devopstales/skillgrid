package store

import (
	"strings"
	"testing"
)

// TestSplitSQLStatementsTriggerLiteralEnd guards the inTrig state against
// END/BEGIN words inside trigger-body string literals: a literal containing
// 'END' (or 'BEGIN') must not terminate the trigger body, or the file shreds
// into mis-ordered fragments that re-fail on every open (review SEC2).
func TestSplitSQLStatementsTriggerLiteralEnd(t *testing.T) {
	script := strings.Join([]string{
		"CREATE TABLE z (id INTEGER PRIMARY KEY, s TEXT);",
		"",
		"CREATE TRIGGER t AFTER UPDATE OF s ON z BEGIN",
		"  UPDATE z SET s = 'the END is near' WHERE id = OLD.id;",
		"  DELETE FROM z WHERE s LIKE 'BEGIN%';",
		"END;",
		"",
		"CREATE TABLE after (id INTEGER);",
	}, "\n")

	stmts := splitSQLStatements(script)
	// Drop the empty trailing fragment (the file ends with a newline).
	n := len(stmts)
	if n > 0 && stmts[n-1] == "" {
		stmts = stmts[:n-1]
	}
	if len(stmts) != 3 {
		t.Fatalf("got %d statements, want 3: %q", len(stmts), stmts)
	}
	if !strings.HasPrefix(stmts[0], "CREATE TABLE z") {
		t.Errorf("stmt 0 = %q, want CREATE TABLE z", stmts[0])
	}
	if !strings.HasPrefix(stmts[1], "CREATE TRIGGER t") || !strings.HasSuffix(stmts[1], "END;") {
		t.Errorf("stmt 1 = %q, want the whole CREATE TRIGGER ... END;", stmts[1])
	}
	if !strings.Contains(stmts[1], "'the END is near'") {
		t.Errorf("stmt 1 lost the literal: %q", stmts[1])
	}
	if !strings.Contains(stmts[1], "LIKE 'BEGIN%'") {
		t.Errorf("stmt 1 lost the BEGIN literal: %q", stmts[1])
	}
	if !strings.HasPrefix(stmts[2], "CREATE TABLE after") {
		t.Errorf("stmt 2 = %q, want CREATE TABLE after", stmts[2])
	}
}
