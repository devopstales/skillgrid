package setup

import (
	"fmt"
	"os"
	"path/filepath"

	jsonc "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SetupOpenCode copies the Mnemonic plugin and registers MCP + plugin path in OpenCode config.
func SetupOpenCode(home, repoRoot string, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	pluginDst := filepath.Join(opencodeDir, "plugins", "mnemonic.ts")
	sharedDst := filepath.Join(opencodeDir, "shared", "http-client.ts")

	if err := copyFromRepo(repoRoot, pluginRelPath, pluginDst, dryRun); err != nil {
		return err
	}
	if err := copyFromRepo(repoRoot, httpClientRelPath, sharedDst, dryRun); err != nil {
		return err
	}

	cfgPath := agentConfigPath(home, "opencode")
	if err := ensureConfigFile(cfgPath, dryRun); err != nil {
		return err
	}
	if err := upsertOpenCodeMCP(cfgPath, dryRun); err != nil {
		return err
	}

	pluginRef := tildePath(home, pluginDst)
	return appendPluginPath(cfgPath, pluginRef, dryRun)
}

func upsertOpenCodeMCP(cfgPath string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	path := "mcp." + mcpServerName
	if jsonc.Get(string(data), path).Exists() {
		entry := mnemonicMCPEntry()
		updated, err := sjson.Set(string(data), path, entry)
		if err != nil {
			return fmt.Errorf("upsert mcp: %w", err)
		}
		if dryRun {
			return nil
		}
		return os.WriteFile(cfgPath, []byte(updated), 0o644)
	}
	updated, err := sjson.Set(string(data), path, mnemonicMCPEntry())
	if err != nil {
		return fmt.Errorf("upsert mcp: %w", err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

func appendPluginPath(cfgPath, pluginPath string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	existing := []string{}
	for _, v := range jsonc.Get(string(data), "plugin").Array() {
		existing = append(existing, v.String())
	}
	for _, have := range existing {
		if have == pluginPath {
			return nil
		}
	}
	existing = append(existing, pluginPath)
	updated, err := sjson.Set(string(data), "plugin", existing)
	if err != nil {
		return fmt.Errorf("set plugin: %w", err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}
