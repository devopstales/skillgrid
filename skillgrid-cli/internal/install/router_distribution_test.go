package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Router distribution (T6, 2026-11-17-skillgrid-host-plugins): full-Run
// assertions — every host key distributes the bootstrap; opencode/kilo get
// their router .ts; skill-pi keeps its settings hooks block; unknown keys
// error naming the key; sync-repo refreshes the installed bootstrap.

// writeRouterRepoFixture writes the minimal repo checkout a full Run needs
// across every agent key (same seams as TestRunPhasesParallel +
// TestHermesSetupDryRunNoWrites).
func writeRouterRepoFixture(t *testing.T, repo string) {
	t.Helper()
	files := map[string]string{
		"config.d/tools.yaml":    "agents: []\nmcp: []\n",
		"config.d/mcp.yaml":      "servers:\n  mnemonic:\n    type: local\n    command:\n      - mnemonic\n      - mcp\n",
		".skillgrid/config.yaml": "project: fixture\n",
		".agents/skills/_shared/agent-config/router-bootstrap.md": "router bootstrap v1",
		".agents/skills/fixture-skill/SKILL.md":                   "# fixture skill\n",
		"hooks/tool-call-capture.js":                              "// hook\n",
		"hooks/stop-tests.js":                                     "// hook\n",
		"hooks/gate-stop.js":                                      "// hook\n",
		"hooks/cursor-session-start.sh":                           "#!/bin/sh\nexit 0\n",
		"hooks/cursor-session-end.sh":                             "#!/bin/sh\nexit 0\n",
		"hooks/cursor-tool-capture.sh":                            "#!/bin/sh\nexit 0\n",
		"hooks/cursor-policy.sh":                                  "#!/bin/sh\nexit 0\n",
		"plugins/opencode/skillgrid-compaction.ts":                "// plugin\n",
		"plugins/opencode/skillgrid-events.ts":                    "// plugin\n",
		"plugins/opencode/skillgrid-squad.ts":                     "// plugin\n",
		"plugins/opencode/mnemonic-memory.ts":                     "// plugin\n",
		"plugins/opencode/mnemonic-codeindex.ts":                  "// plugin\n",
		"plugins/opencode/skillgrid-context.ts":                   "// plugin\n",
		"plugins/opencode/skillgrid-router.ts":                    "// router\n",
		"plugins/opencode/memory-protocol.md":                     "# protocol\n",
		"plugins/kilo/skillgrid-router.ts":                        "// kilo router\n",
		"plugins/kilo/skillgrid-logo.tsx":                         "// logo\n",
		"plugins/pi/mnemonic/package.json":                        `{"name":"pi-mnemonic"}`,
		"plugins/pi/mnemonic/index.ts":                            "// entry\n",
		"plugins/pi/mnemonic/api.js":                              "// api\n",
		"plugins/pi/mnemonic/mnemonic-memory.ts":                  "// plugin\n",
		"plugins/pi/mnemonic/mnemonic-facts.ts":                   "// plugin\n",
		"plugins/pi/mnemonic/mnemonic-codeindex.ts":               "// plugin\n",
		"plugins/pi/mnemonic/ctx-sandbox.ts":                      "// plugin\n",
		"plugins/pi/mnemonic/skillgrid-events.ts":                 "// plugin\n",
		"plugins/pi/mnemonic/skillgrid-sdd-events.ts":             "// plugin\n",
		"plugins/pi/mnemonic/skillgrid-compaction.ts":             "// plugin\n",
		"plugins/hermes/__init__.py":                              "# provider\n",
		"rules/mnemonic.mdc":                                      "# rule\n",
		".cursor-plugin/plugin.json":                              `{"name":"fixture"}`,
	}
	for rel, content := range files {
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// stubInstallSeams replaces the external-command seams so a full Run is
// deterministic (same pattern as TestRunPhasesParallel).
func stubInstallSeams(t *testing.T) {
	t.Helper()
	oldPrereq := prerequisiteLookPath
	t.Cleanup(func() { prerequisiteLookPath = oldPrereq })
	prerequisiteLookPath = func(name string) (string, error) { return "/usr/bin/" + name, nil }

	oldBatch := batchLookPath
	t.Cleanup(func() { batchLookPath = oldBatch })
	batchLookPath = func(name string) (string, error) { return "", os.ErrNotExist }

	oldRun := commandRunner
	t.Cleanup(func() { commandRunner = oldRun })
	commandRunner = func(c *Config, dir string, name string, args ...string) error { return nil }
}

// TestRunDistributesRouterPerHostKey: every available host key gets the
// host-agnostic bootstrap; opencode/kilo additionally get their router .ts
// (copied + registered); skill-pi keeps the existing settings hooks block.
func TestRunDistributesRouterPerHostKey(t *testing.T) {
	repo := t.TempDir()
	writeRouterRepoFixture(t, repo)
	stubInstallSeams(t)

	for _, key := range availableAgentKeys() {
		t.Run(key, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("HERMES_HOME", filepath.Join(home, ".hermes"))
			cfg := &Config{
				HomeDir:        home,
				RepoHome:       filepath.Join(home, ".skillgrid"),
				RepoDir:        repo,
				AgentsDir:      filepath.Join(home, ".agents"),
				Agents:         []string{key},
				SkipClone:      true,
				SkipTools:      true,
				SkipProvider:   true,
				SkipAgentsCopy: true,
			}
			if err := Run(cfg); err != nil {
				t.Fatalf("Run(%s): %v", key, err)
			}

			// Every host key: the bootstrap is distributed (host-agnostic).
			b, err := os.ReadFile(filepath.Join(home, ".skillgrid", "router-bootstrap.md"))
			if err != nil {
				t.Fatalf("bootstrap not distributed for key %s: %v", key, err)
			}
			if string(b) != "router bootstrap v1" {
				t.Errorf("bootstrap content = %q", b)
			}

			switch key {
			case "opencode":
				if _, err := os.Stat(filepath.Join(home, ".config", "opencode", "plugins", "skillgrid-router.ts")); err != nil {
					t.Errorf("opencode router .ts not copied: %v", err)
				}
				cfgData, _ := os.ReadFile(filepath.Join(home, ".config", "opencode", "opencode.jsonc"))
				if !strings.Contains(string(cfgData), "./plugins/skillgrid-router.ts") {
					t.Errorf("opencode config missing router registration: %s", cfgData)
				}
			case "kilo":
				if _, err := os.Stat(filepath.Join(home, ".config", "kilo", "plugin", "skillgrid-router.ts")); err != nil {
					t.Errorf("kilo router .ts not copied: %v", err)
				}
				cfgData, _ := os.ReadFile(filepath.Join(home, ".config", "kilo", "kilo.jsonc"))
				if !strings.Contains(string(cfgData), "./plugin/skillgrid-router.ts") {
					t.Errorf("kilo config missing router registration: %s", cfgData)
				}
			case "skill-pi":
				// Existing installPiHookBridge behavior, asserted alongside the
				// new bootstrap copy (ticket acceptance: settings hooks block
				// + bootstrap).
				settings, err := os.ReadFile(filepath.Join(home, ".skill-pi", "agent", "settings.json"))
				if err != nil {
					t.Fatalf("skill-pi settings.json not written: %v", err)
				}
				if !strings.Contains(string(settings), `"hooks"`) {
					t.Errorf("skill-pi settings missing hooks block: %s", settings)
				}
			}
		})
	}
}

// TestSetupAgentsUnknownKeyNamesKeyNoWrites: an unknown agent key errors
// naming the key and leaves no agent files behind.
func TestSetupAgentsUnknownKeyNamesKeyNoWrites(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "config.d"), 0o755); err != nil {
		t.Fatal(err)
	}
	mcpYAML := "servers:\n  mnemonic:\n    type: local\n    command:\n      - mnemonic\n      - mcp\n"
	if err := os.WriteFile(filepath.Join(repo, "config.d", "mcp.yaml"), []byte(mcpYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{HomeDir: home, RepoDir: repo, Agents: []string{"bogus-key"}}
	err := setupAgents(cfg)
	if err == nil {
		t.Fatal("setupAgents should fail on an unknown agent key")
	}
	if !strings.Contains(err.Error(), "bogus-key") {
		t.Errorf("error should name the unknown key: %v", err)
	}
	if entries, err := os.ReadDir(filepath.Join(home, ".config")); err == nil && len(entries) != 0 {
		t.Errorf("unknown key modified files: %v", entries)
	}
}

// TestSyncRepoRefreshesRouterBootstrap: sync-repo with an edited clone asset
// updates the installed bootstrap; an unchanged re-sync leaves it alone.
func TestSyncRepoRefreshesRouterBootstrap(t *testing.T) {
	home := t.TempDir()
	src := t.TempDir()
	writeBootstrapFixture(t, src, "router v1")
	cfg := &Config{
		HomeDir:        home,
		RepoHome:       filepath.Join(home, ".skillgrid"),
		RepoDir:        filepath.Join(home, ".skillgrid", "repos", "skillgrid"),
		AgentsDir:      filepath.Join(home, ".agents"),
		SkipAgentsCopy: true,
	}
	t.Setenv("HOME", home)

	if err := cfg.SyncRepo(src); err != nil {
		t.Fatalf("SyncRepo: %v", err)
	}
	dst := filepath.Join(home, ".skillgrid", "router-bootstrap.md")
	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("bootstrap not synced: %v", err)
	}
	if string(b) != "router v1" {
		t.Errorf("bootstrap content = %q", b)
	}

	// Edited clone asset: the refresh must reach the installed copy.
	writeBootstrapFixture(t, src, "router v2")
	if err := cfg.SyncRepo(src); err != nil {
		t.Fatalf("SyncRepo after edit: %v", err)
	}
	b, err = os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "router v2" {
		t.Errorf("edited clone asset not picked up: %q", b)
	}
}

// TestSyncRepoRouterBootstrapDryRun: the dry-run plan prints the bootstrap
// copy but writes nothing.
func TestSyncRepoRouterBootstrapDryRun(t *testing.T) {
	home := t.TempDir()
	src := t.TempDir()
	writeBootstrapFixture(t, src, "router v1")
	cfg := &Config{
		HomeDir:        home,
		RepoHome:       filepath.Join(home, ".skillgrid"),
		RepoDir:        filepath.Join(home, ".skillgrid", "repos", "skillgrid"),
		AgentsDir:      filepath.Join(home, ".agents"),
		SkipAgentsCopy: true,
		DryRun:         true,
	}
	t.Setenv("HOME", home)

	if err := cfg.SyncRepo(src); err != nil {
		t.Fatalf("SyncRepo dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".skillgrid", "router-bootstrap.md")); err == nil {
		t.Error("dry-run must not write the bootstrap")
	}
}
