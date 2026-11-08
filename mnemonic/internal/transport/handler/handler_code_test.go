package handler

// handler_code_test.go — the TICKET-04 code_search enrichment: per-hit
// blast radius + snippet. The fixtures seed the store directly (file,
// symbol, and edge rows — the FTS floor follows via the symbol_fts
// triggers) and write the working tree to a chdir'd root (the snippet's
// read path is CWD-relative, like the production MCP server's cwd).

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/mnemonic/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
)

// writeTestFile writes one working-tree file for the fixture.
func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// seedSearchGraph seeds the TICKET-04 enrichment fixture: add (math.go
// line 3), called from main (main.go) on mainLines and from TestAdd
// (math_test.go) on testLines — one calls edge row per line (the edges
// unique key includes the line, so two call sites = two rows for the
// same caller). It returns the handler + the add symbol's store row id
// (the resolved hit's StoreID).
func seedSearchGraph(t *testing.T, mainLines, testLines []int) (*CodeHandler, int64) {
	t.Helper()
	st := openTestStore(t, t.TempDir())
	c := config.Default()
	c.DataDir = t.TempDir()
	c.Project.Detect = false
	ch := NewCodeHandler(st, nil, c, "", fixedDetector{})

	root := t.TempDir()
	t.Chdir(root)
	// The symbols are seeded at the line numbers below (add:3, main:2,
	// TestAdd:2) — the working tree matches (the explore read path reads
	// disk at those lines).
	writeTestFile(t, root, "math.go", "package math\n\nfunc add(a, b int) int { return a + b }\n")
	writeTestFile(t, root, "main.go", "package math\nfunc main() { add(1, 2) }\n")
	writeTestFile(t, root, "math_test.go", "package math\nfunc TestAdd(t *testing.T) { add(1, 2) }\n")

	fMath, err := st.UpsertFile(ctx, store.FileRow{Path: "math.go", Language: "go", ContentHash: "h-math"})
	if err != nil {
		t.Fatalf("upsert math.go: %v", err)
	}
	fMain, err := st.UpsertFile(ctx, store.FileRow{Path: "main.go", Language: "go", ContentHash: "h-main"})
	if err != nil {
		t.Fatalf("upsert main.go: %v", err)
	}
	fTest, err := st.UpsertFile(ctx, store.FileRow{Path: "math_test.go", Language: "go", ContentHash: "h-test"})
	if err != nil {
		t.Fatalf("upsert math_test.go: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fMath, []store.Symbol{
		{Name: "add", QualifiedName: "math.add", Kind: "function", Language: "go", StartLine: 3, EndLine: 3, UID: "uid-add"},
	}); err != nil {
		t.Fatalf("upsert add: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fMain, []store.Symbol{
		{Name: "main", QualifiedName: "math.main", Kind: "function", Language: "go", StartLine: 2, EndLine: 2, UID: "uid-main"},
	}); err != nil {
		t.Fatalf("upsert main: %v", err)
	}
	if _, err := st.UpsertSymbols(ctx, fTest, []store.Symbol{
		{Name: "TestAdd", QualifiedName: "math.TestAdd", Kind: "function", Language: "go", StartLine: 2, EndLine: 2, UID: "uid-testadd"},
	}); err != nil {
		t.Fatalf("upsert TestAdd: %v", err)
	}
	symAdd, err := st.SymbolByName(ctx, "add")
	if err != nil || len(symAdd) != 1 {
		t.Fatalf("resolve add: %v (n=%d)", err, len(symAdd))
	}
	symMain, err := st.SymbolByName(ctx, "main")
	if err != nil || len(symMain) != 1 {
		t.Fatalf("resolve main: %v (n=%d)", err, len(symMain))
	}
	symTest, err := st.SymbolByName(ctx, "TestAdd")
	if err != nil || len(symTest) != 1 {
		t.Fatalf("resolve TestAdd: %v (n=%d)", err, len(symTest))
	}
	toAdd := symAdd[0].ID
	var mainEdges, testEdges []store.Edge
	for _, ln := range mainLines {
		mainEdges = append(mainEdges, store.Edge{Kind: "calls", FromID: symMain[0].ID, ToID: &toAdd, Confidence: "EXTRACTED", Line: ln})
	}
	for _, ln := range testLines {
		testEdges = append(testEdges, store.Edge{Kind: "calls", FromID: symTest[0].ID, ToID: &toAdd, Confidence: "EXTRACTED", Line: ln})
	}
	if len(mainEdges) > 0 {
		if err := st.ReplaceEdgesForFile(ctx, fMain, mainEdges); err != nil {
			t.Fatalf("edges main.go: %v", err)
		}
	}
	if len(testEdges) > 0 {
		if err := st.ReplaceEdgesForFile(ctx, fTest, testEdges); err != nil {
			t.Fatalf("edges math_test.go: %v", err)
		}
	}
	return ch, toAdd
}

