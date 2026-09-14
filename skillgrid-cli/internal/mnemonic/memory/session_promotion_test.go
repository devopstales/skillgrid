package memory

import (
	"context"
	"database/sql"
	"testing"
)

// promotionSummary is a session-close summary that clears the default quality
// threshold (>= 100 runes): a structured ## Goal block with real content.
const promotionSummary = `## Goal
Tune the auth token rotation flow and document the race found in the refresh path.

## Key Learnings:
1. JWT refresh tokens need atomic rotation to avoid races
2. bcrypt cost=12 is the right balance for our server
`

// promotionNodeCount counts the graph nodes promoted for sessionID.
func promotionNodeCount(t *testing.T, db *sql.DB, sessionID string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM symbols WHERE kind = 'session' AND uid = ?`, "session:"+sessionID,
	).Scan(&n); err != nil {
		t.Fatalf("count promoted nodes: %v", err)
	}
	return n
}

// promotionObservationID returns the id of the promoted observation for
// sessionID, or 0 when none exists.
func promotionObservationID(t *testing.T, db *sql.DB, sessionID string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(
		`SELECT id FROM observations
		 WHERE project = 'mem-promo' AND type = 'session_log'
		   AND title = ?
		 ORDER BY id LIMIT 1`,
		"[session "+sessionID+"] Session summary",
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		t.Fatalf("lookup promoted observation: %v", err)
	}
	return id
}

// TestSessionEndCreatesGraphNode covers @step-09: ending a session with a
// summary that meets the quality threshold promotes it to a permanent graph
// node. The node (a kind='session' symbol in the codeindex graph) carries the
// full summary as its signature, links to every session observation through
// 'promotes' edges (node -> observation), and is queryable through mem_search
// (the node is an observation row, so the FTS path finds it) and through
// codeindex graph traversal (the node is a symbol in the edges table).
func TestSessionEndCreatesGraphNode(t *testing.T) {
	fx := newFixture(t, "mem-promo")
	ctx := context.Background()
	db := fx.st.DB

	// Seed an indexed file + symbol so one session observation carries a
	// real graph_ref (step 07 linkage) the node can join to in traversal.
	seedSymbolFile(t, db, "/tmp/promo.go", "promoFunc", "uid-promo-1")

	obsA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "Chose atomic token rotation",
		Content:   "refresh tokens must rotate atomically to avoid the race",
		Source:    "/tmp/promo.go",
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	obsB, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "bugfix",
		Title:     "Fixed refresh race",
		Content:   "the refresh path raced on concurrent token swaps",
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}

	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummary); err != nil {
		t.Fatalf("session end: %v", err)
	}

	// 1) A permanent graph node exists for the session (exactly one).
	if n := promotionNodeCount(t, db, fx.sessID); n != 1 {
		t.Fatalf("expected exactly 1 promoted graph node, got %d", n)
	}

	// 2) The node is an observation (mem_search-queryable) and points at the
	// symbol row (codeindex graph) that carries it.
	obsID := promotionObservationID(t, db, fx.sessID)
	if obsID == 0 {
		t.Fatalf("expected a promoted observation row for the session summary")
	}
	node := graphRefOf(t, db, obsID)
	if !node.Valid {
		t.Fatalf("promoted observation must link to its graph symbol (graph_ref), got NULL")
	}

	// 3) The node links to ALL session observations through 'promotes' edges.
	linked := promotionLinkedObservations(t, db, node.Int64)
	for _, want := range []int64{obsA, obsB} {
		found := false
		for _, got := range linked {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("node %d has no 'promotes' edge to observation %d (linked: %v)", node.Int64, want, linked)
		}
	}

	// 4) Traversable through the codeindex graph: the node is reachable from
	// any session observation's symbol through the graph_ref -> edges path.
	var reach int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM edges e
		JOIN observations o ON o.graph_ref = e.to_id
		WHERE e.kind = 'promotes' AND o.id = ?`, obsA).Scan(&reach); err != nil {
		t.Fatalf("traversal query: %v", err)
	}
	if reach == 0 {
		t.Fatalf("node not traversable: no promotes edge reachable through graph_ref from obs %d", obsA)
	}

	// 5) Queryable via mem_search (the existing FTS path — no new query).
	hits, err := fx.svc.Search(ctx, "token rotation", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.ID == obsID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("promoted node not found via mem_search (hits: %d)", len(hits))
	}
}

