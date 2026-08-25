package config

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// McpServer defines an MCP server entry for kila and opencode configs
type McpServer struct {
	Type    string   `json:"type"`
	URL     string   `json:"url,omitempty"`
	Command []string `json:"command,omitempty"`
	Enabled bool     `json:"enabled"`
}

// PartialKiloConfig is the subset of kila.jsonc we modify
type PartialKiloConfig struct {
	MCP map[string]*McpServer `json:"mcp,omitempty"`
}

// PartialOpenCodeConfig is the subset of opencode.jsonc we modify
type PartialOpenCodeConfig struct {
	MCP map[string]*McpServer `json:"mcp,omitempty"`
}

func writeJSON(configPath string, mcp map[string]*McpServer) error {
	data, err := json.MarshalIndent(mcp, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// WriteKiloMerges writes MCP tools into ~/.config/kilo/kila.jsonc under the mcp key
func WriteKiloMerges(tools map[string]*McpServer) error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "kilo", "kilo.jsonc")

	var cfg PartialKiloConfig
	// Read existing config if present
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &cfg)
	} else {
		cfg.MCP = make(map[string]*McpServer)
	}

	for name, server := range tools {
		cfg.MCP[name] = server
	}

	if err := writeJSON(configPath, cfg.MCP); err != nil {
		return err
	}

	fmt.Printf("MCP servers written to %s\n", configPath)
	return nil
}

// WriteOpenCodeMerges writes MCP tools into ~/.config/opencode/opencode.jsonc under the mcp key
func WriteOpenCodeMerges(tools map[string]*McpServer) error {
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "opencode", "opencode.jsonc")

	var cfg PartialOpenCodeConfig
	// Read existing config if present
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &cfg)
	} else {
		cfg.MCP = make(map[string]*McpServer)
	}

	for name, server := range tools {
		cfg.MCP[name] = server
	}

	if err := writeJSON(configPath, cfg.MCP); err != nil {
		return err
	}

	fmt.Printf("MCP servers written to %s\n", configPath)
	return nil
}

// PrecheckDependencies checks that local MCP tools exist on the system
func PrecheckDependencies(tools map[string]*McpServer) error {
	var missing []string

	for name, mcp := range tools {
		if mcp.Type == "local" && len(mcp.Command) > 0 {
			if _, err := exec.LookPath(mcp.Command[0]); err != nil {
				missing = append(missing, name+" ("+strings.Join(mcp.Command, " ")+")")
			}
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("local tools not found:\n%s", strings.Join(missing, "\n"))
	}

	return nil
}
