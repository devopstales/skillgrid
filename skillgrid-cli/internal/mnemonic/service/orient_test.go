package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
)

// indexOrientFixture writes the fixture, indexes it through the resolved
// project (so the store path matches what the service opens), and returns the
// root and project ID.
func indexOrientFixture(t *testing.T) (string, string, *Service) {
	t.Helper()
	dataDir := t.TempDir()
	svc := New(dataDir)
	root := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write("main.go", "package main\n\n// WHY: a map keeps lookups O(1) for the hot path.\nfunc parseConfig() int {\n\treturn 1\n}\n\n// ADR-3 chose a value receiver here.\nfunc loadUserSettings() int {\n\treturn 2\n}\n")
	projectID, err := svc.ResolveProject(root)
	if err != nil {
		t.Fatalf("resolve project: %v", err)
	}
	if _, err := svc.RunCodeIndex(context.Background(), root); err != nil {
		t.Fatalf("index: %v", err)
	}
	return root, projectID, svc
}

// TestSymbolSearchFindsIndexedSymbols covers @step-02 happy: identifier FTS
// finds camelCase/snake_case symbols that were indexed.
func TestSymbolSearchFindsIndexedSymbols(t *testing.T) {
	_, projectID, svc := indexOrientFixture(t)
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer cleanup()
	hits, err := search.SymbolFTS(h.Store().DB, "parseConfig", 10)
	if err != nil {
		t.Fatalf("symbol search: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.Name == "parseConfig" {
			found = true
		}
	}
	if !found {
		t.Errorf("symbol search for parseConfig did not find the indexed symbol; hits=%v", hits)
	}
}

// TestOrientSymbolReturnsSignatureTocMapMetadata covers @step-02 happy:
// orientation returns signature, file TOC, map, list, and metadata for a known
// symbol.
func TestOrientSymbolReturnsSignatureTocMapMetadata(t *testing.T) {
	_, projectID, svc := indexOrientFixture(t)
	out, err := svc.OrientSymbol(context.Background(), projectID, "parseConfig")
	if err != nil {
		t.Fatalf("orient: %v", err)
	}
	if !out.Found {
		t.Fatalf("expected parseConfig to be found, reason=%s", out.Reason)
	}
	if out.Signature == "" {
		t.Errorf("expected a non-empty signature, got %q", out.Signature)
	}
	if len(out.FileTOC) == 0 {
		t.Errorf("expected a file TOC, got none")
	}
	if len(out.List) == 0 {
		t.Errorf("expected a list of symbols, got none")
	}
	if out.Symbol["name"] != "parseConfig" {
		t.Errorf("expected symbol metadata name parseConfig, got %v", out.Symbol["name"])
	}
	if _, ok := out.Symbol["path"]; !ok {
		t.Errorf("expected symbol metadata to carry path, got %v", out.Symbol)
	}
}

// TestOrientSymbolWithRationale covers @step-02 edge: rationale comments are
// returned linked to the enclosing symbol, each carrying source comment text.
func TestOrientSymbolWithRationale(t *testing.T) {
	_, projectID, svc := indexOrientFixture(t)
	out, err := svc.OrientSymbol(context.Background(), projectID, "parseConfig")
	if err != nil {
		t.Fatalf("orient: %v", err)
	}
	if !out.Found {
		t.Fatalf("expected parseConfig found")
	}
	if len(out.Rationale) == 0 {
		t.Errorf("expected rationale linked to parseConfig, got none")
	}
	for _, r := range out.Rationale {
		if r["text"] == nil || r["text"] == "" {
			t.Errorf("rationale has no text: %v", r)
		}
	}
}

// TestOrientUnknownSymbolNotCovered covers @step-02 edge: an unknown symbol
// returns empty or not-found with no fabricated symbol.
func TestOrientUnknownSymbolNotCovered(t *testing.T) {
	_, projectID, svc := indexOrientFixture(t)
	out, err := svc.OrientSymbol(context.Background(), projectID, "doesNotExistSymbol")
	if err != nil {
		t.Fatalf("orient: %v", err)
	}
	if out.Found {
		t.Errorf("expected not-found for unknown symbol, got %+v", out.Symbol)
	}
	if out.Reason == "" {
		t.Errorf("expected a not-found reason, got empty")
	}
}
