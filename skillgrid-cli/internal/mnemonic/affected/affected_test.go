package affected

import (
	"context"
	"database/sql"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// fixtureGraph seeds a store with the 005 symbols/edges schema and a small
// dependency chain: base <- mid <- top (each imports/calls the previous),
// base <- base_test (tests_for), and a dangling (no dependents) leaf. All
// dependents live in test files (or are the dangling leaf) so the affected
// set is exactly the test files.
func fixtureGraph(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(t.TempDir(), "affected-probe")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := context.Background()
	db := st.DB
	seed := `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at) VALUES
			('src/base.go', 1, 1, 'a', 'now'),
			('src/mid.go', 2, 2, 'b', 'now'),
			('src/top.go', 3, 3, 'c', 'now'),
			('src/base_test.go', 4, 4, 'd', 'now'),
			('src/dangle.go', 5, 5, 'e', 'now'),
			('src/top_test.go', 6, 6, 'f', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'base', 'base', 'function', 'go', 'func base()', 1, 5, 'h', 'uid-base' FROM files f WHERE f.path = 'src/base.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'mid', 'mid', 'function', 'go', 'func mid()', 1, 5, 'h', 'uid-mid' FROM files f WHERE f.path = 'src/mid.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'top', 'top', 'function', 'go', 'func top()', 1, 5, 'h', 'uid-top' FROM files f WHERE f.path = 'src/top.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'TestBase', 'TestBase', 'function', 'go', 'func TestBase()', 1, 5, 'h', 'uid-test-base' FROM files f WHERE f.path = 'src/base_test.go';
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'TestTop', 'TestTop', 'function', 'go', 'func TestTop()', 1, 5, 'h', 'uid-test-top' FROM files f WHERE f.path = 'src/top_test.go';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-mid'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-mid'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 10;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-top'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-top'),
			(SELECT id FROM symbols WHERE uid = 'uid-mid'), 'mid', 'EXTRACTED', 10;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'tests_for',
			(SELECT id FROM symbols WHERE uid = 'uid-test-base'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-test-base'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 20;
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'imports',
			(SELECT id FROM symbols WHERE uid = 'uid-test-top'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-test-top'),
			(SELECT id FROM symbols WHERE uid = 'uid-top'), 'top', 'EXTRACTED', 10;
	`
	if _, err := db.ExecContext(ctx, seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return st
}

// TestCodeAffected covers @step-02 (Scenario: code_affected returns affected
// test files): changed sources surface the test files that (transitively)
// import them via import + tests_for edges; --depth caps traversal (default
// 5); --filter restricts reported test files; the result is paths and
// relationships from the existing graph only (no invented edges).
func TestCodeAffected(t *testing.T) {
	st := fixtureGraph(t)
	ctx := context.Background()

	// Changed: base only. Transitively affected test file: src/base_test.go
	// (tests_for base).
	res, err := Affected(ctx, st.DB, Options{Changed: []string{"src/base.go"}})
	if err != nil {
		t.Fatalf("affected: %v", err)
	}
	if res.Message != "" {
		t.Fatalf("unexpected message: %s", res.Message)
	}
	if got := res.TestFiles; len(got) == 0 || got[0] != "src/base_test.go" {
		t.Fatalf("affected tests for base must start with src/base_test.go, got %v", got)
	}
	// The whole chain is affected: changing base also reaches top's test
	// (top imports mid which imports base) within the default depth 5.
	if !contains(res.TestFiles, "src/top_test.go") {
		t.Errorf("transitively affected test src/top_test.go missing, got %v", res.TestFiles)
	}

	// --depth 1 caps traversal: mid is 2 hops from base, so its test file is
	// not reached (only base's direct test).
	res3, err := Affected(ctx, st.DB, Options{Changed: []string{"src/base.go"}, Depth: 1})
	if err != nil {
		t.Fatalf("affected depth1: %v", err)
	}
	if got := res3.TestFiles; len(got) != 1 || got[0] != "src/base_test.go" {
		t.Fatalf("depth-1 affected tests = %v, want [src/base_test.go]", got)
	}

	// --filter restricts reported test files.
	res4, err := Affected(ctx, st.DB, Options{Changed: []string{"src/base.go", "src/top.go"}, Filter: "top_test"})
	if err != nil {
		t.Fatalf("affected filter: %v", err)
	}
	if got := res4.TestFiles; len(got) != 1 || got[0] != "src/top_test.go" {
		t.Fatalf("filtered affected tests = %v, want [src/top_test.go]", got)
	}
}

// TestCodeAffectedEmpty covers @step-02 (Scenario: code_affected with no
// changed files returns empty): an empty changed set is an empty result with
// a clear message, not an error.
func TestCodeAffectedEmpty(t *testing.T) {
	st := fixtureGraph(t)
	res, err := Affected(context.Background(), st.DB, Options{})
	if err != nil {
		t.Fatalf("affected(empty) must not error, got %v", err)
	}
	if len(res.TestFiles) != 0 {
		t.Errorf("empty changed set must report no test files, got %v", res.TestFiles)
	}
	if res.Message == "" {
		t.Errorf("empty changed set must carry a clear message, got empty")
	}
}

// TestBlastRadius covers @step-02 (Scenario: Dropped references are excluded
// from blast radius): a reference dropped at extraction (step-01 drop
// policy, no stored edge) contributes nothing — only the resolved edges are
// traversed, so a false positive can never inflate the reported radius.
func TestBlastRadius(t *testing.T) {
	st := fixtureGraph(t)
	ctx := context.Background()
	db := st.DB

	// A route node with a RESOLVED references edge to base (step-01 style):
	// it IS traversed and surfaces in the blast radius.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('urls.py', 7, 7, 'u', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'GET /base/', 'GET /base/', 'route', 'python', 'path("/base/")', 1, 5, 'h', 'uid-route-resolved'
		FROM files f WHERE f.path = 'urls.py';
		INSERT INTO edges (kind, from_id, file_id, to_id, to_name, confidence, line)
		SELECT 'references',
			(SELECT id FROM symbols WHERE uid = 'uid-route-resolved'),
			(SELECT file_id FROM symbols WHERE uid = 'uid-route-resolved'),
			(SELECT id FROM symbols WHERE uid = 'uid-base'), 'base', 'EXTRACTED', 30;
	`); err != nil {
		t.Fatalf("seed resolved route: %v", err)
	}
	// A route node whose handler reference was DROPPED at extraction (no
	// stored edge — drop-not-guess): it must NOT inflate the radius.
	if _, err := db.ExecContext(ctx, `
		INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
		VALUES ('urls2.py', 8, 8, 'u2', 'now');
		INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
		SELECT f.id, 'GET /mystery/', 'GET /mystery/', 'route', 'python', 'path("/mystery/")', 1, 5, 'h', 'uid-route-dropped'
		FROM files f WHERE f.path = 'urls2.py';
	`); err != nil {
		t.Fatalf("seed dropped route: %v", err)
	}

	res, err := Affected(ctx, st.DB, Options{Changed: []string{"src/base.go"}})
	if err != nil {
		t.Fatalf("affected: %v", err)
	}
	var resolvedSeen, droppedSeen bool
	for _, r := range res.Relationships {
		if r.Via == "uid-route-resolved" {
			resolvedSeen = true
			if r.Kind != "references" {
				t.Errorf("resolved-route hop kind = %q, want references", r.Kind)
			}
		}
		if r.Via == "uid-route-dropped" {
			droppedSeen = true
		}
	}
	if !resolvedSeen {
		t.Errorf("the resolved route->base references edge must be traversed, got %+v", res.Relationships)
	}
	if droppedSeen {
		t.Errorf("a dropped (unresolvable) reference must not inflate the blast radius, got %+v", res.Relationships)
	}
	if !contains(res.TestFiles, "src/base_test.go") {
		t.Errorf("resolved-edge blast radius must reach src/base_test.go, got %v", res.TestFiles)
	}
	// top_test is reachable via the resolved import chain, so it is expected;
	// the dropped route (urls2.py) must simply be absent from the hops.
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// TestCodeAffectedNoInventedEdges covers @step-02 (query only): a changed
// file with no graph edges yields no affected test files — the traversal
// never invents an edge.
func TestCodeAffectedNoInventedEdges(t *testing.T) {
	st := fixtureGraph(t)
	res, err := Affected(context.Background(), st.DB, Options{Changed: []string{"src/dangle.go"}})
	if err != nil {
		t.Fatalf("affected: %v", err)
	}
	if len(res.TestFiles) != 0 || len(res.Relationships) != 0 {
		t.Errorf("a dangling file has no dependents; got %+v", res)
	}
}

var _ = sql.ErrNoRows
