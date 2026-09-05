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
	"regexp"
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
	Source    string
	// CapturePrompt, when true, best-effort links the most recent user prompt
	// recorded for SessionID (via the prompts table) to this observation. When
	// false, no prompt is linked. This mirrors Engram's capture_prompt switch.
	CapturePrompt bool
	// ProjectName, when non-empty, names the logical project this observation
	// belongs to. It is validated against project_aliases: if the name has been
	// recorded as an alias of a canonical project, the caller is expected to
	// write to the canonical name instead. See ProjectDrift.
	ProjectName string
	// ToolName is optional provenance for which agent tool produced the save
	// (e.g. "mem_save"). Empty leaves the column NULL.
	ToolName string
}

// PassiveInput is a raw block of text (assistant reply, Task output, etc.)
// from which the server extracts structured learnings. Extraction happens
// server-side so the agent does not have to do it.
type PassiveInput struct {
	Content   string `json:"content"`
	SessionID string `json:"session_id"`
	Source    string `json:"source,omitempty"`
	Project   string `json:"project,omitempty"`
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
	Source         string `json:"source,omitempty"`
	NormalizedHash string `json:"normalized_hash,omitempty"`
	RevisionCount  int    `json:"revision_count"`
	PromptID       *int64 `json:"prompt_id,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	// Pinned is a first-class, local-only "sticky" marker. When true the
	// observation sorts ahead of everything else in mem_context and boosts
	// mem_search ordering.
	Pinned         bool   `json:"pinned,omitempty"`
	DuplicateCount int    `json:"duplicate_count,omitempty"`
	LastSeenAt     string `json:"last_seen_at,omitempty"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
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

	source := in.Source
	if source == "" {
		source = "agent"
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
		s.BumpDuplicate(ctx, existingID)
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
			// last_seen_at mirrors Engram's last_seen_at semantics here: any
			// save that matches an existing topic_key refreshes "last seen".
			_, err = s.store.DB.ExecContext(ctx, `
				UPDATE observations SET
					source = ?,
					session_id = ?, type = ?, title = ?, content = ?,
					normalized_hash = ?, revision_count = revision_count + 1,
					last_seen_at = ?,
					updated_at = ?
				WHERE id = ?`,
				source, in.SessionID, in.Type, in.Title, in.Content, hash, now, now, topicID,
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

	var promptID sql.NullInt64
	if in.CapturePrompt {
		promptID = s.latestPromptForSession(ctx, in.SessionID)
	}
	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO observations (
			session_id, type, title, content, project, scope, topic_key,
			normalized_hash, revision_count, created_at, updated_at, source, prompt_id, tool_name
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?, ?)`,
		in.SessionID, in.Type, in.Title, in.Content, s.projectID, in.Scope, nullString(in.TopicKey),
		hash, now, now, source, nullableInt(promptID), nullString(in.ToolName),
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

// latestPromptForSession returns the most recent prompt recorded for the
// session, or a NULL value when none exists. Best-effort: any error yields NULL
// so that prompt linking never blocks the save.
func (s *Service) latestPromptForSession(ctx context.Context, sessionID string) sql.NullInt64 {
	if s == nil || s.store == nil || s.store.DB == nil || strings.TrimSpace(sessionID) == "" {
		return sql.NullInt64{}
	}
	var pid int64
	err := s.store.DB.QueryRowContext(ctx, `
		SELECT id FROM prompts
		WHERE session_id = ? AND project = ?
		ORDER BY id DESC LIMIT 1`,
		sessionID, s.projectID,
	).Scan(&pid)
	if err != nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: pid, Valid: true}
}

// nullableInt converts a NullInt64 to a value suitable for an ExecContext arg.
func nullableInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

// Search runs FTS over observations with matchMode "all" (AND) or "any" (OR).
// scope, when non-empty, additionally restricts results to that visibility
// scope (project|user|global).
func (s *Service) Search(ctx context.Context, query string, matchMode string, limit int) ([]Observation, error) {
	return s.SearchWithScope(ctx, query, matchMode, "", limit)
}

// SearchWithScope is like Search but accepts a scope filter ("" = any scope).
func (s *Service) SearchWithScope(ctx context.Context, query, matchMode, scope string, limit int) ([]Observation, error) {
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
	scopeClause := ""
	var args []any
	args = append(args, ftsQuery, s.projectID)
	scope = strings.TrimSpace(scope)
	if scope != "" {
		scopeClause = " AND o.scope = ?"
		args = append(args, scope)
	}
	args = append(args, limit)
	rows, err := s.store.DB.QueryContext(ctx, `
		SELECT o.id, o.session_id, o.type, o.title, o.content, o.project, o.scope,
		       o.topic_key, o.source, o.normalized_hash, o.revision_count, o.prompt_id, o.created_at, o.updated_at,
		       COALESCE(o.pinned, 0), COALESCE(o.duplicate_count, 0), o.last_seen_at, o.expires_at, o.tool_name
		FROM observations o
		INNER JOIN observations_fts ON observations_fts.rowid = o.id
		WHERE observations_fts MATCH ? AND o.deleted_at IS NULL AND o.project = ?`+scopeClause+`
		  AND (o.expires_at IS NULL OR o.expires_at = '' OR strftime('%s', o.expires_at) > strftime('%s', 'now'))
		ORDER BY COALESCE(o.pinned, 0) DESC, bm25(observations_fts)
		LIMIT ?`,
		args...,
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
		SELECT `+obsSelectCols+`
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
		SELECT `+obsSelectCols+`
		FROM observations
		WHERE id = ? AND deleted_at IS NULL AND project = ?`,
		id, s.projectID,
	)
	var obs Observation
	var topicKey sql.NullString
	var promptID sql.NullInt64
	var lastSeen, expires, toolName sql.NullString
	var pinned, dups int
	err := row.Scan(
		&obs.ID, &obs.SessionID, &obs.Type, &obs.Title, &obs.Content, &obs.Project, &obs.Scope,
		&topicKey, &obs.Source, &obs.NormalizedHash, &obs.RevisionCount, &promptID, &obs.CreatedAt, &obs.UpdatedAt,
		&pinned, &dups, &lastSeen, &expires, &toolName,
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
	if promptID.Valid {
		v := promptID.Int64
		obs.PromptID = &v
	}
	obs.Pinned = pinned == 1
	obs.DuplicateCount = dups
	if lastSeen.Valid {
		obs.LastSeenAt = lastSeen.String
	}
	if expires.Valid {
		obs.ExpiresAt = expires.String
	}
	if toolName.Valid {
		obs.ToolName = toolName.String
	}
	return obs, nil
}

// SessionStartByClientID registers a session under the caller-supplied ID
// (e.g. the agent's OpenCode session UUID) instead of generating a fresh UUID.
// Idempotent: if the session already exists it is not modified. Returns the
// session ID, the project ID, and whether it was newly created or already
// existed. The caller's ID is authoritative so that later saves keyed by the
// same ID land in the same row — this is what keeps plugin reloads, reconnects,
// and compaction recovery consistent.
func (s *Service) SessionStartByClientID(ctx context.Context, clientSessionID, directory, title string) (sessionID, projectID string, existed bool, err error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return "", "", false, errors.New("memory service not initialized")
	}
	clientSessionID = strings.TrimSpace(clientSessionID)
	if clientSessionID == "" {
		return "", "", false, errors.New("session id is required")
	}
	if strings.TrimSpace(directory) == "" {
		directory = "."
	}
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve directory: %w", err)
	}
	projectID, err = project.Resolve(absDir)
	if err != nil {
		return "", "", false, fmt.Errorf("resolve project: %w", err)
	}

	var existing int
	err = s.store.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE id = ? AND project = ?`,
		clientSessionID, projectID,
	).Scan(&existing)
	if err != nil {
		return "", "", false, fmt.Errorf("check session: %w", err)
	}
	if existing > 0 {
		// Best-effort backfill: older rows predating title support may be
		// missing a title or directory; fill blanks only, never overwrite.
		if title != "" {
			_, _ = s.store.DB.ExecContext(ctx, `
				UPDATE sessions
				SET title = COALESCE(NULLIF(TRIM(title), ''), ?),
				    directory = COALESCE(NULLIF(TRIM(directory), ''), ?)
				WHERE id = ? AND project = ?`,
				title, absDir, clientSessionID, projectID)
		}
		return clientSessionID, projectID, true, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	var titleNull sql.NullString
	if t := strings.TrimSpace(title); t != "" {
		titleNull = sql.NullString{String: t, Valid: true}
	}
	_, err = s.store.DB.ExecContext(ctx, `
		INSERT INTO sessions (id, project, directory, title, started_at, status)
		VALUES (?, ?, ?, ?, ?, 'active')`,
		clientSessionID, projectID, absDir, titleNull, now,
	)
	if err != nil {
		return "", "", false, fmt.Errorf("insert session: %w", err)
	}
	return clientSessionID, projectID, false, nil
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

// ── Prompts capture (Engram-aligned) ────────────────────────────────────────

// PromptInput holds a captured user prompt.
type PromptInput struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Project   string `json:"project,omitempty"`
}

// server-side so the agent does not have to do it.

// Prompt is a stored user prompt.
type Prompt struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Project   string `json:"project"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// minimum useful prompt length; shorter messages are usually "ok", "done", etc.
const minPromptLength = 11

// MaxPromptLength caps how much of a prompt is ever stored.
const MaxPromptLength = 2000

// MaxPassiveLength caps raw text considered by the passive extractor.
const MaxPassiveLength = 32000

// MinPassiveLength: below this, output rarely contains extractable learnings.
const MinPassiveLength = 50

// SavePrompt stores a captured user prompt, trimmed and bounded.
func (s *Service) SavePrompt(ctx context.Context, in PromptInput) (int64, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("memory service not initialized")
	}
	sessionID := strings.TrimSpace(in.SessionID)
	if sessionID == "" {
		return 0, errors.New("session_id is required")
	}
	content := strings.TrimSpace(in.Content)
	if len(content) < minPromptLength {
		return 0, ErrPromptTooSmall
	}
	if len(content) > MaxPromptLength {
		content = content[:MaxPromptLength]
	}
	var valid int
	if err := s.store.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sessions WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	).Scan(&valid); err != nil {
		return 0, fmt.Errorf("verify session: %w", err)
	}
	if valid == 0 {
		return 0, fmt.Errorf("session %s not found", sessionID)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO prompts (session_id, project, content, created_at)
		VALUES (?, ?, ?, ?)`,
		sessionID, s.projectID, content, now,
	)
	if err != nil {
		return 0, fmt.Errorf("insert prompt: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

// ErrPromptTooSmall is returned by SavePrompt when the prompt is shorter than
// the minimum useful length (usually an acknowledgement like "ok"/"done").
var ErrPromptTooSmall = errors.New("prompt too small to capture")

// errNoLearnings is returned by CapturePassive when the text contains no
// extractable learnings. It is not a failure — it means "nothing to save".
var errNoLearnings = errors.New("no learnings extracted")

// CapturePassiveResult reports what the passive extractor found and saved.
type CapturePassiveResult struct {
	Saved     int     `json:"saved"`
	Skipped   int     `json:"skipped"`
	Exhausted bool    `json:"exhausted"`
	IDs       []int64 `json:"ids,omitempty"`
}

// CapturePassive extracts structured learnings from a block of free text and
// stores them as observations sourced "passive". It is deliberately dumb:
// it recognises a handful of well-known structures ("Key Learnings:" sections,
// bullet lists after a "## Learnings" heading, "Lesson:" / "Discovery:"
// lines). Anything it cannot confidently attribute is left out rather than
// stored, to avoid polluting the memory store with noise.
//
// It is idempotent: the Save() path deduplicates by (title+content+type) hash
// within 24h, so repeated captures of the same output do not duplicate rows.
func (s *Service) CapturePassive(ctx context.Context, in PassiveInput) (CapturePassiveResult, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return CapturePassiveResult{}, errors.New("memory service not initialized")
	}
	if strings.TrimSpace(in.SessionID) == "" {
		return CapturePassiveResult{}, errors.New("session_id is required")
	}
	source := in.Source
	if source == "" {
		source = "passive"
	}
	text := in.Content
	if len(text) > MaxPassiveLength {
		text = text[:MaxPassiveLength]
	}
	if len(strings.TrimSpace(text)) < MinPassiveLength {
		return CapturePassiveResult{Exhausted: true}, nil
	}

	rawItems := extractLearnings(text)
	if len(rawItems) == 0 {
		return CapturePassiveResult{Exhausted: true}, nil
	}

	result := CapturePassiveResult{Skipped: len(rawItems)}
	seen := make(map[string]bool, len(rawItems))
	for _, item := range rawItems {
		title, typ := shapePassiveItem(item)
		if title == "" || !IsValidType(typ) {
			result.Skipped++
			continue
		}
		title = truncateTitle(title, 120)
		id, err := s.Save(ctx, SaveInput{
			Title:     title,
			Type:      typ,
			Content:   shapePassiveContent(item, title, typ),
			Scope:     "project",
			SessionID: in.SessionID,
			TopicKey:  passiveTopicKey(typ, title),
			Source:    source,
		})
		if err != nil {
			if errors.Is(err, errNoUseful) {
				result.Skipped++
				continue
			}
			return result, fmt.Errorf("save passive observation: %w", err)
		}
		if seen[title+"|"+typ] {
			result.Skipped++
			continue
		}
		seen[title+"|"+typ] = true
		result.Saved++
		result.Skipped--
		result.IDs = append(result.IDs, id)
	}
	return result, nil
}

var errNoUseful = errors.New("no useful content")

// PassiveItem is a raw extracted learning.
type PassiveItem struct {
	Text    string
	Heading string // section heading if the item came from one (e.g. "Key Learnings")
}

// learnHeadingRe matches markdown headings that signal a learnings section.
var learnHeadingRe = regexp.MustCompile(`(?im)^\s*#{1,6}\s*(key learnings|learnings|learned|discovery(?:\s*|\s+notes)?|what we learned)\b`)

// bulletRe matches a bulleted or numbered list item.
var bulletRe = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(?P<body>.+)$`)

// extractLearnings pulls out candidate learnings from free text. It prefers
// explicit structured sections and falls back to "Lesson:" / "Discovery:" /
// "Finding:" inline labels. It never invents content beyond what is present.
func extractLearnings(text string) []PassiveItem {
	var out []PassiveItem

	// 1) Sectioned: a "## Key Learnings:" style heading followed by bullets.
	lines := strings.Split(text, "\n")
	heading := ""
	inLearnings := false
	var section []PassiveItem
	flush := func() {
		if inLearnings && len(section) > 0 {
			out = append(out, section...)
		}
		section = section[:0]
		inLearnings = false
	}
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if learnHeadingRe.MatchString(line) {
			if len(section) > 0 && inLearnings {
				out = append(out, section...)
				section = section[:0]
			}
			inLearnings = true
			heading = strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(line), "#"), "#"))
			continue
		}
		if !inLearnings {
			continue
		}
		if strings.HasPrefix(line, "#") {
			// new heading while in a learnings section — stop capturing
			flush()
			continue
		}
		body := strings.TrimSpace(line)
		if body == "" {
			continue
		}
		if bm := bulletRe.FindStringSubmatch(line); bm != nil {
			body = strings.TrimSpace(bm[1])
		}
		if len(body) < 8 {
			continue
		}
		section = append(section, PassiveItem{Text: body, Heading: heading})
	}
	flush()

	if len(out) > 0 {
		return out
	}

	// 2) Inline labelled lines: "Lesson: ...", "Discovery: ...".
	labelRe := regexp.MustCompile(`(?im)^\s*(?:lesson|discovery|finding|note)[:]\s*(?P<body>.+)$`)
	for _, raw := range lines {
		if m := labelRe.FindStringSubmatch(raw); m != nil {
			body := strings.TrimSpace(m[1])
			if len(body) >= 8 {
				out = append(out, PassiveItem{Text: body, Heading: "label"})
			}
		}
	}
	return out
}

// shapePassiveItem classifies an extracted item by keyword and returns a
// stable title for the observation.
func shapePassiveItem(item PassiveItem) (title, typ string) {
	text := strings.ToLower(item.Text)
	switch {
	case strings.Contains(text, "decision") || strings.Contains(text, "chose") || strings.Contains(text, "chosen"):
		typ = "decision"
	case strings.Contains(text, "bug") || strings.Contains(text, "fixed") || strings.Contains(text, "fix for"):
		typ = "bugfix"
	case strings.Contains(text, "architecture") || strings.Contains(text, "structure of") || strings.Contains(text, "layout"):
		typ = "architecture"
	case strings.Contains(text, "pattern") || strings.Contains(text, "convention") || strings.Contains(text, "idiom"):
		typ = "pattern"
	case strings.Contains(text, "config") || strings.Contains(text, "setup") || strings.Contains(text, "env") || strings.Contains(text, "port"):
		typ = "config"
	case strings.Contains(text, "preference") || strings.Contains(text, "user likes") || strings.Contains(text, "prefer"):
		typ = "preference"
	case strings.Contains(text, "discovery") || strings.Contains(text, "found that") || strings.Contains(text, "turns out") || strings.Contains(text, "gotcha"):
		typ = "discovery"
	case strings.Contains(text, "learned") || strings.Contains(text, "note on") || strings.Contains(text, "realized"):
		typ = "learning"
	case strings.Contains(text, "lesson"):
		typ = "lesson"
	default:
		typ = "learning"
	}
	title = collapseWhitespace(item.Text)
	if len(title) > 120 {
		title = title[:120]
	}
	return strings.TrimSpace(title), typ
}

// shapePassiveContent builds a structured What/Why/Where/Learned-shaped body
// from an extracted item. For passive captures "What" is the item verbatim
// and nothing is invented.
func shapePassiveContent(item PassiveItem, title, typ string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "**What**: %s\n", collapseWhitespace(item.Text))
	if item.Heading != "" && item.Heading != "label" {
		fmt.Fprintf(&b, "**Section**: %s\n", collapseWhitespace(item.Heading))
	}
	fmt.Fprintf(&b, "**Type**: %s\n", typ)
	return b.String()
}

var wsRe = regexp.MustCompile(`\s+`)

func collapseWhitespace(s string) string {
	return strings.TrimSpace(wsRe.ReplaceAllString(s, " "))
}

func truncateTitle(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimRight(s[:n], " ,;") + "…"
}

func passiveTopicKey(typ, title string) string {
	seg := strings.ToLower(strings.TrimSpace(title))
	seg = strings.TrimRight(strings.ReplaceAll(seg, " ", "-"), "-")
	if len(seg) > 60 {
		seg = seg[:60]
	}
	if seg == "" {
		seg = "untitled"
	}
	return "passive/" + typ + "/" + seg
}

// ── Project migration (Engram-aligned) ─────────────────────────────────────

// MigrateProjects renames rows tagged oldProject to newProject within the
// same project store. (Cross-store file moves are handled by the facade.)
// One-time, idempotent; a trailing project_migrations row is written so
// subsequent calls are effectively no-ops.
//
// Called when `skillgrid setup` detects a project id change (e.g. the dir is
// renamed, the git remote origin's repo base name differs, or the user is
// migrating from a legacy naming scheme).
func (s *Service) MigrateProjects(ctx context.Context, oldProject, newProject string) (int, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return 0, errors.New("memory service not initialized")
	}
	oldProject = strings.TrimSpace(oldProject)
	newProject = strings.TrimSpace(newProject)
	if oldProject == "" || newProject == "" {
		return 0, errors.New("old and new project are required")
	}
	if oldProject == newProject {
		return 0, nil
	}
	now := time.Now().UTC().Format(time.RFC3339)

	var moved int
	for _, table := range []string{"observations", "sessions", "web_cache", "prompts"} {
		res, err := s.store.DB.ExecContext(ctx, `
			UPDATE `+table+` SET project = ? WHERE project = ?`,
			newProject, oldProject)
		if err != nil {
			// table may not exist in older schemas — ignore, try next
			if strings.Contains(err.Error(), "no such table") {
				continue
			}
			return moved, fmt.Errorf("migrate %s: %w", table, err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return moved, fmt.Errorf("rows affected %s: %w", table, err)
		}
		moved += int(n)
	}

	if _, err := s.store.DB.ExecContext(ctx, `
		INSERT INTO project_migrations (old_project, new_project, migrated_at)
		VALUES (?, ?, ?)`,
		oldProject, newProject, now); err != nil {
		return moved, fmt.Errorf("record migration: %w", err)
	}
	return moved, nil
}

// LastObservationAt returns the newest observation timestamp for the project,
// or zero if none exist. Used by the plugin's save-nudge logic.
func (s *Service) LastObservationAt(ctx context.Context) (time.Time, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return time.Time{}, errors.New("memory service not initialized")
	}
	var ts string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT created_at FROM observations WHERE project = ? AND deleted_at IS NULL
		 ORDER BY created_at DESC, id DESC LIMIT 1`,
		s.projectID,
	).Scan(&ts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("last observation at: %w", err)
	}
	return time.Parse(time.RFC3339, ts)
}

