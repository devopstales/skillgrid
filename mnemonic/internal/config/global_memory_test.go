package config

// GlobalMemory char budgets (change 2026-10-24-global-llm-wiki, Change A,
// TICKET-04): the bounded global fact store has two per-category char limits
// (memory 2200 / user 1375, the Hermes MEMORY.md / USER.md budgets), read
// from the mnemonic.global_memory section of config.d/indexing.yaml.

import (
	"os"
	"path/filepath"
	"testing"
)

func writeGlobalMemoryConfig(t *testing.T, dir, body string) string {
	t.Helper()
	cfgDir := filepath.Join(dir, ".skillgrid", "config.d")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(cfgDir, "indexing.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestGlobalMemoryCharLimitsDefault pins the production defaults: absent
// section → memory 2200 / user 1375 (the Hermes budgets).
func TestGlobalMemoryCharLimitsDefault(t *testing.T) {
	dir := t.TempDir()
	got := Load(dir)
	if got.GlobalMemory.MemoryCharLimit != 2200 {
		t.Fatalf("default: global_memory.memory_char_limit = %d, want 2200", got.GlobalMemory.MemoryCharLimit)
	}
	if got.GlobalMemory.UserCharLimit != 1375 {
		t.Fatalf("default: global_memory.user_char_limit = %d, want 1375", got.GlobalMemory.UserCharLimit)
	}

	// A config without the section → still the defaults.
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid", "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml"),
		[]byte("mnemonic:\n  ttl: 72h\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = Load(dir)
	if got.GlobalMemory.MemoryCharLimit != 2200 || got.GlobalMemory.UserCharLimit != 1375 {
		t.Fatalf("absent section: got memory=%d user=%d, want 2200/1375", got.GlobalMemory.MemoryCharLimit, got.GlobalMemory.UserCharLimit)
	}
}

// TestGlobalMemoryCharLimitsOverride: an explicit section overrides the
// defaults; a partial section (only one key) keeps the other default.
func TestGlobalMemoryCharLimitsOverride(t *testing.T) {
	dir := t.TempDir()

	writeGlobalMemoryConfig(t, dir, `
mnemonic:
  global_memory:
    memory_char_limit: 3000
    user_char_limit: 2000
`)
	got := Load(dir)
	if got.GlobalMemory.MemoryCharLimit != 3000 || got.GlobalMemory.UserCharLimit != 2000 {
		t.Fatalf("override: got memory=%d user=%d, want 3000/2000", got.GlobalMemory.MemoryCharLimit, got.GlobalMemory.UserCharLimit)
	}

	// Partial section: only memory_char_limit set → user stays 1375.
	writeGlobalMemoryConfig(t, dir, `
mnemonic:
  global_memory:
    memory_char_limit: 999
`)
	got = Load(dir)
	if got.GlobalMemory.MemoryCharLimit != 999 {
		t.Fatalf("partial override: memory_char_limit = %d, want 999", got.GlobalMemory.MemoryCharLimit)
	}
	if got.GlobalMemory.UserCharLimit != 1375 {
		t.Fatalf("partial override: user_char_limit = %d, want the default 1375", got.GlobalMemory.UserCharLimit)
	}
}
