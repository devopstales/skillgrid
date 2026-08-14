package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func ResolveMCPServers(packPath string, p home.Paths, present map[string]bool) (servers map[string]any, warnings []string, err error) {
	data, err := os.ReadFile(packPath)
	if err != nil {
		return nil, nil, fmt.Errorf("read pack: %w", err)
	}

	var pack struct {
		MCPServers map[string]any `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &pack); err != nil {
		return nil, nil, fmt.Errorf("parse pack: %w", err)
	}

	// Build replacement map. MCP commands point at absolute managed executables;
	// the managed prefix has no npx of its own to shell out through.
	repl := map[string]string{
		"{{AISKILLGRID_NPM}}":        p.NpmDir,
		"{{AISKILLGRID_NPM_CACHE}}":  p.NpmCacheDir,
		"{{AISKILLGRID_BIN}}":        p.DepsBinDir,
		"{{AISKILLGRID_ENGRAM}}":     filepath.Join(p.DepsBinDir, "engram"),
		"{{AISKILLGRID_GITNEXUS}}":   ManagedBinOrDefault(p, "gitnexus"),
		"{{AISKILLGRID_BACKLOG}}":    ManagedBinOrDefault(p, "backlog"),
		"{{AISKILLGRID_OPENSPEC}}":   ManagedBinOrDefault(p, "openspec"),
		"{{AISKILLGRID_CONTEXT7}}":   ManagedBinOrDefault(p, "context7-mcp"),
		"{{AISKILLGRID_PLAYWRIGHT}}": ManagedBinOrDefault(p, "playwright-mcp", "mcp-server-playwright"),
	}

	servers = make(map[string]any)
	warnings = []string{}

	for name, val := range pack.MCPServers {
		entry, ok := val.(map[string]any)
		if !ok {
			continue
		}

		// Check requires
		if req, ok := entry["requires"].(string); ok {
			if !present[req] {
				warnings = append(warnings, fmt.Sprintf("skipped %s: missing %s", name, req))
				continue
			}
			// Strip requires from the entry
			delete(entry, "requires")
		}

		// Substitute placeholders in all string fields recursively
		entry = substituteStrings(entry, repl).(map[string]any)

		// A resolved absolute command that is missing would make the agent fail
		// to start the server, so drop the entry instead of writing it out.
		if cmd, ok := entry["command"].(string); ok && filepath.IsAbs(cmd) && !fileExists(cmd) {
			warnings = append(warnings, fmt.Sprintf("skipped %s: command not found: %s", name, cmd))
			continue
		}

		servers[name] = entry
	}

	return servers, warnings, nil
}

func substituteStrings(val any, repl map[string]string) any {
	switch v := val.(type) {
	case string:
		result := v
		for old, new := range repl {
			result = strings.ReplaceAll(result, old, new)
		}
		return result
	case map[string]any:
		m := make(map[string]any)
		for k, subVal := range v {
			m[k] = substituteStrings(subVal, repl)
		}
		return m
	case []any:
		arr := make([]any, len(v))
		for i, subVal := range v {
			arr[i] = substituteStrings(subVal, repl)
		}
		return arr
	default:
		return v
	}
}
