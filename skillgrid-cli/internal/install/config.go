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

	// Provider setup (ADR-0023, TICKET-03): which LLM/embedder host to wire
	// into the home config. Provider is "local" (Ollama) or "external"
	// (OpenAI-compatible); "" defaults to local. LLMBaseURL/LLMApiKey feed the
	// external mode (flag or env); they are ignored in local mode.
	Provider     string
	SkipProvider bool
	LLMBaseURL   string
	LLMApiKey    string
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
	Bin  string // PATH binary; empty means no npm CLI (e.g. Cursor app)
	Hint string
}

// AvailableAgents returns the installable agents (first cut: opencode, kilo, cursor).
func AvailableAgents() []Agent {
	return []Agent{
		{Key: "opencode", Name: "OpenCode", NPM: "opencode-ai", Bin: "opencode", Hint: "npm: opencode-ai"},
		{Key: "kilo", Name: "Kilo", NPM: "@kilocode/cli", Bin: "kilo", Hint: "npm: @kilocode/cli"},
		{Key: "cursor", Name: "Cursor", NPM: "", Bin: "", Hint: "app-side only"},
	}
}

type Tool struct {
	Name string
	NPM  string
	Bin  string // PATH binary name; empty means no binary check
}

// GlobalTools returns the global npm tools installed regardless of agent selection.
// install-mcp is excluded here because it is installed separately via installInstallMcp()
// before MCP server packages, so it never reaches installTools().
func GlobalTools() []Tool {
	return []Tool{
		{Name: "skills", NPM: "skills", Bin: "skills"},
		{Name: "cucumber", NPM: "@cucumber/cucumber", Bin: "cucumber-js"},
		{Name: "backlog.md", NPM: "backlog.md", Bin: "backlog"},
		{Name: "jscpd", NPM: "jscpd", Bin: "jscpd"},
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

// SecTool describes a security scanning tool the installer can bring up. It is
// installed with its own manager (not npm), so it sits beside GlobalTools.
type SecTool struct {
	Name    string
	Manager string // "uv" or "go"
	// InstallArgs are the exact arguments passed to the manager, after its name.
	// The manager binary is prepended by the caller.
	InstallArgs []string
	// Bin is the resulting PATH binary; when present and found, the install is skipped.
	Bin string
	// Hint is the manual fallback shown when the manager is missing from PATH.
	Hint string
}

// SecurityTools returns the security scanners installed with their own
// managers, run after the npm tools.
func SecurityTools() []SecTool {
	return []SecTool{
		{Name: "wapiti3", Manager: "uv", InstallArgs: []string{"tool", "install", "wapiti3"}, Bin: "wapiti3", Hint: "curl -LsSf https://astral.sh/uv/install.sh | sh"},
		{Name: "akca", Manager: "go", InstallArgs: []string{"install", "github.com/akha-security/akca/engine/cmd/akca@latest"}, Bin: "akca", Hint: "install Go, then: " + "go install github.com/akha-security/akca/engine/cmd/akca@latest"},
		{Name: "nuclei", Manager: "go", InstallArgs: []string{"install", "-v", "github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest"}, Bin: "nuclei", Hint: "install Go, then: " + "go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest"},
		{Name: "trivy", Manager: "go", InstallArgs: []string{"install", "github.com/aquasecurity/trivy/cmd/trivy@latest"}, Bin: "trivy", Hint: "install Go, then: " + "go install github.com/aquasecurity/trivy/cmd/trivy@latest"},
	}
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
