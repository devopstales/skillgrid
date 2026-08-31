// Package service is the shared facade over memory, codeindex, webcache, and search.
package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/project"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/search"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/webcache"
)

// Service is the shared facade over memory, codeindex, webcache, and search.
type Service struct {
	dataDir string
}

// New creates a service using dataDir for per-project SQLite stores.
func New(dataDir string) *Service {
	return &Service{dataDir: dataDir}
}

// DefaultDataDir returns the mnemonic data directory from env or ~/.skillgrid/mnemonic.
func DefaultDataDir() (string, error) {
	if v := os.Getenv("SKILLGRID_MNEMONIC_DATA_DIR"); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".skillgrid", "mnemonic"), nil
}

type projectHandle struct {
	store     *store.Store
	projectID string
	memory    *memory.Service
	web       *webcache.Service
}

func (s *Service) openProject(projectID, configRoot string) (*projectHandle, func(), error) {
	if s == nil {
		return nil, nil, fmt.Errorf("service not initialized")
	}
	st, err := store.Open(s.dataDir, projectID)
	if err != nil {
		return nil, nil, err
	}
	cfg := config.Load(configRoot)
	h := &projectHandle{
		store:     st,
		projectID: projectID,
		memory:    memory.New(st, projectID),
		web:       webcache.New(st, projectID, cfg.WebCache),
	}
	return h, func() { st.Close() }, nil
}

func (s *Service) openProjectForDirectory(directory string) (*projectHandle, func(), error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve directory: %w", err)
	}
	projectID, err := project.Resolve(absDir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve project: %w", err)
	}
	return s.openProject(projectID, absDir)
}

func (s *Service) openProjectFromCWD() (*projectHandle, func(), error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, err
	}
	return s.openProjectForDirectory(cwd)
}

// ListProjects returns the project IDs with a store in dataDir, sorted.
func (s *Service) ListProjects() ([]string, error) {
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sqlite") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".sqlite")
		if strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// ObservationsRecent returns stored observations, newest first.
func (s *Service) ObservationsRecent(ctx context.Context, projectID string, limit int) ([]memory.Observation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.Recent(ctx, limit)
}

// ResolveProject returns the project ID for directory.
func (s *Service) ResolveProject(directory string) (string, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return project.Resolve(absDir)
}

// SessionStart creates a workspace session for directory. title is the
// session name shown in the web dashboard (mem-sessions).
func (s *Service) SessionStart(ctx context.Context, directory, title string) (sessionID, projectID string, err error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	sessionID, err = h.memory.SessionStart(ctx, directory, title)
	if err != nil {
		return "", "", err
	}
	return sessionID, h.projectID, nil
}

// SessionSetTitle renames a session so the dashboard session list shows it.
func (s *Service) SessionSetTitle(ctx context.Context, projectID, sessionID, title string) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.SessionSetTitle(ctx, sessionID, title)
}

// SessionEnd ends a session with optional summary.
func (s *Service) SessionEnd(ctx context.Context, projectID, sessionID, summary string) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.SessionEnd(ctx, sessionID, summary)
}

// SessionSummary stores an end-of-session summary.
func (s *Service) SessionSummary(ctx context.Context, projectID, sessionID, summary string) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.SessionSummary(ctx, sessionID, summary)
}

// SessionStartByClientID registers a session under the caller's ID (idempotent).
func (s *Service) SessionStartByClientID(ctx context.Context, sessionID, directory, title string) (id, projectID string, existed bool, err error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return "", "", false, err
	}
	defer cleanup()
	id, projID, existed, err := h.memory.SessionStartByClientID(ctx, sessionID, directory, title)
	if err != nil {
		return "", "", false, err
	}
	return id, projID, existed, nil
}

// PromptInput is a captured user prompt.
type PromptInput = memory.PromptInput

// SavePrompt stores a captured user prompt.
func (s *Service) SavePrompt(ctx context.Context, projectID string, in PromptInput) (int64, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return 0, err
	}
	defer cleanup()
	return h.memory.SavePrompt(ctx, in)
}

// PassiveInput is free text for server-side learnings extraction.
type PassiveInput = memory.PassiveInput

