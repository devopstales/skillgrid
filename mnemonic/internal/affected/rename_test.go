package affected

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/store"
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

// TestCodeRenameImportEdges covers the regression where the rename plan
// builder missed name-only edges (to_id IS NULL). Import edges are stored
// with to_id NULL (the import target is a module path, not a symbol UID),
// so the old `WHERE e.to_id = ?` predicate never saw them. A file that
// imports the target package must appear as a graph edit.
func TestCodeRenameImportEdges(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()

	// A file that imports the base package: a name-only imports edge
	// (to_id NULL, to_name 'base') — the exact shape the indexer writes
	// for Python/Go imports whose target is not a resolvable symbol UID.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/importer.py', 4, 4, 'i', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'importer', 'importer', 'function', 'python', 'def importer()', 1, 5, 'h', 'uid-importer'
		FROM files f WHERE f.path = 'src/importer.py';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-importer'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-importer'),
			NULL, 'base', 'lib/base.py', 'EXTRACTED', 1;
	`); err != nil {
		t.Fatalf("seed import edge: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if plan.Ambiguous {
		t.Fatalf("uid narrow must disambiguate, got %+v", plan)
	}
	var importerFound bool
	for _, e := range plan.GraphEdits {
		if e.Path == "src/importer.py" {
			importerFound = true
		}
	}
	if !importerFound {
		t.Errorf("the importing file (src/importer.py, name-only imports edge) must be a graph edit, got %+v", plan.GraphEdits)
	}
}

// TestCodeRenameOccurrenceCount covers the regression where the occurrence
// counter on graph edits counted only symbols (always 0 for call sites).
// A file with a calls edge to the target must report a non-zero count.
func TestCodeRenameOccurrenceCount(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	// The caller (src/mid.go) has a calls edge to base. The occurrence
	// counter must count edge endpoints, not just symbols.
	var midEdit *Edit
	for i := range plan.GraphEdits {
		if plan.GraphEdits[i].Path == "src/mid.go" {
			midEdit = &plan.GraphEdits[i]
			break
		}
	}
	if midEdit == nil {
		t.Fatalf("expected src/mid.go in graph edits, got %+v", plan.GraphEdits)
	}
	// The count must be >= 1 (the calls edge to_name='base' in src/mid.go).
	if !strings.Contains(midEdit.Note, "occurrences:") {
		t.Fatalf("expected occurrences note, got %q", midEdit.Note)
	}
	var count int
	fmt.Sscanf(midEdit.Note, "occurrences: %d", &count)
	if count < 1 {
		t.Errorf("src/mid.go occurrence count must be >= 1 (has a calls edge to base), got %d", count)
	}
}

// TestCodeRenameEdgeTextSearch covers the regression where the text-search
// bucket only looked at the symbols table, missing call-site edges. A file
// that has an edge with to_name='base' but to_id NULL (a name-only reference
// that does NOT match the graph bucket predicate) must appear in the
// text-search bucket.
func TestCodeRenameEdgeTextSearch(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()

	// A file with a name-only edge (to_id NULL, to_name 'base'). The graph
	// bucket predicate (to_id = target.ID OR (to_id IS NULL AND to_name =
	// target.Name)) WOULD match this, so it lands in the graph bucket. To
	// test the text-search bucket's edge query, we need a file with an edge
	// to_name='base' that is NOT matched by the graph bucket. Use a different
	// edge kind not in renameRefKinds ('defines' is not a reference kind).
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/ref.go', 6, 6, 'r', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'helper', 'helper', 'function', 'go', 'func helper()', 1, 5, 'h', 'uid-helper'
		FROM files f WHERE f.path = 'src/ref.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'defines',
			(SELECT id FROM symbols WHERE uid = 'uid-helper'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-helper'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 5;
	`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	// 'defines' is not in renameRefKinds, so src/ref.go is NOT in the graph
	// bucket. But the text-search bucket's edge query (to_name = 'base')
	// DOES match it. It must appear in the text-search bucket.
	var refInTextSearch bool
	for _, e := range plan.TextSearchEdits {
		if e.Path == "src/ref.go" {
			refInTextSearch = true
		}
	}
	if !refInTextSearch {
		t.Errorf("src/ref.go (edge to_name='base', kind 'defines' not in refKinds) must be a text-search edit, got %+v", plan.TextSearchEdits)
	}
}

