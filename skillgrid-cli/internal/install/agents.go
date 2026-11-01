package install

// Harness config distribution for the supported AI agents (opencode, kilo,
// cursor). This package owns everything that configures the *harness* — agent
// config files, backups, and the TUI logo/theme. The memory component
// (mnemonic/internal/setup) owns MCP registration, the 5 native TS plugins
// (ADR-0032), the shared hook workers, and the memory-protocol block. This
// package does not register plugins or hook scripts.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// nativePluginBases are the 5 native TS plugin file bases (ADR-0032). Both
// harnesses install them from plugins/opencode/<base>.ts; they double as the
// repo-root marker and are registered by the memory installer.
var nativePluginBases = []string{
	"skillgrid-compaction",
	"skillgrid-events",
	"skillgrid-squad",
	"mnemonic-memory",
	"mnemonic-codeindex",
}

// nativePluginRels returns the repo-relative plugin source paths.
func nativePluginRels() []string {
	rels := make([]string, 0, len(nativePluginBases))
	for _, base := range nativePluginBases {
		rels = append(rels, "plugins/opencode/"+base+".ts")
	}
	return rels
}

// Repo-relative paths for the assets the installer copies into the user's
// harness config. These are distribution facts (where in the repo an asset
// lives), not memory facts, so they live with the installer.
const (
	cursorPluginRel = ".cursor-plugin/plugin.json"
	cursorRuleRel   = "rules/mnemonic.mdc"
	// opencodeLogoRel is the PRE-BUILT ESM bundle (JSX already compiled to
	// @opentui/solid calls by tsup + esbuild-plugin-solid). OpenCode never
	// transpiles raw TSX, so the committed .js is the only artifact the
	// installer ships (mirrors Gentle-AI PR #5409 / issue #5364).
	opencodeLogoRel = "plugins/opencode/skillgrid-logo.js"
	kiloLogoRel     = "plugins/kilo/skillgrid-logo.tsx"
)

// skillgrid-logo TUI bridge: the V2 host loads cli.json (V2 "plugins" key) and
// the managed opencode.jsonc; a relative bridge-dir package entry resolves the
// tui subpath via the bridge's package.json. The .tsx is retired.
const (
	logoBridgeRel   = "plugins/opencode/skillgrid-logo" // bridge dir in repo (committed)
	logoBridgeEntry = "./tui-plugins/skillgrid-logo"   // relative entry in V2 cli.json/opencode.jsonc
)