// PassiveResult reports what the passive extractor found.
type PassiveResult = memory.CapturePassiveResult

// CapturePassive extracts learnings from raw text and persists them.
func (s *Service) CapturePassive(ctx context.Context, projectID string, in PassiveInput) (PassiveResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return PassiveResult{}, err
	}
	defer cleanup()
	return h.memory.CapturePassive(ctx, in)
}

// CompactionContext assembles session-scoped context for the compaction prompt.
type CompactionContext = memory.CompactionContext

// ContextForCompaction returns the compaction context.
func (s *Service) ContextForCompaction(ctx context.Context, projectID, sessionID string, limit int) (CompactionContext, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return CompactionContext{}, err
	}
	defer cleanup()
	return h.memory.CompactionContext(ctx, sessionID, limit)
}

// MigrateProjects rolls data recorded under oldProject into the newProject
// store. In Mnemonic each project has its own SQLite file, so the data lives
// in <oldProject>.sqlite while new writes go to <newProject>.sqlite. We open
// the old store, ATTACH the new one, copy every row tagged oldProject across
// (observations, sessions, web_cache, prompts, files), re-tag it as
// newProject, and record the migration for idempotency.
//
// Idempotent: if no rows tagged oldProject remain, nothing is copied and we
// just ensure the record exists. If the old store file is missing, this is a
// clean no-op (0 moved, nil error).
//
// Called from the agent plugin on first run when the project id it computed
// differs from one previously recorded in the data dir.
func (s *Service) MigrateProjects(ctx context.Context, oldProject, newProject string) (int, error) {
	oldProject = strings.TrimSpace(oldProject)
	newProject = strings.TrimSpace(newProject)
	if oldProject == "" || newProject == "" || oldProject == newProject {
		return 0, nil
	}

	oldPath := filepath.Join(s.dataDir, oldProject+".sqlite")
	if _, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return 0, nil // nothing to migrate
		}
		return 0, err
	}
	// Ensure the destination store exists (and is migrated) before we attach.
	newStore, err := store.Open(s.dataDir, newProject)
	if err != nil {
		return 0, fmt.Errorf("open destination store: %w", err)
	}
	defer newStore.Close()
	oldStore, err := store.Open(s.dataDir, oldProject)
	if err != nil {
		return 0, fmt.Errorf("open source store: %w", err)
	}
	defer oldStore.Close()

	now := time.Now().UTC().Format(time.RFC3339)
	attachSQL := "ATTACH DATABASE ? AS newdb"
	res, err := oldStore.DB.ExecContext(ctx, attachSQL, newStore.Path())
	if err != nil {
		return 0, fmt.Errorf("attach new store: %w", err)
	}
	defer func() {
		_, _ = oldStore.DB.Exec("DETACH DATABASE newdb")
	}()
	_ = res

	t := oldStore.DB

	var total int
	copied := []string{}
	for _, table := range []string{"observations", "sessions", "web_cache", "prompts", "files"} {
		var n int
		err := t.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM `+table+` WHERE project = ? AND deleted_at IS NULL`,
			oldProject).Scan(&n)
		if err != nil || n == 0 {
			continue
		}
		// INSERT OR IGNORE so re-runs do not fail on PK collisions.
		res, err := t.ExecContext(ctx, `
			INSERT OR IGNORE INTO newdb.`+table+`
			SELECT * FROM `+table+` WHERE project = ? AND (deleted_at IS NULL OR deleted_at = '')`,
			oldProject)
		if err != nil {
			if strings.Contains(err.Error(), "no such table") || strings.Contains(err.Error(), "duplicate column") || strings.Contains(err.Error(), "datatype mismatch") {
				continue
			}
			return total, fmt.Errorf("copy %s: %w", table, err)
		}
		affected, _ := res.RowsAffected()
		total += int(affected)
		copied = append(copied, fmt.Sprintf("%s=%d", table, affected))
		// Re-tag copied rows in the destination.
		if _, err := t.ExecContext(ctx, `
			UPDATE newdb.`+table+` SET project = ? WHERE project = ?`,
			newProject, oldProject); err != nil {
			if !strings.Contains(err.Error(), "no such table") {
				return total, fmt.Errorf("retag %s: %w", table, err)
			}
		}
	}

	// Record the migration in the destination for idempotency.
	if _, err := newStore.DB.ExecContext(ctx, `
		INSERT INTO project_migrations (old_project, new_project, migrated_at)
		VALUES (?, ?, ?)`,
		oldProject, newProject, now); err != nil && !strings.Contains(err.Error(), "no such table") {
		// ignore — best effort
	}

	return total, nil
}

// LastObservationAt returns the newest observation time for the project, if any.
func (s *Service) LastObservationAt(ctx context.Context, projectID string) (time.Time, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return time.Time{}, err
	}
	defer cleanup()
	return h.memory.LastObservationAt(ctx)
}

// RecordRelation stores a semantic link between two observations in the
// given project (mem_judge / mem_compare).
func (s *Service) RecordRelation(ctx context.Context, projectID string, srcID, dstID int64, relation, reason string, confidence *float64) (memory.Relation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return memory.Relation{}, err
	}
	defer cleanup()
	return h.memory.RecordRelation(ctx, srcID, dstID, relation, reason, confidence)
}

// RemoveRelation clears a live link between two observations (mem_judge
// not_conflict verdict).
func (s *Service) RemoveRelation(ctx context.Context, projectID string, srcID, dstID int64, relation string) (bool, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return false, err
	}
	defer cleanup()
	return h.memory.RemoveRelation(ctx, srcID, dstID, relation)
}

// RelationsOf returns every live relation touching observation id.
func (s *Service) RelationsOf(ctx context.Context, projectID string, id int64) ([]memory.Relation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.RelationsOf(ctx, id)
}

// RelationsBetween returns the live links between two specific observations.
func (s *Service) RelationsBetween(ctx context.Context, projectID string, srcID, dstID int64) ([]memory.Relation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.RelationsBetween(ctx, srcID, dstID)
}

// ProjectDrift mirrors the memory package's drift report, for API consumers.
type ProjectDrift = memory.ProjectDrift

// CheckProjectDrift returns a drift report when projectName is a known alias
// of a canonical project. It probes the store that most likely holds the
// alias row (the canonical project's store, then the current CWD store, then
// every known store) and does not modify state.
func (s *Service) CheckProjectDrift(ctx context.Context, projectName string) (*ProjectDrift, error) {
	name := strings.TrimSpace(projectName)
	if name == "" {
		return nil, nil
	}
	probe := func(storeID string) (*ProjectDrift, bool) {
		if storeID == "" {
			return nil, false
		}
		h, cleanup, err := s.openProject(storeID, ".")
		if err != nil {
			return nil, false
		}
		d, err := h.memory.CheckProjectDrift(ctx, name)
		cleanup()
		if err != nil {
			return nil, false
		}
		if d != nil {
			return d, true
		}
		return nil, false
	}
	// 1) The canonical project (if the alias points at one of our stores).
	if canonical := s.canonicalForAlias(ctx, name); canonical != "" {
		if d, ok := probe(canonical); ok {
			return d, nil
		}
	}
	// 2) The explicitly-named store (it may hold its own alias row).
	if n := storeIDFor(name); n != "" {
		if d, ok := probe(n); ok {
			return d, nil
		}
	}
	// 3) Current CWD store.
	if cwd, err := os.Getwd(); err == nil {
		if pid, err := s.ResolveProject(cwd); err == nil {
			if d, ok := probe(pid); ok {
				return d, nil
			}
		}
	}
	return nil, nil
}

// storeIDFor normalizes a project name into its store ID (best-effort — no IO).
func storeIDFor(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return ""
	}
	return n
}

// canonicalForAlias scans existing stores for a project_aliases row naming
// projectName as an alias, and returns the canonical name. Best-effort.
func (s *Service) canonicalForAlias(ctx context.Context, alias string) string {
	projects, err := s.ListProjects()
	if err != nil {
		return ""
	}
	for _, p := range projects {
		h, cleanup, err := s.openProject(p, ".")
		if err != nil {
			continue
		}
		var canonical string
		err = h.store.DB.QueryRowContext(ctx, `
			SELECT canonical FROM project_aliases WHERE alias = ? LIMIT 1`, alias,
		).Scan(&canonical)
		cleanup()
		if err == nil && canonical != "" {
			return canonical
		}
	}
	return ""
}

// MergeProjects consolidates data recorded under the source project name into
// the canonical project, then records a project alias. Returns the number of
// rows moved and the canonical name.
//
// Behaviour:
//   - If source == canonical or either is blank, no-op (0, canonical).
//   - If the source store file does not exist, no-op (0, canonical) — but the
//     alias is still recorded so future writes to the source name land in the
//     canonical store.
//   - Otherwise: copies observations/sessions/web_cache/prompts/files rows
//     tagged source into the canonical store (INSERT OR IGNORE so re-runs do
//     not fail on PK collisions), re-tags them as canonical, and records the
//     migration + alias (idempotent via unique primary key).
func (s *Service) MergeProjects(ctx context.Context, source, canonical string) (int, string, error) {
	source = strings.TrimSpace(source)
	canonical = strings.TrimSpace(canonical)
	if source == "" || canonical == "" || source == canonical {
		return 0, canonical, nil
	}

	// Record the alias first so that future resolution always lands in the
	// canonical store regardless of whether the copy below succeeds.
	if err := s.recordProjectAlias(ctx, canonical, source); err != nil {
		return 0, canonical, fmt.Errorf("record alias: %w", err)
	}

	// No data to move if the source store file is missing.
	srcPath := filepath.Join(s.dataDir, source+".sqlite")
	if _, err := os.Stat(srcPath); err != nil {
		if os.IsNotExist(err) {
			return 0, canonical, nil
		}
		return 0, canonical, err
	}

	moved, err := s.MigrateProjects(ctx, source, canonical)
	if err != nil {
		return 0, canonical, err
	}
	return moved, canonical, nil
}

// recordProjectAlias marks source as an alias of canonical in the canonical
// store. Idempotent: re-merge keeps the first merge time.
func (s *Service) recordProjectAlias(ctx context.Context, canonical, source string) error {
	newStore, err := store.Open(s.dataDir, canonical)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such table") {
			return nil // no canonical store yet — alias is implicit
		}
		return err
	}
	defer newStore.Close()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = newStore.DB.ExecContext(ctx, `
		INSERT INTO project_aliases (alias, canonical, merged_at)
		VALUES (?, ?, ?)
		ON CONFLICT(alias) DO NOTHING`,
		source, canonical, now,
	)
	return err
}

// SessionStartedAt returns the started_at of a session (zero if missing).
func (s *Service) SessionStartedAt(ctx context.Context, projectID, sessionID string) (time.Time, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return time.Time{}, err
	}
	defer cleanup()
	return h.memory.SessionStartedAt(ctx, sessionID)
}

// ListProjectsForMigration returns all project ids with a store file.
func (s *Service) ListProjectsForMigration() ([]string, error) {
	return s.ListProjects()
}

// SaveObservationInput holds fields for saving an observation (HTTP + MCP).
type SaveObservationInput struct {
	Title     string `json:"title"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	Scope     string `json:"scope"`
	TopicKey  string `json:"topic_key"`
	SessionID string `json:"session_id"`
	// CapturePrompt, when true, best-effort links the session's most recent
	// user prompt to the observation (mem_save capture_prompt).
	CapturePrompt bool `json:"capture_prompt"`
	// ProjectName, when non-empty, names the logical project for the save. It
	// is validated against project_aliases and a drift warning surfaced by the
	// caller when the name has been retired by mem_merge_projects.
	ProjectName string `json:"project_name"`
}

