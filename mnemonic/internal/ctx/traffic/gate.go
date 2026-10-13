// Package traffic — gate.go
//
// Gate is the human-gate for traffic memories (plan §18, M23). The learner
// (T12) only emits candidate memories; a human promotes one via
// `mnemonic ctx learn apply <id>` (→ status "applied") or demotes it via
// `mnemonic ctx learn revoke <id>` (→ status "revoked"). Only existing
// candidates can be promoted — a non-existent id is a clear error (plan §18.7
// "No automatic promotion").
//
// The Gate is a thin wrapper over *ctxstore.Store (ADR-0033 shared handle).
// It is testable: inject a *ctxstore.Store built on any *store.Store.
package traffic

import (
	"fmt"
	"time"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// Gate is the human-gate for traffic memory promotion/demotion. It wraps the
// ctx store accessor (nil-safe: a nil store degrades every method to a
// fail-open no-op, ADR-0016).
type Gate struct {
	store *ctxstore.Store
}

// NewGate builds a Gate over the given ctx store accessor. A nil store is
// accepted and makes every method fail-open (ADR-0016).
func NewGate(s *ctxstore.Store) *Gate {
	return &Gate{store: s}
}

// Apply promotes a traffic memory to applied (status "applied"). It returns a clear
// "not found" error when the memory id does not exist (the human-gate: only
// existing candidates can be applied). Re-applying an already-applied memory
// is an idempotent no-op (it refreshes applied_at). A nil/missing store is a
// fail-open no-op (ADR-0016) — no error, no panic.
func (g *Gate) Apply(memoryID string) error {
	if g.store == nil {
		return nil // fail-open (ADR-0016)
	}
	m, err := g.store.GetTrafficMemory(memoryID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("traffic memory %q not found (only existing candidates can be applied)", memoryID)
	}
	now := time.Now().UnixMilli()
	_, err = g.store.SetTrafficMemoryStatus(memoryID, "applied", &now)
	return err
}

// Revoke demotes a traffic memory to revoked (status "revoked"). It returns a clear
// "not found" error when the memory id does not exist. Re-revoking is
// idempotent. A nil/missing store is a fail-open no-op (ADR-0016).
func (g *Gate) Revoke(memoryID string) error {
	if g.store == nil {
		return nil // fail-open (ADR-0016)
	}
	m, err := g.store.GetTrafficMemory(memoryID)
	if err != nil {
		return err
	}
	if m == nil {
		return fmt.Errorf("traffic memory %q not found (only existing memories can be revoked)", memoryID)
	}
	_, err = g.store.SetTrafficMemoryStatus(memoryID, "revoked", nil)
	return err
}

// ActiveMemories returns the applied (active) traffic memories for a session
// that feed compression context. On a pre-068 schema (no status column) every
// row reads as "candidate" (COALESCE), so this returns empty — fail-open
// (ADR-0016). A nil store returns an empty slice, no error.
func (g *Gate) ActiveMemories(sessionID string) ([]*ctxstore.TrafficMemory, error) {
	if g.store == nil {
		return nil, nil // fail-open (ADR-0016)
	}
	all, err := g.store.ListTrafficMemories(sessionID)
	if err != nil {
		return nil, err
	}
	var active []*ctxstore.TrafficMemory
	for _, m := range all {
		if m.Status == "applied" {
			active = append(active, m)
		}
	}
	return active, nil
}
