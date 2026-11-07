package projectwiki

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

const (
	VaultConfigFile       = "config.json"
	VaultSchemaFile       = "PROJECTWIKI_SCHEMA.md"
	VaultIgnoreFile       = ".ignore"
	legacyVaultIgnoreFile = ".knowledgeignore" // pre-rename name (read fallback)
	VaultIndexDir         = "index"
)

var PageTypeDirs = []string{
	"decisions",
	"invariants",
	"workflows",
	"architecture",
	"gotchas",
	"glossary",
}

type Options struct {
	Project      string
	AnchorEngine string
	Strict       bool
}

type Scaffold struct {
	Root    string
	Created []string
	Skipped []string
}

// Bootstrap scaffolds a project wiki vault at root. Idempotent: existing
// sentinel config.json → no-op.
func Bootstrap(ctx context.Context, st store.Store, root string, opts Options) (*Scaffold, error) {
	sc := &Scaffold{Root: root}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("projectwiki: mkdir %s: %w", root, err)
	}
	sentinel := filepath.Join(root, VaultConfigFile)
	if _, err := os.Stat(sentinel); err == nil {
		sc.Skipped = append(sc.Skipped, VaultConfigFile)
		return sc, nil
	}
	created := []string{}
	cfg := map[string]any{
		"schema_version": 1,
		"project":        opts.Project,
		"page_types":     PageTypeDirs,
		"anchor_engine":  opts.AnchorEngine,
		"strict":         opts.Strict,
	}
	cfgBytes, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("projectwiki: marshal config: %w", err)
	}
	if err := os.WriteFile(sentinel, append(cfgBytes, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultConfigFile, err)
	}
	created = append(created, VaultConfigFile)
	schema := "# ProjectWiki Schema\n\nOperating manual for the project wiki vault at this root.\n"
	if err := os.WriteFile(filepath.Join(root, VaultSchemaFile), []byte(schema), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultSchemaFile, err)
	}
	created = append(created, VaultSchemaFile)
	if err := os.WriteFile(filepath.Join(root, VaultIgnoreFile), []byte("# Project wiki read boundary\n"), 0o644); err != nil {
		return nil, fmt.Errorf("projectwiki: write %s: %w", VaultIgnoreFile, err)
	}
	created = append(created, VaultIgnoreFile)
	dirs := append([]string{VaultIndexDir}, PageTypeDirs...)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return nil, fmt.Errorf("projectwiki: mkdir %s: %w", d, err)
		}
	}
	return &Scaffold{Root: root, Created: created}, nil
}

// Root returns the vault root for cwd: .skillgrid/projectwiki/ when it
// exists, else the legacy pre-rename vault dir, else the new path (the
// bootstrap target).
func Root(cwd string) string {
	newRoot := filepath.Join(cwd, ".skillgrid", "projectwiki")
	if dirExists(newRoot) {
		return newRoot
	}
	legacy := filepath.Join(cwd, ".skillgrid", "knowledge")
	if dirExists(legacy) {
		return legacy
	}
	return newRoot
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}
