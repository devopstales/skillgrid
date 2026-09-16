package install

// Harness config distribution for the supported AI agents (opencode, kilo,
// cursor). This package owns everything that configures the *harness* — agent
// config files, backups, the TUI logo/theme, and the plugin-path append. The
// memory component (internal/mnemonic/setup) owns only MCP registration, the
// mnemonic.ts plugin copy, and the memory-protocol block.

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/setup"
)

// Repo-relative paths for the assets the installer copies into the user's
// harness config. These are distribution facts (where in the repo an asset
// lives), not memory facts, so they live with the installer.
const (
	opencodePluginRel = "plugins/opencode/mnemonic.ts"
	kiloPluginRel     = "plugins/kilo/mnemonic.ts"
	cursorTemplateRel = "plugins/cursor/mnemonic.mdc"
	opencodeLogoRel   = "plugins/opencode/skillgrid-logo.tsx"
	kiloLogoRel       = "plugins/kilo/skillgrid-logo.tsx"
)

// findRepoRoot walks upward from start (or cwd when empty) for a marker asset,
// falling back to the synced repo under ~/.skillgrid. Mirrors the detection the
// memory setup uses, so the installer and the memory component agree on which
// checkout is canonical.
func findRepoRoot(start string) string {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	for {
		for _, rel := range []string{opencodePluginRel, kiloPluginRel, cursorTemplateRel} {
			if _, err := os.Stat(filepath.Join(dir, rel)); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if home, err := os.UserHomeDir(); err == nil {
		for _, rel := range []string{opencodePluginRel, kiloPluginRel, cursorTemplateRel} {
			synced := filepath.Join(home, ".skillgrid", "repos", "skillgrid", rel)
			if _, err := os.Stat(synced); err == nil {
				return filepath.Join(home, ".skillgrid", "repos", "skillgrid")
			}
		}
	}
	return ""
}

// configHome returns the home dir for config writes, preferring the injected
// c.HomeDir (test seam) and falling back to the real user home.
func configHome(c *Config) (string, error) {
	if c.HomeDir != "" {
		return c.HomeDir, nil
	}
	return os.UserHomeDir()
}

// ensureConfigFile creates an empty JSON object at path when missing.
func ensureConfigFile(path string, dryRun bool) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if dryRun {
		Out("      [dry-run] create", path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("{}\n"), 0o644)
}

// backupConfigFile copies a single config file into the timestamped backup dir.
func backupConfigFile(home, agent, path string, dryRun bool) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	backupDir := filepath.Join(home, ".skillgrid", "backup", agent)
	base := filepath.Base(path)
	timestamp := time.Now().Format("2006-01-02-15:04")
	bak := filepath.Join(backupDir, base+"-"+timestamp+".back")
	if dryRun {
		Out("      [dry-run] backup", path, "→", bak)
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("backup read %s: %w", path, err)
	}
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("backup mkdir %s: %w", backupDir, err)
	}
	if err := os.WriteFile(bak, data, 0o644); err != nil {
		return fmt.Errorf("backup write %s: %w", bak, err)
	}
	Out("      backed up", path, "→", bak)
	return nil
}

// backupConfigDir copies a config directory tree into the timestamped backup dir.
func backupConfigDir(home, agent, dir string, dryRun bool) error {
	if _, err := os.Stat(dir); err != nil {
		return nil
	}
	timestamp := time.Now().Format("2006-01-02-15:04")
	bakRoot := filepath.Join(home, ".skillgrid", "backup", agent, filepath.Base(dir)+"-"+timestamp)
	if dryRun {
		Out("      [dry-run] backup dir", dir, "→", bakRoot)
		return nil
	}
	return copyAll(dir, bakRoot)
}

