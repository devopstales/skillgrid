package setup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Router plugin distribution (T6, 2026-11-17-skillgrid-host-plugins):
// per-agent copy-set entries — opencode loads plugins/opencode/<base>.ts,
// kilo loads its own plugins/kilo/<base>.ts variant.

// writeNativePluginFixture writes the native plugin sources (from
// plugins/opencode/) plus extra repo-relative files.
func writeNativePluginFixture(t *testing.T, repo string, extra map[string]string) {
	t.Helper()
	for _, base := range nativePluginBases {
		rel := "plugins/opencode/" + base + ".ts"
		if err := os.MkdirAll(filepath.Join(repo, filepath.Dir(rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, rel), []byte("// "+base+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for rel, content := range extra {
		p := filepath.Join(repo, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TestInstallOpenCodeNativePluginsSkipsMissingRouter: an older checkout with
// no router source installs without it (skip, no error, no registration).
func TestInstallOpenCodeNativePluginsSkipsMissingRouter(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	writeNativePluginFixture(t, repo, nil)
	cfgDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "opencode.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installOpenCodeNativePlugins(home, repo, cfgPath, false); err != nil {
		t.Fatalf("installOpenCodeNativePlugins: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugins", "skillgrid-router.ts")); err == nil {
		t.Error("router .ts must not be copied when the source is missing")
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(cfgData), "skillgrid-router") {
		t.Errorf("config must not register a router that was not copied: %s", cfgData)
	}
}

// TestInstallNativePluginsSkipsMissingKiloRouter: same contract for the kilo
// variant.
func TestInstallNativePluginsSkipsMissingKiloRouter(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	writeNativePluginFixture(t, repo, nil)
	cfgDir := filepath.Join(home, ".config", "kilo")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "kilo.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installNativePlugins(home, repo, cfgPath, "kilo", false); err != nil {
		t.Fatalf("installNativePlugins: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugin", "skillgrid-router.ts")); err == nil {
		t.Error("kilo router .ts must not be copied when the source is missing")
	}
}

// TestInstallOpenCodeNativePluginsCopiesRouter: the opencode copy set gains
// the standalone router plugin (copied into the v2 discovery dir + registered
// in the config).
func TestInstallOpenCodeNativePluginsCopiesRouter(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	writeNativePluginFixture(t, repo, map[string]string{
		"plugins/opencode/skillgrid-router.ts": "// router\n",
	})
	cfgDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "opencode.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installOpenCodeNativePlugins(home, repo, cfgPath, false); err != nil {
		t.Fatalf("installOpenCodeNativePlugins: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugins", "skillgrid-router.ts")); err != nil {
		t.Errorf("opencode router .ts not copied: %v", err)
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "./plugins/skillgrid-router.ts") {
		t.Errorf("opencode config missing router registration: %s", cfgData)
	}
}

// TestInstallNativePluginsCopiesKiloRouter: the kilo copy set gains the
// kilo-variant router plugin (copied from plugins/kilo/ + registered in the
// kilo config).
func TestInstallNativePluginsCopiesKiloRouter(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	writeNativePluginFixture(t, repo, map[string]string{
		"plugins/kilo/skillgrid-router.ts": "// kilo router\n",
	})
	cfgDir := filepath.Join(home, ".config", "kilo")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "kilo.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installNativePlugins(home, repo, cfgPath, "kilo", false); err != nil {
		t.Fatalf("installNativePlugins: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugin", "skillgrid-router.ts")); err != nil {
		t.Errorf("kilo router .ts not copied: %v", err)
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "./plugin/skillgrid-router.ts") {
		t.Errorf("kilo config missing router registration: %s", cfgData)
	}
}

// TestInstallOpenCodeNativePluginsRouterSurvivesMissingNatives: the router is
// distributed BEFORE the native copy set (T6 follow-up), so a checkout whose
// native sources are missing (the release/2 state after d5148778) still gets
// the router plugin + registration; the missing native sources are skipped,
// not fatal (ADR-0067 reworks that set).
func TestInstallOpenCodeNativePluginsRouterSurvivesMissingNatives(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	// Only the router source — no native sources.
	srcDir := filepath.Join(repo, "plugins", "opencode")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, routerPluginRel+".ts"),
		[]byte("// router\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "opencode.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installOpenCodeNativePlugins(home, repo, cfgPath, false); err != nil {
		t.Fatalf("missing native sources are skipped, not fatal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugins", routerPluginRel+".ts")); err != nil {
		t.Errorf("router .ts must land before the native loop: %v", err)
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "./plugins/"+routerPluginRel+".ts") {
		t.Errorf("config must register the router when native sources are skipped: %s", cfgData)
	}
}

// TestInstallNativePluginsKiloRouterSurvivesMissingNatives: same contract for
// the kilo variant.
func TestInstallNativePluginsKiloRouterSurvivesMissingNatives(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	srcDir := filepath.Join(repo, "plugins", "kilo")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, routerPluginRel+".ts"),
		[]byte("// kilo router\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgDir := filepath.Join(home, ".config", "kilo")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "kilo.jsonc")
	if err := os.WriteFile(cfgPath, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installNativePlugins(home, repo, cfgPath, "kilo", false); err != nil {
		t.Fatalf("missing native sources are skipped, not fatal: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "plugin", routerPluginRel+".ts")); err != nil {
		t.Errorf("kilo router .ts must land before the native loop: %v", err)
	}
	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfgData), "./plugin/"+routerPluginRel+".ts") {
		t.Errorf("kilo config must register the router when native sources are skipped: %s", cfgData)
	}
}
