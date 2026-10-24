// Scope routing for the two-target fact store (change
// 2026-10-24-global-llm-wiki, Change A, TICKET-02).
//
// A fact operation needs two store roles: the DATA store (where the fact row
// lives) and the TRAIL store (where the session_events row goes). They are
// the same store for scope=project; for scope=global the data store is the
// __global__ pseudo-project and the trail store is the caller's project store
// (C3: the session — and its session_events FK — lives in the project store,
// so a global fact_add must trail there or roll back on the FK).
//
// This file holds the Store methods (Add / Search / Forget / Decay /
// DecayAll) with the category + trail-store seams, the Scope type, the
// Route resolver, and the Category mapping. The Store handle and the
// read-only listing live in facts.go.
package facts

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
)

// Scope is the fact-store routing scope. project (the default) keeps the
// pre-change behavior; global routes the data to the cross-project store;
// all is read-only (fused search, Task 3) and errors for write ops.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeGlobal  Scope = "global"
	ScopeAll     Scope = "all"
)

// Valid reports whether the scope is one of project|global|all.
func (s Scope) Valid() bool {
	return s == ScopeProject || s == ScopeGlobal || s == ScopeAll
}

// GlobalProjectID is the pseudo-project id of the cross-project fact store
// (file name __global__.sqlite, created by the TICKET-01 bootstrap). It
// duplicates service.GlobalProjectID: the facts package cannot import
// service (import cycle via hybrid), so the constant is pinned here and the
// two are asserted equal by the MCP layer's scope tests.
const GlobalProjectID = "__global__"

// Route resolves a scope to its (dataStore, trailStore) pair. The caller
// opens the store handles (Service.Open / OpenGlobal) and passes them here;
// Route only wraps them in facts.Store. For scope=project the caller passes
// the project store for both (the pre-change behavior, unchanged); for
// scope=global the caller passes the __global__ store as dataDB and the
// project store as trailDB (C3: the session — and its session_events FK —
// lives in the project store).
func Route(scope Scope, dataDB, trailDB *sql.DB) (data, trail *Store, err error) {
	switch scope {
	case ScopeProject, "":
		if dataDB == nil {
			return nil, nil, errors.New("facts: route requires the project store handle")
		}
		s := New(dataDB, "")
		return s, s, nil
	case ScopeGlobal:
		if dataDB == nil || trailDB == nil {
			return nil, nil, errors.New("facts: scope global requires both the __global__ store and the project store handles")
		}
		return New(dataDB, GlobalProjectID), New(trailDB, ""), nil
	case ScopeAll:
		return nil, nil, errors.New("facts: scope 'all' is read-only (fused search); writes need scope project or global")
	default:
		return nil, nil, fmt.Errorf("facts: unknown scope %q (want project, global, or all)", scope)
	}
}

// Category constants for the two fact targets (migration 072, change
// 2026-10-24-global-llm-wiki): 'memory' (agent operational notes, the schema
// default) and 'user' (user profile).
const (
	CategoryMemory = "memory"
	CategoryUser   = "user"
)

// CategoryForTarget maps the MCP target param to the facts.category value.
// Blank maps to the default 'memory' (the pre-change behavior for callers
// that omit the param); an unrecognized target is a clean error.
// ponytail: the per-category char budgets (memory 2200 / user 1375) are
// enforced by the batch ops ticket (TICKET-04); this mapping is the routing
// that makes the right budget applicable.
func CategoryForTarget(target string) (string, error) {
	switch target {
	case "", CategoryMemory:
		return CategoryMemory, nil
	case CategoryUser:
		return CategoryUser, nil
	default:
		return "", fmt.Errorf("facts: unknown target %q (want memory or user)", target)
	}
}