// findRepoRoot walks upward from start (or cwd when empty) for a marker asset,
// falling back to the synced repo under ~/.skillgrid. Mirrors the detection the
// memory setup uses (the 5 native plugins + the cursor assets), so the installer
// and the memory component agree on which checkout is canonical.
func findRepoRoot(start string) string {
	markers := append(nativePluginRels(), cursorPluginRel, cursorRuleRel)
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	for {
		for _, rel := range markers {
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
		for _, rel := range markers {
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

// copyFromRepo copies a single repo-relative file to dst, overwriting.
func copyFromRepo(repoRoot, relPath, dst string, dryRun bool) error {
	if err := copyFile(filepath.Join(repoRoot, relPath), dst, dryRun); err != nil {
		return fmt.Errorf("copy %s: %w", relPath, err)
	}
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

// installAgentConfig applies the harness (non-memory) config for one selected
// agent: TUI logo/theme and the kilo→opencode first-write-wins bridges.
// Hook plugins are registered by setup, not here. It is the harness
// counterpart to setup.RunSetup, which handles the memory parts.
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
		// V2 logo: pre-built ESM bundle + bridge-dir package (OpenCode never
		// transpiles raw TSX — see opencodeLogoRel). Registered via the
		// relative bridge-dir entry in the V2 "plugins" array (cli.json) and
		// the managed opencode.jsonc. The stale absolute .tsx entry (written
		// by older installers) is retired.
		if err := installOpenCodeLogo(repoRoot, dir, dry); err != nil {
			return err
		}
		if err := setJSON(filepath.Join(dir, "tui.json"), "theme", "tokyonight", dry); err != nil {
			return err
		}
		return appendJSONArrayUnique(filepath.Join(dir, "tui.json"), "plugin", "opencode-subagent-statusline", dry)
	case "kilo":
		dir := filepath.Join(home, ".config", "kilo")
		tuiPath := filepath.Join(dir, "tui.json")
		logoDst := filepath.Join(dir, "tui-plugins", "skillgrid-logo.tsx")

		// The shell-era hooks.yaml adapter is retired (ADR-0032); the 5 native TS
		// plugins are installed by the memory component. Only the TUI config
		// bridge from opencode remains.
		opencodeDir := filepath.Join(home, ".config", "opencode")
		if err := copyFirstWriteWins(filepath.Join(opencodeDir, "tui.json"), tuiPath, dry); err != nil {
			return err
		}
		if err := ensureConfigFile(tuiPath, dry); err != nil {
			return err
		}
		if err := copyAsset(repoRoot, kiloLogoRel, logoDst, dry); err != nil {
			return err
		}
		if err := setJSON(tuiPath, "theme", "tokyonight", dry); err != nil {
			return err
		}
		if err := appendJSONArrayUnique(tuiPath, "plugin", "opencode-subagent-statusline", dry); err != nil {
			return err
		}
		return appendJSONArrayUnique(tuiPath, "plugin", logoDst, dry)
	default:
		return nil
	}
}

// installOpenCodeLogo ships the pre-built SkillGrid TUI logo bundle and
// registers it with the OpenCode V2 host (mirrors Gentle-AI PR #5409 /
// issue #5364). OpenCode never transpiles raw TSX, so the committed ESM
// bundle (skillgrid-logo.js) is the payload; a bridge-dir package lets the V2
// "plugins" array reference a relative directory and resolve the tui subpath.
// The stale absolute .tsx entry older installers wrote is retired from both
// the V2 cli.json and the managed opencode.jsonc.
func installOpenCodeLogo(repoRoot, dir string, dryRun bool) error {
	pluginDir := filepath.Join(dir, "tui-plugins")
	if dryRun {
		Out("      [dry-run] mkdir -p", pluginDir)
	} else if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}

	// 1. Pre-built ESM bundle (the actual plugin payload).
	bundleDst := filepath.Join(pluginDir, "skillgrid-logo.js")
	if err := copyAsset(repoRoot, opencodeLogoRel, bundleDst, dryRun); err != nil {
		return err
	}

	// 2. Bridge-dir package: tui.js re-exports the bundle; package.json maps
	//    "." and "./tui" to it so the relative dir entry resolves.
	bridgeDir := filepath.Join(pluginDir, "skillgrid-logo")
	if dryRun {
		Out("      [dry-run] mkdir -p", bridgeDir)
	} else if err := os.MkdirAll(bridgeDir, 0o755); err != nil {
		return err
	}
	tuiBridge := `// OpenCode v2 directory bridge for the skillgrid-logo TUI plugin
export * from "../skillgrid-logo.js";
export { default } from "../skillgrid-logo.js";
`
	manifest := `{"name":"skillgrid-logo-tui","private":true,"type":"module","exports":{".":"./tui.js","./tui":"./tui.js"}}
`
	for rel, content := range map[string]string{
		"tui.js":       tuiBridge,
		"package.json": manifest,
	} {
		dst := filepath.Join(bridgeDir, rel)
		if _, err := os.Stat(dst); err == nil {
			continue // idempotent: leave an existing bridge file alone
		}
		if dryRun {
			Out("      [dry-run] write", dst)
			continue
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return err
		}
	}

	// 3. Retire the stale absolute .tsx entry older installers registered.
	stale := filepath.Join(pluginDir, "skillgrid-logo.tsx")
	if dryRun {
		Out("      [dry-run] rm (if present)", stale)
	} else if _, err := os.Stat(stale); err == nil {
		if err := os.Remove(stale); err != nil {
			return err
		}
	}

	// 4. Register the relative bridge-dir entry in the V2 config(s). The V2
	//    host reads cli.json (and the managed opencode.jsonc); tui.json is a
	//    V1 artifact the V2 host ignores, so it is not written here.
	for _, cfg := range []string{"cli.json", "opencode.jsonc", "opencode.json"} {
		p := filepath.Join(dir, cfg)
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := dropRetiredLogoEntry(p, dryRun); err != nil {
			return err
		}
		if err := appendJSONArrayUnique(p, "plugins", logoBridgeEntry, dryRun); err != nil {
			return err
		}
	}
	return nil
}

// dropRetiredLogoEntry removes the stale absolute .tsx registration (written
// by pre-#5364 installers) from a config's "plugins" array, keeping every
// other entry.
func dropRetiredLogoEntry(cfgPath string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return err
	}
	arr := gjson.Parse(string(data)).Get("plugins").Array()
	if len(arr) == 0 {
		return nil
	}
	kept := make([]string, 0, len(arr))
	dropped := false
	for _, v := range arr {
		s := v.String()
		if strings.HasSuffix(s, "tui-plugins/skillgrid-logo.tsx") {
			dropped = true
			continue
		}
		if s != "" {
			kept = append(kept, s)
		}
	}
	if !dropped {
		return nil
	}
	updated, err := sjson.Set(string(data), "plugins", kept)
	if err != nil {
		return err
	}
	if dryRun {
		Out("      [dry-run] retire stale skillgrid-logo.tsx entry in", cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}
