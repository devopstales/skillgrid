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

	// Build replacement map
	repl := map[string]string{
		"{{AISKILLGRID_NPM}}":       p.NpmDir,
		"{{AISKILLGRID_NPM_CACHE}}": p.NpmCacheDir,
		"{{AISKILLGRID_BIN}}":       p.DepsBinDir,
		"{{AISKILLGRID_NPX}}":       filepath.Join(p.NpmBinDir, "npx"),
		"{{AISKILLGRID_ENGRAM}}":    filepath.Join(p.DepsBinDir, "engram"),
		"{{AISKILLGRID_GITNEXUS}}":  ManagedBin(p, "gitnexus"),
		"{{AISKILLGRID_BACKLOG}}":   ManagedBin(p, "backlog"),
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
