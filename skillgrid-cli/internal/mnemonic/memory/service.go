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

// Session is a workspace session with optional summary.
type Session struct {
	ID        string
	Project   string
	Directory string
	StartedAt string
	EndedAt   string
	Summary   string
	Status    string
}

// Observation is a stored memory entry.
type Observation struct {
	ID             int64
	SessionID      string
	Type           string
	Title          string
	Content        string
	Project        string
	Scope          string
	TopicKey       string
	NormalizedHash string
	RevisionCount  int
	CreatedAt      string
	UpdatedAt      string
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
	if strings.TrimSpace(in.Type) == "" {
		return 0, errors.New("type is required")
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

// SessionStart creates a new active session for directory.
func (s *Service) SessionStart(ctx context.Context, directory string) (string, error) {
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
	_, err = s.store.DB.ExecContext(ctx, `
		INSERT INTO sessions (id, project, directory, started_at, status)
		VALUES (?, ?, ?, ?, 'active')`,
		sessionID, projectID, absDir, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return sessionID, nil
}

// SessionSummary stores an end-of-session summary.
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
	res, err := s.store.DB.ExecContext(ctx, `
		UPDATE sessions SET summary = ?, ended_at = ?
		WHERE id = ? AND project = ?`,
		summary, now, sessionID, s.projectID,
	)
	if err != nil {
		return fmt.Errorf("update session summary: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("session summary rows affected: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("session %s not found", sessionID)
	}
	return nil
}

// SessionEnd ends a session with an optional summary.
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
		res, err = s.store.DB.ExecContext(ctx, `
			UPDATE sessions SET summary = ?, ended_at = ?, status = 'ended'
			WHERE id = ? AND project = ?`,
			summary, now, sessionID, s.projectID,
		)
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

// RecentContext returns the most recent sessions that have summaries.
func (s *Service) RecentContext(ctx context.Context, limit int) ([]Session, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return nil, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = defaultContextLimit
	}
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT id, project, directory, started_at, ended_at, summary, status
		FROM sessions
		WHERE project = ? AND summary IS NOT NULL AND TRIM(summary) != ''
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
		var endedAt, summary sql.NullString
		if err := rows.Scan(
			&sess.ID, &sess.Project, &sess.Directory, &sess.StartedAt,
			&endedAt, &summary, &sess.Status,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if endedAt.Valid {
			sess.EndedAt = endedAt.String
		}
		if summary.Valid {
			sess.Summary = summary.String
		}
		out = append(out, sess)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}
	return out, nil
}
