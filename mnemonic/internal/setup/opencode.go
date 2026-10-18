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

// installNativePlugins copies the native TS assets from plugins/opencode/ into
// ~/.config/<harness>/plugins/ and ~/.config/<harness>/tools/ (shared by the
// OpenCode and Kilo installers per ADR-0032 — Kilo copies from the same opencode
// source dir).
//
// It no longer registers a "plugin" config array: local plugins in
// ~/.config/<harness>/plugins/ are auto-discovered by OpenCode, and the 21
// custom tools in tools/ are auto-discovered the same way. Registering
// "./plugin/<name>.ts" in the config array pointed at the wrong (singular)
// path and double-registered each plugin, so it was dropped (see TASK-082).
//
// The set copied: the 5 plugin sources, the shared v2 SSE helper
// (skillgrid-v2sse.ts, imported by the compaction/events plugins), and every
// tool file under plugins/opencode/tools/ (one file per tool; filename = tool
// name for the OpenCode tools/ loader).
func installNativePlugins(home, repoRoot, cfgPath, harness string, dryRun bool) error {
	pluginDir := filepath.Join(home, ".config", harness, "plugins")
	toolDir := filepath.Join(home, ".config", harness, "tools")

	for _, base := range nativePluginBases {
		srcRel := "plugins/opencode/" + base + ".ts"
		dst := filepath.Join(pluginDir, base+".ts")
		if err := copyFromRepo(repoRoot, srcRel, dst, dryRun); err != nil {
			return err
		}
	}

	// Shared v2 SSE helper (the compaction + events plugins import it for their
	// v2 `setup` hook wiring).
	if err := copyFromRepo(repoRoot, "plugins/opencode/skillgrid-v2sse.ts",
		filepath.Join(pluginDir, "skillgrid-v2sse.ts"), dryRun); err != nil {
		return err
	}

	// The custom tools (one file per tool). The repo source dir is
	// plugins/opencode/tools/; the installed dir is ~/.config/<harness>/tools/.
	if err := copyToolDir(repoRoot, "plugins/opencode/tools", toolDir, dryRun); err != nil {
		return err
	}
	return nil
}

// copyToolDir copies every *.ts file in the repo tools/ source dir into the
// installed tools/ dir, creating the destination. A missing source dir is not
// an error (the tools are optional; an older checkout without them still
// installs the plugins).
func copyToolDir(repoRoot, srcRel, dstDir string, dryRun bool) error {
	srcDir := filepath.Join(repoRoot, srcRel)
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read tools dir %s: %w", srcDir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ts") {
			continue
		}
		srcRelFile := srcRel + "/" + e.Name()
		dst := filepath.Join(dstDir, e.Name())
		if err := copyFromRepo(repoRoot, srcRelFile, dst, dryRun); err != nil {
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