// SaveObservation stores an observation with scope normalization matching MCP mem_save.
func (s *Service) SaveObservation(ctx context.Context, projectID string, in SaveObservationInput) (int64, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return 0, err
	}
	defer cleanup()
	scope := in.Scope
	if scope == "" {
		scope = "project"
	}
	if scope == "personal" {
		scope = "user"
	}
	return h.memory.Save(ctx, memory.SaveInput{
		Title:         in.Title,
		Type:          in.Type,
		Content:       in.Content,
		Scope:         scope,
		TopicKey:      in.TopicKey,
		SessionID:     in.SessionID,
		CapturePrompt: in.CapturePrompt,
		ProjectName:   in.ProjectName,
	})
}

// SearchObservations runs FTS over observations (scope = any).
func (s *Service) SearchObservations(ctx context.Context, projectID, query, matchMode string, limit int) ([]memory.Observation, error) {
	return s.SearchObservationsScoped(ctx, projectID, query, matchMode, "", limit)
}

// SearchObservationsScoped runs FTS over observations, restricting to a
// visibility scope when scope is non-empty (project|user|global).
func (s *Service) SearchObservationsScoped(ctx context.Context, projectID, query, matchMode, scope string, limit int) ([]memory.Observation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.SearchWithScope(ctx, query, matchMode, scope, limit)
}