// promotionLinkedObservations lists the observation ids a promoted graph node
// links to through 'promotes' edges.
func promotionLinkedObservations(t *testing.T, db *sql.DB, nodeID int64) []int64 {
	t.Helper()
	rows, err := db.Query(
		`SELECT to_id FROM edges WHERE kind = 'promotes' AND from_id = ? ORDER BY to_id`, nodeID,
	)
	if err != nil {
		t.Fatalf("linked observations query: %v", err)
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan linked: %v", err)
		}
		out = append(out, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("linked rows: %v", err)
	}
	return out
}

// TestSessionEndNoPromotionBelowThreshold covers @step-09: an empty summary
// or one below the quality threshold promotes nothing, while the session's
// observations stay accessible via mem_search (promotion is additive, not a
// gate on existing access).
func TestSessionEndNoPromotionBelowThreshold(t *testing.T) {
	fx := newFixture(t, "mem-promo")
	ctx := context.Background()
	db := fx.st.DB

	// Observations exist BEFORE the end — they must stay searchable no matter
	// what promotion does.
	obsA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "Chose trigram FTS",
		Content:   "trigram FTS5 keeps substring search fast on code",
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}

	// 1) Empty summary: the session still ends successfully, no node.
	if err := fx.svc.PromoteSession(ctx, fx.sessID, ""); err != nil {
		t.Fatalf("session end (empty): %v", err)
	}
	if n := promotionNodeCount(t, db, fx.sessID); n != 0 {
		t.Fatalf("empty summary must not promote, got %d nodes", n)
	}

	// 2) Low-quality summary (below the 100-rune threshold): no node either.
	if err := fx.svc.PromoteSession(ctx, fx.sessID, "done"); err != nil {
		t.Fatalf("session end (short): %v", err)
	}
	if n := promotionNodeCount(t, db, fx.sessID); n != 0 {
		t.Fatalf("below-threshold summary must not promote, got %d nodes", n)
	}

	// 3) The observations remain queryable via mem_search.
	hits, err := fx.svc.Search(ctx, "trigram", "any", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	found := false
	for _, h := range hits {
		if h.ID == obsA {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("session observation not found via mem_search after non-promoting end")
	}
}

// TestSessionEndIdempotentPromotion covers @step-09: ending the same session
// again (re-trigger) does not create a duplicate graph node — the existing
// node observation, its symbol, and its 'promotes' edge set are reused as-is.
func TestSessionEndIdempotentPromotion(t *testing.T) {
	fx := newFixture(t, "mem-promo")
	ctx := context.Background()
	db := fx.st.DB

	obsA, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "Chose idempotent upsert",
		Content:   "re-triggered promotion must reuse the existing node",
	})
	if err != nil {
		t.Fatalf("save A: %v", err)
	}
	obsB, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "discovery",
		Title:     "Discovered dedup key",
		Content:   "the dedup key is the node observation identity",
	})
	if err != nil {
		t.Fatalf("save B: %v", err)
	}

	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummary); err != nil {
		t.Fatalf("first end: %v", err)
	}
	obsID := promotionObservationID(t, db, fx.sessID)
	if obsID == 0 {
		t.Fatalf("expected a promoted node after the first end")
	}
	node := graphRefOf(t, db, obsID)
	if !node.Valid {
		t.Fatalf("first promotion must link the node observation to a symbol")
	}
	firstLinked := promotionLinkedObservations(t, db, node.Int64)
	firstEdges := promotionEdgeCount(t, db, node.Int64)
	if len(firstLinked) != 2 || firstEdges != len(firstLinked) {
		t.Fatalf("first promotion: expected edges to both observations, got %v (%d edges)", firstLinked, firstEdges)
	}

	// Re-trigger: ending the same session again must not duplicate anything.
	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummary); err != nil {
		t.Fatalf("second end: %v", err)
	}
	if n := promotionNodeCount(t, db, fx.sessID); n != 1 {
		t.Fatalf("re-triggered end must not duplicate the node, got %d", n)
	}
	obsID2 := promotionObservationID(t, db, fx.sessID)
	if obsID2 != obsID {
		t.Fatalf("re-triggered end must reuse the existing node (got observation %d, want %d)", obsID2, obsID)
	}
	node2 := graphRefOf(t, db, obsID2)
	if !node2.Valid || node2.Int64 != node.Int64 {
		t.Fatalf("reused node must keep its symbol (got %v, want %d)", node2, node.Int64)
	}
	secondLinked := promotionLinkedObservations(t, db, node.Int64)
	if len(secondLinked) != 2 {
		t.Fatalf("re-triggered end must not duplicate 'promotes' edges: %v (want both observations exactly once)", secondLinked)
	}
	for _, want := range []int64{obsA, obsB} {
		count := 0
		for _, got := range secondLinked {
			if got == want {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("observation %d linked %d times after re-trigger (want exactly 1)", want, count)
		}
	}
	if got := promotionEdgeCount(t, db, node.Int64); got != firstEdges {
		t.Fatalf("re-triggered end must not duplicate 'promotes' edges (got %d, want %d)", got, firstEdges)
	}
}

