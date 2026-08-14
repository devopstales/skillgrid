// Package plugins installs harness-native plugins (not skills-only copies).
package plugins

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/mcpmerge"
	"github.com/aiskillgrid/aiskillgrid/sync"
)

const (
	SuperpowersRepoURL      = "https://github.com/obra/superpowers.git"
	SuperpowersPluginSpec   = "superpowers@git+https://github.com/obra/superpowers.git"
	SuperpowersMarketplace  = "obra/superpowers-marketplace"
	SuperpowersCopilotSpec  = "superpowers@superpowers-marketplace"
	SuperpowersCheckoutName = "superpowers"
)

// Options control Superpowers plugin install for selected agents.
type Options struct {
	Agents     []string
	Scope      home.Scope
	ProjectDir string
	HomeRoot   string
	ConfigDir  string
	DepsDir    string
}

// Result summarizes plugin install outcomes.
type Result struct {
	Checkout string
	Rev      string
	Written  map[string][]string
	Warnings []string
}

// InstallSuperpowers clones/updates obra/superpowers under DepsDir and installs
// it as a native plugin for each selected agent (not merely as skills via skills add).
func InstallSuperpowers(opts Options) (Result, error) {
	out := Result{Written: map[string][]string{}}
	if opts.DepsDir == "" {
		return out, fmt.Errorf("DepsDir is required")
	}
	checkout := filepath.Join(opts.DepsDir, SuperpowersCheckoutName)
	res, err := sync.Sync(checkout, SuperpowersRepoURL)
	if err != nil {
		return out, fmt.Errorf("superpowers checkout: %w", err)
	}
	out.Checkout = res.Path
	out.Rev = res.Rev

	for _, name := range opts.Agents {
		paths, warns, err := installForAgent(name, opts, checkout)
		out.Warnings = append(out.Warnings, warns...)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if len(paths) > 0 {
			out.Written[name] = paths
		}
	}
	return out, nil
}

func installForAgent(name string, opts Options, checkout string) ([]string, []string, error) {
	switch name {
	case "opencode":
		return installOpenCodePlugin(opts)
	case "kilo":
		return installKiloPlugin(opts)
	case "cursor":
		return installCursorPlugin(opts.HomeRoot, checkout)
	case "copilot":
		return installCopilotPlugin()
	default:
		return nil, []string{fmt.Sprintf("superpowers plugin: unknown agent %q skipped", name)}, nil
	}
}

func installOpenCodePlugin(opts Options) ([]string, []string, error) {
	path := openCodeConfigPath(opts)
	if err := ensurePluginEntry(path, SuperpowersPluginSpec); err != nil {
		return nil, nil, err
	}
	return []string{path}, nil, nil
}

func openCodeConfigPath(opts Options) string {
	if opts.Scope == home.ScopeProject {
		return filepath.Join(opts.ProjectDir, "opencode.json")
	}
	return filepath.Join(opts.ConfigDir, "opencode", "opencode.json")
}

func installKiloPlugin(opts Options) ([]string, []string, error) {
	var warns []string
	// Prefer native CLI when available (writes plugin into kilo config).
	if bin, err := exec.LookPath("kilo"); err == nil {
		args := []string{"plugin", "install", SuperpowersPluginSpec}
		if opts.Scope == home.ScopeGlobal {
			args = append(args, "--global")
		}
		cmd := exec.Command(bin, args...)
		if opts.Scope == home.ScopeProject {
			cmd.Dir = opts.ProjectDir
		}
		if out, err := cmd.CombinedOutput(); err != nil {
			warns = append(warns, fmt.Sprintf("kilo plugin install failed (%v); merging config instead: %s", err, strings.TrimSpace(string(out))))
		} else {
			cfg := kiloConfigPath(opts)
			return []string{cfg}, warns, nil
		}
	} else {
		warns = append(warns, "kilo CLI not on PATH; merging Superpowers into kilo.jsonc plugin array")
	}
	path := kiloConfigPath(opts)
	if err := ensurePluginEntry(path, SuperpowersPluginSpec); err != nil {
		return nil, warns, err
	}
	return []string{path}, warns, nil
}

func kiloConfigPath(opts Options) string {
	if opts.Scope == home.ScopeProject {
		return filepath.Join(opts.ProjectDir, ".kilo", "kilo.jsonc")
	}
	return filepath.Join(opts.ConfigDir, "kilo", "kilo.jsonc")
}

func installCursorPlugin(homeRoot, checkout string) ([]string, []string, error) {
	if homeRoot == "" {
		return nil, nil, fmt.Errorf("HomeRoot is required for cursor plugin install")
	}
	dst := filepath.Join(homeRoot, ".cursor", "plugins", "local", "superpowers")
	if err := os.RemoveAll(dst); err != nil {
		return nil, nil, err
	}
	if err := copyDirSkipGit(checkout, dst); err != nil {
		return nil, nil, err
	}
	warn := "cursor: Superpowers installed under ~/.cursor/plugins/local/superpowers — reload Cursor (Developer: Reload Window) if skills do not appear; marketplace /add-plugin remains an alternative"
	return []string{dst}, []string{warn}, nil
}

func installCopilotPlugin() ([]string, []string, error) {
	bin, err := exec.LookPath("copilot")
	if err != nil {
		return nil, []string{
			"copilot: Copilot CLI not on PATH — install GitHub Copilot CLI then re-run, or run: copilot plugin marketplace add obra/superpowers-marketplace && copilot plugin install superpowers@superpowers-marketplace",
		}, nil
	}
	if out, err := exec.Command(bin, "plugin", "marketplace", "add", SuperpowersMarketplace).CombinedOutput(); err != nil {
		msg := strings.TrimSpace(string(out))
		// Marketplace may already be registered; continue to install.
		if !strings.Contains(strings.ToLower(msg), "already") {
			return nil, []string{fmt.Sprintf("copilot marketplace add: %v (%s)", err, msg)}, nil
		}
	}
	if out, err := exec.Command(bin, "plugin", "install", SuperpowersCopilotSpec).CombinedOutput(); err != nil {
		return nil, []string{fmt.Sprintf("copilot plugin install failed: %v (%s)", err, strings.TrimSpace(string(out)))}, nil
	}
	return []string{"copilot:" + SuperpowersCopilotSpec}, nil, nil
}

// ensurePluginEntry merges entry into the "plugin" string array of a JSON/JSONC config.
func ensurePluginEntry(path, entry string) error {
	root, err := mcpmerge.LoadOrEmpty(path)
	if err != nil {
		return err
	}
	arr, _ := root["plugin"].([]any)
	name := pluginName(entry)
	var next []any
	found := false
	for _, v := range arr {
		s, ok := v.(string)
		if !ok {
			next = append(next, v)
			continue
		}
		if pluginName(s) == name {
			if !found {
				next = append(next, entry)
				found = true
			}
			continue
		}
		next = append(next, s)
	}
	if !found {
		next = append(next, entry)
	}
	root["plugin"] = next
	return mcpmerge.WriteObject(path, root)
}

func pluginName(spec string) string {
	if strings.HasPrefix(spec, "file://") || strings.HasPrefix(spec, "/") || strings.HasPrefix(spec, "~") {
		return filepath.Base(strings.TrimSuffix(spec, string(filepath.Separator)))
	}
	if i := strings.Index(spec, "@"); i > 0 {
		return spec[:i]
	}
	return spec
}

func copyDirSkipGit(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