// Add inserts one fact with the given category and records a session_events
// trail in the same store, in one transaction: if the session is unknown
// (FK on session_events.session_id) the fact is rolled back with the trail —
// no partial write. Use AddWith for the C3 cross-store case (the trail lands
// in the caller's project store while the fact lands in the global store).
func (s *Store) Add(ctx context.Context, sessionID, content, category string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("facts store not initialized")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, errors.New("fact content is required")
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = CategoryMemory
	}
	if category != CategoryMemory && category != CategoryUser {
		return 0, fmt.Errorf("fact category %q is not one of memory|user", category)
	}
	// Pre-check the session so the error names it (the acceptance contract)
	// instead of surfacing a bare FK constraint failure.
	if !s.sessionExists(ctx, sessionID) {
		return 0, fmt.Errorf("fact add: unknown session %q (the session must exist in this project store)", sessionID)
	}
	now := time.Now().UTC().Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin fact add: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, content, category, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert fact: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("fact id: %w", err)
	}

	payload, err := json.Marshal(map[string]any{"fact_id": id})
	if err != nil {
		return 0, fmt.Errorf("marshal fact payload: %w", err)
	}
	var seq int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return 0, fmt.Errorf("next event sequence: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, payload, timestamp)
		VALUES (?, ?, ?, 'fact_add', 'success', ?, ?)`,
		sessionID, s.project, seq, string(payload), now,
	); err != nil {
		return 0, fmt.Errorf("insert fact_add event for session %q: %w", sessionID, err)
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit fact add: %w", err)
	}
	return id, nil
}

// AddWith is Add with the trail written to the trail store (C3). When
// trailStore is nil (or wraps the same *sql.DB as the data store) the
// pre-change single-store behavior applies: the trail goes to the data
// store's session_events in the same transaction as the fact insert (one
// rollback covers both).
//
// Cross-store contract: with distinct data and trail stores, the fact insert
// and the trail insert are NOT one atomic transaction (SQLite has no
// cross-connection transaction). The trail is written FIRST so an unknown
// session (FK against the trail store's sessions) fails before the fact row
// lands — "unknown session → no fact row, clean error". A crash between the
// two commits can leave an orphan trail row (no fact) but never an orphan
// fact row with a missing trail. The cross-store trail payload carries the
// category (the fact id is not observable: the id is allocated by the data
// store, and the trail row exists only as a session audit entry).
func (s *Store) AddWith(ctx context.Context, sessionID, content, category string, trailStore *Store) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("facts store not initialized")
	}
	if trailStore == nil || trailStore.db == nil || trailStore.db == s.db {
		return s.Add(ctx, sessionID, content, category)
	}
	category = strings.TrimSpace(category)
	if category == "" {
		category = CategoryMemory
	}
	if category != CategoryMemory && category != CategoryUser {
		return 0, fmt.Errorf("fact category %q is not one of memory|user", category)
	}
	// The session must exist in the TRAIL store (where the trail row's FK
	// lives) — pre-checked so the error names it.
	if !trailStore.sessionExists(ctx, sessionID) {
		return 0, fmt.Errorf("fact add: unknown session %q (the session must exist in the project store)", sessionID)
	}
	// Trail first: an FK failure here (the pre-check races) must not leave a
	// fact row behind. The payload carries the target category (the fact id
	// is not observable cross-store — see the contract above).
	if err := trailStore.insertEvent(ctx, sessionID, "fact_add",
		map[string]any{"category": category}); err != nil {
		return 0, err
	}
	id, err := s.addFact(ctx, content, category)
	if err != nil {
		return 0, fmt.Errorf("add fact (trail already written for session %q): %w", sessionID, err)
	}
	return id, nil
}

// addFact inserts the fact row only (no trail) and returns its id.
func (s *Store) addFact(ctx context.Context, content, category string) (int64, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return 0, errors.New("fact content is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO facts (content, category, created_at, updated_at)
		VALUES (?, ?, ?, ?)`, content, category, now, now)
	if err != nil {
		return 0, fmt.Errorf("insert fact: %w", err)
	}
	return res.LastInsertId()
}

// Search is the default lexical fact search (FTS5 over facts_fts, bm25-
// ranked, soft-deleted facts excluded). Every call appends a session_events
// trail (action_type="fact_search"); with no matches nothing is written.
// The trailStore param (C3) routes the trail to the caller's project store
// when the data store is the global store; nil means the data store (the
// pre-change behavior).
func (s *Store) Search(ctx context.Context, sessionID, query string, limit int, trailStore *Store) ([]Fact, error) {
	return s.SearchWith(ctx, sessionID, query, limit, false, trailStore)
}

