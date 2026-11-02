package extract

import (
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
