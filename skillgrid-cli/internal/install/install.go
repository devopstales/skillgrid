package install

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/devopstales/skillgrid/mnemonic"
)

const installMcpPkg = "install-mcp"

// Run executes the install flow in the order defined by the project spec:
//  1. validate prerequisites (node, npm, uv, go) — stop before any writes
//  2. create ~/.skillgrid structure
//  3. sync the skillgrid repository (clone or pull)
//  4. ask which agents to install (or use preset/--yes defaults)
//  5. npm install -g each selected agent (skip if binary already on PATH)
//  6. npm install -g install-mcp, then npm install -g each MCP package from tools.yaml
//  7. configure selected agents (plugins + mcp.yaml merge) — sole writer of agent MCP config
//  8. npm install -g for the remaining shared tools
//  9. install security scanners (uv tool install wapiti3; go install akca, nuclei)
//
// 10. override ~/.agents from the repo's .agents/
//
// Any hard failure stops the run with a descriptive error; individual
// non-fatal issues are reported but the run continues.
func Run(c *Config) error {
	start := time.Now()

	info := func(s string) { Out("  •", s) }
	verb := func(s string, a ...any) { VerboseOut(c, append([]any{s}, a...)...) }

	info("checking prerequisites (node, npm, uv, go)")
	if err := checkPrerequisites(c); err != nil {
		return err
	}

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

	info("mirroring .agents, docs, git-hooks, hooks to ~/.skillgrid")
	if err := syncMirrorDirs(c); err != nil {
		return err
	}
	info("wiring git hooks (core.hooksPath)")
	if err := wireGitHooks(c); err != nil {
		Out("  warning: git hooks wiring:", err)
	}

	info("selecting agents")
	if err := selectAgents(c); err != nil {
		return err
	}

	// Fan out the independent install phases in parallel (capped at 4).
	// Phase A: agents (npm batch → setup → config → personas → LSP config)
	// Phase B: security tools (go/uv, parallel within)
	// Phase C: LSP tools (go/npm/cargo, parallel within) — only if --lsp
	// Phase D: provider setup (ollama) — non-fatal
	//
	// Phase A's npm batch must complete before setupAgents (needs binaries on
	// PATH), but A is independent of B, C, D. runParallel caps total
	// concurrency at 4 to avoid disk thrash.

	var phases []func() error

	// Phase A: agents.
	if len(c.Agents) > 0 {
		phases = append(phases, func() error {
			// Hermes prerequisite: non-fatal.
			if agentSelected(c, "hermes") {
				if _, err := prerequisiteLookPath("hermes"); err != nil {
					Out("      warn", "hermes not on PATH — uv tool install hermes-agent; continuing")
				}
			}
			info("installing npm packages (batched)")
			if err := installNPMBatch(c); err != nil {
				return err
			}
			info("configuring agents (memory): " + strings.Join(c.Agents, ", "))
			if err := setupAgents(c); err != nil {
				return err
			}
			if err := backupAgentConfigs(c); err != nil {
				return fmt.Errorf("backup agent configs: %w", err)
			}
			info("configuring agents (harness): " + strings.Join(c.Agents, ", "))
			for _, key := range c.Agents {
				if err := installAgentConfig(c, key); err != nil {
					return fmt.Errorf("harness config %s: %w", key, err)
				}
				if key == "skill-pi" {
					if err := installPiHookBridge(c); err != nil {
						return fmt.Errorf("harness config %s: %w", key, err)
					}
				}
			}
			if !c.SkipAgentsCopy {
				info("installing personas: " + strings.Join(c.Agents, ", "))
				for _, key := range c.Agents {
					if err := installPersonas(c, key); err != nil {
						return fmt.Errorf("install personas %s: %w", key, err)
					}
				}
			}
			if c.LSP {
				info("configuring LSP for agents: " + strings.Join(c.Agents, ", "))
				if err := configureLSP(c); err != nil {
					return err
				}
			}
			return nil
		})
	}

	// Phase B + C: security tools + LSP tools.
	if !c.SkipTools {
		phases = append(phases, func() error {
			info("installing security tools (wapiti3, akca, nuclei)")
			if err := installSecurityTools(c); err != nil {
				return err
			}
			if c.LSP {
				info("installing LSP servers (gopls, pyright, cclsp, typescript, rust-analyzer, jj-language-server)")
				if err := installLSPTools(c); err != nil {
					return err
				}
			} else {
				verb("skipping LSP server install (use --lsp)")
			}
			return nil
		})
	} else {
		verb("skipping global tool install (--skip-tools)")
	}

	// Phase D: provider setup (non-fatal).
	if !c.SkipProvider {
		phases = append(phases, func() error {
			info("setting up LLM/embedder provider (" + providerLabel(c) + ")")
			if err := setupProvider(c); err != nil {
				Out("  warning: provider setup:", err)
				return nil // non-fatal
			}
			return nil
		})
	} else {
		verb("skipping provider setup (--skip-provider)")
	}

	// Run all phases in parallel, capped at 4 concurrent.
	if err := runParallel(4, phases); err != nil {
		return err
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

// prerequisiteLookPath is exec.LookPath, replaced in tests.
var prerequisiteLookPath = exec.LookPath

// checkPrerequisites fails the install before any write when node, npm, uv,
// or go is not on PATH. Later steps need all four: npm for agents and tools,
// uv and go for the security scanners.
func checkPrerequisites(c *Config) error {
	var missing []string
	for _, bin := range []string{"node", "npm", "uv", "go"} {
		if _, err := prerequisiteLookPath(bin); err != nil {
			missing = append(missing, bin)
		}
	}
	if len(missing) == 0 {
		verbPrereqs(c)
		return nil
	}
	fmt.Fprintln(os.Stderr, "\n      missing:", strings.Join(missing, ", "))
	if prerequisiteMissing(missing, "node") || prerequisiteMissing(missing, "npm") {
		script := filepath.Join(c.RepoDir, "scripts", "install_node.sh")
		fmt.Fprintln(os.Stderr, "      node + npm: bash", script)
	}
	if prerequisiteMissing(missing, "uv") {
		fmt.Fprintln(os.Stderr, "      uv: curl -LsSf https://astral.sh/uv/install.sh | sh")
	}
	if prerequisiteMissing(missing, "go") {
		fmt.Fprintln(os.Stderr, "      go: install Go from https://go.dev/dl/")
	}
	fmt.Fprintln(os.Stderr, "      then re-run: skillgrid install")
	return fmt.Errorf("missing from PATH: %s", strings.Join(missing, ", "))
}

// agentSelected reports whether the given agent key is in the selection.
func agentSelected(c *Config, key string) bool {
	for _, k := range c.Agents {
		if k == key {
			return true
		}
	}
	return false
}

func prerequisiteMissing(missing []string, bin string) bool {
	for _, m := range missing {
		if m == bin {
			return true
		}
	}
	return false
}

func verbPrereqs(c *Config) {
	var parts []string
	for _, bin := range []string{"node", "npm", "uv", "go"} {
		out, err := exec.Command(bin, "--version").Output()
		if err != nil {
			continue
		}
		line := strings.TrimSpace(string(out))
		if i := strings.IndexByte(line, '\n'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		parts = append(parts, bin+" "+line)
	}
	if len(parts) > 0 {
		VerboseOut(c, strings.Join(parts, " / "))
	}
}

// selectAgents asks which agents to configure. Default is ALL UNSELECTED:
// with no --agent flags the user must opt in, so a plain `skillgrid install`
// never configures a harness the user didn't ask for. --yes is the explicit
// "configure everything" path. skill-pi and pi are optional agents (ADR-0043)
// — they appear in the list but are never auto-selected.
func selectAgents(c *Config) error {
	if len(c.Agents) > 0 {
		return nil
	}
	if c.Yes {
		c.Agents = availableAgentKeys()
		VerboseOut(c, "default selection (yes mode): all agents ("+strings.Join(c.Agents, ", ")+")")
		return nil
	}
	if !mnemonic.Interactive() {
		// Non-interactive and no explicit selection: nothing to configure.
		VerboseOut(c, "non-interactive, no agents selected — skipping agent install (use --agent or --yes)")
		return nil
	}

	agents := AvailableAgents()
	opts := make([]mnemonic.Option, len(agents))
	for i, a := range agents {
		opts[i] = mnemonic.Option{
			Label: a.Name,
			Value: a.Key,
		}
	}
	selected, _, err := mnemonic.MultiSelect("Install skillgrid for which agents? (none selected by default, a=all, q=cancel)", opts)
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

// availableAgentKeys returns the installable agent keys in display order.
func availableAgentKeys() []string {
	agents := AvailableAgents()
	keys := make([]string, len(agents))
	for i, a := range agents {
		keys[i] = a.Key
	}
	return keys
}

func installAgents(c *Config) error {
	agents := AvailableAgents()
	for _, key := range c.Agents {
		for _, a := range agents {
			if a.Key != key || a.NPM == "" {
				continue
			}
			if a.Bin != "" {
				if _, err := exec.LookPath(a.Bin); err == nil {
					Out("      skip", a.NPM, "(already installed)")
					continue
				}
			}
			args := npmInstallGlobalArgs(a.NPM)
			if c.DryRun {
				Out(append([]any{"      [dry-run] npm"}, toAny(args)...)...)
				continue
			}
			Out(append([]any{"      npm"}, toAny(args)...)...)
			if err := run(c, "", "npm", args...); err != nil {
				return fmt.Errorf("npm %s: %w", strings.Join(args, " "), err)
			}
		}
	}
	return nil
}

// npmInstallGlobalArgs builds `npm install -g …` args. Packages with native
// postinstalls need --allow-scripts=<name> or npm skips the script.
func npmInstallGlobalArgs(pkg string) []string {
	args := []string{"install", "-g"}
	switch pkg {
	case "opencode-ai", "agent-browser":
		args = append(args, "--allow-scripts="+pkg)
	}
	return append(args, pkg)
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func setupAgents(c *Config) error {
	repoRoot := mnemonic.FindRepoRoot(c.RepoDir)
	if repoRoot == "" {
		repoRoot = c.RepoDir
	}
	mcpEntries, err := mnemonic.LoadMCPConfig(repoRoot)
	if err != nil {
		return fmt.Errorf("load mcp config: %w", err)
	}
	for _, key := range c.Agents {
		if err := mnemonic.RunSetup(key, repoRoot, mcpEntries, c.HomeDir, c.DryRun); err != nil {
			return fmt.Errorf("setup %s: %w", key, err)
		}
	}
	return nil
}

// installInstallMcp npm installs the install-mcp CLI globally (skipped if on PATH).
func installInstallMcp(c *Config) error {
	if _, err := exec.LookPath(installMcpPkg); err == nil {
		Out("      skip", installMcpPkg, "(already installed)")
		return nil
	}
	pkg := installMcpPkg
	if c.DryRun {
		Out("      [dry-run] npm install -g", pkg)
		return nil
	}
	Out("      npm install -g", pkg)
	return run(c, "", "npm", "install", "-g", pkg)
}

// npmPkgName strips a @version suffix from an npm package spec
// ("@playwright/mcp@latest" → "@playwright/mcp").
func npmPkgName(spec string) string {
	if i := strings.LastIndex(spec, "@"); i > 0 {
		return spec[:i]
	}
	return spec
}

// isGloballyInstalled reports whether an npm package is already installed
// globally. Uses `npm ls -g <name>` (exit 0 = installed).
func isGloballyInstalled(name string) bool {
	cmd := exec.Command("npm", "ls", "-g", name)
	return cmd.Run() == nil
}

// installMCPServers npm-installs MCP server packages from config.d/tools.yaml
// once (not per agent). Client registration is left to setupAgents + mcp.yaml.
// Packages already installed globally are skipped.
func installMCPServers(c *Config) error {
	toolsCfg, err := LoadToolsConfig(c.RepoDir)
	if err != nil {
		return fmt.Errorf("load tools config: %w", err)
	}
	globalNPM := map[string]bool{}
	for _, t := range GlobalTools() {
		globalNPM[t.NPM] = true
	}
	for _, pkg := range toolsCfg.MCP {
		npmPkg := normalizeNPMPackage(pkg)
		if globalNPM[npmPkg] {
			VerboseOut(c, "skip MCP pkg "+npmPkg+" (also a global tool)")
			continue
		}
		if c.DryRun {
			Out(append([]any{"      [dry-run] npm"}, toAny(npmInstallGlobalArgs(npmPkg))...)...)
			continue
		}
		name := npmPkgName(npmPkg)
		if isGloballyInstalled(name) {
			Out("      skip", name, "(already installed)")
			continue
		}
		args := npmInstallGlobalArgs(npmPkg)
		Out(append([]any{"      npm"}, toAny(args)...)...)
		if err := run(c, "", "npm", args...); err != nil {
			return fmt.Errorf("npm %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

// normalizeNPMPackage maps tools.yaml entries to npm install targets.
// GitHub-style owner/repo (no @scope) becomes github:owner/repo.
func normalizeNPMPackage(pkg string) string {
	if strings.HasPrefix(pkg, "@") {
		return pkg
	}
	if strings.Count(pkg, "/") == 1 && !strings.HasPrefix(pkg, "github:") {
		return "github:" + pkg
	}
	return pkg
}

func installTools(c *Config) error {
	for _, t := range GlobalTools() {
		if t.Bin != "" {
			if _, err := exec.LookPath(t.Bin); err == nil {
				Out("      skip", t.NPM, "(already installed)")
				continue
			}
		}
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

// installSecurityTools brings up the security scanners (SecurityTools) with
// their own managers: `uv tool install` for wapiti3 and `go install` for the
// Go-based scanners. These are non-fatal: a missing manager warns and skips its
// tools with the manual fallback, rather than aborting the whole install (which
// is gated on node + npm, not on go/uv). Tools install in parallel (capped at
// 3 concurrent) to cut the serial go-install compile time.
func installSecurityTools(c *Config) error {
	var fns []func() error
	for _, t := range SecurityTools() {
		t := t
		if t.Bin != "" {
			if _, err := exec.LookPath(t.Bin); err == nil {
				Out("      skip", t.Name, "(already installed)")
				continue
			}
		}
		if _, err := exec.LookPath(t.Manager); err != nil {
			Out("      warn", t.Name, "skipped —", t.Manager, "not on PATH; "+t.Hint)
			continue
		}
		if c.DryRun {
			Out(append([]any{"      [dry-run]", t.Manager}, toAny(t.InstallArgs)...)...)
			continue
		}
		Out(append([]any{"      ", t.Manager}, toAny(t.InstallArgs)...)...)
		fns = append(fns, func() error {
			if err := run(c, "", t.Manager, t.InstallArgs...); err != nil {
				return fmt.Errorf("%s %s: %w", t.Manager, strings.Join(t.InstallArgs, " "), err)
			}
			return nil
		})
	}
	return runParallel(3, fns)
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

// collectNPMPackages gathers all npm package names needed by the install:
// selected agents, the install-mcp CLI, MCP server packages from tools.yaml,
// and global tools. Deduplicated. Used by installNPMBatch to make a single
// `npm install -g` call instead of N serial calls.
func collectNPMPackages(c *Config) []string {
	seen := map[string]bool{}
	var pkgs []string
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			pkgs = append(pkgs, p)
		}
	}

	// Selected agents with npm packages.
	for _, key := range c.Agents {
		for _, a := range AvailableAgents() {
			if a.Key == key {
				add(a.NPM)
			}
		}
	}

	// install-mcp CLI (always needed when agents are selected).
	if len(c.Agents) > 0 {
		add(installMcpPkg)
	}

	// MCP server packages from tools.yaml.
	if toolsCfg, err := LoadToolsConfig(c.RepoDir); err == nil {
		for _, pkg := range toolsCfg.MCP {
			add(normalizeNPMPackage(pkg))
		}
	}

	// Global tools.
	for _, t := range GlobalTools() {
		add(t.NPM)
	}

	return pkgs
}

// installNPMBatch installs all collected npm packages in a single
// `npm install -g` call. Packages already on PATH are skipped.
// This replaces the N serial npm calls in installAgents, installInstallMcp,
// installMCPServers, and installTools with one batched call.
func installNPMBatch(c *Config) error {
	pkgs := collectNPMPackages(c)

	// Filter out packages already installed (binary on PATH).
	var toInstall []string
	for _, pkg := range pkgs {
		// Find the binary name for this package to check if it's already installed.
		if bin := npmBinForPkg(pkg); bin != "" {
			if _, err := exec.LookPath(bin); err == nil {
				Out("      skip", pkg, "(already installed)")
				continue
			}
		}
		toInstall = append(toInstall, pkg)
	}

	if len(toInstall) == 0 {
		VerboseOut(c, "all npm packages already installed")
		return nil
	}

	// Build the single npm install -g command.
	args := []string{"install", "-g"}
	for _, pkg := range toInstall {
		// Packages with native postinstalls need --allow-scripts.
		switch pkg {
		case "opencode-ai", "agent-browser":
			args = append(args, "--allow-scripts="+pkg)
		}
		args = append(args, pkg)
	}

	if c.DryRun {
		Out(append([]any{"      [dry-run] npm"}, toAny(args)...)...)
		return nil
	}
	Out(append([]any{"      npm"}, toAny(args)...)...)
	if err := run(c, "", "npm", args...); err != nil {
		return fmt.Errorf("npm install -g %s: %w", strings.Join(toInstall, " "), err)
	}
	return nil
}

// npmBinForPkg returns the PATH binary name for an npm package, or "" if
// the package has no known binary (e.g. MCP server packages).
func npmBinForPkg(pkg string) string {
	switch pkg {
	case "opencode-ai":
		return "opencode"
	case "@kilocode/cli":
		return "kilo"
	case "@devopstales/skill-pi":
		return "skill-pi"
	case "install-mcp":
		return "install-mcp"
	case "skills":
		return "skills"
	case "@cucumber/cucumber":
		return "cucumber-js"
	case "backlog.md":
		return "backlog"
	case "jscpd":
		return "jscpd"
	default:
		return ""
	}
}

// runParallel executes fns concurrently with at most limit goroutines.
// It collects all errors (not fail-fast) and returns a combined error if
// any function failed. Successful functions complete even if others fail.
func runParallel(limit int, fns []func() error) error {
	if len(fns) == 0 {
		return nil
	}
	if limit < 1 {
		limit = 1
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	sem := make(chan struct{}, limit)
	for _, fn := range fns {
		fn := fn
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			if err := fn(); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if len(errs) == 0 {
		return nil
	}
	// Join all errors into a single message.
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Errorf("%d install(s) failed: %s", len(errs), strings.Join(msgs, "; "))
}

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

// copyAll recursively copies src to dst. Symlinks are recreated as symlinks
// (dangling links are preserved rather than failing on read).
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
		if d.Type()&fs.ModeSymlink != 0 {
			os.MkdirAll(filepath.Dir(target), 0o755)
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			os.Remove(target)
			return os.Symlink(link, target)
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
