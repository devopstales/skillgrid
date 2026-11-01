package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/degrade"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/intelligence/llm"
)

// dedup.go — the FINAL dedup entry point (Task 8 fills the LLM path's
// internals; this shape is stable from Task 5).

// dedupWindowSize bounds the recent-window pre-filter (spec 004-dedup §3:
// ≤20 most-recent live rows for the LLM's candidate set).
const dedupWindowSize = 20

// runDedup executes the spec 004-dedup §4 decision:
//
//	decision, degraded := runDedup(ctx, llm, st, project, scope, newContent, typ, title)
//
// It is degrade.Run(seam: LLM.Classify over the ≤20 recent-window
// pre-filter, floor: hashFloor, "dedup-classify") + trust-but-verify
// (Task 8). A nil seam → floor + degraded=true (no retry — absent ≠
// failing).
//
// Task 5 ships the floor as the whole path: the seam is plumbed through
// degrade.Run's seam slot, but the LLM branch's classification internals
// (Classify over the pre-filter + trust-but-verify pointer check) land in
// Task 8.
func runDedup(ctx context.Context, s llm.LLM, st store.Store, project, scope, newContent, typ, title string) (llm.DedupDecision, bool) {
	if s == nil {
		// Absent seam: the floor runs immediately, no retry (INV-3).
		return hashFloor(ctx, st, project, scope, newContent, typ, title), true
	}
	// Task 8: degrade.Run(ctx, &classifySeam{llm: s, newContent: newContent,
	// candidates: preFilter(ctx, st, project) contents}, floor, "dedup-classify")
	// + trust-but-verify (a positive CandidateID that is not a live row in
	// this project falls back to the most-recent live candidate).
	// ponytail: Task 5 returns the floor verdict even with a seam present —
	// the seam is plumbed (Task 8's degrade.Run wiring needs zero
	// call-site changes) but the LLM branch's internals are not built yet.
	// Ceiling: Task 8 replaces this branch; the verdict + reason contract
	// ("llm" vs "hash") does not change.
	_ = s
	_ = degrade.Run // Task 8's wiring target (kept referenced so the import is live)
	return hashFloor(ctx, st, project, scope, newContent, typ, title), true
}

// classifySeam adapts llm.LLM.Classify onto degrade.Seam for Task 8's
// wiring (the floor returns the same concrete type, llm.DedupDecision).
type classifySeam struct {
	llm        llm.LLM
	newContent string
	candidates []string
}

func (c *classifySeam) Call(ctx context.Context, a, b any) (any, error) {
	return c.llm.Classify(ctx, c.newContent, c.candidates)
}

// hashFloor is the deterministic dedup floor (spec 004-dedup §5): look up
// (normalized_hash, project, scope, type, title) via the idx_obs_dedupe
// index. Hit → noop on the matched row (the pipeline bumps duplicate_count
// + last_seen_at + updated_at — no new row). Miss → add. A failed read is
// not evidence of absence (spec 004-dedup §6): it falls through to add
// rather than invent a duplicate.
func hashFloor(ctx context.Context, st store.Store, project, scope, newContent, typ, title string) llm.DedupDecision {
	// ponytail: the floor reads through the gate's read-only lookup (a fresh
	// txn, rolled back immediately) so the hash identity check works even
	// when the store's own hash (Task 1's AddObservation) was written by the
	// store's normalizer. Ceiling: Task 6's replay ledger path makes the
	// floor's lookup transactional inside the save txn.
	txn, err := beginSaveTxn(ctx, st)
	if err != nil {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"}
	}
	defer func() { _ = txn.Rollback() }()
	id, found, err := txn.LookupHash(ctx, project, scope, typ, title, storeNormalizedHash(newContent))
	if err != nil || !found {
		return llm.DedupDecision{Verdict: llm.VerdictAdd, Reason: "hash"}
	}
	return llm.DedupDecision{Verdict: llm.VerdictNoop, CandidateID: id, Reason: "hash"}
}

// beginSaveTxn narrows a store to the save-transaction gate (core/store
// SaveTXN — the single adapter; INV-2: the pipeline never touches a raw
// handle) and opens the save transaction. A store without the gate seam
// (a test fake) is an error: the save pipeline needs the transactional
// path.
func beginSaveTxn(ctx context.Context, st store.Store) (store.SaveTXN, error) {
	type beginner interface {
		BeginSaveTxn(ctx context.Context) (store.SaveTXN, error)
	}
	backend, ok := st.(beginner)
	if !ok {
		return nil, fmt.Errorf("memory: store %T does not implement the save-transaction seam", st)
	}
	return backend.BeginSaveTxn(ctx)
}

// preFilter returns the ≤dedupWindowSize most-recent live observations for
// the project (the LLM's candidate set, spec 004-dedup §3). Task 8
// consults it; the window bound has one owner here.
func preFilter(ctx context.Context, st store.Store, project string) []llm.Observation {
	res, err := st.Search(ctx, store.SearchQuery{Query: "trigram", Project: project, Limit: dedupWindowSize})
	if err != nil {
		return nil
	}
	out := make([]llm.Observation, 0, len(res.Observations))
	for _, o := range res.Observations {
		out = append(out, llm.Observation{ID: o.ID, SyncID: o.SyncID, Type: o.Type, Title: o.Title, Content: o.Content})
	}
	return out
}

// storeNormalizedHash is the store's normalization (lowercase + whitespace
// collapse + SHA-256 hex — Engram store.go:13665, the same bytes the gate
// hashes into observations.normalized_hash, so the floor's lookup matches
// the stored column). Duplicated rather than exported to keep the
// dependency direction (memory → core/store); the normalization test pins
// the agreement.
func storeNormalizedHash(content string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(content), " "))
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}