// promotionEdgeCount counts a node's 'promotes' edges.
func promotionEdgeCount(t *testing.T, db *sql.DB, nodeID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM edges WHERE kind = 'promotes' AND from_id = ?`, nodeID,
	).Scan(&n); err != nil {
		t.Fatalf("count promotes edges: %v", err)
	}
	return n
}

// promotionSummaryChanged is a SECOND close summary that also clears the
// quality threshold (>= 100 runes, >= 1 "## " heading) but differs from
// promotionSummary — the "re-derivation" case where the L0 summary is refreshed
// between two SessionEnd calls (014 step 09 M1).
const promotionSummaryChanged = `## Goal
Document the refresh-token rotation race and the atomic rotation fix.

## Key Learnings:
1. Rotating JWT refresh tokens must be atomic to avoid races
2. bcrypt cost=12 is the balance chosen for the auth server
`

// TestSessionEndIdempotentPromotionChangedSummary covers 014 step 09 M1:
// re-triggering SessionEnd for the same session with a CHANGED summary must
// reuse the existing node observation (updating its content in place) rather
// than inserting a second one. Pre-fix the dedup was content-keyed, so a
// changed summary missed the dedup and created a second node.
func TestSessionEndIdempotentPromotionChangedSummary(t *testing.T) {
	fx := newFixture(t, "mem-promo")
	ctx := context.Background()
	db := fx.st.DB

	// Two session observations the node will link to (excluded from the node's
	// own row by sessionObservationIDs' title filter).
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "decision",
		Title:     "Chose atomic rotation",
		Content:   "refresh tokens must rotate atomically",
	}); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if _, err := fx.svc.Save(ctx, SaveInput{
		SessionID: fx.sessID,
		Type:      "discovery",
		Title:     "Found the race",
		Content:   "the refresh path raced on concurrent swaps",
	}); err != nil {
		t.Fatalf("save B: %v", err)
	}

	// 1) Promote with summary X (clears the threshold).
	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummary); err != nil {
		t.Fatalf("first end (summary A): %v", err)
	}
	obsID := promotionObservationID(t, db, fx.sessID)
	if obsID == 0 {
		t.Fatalf("expected a promoted node after the first end")
	}
	var contentA string
	if err := db.QueryRow(
		`SELECT content FROM observations WHERE id = ?`, obsID,
	).Scan(&contentA); err != nil {
		t.Fatalf("read node content A: %v", err)
	}
	if contentA != promotionSummary {
		t.Fatalf("node content must be the first summary, got %q", contentA)
	}

	// 2) Promote AGAIN with a DIFFERENT summary Y (also clears the threshold).
	if err := fx.svc.PromoteSession(ctx, fx.sessID, promotionSummaryChanged); err != nil {
		t.Fatalf("second end (summary B): %v", err)
	}

	// 3) Exactly ONE node observation must exist (no second node created).
	var nodeCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM observations
		 WHERE project = 'mem-promo' AND type = 'session_log'
		   AND title = ?`,
		"[session "+fx.sessID+"] Session summary",
	).Scan(&nodeCount); err != nil {
		t.Fatalf("count node observations: %v", err)
	}
	if nodeCount != 1 {
		t.Fatalf("changed-summary re-trigger must not duplicate the node observation, got %d", nodeCount)
	}

	// 4) The existing node is reused (same id) and its content is refreshed to
	// the new summary (the FTS update trigger reindexes it).
	var contentB string
	if err := db.QueryRow(
		`SELECT content FROM observations WHERE id = ?`, obsID,
	).Scan(&contentB); err != nil {
		t.Fatalf("read node content B: %v", err)
	}
	if contentB != promotionSummaryChanged {
		t.Fatalf("node content must be refreshed to the new summary, got %q", contentB)
	}
	if got := promotionObservationID(t, db, fx.sessID); got != obsID {
		t.Fatalf("re-trigger must reuse the existing node observation (got %d, want %d)", got, obsID)
	}

	// 5) The node still links to both session observations (no duplicate edges).
	if n := promotionNodeCount(t, db, fx.sessID); n != 1 {
		t.Fatalf("changed-summary re-trigger must keep exactly 1 graph symbol, got %d", n)
	}
}