// backupAgentConfigs backs up the agent config files and directories before
// setup mutates them, so a failed or unwanted install run can be reverted from
// ~/.skillgrid/backup/<agent>/.
func backupAgentConfigs(c *Config) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	for _, agent := range c.Agents {
		var files []string
		var dirs []string
		switch agent {
		case "opencode":
			dir := filepath.Join(home, ".config", "opencode")
			files = []string{filepath.Join(dir, "opencode.jsonc"), filepath.Join(dir, "opencode.json")}
			dirs = []string{filepath.Join(dir, "agents")}
		case "kilo":
			dir := filepath.Join(home, ".config", "kilo")
			files = []string{
				filepath.Join(dir, "kilo.jsonc"),
				filepath.Join(dir, "opencode.json"),
				filepath.Join(dir, "opencode.jsonc"),
				filepath.Join(dir, "AGENTS.md"),
			}
			dirs = []string{filepath.Join(dir, "agent")}
		case "cursor":
			files = []string{filepath.Join(home, ".cursor", "mcp.json")}
			dirs = []string{filepath.Join(home, ".cursor", "agents")}
		default:
			continue
		}
		for _, p := range files {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			if err := backupConfigFile(home, agent, p, c.DryRun); err != nil {
				return err
			}
		}
		for _, d := range dirs {
			if err := backupConfigDir(home, agent, d, c.DryRun); err != nil {
				return err
			}
		}
	}
	return nil
}

// tildePath renders an absolute path under home as a ~-prefixed path.
func tildePath(home, absPath string) string {
	if home != "" && hasPrefixPath(absPath, home) {
		return "~" + absPath[len(home):]
	}
	return absPath
}

func hasPrefixPath(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}

// copyFromRepo copies a single repo-relative file to dst, overwriting.
func copyFromRepo(repoRoot, relPath, dst string, dryRun bool) error {
	src := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if dryRun {
		Out("      [dry-run] cp", src, dst)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	Out("      copied", dst)
	return nil
}

// copyFirstWriteWins copies src to dst only when dst is absent.
func copyFirstWriteWins(src, dst string, dryRun bool) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if dryRun {
		Out("      [dry-run] cp", src, dst)
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	Out("      copied", dst)
	return nil
}

// setJSON sets a top-level key in a JSON file.
func setJSON(jsonPath, key, value string, dryRun bool) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", jsonPath, err)
	}
	updated, err := sjson.Set(string(data), key, value)
	if err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	if dryRun {
		Out("      [dry-run] set", key, "in", jsonPath)
		return nil
	}
	return os.WriteFile(jsonPath, []byte(updated), 0o644)
}

// appendJSONArrayUnique appends value to a JSON array key, skipping when present.
func appendJSONArrayUnique(jsonPath, key, value string, dryRun bool) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", jsonPath, err)
	}
	arr := gjson.Parse(string(data)).Get(key).Array()
	strs := make([]string, 0, len(arr)+1)
	exists := false
	for _, v := range arr {
		s := jsonArrayString(v)
		if s == "" {
			continue
		}
		if s == value {
			exists = true
		}
		strs = append(strs, s)
	}
	if exists {
		if arrayNeedsRewrite(arr, strs) {
			updated, err := sjson.Set(string(data), key, strs)
			if err != nil {
				return fmt.Errorf("append %s: %w", key, err)
			}
			if dryRun {
				return nil
			}
			return os.WriteFile(jsonPath, []byte(updated), 0o644)
		}
		return nil
	}
	strs = append(strs, value)
	updated, err := sjson.Set(string(data), key, strs)
	if err != nil {
		return fmt.Errorf("append %s: %w", key, err)
	}
	if dryRun {
		Out("      [dry-run] append", value, "to", key, "in", jsonPath)
		return nil
	}
	return os.WriteFile(jsonPath, []byte(updated), 0o644)
}

// jsonArrayString recovers a string from a JSON array element, healing the
// legacy gjson.Result object marshal.
func jsonArrayString(v gjson.Result) string {
	if v.Type == gjson.String {
		return v.Str
	}
	if v.IsObject() {
		if s := v.Get("Str").Str; s != "" {
			return s
		}
	}
	return ""
}

