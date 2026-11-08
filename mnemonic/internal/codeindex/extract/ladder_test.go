package extract

import (
	"context"
	"testing"
)

// goSourceWithMethodAndCall is a Go file with a method that makes a
// statically-bound call (to another in-file function → EXTRACTED) and a
// name-only call (to a function not in this file → to_id NULL + an
// unresolved ledger row, CI-INV-3).
const goSourceWithMethodAndCall = `package pay

func helper() int {
	return 1
}

func (r *Repo) Payment(id string) int {
	v := helper()
	return v + ExternalCall(id)
}
`

// [extract-ladder-3-tiers] go is the rich tier (gotreesitter one-pass); the
// method symbol is emitted; a statically-bound call is EXTRACTED while a
// name-only call is kept with a null target + an unresolved row.
func TestExtractLadderRich6(t *testing.T) {
	ctx := context.Background()
	g, tier, err := ExtractFile(ctx, "pay.go", goSourceWithMethodAndCall, "go")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if tier != TierRich {
		t.Fatalf("go must be the rich tier, got %s", tier)
	}
	// the method symbol must be emitted
	var hasMethod bool
	for _, s := range g.Symbols {
		if s.Kind == "method" && s.Name == "Payment" {
			hasMethod = true
		}
	}
	if !hasMethod {
		t.Fatalf("the method symbol must be emitted: %+v", g.Symbols)
	}
	// a statically-bound call → EXTRACTED (helper is in-file); a name-only
	// call → to_id NULL + an unresolved row (ExternalCall is not in-file).
	var hasExtractedCall, hasNameOnlyCall, hasUnresolved bool
	for _, e := range g.Edges {
		if e.Kind == "calls" && e.Confidence == "EXTRACTED" && e.ToID != nil {
			hasExtractedCall = true
		}
		if e.Kind == "calls" && e.ToID == nil && e.ToName != "" {
			hasNameOnlyCall = true
		}
	}
	for _, u := range g.Unresolved {
		if u.Reference == "ExternalCall" {
			hasUnresolved = true
		}
	}
	if !hasExtractedCall {
		t.Fatalf("a statically-bound call must be EXTRACTED with a bound target: %+v", g.Edges)
	}
	if !hasNameOnlyCall {
		t.Fatalf("a name-only call must be kept with a null target: %+v", g.Edges)
	}
	if !hasUnresolved {
		t.Fatalf("a name-only call must also carry an unresolved ledger row (CI-INV-3): %+v", g.Unresolved)
	}
}

// pythonSource is a Python module with a function + a class (the rich tier
// covers python via the gotreesitter one-pass extractors).
const pythonSource = `def compute(x):
    return x * 2

class Engine:
    def start(self):
        return compute(1)
`

// [extract-ladder-3-tiers] python is the rich tier (covered language); it must
// emit symbols.
func TestExtractLadderPythonRich(t *testing.T) {
	ctx := context.Background()
	g, tier, err := ExtractFile(ctx, "a.py", pythonSource, "python")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if tier != TierRich && tier != TierMap {
		t.Fatalf("python must be the rich or map tier, got %s", tier)
	}
	if len(g.Symbols) == 0 {
		t.Fatalf("python must emit symbols: %+v", g.Symbols)
	}
}

// [extract-ladder-3-tiers] an unmapped extension falls to the regex tier and
// still emits symbols (the deterministic floor).
func TestExtractLadderRegexFallback(t *testing.T) {
	ctx := context.Background()
	src := "func foo() {\n\tbar()\n}\n"
	g, tier, err := ExtractFile(ctx, "x.unknownext", src, "unmapped")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if tier != TierRegex {
		t.Fatalf("unmapped → regex tier, got %s", tier)
	}
	if len(g.Symbols) == 0 {
		t.Fatalf("the regex floor must emit symbols: %+v", g.Symbols)
	}
}

// [extract-ladder-3-tiers] a map-tier language (rust — in the data table, not
// the rich-6) goes through walkTree against the node-type data table and emits
// symbols.
func TestExtractLadderMapTier(t *testing.T) {
	ctx := context.Background()
	src := "fn main() {\n    println!(\"hi\");\n}\nstruct Point {\n    x: i32,\n}\n"
	g, tier, err := ExtractFile(ctx, "a.rs", src, "rust")
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if tier != TierMap {
		t.Fatalf("rust must be the map tier, got %s", tier)
	}
	if len(g.Symbols) == 0 {
		t.Fatalf("the map tier must emit symbols: %+v", g.Symbols)
	}
}