// GetObservation returns a single observation by ID.
func (s *Service) GetObservation(ctx context.Context, projectID string, id int64) (memory.Observation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return memory.Observation{}, err
	}
	defer cleanup()
	return h.memory.Get(ctx, id)
}

// RecentContext returns recent session summaries.
func (s *Service) RecentContext(ctx context.Context, projectID string, limit int) ([]memory.Session, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.RecentContext(ctx, limit)
}

// UpdateObservation modifies an existing observation by ID. Only non-empty
// fields in in are applied. Bumps updated_at; FTS trigger keeps the index
// in sync.
func (s *Service) UpdateObservation(ctx context.Context, projectID string, id int64, in memory.UpdateInput) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.Update(ctx, id, in)
}

// DeleteObservation removes an observation. Soft-delete by default; hard
// when hardDelete is true.
func (s *Service) DeleteObservation(ctx context.Context, projectID string, id int64, hardDelete bool) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.Delete(ctx, id, hardDelete)
}

// Timeline returns the progressive-disclosure window around an observation.
func (s *Service) ObservationTimeline(ctx context.Context, projectID string, anchorID int64, window time.Duration, limit int) (memory.Timeline, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return memory.Timeline{}, err
	}
	defer cleanup()
	return h.memory.Timeline(ctx, anchorID, window, limit)
}

