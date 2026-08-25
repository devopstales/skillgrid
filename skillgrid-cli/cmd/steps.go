package main

import (
	"skillgrid-cli/internal/config"
	"skillgrid-cli/internal/logging"
	jsonc "github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureNode verifies node is usable; if missing it runs the repo's
// scripts/install_node.sh (docs step 2).
func ensureNode(baseDir string) error {
	if _, err := exec.LookPath("node"); err == nil {
		if _, verr := exec.Command("node", "--version").Output(); verr == nil {
			return nil
		}
	}
	script := filepath.Join(baseDir, "repos", "skillgrid", "scripts", "install_node.sh")
	if _, err := os.Stat(script); err != nil {
		return fmt.Errorf("node not found and installer script missing at %s (run install_node.sh manually)", script)
	}
	return exec.Command("bash", script).Run()
}

// selectAgents prints the agent selector (docs step 4) and returns the
// selected agents. Non-interactive runs default to all agents.
func selectAgents(agents []string) []string {
	fmt.Fprintln(os.Stdout, "\nAgent selector")
	for i, a := range agents {
		fmt.Fprintf(os.Stdout, "  [%d] %s\n", i+1, a)
	}
	fmt.Fprint(os.Stdout, "Select agents (comma-separated, default all): ")
	var line string
	if _, err := fmt.Scanln(&line); err != nil {
		return agents
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return agents
	}
	var out []string
	for _, part := range strings.Split(line, ",") {
		part = strings.TrimSpace(part)
		for _, a := range agents {
			if part == a {
				out = append(out, a)
			}
		}
	}
	if len(out) == 0 {
		return agents
	}
	return out
}

const superpowersRef = "superpowers@git+https://github.com/obra/superpowers.git"

// ensureSkillPaths registers the superpowers skills directory in the agent
// config's "skills.paths". Kilo only trusts skills under absolute paths
// listed there; plugin-injected skill paths are treated as untrusted and
// fail parsing with "blocked file reference outside project config scope".
func ensureSkillPaths(baseDir string, agents []string, dryRun bool) {
	for _, agent := range agents {
		prefix := agentConfigPrefix(agent)
		if prefix == "" {
			continue
		}
		cfgPath := agentConfigPath(agent)
		if _, err := os.Stat(cfgPath); err != nil {
			logging.Warn("skill paths skipped for " + agent + ": config not found " + cfgPath)
			continue
		}
		skillsDir := filepath.Join(prefix, "node_modules", "superpowers", "skills")
		if home, err := os.UserHomeDir(); err == nil {
			skillsDir = strings.ReplaceAll(skillsDir, home, "~")
		}
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			continue
		}
		existing := []string{}
		for _, v := range jsonc.Get(string(data), "skills.paths").Array() {
			existing = append(existing, v.String())
		}
		found := false
		for _, have := range existing {
			if have == skillsDir {
				found = true
			}
		}
		if found {
			continue
		}
		if dryRun {
			logging.Info("[dry-run] skills.paths += " + skillsDir + " in " + cfgPath)
			continue
		}
		backUpConfig(baseDir, cfgPath, dryRun)
		existing = append(existing, skillsDir)
		updated, err := sjson.Set(string(data), "skills.paths", existing)
		if err != nil {
			logging.Warn("skills.paths: " + err.Error())
			continue
		}
		if werr := os.WriteFile(cfgPath, []byte(updated), 0644); werr == nil {
			logging.Info("skills.paths += " + skillsDir + " in " + cfgPath)
		}
	}
}

