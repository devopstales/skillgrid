package main

import (
	"skillgrid-cli/internal/config"
	"skillgrid-cli/internal/engram"
	"skillgrid-cli/internal/logging"
	"skillgrid-cli/internal/mcp"
	"skillgrid-cli/internal/repo"
	jsonc "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func runInstall(skipClone bool, syncRepo string, dryRun bool, verbose bool, nonInteractive bool) {
	baseDir := mustExpandHomePath("~/.skillgrid")
	if err := logging.Init(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logging: %v\n", err)
		return
	}
	logging.Info("install started")

	allAgents := []string{"kilo", "opencode"}

	// Step 0: interactive agent selector (first, before any setup)
	agents := allAgents
	if !dryRun && !nonInteractive {
		agents = selectAgents(allAgents)
	}
	logging.Info("agents selected: " + strings.Join(agents, ", "))

	switch {
	case syncRepo != "":
		logging.Info("syncing repo from " + syncRepo)
		if err := repo.Sync(mustExpandHomePath(syncRepo), baseDir); err != nil {
			logging.Error("sync repo failed: " + err.Error())
			return
		}
	case !skipClone:
		logging.Info("cloning repo")
		if err := repo.Clone(baseDir, repoURL()); err != nil {
			logging.Error("clone failed: " + err.Error())
			return
		}
	default:
		logging.Info("skipping repo step (-skip-clone)")
	}

	// Step 2: check and install node
	if err := ensureNode(baseDir); err != nil {
		logging.Warn("node check: " + err.Error())
	}

	// Step 3: engram binary
	if err := engram.InstallBinary(baseDir); err != nil {
		logging.Warn("engram install failed: " + err.Error())
	}

	// Step 4: install selected agents and tools from tools.yaml
	tools, err := config.LoadToolsYAML(filepath.Join(baseDir, "config.d", "tools.yaml"))
	if err != nil {
		logging.Error("load tools config failed: " + err.Error())
		return
	}
	if dryRun {
		prefix := mustExpandHomePath("~/.skillgrid/npm")
		cache := filepath.Join(prefix, "cache")
		var pkgs []string
		for _, p := range append(append([]string{}, tools.Agents...), tools.Tools...) {
			pkgs = append(pkgs, resolveNPMPackage(p))
		}
		registry, git := splitNPMPackages(pkgs)
		if len(registry) > 0 {
			logging.Info("[dry-run] HUSKY=0 npm " + strings.Join(npmInstallArgs(prefix, cache, registry, false), " "))
		}
		if len(git) > 0 {
			logging.Info("[dry-run] HUSKY=0 npm " + strings.Join(npmInstallArgs(prefix, cache, git, true), " "))
		}
	} else if err := installNPM(baseDir); err != nil {
		logging.Warn("npm install failed: " + err.Error())
	}

	// Step 4b: run agent-browser install (downloads Chrome)
	if hasTool(tools.Tools, "agent-browser") {
		agentBrowserBin := filepath.Join(mustExpandHomePath("~/.skillgrid/npm"), "bin", "agent-browser")
		if dryRun {
			logging.Info("[dry-run] " + agentBrowserBin + " install")
		} else if err := exec.Command(agentBrowserBin, "install").Run(); err != nil {
			logging.Warn("agent-browser install failed: " + err.Error())
		}
	}

	// Step 5: install plugins
	installPlugins(baseDir, agents, dryRun)
	ensureSkillPaths(baseDir, agents, dryRun)

	// Step 6: install skills from skills.yaml
	installSkills(baseDir, dryRun)

	// Step 7: install mcp from mcp.yaml
	mcpServers, err := mcp.LoadRegistry(filepath.Join(baseDir, "config.d"))
	if err != nil {
		logging.Error("load mcp config failed: " + err.Error())
		return
	}
	for _, agent := range agents {
		configPath := agentConfigPath(agent)
		if _, err := os.Stat(configPath); err != nil {
			logging.Warn("agent config not found, skipping: " + configPath)
			continue
		}
		backUpConfig(baseDir, configPath, dryRun)
		plan, err := config.MergeMCP(configPath, mcpServers, dryRun)
		if err != nil {
			logging.Warn("merge failed for " + agent + ": " + err.Error())
			continue
		}
		if len(plan.Changes) > 0 {
			if verbose {
				for _, ch := range plan.Changes {
					logging.Info(fmt.Sprintf("%s %s=%s", ch.Action, ch.Key, ch.Value))
				}
			} else {
				logging.Info("mcp configured in " + configPath)
			}
		}
	}

	// Step 8: install rules
	copyRules(baseDir, agents, dryRun)

	logging.Info("install finished")
	fmt.Fprintln(os.Stdout)
	if err := config.WritePathInstructions(baseDir, os.Stdout); err != nil {
		logging.Error("write PATH instructions failed: " + err.Error())
	}
}

