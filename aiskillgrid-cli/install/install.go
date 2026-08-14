package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aiskillgrid/aiskillgrid/agents"
	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/sync"
	"github.com/aiskillgrid/aiskillgrid/tools"
)

type Options struct {
	Scope      home.Scope
	Agents     []string
	ProjectDir string
	Yes        bool
	SkipSync   bool
	RepoURL    string // override; empty uses config
}

func Run(opts Options) error {
	root, err := home.Root()
	if err != nil {
		return err
	}
	paths := home.Resolve(root)
	if err := home.EnsureLayout(paths); err != nil {
		return err
	}
	cfg, err := home.LoadConfig(paths.ConfigFile)
	if err != nil {
		return err
	}
	repoURL := opts.RepoURL
	if repoURL == "" {
		repoURL = cfg.RepoURL
	}

	scope := opts.Scope
	agentNames := opts.Agents
	if !opts.Yes {
		if scope == "" {
			var scopeAns string
			prompt := &survey.Select{
				Message: "Install scope:",
				Options: []string{string(home.ScopeGlobal), string(home.ScopeProject)},
				Default: cfg.DefaultScope,
			}
			if err := survey.AskOne(prompt, &scopeAns); err != nil {
				return err
			}
			scope = home.Scope(scopeAns)
		}
		if len(agentNames) == 0 {
			prompt := &survey.MultiSelect{
				Message: "Which clients should Skillgrid install to? (v1: kilo, opencode, cursor, copilot)",
				Options: []string{"kilo", "opencode", "cursor", "copilot"},
				Default: cfg.DefaultAgents,
			}
			if err := survey.AskOne(prompt, &agentNames); err != nil {
				return err
			}
		}
	}
	if scope == "" {
		scope = home.Scope(cfg.DefaultScope)
		if scope == "" {
			scope = home.ScopeGlobal
		}
	}
	if len(agentNames) == 0 {
		return fmt.Errorf("no agents selected")
	}

	projectDir := opts.ProjectDir
	if projectDir == "" {
		projectDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	var rev string
	if !opts.SkipSync {
		res, err := sync.Sync(paths.ToolsDir, repoURL)
		if err != nil {
			return fmt.Errorf("sync: %w", err)
		}
		rev = res.Rev
		_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("sync ok rev=%s", rev))
	} else if short, err := sync.RevParse(paths.ToolsDir); err == nil {
		rev = short
	}

	packRoot := paths.ToolsDir
	// Prefer local checkout packs when developing from this repo and tools not synced.
	if _, err := os.Stat(filepath.Join(packRoot, "packs", "skills")); err != nil {
		if wd, err2 := os.Getwd(); err2 == nil {
			cand := findRepoRoot(wd)
			if cand != "" {
				packRoot = cand
			}
		}
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir, err := home.UserConfigDir()
	if err != nil {
		return err
	}

	selected, err := agents.ByNames(agentNames)
	if err != nil {
		return err
	}

	// Run install phase to get resolved MCP servers
	phase, err := tools.RunInstallPhase(paths, packRoot, tools.PhaseOptions{})
	if err != nil {
		return err
	}
	for _, w := range phase.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		_ = home.AppendLog(paths.LogsDir, "warn: "+w)
	}

	ctx := agents.Context{
		Scope:       scope,
		ProjectDir:  projectDir,
		HomeRoot:    userHome,
		ConfigDir:   configDir,
		PackRoot:    packRoot,
		ResolvedMCP: phase.Servers,
	}

	state := home.State{
		Scope:        string(scope),
		ProjectDir:   "",
		Agents:       agentNames,
		RepoURL:      repoURL,
		RepoRev:      rev,
		WrittenPaths: map[string][]string{},
	}
	if scope == home.ScopeProject {
		state.ProjectDir = projectDir
	}

	var failed []string
	for _, a := range selected {
		res, err := a.Install(ctx)
		if err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", a.Name(), err))
			_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("agent %s failed: %v", a.Name(), err))
			continue
		}
		state.WrittenPaths[a.Name()] = res.Written
		_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("agent %s ok paths=%d", a.Name(), len(res.Written)))
		fmt.Printf("Installed for %s (%d paths)\n", a.Name(), len(res.Written))
	}

	if err := home.SaveState(paths.StateFile, state); err != nil {
		return err
	}

	if len(failed) > 0 {
		return fmt.Errorf("some agents failed:\n  %s", strings.Join(failed, "\n  "))
	}
	fmt.Println("Install complete.")
	return nil
}

func findRepoRoot(start string) string {
	dir := start
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "packs", "skills")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
