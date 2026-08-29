// Package memory provides save and search over project-scoped observations.
package memory

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

const (
	defaultSearchLimit  = 20
	defaultContextLimit = 5
)

// Service provides save and search over project-scoped observations.
type Service struct {
	store     *store.Store
	projectID string
}

// SaveInput holds fields for a new or updated observation.
type SaveInput struct {
	Title     string
	Type      string
	Content   string
	Scope     string
	TopicKey  string
	SessionID string
}

// Session is a workspace session with optional title and summary.
type Session struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Directory string `json:"directory"`
	Title     string `json:"title,omitempty"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	Summary   string `json:"summary,omitempty"`
	Status    string `json:"status"`
}

// Observation is a stored memory entry.
type Observation struct {
	ID             int64  `json:"id"`
	SessionID      string `json:"session_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Content        string `json:"content"`
	Project        string `json:"project"`
	Scope          string `json:"scope"`
	TopicKey       string `json:"topic_key,omitempty"`
	NormalizedHash string `json:"normalized_hash,omitempty"`
	RevisionCount  int    `json:"revision_count"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// Status holds aggregate memory statistics.
type Status struct {
	ObservationCount int            `json:"observation_count"`
	ByType           map[string]int `json:"by_type"`
	ActiveSessions   int            `json:"active_sessions"`
	TotalSessions    int            `json:"total_sessions"`
	OldestCreated    string         `json:"oldest_created,omitempty"`
	NewestCreated    string         `json:"newest_created,omitempty"`
}

// New creates a memory service for the given store and project ID.
func New(st *store.Store, projectID string) *Service {
	return &Service{store: st, projectID: projectID}
}

// Save stores an observation, deduplicating by hash within 24h or upserting by topic_key.
func (s *Service) Save(ctx context.Context, in SaveInput) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("memory service not initialized")
	}
	if strings.TrimSpace(in.SessionID) == "" {
		return 0, errors.New("session_id is required")
	}
	if strings.TrimSpace(in.Title) == "" {
		return 0, errors.New("title is required")
	}
	if strings.TrimSpace(in.Content) == "" {
		return 0, errors.New("content is required")
	}
	if strings.TrimSpace(in.Type) == "" {
		return 0, errors.New("type is required")
	}
	if !IsValidType(in.Type) {
		return 0, fmt.Errorf("invalid type %q (allowed: standing, preference, convention, decision, architecture, bugfix, pattern, config, correction, discovery, learning, lesson, session_log)", in.Type)
	}

	hash := normalizedHash(in.Title, in.Content, in.Type)
	now := time.Now().UTC().Format(time.RFC3339)

	var existingID int64
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT id FROM observations
		WHERE normalized_hash = ? AND project = ? AND deleted_at IS NULL
		  AND datetime(created_at) > datetime('now', '-24 hours')
		ORDER BY id DESC LIMIT 1`,
		hash, s.projectID,
	).Scan(&existingID)
	if err == nil {
		return existingID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("dedup lookup: %w", err)
	}

	if in.TopicKey != "" {
		var topicID int64
		err := s.store.DB.QueryRowContext(ctx, `
			SELECT id FROM observations
			WHERE project = ? AND scope = ? AND topic_key = ? AND deleted_at IS NULL
			LIMIT 1`,
			s.projectID, in.Scope, in.TopicKey,
		).Scan(&topicID)
		if err == nil {
			_, err = s.store.DB.ExecContext(ctx, `
				UPDATE observations SET
					session_id = ?, type = ?, title = ?, content = ?,
					normalized_hash = ?, revision_count = revision_count + 1,
					updated_at = ?
				WHERE id = ?`,
				in.SessionID, in.Type, in.Title, in.Content, hash, now, topicID,
			)
			if err != nil {
				return 0, fmt.Errorf("topic_key upsert: %w", err)
			}
			return topicID, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("topic_key lookup: %w", err)
		}
	}

	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO observations (
			session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)`,
		in.SessionID, in.Type, in.Title, in.Content, s.projectID, in.Scope, nullString(in.TopicKey),
		hash, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert observation: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

// Search runs FTS over observations with matchMode "all" (AND) or "any" (OR).
func (s *Service) Search(ctx context.Context, query string, matchMode string, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	ftsQuery := buildFTSQuery(query, matchMode)
	if ftsQuery == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.normalized_hash, o.revision_count, o.created_at, o.updated_at
		FROM observations o
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?
		ORDER BY bm25(observations_fts)
		LIMIT ?`,
		ftsQuery, s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search observations: %w", err)
	}
	defer rows.Close()
	return scanObservations(rows)
}

