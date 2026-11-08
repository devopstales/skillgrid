package extract

import "sync"

// PoolConfig bounds the self-healing pool: MaxRespawns is how many times a
// crashing grammar worker respawns before the circuit breaker trips;
// TripAfter is the death count that trips the per-language breaker.
type PoolConfig struct {
	MaxRespawns int
	TripAfter   int
}

// Pool is the self-healing grammar pool + per-language circuit breaker (the
// v1 extract.Pool shape, MaxRespawns=3, TripAfter=3). A panicking grammar
// quarantines the file to regex and records a death; the breaker trips on
// repeated deaths so a bad grammar does not re-panic on every file of its
// language.
type Pool struct {
	mu          sync.Mutex
	quarantined map[string]map[string]bool // lang -> path -> true
	breaker     map[string]bool            // lang -> tripped
	respawns    map[string]int             // lang -> respawns used
	deaths      map[string]int             // lang -> deaths seen
	MaxRespawns int
	TripAfter   int
}

// NewPool returns a self-healing pool. cfg zero values fall back to the
// defaults (MaxRespawns=3, TripAfter=3).
func NewPool(cfg PoolConfig) *Pool {
	mr := cfg.MaxRespawns
	if mr <= 0 {
		mr = 3
	}
	ta := cfg.TripAfter
	if ta <= 0 {
		ta = 3
	}
	return &Pool{
		quarantined: map[string]map[string]bool{},
		breaker:     map[string]bool{},
		respawns:    map[string]int{},
		deaths:      map[string]int{},
		MaxRespawns: mr,
		TripAfter:   ta,
	}
}

// IsQuarantined reports whether path for lang is quarantined (a prior panic).
func (p *Pool) IsQuarantined(lang, path string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.quarantined[lang][path]
}

// IsBreakerTripped reports whether the circuit breaker for lang has tripped.
func (p *Pool) IsBreakerTripped(lang string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.breaker[lang]
}

// Tripped is the test-facing alias of IsBreakerTripped.
func (p *Pool) Tripped(lang string) bool {
	return p.IsBreakerTripped(lang)
}

// RespawnsUsed returns how many respawns were recorded for lang.
func (p *Pool) RespawnsUsed(lang string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.respawns[lang]
}

// Quarantine marks path for lang as quarantined (subsequent extracts go
// straight to regex).
func (p *Pool) Quarantine(lang, path string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.quarantined[lang] == nil {
		p.quarantined[lang] = map[string]bool{}
	}
	p.quarantined[lang][path] = true
}

// RecordDeath counts a grammar death, respawns the worker up to the bound, and
// trips the per-language breaker on repeated deaths.
func (p *Pool) RecordDeath(lang string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deaths[lang]++
	if p.deaths[lang] <= p.MaxRespawns {
		p.respawns[lang]++
	}
	if p.deaths[lang] >= p.TripAfter {
		p.breaker[lang] = true
	}
}

// TierFor reports the tier a file of lang at path should use given the pool's
// current state: quarantined or breaker-tripped → regex; otherwise the caller
// proceeds with the grammar (rich or map). This is the "a 4th go file goes
// straight to regex" decision point.
func (p *Pool) TierFor(lang, path string) Tier {
	if p.IsQuarantined(lang, path) || p.IsBreakerTripped(lang) {
		return TierRegex
	}
	// Not quarantined: the caller picks rich vs map based on coverage.
	if coveredLanguages[lang] {
		return TierRich
	}
	if langMapsFor(lang) != nil {
		return TierMap
	}
	return TierRegex
}
