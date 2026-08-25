package mcp

import (
	"skillgrid-cli/internal/config"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadRegistry(configDir string) (map[string]*config.McpServer, error) {
	path := filepath.Join(configDir, "mcp.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mcp config: %w", err)
	}
	var mcpCfg config.MCPConfig
	if err := yaml.Unmarshal(data, &mcpCfg); err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}
	out := make(map[string]*config.McpServer)
	for name, s := range mcpCfg.Servers {
		out[name] = &config.McpServer{
			Type:    s.Type,
			URL:     s.URL,
			Command: s.Command,
			Enabled: true,
		}
	}
	return out, nil
}

func PrecheckDependencies(servers map[string]*config.McpServer) []string {
	var missing []string
	for name, srv := range servers {
		if srv.Type == "local" && len(srv.Command) > 0 {
			if _, err := exec.LookPath(srv.Command[0]); err != nil {
				missing = append(missing, name+" ("+strings.Join(srv.Command, " ")+")")
			}
		}
	}
	return missing
}