// SearchWith is Search with an explicit soft-delete filter.
func (s *Store) SearchWith(ctx context.Context, sessionID, query string, limit int, includeDeleted bool, trailStore *Store) ([]Fact, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("facts store not initialized")
	}
	ftsQuery, err := buildFactFTSQuery(query)
	if err != nil {
		return nil, err
	}
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	return s.searchRanked(ctx, sessionID, query, ftsQuery, limit, includeDeleted, trailStore)
}

// searchRanked runs the FTS5 bm25-rank query and — when a session is given —
// records the fact_search trail on the trail store (nil = this store).
// sessionID "" suppresses the per-store trail (FusedSearch suppresses both
// and records one fused trail in the project store itself).
func (s *Store) searchRanked(ctx context.Context, sessionID, query, ftsQuery string, limit int, includeDeleted bool, trailStore *Store) ([]Fact, error) {
	deletedClause := ""
	if !includeDeleted {
		deletedClause = " AND f.deleted_at IS NULL"
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT f.id, f.content, f.category, f.importance_score, f.recency_decay, f.maturity_tier,
		       f.retrieval_usage, f.created_at, f.updated_at
		FROM facts f
		INNER JOIN facts_fts ON facts_fts.rowid = f.id
		WHERE facts_fts MATCH ?`+deletedClause+`
		ORDER BY bm25(facts_fts)
		LIMIT ?`, ftsQuery, limit)
	if err != nil {
		return nil, fmt.Errorf("search facts: %w", err)
	}
	defer rows.Close()
	var out []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Content, &f.Category, &f.ImportanceScore, &f.RecencyDecay,
			&f.MaturityTier, &f.RetrievalUsage, &f.CreatedAt, &f.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan fact: %w", err)
		}
		out = append(out, f)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fact search: %w", err)
	}
	if sessionID == "" || len(out) == 0 {
		return out, nil
	}
	trailer := s
	if trailStore != nil && trailStore.db != s.db {
		trailer = trailStore
	}
	ids := make([]int64, len(out))
	for i, f := range out {
		ids[i] = f.ID
	}
	return out, trailer.insertEvent(ctx, sessionID, "fact_search", map[string]any{
		"query":    query,
		"mode":     "fts",
		"fact_ids": ids,
	})
}

// FusedHit is one hit in a FusedSearch result: the fact row plus the labels
// that name where it came from (scope: global|project; category: memory|user).
// The labels are required by the scope=all contract — the agent must be able
// to tell the two disjoint stores apart from the result alone.
type FusedHit struct {
	Fact
	Scope    Scope  `json:"scope"`
	Category string `json:"category"`
}

// FusedSearch is the scope=all read path: it searches both the global and the
// project store and STABLE-MERGES the results by each store's own BM25 rank —
// not RRF, no score normalization (blueprint M1). The stores are disjoint
// (a fact lives in exactly one store), so "take the best-ranked hit from
// either store, then the next, ..." is well defined: the merge walks both
// rank-ordered lists and emits the hit whose own rank is lowest, ties broken
// in favor of the global store (deterministic order).
//
// Each hit is labeled with its store's scope and the fact's stored category.
// The session_events trail (action_type="fact_search", payload mode "fused",
// matched fact ids tagged by store scope) goes to the PROJECT store only —
// never to the global store (blueprint M2). With no matches nothing is
// written; an empty store contributes no hits and no error.
func FusedSearch(ctx context.Context, sessionID, query string, limit int, includeDeleted bool, global, project *Store) ([]FusedHit, error) {
	if global == nil || global.db == nil || project == nil || project.db == nil {
		return nil, errors.New("facts: fused search requires both the __global__ store and the project store handles")
	}
	ftsQuery, err := buildFactFTSQuery(query)
	if err != nil {
		return nil, err
	}
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	// Both per-store searches pass an empty session: no per-store trail. The
	// single fused trail is recorded in the project store below.
	gRes, gErr := global.searchRanked(ctx, "", query, ftsQuery, limit, includeDeleted, nil)
	pRes, pErr := project.searchRanked(ctx, "", query, ftsQuery, limit, includeDeleted, nil)
	if gErr != nil {
		return nil, fmt.Errorf("fused search (global store): %w", gErr)
	}
	if pErr != nil {
		return nil, fmt.Errorf("fused search (project store): %w", pErr)
	}
	merged := mergeFusedHits(gRes, pRes, limit)
	if len(merged) > 0 && sessionID != "" {
		globalIDs := make([]int64, 0)
		projectIDs := make([]int64, 0)
		for _, h := range merged {
			if h.Scope == ScopeGlobal {
				globalIDs = append(globalIDs, h.ID)
			} else {
				projectIDs = append(projectIDs, h.ID)
			}
		}
		if err := project.insertEvent(ctx, sessionID, "fact_search", map[string]any{
			"query":       query,
			"mode":        "fused",
			"global_ids":  globalIDs,
			"project_ids": projectIDs,
		}); err != nil {
			return nil, err
		}
	}
	return merged, nil
}

// mergeFusedHits stable-merges two rank-ordered hit lists (list order IS the
// store's bm25 rank). Ties on the same rank go to the global list first —
// the global store is the cross-project one, and a deterministic tie-break
// keeps results reproducible.
func mergeFusedHits(g, p []Fact, limit int) []FusedHit {
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	out := make([]FusedHit, 0, limit)
	for gi, pi := 0, 0; len(out) < limit && (gi < len(g) || pi < len(p)); {
		gTake := gi < len(g) && (pi >= len(p) || gi <= pi)
		if gTake {
			var category string
			if g[gi].Category == "" {
				category = CategoryMemory
			} else {
				category = g[gi].Category
			}
			out = append(out, FusedHit{Fact: g[gi], Scope: ScopeGlobal, Category: category})
			gi++
		} else {
			var category string
			if p[pi].Category == "" {
				category = CategoryMemory
			} else {
				category = p[pi].Category
			}
			out = append(out, FusedHit{Fact: p[pi], Scope: ScopeProject, Category: category})
			pi++
		}
	}
	return out
}

// Forget soft-deletes a fact (deleted_at = now, updated_at refreshed). The
// trail (action_type="fact_forget") goes to the trailStore when given
// (C3), else the data store. Forgetting an already-forgotten fact is a
// no-op; an unknown id is an error and logs no event.
func (s *Store) Forget(ctx context.Context, sessionID string, factID int64, trailStore *Store) error {
	if s == nil || s.db == nil {
		return errors.New("facts store not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		UPDATE facts SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL`, now, now, factID)
	if err != nil {
		return fmt.Errorf("forget fact: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var exists int
		if err := s.db.QueryRowContext(ctx,
			`SELECT 1 FROM facts WHERE id = ?`, factID).Scan(&exists); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("forget fact: unknown fact id %d", factID)
			}
			return fmt.Errorf("forget fact: %w", err)
		}
		return nil // already soft-deleted: idempotent no-op
	}
	trailer := s
	if trailStore != nil && trailStore.db != s.db {
		trailer = trailStore
	}
	return trailer.insertEvent(ctx, sessionID, "fact_forget", map[string]any{
		"fact_id": factID,
	})
}

