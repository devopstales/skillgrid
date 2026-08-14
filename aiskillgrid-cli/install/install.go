package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aiskillgrid/aiskillgrid/agents"
	"github.com/aiskillgrid/aiskillgrid/githooks"
	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/plugins"
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

	if hookPath, err := githooks.InstallCommitMsgHook(projectDir, packRoot); err != nil {
		failed = append(failed, fmt.Sprintf("git-hook: %v", err))
		_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("git-hook failed: %v", err))
	} else if hookPath != "" {
		state.WrittenPaths["git-hooks"] = []string{hookPath}
		_ = home.AppendLog(paths.LogsDir, "git-hook commit-msg ok path="+hookPath)
		fmt.Printf("Installed git commit-msg hook (strip AI co-authors): %s\n", hookPath)
	}

	pluginRes, err := plugins.InstallSuperpowers(plugins.Options{
		Agents:     agentNames,
		Scope:      scope,
		ProjectDir: projectDir,
		HomeRoot:   userHome,
		ConfigDir:  configDir,
		DepsDir:    paths.DepsDir,
	})
	if err != nil {
		failed = append(failed, fmt.Sprintf("superpowers-plugin: %v", err))
		_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("superpowers plugin failed: %v", err))
	} else {
		recordPluginResult(&state, paths.LogsDir, "superpowers", pluginRes)
	}

	karpathyRes, err := plugins.InstallKarpathyGuidelines(plugins.Options{
		Agents:     agentNames,
		Scope:      scope,
		ProjectDir: projectDir,
		HomeRoot:   userHome,
		ConfigDir:  configDir,
		DepsDir:    paths.DepsDir,
	})
	if err != nil {
		failed = append(failed, fmt.Sprintf("karpathy-guidelines: %v", err))
		_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("karpathy guidelines failed: %v", err))
	} else {
		recordPluginResult(&state, paths.LogsDir, "karpathy", karpathyRes)
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

func recordPluginResult(state *home.State, logsDir, label string, res plugins.Result) {
	labelTitle := label
	switch label {
	case "superpowers":
		labelTitle = "Superpowers plugin"
	case "karpathy":
		labelTitle = "Karpathy guidelines"
	}
	for agent, wpaths := range res.Written {
		key := "plugin-" + label + "-" + agent
		state.WrittenPaths[key] = wpaths
		fmt.Printf("Installed %s for %s (%d paths)\n", labelTitle, agent, len(wpaths))
		_ = home.AppendLog(logsDir, fmt.Sprintf("%s %s ok rev=%s", label, agent, res.Rev))
	}
	for _, w := range res.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
		_ = home.AppendLog(logsDir, "warn: "+w)
	}
	if res.Checkout != "" {
		state.WrittenPaths["plugin-"+label+"-checkout"] = []string{res.Checkout}
	}
}