// installPlugins installs the superpowers plugin per agent (docs step 6).
// For Kilo, copies the plugin JS directly to ~/.config/kilo/plugins/ instead
// of adding a "plugin" entry to kilo.jsonc. For OpenCode, registers it under
// the "plugin" key.
func installPlugins(baseDir string, agents []string, dryRun bool) {
	for _, agent := range agents {
		prefix := agentConfigPrefix(agent)
		if prefix == "" {
			continue
		}
		if !dryRun {
			npmCache := filepath.Join(prefix, "npm", "cache")
			if err := exec.Command("npm", "install", superpowersRef, "--prefix", prefix, "--cache", npmCache).Run(); err != nil {
				logging.Warn("plugin install failed for " + agent + ": " + err.Error())
			}
		}
		if agent == "kilo" {
			srcPlugin := filepath.Join(prefix, "node_modules", "superpowers", ".opencode", "plugins", "superpowers.js")
			dstPlugin := filepath.Join(prefix, "plugins", "superpowers.js")
			if dryRun {
				if _, err := os.Stat(srcPlugin); err == nil {
					logging.Info("[dry-run] cp " + srcPlugin + " " + dstPlugin)
				}
				continue
			}
			if _, err := os.Stat(dstPlugin); os.IsNotExist(err) {
				if data, err := os.ReadFile(srcPlugin); err == nil {
					if err := os.MkdirAll(filepath.Dir(dstPlugin), 0o755); err == nil {
						if err := os.WriteFile(dstPlugin, data, 0o644); err == nil {
							logging.Info("plugin copied to " + dstPlugin)
						}
					}
				}
			}
			continue
		}
		pluginPath := filepath.Join(prefix, "node_modules", "superpowers")
		if home, err := os.UserHomeDir(); err == nil {
			pluginPath = strings.ReplaceAll(pluginPath, home, "~")
		}
		backUpConfig(baseDir, agentConfigPath(agent), dryRun)
		setConfigArray(agentConfigPath(agent), "plugin", []string{pluginPath})
		logging.Info("plugin installed for " + agent)
	}

	engramHome := filepath.Join(mustExpandHomePath("~"), ".config")
	srcEngram := filepath.Join(engramHome, "opencode", "plugins", "engram.ts")
	dstEngram := filepath.Join(engramHome, "kilo", "plugins", "engram.ts")
	srcTUI := filepath.Join(engramHome, "opencode", "tui.json")
	dstTUI := filepath.Join(engramHome, "kilo", "tui.json")
	if dryRun {
		if _, err := os.Stat(filepath.Join(engramHome, "opencode", "plugins")); err == nil {
			logging.Info("[dry-run] cp " + srcEngram + " " + dstEngram)
		}
		if _, err := os.Stat(srcTUI); err == nil {
			logging.Info("[dry-run] cp " + srcTUI + " " + dstTUI)
		}
		return
	}
	if hasAgent(agents, "opencode") {
		engramBin := filepath.Join(baseDir, "bin", "engram")
		if _, err := os.Stat(engramBin); err != nil {
			engramBin = "engram"
		}
		if err := exec.Command(engramBin, "setup", "opencode").Run(); err != nil {
			logging.Warn("engram setup opencode failed: " + err.Error())
		}
	}
	if _, err := os.Stat(dstEngram); os.IsNotExist(err) {
		if data, err := os.ReadFile(srcEngram); err == nil {
			if err := os.MkdirAll(filepath.Dir(dstEngram), 0o755); err == nil {
				if err := os.WriteFile(dstEngram, data, 0o644); err == nil {
					logging.Info("engram plugin copied to " + dstEngram)
				}
			}
		}
	}
	if _, err := os.Stat(dstTUI); os.IsNotExist(err) {
		if data, err := os.ReadFile(srcTUI); err == nil {
			if err := os.MkdirAll(filepath.Dir(dstTUI), 0o755); err == nil {
				if err := os.WriteFile(dstTUI, data, 0o644); err == nil {
					logging.Info("tui.json copied to " + dstTUI)
				}
			}
		}
	}
}

// setConfigArray appends values to a top-level string array key, creating
// the array if absent.
func setConfigArray(cfgPath string, key string, values []string) {
	if cfgPath == "" {
		return
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return
	}
	existing := []string{}
	for _, v := range jsonc.Get(string(data), key).Array() {
		existing = append(existing, v.String())
	}
	for _, want := range values {
		found := false
		for _, have := range existing {
			if have == want {
				found = true
			}
		}
		if !found {
			existing = append(existing, want)
		}
	}
	updated, err := sjson.Set(string(data), key, existing)
	if err != nil {
		logging.Warn(key + ": " + err.Error())
		return
	}
	if werr := os.WriteFile(cfgPath, []byte(updated), 0644); werr == nil {
		logging.Info(key + " updated in " + cfgPath)
	}
}

// installSkills installs skills from config.d/skills.yaml (docs step 7)
// via the local skills CLI.
func installSkills(baseDir string, dryRun bool) {
	cfg, err := config.LoadSkillsYAML(filepath.Join(baseDir, "config.d", "skills.yaml"))
	if err != nil {
		logging.Warn("skills: " + err.Error())
		return
	}
	skillsBin := filepath.Join(mustExpandHomePath("~/.skillgrid/npm"), "bin", "skills")
	if _, err := os.Stat(skillsBin); err != nil {
		skillsBin = "skills"
	}
	for _, s := range cfg.Skills {
		if s.Repo == "" {
			continue
		}
		agent := s.Agent
		if agent == "" {
			agent = "amp"
		}
		skill := s.Skill
		if skill == "" {
			skill = "*"
		}
		args := []string{"add", s.Repo, "--agent", agent, "-g", "-s", skill, "-y"}
		if dryRun {
			logging.Info("[dry-run] skills " + strings.Join(args, " "))
			continue
		}
		if err := exec.Command(skillsBin, args...).Run(); err != nil {
			logging.Warn("skills add failed for " + s.Repo + " (" + skill + "): " + err.Error())
		} else {
			logging.Info("skills added: " + s.Repo + " (" + skill + ")")
		}
	}
}

func hasAgent(agents []string, want string) bool {
	for _, a := range agents {
		if a == want {
			return true
		}
	}
	return false
}

func hasTool(tools []string, want string) bool {
	for _, t := range tools {
		if t == want || strings.HasSuffix(t, "/"+want) {
			return true
		}
	}
	return false
}

func agentConfigPrefix(agent string) string {
	home := mustExpandHomePath("~")
	switch agent {
	case "kilo":
		return filepath.Join(home, ".config", "kilo")
	case "opencode":
		return filepath.Join(home, ".config", "opencode")
	}
	return ""
}