// Decay applies the 014 AKL importance decay to one fact:
//
//	importance_score = importance_score * exp(-decay_rate * age_days)
//
// reusing the memory package's recencyFactor. A zero rate is a no-op that
// still logs the trail row; an unknown id is an error. The trail
// (action_type="fact_decay") goes to the trailStore when given (C3).
func (s *Store) Decay(ctx context.Context, sessionID string, factID int64, trailStore *Store) (float64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("facts store not initialized")
	}
	var score, rate float64
	var createdAt string
	if err := s.db.QueryRowContext(ctx, `
		SELECT importance_score, recency_decay, created_at
		FROM facts WHERE id = ?`, factID).
		Scan(&score, &rate, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("decay fact: unknown fact id %d", factID)
		}
		return 0, fmt.Errorf("decay fact: %w", err)
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil || created.IsZero() {
		created = time.Now().UTC()
	}
	newScore := score * memory.RecencyFactor(time.Since(created), rate)
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE facts SET importance_score = ?, updated_at = ? WHERE id = ?`,
		newScore, now, factID); err != nil {
		return 0, fmt.Errorf("decay fact update: %w", err)
	}
	trailer := s
	if trailStore != nil && trailStore.db != s.db {
		trailer = trailStore
	}
	if err := trailer.insertEvent(ctx, sessionID, "fact_decay", map[string]any{
		"fact_id":   factID,
		"old_score": score,
		"new_score": newScore,
		"mode":      "akl",
	}); err != nil {
		return 0, err
	}
	return newScore, nil
}

// DecayAll runs the batch decay + below-threshold purge over every live fact
// in the data store. The single trail row (action_type="fact_decay_batch")
// goes to the trailStore when given (C3). An empty store is a no-op that
// still logs the batch event with zero counts.
func (s *Store) DecayAll(ctx context.Context, sessionID string, threshold float64, trailStore *Store) (decayed int, purged int, err error) {
	if s == nil || s.db == nil {
		return 0, 0, errors.New("facts store not initialized")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, importance_score, recency_decay, created_at
		FROM facts WHERE deleted_at IS NULL`)
	if err != nil {
		return 0, 0, fmt.Errorf("decay all query: %w", err)
	}
	defer rows.Close()
	type liveFact struct {
		id      int64
		score   float64
		rate    float64
		created string
	}
	var live []liveFact
	for rows.Next() {
		var f liveFact
		if err := rows.Scan(&f.id, &f.score, &f.rate, &f.created); err != nil {
			return 0, 0, fmt.Errorf("decay all scan: %w", err)
		}
		live = append(live, f)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("decay all iterate: %w", err)
	}
	for _, f := range live {
		created, parseErr := time.Parse(time.RFC3339, f.created)
		if parseErr != nil || created.IsZero() {
			created = time.Now().UTC()
		}
		newScore := f.score * memory.RecencyFactor(time.Since(created), f.rate)
		if _, err := s.db.ExecContext(ctx, `
			UPDATE facts SET importance_score = ?, updated_at = ? WHERE id = ?`,
			newScore, now, f.id); err != nil {
			return 0, 0, fmt.Errorf("decay all update %d: %w", f.id, err)
		}
		if newScore < threshold {
			if _, err := s.db.ExecContext(ctx, `
				UPDATE facts SET deleted_at = ?, updated_at = ?
				WHERE id = ? AND deleted_at IS NULL`, now, now, f.id); err != nil {
				return 0, 0, fmt.Errorf("purge fact %d: %w", f.id, err)
			}
			purged++
		}
		decayed++
	}
	trailer := s
	if trailStore != nil && trailStore.db != s.db {
		trailer = trailStore
	}
	if err := trailer.insertEvent(ctx, sessionID, "fact_decay_batch", map[string]any{
		"decayed":   decayed,
		"purged":    purged,
		"threshold": threshold,
	}); err != nil {
		return 0, 0, err
	}
	return decayed, purged, nil
}