// ListReviews returns observations due for review, oldest review_after first.
func (s *Service) ListReviews(ctx context.Context, projectID string, limit int) ([]memory.ReviewDue, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.ListReviews(ctx, limit)
}

// MarkReviewed advances an observation's review cycle.
func (s *Service) MarkReviewReviewed(ctx context.Context, projectID string, id int64) (string, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return "", err
	}
	defer cleanup()
	return h.memory.MarkReviewed(ctx, id)
}

// SetReviewAfter sets the review_after for an observation.
func (s *Service) SetObservationReviewAfter(ctx context.Context, projectID string, id int64, reviewAfter string) error {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return err
	}
	defer cleanup()
	return h.memory.SetReviewAfter(ctx, id, reviewAfter)
}

// ProjectInfo describes the resolved project for cwd: id, source, all known projects.
type ProjectInfo struct {
	Project   string   `json:"project"`
	Source    string   `json:"source"`
	Directory string   `json:"directory"`
	Projects  []string `json:"projects"`
}

// CurrentProject returns the resolved project for cwd along with all
// available projects.
func (s *Service) CurrentProject(directory string) (ProjectInfo, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return ProjectInfo{}, err
	}
	id, src, err := project.ResolveDetailed(absDir)
	if err != nil {
		return ProjectInfo{}, err
	}
	projects, err := s.ListProjects()
	if err != nil {
		return ProjectInfo{}, err
	}
	return ProjectInfo{Project: id, Source: string(src), Directory: absDir, Projects: projects}, nil
}

// MemoryDoctor describes mnemonic store health for a project.
type MemoryDoctor struct {
	SchemaVersion   int            `json:"schema_version"`
	WALMode         string         `json:"wal_mode,omitempty"`
	Observations    int            `json:"observations"`
	ObservationsFTS int            `json:"observations_fts"`
	Files           int            `json:"files"`
	Chunks          int            `json:"chunks"`
	ChunksFTS       int            `json:"chunks_fts"`
	WebCache        int            `json:"web_cache"`
	WebCacheFTS     int            `json:"web_cache_fts"`
	Prompts         int            `json:"prompts"`
	ByType          map[string]int `json:"by_type"`
	DiskSizeBytes   int64          `json:"disk_size_bytes"`
	FTSIntegrityOK  bool           `json:"fts_integrity_ok"`
	FTSDrift        int            `json:"fts_drift"`
}

