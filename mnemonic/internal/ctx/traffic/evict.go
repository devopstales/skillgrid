package traffic

import (
	"sort"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
)

// EvictToBound enforces the global max_memories bound (plan §18.4): it loads
// every ctx_traffic_memories row (all sessions), and if the total exceeds max
// it deletes the oldest low-confidence rows first — ORDER BY confidence ASC,
// created_at ASC — until the count is back within max. It returns the number
// of rows evicted.
//
// Fail-open (ADR-0016): a store error degrades to (0, nil) — eviction is a
// maintenance path, never a hard failure. A nil store or non-positive max is a
// no-op.
func EvictToBound(cs *ctxstore.Store, max int) (int, error) {
	if cs == nil || max <= 0 {
		return 0, nil
	}
	all, err := cs.ListAllTrafficMemories()
	if err != nil {
		return 0, nil // fail-open: a store fault is not fatal to the learner
	}
	if len(all) <= max {
		return 0, nil
	}
	// Eviction order: lowest confidence first, then oldest created_at, then
	// memory_id for a stable tiebreak.
	sort.Slice(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.Confidence != b.Confidence {
			return a.Confidence < b.Confidence
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.MemoryID < b.MemoryID
	})
	excess := len(all) - max
	evicted := 0
	for _, m := range all[:excess] {
		if err := cs.DeleteTrafficMemory(m.MemoryID); err != nil {
			// Fail-open: skip a row that would not delete; do not abort.
			continue
		}
		evicted++
	}
	return evicted, nil
}
