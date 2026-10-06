package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The session-to-graph auto-promotion (change 014, step 09) turns a session
// close with a quality summary into a PERMANENT graph node.
//
// Representation: the node is a single row in each of the two stores that
// make it queryable:
//
//   - mem_search: an observation (type 'session_log', the existing "session
//     narrative" type — no new type needed) whose title/content carry the
//     close summary. It is indexed by the existing observations_fts triggers,
//     so the existing FTS search path finds it with no new query.
//   - codeindex graph traversal: a kind='session' symbol whose signature is
//     the summary (the symbol_fts trigger indexes it) and whose
//     observations.graph_ref points at it — the same triple-store cross-link
//     shape step 07 established for code observations.
//
// The node links to the session's observations through 'promotes' edges in
// the existing edges table (from_id = the node symbol, to_id = the
// observation id): the endpoint is polymorphic (021 relaxed the symbols FK),
// the edges.file_id points at a session pseudo-file so the file FK holds, and
// the kind='promotes' namespace does not collide with any codeindex edge kind
// (the 011 indexer owns kinds like 'calls'/'references'; 'promotes' never
// appears in its upserts, so the unique constraint cannot collide).
//
// The brief's `LayerSummary` does not exist as a struct: at SessionEnd the
// available layer data is the `summary` string argument, which IS the L0
// layer summary (the distill hook consumes it as such). The quality check
// therefore runs on that string.

// sessionNodeType is the symbols.kind marker for a promoted session node.
const sessionNodeType = "session"

// sessionNodeEdgeKind is the edges.kind for node -> observation links.
const sessionNodeEdgeKind = "promotes"

// defaultPromotionMinLength is the minimum rune count a close summary must
// reach to be promoted (014 step 09). A one-line "done" or a title-only
// stub does not carry reusable session knowledge; a real summary does.
// Configurable via mnemonic.promotion.min_length (see PromotionConfig).
const defaultPromotionMinLength = 100

// defaultPromotionMinSections is the minimum number of "## " markdown
// headings a summary must contain (a structured session summary has at least
// one section). Fixed at the documented default; not currently configurable.
const defaultPromotionMinSections = 1

// PromotionConfig tunes the session-close quality threshold (014 step 09).
type PromotionConfig struct {
	// MinLength is the minimum rune count of the summary to promote.
	MinLength int
	// MinSections is the minimum number of "## " headings required.
	MinSections int
}

func defaultPromotionConfig() PromotionConfig {
	return PromotionConfig{MinLength: defaultPromotionMinLength, MinSections: defaultPromotionMinSections}
}

// promotionMeetsThreshold reports whether summary clears the quality
// threshold: it must be at least MinLength runes long AND contain at least
// MinSections "## " markdown headings. An empty summary always fails.
func (s *Service) promotionMeetsThreshold(summary string) bool {
	cfg := s.promotionCfg
	if cfg.MinLength <= 0 || cfg.MinSections <= 0 {
		cfg = defaultPromotionConfig()
	}
	trimmed := strings.TrimSpace(summary)
	if trimmed == "" {
		return false
	}
	if len([]rune(trimmed)) < cfg.MinLength {
		return false
	}
	sections := 0
	for _, line := range strings.Split(trimmed, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			sections++
		}
	}
	return sections >= cfg.MinSections
}

// PromoteSession ends a session with summary AND, when the summary meets the
// quality threshold, promotes it to a permanent graph node.
//
// Ordering: this runs BEFORE the session-close distill hook (step 013) and
// SYNCHRONOUSLY — the promoted node is durable and readable the moment
// SessionEnd returns, so a follow-up read in the next session can already see
// it. The distill hook stays detached/async after it (best-effort by design):
// promotion is the durability guarantee, distillation is the opportunistic L1
// extraction. Both are best-effort: a promotion failure is logged to stderr
// and does not fail the close.
//
// Idempotency: re-triggering SessionEnd for the same session finds the
// existing node (dedup on observations (project, type, title) — session-keyed,
// so a changed summary refreshes the node in place rather than duplicating it)
// and reuses it; the 'promotes' edges are upserted, so no duplicates accumulate.
func (s *Service) PromoteSession(ctx context.Context, sessionID, summary string) error {
	if err := s.SessionEnd(ctx, sessionID, summary); err != nil {
		return err
	}
	if !s.promotionMeetsThreshold(summary) {
		fmt.Printf("[mnemonic] session %s: summary below promotion threshold, skipping graph promotion\n", sessionID)
		return nil
	}
	if id, err := s.promoteSessionToGraph(ctx, sessionID, summary); err != nil {
		// Best-effort: promotion must never break session close.
		fmt.Printf("[mnemonic] session %s: graph promotion failed (best-effort): %v\n", sessionID, err)
		return nil
	} else {
		fmt.Printf("[mnemonic] session %s: promoted to graph node %d\n", sessionID, id)
	}
	return nil
}

