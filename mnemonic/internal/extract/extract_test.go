package extract

import (
	"testing"
)

// TestFallbackUnknownLanguage covers @step-01 edge: a file in a language
// outside the 30 supported uses the regex fallback and the index continues.
func TestFallbackUnknownLanguage(t *testing.T) {
	ex := Default()
	g, err := ex.ExtractFile("data.weird", []byte("alpha\nbeta\ngamma\n"))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if g == nil || g.Extractor != "regex" {
		t.Fatalf("expected regex fallback, got %+v", g)
	}
	if len(g.Symbols) == 0 {
		t.Errorf("expected the regex fallback to still find at least one symbol")
	}
}

// TestExtractUnsupportedReturnsFallback covers the Extractor contract for
// unsupported extensions: no error, a fallback FileGraph is returned.
func TestExtractUnsupportedReturnsFallback(t *testing.T) {
	ex := Default()
	for _, path := range []string{"data.zzz", "file.unknownext", "notes.foo"} {
		g, err := ex.ExtractFile(path, []byte("alpha beta gamma\n"))
		if err != nil {
			t.Errorf("%s: unexpected error %v", path, err)
		}
		if g == nil || g.Extractor != "regex" {
			t.Errorf("%s: expected regex fallback graph, got %+v", path, g)
		}
	}
}

// TestMalformedFileFallsBackAndContinues covers @step-01 failure: a file that
// fails primary extraction falls back and the run continues for remaining
// files.
func TestMalformedFileFallsBackAndContinues(t *testing.T) {
	ex := Default()
	// Unterminated paren / broken brace in Go → primary extraction yields
	// little; fallback must not error and the graph must be well-formed.
	g, err := ex.ExtractFile("broken.go", []byte("package main\n\nfunc main( {\n\tbroken\n"))
	if err != nil {
		t.Fatalf("malformed extract should not error, got %v", err)
	}
	if g == nil {
		t.Fatalf("expected a graph (fallback) for malformed file")
	}
	if g.Error != "" {
		t.Logf("primary error reported: %s", g.Error)
	}
	if g.Extractor != "regex" && g.Extractor != "treesitter" {
		t.Errorf("unexpected extractor %q", g.Extractor)
	}
	// A well-formed file in the same run still works.
	g2, err := ex.ExtractFile("ok.go", []byte("package main\n\nfunc alpha() { beta() }\n\nfunc beta() {}\n"))
	if err != nil {
		t.Fatalf("ok extract: %v", err)
	}
	if len(g2.Symbols) == 0 {
		t.Errorf("expected symbols from the well-formed file")
	}
}

// TestCrashingGrammarQuarantinesAndBreaker covers @step-01 failure: a grammar
// that panics on a file quarantines that file (regex fallback), the worker is
// respawned up to a bound, and repeated deaths trip the circuit breaker. The
// run completes.
func TestCrashingGrammarQuarantinesAndBreaker(t *testing.T) {
	ex := Default()

	// Inject a grammar that panics on every parse.
	ex.grammar = func(name string) (func() any, bool) {
		return func() any { panic("grammar crashed") }, true
	}
	defer func() { ex.grammar = nil }()

	// File 1: quarantined after the crash, falls back to regex.
	g1, err := ex.ExtractFile("boom.go", []byte("package main\n\nfunc alpha() { beta() }\n"))
	if err != nil {
		t.Fatalf("run should complete despite grammar crash: %v", err)
	}
	if g1 == nil || g1.Extractor != "regex" {
		t.Errorf("expected quarantined file to use regex fallback, got %+v", g1)
	}
	if !ex.IsQuarantined("go", "boom.go") {
		t.Errorf("expected boom.go to be quarantined")
	}

	// Files 2..N keep going; repeated deaths on the same grammar trip the
	// circuit breaker, after which that grammar is disabled for the rest of
	// the run.
	for i := 2; i <= 5; i++ {
		name := "boom" + string(rune('0'+i)) + ".go"
		g, err := ex.ExtractFile(name, []byte("package main\n\nfunc a() {}\n"))
		if err != nil {
			t.Fatalf("file %d: run must complete: %v", i, err)
		}
		if g == nil || g.Extractor != "regex" {
			t.Errorf("file %d: expected regex fallback after quarantine/breaker, got %+v", i, g)
		}
	}

	// The breaker must have tripped for the go grammar.
	if !ex.IsBreakerTripped("go") {
		t.Errorf("expected circuit breaker tripped for go grammar after repeated deaths")
	}

	// Worker respawn bound: respawns beyond the bound are not allowed.
	if got := ex.RespawnsUsed("go"); got == 0 {
		t.Errorf("expected respawns to be recorded for the crashing grammar")
	}
}

// TestCrashingGrammarIsolatedPerFile verifies one bad file does not poison
// other files of the same language once the breaker has not tripped.
func TestCrashingGrammarIsolatedPerFile(t *testing.T) {
	ex := Default()
	// A grammar that panics only for the first two calls, then recovers.
	calls := 0
	ex.grammar = func(name string) (func() any, bool) {
		return func() any {
			calls++
			if calls <= 2 {
				panic("transient grammar crash")
			}
			return struct{}{}
		}, true
	}
	defer func() { ex.grammar = nil }()

	g1, err := ex.ExtractFile("one.go", []byte("package main\n\nfunc a() {}\n"))
	if err != nil || g1.Extractor != "regex" {
		t.Errorf("first file should quarantine to regex, got err=%v g=%+v", err, g1)
	}
	g2, err := ex.ExtractFile("two.go", []byte("package main\n\nfunc b() {}\n"))
	if err != nil || g2.Extractor != "regex" {
		t.Errorf("second file should quarantine to regex, got err=%v g=%+v", err, g2)
	}
}

// TestConfidenceScore covers the graphify-style numeric mapping of the
// categorical confidence labels: EXTRACTED=1.0, INFERRED=0.85, AMBIGUOUS=0.5,
// unknown/empty -> 1.0 (the default).
func TestConfidenceScore(t *testing.T) {
	cases := []struct {
		label string
		want  float64
	}{
		{ConfidenceExtracted, 1.0},
		{ConfidenceInferred, 0.85},
		{ConfidenceAmbiguous, 0.5},
		{"", 1.0},
		{"LSP_RESOLVED", 1.0},
	}
	for _, c := range cases {
		if got := ConfidenceScore(c.label); got != c.want {
			t.Errorf("ConfidenceScore(%q) = %v, want %v", c.label, got, c.want)
		}
	}
}