// SessionStartedAt returns the started_at of a session, or a zero time.
func (s *Service) SessionStartedAt(ctx context.Context, sessionID string) (time.Time, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return time.Time{}, errors.New("memory service not initialized")
	}
	var ts string
	err := s.store.DB.QueryRowContext(ctx,
		`SELECT started_at FROM sessions WHERE id = ? AND project = ?`,
		sessionID, s.projectID,
	).Scan(&ts)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("session started_at: %w", err)
	}
	return time.Parse(time.RFC3339, ts)
}

// ── Compaction context (Engram-aligned) ─────────────────────────────────────

// CompactionContext assembles what a compaction recovery needs: the current
// session's title + summary if any, plus the most recent observations for the
// project. The plugin injects this into the compaction prompt.
type CompactionContext struct {
	SessionID    string   `json:"session_id"`
	Title        string   `json:"title,omitempty"`
	Summary      string   `json:"summary,omitempty"`
	Observations []string `json:"observations,omitempty"`
	GeneratedAt  string   `json:"generated_at"`
}

// CompactionContext builds a compact, session-scoped context block for the
// compaction prompt. Observations are capped by limit (default 5) and trimmed
// to one line each — full content is not needed at compaction time.
func (s *Service) CompactionContext(ctx context.Context, sessionID string, limit int) (CompactionContext, error) {
	if s == nil || s.store == nil || s.store.DB == nil {
		return CompactionContext{}, errors.New("memory service not initialized")
	}
	if limit <= 0 {
		limit = 5
	}
	out := CompactionContext{
		SessionID:   sessionID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if sessionID != "" {
		var title, summary sql.NullString
		err := s.store.DB.QueryRowContext(ctx, `
			SELECT title, summary FROM sessions WHERE id = ? AND project = ?`,
			sessionID, s.projectID,
		).Scan(&title, &summary)
		if err == nil {
			if title.Valid {
				out.Title = title.String
			}
			if summary.Valid {
				out.Summary = summary.String
			}
		}
	}
	obs, err := s.Recent(ctx, limit)
	if err != nil {
		return out, nil
	}
	for _, o := range obs {
		line := o.Title
		if first := collapseWhitespace(o.Content); first != "" {
			line = o.Title + " — " + collapseWhitespace(o.Content)
		}
		if len(line) > 300 {
			line = line[:300] + "…"
		}
		out.Observations = append(out.Observations, line)
	}
	return out, nil
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

// obsSelectCols is the shared column list every observation SELECT uses, so that
// the SELECT and the Scan in scanObservations cannot drift apart.
const obsSelectCols = `
	id, session_id, type, title, content, project, scope,
	topic_key, source, normalized_hash, revision_count, prompt_id, created_at, updated_at,
	COALESCE(pinned, 0), COALESCE(duplicate_count, 0), last_seen_at, expires_at, tool_name`

func scanObservations(rows *sql.Rows) ([]Observation, error) {
	var out []Observation
	for rows.Next() {
		var obs Observation
		var topicKey sql.NullString
		var promptID sql.NullInt64
		var lastSeen, expires, toolName sql.NullString
		var pinned, dups int
		if err := rows.Scan(
			&obs.ID, &obs.SessionID, &obs.Type, &obs.Title, &obs.Content, &obs.Project, &obs.Scope,
			&topicKey, &obs.Source, &obs.NormalizedHash, &obs.RevisionCount, &promptID, &obs.CreatedAt, &obs.UpdatedAt,
			&pinned, &dups, &lastSeen, &expires, &toolName,
		); err != nil {
			return nil, fmt.Errorf("scan observation: %w", err)
		}
		if topicKey.Valid {
			obs.TopicKey = topicKey.String
		}
		if promptID.Valid {
			v := promptID.Int64
			obs.PromptID = &v
		}
		obs.Pinned = pinned == 1
		obs.DuplicateCount = dups
		if lastSeen.Valid {
			obs.LastSeenAt = lastSeen.String
		}
		if expires.Valid {
			obs.ExpiresAt = expires.String
		}
		if toolName.Valid {
			obs.ToolName = toolName.String
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
