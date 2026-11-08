package projectwiki

import (
	"os"
	"path/filepath"
	"testing"
)

// [projectwiki-root-legacy-fallback] Root prefers the new .skillgrid/projectwiki
// vault, falls back to the legacy pre-rename vault, and defaults to
// the new path when neither exists (the bootstrap target).
func TestRootResolution(t *testing.T) {
	// legacy-only
	legacyOnly := t.TempDir()
	if err := os.MkdirAll(filepath.Join(legacyOnly, ".skillgrid", "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := Root(legacyOnly), filepath.Join(legacyOnly, ".skillgrid", "knowledge"); got != want {
		t.Fatalf("legacy-only root = %s, want %s", got, want)
	}

	// both present → the new path wins
	both := t.TempDir()
	if err := os.MkdirAll(filepath.Join(both, ".skillgrid", "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(both, ".skillgrid", "projectwiki"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := Root(both), filepath.Join(both, ".skillgrid", "projectwiki"); got != want {
		t.Fatalf("both root = %s, want %s", got, want)
	}

	// neither → the new path (bootstrap target)
	neither := t.TempDir()
	if got, want := Root(neither), filepath.Join(neither, ".skillgrid", "projectwiki"); got != want {
		t.Fatalf("neither root = %s, want %s", got, want)
	}
}

// [projectwiki-ignore-legacy-fallback] loadIgnore reads the legacy
// .knowledgeignore when the new .ignore is absent (pre-rename vaults), and
// the new .ignore wins when both exist.
func TestLoadIgnoreLegacyFallback(t *testing.T) {
	vault := t.TempDir()
	writeVaultPage(t, vault, "gotchas/legacy.md", "body")
	if err := os.WriteFile(filepath.Join(vault, ".knowledgeignore"),
		[]byte("gotchas/legacy.md\n"), 0o644); err != nil {
		t.Fatalf("write legacy ignore: %v", err)
	}
	ignore, err := loadIgnore(vault)
	if err != nil {
		t.Fatalf("loadIgnore: %v", err)
	}
	if !ignore("gotchas/legacy.md") {
		t.Fatal("legacy .knowledgeignore not honored (fallback broken)")
	}

	// the new file takes precedence when present
	if err := os.WriteFile(filepath.Join(vault, ".ignore"),
		[]byte("# new\n"), 0o644); err != nil {
		t.Fatalf("write new ignore: %v", err)
	}
	ignore, err = loadIgnore(vault)
	if err != nil {
		t.Fatalf("loadIgnore again: %v", err)
	}
	if ignore("gotchas/legacy.md") {
		t.Fatal("legacy .knowledgeignore still applied while .ignore exists")
	}
}
