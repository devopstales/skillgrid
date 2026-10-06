package setup

import (
	"fmt"
	"os"
	"path/filepath"

)

// SetupKiloCode registers Mnemonic for Kilo: the MCP servers, the AGENTS.md
// memory-protocol block, opencode-yaml-hooks, and skillgrid-checkpoint.ts.
// Retired plugin entries (opencode-command-hooks, mnemonic.ts) are removed.
// Harness config (TUI logo/theme, kilo→opencode bridges) is owned by
// internal/install (installAgentConfig), not the memory component.
// privateTools mirrors the config always-private allowlist into the harness
// env map (SKILLGRID_MNEMONIC_PRIVATE_TOOLS) so capture strips the same
// tools the Go config redacts.
func SetupKiloCode(home, repoRoot string, mcpEntries []MCPServerConfig, privateTools []string, dryRun bool) error {
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
	if err := upsertPrivateToolsEnv(cfgPath, privateTools, dryRun); err != nil {
		return err
	}
	if err := dropRetiredPlugins(cfgPath, dryRun); err != nil {
		return err
	}
	if err := upsertPluginKey(cfgPath, "opencode-yaml-hooks", dryRun); err != nil {
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
		logInfo("[dry-run] write " + agentsPath)
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
	hooksDst := filepath.Join(kiloDir, "hook", "hooks.yaml")
	if err := copyFromRepo(repoRoot, kiloPluginRel, hooksDst, dryRun); err != nil {
		return err
	}
	if err := installOpenCodeHookScripts(home, repoRoot, dryRun); err != nil {
		return err
	}
	checkpointDst := filepath.Join(kiloDir, "plugin", "skillgrid-checkpoint.ts")
	if err := copyFromRepo(repoRoot, kiloCheckpointPluginRel, checkpointDst, dryRun); err != nil {
		return err
	}
	return upsertPluginKey(cfgPath, "./plugin/skillgrid-checkpoint.ts", dryRun)
}
