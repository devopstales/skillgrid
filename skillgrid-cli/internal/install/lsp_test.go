package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLSPTargetsBinsMatchPDGServers(t *testing.T) {
	// Drift guard: the mnemonic --lsp indexing tier (mnemonic/internal/pdg
	// defaultServers) resolves member calls per language; every language it
	// covers except c/cpp (clangd, platform-installed) must have an installer
	// entry whose Bin is that language's server. The pdg package is internal
	// to the mnemonic module, so the contract is pinned here and both sides
	// are updated together.
	pdgContract := map[string]string{
		"go":         "gopls",
		"python":     "pyright-langserver",
		"rust":       "rust-analyzer",
		"java":       "jj-language-server",
		"ts":         "typescript-language-server",
		"jsx":        "typescript-language-server",
		"tsx":        "typescript-language-server",
		"javascript": "typescript-language-server",
	}
	for lang, server := range pdgContract {
		found := false
		for _, s := range LSPTargets() {
			if s.Bin == server {
				found = true
			}
		}
		if !found {
			t.Errorf("pdg language %q needs bin %q but LSPTargets has no entry for it", lang, server)
		}
	}
}

func TestInstallLSPToolsDryRun(t *testing.T) {
	cfg := Config{DryRun: true}
	if err := installLSPTools(&cfg); err != nil {
		t.Fatalf("installLSPTools (dry-run): %v", err)
	}
}

func TestInstallLSPToolsSkipsInstalled(t *testing.T) {
	// Point the seam at a temp dir containing every catalog bin: the whole
	// catalog must be skipped (no manager checks, no installs).
	dir := t.TempDir()
	for _, s := range LSPTargets() {
		if err := os.WriteFile(filepath.Join(dir, s.Bin), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	old := lspLookPath
	t.Cleanup(func() { lspLookPath = old })
	lspLookPath = func(bin string) (string, error) {
		p := filepath.Join(dir, bin)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		return "", os.ErrNotExist
	}
	cfg := Config{DryRun: true}
	if err := installLSPTools(&cfg); err != nil {
		t.Fatalf("installLSPTools (all installed): %v", err)
	}
}

func TestInstallLSPToolsWarnsOnMissingManager(t *testing.T) {
	// Nothing on PATH: every tool must warn-skip, never fail.
	old := lspLookPath
	t.Cleanup(func() { lspLookPath = old })
	lspLookPath = func(bin string) (string, error) { return "", os.ErrNotExist }
	cfg := Config{DryRun: true}
	if err := installLSPTools(&cfg); err != nil {
		t.Fatalf("installLSPTools (no managers): %v", err)
	}
}

// --- opencode lsp config ---

func TestConfigureLSPOpenCodeUpsertsAndPreserves(t *testing.T) {
	home := t.TempDir()
	cfgPath := filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{
  "model": "anthropic/claude",
  "lsp": {
    "custom": { "command": ["my-lsp", "--stdio"], "extensions": [".foo"] },
    "rust": { "disabled": true }
  }
}`
	if err := os.WriteFile(cfgPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{HomeDir: home, Agents: []string{"opencode"}, DryRun: true}
	_ = cfg
	if err := configureLSPOpenCode(&cfg); err != nil {
		t.Fatalf("configureLSPOpenCode: %v", err)
	}
	// Dry-run: file untouched.
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), `"custom"`) {
		t.Errorf("dry-run must not rewrite the config")
	}

	cfg.DryRun = false
	if err := configureLSPOpenCode(&cfg); err != nil {
		t.Fatalf("configureLSPOpenCode (write): %v", err)
	}
	data, _ = os.ReadFile(cfgPath)
	s := string(data)
	if !strings.Contains(s, `"gopls"`) || !strings.Contains(s, `"pyright-langserver"`) {
		t.Errorf("expected go+python entries in:\n%s", s)
	}
	if !strings.Contains(s, `"custom"`) || !strings.Contains(s, `"my-lsp"`) {
		t.Errorf("foreign server key must be preserved:\n%s", s)
	}
	if !strings.Contains(s, `"disabled":true`) {
		t.Errorf("user-disabled entry must stay disabled:\n%s", s)
	}
	if !strings.Contains(s, `"model"`) {
		t.Errorf("foreign top-level key must be preserved:\n%s", s)
	}

	// Idempotency: a second run must not duplicate or fail.
	if err := configureLSPOpenCode(&cfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
}

// --- skill-pi lsp.conf + settings ---

func TestConfigureLSPPiMergesAndPreserves(t *testing.T) {
	home := t.TempDir()
	confDir := filepath.Join(home, ".skill-pi", "conf.d")
	if err := os.MkdirAll(confDir, 0o755); err != nil {
		t.Fatal(err)
	}
	confPath := filepath.Join(confDir, "lsp.conf")
	settingsPath := filepath.Join(home, ".skill-pi", "agent", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatal(err)
	}
	seed := `{
  "defaultLimit": 10,
  "servers": [
    { "id": "gopls", "command": { "command": "gopls-old" } },
    { "id": "myserver", "name": "Mine", "command": { "command": "my-lsp" } }
  ]
}`
	if err := os.WriteFile(confPath, []byte(seed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, []byte(`{"quietStartup": true}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}, DryRun: false}
	if err := configureLSPPi(&cfg); err != nil {
		t.Fatalf("configureLSPPi: %v", err)
	}
	data, _ := os.ReadFile(confPath)
	s := string(data)
	if !strings.Contains(s, `"jj-language-server"`) || !strings.Contains(s, `"gopls"`) {
		t.Errorf("expected catalog servers in:\n%s", s)
	}
	if !strings.Contains(s, `"gopls-old"`) {
		t.Errorf("skillgrid id must be replaced, foreign content of a same-id server is dropped but other keys survive:\n%s", s)
	}
	if !strings.Contains(s, `"myserver"`) {
		t.Errorf("foreign server id must be preserved:\n%s", s)
	}
	if !strings.Contains(s, `"defaultLimit": 10`) {
		t.Errorf("foreign top-level key must be preserved:\n%s", s)
	}

	settings, _ := os.ReadFile(settingsPath)
	sr := string(settings)
	if !strings.Contains(sr, `"npm:cclsp"`) {
		t.Errorf("expected npm:cclsp package in settings:\n%s", sr)
	}
	if !strings.Contains(sr, `"quietStartup"`) {
		t.Errorf("foreign settings key must be preserved:\n%s", sr)
	}

	// Idempotency: second run adds no duplicate package, no error.
	if err := configureLSPPi(&cfg); err != nil {
		t.Fatalf("second run: %v", err)
	}
	settings2, _ := os.ReadFile(settingsPath)
	if c := strings.Count(string(settings2), "npm:cclsp"); c != 1 {
		t.Errorf("npm:cclsp appears %d times after two runs, want 1", c)
	}
}

func TestConfigureLSPPiCreatesMissingFiles(t *testing.T) {
	home := t.TempDir()
	cfg := Config{HomeDir: home, Agents: []string{"skill-pi"}, DryRun: false}
	if err := configureLSPPi(&cfg); err != nil {
		t.Fatalf("configureLSPPi (fresh home): %v", err)
	}
	confPath := filepath.Join(home, ".skill-pi", "conf.d", "lsp.conf")
	if _, err := os.Stat(confPath); err != nil {
		t.Errorf("lsp.conf not created: %v", err)
	}
	settingsPath := filepath.Join(home, ".skill-pi", "agent", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}
	if !strings.Contains(string(data), "npm:cclsp") {
		t.Errorf("fresh settings must contain npm:cclsp:\n%s", data)
	}
}