// findAddHit returns the resolved hit for the add symbol from a search
// result (nil when absent).
func findAddHit(t *testing.T, res SearchAnswer, addID int64) *SearchHit {
	t.Helper()
	for i := range res.Hits {
		if res.Hits[i].StoreID == addID {
			return &res.Hits[i]
		}
	}
	return nil
}

// TestSearchHitBlastRadiusShape pins the enriched hit: 2 distinct
// callers (main + TestAdd), the test file among the callers, and the
// span-start snippet.
func TestSearchHitBlastRadiusShape(t *testing.T) {
	ch, addID := seedSearchGraph(t, []int{2}, []int{2})
	res, err := ch.Search(ctx, "add", search.SearchQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	hit := findAddHit(t, res, addID)
	if hit == nil {
		t.Fatalf("no resolved hit for add in %+v", res.Hits)
	}
	if hit.Callers != 2 {
		t.Errorf("Callers = %d, want 2", hit.Callers)
	}
	if len(hit.TestFiles) != 1 || hit.TestFiles[0] != "math_test.go" {
		t.Errorf("TestFiles = %v, want [math_test.go]", hit.TestFiles)
	}
	const wantSnippet = "func add(a, b int) int { return a + b }"
	if hit.Snippet != wantSnippet {
		t.Errorf("Snippet = %q, want %q", hit.Snippet, wantSnippet)
	}
}

// TestSearchUnresolvedHitDegrades pins "unresolved hit: fields omitted,
// no error": the structural leg's hits carry no store row (StoreID 0) —
// the enrichment stays absent and the search still answers.
func TestSearchUnresolvedHitDegrades(t *testing.T) {
	ch, _ := seedSearchGraph(t, nil, nil)
	res, err := ch.Search(ctx, `(function_declaration) \fn`, search.SearchQuery{Language: "go", Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	found := false
	for i := range res.Hits {
		h := &res.Hits[i]
		if h.StoreID != 0 {
			continue
		}
		found = true
		if h.Callers != 0 || len(h.TestFiles) != 0 || h.Snippet != "" {
			t.Errorf("unresolved hit enriched: %+v", h)
		}
	}
	if !found {
		t.Fatalf("no unresolved (StoreID 0) hit in %+v", res.Hits)
	}
}

// TestSearchDuplicateEdgesOnce pins "doubly-recorded caller edges count
// once": main has two call-site edges to add (two lines) — the caller
// count is distinct caller symbols (main + TestAdd = 2, not 3).
func TestSearchDuplicateEdgesOnce(t *testing.T) {
	ch, addID := seedSearchGraph(t, []int{2, 3}, []int{2})
	res, err := ch.Search(ctx, "add", search.SearchQuery{Limit: 10})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	hit := findAddHit(t, res, addID)
	if hit == nil {
		t.Fatalf("no resolved hit for add in %+v", res.Hits)
	}
	if hit.Callers != 2 {
		t.Errorf("Callers = %d, want 2 (the caller deduped across call sites)", hit.Callers)
	}
	if len(hit.TestFiles) != 1 || hit.TestFiles[0] != "math_test.go" {
		t.Errorf("TestFiles = %v, want [math_test.go]", hit.TestFiles)
	}
}
