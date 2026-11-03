package memory

import (
	"context"
	"fmt"
	"strings"
)

// Knowledge-class dimension (ADR-0046, TICKET-03): a nullable durability
// label on observations distinguishing episodic (session-specific) from
// convention (durable) knowledge. The column lives on observations_core
// (migration 070, one-way); existing rows stay NULL until
// BackfillKnowledgeClass labels them.
//
// The class is a HEURISTIC (non-load-bearing): it is inferred from the
// coarse type column on save when the caller does not supply one, and it is
// never a source of truth on its own — it labels, it does not gate. The
// dimension is NOT a KB-per-class: every class lives in the same store
// (ADR-0012 keeps SQLite the single source of truth).

const (
	KnowledgeClassEpisodic     = "episodic"
	KnowledgeClassConvention   = "convention"
	KnowledgeClassDoc          = "doc"
	KnowledgeClassUnclassified = "unclassified"
)

// knowledgeClasses is the closed set of valid durability classes. It is the
// single source of truth for validation (IsValidKnowledgeClass).
var knowledgeClasses = map[string]struct{}{
	KnowledgeClassEpisodic:     {},
	KnowledgeClassConvention:   {},
	KnowledgeClassDoc:          {},
	KnowledgeClassUnclassified: {},
}

// IsValidKnowledgeClass reports whether c is one of the four durability
// classes. Case-insensitive.
func IsValidKnowledgeClass(c string) bool {
	_, ok := knowledgeClasses[strings.ToLower(strings.TrimSpace(c))]
	return ok
}

// InferKnowledgeClass maps the coarse observation type to its durability
// class (the ADR-0046 heuristic):
//
//	decision, convention, architecture, config, standing -> convention
//	session_log, learning, discovery, bugfix             -> episodic
//	doc, prd, research, roadmap                           -> doc
//	anything else / empty                                 -> unclassified
//
// It is a save-time inference only: the result is written to a nullable
// column and callers may read it back, but nothing routes on it. doc/prd/
// research/roadmap are not in the coarse validTypes set, so they only ever
// fire on types a caller coerces or on a future taxonomy expansion.
func InferKnowledgeClass(typ string) string {
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "decision", "convention", "architecture", "config", "standing":
		return KnowledgeClassConvention
	case "session_log", "learning", "discovery", "bugfix":
		return KnowledgeClassEpisodic
	case "doc", "prd", "research", "roadmap":
		return KnowledgeClassDoc
	default:
		return KnowledgeClassUnclassified
	}
}

// BackfillKnowledgeClass fills knowledge_class for this project's rows where
// it is still NULL, using the inferKnowledgeClass mapping on each row's type.
// Re-runnable: it only touches NULL rows, so a second run (and every
// subsequent run) is a no-op. The IS NULL predicate IS the idempotency guard
// (a pure SQL UPDATE needs no external bookkeeping). Returns the number of
// rows updated.
func (s *Service) BackfillKnowledgeClass(ctx context.Context) (int, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, fmt.Errorf("memory service not initialized")
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE observations_core
		SET knowledge_class = CASE type
			WHEN 'decision'     THEN 'convention'
			WHEN 'convention'   THEN 'convention'
			WHEN 'architecture' THEN 'convention'
			WHEN 'config'       THEN 'convention'
			WHEN 'standing'     THEN 'convention'
			WHEN 'session_log'  THEN 'episodic'
			WHEN 'learning'     THEN 'episodic'
			WHEN 'discovery'    THEN 'episodic'
			WHEN 'bugfix'       THEN 'episodic'
			WHEN 'doc'          THEN 'doc'
			WHEN 'prd'          THEN 'doc'
			WHEN 'research'     THEN 'doc'
			WHEN 'roadmap'      THEN 'doc'
			ELSE 'unclassified'
		END
		WHERE project = ? AND knowledge_class IS NULL`, s.projectID)
	if err != nil {
		return 0, fmt.Errorf("backfill knowledge_class: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("backfill knowledge_class rows affected: %w", err)
	}
	return int(n), nil
}

// classFilterClause is the knowledge_class WHERE fragment ("" = unfiltered,
// byte-identical SQL to pre-ADR-0046). A non-empty class must be one of the
// four valid values (exact match — unclassified rows are only returned when
// the caller explicitly asks for unclassified).
func classFilterClause(class string) (string, error) {
	class = strings.ToLower(strings.TrimSpace(class))
	if class == "" {
		return "", nil
	}
	if !IsValidKnowledgeClass(class) {
		return "", fmt.Errorf("invalid knowledge_class %q (allowed: episodic, convention, doc, unclassified)", class)
	}
	return " AND COALESCE(o.knowledge_class, 'unclassified') = ?", nil
}

// SearchWithKnowledgeClass is Search (default scope) + the optional
// knowledge_class filter + per-row class labeling. An empty class leaves the
// query UNCHANGED from pre-ADR-0046 (no WHERE fragment, no extra args) — the
// no-regression contract — while every hit carries its per-row class label
// (a NULL row is surfaced as "unclassified").
func (s *Service) SearchWithKnowledgeClass(ctx context.Context, query, matchMode, class string, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, fmt.Errorf("memory service not initialized")
	}
	classClause, err := classFilterClause(class)
	if err != nil {
		return nil, err
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	args := []any{ftsQuery, s.projectID}
	if classClause != "" {
		args = append(args, class)
	}
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT `+obsSplitSelectCols+`
		FROM `+obsSplitFrom+`
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?`+classClause+`
		  AND (st.expires_at IS NULL OR st.expires_at = '' OR strftime('%s', st.expires_at) > strftime('%s', 'now'))
		  AND (tm.invalid_at IS NULL OR tm.invalid_at = '' OR strftime('%s', tm.invalid_at) > strftime('%s', 'now'))
		ORDER BY COALESCE(st.pinned, 0) DESC, bm25(observations_fts)
		LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("search observations: %w", err)
	}
	defer rows.Close()
	out, err := scanObservations(rows)
	if err != nil {
		return nil, err
	}
	// Per-class labeling (ADR-0046): a row whose class is still NULL
	// (pre-backfill, or a type the heuristic does not cover) is surfaced as
	// "unclassified" so the caller always knows each hit's class.
	for i := range out {
		if out[i].KnowledgeClass == "" {
			out[i].KnowledgeClass = KnowledgeClassUnclassified
		}
	}
	for _, o := range out {
		s.BumpRetrievalUsage(ctx, o.ID)
	}
	return out, nil
}
