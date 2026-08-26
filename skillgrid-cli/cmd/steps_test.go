package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"skillgrid-cli/internal/logging"
)

func TestMnemonicSetupDryRun(t *testing.T) {
	baseDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	logging.ResetForTest()
	if err := logging.Init(baseDir); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}

	configDir := filepath.Join(baseDir, "config.d")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "indexing.yaml"), []byte("profile: mnemonic\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	repoRoot := filepath.Join(baseDir, "repos", "skillgrid")
	writeMnemonicRepo(t, repoRoot)

	opencodeDir := filepath.Join(home, ".config", "opencode")
	kiloDir := filepath.Join(home, ".config", "kilo")
	cursorDir := filepath.Join(home, ".cursor")
	for _, dir := range []string{opencodeDir, kiloDir, cursorDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{
		filepath.Join(opencodeDir, "opencode.jsonc"),
		filepath.Join(kiloDir, "kilo.jsonc"),
		filepath.Join(cursorDir, "mcp.json"),
	} {
		if err := os.WriteFile(p, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repoRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	agents := []string{"kilo", "opencode", "cursor"}
	installMnemonicPlugins(baseDir, agents, true)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()

	for _, want := range []string{
		"[dry-run] skillgrid setup opencode",
		"[dry-run] skillgrid setup kilocode",
		"[dry-run] skillgrid setup cursor",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in output, got:\n%s", want, out)
		}
	}
}

func TestMnemonicSetupSkippedWhenProfileNotMnemonic(t *testing.T) {
	baseDir := t.TempDir()
	logging.ResetForTest()
	if err := logging.Init(baseDir); err != nil {
		t.Fatalf("logging.Init: %v", err)
	}

	configDir := filepath.Join(baseDir, "config.d")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "indexing.yaml"), []byte("profile: default\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	installMnemonicPlugins(baseDir, []string{"opencode"}, true)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if strings.Contains(out, "[dry-run] skillgrid setup") {
		t.Fatalf("expected no mnemonic setup logs, got:\n%s", out)
	}
}

func writeMnemonicRepo(t *testing.T, repoRoot string) {
	t.Helper()
	files := map[string]string{
		"plugins/mnemonic/opencode/mnemonic.ts":     "// mnemonic plugin",
		"plugins/mnemonic/shared/http-client.ts":    "// http client",
		"plugins/mnemonic/shared/memory-protocol.md": "# Memory protocol",
		"plugins/mnemonic/cursor/mnemonic.mdc":      "---\nalwaysApply: true\n---\n{{MEMORY_PROTOCOL}}",
	}
	for rel, content := range files {
		path := filepath.Join(repoRoot, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
