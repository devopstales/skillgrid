package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetupPi_InstallsExtensionFolder(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	if err := SetupPi(home, repoRoot, false); err != nil {
		t.Fatalf("SetupPi: %v", err)
	}
	dstDir := filepath.Join(home, ".skill-pi", "agent", "extensions", "mnemonic")
	for _, name := range piExtensionFiles {
		data, err := os.ReadFile(filepath.Join(dstDir, name))
		if err != nil {
			t.Fatalf("missing pi extension file %s: %v", name, err)
		}
		if len(data) == 0 {
			t.Fatalf("pi extension file %s is empty", name)
		}
	}
	// The entry point must resolve under the launcher's own convention
	// (manifest path or index.ts); normalize the "./" prefix.
	if entry := strings.TrimPrefix(piExtensionEntry(dstDir), "./"); entry != "index.ts" {
		t.Fatalf("pi extension entry point = %q, want index.ts", entry)
	}

	// Dry-run must not write.
	homeDry := t.TempDir()
	if err := SetupPi(homeDry, repoRoot, true); err != nil {
		t.Fatalf("SetupPi dry-run: %v", err)
	}
	if _, err := os.Stat(dstDirOf(homeDry)); err == nil {
		t.Fatal("dry-run must not write the pi extension")
	}
}

func TestSetupPi_HonorsAgentDirEnv(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	home := t.TempDir()
	custom := filepath.Join(home, "custom-agent-dir")
	t.Setenv("SKILL_PI_AGENT_DIR", custom)
	if err := SetupPi(home, repoRoot, false); err != nil {
		t.Fatalf("SetupPi: %v", err)
	}
	dst := filepath.Join(custom, "extensions", "mnemonic", "index.ts")
	if _, err := os.Stat(dst); err != nil {
		t.Fatalf("SKILL_PI_AGENT_DIR not honored: %v", err)
	}
}

func TestRunSetup_RoutesPiAgent(t *testing.T) {
	repoRoot := FindRepoRoot("")
	if repoRoot == "" {
		t.Fatal("repo root not found")
	}
	for _, agent := range []string{"pi", "skill-pi"} {
		h := t.TempDir()
		if err := RunSetup(agent, repoRoot, nil, h, false); err != nil {
			t.Fatalf("RunSetup(%s): %v", agent, err)
		}
		dst := filepath.Join(h, ".skill-pi", "agent", "extensions", "mnemonic", "index.ts")
		if _, err := os.Stat(dst); err != nil {
			t.Fatalf("RunSetup(%s) did not install the extension: %v", agent, err)
		}
	}
}

func TestPiExtensionEntry(t *testing.T) {
	dir := t.TempDir()
	// No manifest: index.ts wins.
	if err := os.WriteFile(filepath.Join(dir, "index.ts"), []byte("// x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := piExtensionEntry(dir); got != "index.ts" {
		t.Fatalf("piExtensionEntry = %q, want index.ts", got)
	}
	// Manifest path wins when it exists.
	manifest := `{"pi":{"extensions":["./src/entry.ts"]}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "entry.ts"), []byte("// x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := piExtensionEntry(dir); got != "./src/entry.ts" {
		t.Fatalf("piExtensionEntry = %q, want ./src/entry.ts", got)
	}
}

func dstDirOf(home string) string {
	return filepath.Join(home, ".skill-pi", "agent", "extensions", "mnemonic")
}
