package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aiskillgrid-cli/internal/config"
)

// AgentOption represents a selectable agent in the interactive UI
type AgentOption struct {
	ID         string
	Name       string
	ConfigPath string
}

// Agents returns the list of available agents for selection
func Agents() []AgentOption {
	return []AgentOption{
		{"kilo", "Kilo", "~/.config/kilo/kilo.jsonc"},
		{"opencode", "OpenCode", "~/.config/opencode/opencode.jsonc"},
	}
}

// Install runs the agent selection and installs MCP servers for selected agents
func Install() {
	agentOpts := Agents()

	fmt.Println("Available AI agents:")
	for i, a := range agentOpts {
		fmt.Printf("  [%d] %s\n", i+1, a.Name)
	}
	fmt.Println()

	selectedIndices := promptSelection(agentOpts)
	if len(selectedIndices) == 0 {
		fmt.Println("No agents selected.")
		return
	}

	selectedAgents := make([]AgentOption, 0, len(selectedIndices))
	for _, idx := range selectedIndices {
		if idx >= 0 && idx < len(agentOpts) {
			selectedAgents = append(selectedAgents, agentOpts[idx])
		}
	}

	mcpServers := buildMCPConfig()

	if err := config.PrecheckDependencies(mcpServers); err != nil {
		fmt.Printf("Warning: %v\n", err)
		fmt.Println("Continuing anyway...")
	}

	for _, agent := range selectedAgents {
		configPath := expandHome(agent.ConfigPath)

		var cfg map[string]interface{}
		data, err := os.ReadFile(configPath)
		if err == nil {
			json.Unmarshal(data, &cfg)
		}

		mcpKey := "mcp"
		currentMCPs, ok := cfg[mcpKey].(map[string]interface{})
		if !ok {
			currentMCPs = make(map[string]interface{})
		}

		for name, srv := range mcpServers {
			entry := map[string]interface{}{
				"type":    srv.Type,
				"enabled": true,
			}
			if srv.URL != "" {
				entry["url"] = srv.URL
			}
			if len(srv.Command) > 0 {
				entry["command"] = srv.Command
			}
			currentMCPs[name] = entry
			cfg[mcpKey] = currentMCPs
		}

		outData, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling %s config: %v\n", agent.Name, err)
			continue
		}

		configDir := filepath.Dir(configPath)
		os.MkdirAll(configDir, 0755)

		if err := os.WriteFile(configPath, outData, 0644); err != nil {
			fmt.Printf("Error writing %s config: %v\n", agent.Name, err)
			continue
		}

		fmt.Printf("\u2713 MCP servers installed in %s\n", configPath)
	}
}

func promptSelection(agents []AgentOption) []int {
	var reader = bufio.NewReader(os.Stdin)
	fmt.Print("Select agents to configure by entering numbers (comma-separated, e.g. '1,2'): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return nil
	}

	var selected []int
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if n, err := strconv.Atoi(part); err == nil && n > 0 && n <= len(agents) {
			selected = append(selected, n-1)
		}
	}

	return selected
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home := os.Getenv("HOME")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
		path = filepath.Join(home, path[1:])
	}
	return path
}

func buildMCPConfig() map[string]*config.McpServer {
	out := make(map[string]*config.McpServer)

	configs := []struct {
		name    string
		mcpType string
		url     string
		command []string
	}{
		{"context7-http", "remote", "https://mcp.context7.com/mcp", nil},
		{"deepwiki-http", "remote", "https://mcp.deepwiki.com/mcp", nil},
		{"exa-http", "remote", "https://mcp.exa.ai/mcp", nil},
		{"engram", "local", "", []string{"engram", "mcp"}},
		{"ccc", "local", "", []string{"ccc", "mcp"}},
		{"gitnexus", "local", "", []string{"npx", "-y", "gitnexus@1.3.11", "mcp"}},
		{"trivy", "local", "", []string{"trivy", "mcp"}},
	}

	for _, c := range configs {
		out[c.name] = &config.McpServer{
			Type:    c.mcpType,
			URL:     c.url,
			Command: c.command,
			Enabled: true,
		}
	}

	return out
}
