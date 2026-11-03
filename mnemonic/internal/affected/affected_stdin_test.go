package affected

import (
	"context"
	"strings"
	"testing"
)

// TestCodeAffectedStdin covers @step-02 (Scenario: code_affected consumes a
// diff file list from stdin): a git-diff --name-only-style list is parsed into
// the changed set and the same import/tests_for traversal runs; an empty diff
// (empty stdin) returns an empty result with a clear message, not an error.
func TestCodeAffectedStdin(t *testing.T) {
	st := fixtureGraph(t)
	ctx := context.Background()

	stdin := "src/base.go\nsrc/mid.go\n"
	res, err := AffectedFromStdin(ctx, st.DB, stdin, Options{})
	if err != nil {
		t.Fatalf("affected(stdin): %v", err)
	}
	if res.Message != "" {
		t.Fatalf("unexpected message: %s", res.Message)
	}
	if !contains(res.TestFiles, "src/base_test.go") {
		t.Fatalf("stdin changed src/base.go must surface src/base_test.go, got %v", res.TestFiles)
	}
	if !contains(res.Changed, "src/base.go") || !contains(res.Changed, "src/mid.go") {
		t.Errorf("stdin file list must be the reported changed set, got %v", res.Changed)
	}

	// Blank lines and CRLF are tolerated (git diff output hygiene).
	stdin2 := "\r\nsrc/top_test.go\r\n\n"
	res2, err := AffectedFromStdin(ctx, st.DB, stdin2, Options{})
	if err != nil {
		t.Fatalf("affected(stdin2): %v", err)
	}
	// top_test.go is itself a test file with no dependents → no affected tests.
	if len(res2.TestFiles) != 0 {
		t.Errorf("a changed test file with no dependents must report no affected tests, got %v", res2.TestFiles)
	}

	// Empty stdin (empty diff): empty result, clear message, no error.
	res3, err := AffectedFromStdin(ctx, st.DB, "", Options{})
	if err != nil {
		t.Fatalf("empty stdin must not error, got %v", err)
	}
	if len(res3.TestFiles) != 0 {
		t.Errorf("empty diff must report no test files, got %v", res3.TestFiles)
	}
	if res3.Message == "" || !strings.Contains(res3.Message, "no changed") {
		t.Errorf("empty diff must carry a clear message, got %q", res3.Message)
	}
}