// InsertEvent appends one session_events trail row to this store. Exported
// seam for the MCP batch path (handleFactBatch), whose trail store it must
// write through without reaching into the facts package's unexported SQL.
func (s *Store) InsertEvent(ctx context.Context, sessionID, actionType string, payload map[string]any) error {
	return s.insertEvent(ctx, sessionID, actionType, payload)
}

// insertEvent appends one session_events trail row with the next sequence
// for the session (the same monotonic-sequence convention as Add).
func (s *Store) insertEvent(ctx context.Context, sessionID, actionType string, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", actionType, err)
	}
	var seq int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(sequence),-1)+1 FROM session_events WHERE session_id = ?`,
		sessionID,
	).Scan(&seq); err != nil {
		return fmt.Errorf("next event sequence: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO session_events (session_id, project, sequence, action_type, result_status, payload, timestamp)
		VALUES (?, ?, ?, ?, 'success', ?, ?)`,
		sessionID, s.project, seq, actionType, string(data), now,
	); err != nil {
		return fmt.Errorf("insert %s event for session %q: %w", actionType, sessionID, err)
	}
	return nil
}

// sessionExists reports whether sessionID is a row in this store's sessions
// table (the FK target of session_events).
func (s *Store) sessionExists(ctx context.Context, sessionID string) bool {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sessions WHERE id = ?`, sessionID).
		Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// buildFactFTSQuery converts a plain-text query into a safe FTS5 MATCH
// expression: terms are quoted (FTS special characters can't break the
// syntax), OR-joined (any-term recall, the default search mode). Returns ""
// for a blank query.
func buildFactFTSQuery(query string) (string, error) {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return "", nil
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		escaped[i] = `"` + strings.ReplaceAll(term, `"`, `""`) + `"`
	}
	return strings.Join(escaped, " OR "), nil
}
