package smoke

import (
	"skillgrid-cli/internal/logging"
	"os"
	"path/filepath"
	"testing"
)

func TestDryRunSmoke(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)
	os.Setenv("USERPROFILE", tmpHome)

	configDir := filepath.Join(tmpHome, ".skillgrid", "config.d")
	os.MkdirAll(configDir, 0755)
	os.WriteFile(filepath.Join(configDir, "tools.yaml"), []byte("agents:\n  - \"@kilocode/cli\"\ntools:\n  - \"vercel-labs/skills\"\n"), 0644)
	os.WriteFile(filepath.Join(configDir, "mcp.yaml"), []byte("servers:\n  context7-http:\n    type: remote\n    url: https://mcp.context7.com/mcp\n"), 0644)

	os.MkdirAll(filepath.Join(tmpHome, ".config", "kilo"), 0755)
	os.WriteFile(filepath.Join(tmpHome, ".config", "kilo", "kilo.jsonc"), []byte(`{"mcp":{}}`), 0644)

	logging.ResetForTest()
	if err := logging.Init(tmpHome); err != nil {
		t.Fatalf("logging init failed: %v", err)
	}
	logging.Info("smoke test")

	logPath := filepath.Join(tmpHome, "logs", "install.log")
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("log file not created: %v", err)
	}
}