// MemoryDoctor runs read-only diagnostics: schema count, FTS row counts
// and drift, WAL state, and on-disk size.
func (s *Service) MemoryDoctor(ctx context.Context, projectID string) (MemoryDoctor, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return MemoryDoctor{}, err
	}
	defer cleanup()

	out := MemoryDoctor{}
	if err := h.store.DB.QueryRowContext(ctx, `SELECT schema_version FROM index_meta WHERE key='schema_version'`).Scan(&out.SchemaVersion); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return out, fmt.Errorf("schema_version: %w", err)
		}
	}
	if err := h.store.DB.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&out.WALMode); err != nil {
		return out, fmt.Errorf("journal_mode: %w", err)
	}
	byType := map[string]int{}
	h.rowCount(ctx, "observations", &out.Observations)
	h.rowCount(ctx, "files", &out.Files)
	h.rowCount(ctx, "chunks", &out.Chunks)
	h.rowCount(ctx, "web_cache", &out.WebCache)
	h.rowCount(ctx, "prompts", &out.Prompts)

	ftsObs, ftsChunks, ftsWeb := int64(0), int64(0), int64(0)
	_ = h.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM observations_fts`).Scan(&ftsObs)
	_ = h.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks_fts`).Scan(&ftsChunks)
	_ = h.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM web_cache_fts`).Scan(&ftsWeb)
	out.ObservationsFTS = int(ftsObs)
	out.ChunksFTS = int(ftsChunks)
	out.WebCacheFTS = int(ftsWeb)

	rows, err := h.store.DB.QueryContext(ctx, `
		SELECT type, COUNT(*) FROM observations
		WHERE project = ? AND deleted_at IS NULL GROUP BY type`, projectID)
	if err == nil {
		for rows.Next() {
			var t string
			var c int
			if rows.Scan(&t, &c) == nil {
				byType[t] = c
			}
		}
		rows.Close()
	}
	out.ByType = byType

	if info, err := os.Stat(h.store.Path()); err == nil {
		out.DiskSizeBytes = info.Size()
	}
	out.FTSDrift = int(ftsObs) - out.Observations
	out.FTSIntegrityOK = out.FTSDrift >= 0
	return out, nil
}

// rowCount helper (keeps MemoryDoctor readable).
func (h *projectHandle) rowCount(ctx context.Context, table string, out *int) {
	_ = h.store.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(out)
}

// CodeStatus returns index stats and whether the index is stale.
func (s *Service) CodeStatus(ctx context.Context, projectID string) (codeindex.Status, bool, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return codeindex.Status{}, false, err
	}
	defer cleanup()
	status, err := codeindex.GetStatus(h.store)
	if err != nil {
		return codeindex.Status{}, false, err
	}
	stale := status.FileCount == 0 || status.LastIndexed == ""
	return status, stale, nil
}

// CodeSearch runs BM25 FTS over indexed code chunks.
func (s *Service) CodeSearch(ctx context.Context, projectID, query string, limit int) ([]search.CodeHit, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return search.CodeSearch(h.store.DB, query, limit)
}

// CodeFiles returns all indexed file paths, sorted.
func (s *Service) CodeFiles(ctx context.Context, projectID string) ([]string, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	rows, err := h.store.DB.QueryContext(ctx, `SELECT path FROM files ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("list files: %w", err)
	}
	defer rows.Close()
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return paths, nil
}

// ReadIndexedCode returns indexed source for path and optional line range.
func (s *Service) ReadIndexedCode(ctx context.Context, projectID, path string, startLine, endLine int) (map[string]any, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return readIndexedCode(h.store.DB, path, startLine, endLine)
}