// TestCodeRenameNoDoubleList covers the round-3 regression where a file with
// two edges to the same target (an import + a call) was listed TWICE as two
// duplicate graph edits. After the fix it appears once, with a summed
// occurrence count.
func TestCodeRenameNoDoubleList(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	// workload.go: both an imports edge (name-only, to_id NULL) and a calls
	// edge (resolved, to_id = base) to the target — two edges, one file.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/workload.go', 5, 5, 'w', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'workload', 'workload', 'function', 'go', 'func workload()', 1, 5, 'h', 'uid-workload'
		FROM files f WHERE f.path = 'src/workload.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-workload'),
			NULL, 'base', 'lib/base', 'EXTRACTED', 9;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 574;
	`); err != nil {
		t.Fatalf("seed workload edges: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	var workloadCount int
	var workloadNote string
	for _, e := range plan.GraphEdits {
		if e.Path == "src/workload.go" {
			workloadCount++
			workloadNote = e.Note
		}
	}
	if workloadCount != 1 {
		t.Errorf("src/workload.go must appear exactly once in graph edits, got %d (plan %+v)", workloadCount, plan.GraphEdits)
	}
	// Two stored edges (import + call) → occurrence count 2, not 1 or a
	// duplicate listing.
	if workloadNote != "occurrences: 2" {
		t.Errorf("workload.go occurrences must be 2 (import + call edge), got %q", workloadNote)
	}
}

// TestCodeRenameReferencesEdge covers the round-3 regression where a
// route→handler 'references' edge (the Flask call site at blueprint/
// workload/workload.py:366) was invisible to rename because 'references'
// (plural) was missing from renameRefKinds, even though code_impact surfaced
// it. A file whose only link to the target is a 'references' edge must be a
// graph edit.
func TestCodeRenameReferencesEdge(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	// blueprint/workload/workload.py: a route handler that calls the factory.
	// The indexer emits a 'references' edge (route→handler) for this call site.
	// This file has NO calls/imports edge to base — only the 'references' edge.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('blueprint/workload/workload.py', 7, 7, 'bp', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'get_workload', 'get_workload', 'route', 'python', 'def get_workload()', 360, 370, 'h', 'uid-getworkload'
		FROM files f WHERE f.path = 'blueprint/workload/workload.py';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'references',
			(SELECT id FROM symbols WHERE uid = 'uid-getworkload'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-getworkload'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 366;
	`); err != nil {
		t.Fatalf("seed references edge: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	var found bool
	for _, e := range plan.GraphEdits {
		if e.Path == "blueprint/workload/workload.py" {
			found = true
		}
	}
	if !found {
		t.Errorf("the route→handler references edge file (blueprint/workload/workload.py) must be a graph edit, got %+v", plan.GraphEdits)
	}
}

// TestCodeRenameTransitiveCaller covers the round-4 regression where the plan
// was scoped to depth-1 dependents only, so a transitive caller (a function
// that itself calls the target) was invisible — code_impact surfaced it at
// depth 2 but code_rename did not. The BFS over dependent edges must reach
// src/top.go (which calls src/mid.go, which calls base) and list it as a
// graph edit.
func TestCodeRenameTransitiveCaller(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	// src/top.go: a function that calls mid (which calls base). This makes
	// top a depth-2 dependent of base.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('src/top.go', 8, 8, 'top', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'top', 'top', 'function', 'go', 'func top()', 1, 5, 'h', 'uid-top'
		FROM files f WHERE f.path = 'src/top.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid = 'uid-top'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-top'),
			(SELECT id FROM symbols WHERE uid = 'uid-mid'), 'mid', 'EXTRACTED', 30;
	`); err != nil {
		t.Fatalf("seed top caller: %v", err)
	}

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	var found bool
	for _, e := range plan.GraphEdits {
		if e.Path == "src/top.go" {
			found = true
		}
	}
	if !found {
		t.Errorf("the transitive caller (src/top.go, depth 2) must be a graph edit, got %+v", plan.GraphEdits)
	}
}

// TestCodeRenameRawOccurrenceCount covers the round-4 undercount: the
// occurrence note must count raw text matches of the old name in the file, so
// the import line (a name-only edge to a module path) is not dropped. A real
// file whose content references the target on the import line AND two call
// sites must report occurrences: 3, not the 2 the graph edge count would give.
func TestCodeRenameRawOccurrenceCount(t *testing.T) {
	st := renameFixture(t)
	ctx := context.Background()
	dir := t.TempDir()
	p := dir + "/workload.go"
	// 3 raw occurrences: line 9 import + call sites at 574 and 617.
	content := "import base\n" +
		strings.Repeat("x", 50) + "\n" +
		"base()\n" + // call site
		"base()\n" + // call site
		""
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// A workload.go that imports base (name-only edge) and calls it (calls
	// edge): the graph sees 2 reference edges.
	if _, err := st.DB.Exec(`
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('workload.go', 9, 9, 'wl', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'workload', 'workload', 'function', 'go', 'func workload()', 1, 5, 'h', 'uid-workload'
		FROM files f WHERE f.path = 'workload.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-workload'),
			NULL, 'base', 'lib/base', 'EXTRACTED', 9;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'calls',
			(SELECT id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-workload'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 574;
	`); err != nil {
		t.Fatalf("seed workload edges: %v", err)
	}

	// rawOccurrences resolves the indexed path against CWD, so run from the
	// fixture dir to make workload.go resolvable on disk.
	oldwd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	plan, err := Rename(ctx, st.DB, RenameOptions{Old: "base", New: "verifyBase", UID: "uid-base"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	var note string
	for _, e := range plan.GraphEdits {
		if e.Path == "workload.go" {
			note = e.Note
		}
	}
	var count int
	if n, err := fmt.Sscanf(note, "occurrences: %d", &count); n != 1 || err != nil {
		t.Fatalf("workload.go must have an occurrences note, got %q", note)
	}
	// Raw text matches (import + 2 calls) = 3, not the graph edge count 2.
	if count != 3 {
		t.Errorf("workload.go occurrences must be 3 (import + 2 call sites, rg-style), got %d", count)
	}
}