func repoURL() string {
	if u := os.Getenv("SKILLGRID_REPO_URL"); u != "" {
		return u
	}
	return repo.DefaultRepoURL
}

const maxBackups = 10

func backUpConfig(baseDir, configPath string, dryRun bool) {
	if dryRun || configPath == "" {
		return
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	dir := filepath.Join(baseDir, "backups")
	if err := os.MkdirAll(dir, 0755); err != nil {
		logging.Warn("backup: " + err.Error())
		return
	}
	baseFile := filepath.Base(configPath) + "." + time.Now().Format("20060102-150405") + ".bak"
	dst := filepath.Join(dir, baseFile)
	for n := 2; ; n++ {
		if _, err := os.Stat(dst); os.IsNotExist(err) {
			break
		}
		dst = filepath.Join(dir, fmt.Sprintf("%s.%d", baseFile, n))
		if n > 1000 {
			break
		}
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		logging.Warn("backup: " + err.Error())
		return
	}
	pruneBackups(dir, filepath.Base(configPath))
	logging.Info("backup created at " + dst)
}

func pruneBackups(dir, baseName string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var baks []string
	for _, e := range entries {
		n := e.Name()
		if strings.HasPrefix(n, baseName+".") && (strings.HasSuffix(n, ".bak") || regexp.MustCompile(`\.bak\.\d+$`).MatchString(n)) {
			baks = append(baks, n)
		}
	}
	for i := 0; i < len(baks)-maxBackups; i++ {
		os.Remove(filepath.Join(dir, baks[i]))
	}
}

func copyRules(baseDir string, agents []string, dryRun bool) {
	src := filepath.Join(mustExpandHomePath("~/.skillgrid"), "config.d", "AGENTS.md")
	dstDir := filepath.Join(mustExpandHomePath("~"), ".agents")
	dst := filepath.Join(dstDir, "AGENTS.md")
	data, err := os.ReadFile(src)
	if err != nil {
		logging.Warn("rules: " + err.Error())
		return
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		logging.Warn("rules: " + err.Error())
		return
	}
	if err := os.WriteFile(dst, data, 0644); err != nil {
		logging.Warn("rules: " + err.Error())
		return
	}
	logging.Info("rules copied to " + dst)
	for _, agent := range agents {
		cfgPath := agentConfigPath(agent)
		if cfgPath == "" {
			continue
		}
		if _, err := os.Stat(cfgPath); err != nil {
			continue
		}
		backUpConfig(baseDir, cfgPath, dryRun)
		ensureRulesReference(cfgPath, dst)
	}
}

// ensureRulesReference appends rulesPath to the top-level "instructions"
// array of cfgPath (shared by both Kilo and OpenCode). It is JSON-aware and
// never rewrites other keys.
func ensureRulesReference(cfgPath, rulesPath string) {
	if cfgPath == "" {
		return
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return
	}
	existing := []string{}
	for _, v := range jsonc.Get(string(data), "instructions").Array() {
		existing = append(existing, v.String())
	}
	if home, herr := os.UserHomeDir(); herr == nil {
		rulesPath = strings.ReplaceAll(rulesPath, home, "~")
	}
	for _, v := range existing {
		if v == rulesPath {
			return
		}
	}
	existing = append(existing, rulesPath)
	updated, err := sjson.Set(string(data), "instructions", existing)
	if err != nil {
		logging.Warn("rules: " + err.Error())
		return
	}
	if werr := os.WriteFile(cfgPath, []byte(updated), 0644); werr == nil {
		logging.Info("rules added to " + cfgPath)
	}
}

func runSyncRepo(extraPath string) {
	baseDir := mustExpandHomePath("~/.skillgrid")
	if err := logging.Init(baseDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logging: %v\n", err)
		return
	}
	logging.Info("sync-repo started")
	if err := repo.Sync(mustExpandHomePath(extraPath), baseDir); err != nil {
		logging.Error("sync repo failed: " + err.Error())
		return
	}
	logging.Info("sync-repo finished")
}

func mustExpandHome(p string) string {
	return mustExpandHomePath(p)
}

func mustExpandHomePath(p string) string {
	if len(p) > 0 && p[0] == '~' {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[1:])
		}
	}
	return p
}

func agentConfigPath(agent string) string {
	home := mustExpandHome("~")
	switch agent {
	case "kilo":
		return filepath.Join(home, ".config", "kilo", "kilo.jsonc")
	case "opencode":
		return filepath.Join(home, ".config", "opencode", "opencode.jsonc")
	default:
		return ""
	}
}
