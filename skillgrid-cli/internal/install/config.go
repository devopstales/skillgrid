package install

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultRepoURL = "https://github.com/devopstales/skillgrid"
	DefaultBranch  = "release/2"
)

// Config holds all state for a run.
type Config struct {
	Version string
	DryRun  bool
	Verbose bool
	Yes     bool
	Agents  []string // preset agent keys (from --agents); empty = prompt at runtime

	HomeDir   string
	RepoHome  string
	RepoDir   string
	AgentsDir string

	RepoURL        string
	Branch         string
	SkipClone      bool
	SkipTools      bool
	SkipAgentsCopy bool
}

// Out writes an install log line to stderr (keeps stdout clean for scripts).
func Out(v ...any) {
	fmt.Fprintln(os.Stderr, v...)
}

// VerboseOut prints when --verbose is set; in dry-run it always prefixes the line.
func VerboseOut(c *Config, v ...any) {
	if c.DryRun {
		v = append([]any{"[dry-run]"}, v...)
	} else if !c.Verbose {
		return
	}
	Out(v...)
}

type Agent struct {
	Key  string
	Name string
	NPM  string
	Hint string
}

// AvailableAgents returns the installable agents (first cut: opencode, kilo, cursor).
func AvailableAgents() []Agent {
	return []Agent{
		{Key: "opencode", Name: "OpenCode", NPM: "opencode-ai", Hint: "npm: opencode-ai"},
		{Key: "kilo", Name: "Kilo", NPM: "@kilocode/cli", Hint: "npm: @kilocode/cli"},
		{Key: "cursor", Name: "Cursor", NPM: "", Hint: "app-side only"},
	}
}

type Tool struct {
	Name string
	NPM  string
}

// GlobalTools returns the global npm tools installed regardless of agent selection.
// install-mcp is excluded here because it is installed separately via installInstallMcp()
// before MCP servers, so it never reaches installTools().
func GlobalTools() []Tool {
	return []Tool{
		{Name: "skills", NPM: "skills"},
		{Name: "openspec", NPM: "@fission-ai/openspec@latest"},
		{Name: "cucumber", NPM: "@cucumber/cucumber"},
	}
}

type ToolsConfig struct {
	Agents []string `yaml:"agents"`
	MCP    []string `yaml:"mcp"`
}

type MCPConfig struct {
	Servers map[string]MCPServer `yaml:"servers"`
}

type MCPServer struct {
	Type    string   `yaml:"type"`
	URL     string   `yaml:"url"`
	Command []string `yaml:"command"`
}

func LoadToolsConfig(repoDir string) (*ToolsConfig, error) {
	path := filepath.Join(repoDir, "config.d", "tools.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read tools config: %w", err)
	}
	var cfg ToolsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse tools config: %w", err)
	}
	return &cfg, nil
}

func LoadMCPConfig(repoDir string) (*MCPConfig, error) {
	path := filepath.Join(repoDir, "config.d", "mcp.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mcp config: %w", err)
	}
	var cfg MCPConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}
	return &cfg, nil
}
