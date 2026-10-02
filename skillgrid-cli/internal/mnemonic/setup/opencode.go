package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"
)

// SetupOpenCode registers Mnemonic for OpenCode: the MCP servers from
// mcp.yaml plus the mnemonic.ts plugin. Harness config (TUI logo/theme,
// plugin-path append) is owned by internal/install (installAgentConfig), not
// the memory component. privateTools mirrors the config always-private
// allowlist into the harness env map (SKILLGRID_MNEMONIC_PRIVATE_TOOLS) so the
// plugin strips the same tools the Go config redacts.
func SetupOpenCode(home, repoRoot string, mcpEntries []MCPServerConfig, privateTools []string, dryRun bool) error {
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
	if err := upsertPrivateToolsEnv(cfgPath, privateTools, dryRun); err != nil {
		return err
	}
	if err := upsertPluginKey(cfgPath, "opencode-yaml-hooks", dryRun); err != nil {
		return err
	}
	hooksDst := filepath.Join(opencodeDir, "hook", "hooks.yaml")
	if err := copyFromRepo(repoRoot, opencodePluginRel, hooksDst, dryRun); err != nil {
		return err
	}
	checkpointDst := filepath.Join(opencodeDir, "plugin", "skillgrid-checkpoint.ts")
	if err := copyFromRepo(repoRoot, opencodeCheckpointPluginRel, checkpointDst, dryRun); err != nil {
		return err
	}
	return upsertPluginKey(cfgPath, "./plugin/skillgrid-checkpoint.ts", dryRun)
}

// upsertPrivateToolsEnv writes SKILLGRID_MNEMONIC_PRIVATE_TOOLS (comma-joined)
// into the opencode JSONC env map when the config allowlist is non-empty,
// keeping the Go config and the plugin env in sync. An empty allowlist leaves
// the config untouched (no env key is created).
func upsertPrivateToolsEnv(cfgPath string, privateTools []string, dryRun bool) error {
	if len(privateTools) == 0 {
		return nil
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	updated, err := sjson.SetBytes(data, "env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS", strings.Join(privateTools, ","))
	if err != nil {
		return fmt.Errorf("set env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS: %w", err)
	}
	if dryRun {
		logging.Info("[dry-run] set env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, updated, 0o644)
}
