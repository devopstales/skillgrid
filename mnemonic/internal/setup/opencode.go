package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"
)

// SetupOpenCode registers Mnemonic for OpenCode: the MCP servers from
// mcp.yaml plus the 5 native TS plugins (ADR-0032) copied from
// plugins/opencode/ and registered in the harness config. Retired plugin
// entries (opencode-command-hooks, mnemonic.ts, opencode-yaml-hooks,
// skillgrid-checkpoint.ts) are removed. The shell-era hooks.yaml and
// opencode-*.sh adapters are no longer installed. Harness config
// (TUI logo/theme) is owned by internal/install (installAgentConfig), not the
// memory component. privateTools mirrors the config always-private allowlist
// into the harness env map (SKILLGRID_MNEMONIC_PRIVATE_TOOLS) so capture
// strips the same tools the Go config redacts.
func SetupOpenCode(home, repoRoot string, mcpEntries []MCPServerConfig, privateTools []string, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

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
	if err := dropRetiredPlugins(cfgPath, dryRun); err != nil {
		return err
	}
	if err := installOpenCodeHookScripts(home, repoRoot, dryRun); err != nil {
		return err
	}
	return installNativePlugins(home, repoRoot, cfgPath, "opencode", dryRun)
}

// installNativePlugins copies the 5 native TS plugins from plugins/opencode/
// into ~/.config/<harness>/plugin/ and registers each "./plugin/<name>.ts" in
// the harness config (shared by the OpenCode and Kilo installers per
// ADR-0032 — Kilo copies from the same opencode source dir).
func installNativePlugins(home, repoRoot, cfgPath, harness string, dryRun bool) error {
	pluginDir := filepath.Join(home, ".config", harness, "plugin")
	for _, base := range nativePluginBases {
		srcRel := "plugins/opencode/" + base + ".ts"
		dst := filepath.Join(pluginDir, base+".ts")
		if err := copyFromRepo(repoRoot, srcRel, dst, dryRun); err != nil {
			return err
		}
		if err := upsertPluginKey(cfgPath, "./plugin/"+base+".ts", dryRun); err != nil {
			return err
		}
	}
	return nil
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
		logInfo("[dry-run] set env.SKILLGRID_MNEMONIC_PRIVATE_TOOLS in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, updated, 0o644)
}
