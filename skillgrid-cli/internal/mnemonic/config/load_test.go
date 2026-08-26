package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"skillgrid-cli/internal/mnemonic/config"
)

func TestLoadNoFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Load(dir)

	defaults := config.DefaultIndexing()
	if cfg.ChunkLines != defaults.ChunkLines {
		t.Fatalf("ChunkLines = %d, want %d", cfg.ChunkLines, defaults.ChunkLines)
	}
	if len(cfg.Include) != len(defaults.Include) {
		t.Fatalf("Include len = %d, want %d", len(cfg.Include), len(defaults.Include))
	}
}

func TestLoadFromRepoConfigD(t *testing.T) {
	root := t.TempDir()
	configDir := filepath.Join(root, "config.d")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `profile: mnemonic
mnemonic:
  include: ["**/*.py"]
  exclude: ["**/venv/**"]
  chunk_lines: 40
  chunk_overlap: 5
  web_cache:
    enabled: false
    max_entry_bytes: 1024
    ttl:
      context7: 1h
    sources: [context7]
`
	if err := os.WriteFile(filepath.Join(configDir, "indexing.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	nested := filepath.Join(root, "pkg", "sub")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	cfg := config.Load(nested)
	if len(cfg.Include) != 1 || cfg.Include[0] != "**/*.py" {
		t.Fatalf("Include = %#v, want [**/*.py]", cfg.Include)
	}
	if cfg.ChunkLines != 40 {
		t.Fatalf("ChunkLines = %d, want 40", cfg.ChunkLines)
	}
	if cfg.ChunkOverlap != 5 {
		t.Fatalf("ChunkOverlap = %d, want 5", cfg.ChunkOverlap)
	}
	if cfg.WebCache.Enabled {
		t.Fatal("WebCache.Enabled = true, want false")
	}
	if cfg.WebCache.MaxEntryBytes != 1024 {
		t.Fatalf("MaxEntryBytes = %d, want 1024", cfg.WebCache.MaxEntryBytes)
	}
	if cfg.WebCache.TTL["context7"] != time.Hour {
		t.Fatalf("TTL context7 = %v, want 1h", cfg.WebCache.TTL["context7"])
	}
}
