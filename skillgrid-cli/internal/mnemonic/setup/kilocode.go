package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

// SetupKiloCode registers MCP, writes the AGENTS.md protocol block, and bridges OpenCode plugin files.
func SetupKiloCode(home, repoRoot string, dryRun bool) error {
	cfgPath := agentConfigPath(home, "kilo")
	if err := ensureConfigFile(cfgPath, dryRun); err != nil {
		return err
	}
	if err := upsertOpenCodeMCP(cfgPath, dryRun); err != nil {
		return err
	}

	protocol := ProtocolMarkdownFromRepo(repoRoot)
	if protocol == "" {
		return fmt.Errorf("memory protocol not found (sync repo or run from checkout)")
	}

	agentsPath := filepath.Join(home, ".config", "kilo", "AGENTS.md")
	var content []byte
	if data, err := os.ReadFile(agentsPath); err == nil {
		content = data
	}
	updated := upsertMarkerBlock(string(content), kiloBeginMarker, kiloEndMarker, protocol)
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(agentsPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(agentsPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", agentsPath, err)
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	kiloDir := filepath.Join(home, ".config", "kilo")
	bridges := []struct{ src, dst string }{
		{
			filepath.Join(opencodeDir, "plugins", "mnemonic.ts"),
			filepath.Join(kiloDir, "plugins", "mnemonic.ts"),
		},
		{
			filepath.Join(opencodeDir, "shared", "http-client.ts"),
			filepath.Join(kiloDir, "shared", "http-client.ts"),
		},
		{
			filepath.Join(opencodeDir, "tui.json"),
			filepath.Join(kiloDir, "tui.json"),
		},
	}
	for _, b := range bridges {
		if err := copyFirstWriteWins(b.src, b.dst, dryRun); err != nil {
			return err
		}
	}
	return nil
}
