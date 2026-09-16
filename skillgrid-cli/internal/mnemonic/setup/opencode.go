package setup

import (
	"fmt"
	"path/filepath"
)

// SetupOpenCode registers Mnemonic for OpenCode: the MCP servers from
// mcp.yaml plus the mnemonic.ts plugin. Harness config (TUI logo/theme,
// plugin-path append) is owned by internal/install (installAgentConfig), not
// the memory component.
func SetupOpenCode(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	cfgPath := AgentConfigPath(home, "opencode")
	if err := ensureConfigFile(cfgPath, dryRun); err != nil {
		return err
	}
	if err := backupConfigFile(home, "opencode", cfgPath, dryRun); err != nil {
		return err
	}
	for _, entry := range mcpEntries {
		if err := upsertOpenCodeMCP(cfgPath, entry, dryRun); err != nil {
			return err
		}
	}
	pluginDst := filepath.Join(opencodeDir, "plugins", "mnemonic.ts")
	sharedDst := filepath.Join(opencodeDir, "shared", "http-client.ts")

	if err := copyFromRepo(repoRoot, opencodePluginRel, pluginDst, dryRun); err != nil {
		return err
	}
	return copyFromRepo(repoRoot, kiloPluginRel, sharedDst, dryRun)
}
