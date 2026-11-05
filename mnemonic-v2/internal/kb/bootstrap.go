package kb

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

const (
	VaultConfigFile = "config.json"
	VaultSchemaFile = "KNOWLEDGE_SCHEMA.md"
	VaultIgnoreFile = ".knowledgeignore"
	VaultIndexDir   = "index"
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

// Bootstrap scaffolds a knowledge vault at root. Idempotent: existing
// sentinel config.json → no-op.
func Bootstrap(ctx context.Context, st store.Store, root string, opts Options) (*Scaffold, error) {
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
