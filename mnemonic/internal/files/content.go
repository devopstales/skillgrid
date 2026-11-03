// Package files implements the filesystem data plane for hybrid agent teams.
package files

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Kind selects the content subtree under .skillgrid/files/.
type Kind string

const (
	KindTasks    Kind = "tasks"
	KindMessages Kind = "messages"
	KindReviews  Kind = "reviews"
)

// ContentPlane writes and reads markdown under {project}/.skillgrid/files/.
// Writes are FS-first: the file is created before commit runs; on commit
// failure the file is removed so SQL and disk stay aligned.
type ContentPlane struct {
	ProjectRoot string
	// AfterWrite is an optional post-write seam (no-op consumers ok).
	// Change 003 may hook tiered storage here; this change leaves it unset or no-op.
	AfterWrite func(relPath string, content []byte) error
}

// NewContentPlane returns a ContentPlane rooted at projectRoot.
func NewContentPlane(projectRoot string) *ContentPlane {
	return &ContentPlane{ProjectRoot: projectRoot}
}

// Write stores content at files/{kind}/{id}/{filename} relative to .skillgrid/,
// then calls commit with that relative path. If commit fails, the file is deleted.
func (c *ContentPlane) Write(kind Kind, id, filename string, content []byte, commit func(relPath string) error) (string, error) {
	if c == nil || strings.TrimSpace(c.ProjectRoot) == "" {
		return "", fmt.Errorf("content plane: project root required")
	}
	if err := validateKind(kind); err != nil {
		return "", err
	}
	if strings.TrimSpace(id) == "" || strings.Contains(id, "..") || strings.Contains(id, "/") || strings.Contains(id, `\`) {
		return "", fmt.Errorf("content plane: invalid id %q", id)
	}
	if strings.TrimSpace(filename) == "" || strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, `\`) {
		return "", fmt.Errorf("content plane: invalid filename %q", filename)
	}
	if commit == nil {
		return "", fmt.Errorf("content plane: commit callback required")
	}

	relPath := filepath.ToSlash(filepath.Join("files", string(kind), id, filename))
	absPath := filepath.Join(c.ProjectRoot, ".skillgrid", filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", fmt.Errorf("content plane: mkdir: %w", err)
	}
	if err := os.WriteFile(absPath, content, 0o644); err != nil {
		return "", fmt.Errorf("content plane: write file: %w", err)
	}

	if err := commit(relPath); err != nil {
		_ = os.Remove(absPath)
		return "", fmt.Errorf("content plane: commit: %w", err)
	}

	if c.AfterWrite != nil {
		if err := c.AfterWrite(relPath, content); err != nil {
			return relPath, fmt.Errorf("content plane: after-write seam: %w", err)
		}
	}
	return relPath, nil
}

// Read returns the markdown bytes for a path relative to .skillgrid/ (e.g. files/tasks/…/brief.md).
func (c *ContentPlane) Read(relPath string) ([]byte, error) {
	if c == nil || strings.TrimSpace(c.ProjectRoot) == "" {
		return nil, fmt.Errorf("content plane: project root required")
	}
	relPath = filepath.ToSlash(relPath)
	if relPath == "" || strings.Contains(relPath, "..") || !strings.HasPrefix(relPath, "files/") {
		return nil, fmt.Errorf("content plane: invalid path %q", relPath)
	}
	absPath := filepath.Join(c.ProjectRoot, ".skillgrid", filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("content plane: read: %w", err)
	}
	return data, nil
}

func validateKind(kind Kind) error {
	switch kind {
	case KindTasks, KindMessages, KindReviews:
		return nil
	default:
		return fmt.Errorf("content plane: unknown kind %q", kind)
	}
}
