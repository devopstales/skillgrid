package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// VaultConfigFile is the sentinel that marks a directory as a knowledge base.
	VaultConfigFile = "config.json"
	// VaultSchemaFile is the operating manual (human + agent) for the vault.
	VaultSchemaFile = "KNOWLEDGE_SCHEMA.md"
	// VaultIgnoreFile is the gitignore-style read boundary for the vault.
	VaultIgnoreFile = ".knowledgeignore"
	// VaultIndexDir is the derived run-checkpoint directory (the FTS index
	// itself lives in the shared project DB, per 004-index).
	VaultIndexDir = "index"
)

// PageTypeDirs are the KB page-type folders living directly under the vault
// root (flat layout, no kb/ wrapper — 001-structure).
var PageTypeDirs = []string{
	"decisions",
	"invariants",
	"workflows",
	"architecture",
	"gotchas",
	"glossary",
}

// Options tunes the scaffold written by Bootstrap.
type Options struct {
	Project      string
	AnchorEngine string
	Strict       bool
}

// Scaffold reports what Bootstrap did for one vault root.
type Scaffold struct {
	Root    string
	Created []string
	Skipped []string
}

// Bootstrap scaffolds a knowledge vault at root. It is idempotent: when the
// sentinel config.json already exists the vault is treated as present and the
// run is a no-op (all entries skipped). A missing vault root is created.
func Bootstrap(ctx context.Context, root string, opts Options) (*Scaffold, error) {
	sc := &Scaffold{Root: root}
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
		return nil, fmt.Errorf("kb: marshal config: %w", err)
	}
	if err := os.WriteFile(sentinel, append(cfgBytes, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("kb: write %s: %w", VaultConfigFile, err)
	}
	created = append(created, VaultConfigFile)

	schema := "# Knowledge Schema\n\nOperating manual for the knowledge vault at this root.\n"
	if err := os.WriteFile(filepath.Join(root, VaultSchemaFile), []byte(schema), 0o644); err != nil {
		return nil, fmt.Errorf("kb: write %s: %w", VaultSchemaFile, err)
	}
	created = append(created, VaultSchemaFile)

	if err := os.WriteFile(filepath.Join(root, VaultIgnoreFile), []byte("# Knowledge vault read boundary\n"), 0o644); err != nil {
		return nil, fmt.Errorf("kb: write %s: %w", VaultIgnoreFile, err)
	}
	created = append(created, VaultIgnoreFile)

	dirs := append([]string{VaultIndexDir}, PageTypeDirs...)
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			return nil, fmt.Errorf("kb: mkdir %s: %w", d, err)
		}
	}

	return &Scaffold{Root: root, Created: created}, nil
}