// promoteSessionToGraph creates (or reuses) the permanent graph node for
// sessionID and links it to the session's observations. Returns the node
// observation id.
func (s *Service) promoteSessionToGraph(ctx context.Context, sessionID, summary string) (int64, error) {
	db := s.store.DB
	now := time.Now().UTC().Format(time.RFC3339)

	// 1) The node observation — the mem_search-queryable half. Title is
	// deterministic (session-scoped) and content is the summary, so the
	// dedup query below finds the existing node on a re-trigger.
	nodeTitle := "[session " + sessionID + "] Session summary"
	nodeID, err := s.upsertPromotedObservation(ctx, sessionID, nodeTitle, summary, now)
	if err != nil {
		return 0, fmt.Errorf("promoted observation: %w", err)
	}

	// 2) The node symbol — the codeindex graph half. uid encodes the session
	// so re-indexing (the 011 indexer keys on uid) and the idempotency check
	// both resolve deterministically.
	uid := "session:" + sessionID
	var symbolID int64
	err = db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid = ?`, uid).Scan(&symbolID)
	if err == sql.ErrNoRows {
		fileID, err := s.sessionPseudoFileID(ctx, sessionID, now)
		if err != nil {
			return 0, fmt.Errorf("session pseudo-file: %w", err)
		}
		if _, err := db.ExecContext(ctx, `
			INSERT INTO symbols (file_id, name, qualified_name, kind, language, signature, start_line, end_line, content_hash, uid)
			VALUES (?, ?, ?, ?, 'session', ?, 1, 1, ?, ?)`,
			fileID, nodeTitle, nodeTitle, sessionNodeType, summary, "ch-"+uid, uid,
		); err != nil {
			return 0, fmt.Errorf("insert session symbol: %w", err)
		}
		if err := db.QueryRowContext(ctx, `SELECT id FROM symbols WHERE uid = ?`, uid).Scan(&symbolID); err != nil {
			return 0, fmt.Errorf("lookup session symbol: %w", err)
		}
	} else if err != nil {
		return 0, fmt.Errorf("lookup session symbol: %w", err)
	}

	// 3) Bind the node observation to its symbol — the triple-store cross-link
	// (014 step 07 shape) that makes the node traversable from observations.
	if _, err := db.ExecContext(ctx, `UPDATE observations SET graph_ref = ? WHERE id = ?`, symbolID, nodeID); err != nil {
		return 0, fmt.Errorf("bind node graph_ref: %w", err)
	}

	// 4) Link the node to every session observation: 'promotes' edges
	// (node symbol -> observation id). The 021 schema keeps edges.file_id a
	// NOT-NULL FK to files, but a session node is not a source file — the
	// pseudo-file (path "session:<id>") is its anchor. The pseudo-file is
	// never picked up by the codeindex indexer (which discovers files on
	// disk), so it stays an inert graph anchor. Upserted, so a re-trigger is
	// a no-op.
	fileID, err := s.sessionPseudoFileID(ctx, sessionID, now)
	if err != nil {
		return 0, fmt.Errorf("session pseudo-file: %w", err)
	}
	obsIDs, err := s.sessionObservationIDs(ctx, sessionID)
	if err != nil {
		return 0, fmt.Errorf("list session observations: %w", err)
	}
	for _, obsID := range obsIDs {
		label := strconv.FormatInt(obsID, 10)
		// Upsert: re-triggering must not accumulate duplicate edges. The
		// unique constraint (kind, from_id, file_id, to_id, to_name,
		// target_path, line) includes to_id (the observation id) and
		// to_name/target_path (deterministic labels), so the same (node,
		// observation) pair always resolves to the same edge.
		//
		// SQLite pitfall: in a UNIQUE index, NULL values are NOT considered
		// equal, so a NULL column in the unique key makes the upsert a no-op
		// (every insert is "new"). We therefore populate every unique-key
		// column with a non-NULL deterministic value — to_name and
		// target_path are labels, and line is 0 (session nodes have no
		// source line; 0 is a real value the codeindex indexer never writes
		// for its edges, so there is no collision risk).
		if _, err := db.ExecContext(ctx, `
			INSERT INTO edges (kind, from_id, file_id, to_id, to_name, target_path, confidence, context, confidence_score, line, valid_from)
			VALUES (?, ?, ?, ?, ?, ?, 'EXTRACTED', 'session_promotion', 1.0, 0, ?)
			ON CONFLICT(kind, from_id, file_id, to_id, to_name, target_path, line)
			DO UPDATE SET confidence = excluded.confidence`,
			sessionNodeEdgeKind, symbolID, fileID, obsID, "obs:"+label, "observation:"+label,
			time.Now().Unix(),
		); err != nil {
			return 0, fmt.Errorf("upsert promotes edge: %w", err)
		}
	}
	return nodeID, nil
}

// sessionPseudoFileID returns (creating if needed) the files row a session
// node's edges attach to. See step 4 of promoteSessionToGraph for why a
// pseudo-file is the anchor.
func (s *Service) sessionPseudoFileID(ctx context.Context, sessionID, now string) (int64, error) {
	path := "session:" + sessionID
	var fileID int64
	err := s.store.DB.QueryRowContext(ctx, `SELECT id FROM files WHERE path = ?`, path).Scan(&fileID)
	if err == sql.ErrNoRows {
		if _, err := s.store.DB.ExecContext(ctx, `
			INSERT INTO files (path, mtime_ns, size, content_hash, indexed_at)
			VALUES (?, 0, 0, ?, ?)`, path, "sess-"+sessionID, now); err != nil {
			return 0, err
		}
		if err := s.store.DB.QueryRowContext(ctx, `SELECT id FROM files WHERE path = ?`, path).Scan(&fileID); err != nil {
			return 0, err
		}
	} else if err != nil {
		return 0, err
	}
	return fileID, nil
}

// upsertPromotedObservation inserts the node observation, or — when a node
// already exists for the same session — updates the existing row in place and
// returns its id. Direct SQL (not Save) so the 24h normalized-hash dedup cannot
// collapse the node into an unrelated recent observation.
//
// The dedup is SESSION-keyed (project, type='session_log', title), NOT
// content-keyed (014 step 09 M1): the session summary can CHANGE between two
// SessionEnd calls (a re-derivation with a refreshed L0 summary), and a
// content-keyed dedup would miss and insert a SECOND node observation. Keying
// on the session-stable title (which encodes the session id) makes promotion
// idempotent per session regardless of content, and the UPDATE refreshes the
// content so the FTS update trigger reindexes the node with the latest summary.
func (s *Service) upsertPromotedObservation(ctx context.Context, sessionID, title, content, now string) (int64, error) {
	db := s.store.DB
	var existingID int64
	err := db.QueryRowContext(ctx, `
		SELECT id FROM observations
		WHERE project = ? AND type = 'session_log' AND title = ?
		  AND deleted_at IS NULL
		ORDER BY id LIMIT 1`,
		s.projectID, title,
	).Scan(&existingID)
	if err == nil {
		if _, err := db.ExecContext(ctx, `
			UPDATE observations SET content = ?, last_seen_at = ?, updated_at = ? WHERE id = ?`,
			content, now, now, existingID); err != nil {
			return 0, fmt.Errorf("touch promoted observation: %w", err)
		}
		return existingID, nil
	}
	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("dedup lookup promoted observation: %w", err)
	}
	res, err := db.ExecContext(ctx, `
		INSERT INTO observations (
			session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, created_at, updated_at, source, prompt_id, tool_name,
			owner, visibility, status, retrieval_usage, expires_at
		) VALUES (?, 'session_log', ?, ?, ?, ?, ?, ?, 0, ?, ?, 'session_promotion', NULL, NULL, ?, 'private', 'active', 0, NULL)`,
		sessionID, title, content, s.projectID, "", nullString("session:"+sessionID),
		normalizedHash(title, content, "session_log"), now, now, sessionID,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// sessionObservationIDs lists the live observation ids of the session,
// newest last (stable upsert order). The node's own observation is excluded
// so a re-trigger never re-links a stale id.
func (s *Service) sessionObservationIDs(ctx context.Context, sessionID string) ([]int64, error) {
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id FROM observations
		WHERE project = ? AND session_id = ? AND deleted_at IS NULL
		  AND NOT (type = 'session_log' AND title = ?)
		ORDER BY id`,
		s.projectID, sessionID, "[session "+sessionID+"] Session summary",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
