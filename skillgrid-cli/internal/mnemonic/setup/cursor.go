package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/sjson"
)

// SetupCursor registers the Mnemonic MCP server and writes the always-applied rule.
func SetupCursor(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		repoRoot = FindRepoRoot("")
	}
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	mcpPath := filepath.Join(home, ".cursor", "mcp.json")
	if err := ensureConfigFile(mcpPath, dryRun); err != nil {
		return err
	}
	if err := backupConfigFile(home, "cursor", mcpPath, dryRun); err != nil {
		return err
	}
	for _, entry := range mcpEntries {
		if err := upsertCursorMCP(mcpPath, entry, dryRun); err != nil {
			return err
		}
	}

	protocol := ProtocolMarkdownFromRepo(repoRoot)
	if protocol == "" {
		return fmt.Errorf("memory protocol not found")
	}

	templatePath := filepath.Join(repoRoot, cursorTemplateRel)
	template, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("read cursor template: %w", err)
	}
	body := strings.ReplaceAll(string(template), "{{MEMORY_PROTOCOL}}", protocol)

	rulePath := filepath.Join(home, ".cursor", "rules", "mnemonic.mdc")
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(rulePath, []byte(body), 0o644)
}

func upsertCursorMCP(mcpPath string, entry MCPServerConfig, dryRun bool) error {
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", mcpPath, err)
	}
	path := "mcpServers." + entry.Name
	var mcpEntry map[string]interface{}
	if entry.Type == "remote" {
		mcpEntry = map[string]interface{}{
			"url": entry.URL,
		}
	} else {
		var args []interface{}
		if len(entry.Command) > 1 {
			args = make([]interface{}, len(entry.Command)-1)
			for i, c := range entry.Command[1:] {
				args[i] = c
			}
		}
		mcpEntry = map[string]interface{}{
			"command": entry.Command[0],
			"args":    args,
		}
	}
	updated, err := sjson.Set(string(data), path, mcpEntry)
	if err != nil {
		return fmt.Errorf("upsert cursor mcp: %w", err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(mcpPath, []byte(updated), 0o644)
}
