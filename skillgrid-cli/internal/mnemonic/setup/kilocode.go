package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"
)

// SetupKiloCode registers Mnemonic for Kilo: the MCP servers, the AGENTS.md
// memory-protocol block, and the mnemonic.ts plugin. Harness config (TUI
// logo/theme, plugin-path append, kilo→opencode bridges) is owned by
// internal/install (installAgentConfig), not the memory component.
func SetupKiloCode(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	cfgPath := AgentConfigPath(home, "kilo")
	if err := ensureConfigFile(cfgPath, dryRun); err != nil {
		return err
	}
	if err := backupConfigFile(home, "kilo", cfgPath, dryRun); err != nil {
		return err
	}
	for _, entry := range mcpEntries {
		if err := upsertOpenCodeMCP(cfgPath, entry, dryRun); err != nil {
			return err
		}
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
		logging.Info("[dry-run] write " + agentsPath)
	} else {
		if err := backupConfigFile(home, "kilo", agentsPath, dryRun); err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(agentsPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(agentsPath, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", agentsPath, err)
		}
	}

	kiloDir := filepath.Join(home, ".config", "kilo")
	pluginDst := filepath.Join(kiloDir, "plugins", "mnemonic.ts")
	sharedDst := filepath.Join(kiloDir, "shared", "http-client.ts")

	if err := copyFromRepo(repoRoot, kiloPluginRel, pluginDst, dryRun); err != nil {
		return err
	}
	return copyFromRepo(repoRoot, opencodePluginRel, sharedDst, dryRun)
}
