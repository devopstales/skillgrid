package judgment

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// StoreCache adapts a *store.Store (the ctx accessor) to the judgment.CacheStore
// seam. It is fail-open (ADR-0016): a nil store or any accessor error degrades
// to a cache miss / no-op write, so the judgment pipeline never blocks.
type StoreCache struct {
	s *store.Store
}

// NewStoreCache wraps a ctx store accessor (may be nil — then every call is a
// fail-open no-op / miss).
func NewStoreCache(s *store.Store) *StoreCache {
	return &StoreCache{s: s}
}

// GetJudgmentCache retrieves a cached decision by fingerprint. A nil store, a
// missing row, or any error is a miss (ok=false) — fail-open.
func (c *StoreCache) GetJudgmentCache(fingerprint string) (action string, keepScore float64, ok bool) {
	if c == nil || c.s == nil {
		return "", 0, false
	}
	jc, err := c.s.GetJudgmentCache(fingerprint)
	if err != nil || jc == nil {
		return "", 0, false
	}
	return jc.Decision, jc.Score, true
}

// PutJudgmentCache persists a decision by fingerprint. A nil store or any error
// is a no-op — fail-open.
func (c *StoreCache) PutJudgmentCache(fingerprint, action string, keepScore float64, judgedAt int64) {
	if c == nil || c.s == nil {
		return
	}
	_ = c.s.PutJudgmentCache(fingerprint, action, keepScore, judgedAt)
}
