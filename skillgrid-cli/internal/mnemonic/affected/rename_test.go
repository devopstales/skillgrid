package affected

import (
	"context"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// renameFixture seeds a store where base (src/base.go) is called by mid
// (src/mid.go, a calls edge), so the graph bucket has a definition + a typed
// reference.
func renameFixture(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "rename-probe")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('src/base.go', 1, 1, 'a', 'now'),
			('src/mid.go', 2, 2, 'b', 'now'),
			('src/base_test.go', 3, 3, 'c', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'function', 'go', 'func base()', 1, 5, 'h', 'uid-base' FROM files f WHERE f.path = 'src/base.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'mid', 'mid', 'function', 'go', 'func mid()', 1, 5, 'h', 'uid-mid' FROM files f WHERE f.path = 'src/mid.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'TestBase', 'TestBase', 'function', 'go', 'func TestBase()', 1, 5, 'h', 'uid-test' FROM files f WHERE f.path = 'src/base_test.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid = 'uid-mid'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-mid'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 10;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'tests_for',
			(SELECT id FROM symbols WHERE uid = 'uid-test'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-test'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 20;
	`
	if _, err := st.DB.ExecContext(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return st
}

// TestCodeRename covers @step-02 (Scenario: code_rename returns graph and
// text-search edit buckets): renaming a symbol splits the plan into graph
// edits (high-confidence: definition + typed references from the edges
// table) and text-search edits (lower-confidence: string matches, flagged
// "review carefully"); every edit carries a confidence label; --dry-run
// (default) returns the plan without writing any file.
func TestCodeRename(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if plan.Ambiguous {
		t.Fatalf("single-symbol target must not be ambiguous, got %+v", plan)
	}
	if len(plan.GraphEdits) == 0 {
		t.Fatalf("expected graph edits (definition + typed refs), got none")
	}
	if plan.TotalEdits == 0 {
		t.Fatalf("expected a non-zero edit count, got 0")
	}
	if plan.DryRun != true {
		t.Errorf("dry_run must default true, got %v", plan.DryRun)
	}
	// Every edit carries a confidence label.
	for _, e := range append(plan.GraphEdits, plan.TextSearchEdits...) {
		if e.Confidence == "" {
			t.Errorf("every edit must carry a confidence label, got %+v", e)
		}
	}
	// The graph bucket is high-confidence (the edges table is typed).
	for _, e := range plan.GraphEdits {
		if e.Confidence != "high" {
			t.Errorf("graph edits are high-confidence, got %q", e.Confidence)
		}
	}
	// The definition site is in the graph bucket (base's own file).
	var defFound bool
	for _, e := range plan.GraphEdits {
		if e.Path == "src/base.go" {
			defFound = true
		}
	}
	if !defFound {
		t.Errorf("the definition site (src/base.go) must be a graph edit, got %+v", plan.GraphEdits)
	}
	// The typed caller (src/mid.go, via the calls edge) is a graph edit.
	var callerFound bool
	for _, e := range plan.GraphEdits {
		if e.Path == "src/mid.go" {
			callerFound = true
		}
	}
	if !callerFound {
		t.Errorf("the typed caller (src/mid.go) must be a graph edit, got %+v", plan.GraphEdits)
	}
	// dry_run touches nothing: the plan lists files, but no write happened.
	// (The apply path is covered by CodeRenameApply.)
	if plan.FilesAffected == 0 {
		t.Errorf("expected affected files in the plan, got 0")
	}
}

// TestCodeRenameTextSearchBucket covers @step-02 (text-search bucket): a file
// that mentions the old name without a typed reference (e.g. a comment)
// lands in the lower-confidence text-search bucket, flagged "review
// carefully" — distinct from the high-confidence graph bucket.
func TestCodeRenameTextSearchBucket(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	// A file that names `base` (a symbol with the old name) but has NO typed
	// edge to the target: it is a string match, not a typed reference.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/note.go', 10, 10, 'n', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'function', 'go', 'func base()', 1, 5, 'h', 'uid-note-base'
		FROM files f WHERE f.path = 'src/note.go';
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	// `base` is now ambiguous (three symbols named base) — disambiguate to the
	// fixture target by uid.
	if !plan.Ambiguous {
		t.Fatalf("expected ambiguous (3 base symbols), got %+v", plan)
	}
	plan2, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if plan2.Ambiguous {
		t.Fatalf("uid narrow must disambiguate, got %+v", plan2)
	}
	var noteInTextSearch bool
	for _, e := range plan2.TextSearchEdits {
		if e.Path == "src/note.go" {
			noteInTextSearch = true
			if e.Confidence != "low" || !strings.Contains(e.Note, "review carefully") {
				t.Errorf("text-search edits must be low-confidence + flagged, got %+v", e)
			}
		}
	}
	if !noteInTextSearch {
		t.Errorf("src/note.go (string match, no typed edge) must be a text-search edit, got %+v", plan2.TextSearchEdits)
	}
}

// TestCodeRenameAmbiguous covers @step-02 (Scenario: Ambiguous rename target
// returns candidates): a name matching several symbols returns a ranked
// candidate list (most-connected first) — never a silent pick, no edits.
func TestCodeRenameAmbiguous(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	// Add a second symbol named base in a different file.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/other.go', 9, 9, 'o', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'function', 'go', 'func base()', 1, 5, 'h', 'uid-base-2'
		FROM files f WHERE f.path = 'src/other.go';
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !plan.Ambiguous {
		t.Fatalf("a multi-symbol target must be ambiguous, got %+v", plan)
	}
	if len(plan.Candidates) != 2 {
		t.Fatalf("expected 2 ranked candidates, got %d", len(plan.Candidates))
	}
	// No silent pick: no edits are planned until disambiguated.
	if plan.TotalEdits != 0 {
		t.Errorf("an ambiguous rename must not plan edits, got %d", plan.TotalEdits)
	}
	// Ranked: the more-connected base (has a caller) surfaces first.
	if plan.Candidates[0].UID != "uid-base" {
		t.Errorf("most-connected candidate should rank first, got %s", plan.Candidates[0].UID)
	}
}
