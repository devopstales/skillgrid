package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/devopstales/skillgrid/mnemonic/internal/setup"
)

// agentsConfig is the subset of .skillgrid/config.yaml that the preamble
// renderer consumes. Mirrors the onboarding config template's shape.
type agentsConfig struct {
	Project string        `yaml:"project"`
	Commands commandsCfg  `yaml:"commands"`
	Testing  testingCfg   `yaml:"testing"`
	Security securityCfg  `yaml:"security"`
	Agents   agentsFields `yaml:"agents"`
}

type commandsCfg struct {
	Build     string `yaml:"build"`
	Lint      string `yaml:"lint"`
	Format    string `yaml:"format"`
	Typecheck string `yaml:"typecheck"`
}

type testingCfg struct {
	Runner   string   `yaml:"runner"`
	Setup    string   `yaml:"setup"`
	Layers   []string `yaml:"layers"`
	Coverage string   `yaml:"coverage"`
	Mutation string   `yaml:"mutation"`
}

type securityCfg struct {
	Trivy trivyCfg `yaml:"trivy"`
}

type trivyCfg struct {
	Command    string `yaml:"command"`
	Severities string `yaml:"severities"`
	ScanTypes  string `yaml:"scan_types"`
}

type agentsFields struct {
	Directories             []dirEntry `yaml:"directories"`
	SecurityBoundaries      []string   `yaml:"security_boundaries"`
	DependencyPolicies      []string   `yaml:"dependency_policies"`
	ArchitectureConstraints []string   `yaml:"architecture_constraints"`
	DefinitionOfDone        []string   `yaml:"definition_of_done"`
	EngineeringStandards    []string   `yaml:"engineering_standards"`
}

type dirEntry struct {
	Path    string `yaml:"path"`
	Purpose string `yaml:"purpose"`
}

// agentsPreambleRel is the path of the preamble template inside the skill tree.
const agentsPreambleRel = ".agents/skills/lifecycle/onboarding/templates/agents-preamble.md"

// agentsPreamblePath is overridable in tests (mirrors blockMDPath).
var agentsPreamblePath = func(dir string) string {
	return filepath.Join(dir, agentsPreambleRel)
}

var preambleFenceRe = regexp.MustCompile("(?s)^```.*?\\n(.*?)\\n```")

// loadAgentsConfig reads .skillgrid/config.yaml into agentsConfig. A missing
// file yields a zero struct (not an error) so init can render <detect>
// placeholders without failing.
func loadAgentsConfig(dir string) (agentsConfig, error) {
	var cfg agentsConfig
	path := filepath.Join(dir, ".skillgrid", "config.yaml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// loadAgentsPreambleTemplate resolves the fenced preamble template from the
// skill tree (project tree first, then repo root). Mirrors loadSentinelTemplate.
func loadAgentsPreambleTemplate(dir string) (template, path string, err error) {
	for _, candidate := range []string{
		agentsPreamblePath(dir),
		filepath.Join(setup.FindRepoRoot(dir), agentsPreambleRel),
	} {
		data, rerr := os.ReadFile(candidate)
		if rerr != nil {
			continue
		}
		if m := preambleFenceRe.FindSubmatch(data); len(m) > 1 {
			return string(m[1]), candidate, nil
		}
		return strings.TrimSpace(string(data)), candidate, nil
	}
	return "", "", fmt.Errorf("agents-preamble template not found under %s", agentsPreambleRel)
}

// renderPreamble fills the template's {placeholders} from cfg. Empty command
// values fall back to <detect>; the Key Directories section is dropped when
// there are no directory rows.
func renderPreamble(cfg agentsConfig, template string) string {
	detect := func(v string) string {
		if strings.TrimSpace(v) == "" {
			return "<detect>"
		}
		return v
	}

	repl := map[string]string{
		"{language}":        detect(cfg.Commands.Build),
		"{package_manager}": detect(""),
		"{task_runner}":     detect(""),
		"{install}":         detect(cfg.Testing.Setup),
		"{test}":            detect(cfg.Testing.Runner),
		"{lint}":            detect(cfg.Commands.Lint),
		"{format}":          detect(cfg.Commands.Format),
		"{build}":           detect(cfg.Commands.Build),
		"{run}":             detect(""),
		"{docker}":          detect(""),
		"{security_scan}":   detect(cfg.Security.Trivy.Command),
	}

	out := template
	for ph, val := range repl {
		out = strings.ReplaceAll(out, ph, val)
	}

	// Key Directories: drop the whole section when there are no rows.
	if len(cfg.Agents.Directories) == 0 {
		out = dropSection(out, "## Key Directories")
	} else {
		table := "| Path | Purpose |\n|---|---|\n"
		for _, d := range cfg.Agents.Directories {
			table += fmt.Sprintf("| `%s` | %s |\n", d.Path, d.Purpose)
		}
		out = strings.ReplaceAll(out, "{directories_table}", strings.TrimRight(table, "\n"))
	}

	// Bullet-list sections: join as bullets, or a one-line default when empty.
	out = strings.ReplaceAll(out, "{engineering_standards}", bulletOr(cfg.Agents.EngineeringStandards, "engineering standards", ".skillgrid/ASSUMPTIONS.md"))
	out = strings.ReplaceAll(out, "{security_boundaries}", bulletOr(cfg.Agents.SecurityBoundaries, "security boundaries", ".skillgrid/ASSUMPTIONS.md"))
	out = strings.ReplaceAll(out, "{dependency_policies}", bulletOr(cfg.Agents.DependencyPolicies, "dependency policies", "the manifest"))
	out = strings.ReplaceAll(out, "{architecture_constraints}", bulletOr(cfg.Agents.ArchitectureConstraints, "architecture constraints", ".skillgrid/ASSUMPTIONS.md"))
	out = strings.ReplaceAll(out, "{definition_of_done}", bulletOr(cfg.Agents.DefinitionOfDone, "definition of done", "CI config"))

	// Issue tracker pointer (consistent with the sentinel block's tracker).
	out = strings.ReplaceAll(out, "{issue_tracker_line}", "Backlog.md — see the `## Skillgrid` Issue Tracker block for conventions.")

	return out
}

// bulletOr renders a list as `- ` bullets, or a one-line "no X yet" default.
func bulletOr(items []string, topic, where string) string {
	if len(items) == 0 {
		return fmt.Sprintf("No %s yet — see `%s`.", topic, where)
	}
	return strings.Join(prefixEach(items, "- "), "\n")
}

// prefixEach prefixes each item with prefix.
func prefixEach(items []string, prefix string) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = prefix + " " + it
	}
	return out
}

// dropSection removes a `## <heading>` section (heading + body up to the next
// `## ` heading or end of string) from out.
func dropSection(out, heading string) string {
	start := strings.Index(out, heading+"\n")
	if start == -1 {
		return out
	}
	rest := out[start+len(heading)+1:]
	next := strings.Index(rest, "\n## ")
	var end int
	if next == -1 {
		end = len(out)
	} else {
		end = start + len(heading) + 1 + next + 1
	}
	return out[:start] + out[end:]
}