// Recent returns stored observations, newest first, without FTS.
func (s *Service) Recent(ctx context.Context, limit int) ([]Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultSearchLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, session_id, type, title, content, project, scope,
		       topic_key, normalized_hash, revision_count, created_at, updated_at
		FROM observations
		WHERE deleted_at IS NULL AND project = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?`,
		s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent observations: %w", err)
	}
	defer rows.Close()
	return scanObservations(rows)
}

// Get returns a single observation by ID.
func (s *Service) Get(ctx context.Context, id int64) (Observation, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Observation{}, errors.New("memory service not initialized")
	}
	row := s.store.DB.QueryRowContext(ctx, `
		SELECT id, session_id, type, title, content, project, scope,
		       topic_key, normalized_hash, revision_count, created_at, updated_at
		FROM observations
		WHERE id = ? AND deleted_at IS NULL AND project = ?`,
		id, s.projectID,
	)
	var obs Observation
	var topicKey sql.NullString
	err := row.Scan(
		&obs.ID, &obs.SessionID, &obs.Type, &obs.Title, &obs.Content, &obs.Project, &obs.Scope,
		&topicKey, &obs.NormalizedHash, &obs.RevisionCount, &obs.CreatedAt, &obs.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Observation{}, fmt.Errorf("observation %d not found", id)
		}
		return Observation{}, fmt.Errorf("get observation: %w", err)
	}
	if topicKey.Valid {
		obs.TopicKey = topicKey.String
	}
	return obs, nil
}

// SessionStart creates a new active session for directory. title is the
// user/agent-facing name of the session, shown in the web dashboard session
// list (mem-sessions). An empty title leaves the cell unnamed.
func (s *Service) SessionStart(ctx context.Context, directory, title string) (string, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "", errors.New("memory service not initialized")
	}
	if strings.TrimSpace(directory) == "" {
		return "", errors.New("directory is required")
	}
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", fmt.Errorf("resolve directory: %w", err)
	}
	projectID, err := project.Resolve(absDir)
	if err != nil {
		return "", fmt.Errorf("resolve project: %w", err)
	}
	sessionID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)
	var titleNull sql.NullString
	if t := strings.TrimSpace(title); t != "" {
		titleNull = sql.NullString{String: t, Valid: true}
	}
	_, err = s.store.DB.ExecContext(ctx, `
		INSERT INTO sessions (id, project, directory, title, started_at, status)
		VALUES (?, ?, ?, ?, ?, 'active')`,
		sessionID, projectID, absDir, titleNull, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return sessionID, nil
}

// deriveSessionTitle extracts a human-readable title from a structured
// end-of-session summary — the first non-header line after the "## Goal"
// heading (the session's stated purpose). Falls back to the first non-header
// content line in the summary. Returns "" when nothing suitable is found.
func deriveSessionTitle(summary string) string {
	lines := strings.Split(summary, "\n")
	for i, l := range lines {
		if strings.EqualFold(strings.TrimSpace(l), "## Goal") {
			for _, next := range lines[i+1:] {
				t := strings.TrimSpace(next)
				if t == "" {
					continue
				}
				if strings.HasPrefix(t, "##") {
					break // reached the next section
				}
				if !strings.HasPrefix(t, "#") {
					return t
				}
			}
		}
	}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" && !strings.HasPrefix(t, "#") {
			return t
		}
	}
	return ""
}

// displayTitle returns the session's title for the web dashboard session list
// (mem-sessions). Priority:
//  1. the stored title (set by mem_session_start, mem_session_set_title, or
//     by SessionSummary/SessionEnd deriving it from the "## Goal" line)
//  2. the "## Goal" line of the summary (covers pre-title rows written by an
//     older binary)
//  3. the first non-header content line of the summary
//  4. a traceable short ID (first 8 chars of the session UUID) so the row is
//     never a vague placeholder — it lets you find the session in any store.
func displayTitle(sessionID, title, summary string) string {
	if t := strings.TrimSpace(title); t != "" {
		return t
	}
	if d := deriveSessionTitle(summary); d != "" {
		return d
	}
	id := strings.TrimSpace(sessionID)
	if len(id) >= 8 {
		return id[:8]
	}
	if id != "" {
		return id
	}
	return "(no id)"
}

// SessionSetTitle renames a session (idempotent).
func (s *Service) SessionSetTitle(ctx context.Context, sessionID, title string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE sessions SET title = ? WHERE id = ? AND project = ?`,
		title, sessionID, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("update session title: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("session title rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return nil
}

// SessionSummary stores an end-of-session summary, and when the session has no
// explicit title, derives one from the summary's "## Goal" line and persists it
// so the web dashboard (mem-sessions) shows a meaningful name.
func (s *Service) SessionSummary(ctx context.Context, sessionID, summary string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session_id is required")
	}
	if strings.TrimSpace(summary) == "" {
		return errors.New("summary is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	title := deriveSessionTitle(summary)
	if title != "" {
		_, err := s.store.DB.ExecContext(ctx, `
			UPDATE sessions
			SET summary = ?, ended_at = ?,
			    title = COALESCE(NULLIF(TRIM(title), ''), ?)
			WHERE id = ? AND project = ?`,
			summary, now, title, sessionID, s.projectID,
		)
		if err != nil {
			return fmt.Errorf("update session summary: %w", err)
		}
	} else {
		_, err := s.store.DB.ExecContext(ctx, `
			UPDATE sessions SET summary = ?, ended_at = ?
			WHERE id = ? AND project = ?`,
			summary, now, sessionID, s.projectID,
		)
		if err != nil {
			return fmt.Errorf("update session summary: %w", err)
		}
	}
	var n int
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	).Scan(&n); err != nil {
		return fmt.Errorf("session summary look-up: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return nil
}

// SessionEnd ends a session with an optional summary. When a summary is given
// and the session has no title, the title is derived from its "## Goal" line.
func (s *Service) SessionEnd(ctx context.Context, sessionID, summary string) error {
	if s == nil || s.store == nil || s.store.DB == nil {
		return errors.New("memory service not initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session_id is required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	var res sql.Result
	var err error
	if strings.TrimSpace(summary) != "" {
		title := deriveSessionTitle(summary)
		if title != "" {
			res, err = s.store.DB.ExecContext(ctx, `
				UPDATE sessions
				SET summary = ?, ended_at = ?, status = 'ended',
				    title = COALESCE(NULLIF(TRIM(title), ''), ?)
				WHERE id = ? AND project = ?`,
				summary, now, title, sessionID, s.projectID,
			)
		} else {
			res, err = s.store.DB.ExecContext(ctx, `
				UPDATE sessions SET summary = ?, ended_at = ?, status = 'ended'
				WHERE id = ? AND project = ?`,
				summary, now, sessionID, s.projectID,
			)
		}
	} else {
		res, err = s.store.DB.ExecContext(ctx, `
			UPDATE sessions SET ended_at = ?, status = 'ended'
			WHERE id = ? AND project = ?`,
			now, sessionID, s.projectID,
		)
	}
	if err != nil {
		return fmt.Errorf("end session: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("session end rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return nil
}

// RecentContext returns the most recent sessions that have a title or summary.
func (s *Service) RecentContext(ctx context.Context, limit int) ([]Session, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultContextLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, project, directory, title, started_at, ended_at, summary, status
		FROM sessions
		WHERE project = ? AND (
			(title IS NOT NULL AND TRIM(title) != '')
			OR (summary IS NOT NULL AND TRIM(summary) != '')
		)
		ORDER BY COALESCE(ended_at, started_at) DESC
		LIMIT ?`,
		s.projectID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("recent context: %w", err)
	}
	defer rows.Close()
	return scanSessions(rows)
}

// Status returns aggregate statistics for the project memory store.
func (s *Service) Status(ctx context.Context) (Status, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return Status{}, errors.New("memory service not initialized")
	}
	var st Status
	st.ByType = make(map[string]int)
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL`,
		s.projectID,
	).Scan(&st.ObservationCount)
	if err != nil {
		return Status{}, fmt.Errorf("count observations: %w", err)
	}
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sessions WHERE project = ? AND status = 'active'`,
		s.projectID,
	).Scan(&st.ActiveSessions)
	if err != nil {
		return Status{}, fmt.Errorf("count active sessions: %w", err)
	}
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM sessions WHERE project = ?`,
		s.projectID,
	).Scan(&st.TotalSessions)
	if err != nil {
		return Status{}, fmt.Errorf("count sessions: %w", err)
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL GROUP BY type`,
		s.projectID,
	)
	if err != nil {
		return Status{}, fmt.Errorf("count by type: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var typ string
		var count int
		if err := rows.Scan(&typ, &count); err != nil {
			return Status{}, fmt.Errorf("scan type count: %w", err)
		}
		st.ByType[typ] = count
	}
	if err := rows.Err(); err != nil {
		return Status{}, fmt.Errorf("iterate type counts: %w", err)
	}
	var oldest, newest sql.NullString
	err = s.store.DB.QueryRowContext(ctx, `
		SELECT MIN(created_at), MAX(created_at) FROM observations
		WHERE project = ? AND deleted_at IS NULL`,
		s.projectID,
	).Scan(&oldest, &newest)
	if err != nil {
		return Status{}, fmt.Errorf("created range: %w", err)
	}
	if oldest.Valid {
		st.OldestCreated = oldest.String
	}
	if newest.Valid {
		st.NewestCreated = newest.String
	}
	return st, nil
}

var validTypes = map[string]struct{}{
	"standing":     {},
	"preference":   {},
	"convention":   {},
	"decision":     {},
	"architecture": {},
	"bugfix":       {},
	"pattern":      {},
	"config":       {},
	"correction":   {},
	"discovery":    {},
	"learning":     {},
	"lesson":       {},
	"session_log":  {},
}

// IsValidType reports whether typ is one of the allowed observation types.
// Case-insensitive. Includes the skill taxonomy plus the MCP tool's advertised
// aliases (pattern, config, learning, lesson).
func IsValidType(typ string) bool {
	_, ok := validTypes[strings.ToLower(strings.TrimSpace(typ))]
	return ok
}

func normalizedHash(title, content, typ string) string {
	sum := sha256.Sum256([]byte(title + content + typ))
	return hex.EncodeToString(sum[:])
}

func buildFTSQuery(query string, matchMode string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}
	escaped := make([]string, len(terms))
	for i, term := range terms {
		term = strings.ReplaceAll(term, `"`, `""`)
		escaped[i] = `"` + term + `"`
	}
	sep := " OR "
	if matchMode == "all" {
		sep = " AND "
	}
	return strings.Join(escaped, sep)
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func scanObservations(rows *sql.Rows) ([]Observation, error) {
	var out []Observation
	for rows.Next() {
		var obs Observation
		var topicKey sql.NullString
		if err := rows.Scan(
			&obs.ID, &obs.SessionID, &obs.Type, &obs.Title, &obs.Content, &obs.Project, &obs.Scope,
			&topicKey, &obs.NormalizedHash, &obs.RevisionCount, &obs.CreatedAt, &obs.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan observation: %w", err)
		}
		if topicKey.Valid {
			obs.TopicKey = topicKey.String
		}
		out = append(out, obs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate observations: %w", err)
	}
	return out, nil
}

func scanSessions(rows *sql.Rows) ([]Session, error) {
	var out []Session
	for rows.Next() {
		var sess Session
		var title, endedAt, summary sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.Project, &sess.Directory, &title, &sess.StartedAt,
			&endedAt, &summary, &sess.Status,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if title.Valid {
			sess.Title = title.String
		}
		if endedAt.Valid {
			sess.EndedAt = endedAt.String
		}
		if summary.Valid {
			sess.Summary = summary.String
		}
		// Fill in a display title from the stored title, then the summary's
		// "## Goal" line, then the last 8 chars of the session id — so the row
		// in the web dashboard session list (mem-sessions) is always
		// identifiable and never a vague placeholder.
		sess.Title = displayTitle(sess.ID, sess.Title, sess.Summary)
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return out, nil
}
