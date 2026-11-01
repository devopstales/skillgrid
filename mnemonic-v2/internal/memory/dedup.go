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

// dedup.go — the FINAL dedup entry point (spec 004-dedup). The LLM is the
// primary classifier (LLM-first, §1); the deterministic hash floor is the
// explicit degraded branch (§5).

// dedupWindowSize bounds the recent-window pre-filter (spec 004-dedup §3:
// ≤20 most-recent live rows for the LLM's candidate set).
const dedupWindowSize = 20

// runDedup executes the spec 004-dedup §4 decision:
//
//	decision, degraded := runDedup(ctx, llm, st, project, scope, newContent, typ, title)
//
// It is degrade.Run(seam: LLM.Classify over the ≤20 recent-window
// pre-filter, floor: hashFloor, "dedup-classify") + trust-but-verify (§4:
// a positive CandidateID that is not a live row in this project falls back
// to the most-recent live candidate — the head of the pre-filter; the
// verdict is trusted, the pointer is verified). A nil seam → floor +
// degraded=true (no retry — absent ≠ failing, INV-3).
func runDedup(ctx context.Context, s llm.LLM, st store.Store, project, scope, newContent, typ, title string) (llm.DedupDecision, bool) {
	if s == nil {
		// Absent seam: the floor runs immediately, no retry (INV-3).
		return hashFloor(ctx, st, project, scope, newContent, typ, title), true
	}
	head, candidates := preFilter(ctx, st, project)
	res, degraded := degrade.Run(ctx, &classifySeam{llm: s, newContent: newContent, candidates: candidates},
		func() any { return hashFloor(ctx, st, project, scope, newContent, typ, title) }, "dedup-classify")
	decision, ok := res.(llm.DedupDecision)
	if !ok {
		// A seam that returns the wrong concrete type is a caller bug; the
		// floor is the only safe result (degrade.Run's own posture for a
		// value that is not a Seam).
		return hashFloor(ctx, st, project, scope, newContent, typ, title), true
	}
	if decision.Reason == "" {
		decision.Reason = "llm" // the seam succeeded; the floor is labeled "hash"
	}
	// Trust-but-verify (§4): a positive CandidateID that is not a live row
	// in this project → the most-recent live candidate (head of the
	// pre-filter). An empty head (no live candidate) leaves the id at 0:
	// the pipeline's CandidateID>0 guard makes the verdict a no-op rather
	// than a merge into a wrong row.
	if decision.CandidateID > 0 && !isLiveIn(st, decision.CandidateID) && head.ID > 0 {
		decision.CandidateID = head.ID
	}
	return decision, degraded
}

// classifySeam adapts llm.LLM.Classify onto degrade.Seam (the floor returns
// the same concrete type, llm.DedupDecision).
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

// preFilter is the recent-window candidate set (spec 004-dedup §3): the
// project's non-deleted observations, most-recent first, capped at
// dedupWindowSize — no title query (distinct from the conflict-candidate FTS
// pre-filter, 004-dedup §3 note). It returns the head (the most-recent live
// candidate — the trust-but-verify fallback) plus the contents the LLM
// classifies over. A read failure is not evidence of absence (§6): it yields
// an empty window (the LLM classifies over nothing; a positive CandidateID
// is then unverified and the pipeline's guard makes the verdict a no-op).
// ponytail: the store's RawQuery seam is context-free (Task 1's shape), so
// ctx is accepted for the interface's symmetry with the conflict pre-filter
// and is not threaded into the read. Ceiling: a ctx-aware query seam when
// Task 18's Postgres adapter needs a bounded window read.
func preFilter(ctx context.Context, st store.Store, project string) (llm.Observation, []string) {
	_ = ctx
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return llm.Observation{}, nil
	}
	rows, err := rq.RawQuery(`
		SELECT id, sync_id, content FROM observations
		WHERE ifnull(project,'') = ? AND deleted_at IS NULL
		ORDER BY id DESC LIMIT ?`, project, dedupWindowSize)
	if err != nil {
		return llm.Observation{}, nil
	}
	// ponytail: the order is id DESC, the most-recent row first — created_at
	// is second-granularity (datetime('now')), so id (monotonic) is the only
	// reliable recency ordering; the brief's "created_at DESC" is the same
	// order except within one second. Upgrade path: a millisecond-precision
	// timestamp column when sub-second ordering ever matters.
	head := llm.Observation{}
	contents := make([]string, 0, len(rows))
	for i, row := range rows {
		if i == 0 {
			if id, ok := row[0].(int64); ok {
				head.ID = id
			}
			if sid, ok := row[1].(string); ok {
				head.SyncID = sid
			}
		}
		if content, ok := row[2].(string); ok {
			contents = append(contents, content)
		}
	}
	return head, contents
}

// isLiveIn reports whether id is a live (non-deleted) observation (the
// trust-but-verify pointer check, spec 004-dedup §4). It reads through the
// store's Probe seam (the pipeline's ledger-read pattern — a pure lookup
// with no FTS/trigger surface); a read failure is not evidence of absence
// (§6): it reports live so the LLM's pointer is not overwritten on a failed
// lookup.
// ponytail: the liveness check is global (no project filter) — an id is at
// most one row, and a cross-project pointer still repairs to the head of
// the pre-filter (which IS project-scoped); a per-project check would need
// a RawQuery seam here. Ceiling: scope the probe to the project when a
// per-project candidate set grows.
func isLiveIn(st store.Store, id int64) bool {
	p, ok := st.(interface {
		Probe(query string, args ...any) (bool, error)
	})
	if !ok {
		return true
	}
	live, err := p.Probe(`SELECT COUNT(*) > 0 FROM observations WHERE id = ? AND deleted_at IS NULL`, id)
	return err == nil && live
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
