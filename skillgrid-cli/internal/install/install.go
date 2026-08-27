package install

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/ui"
)

// Run executes the install flow in the order defined by the project spec:
//  1. create ~/.skillgrid structure
//  2. sync the skillgrid repository (clone or pull)
//  3. verify node + npm are on PATH
//  4. ask which agents to install (or use preset/--yes defaults)
//  5. npm install -g each selected agent
//  6. npm install -g for the shared tools
//  7. override ~/.agents from the repo's .agents/
//
// Any hard failure stops the run with a descriptive error; individual
// non-fatal issues are reported but the run continues.
func Run(c *Config) error {
	start := time.Now()

	info := func(s string) { Out("  •", s) }
	verb := func(s string, a ...any) { VerboseOut(c, append([]any{s}, a...)...) }

	info("creating ~/.skillgrid structure")
	if err := ensureHomeStruct(c); err != nil {
		return err
	}

	if c.SkipClone {
		verb("skipping repo sync (--skip-clone)")
		if _, err := os.Stat(c.RepoDir); err != nil {
			return fmt.Errorf("--skip-clone set but %s does not exist", c.RepoDir)
		}
	} else {
		info("syncing repo into ~/.skillgrid/repos/skillgrid")
		if err := syncRepo(c); err != nil {
			return err
		}
	}

	info("checking node + npm")
	if err := checkNode(c); err != nil {
		return err
	}

	info("selecting agents")
	if err := selectAgents(c); err != nil {
		return err
	}

	if len(c.Agents) > 0 {
		info("installing agents: " + strings.Join(c.Agents, ", "))
		if err := installAgents(c); err != nil {
			return err
		}
	}

	if c.SkipTools {
		verb("skipping global tool install (--skip-tools)")
	} else {
		info("installing global tools (skills, openspec)")
		if err := installTools(c); err != nil {
			return err
		}
	}

	if c.SkipAgentsCopy {
		verb("skipping ~/.agents copy (--skip-agents)")
	} else {
		info("copying repo .agents/ → ~/.agents/")
		if err := copyAgents(c); err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "\n  done in %s (home: %s)\n", time.Since(start).Round(time.Millisecond), c.RepoHome)
	if c.DryRun {
		fmt.Fprintln(os.Stderr, "  no changes were written (dry run)")
	}
	return nil
}

// --- steps ---

func ensureHomeStruct(c *Config) error {
	dirs := []string{c.RepoHome, filepath.Dir(c.RepoDir), c.RepoDir}
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			continue
		}
		if c.DryRun {
			Out("      [dry-run] mkdir -p", d)
			continue
		}
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func syncRepo(c *Config) error {
	if _, err := os.Stat(filepath.Join(c.RepoDir, ".git")); err == nil {
		if c.DryRun {
			Out("      [dry-run] git pull --ff-only in", c.RepoDir)
			return nil
		}
		Out("      git pull --ff-only in", c.RepoDir)
		return run(c, c.RepoDir, "git", "pull", "--ff-only")
	}

	if c.DryRun {
		Out("      [dry-run] git clone --branch", c.Branch, c.RepoURL, "→", c.RepoDir)
		return nil
	}
	Out("      git clone --branch", c.Branch, c.RepoURL, "→", c.RepoDir)
	return run(c, "", "git", "clone", "--branch", c.Branch, c.RepoURL, c.RepoDir)
}

func checkNode(c *Config) error {
	var missing []string
	for _, bin := range []string{"node", "npm"} {
		if _, err := exec.LookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) > 0 {
		script := filepath.Join(c.RepoDir, "scripts", "install_node.sh")
		fmt.Fprintln(os.Stderr,
			"\n      missing:", strings.Join(missing, ", "))
		fmt.Fprintln(os.Stderr, "      use the install script from the repo:")
		fmt.Fprintln(os.Stderr, "         bash", script)
		fmt.Fprintln(os.Stderr, "      then re-run: skillgrid install")
		return fmt.Errorf("missing from PATH: %s", strings.Join(missing, ", "))
	}
	verbNode(c)
	return nil
}

func verbNode(c *Config) {
	v, err := exec.Command("node", "--version").Output()
	if err != nil {
		return
	}
	n, err := exec.Command("npm", "--version").Output()
	if err != nil {
		return
	}
	VerboseOut(c, "node "+strings.TrimSpace(string(v))+" / npm "+strings.TrimSpace(string(n)))
}

func selectAgents(c *Config) error {
	if len(c.Agents) > 0 {
		return nil
	}
	if c.Yes {
		c.Agents = []string{"opencode", "kilo", "cursor"}
		VerboseOut(c, "default selection (yes mode): opencode, kilo, cursor")
		return nil
	}
	if !ui.Interactive() {
		c.Agents = []string{"opencode", "kilo"}
		VerboseOut(c, "non-interactive default: opencode, kilo")
		return nil
	}

	agents := AvailableAgents()
	opts := make([]ui.Option, len(agents))
	for i, a := range agents {
		opts[i] = ui.Option{
			Label: a.Name,
			Value: a.Key,
		}
	}
	selected, _, err := ui.MultiSelect("Install skillgrid for which agents? (a=all, q=cancel)", opts)
	if err != nil {
		return err
	}
	if len(selected) == 0 {
		VerboseOut(c, "no agents selected — skipping agent install")
		return nil
	}
	c.Agents = selected
	return nil
}

func installAgents(c *Config) error {
	agents := AvailableAgents()
	for _, key := range c.Agents {
		for _, a := range agents {
			if a.Key != key || a.NPM == "" {
				continue
			}
			pkg := a.NPM
			if c.DryRun {
				Out("      [dry-run] npm install -g", pkg)
				continue
			}
			Out("      npm install -g", pkg)
			if err := run(c, "", "npm", "install", "-g", pkg); err != nil {
				return fmt.Errorf("npm install -g %s: %w", pkg, err)
			}
		}
	}
	return nil
}

func installTools(c *Config) error {
	for _, t := range GlobalTools() {
		if c.DryRun {
			Out("      [dry-run] npm install -g", t.NPM)
			continue
		}
		Out("      npm install -g", t.NPM)
		if err := run(c, "", "npm", "install", "-g", t.NPM); err != nil {
			return fmt.Errorf("npm install -g %s: %w", t.NPM, err)
		}
	}
	return nil
}

func copyAgents(c *Config) error {
	src := filepath.Join(c.RepoDir, ".agents")
	if _, err := os.Stat(src); err != nil {
		VerboseOut(c, "no .agents/ in repo — skipping copy (source missing)")
		return nil
	}
	if c.DryRun {
		Out("      [dry-run] copy", src, "→", c.AgentsDir)
		return nil
	}
	if err := copyAll(src, c.AgentsDir); err != nil {
		return err
	}
	return nil
}

// --- helpers ---

func run(c *Config, dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	// Always capture output so we can surface it on failure; only echo it when verbose
	// or when the command fails, to keep normal runs readable.
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if buf.Len() > 0 {
		out := strings.TrimSpace(buf.String())
		if err != nil {
			Out("      " + out)
		} else if c.Verbose {
			Out("      " + out)
		}
	}
	return err
}

// copyAll recursively copies src to dst.
func copyAll(src, dst string) error {
	os.MkdirAll(dst, 0o755)
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		os.MkdirAll(filepath.Dir(target), 0o755)
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