func arrayNeedsRewrite(raw []gjson.Result, strs []string) bool {
	if len(raw) != len(strs) {
		return true
	}
	for _, v := range raw {
		if v.Type != gjson.String {
			return true
		}
	}
	return false
}

// appendPluginPath appends path to a "plugin" array in a JSONC config, skipping
// the write when already present.
func appendPluginPath(cfgPath, path string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	arr := gjson.Get(string(data), "plugin").Array()
	strs := make([]string, 0, len(arr)+1)
	exists := false
	for _, v := range arr {
		s := jsonArrayString(v)
		if s == "" {
			continue
		}
		if s == path {
			exists = true
		}
		strs = append(strs, s)
	}
	if exists {
		if arrayNeedsRewrite(arr, strs) {
			updated, err := sjson.Set(string(data), "plugin", strs)
			if err != nil {
				return fmt.Errorf("set plugin: %w", err)
			}
			if dryRun {
				return nil
			}
			return os.WriteFile(cfgPath, []byte(updated), 0o644)
		}
		return nil
	}
	strs = append(strs, path)
	updated, err := sjson.Set(string(data), "plugin", strs)
	if err != nil {
		return fmt.Errorf("set plugin: %w", err)
	}
	if dryRun {
		Out("      [dry-run] append", path, "to plugin[] in", cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

// installAgentConfig applies the harness (non-memory) config for one selected
// agent: TUI logo/theme, the plugin-path append (kilo), and the kilo→opencode
// first-write-wins bridges. It is the harness counterpart to setup.RunSetup,
// which handles the memory parts.
func installAgentConfig(c *Config, agent string) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	repoRoot := findRepoRoot(c.RepoDir)
	if repoRoot == "" {
		repoRoot = c.RepoDir
	}
	dry := c.DryRun

	switch agent {
	case "opencode":
		dir := filepath.Join(home, ".config", "opencode")
		tuiPath := filepath.Join(dir, "tui.json")
		if err := ensureConfigFile(tuiPath, dry); err != nil {
			return err
		}
		logoDst := filepath.Join(dir, "tui-plugins", "skillgrid-logo.tsx")
		if err := copyFromRepo(repoRoot, opencodeLogoRel, logoDst, dry); err != nil {
			return err
		}
		if err := setJSON(tuiPath, "theme", "tokyonight", dry); err != nil {
			return err
		}
		return appendJSONArrayUnique(tuiPath, "plugin", logoDst, dry)
	case "kilo":
		dir := filepath.Join(home, ".config", "kilo")
		pluginDst := filepath.Join(dir, "plugins", "mnemonic.ts")
		sharedDst := filepath.Join(dir, "shared", "http-client.ts")
		tuiPath := filepath.Join(dir, "tui.json")
		logoDst := filepath.Join(dir, "tui-plugins", "skillgrid-logo.tsx")

		opencodeDir := filepath.Join(home, ".config", "opencode")
		bridges := []struct{ src, dst string }{
			{filepath.Join(opencodeDir, "plugins", "mnemonic.ts"), pluginDst},
			{filepath.Join(opencodeDir, "shared", "http-client.ts"), sharedDst},
			{filepath.Join(opencodeDir, "tui.json"), tuiPath},
		}
		for _, b := range bridges {
			if err := copyFirstWriteWins(b.src, b.dst, dry); err != nil {
				return err
			}
		}
		if err := ensureConfigFile(tuiPath, dry); err != nil {
			return err
		}
		if err := copyFromRepo(repoRoot, kiloLogoRel, logoDst, dry); err != nil {
			return err
		}
		if err := setJSON(tuiPath, "theme", "tokyonight", dry); err != nil {
			return err
		}
		if err := appendJSONArrayUnique(tuiPath, "plugin", logoDst, dry); err != nil {
			return err
		}
		cfgPath := setup.AgentConfigPath(home, "kilo")
		pluginRef := tildePath(home, pluginDst)
		return appendPluginPath(cfgPath, pluginRef, dry)
	default:
		return nil
	}
}