// RunCodeIndex runs incremental code indexing for directory.
func (s *Service) RunCodeIndex(ctx context.Context, directory string) (codeindex.Stats, error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return codeindex.Stats{}, err
	}
	defer cleanup()
	cfg := config.Load(directory)
	idxCfg := codeindex.Config{
		Include:      cfg.Include,
		Exclude:      cfg.Exclude,
		ChunkLines:   cfg.ChunkLines,
		ChunkOverlap: cfg.ChunkOverlap,
	}
	return codeindex.New(h.store).Run(ctx, directory, idxCfg)
}

// WebLookup checks the web cache for a matching entry.
func (s *Service) WebLookup(ctx context.Context, projectID string, in webcache.LookupInput) (webcache.LookupResult, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return webcache.LookupResult{}, err
	}
	defer cleanup()
	return h.web.Lookup(ctx, in)
}

// WebSave persists a web research snapshot.
func (s *Service) WebSave(ctx context.Context, projectID string, in webcache.SaveWebInput) (int64, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return 0, err
	}
	defer cleanup()
	return h.web.Save(ctx, in)
}

// WebSearch runs FTS over cached web snapshots.
func (s *Service) WebSearch(ctx context.Context, projectID, query, source string, freshOnly bool, limit int) ([]webcache.WebHit, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.web.Search(ctx, query, source, freshOnly, limit)
}

// WebGet returns a full cached snapshot by ID.
func (s *Service) WebGet(ctx context.Context, projectID string, id int64) (webcache.WebEntry, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return webcache.WebEntry{}, err
	}
	defer cleanup()
	return h.web.Get(ctx, id)
}

// MemoryStatus returns memory store health stats.
func (s *Service) MemoryStatus(ctx context.Context, projectID string) (memory.Status, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return memory.Status{}, err
	}
	defer cleanup()
	return h.memory.Status(ctx)
}

// WebCacheStatus returns web cache health stats.
func (s *Service) WebCacheStatus(ctx context.Context, projectID string) (webcache.Status, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return webcache.Status{}, err
	}
	defer cleanup()
	return h.web.CacheStatus(ctx)
}

// OpenForCWD opens project-scoped services from the current working directory.
func (s *Service) OpenForCWD() (*projectHandle, func(), error) {
	return s.openProjectFromCWD()
}

// OpenForDirectory opens project-scoped services for directory.
func (s *Service) OpenForDirectory(directory string) (*projectHandle, func(), error) {
	return s.openProjectForDirectory(directory)
}

// ProjectID returns the project ID for an open handle.
func (h *projectHandle) ProjectID() string { return h.projectID }

// Memory returns the memory service for an open handle.
func (h *projectHandle) Memory() *memory.Service { return h.memory }

// Web returns the webcache service for an open handle.
func (h *projectHandle) Web() *webcache.Service { return h.web }

// Store returns the underlying store for an open handle.
func (h *projectHandle) Store() *store.Store { return h.store }

func readIndexedCode(db *sql.DB, path string, startLine, endLine int) (map[string]any, error) {
	var fileID int64
	err := db.QueryRow(`SELECT id FROM files WHERE path = ?`, path).Scan(&fileID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("file not indexed: %s", path)
	}
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if startLine > 0 {
		if endLine <= 0 {
			endLine = startLine
		}
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ? AND start_line <= ? AND end_line >= ?
			ORDER BY start_line`,
			fileID, endLine, startLine,
		)
	} else {
		rows, err = db.Query(`
			SELECT start_line, end_line, text FROM chunks
			WHERE file_id = ?
			ORDER BY start_line`,
			fileID,
		)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var parts []string
	firstLine := 0
	lastLine := 0
	for rows.Next() {
		var chunkStart, chunkEnd int
		var text string
		if err := rows.Scan(&chunkStart, &chunkEnd, &text); err != nil {
			return nil, err
		}
		if firstLine == 0 || chunkStart < firstLine {
			firstLine = chunkStart
		}
		if chunkEnd > lastLine {
			lastLine = chunkEnd
		}
		parts = append(parts, text)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("no indexed chunks for %s", path)
	}
	return map[string]any{
		"path":       path,
		"start_line": firstLine,
		"end_line":   lastLine,
		"text":       strings.Join(parts, "\n"),
	}, nil
}
