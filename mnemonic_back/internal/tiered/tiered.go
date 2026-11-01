package tiered

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Store registers and reads L0/L1/L2 tiered content on the filesystem + SQL.
type Store struct {
	DB         *sql.DB
	Summarizer Summarizer
	// Logf is optional; used when summarizer/tier generation fails (warn+continue).
	Logf func(format string, args ...any)
}

func (s *Store) logf(format string, args ...any) {
	if s != nil && s.Logf != nil {
		s.Logf(format, args...)
	}
}

func (s *Store) summarizer() Summarizer {
	if s != nil && s.Summarizer != nil {
		return s.Summarizer
	}
	return HeuristicSummarizer{}
}

// SidecarPaths returns the L0 (.abstract) and L1 (.overview) paths for an L2 file.
func SidecarPaths(fullPath string) (abstractPath, overviewPath string) {
	return fullPath + ".abstract", fullPath + ".overview"
}

// GenerateTiers writes L0/L1 sidecars from L2 and upserts tiered_contents.
// On summarizer failure it logs, leaves L2 intact, and returns the error
// (callers that must warn+continue should ignore or wrap).
func (s *Store) GenerateTiers(ctx context.Context, project, fullPath, title string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("tiered store is nil")
	}
	_ = ctx
	fullPath = filepath.Clean(fullPath)
	body, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("read L2: %w", err)
	}
	absPath, overPath := SidecarPaths(fullPath)
	sum := s.summarizer()
	abstract, err := sum.Abstract(string(body))
	if err != nil {
		s.logf("tiered: abstract failed for %s: %v", fullPath, err)
		return fmt.Errorf("abstract: %w", err)
	}
	overview, err := sum.Overview(string(body))
	if err != nil {
		s.logf("tiered: overview failed for %s: %v", fullPath, err)
		return fmt.Errorf("overview: %w", err)
	}
	if err := os.WriteFile(absPath, []byte(abstract), 0o644); err != nil {
		return fmt.Errorf("write abstract: %w", err)
	}
	if err := os.WriteFile(overPath, []byte(overview), 0o644); err != nil {
		return fmt.Errorf("write overview: %w", err)
	}
	return s.Register(ctx, project, fullPath, absPath, overPath, title)
}

// Register upserts path columns in tiered_contents.
func (s *Store) Register(ctx context.Context, project, fullPath, abstractPath, overviewPath, title string) error {
	if s == nil || s.DB == nil {
		return fmt.Errorf("tiered store is nil")
	}
	_ = ctx
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.DB.Exec(`
		INSERT INTO tiered_contents (project, full_path, abstract_path, overview_path, title, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project, full_path) DO UPDATE SET
			abstract_path = excluded.abstract_path,
			overview_path = excluded.overview_path,
			title = COALESCE(excluded.title, tiered_contents.title),
			updated_at = excluded.updated_at
	`, project, fullPath, nullIfEmpty(abstractPath), nullIfEmpty(overviewPath), nullIfEmpty(title), now, now)
	if err != nil {
		return fmt.Errorf("register tiered_contents: %w", err)
	}
	return nil
}

// ReadL2 returns full markdown at fullPath.
func ReadL2(fullPath string) (string, error) {
	b, err := os.ReadFile(filepath.Clean(fullPath))
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadL0 returns the abstract sidecar for fullPath (or registered abstract_path).
func ReadL0(fullPath string) (string, error) {
	abs, _ := SidecarPaths(fullPath)
	b, err := os.ReadFile(abs)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ReadL1 returns the overview sidecar for fullPath.
func ReadL1(fullPath string) (string, error) {
	_, over := SidecarPaths(fullPath)
	b, err := os.ReadFile(over)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// MigrateTier walks contentRoot for L2 markdown files (not sidecars), generates
// missing L0/L1, and registers paths. L2 bytes are never rewritten.
func MigrateTier(ctx context.Context, s *Store, project, contentRoot string) (int, error) {
	if s == nil {
		return 0, fmt.Errorf("tiered store is nil")
	}
	contentRoot = filepath.Clean(contentRoot)
	var n int
	err := filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".abstract") || strings.HasSuffix(name, ".overview") {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}
		abs, over := SidecarPaths(path)
		if fileExists(abs) && fileExists(over) {
			// Still ensure SQL registration.
			if regErr := s.Register(ctx, project, path, abs, over, ""); regErr != nil {
				return regErr
			}
			n++
			return nil
		}
		if genErr := s.GenerateTiers(ctx, project, path, ""); genErr != nil {
			s.logf("tiered migrate: skip %s: %v", path, genErr)
			// warn+continue — do not abort the whole walk
			return nil
		}
		n++
		return nil
	})
	return n, err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func nullIfEmpty(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}
