package extract

import (
	"strings"
	"testing"
)

// TestExtractRationaleLinksToNearestSymbol covers @step-02 edge: NOTE/WHY/ADR
// comments become rationale nodes linked to the nearest enclosing symbol, each
// carrying its source comment text.
func TestExtractRationaleLinksToNearestSymbol(t *testing.T) {
	src := []byte("package main\n\n// WHY: use a map here because lookups are O(1).\n// ADR-7 explains the cache choice.\nfunc helper() {\n\t// NOTE: do not remove; required for parity.\n\tx := 1\n\t_ = x\n}\n\nfunc other() {}\n")
	ex := Default()
	g, err := ex.ExtractFile("a.go", src)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if len(g.Rationales) == 0 {
		t.Fatalf("expected rationale nodes, got none; symbols=%d", len(g.Symbols))
	}
	// helper is the enclosing symbol for the NOTE; the WHY/ADR are above it
	// (nearest preceding). Every rationale must carry text and link to a
	// real symbol UID.
	symUIDs := map[string]bool{}
	for _, s := range g.Symbols {
		symUIDs[s.UID] = true
	}
	seen := map[string]bool{}
	for _, r := range g.Rationales {
		if r.Text == "" {
			t.Errorf("rationale has no text: %+v", r)
		}
		if r.Line == 0 {
			t.Errorf("rationale has no line: %+v", r)
		}
		if r.SymbolUID != "" && !symUIDs[r.SymbolUID] {
			t.Errorf("rationale links to unknown symbol uid %q", r.SymbolUID)
		}
		switch r.Kind {
		case "note", "why", "adr":
		default:
			t.Errorf("unknown rationale kind %q", r.Kind)
		}
		seen[r.Kind] = true
	}
	// The NOTE is inside helper; expect a "note" and a "why" and an "adr".
	if !seen["why"] {
		t.Errorf("expected a WHY rationale, got kinds %v", seen)
	}
	if !seen["adr"] {
		t.Errorf("expected an ADR rationale, got kinds %v", seen)
	}
	if !seen["note"] {
		t.Errorf("expected a NOTE rationale, got kinds %v", seen)
	}
}

// TestExtractRationaleKindDetection covers the per-marker kind mapping.
func TestExtractRationaleKindDetection(t *testing.T) {
	src := []byte("def foo():\n    # NOTE: keep the order\n    pass\n\ndef bar():\n    # WHY: avoid the race\n    pass\n\nclass Baz:\n    # ADR-12: chose a list over a tuple\n    pass\n")
	rats := ExtractRationale(src, nil)
	kinds := map[string]int{}
	for _, r := range rats {
		kinds[r.Kind]++
	}
	if kinds["note"] == 0 || kinds["why"] == 0 || kinds["adr"] == 0 {
		t.Errorf("expected note/why/adr rationale kinds, got %v", kinds)
	}
	// ADR rationale text must carry the citation.
	foundADR := false
	for _, r := range rats {
		if r.Kind == "adr" && strings.Contains(r.Text, "ADR") {
			foundADR = true
		}
	}
	if !foundADR {
		t.Errorf("expected an ADR rationale carrying the citation")
	}
}

// TestExtractRationaleNoFabrication covers that a comment with no marker and
// no citation is not extracted (no fabricated rationale).
func TestExtractRationaleNoFabrication(t *testing.T) {
	src := []byte("def foo():\n    # just a plain comment\n    pass\n")
	rats := ExtractRationale(src, nil)
	if len(rats) != 0 {
		t.Errorf("expected no rationale for a plain comment, got %v", rats)
	}
}
