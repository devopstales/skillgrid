package install

import (
	"fmt"
	"os"
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
func GlobalTools() []Tool {
	return []Tool{
		{Name: "skills", NPM: "skills"},
		{Name: "openspec", NPM: "@fission-ai/openspec@latest"},
	}
}
