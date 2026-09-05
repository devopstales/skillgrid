package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SetupOpenCode registers MCP from mcp.yaml, copies the mnemonic plugin, and
// applies the skillgrid-logo TUI plugin with tokyonight theme.
func SetupOpenCode(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	cfgPath := agentConfigPath(home, "opencode")
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
	logoDst := filepath.Join(opencodeDir, "tui-plugins", "skillgrid-logo.tsx")

	if err := copyFromRepo(repoRoot, opencodePluginRel, pluginDst, dryRun); err != nil {
		return err
	}
	if err := copyFromRepo(repoRoot, kiloPluginRel, sharedDst, dryRun); err != nil {
		return err
	}

	tuiJsonPath := filepath.Join(opencodeDir, "tui.json")
	if err := ensureConfigFile(tuiJsonPath, dryRun); err != nil {
		return err
	}
	if err := copyFromRepo(repoRoot, opencodeLogoRel, logoDst, dryRun); err != nil {
		return err
	}
	if err := setJSON(tuiJsonPath, "theme", "tokyonight", dryRun); err != nil {
		return err
	}
	if err := appendJSONArrayUnique(tuiJsonPath, "plugin", logoDst, dryRun); err != nil {
		return err
	}

	return nil
}

func setJSON(jsonPath, key, value string, dryRun bool) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", jsonPath, err)
	}
	updated, err := sjson.Set(string(data), key, value)
	if err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(jsonPath, []byte(updated), 0o644)
}

func appendJSONArrayUnique(jsonPath, key, value string, dryRun bool) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", jsonPath, err)
	}
	arr := gjson.Parse(string(data)).Get(key).Array()
	exists := false
	for _, v := range arr {
		if v.Str == value {
			exists = true
			break
		}
	}
	if exists {
		return nil
	}
	newArr := append(arr, gjson.Result{Str: value})
	updated, err := sjson.Set(string(data), key, newArr)
	if err != nil {
		return fmt.Errorf("append %s: %w", key, err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(jsonPath, []byte(updated), 0o644)
}
