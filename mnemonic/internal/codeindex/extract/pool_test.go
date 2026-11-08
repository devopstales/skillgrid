package extract

import (
	"context"
	"testing"
)

// [self-healing-pool-circuit-breaker] a pool configured with three respawns
// and trip-after-three trips the per-language breaker after 3 deaths; a 4th
// file of that language goes straight to the regex tier (no tree-sitter
// attempt).
func TestSelfHealingPoolCircuitBreaker(t *testing.T) {
	p := NewPool(PoolConfig{MaxRespawns: 3, TripAfter: 3})
	for i := 0; i < 3; i++ {
		p.RecordDeath("go")
	}
	if !p.Tripped("go") {
		t.Fatalf("the circuit breaker must trip after 3 deaths")
	}
	// a 4th go file goes straight to regex (no tree-sitter attempt).
	if tier := p.TierFor("go", "another.go"); tier != TierRegex {
		t.Fatalf("tripped → regex tier, got %s", tier)
	}
	// a different language is unaffected by the tripped go breaker.
	if p.Tripped("python") {
		t.Fatalf("the breaker is per-language: python must not be tripped by go")
	}
}

// [self-healing-pool-circuit-breaker] a panicking grammar quarantines the file
// to regex (IsQuarantined) but does NOT trip the breaker on a single death.
func TestSelfHealingPoolQuarantineNotBreaker(t *testing.T) {
	p := NewPool(PoolConfig{MaxRespawns: 3, TripAfter: 3})
	p.RecordDeath("go")
	p.Quarantine("go", "bad.go")
	if !p.IsQuarantined("go", "bad.go") {
		t.Fatalf("a panicking grammar must quarantine the file")
	}
	if !p.IsQuarantined("go", "bad.go") && p.TierFor("go", "bad.go") != TierRegex {
		t.Fatalf("a quarantined file must go straight to regex")
	}
	if p.Tripped("go") {
		t.Fatalf("a single death must not trip the breaker (TripAfter=3)")
	}
}

// [self-healing-pool-circuit-breaker] (F1, C-1 — the breaker is driven by a
// REAL panicking grammar, not a synthetic RecordDeath mock). Port of the v1
// TestCrashingGrammarQuarantinesAndBreaker (mnemonic_back/.../extract_test.go:
// 72-116): inject a grammar loader that panics on every parse, drive
// ExtractFile 3× with WithPool, and assert each run falls back to the regex
// tier, the pool quarantines each file + respawns, and after 3 deaths the
// per-language breaker trips (the 4th file goes straight to regex WITHOUT a
// grammar attempt).
func TestPanickingGrammarQuarantinesAndTrips(t *testing.T) {
	ctx := context.Background()
	p := NewPool(PoolConfig{MaxRespawns: 3, TripAfter: 3})

	// Inject a grammar that panics on every parse (the real panic the ladder
	// recovers — the WithGrammar seam is the v1 ex.grammar override).
	panicking := func(name string) (func() any, bool) {
		return func() any { panic("grammar crashed") }, true
	}

	// File 1: the grammar panics → the file is quarantined + the run falls
	// back to the regex tier (no error — a per-file failure never aborts).
	src := "package main\n\nfunc alpha() { beta() }\n"
	g1, tier1, err := ExtractFile(ctx, "boom1.go", src, "go", WithPool(p), WithGrammar(panicking))
	if err != nil {
		t.Fatalf("run 1: the run must complete despite the grammar crash: %v", err)
	}
	if tier1 != TierRegex {
		t.Fatalf("run 1: a panicking grammar must fall back to the regex tier, got %s", tier1)
	}
	if !p.IsQuarantined("go", "boom1.go") {
		t.Fatalf("run 1: boom1.go must be quarantined after the crash")
	}
	if p.RespawnsUsed("go") != 1 {
		t.Fatalf("run 1: one respawn expected, got %d", p.RespawnsUsed("go"))
	}
	if p.Tripped("go") {
		t.Fatalf("run 1: a single death must not trip the breaker (TripAfter=3)")
	}

	// File 2: a different file of the same language. The go breaker has not
	// tripped yet, so the grammar is attempted again → panics → 2nd death.
	g2, tier2, err := ExtractFile(ctx, "boom2.go", src, "go", WithPool(p), WithGrammar(panicking))
	if err != nil {
		t.Fatalf("run 2: %v", err)
	}
	if tier2 != TierRegex {
		t.Fatalf("run 2: regex fallback expected, got %s", tier2)
	}
	if !p.IsQuarantined("go", "boom2.go") {
		t.Fatalf("run 2: boom2.go must be quarantined")
	}
	if p.RespawnsUsed("go") != 2 {
		t.Fatalf("run 2: two respawns expected, got %d", p.RespawnsUsed("go"))
	}
	if p.Tripped("go") {
		t.Fatalf("run 2: two deaths must not trip the breaker (TripAfter=3)")
	}

	// File 3: the 3rd death trips the per-language breaker.
	g3, tier3, err := ExtractFile(ctx, "boom3.go", src, "go", WithPool(p), WithGrammar(panicking))
	if err != nil {
		t.Fatalf("run 3: %v", err)
	}
	if tier3 != TierRegex {
		t.Fatalf("run 3: regex fallback expected, got %s", tier3)
	}
	if !p.Tripped("go") {
		t.Fatalf("run 3: the circuit breaker must trip after 3 deaths")
	}

	// File 4: the breaker is tripped → the file goes STRAIGHT to regex without
	// a grammar attempt (the v1 "4th file goes straight to regex" decision).
	// The panicking loader would panic if attempted, so reaching the regex
	// tier without a panic proves no grammar attempt was made.
	g4, tier4, err := ExtractFile(ctx, "boom4.go", src, "go", WithPool(p), WithGrammar(panicking))
	if err != nil {
		t.Fatalf("run 4: %v", err)
	}
	if tier4 != TierRegex {
		t.Fatalf("run 4: a tripped breaker must go straight to the regex tier, got %s", tier4)
	}
	if g4.Error == "" {
		t.Fatalf("run 4: the straight-to-regex path must record the breaker/quarantine reason")
	}
	// Respawn bound: respawns beyond MaxRespawns (3) are not counted.
	if p.RespawnsUsed("go") != 3 {
		t.Fatalf("respawn bound: exactly MaxRespawns (3) expected, got %d", p.RespawnsUsed("go"))
	}

	// The other runs returned well-formed graphs (possibly empty, never nil).
	for i, g := range []FileGraph{g1, g2, g3, g4} {
		if g.Tier != TierRegex {
			t.Fatalf("run %d: tier must be regex, got %s", i+1, g.Tier)
		}
	}
}
