package setup

import (
	"fmt"
	"os"
	"path/filepath"

	jsonc "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// SetupOpenCode copies the Mnemonic plugin and registers MCP + plugin path in OpenCode config.
func SetupOpenCode(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	pluginDst := filepath.Join(opencodeDir, "plugins", "mnemonic.ts")
	sharedDst := filepath.Join(opencodeDir, "shared", "http-client.ts")

	if err := copyFromRepo(repoRoot, opencodePluginRel, pluginDst, dryRun); err != nil {
		return err
	}
	if err := copyFromRepo(repoRoot, kiloPluginRel, sharedDst, dryRun); err != nil {
		return err
	}

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

	pluginRef := tildePath(home, pluginDst)
	return appendPluginPath(cfgPath, pluginRef, dryRun)
}

func upsertOpenCodeMCP(cfgPath string, entry MCPServerConfig, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	path := "mcp." + entry.Name
	var mcpEntry map[string]interface{}
	if entry.Type == "remote" {
		mcpEntry = map[string]interface{}{
			"type": "remote",
			"url":  entry.URL,
		}
	} else {
		cmd := make([]interface{}, len(entry.Command))
		for i, c := range entry.Command {
			cmd[i] = c
		}
		mcpEntry = map[string]interface{}{
			"type":    "local",
			"command": cmd,
			"enabled": true,
		}
	}
	updated, err := sjson.Set(string(data), path, mcpEntry)
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
