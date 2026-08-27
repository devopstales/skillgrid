// Package service is the shared facade over memory, codeindex, webcache, and search.
package service

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

// ResolveProject returns the project ID for directory.
func (s *Service) ResolveProject(directory string) (string, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	return project.Resolve(absDir)
}

// SessionStart creates a workspace session for directory.
func (s *Service) SessionStart(ctx context.Context, directory string) (sessionID, projectID string, err error) {
	h, cleanup, err := s.openProjectForDirectory(directory)
	if err != nil {
		return "", "", err
	}
	defer cleanup()
	sessionID, err = h.memory.SessionStart(ctx, directory)
	if err != nil {
		return "", "", err
	}
	return sessionID, h.projectID, nil
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

// SaveObservationInput holds fields for saving an observation (HTTP + MCP).
type SaveObservationInput struct {
	Title     string
	Type      string
	Content   string
	Scope     string
	TopicKey  string
	SessionID string
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
		Title:     in.Title,
		Type:      in.Type,
		Content:   in.Content,
		Scope:     scope,
		TopicKey:  in.TopicKey,
		SessionID: in.SessionID,
	})
}

// SearchObservations runs FTS over observations.
func (s *Service) SearchObservations(ctx context.Context, projectID, query, matchMode string, limit int) ([]memory.Observation, error) {
	h, cleanup, err := s.openProject(projectID, ".")
	if err != nil {
		return nil, err
	}
	defer cleanup()
	return h.memory.Search(ctx, query, matchMode, limit)
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
