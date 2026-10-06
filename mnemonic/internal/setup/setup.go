// Package setup configures Mnemonic for supported AI agents (opencode, kilocode, cursor).
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"

	"github.com/devopstales/skillgrid/mnemonic/internal/config"
)

const (
	mcpServerName = "skillgrid-mnemonic"

	opencodePluginRel           = "plugins/opencode/hooks.yaml"
	opencodeCheckpointPluginRel = "plugins/opencode/skillgrid-checkpoint.ts"
	kiloPluginRel               = "plugins/kilo/hooks.yaml"
	kiloCheckpointPluginRel     = "plugins/kilo/skillgrid-checkpoint.ts"
	cursorPluginRel             = ".cursor-plugin/plugin.json"
	cursorRuleRel               = "rules/mnemonic.mdc"

	kiloBeginMarker = "<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->"
	kiloEndMarker   = "<!-- END SKILLGRID MNEMONIC -->"
)

// RunSetup configures Mnemonic for the given agent (opencode, kilocode, cursor).
// home is the harness home dir the agent config and config.d/indexing.yaml
// resolve under (os.UserHomeDir when empty) — it is what the installer passes
// so the private-tools env mirror reads the same config the operator installed.
func RunSetup(agent, repoRoot string, mcpEntries []MCPServerConfig, home string, dryRun bool) error {
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return err
		}
	}
	if repoRoot == "" {
		repoRoot = FindRepoRoot("")
	}
	if dryRun {
		logInfo("[dry-run] skillgrid setup " + agent)
	}
	switch agent {
	case "opencode":
		// Mirror the config always-private allowlist into the plugin env
		// (2026-09-24 monitoring). Config load is best-effort here: a missing
		// or malformed indexing.yaml just means no env var (no allowlist).
		return SetupOpenCode(home, repoRoot, mcpEntries, config.Load(home).PrivateTools, dryRun)
	case "kilocode", "kilo":
		return SetupKiloCode(home, repoRoot, mcpEntries, config.Load(home).PrivateTools, dryRun)
	case "cursor":
		return SetupCursor(home, repoRoot, mcpEntries, dryRun)
	default:
		return fmt.Errorf("unknown agent %q (use opencode, kilocode, or cursor)", agent)
	}
}

// MCPServerConfig represents a single MCP server entry from config.d/mcp.yaml.
type MCPServerConfig struct {
	Name    string
	Type    string
	URL     string
	Command []string
}

// LoadMCPConfig reads config.d/mcp.yaml from the repo root.
func LoadMCPConfig(repoRoot string) ([]MCPServerConfig, error) {
	path := filepath.Join(repoRoot, "config.d", "mcp.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read mcp config: %w", err)
	}
	var raw struct {
		Servers map[string]struct {
			Type    string   `yaml:"type"`
			URL     string   `yaml:"url"`
			Command []string `yaml:"command"`
		} `yaml:"servers"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse mcp config: %w", err)
	}
	var entries []MCPServerConfig
	for name, srv := range raw.Servers {
		entries = append(entries, MCPServerConfig{
			Name:    name,
			Type:    srv.Type,
			URL:     srv.URL,
			Command: srv.Command,
		})
	}
	return entries, nil
}

// FindRepoRoot walks upward from start (or cwd when empty) for plugin files.
func FindRepoRoot(start string) string {
	dir := start
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return ""
		}
	}
	for {
		for _, rel := range []string{opencodePluginRel, kiloPluginRel, cursorPluginRel, cursorRuleRel} {
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
		for _, rel := range []string{opencodePluginRel, kiloPluginRel, cursorPluginRel, cursorRuleRel} {
			synced := filepath.Join(home, ".skillgrid", "repos", "skillgrid", rel)
			if _, err := os.Stat(synced); err == nil {
				return filepath.Join(home, ".skillgrid", "repos", "skillgrid")
			}
		}
	}
	return ""
}

func mnemonicMCPEntry() map[string]interface{} {
	return map[string]interface{}{
		"type":    "local",
		"command": []interface{}{"skillgrid", "mcp"},
		"enabled": true,
	}
}

func cursorMCPEntry() map[string]interface{} {
	return map[string]interface{}{
		"command": "skillgrid",
		"args":    []interface{}{"skillgrid", "mcp"},
	}
}

// stagedPluginPath returns the staged ~/.skillgrid/plugins copy of a
// repo-relative plugins/... asset when present, else "". Plugin installs read
// staged-first (the `skillgrid install` hook-tree staging owns that copy);
// the checkout is the fallback.
func stagedPluginPath(rel string) string {
	if !strings.HasPrefix(rel, "plugins/") {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	p := filepath.Join(home, ".skillgrid", "plugins", strings.TrimPrefix(rel, "plugins/"))
	if _, err := os.Stat(p); err != nil {
		return ""
	}
	return p
}

// openCodeHookScripts are the OpenCode/Kilo adapters copied to ~/.skillgrid/hooks/
// by setup. hooks.yaml calls the shell scripts; those call the shared workers.
var openCodeHookScripts = []string{
	"opencode-session-start.sh",
	"opencode-session-end.sh",
	"opencode-policy.sh",
	"opencode-tool-capture.sh",
	"tool-call-capture.js",
	"stop-tests.js",
	"gate-stop.js",
}

func installOpenCodeHookScripts(home, repoRoot string, dryRun bool) error {
	dstDir := filepath.Join(home, ".skillgrid", "hooks")
	for _, name := range openCodeHookScripts {
		src := filepath.Join(repoRoot, "hooks", name)
		data, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("read hook script %s: %w", name, err)
		}
		dst := filepath.Join(dstDir, name)
		if dryRun {
			logInfo("[dry-run] cp " + src + " " + dst)
			continue
		}
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0o755
		}
		if err := os.WriteFile(dst, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
		if err := os.Chmod(dst, mode); err != nil {
			return err
		}
	}
	return nil
}

func copyFromRepo(repoRoot, relPath, dst string, dryRun bool) error {
	src := filepath.Join(repoRoot, relPath)
	if staged := stagedPluginPath(relPath); staged != "" {
		src = staged
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if dryRun {
		logInfo("[dry-run] cp " + src + " " + dst)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	logInfo("copied " + dst)
	return nil
}

func copyFirstWriteWins(src, dst string, dryRun bool) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	if dryRun {
		logInfo("[dry-run] cp " + src + " " + dst)
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
		return err
	}
	logInfo("copied " + dst)
	return nil
}

func backupConfigFile(home, agent, path string, dryRun bool) error {
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	backupDir := filepath.Join(home, ".skillgrid", "backup", agent)
	base := filepath.Base(path)
	timestamp := time.Now().Format("2006-01-02-15:04")
	bak := filepath.Join(backupDir, base+"-"+timestamp+".back")
	if dryRun {
		logInfo("[dry-run] cp " + path + " " + bak)
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
	logInfo("backed up " + path + " → " + bak)
	return nil
}

func ensureConfigFile(path string, dryRun bool) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if dryRun {
		logInfo("[dry-run] create " + path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("{}\n"), 0o644)
}

func upsertMarkerBlock(content, begin, end, body string) string {
	block := begin + "\n" + body + "\n" + end
	start := strings.Index(content, begin)
	stop := strings.Index(content, end)
	if start >= 0 && stop > start {
		return content[:start] + block + content[stop+len(end):]
	}
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" {
		return content + "\n" + block + "\n"
	}
	return block + "\n"
}

// retiredPluginEntry reports plugin registrations Skillgrid no longer loads.
// opencode-yaml-hooks reads hooks.yaml itself; mnemonic.ts was replaced by
// that plugin plus skillgrid-checkpoint.ts.
func retiredPluginEntry(name string) bool {
	if name == "opencode-command-hooks" {
		return true
	}
	if filepath.Base(name) == "mnemonic.ts" {
		return true
	}
	return strings.HasSuffix(filepath.ToSlash(name), "hook/hooks.yaml")
}

// dropRetiredPlugins removes outdated plugin entries from an opencode-style
// JSONC config, leaving every other entry in place.
func dropRetiredPlugins(cfgPath string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	arr := gjson.Parse(string(data)).Get("plugin").Array()
	if len(arr) == 0 {
		return nil
	}
	kept := make([]string, 0, len(arr))
	dropped := false
	for _, v := range arr {
		name := v.String()
		if name == "" {
			continue
		}
		if retiredPluginEntry(name) {
			dropped = true
			continue
		}
		kept = append(kept, name)
	}
	if !dropped {
		return nil
	}
	updated, err := sjson.Set(string(data), "plugin", kept)
	if err != nil {
		return fmt.Errorf("set plugin: %w", err)
	}
	if dryRun {
		logInfo("[dry-run] drop retired plugins in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

// upsertPluginKey sets the "plugin" array in an opencode-style JSONC config
// (opencode.jsonc / kilo.jsonc) to include the given plugin name, preserving
// any existing entries.
func upsertPluginKey(cfgPath, pluginName string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	arr := gjson.Parse(string(data)).Get("plugin").Array()
	exists := false
	for _, v := range arr {
		if v.String() == pluginName {
			exists = true
			break
		}
	}
	if exists {
		return nil
	}
	updated, err := sjson.Set(string(data), "plugin."+strconv.Itoa(len(arr)), pluginName)
	if err != nil {
		return fmt.Errorf("set plugin: %w", err)
	}
	if dryRun {
		logInfo("[dry-run] set plugin in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

// upsertOpenCodeMCP writes a server entry into the "mcp" object of an
// opencode-style JSONC config (kilo.jsonc). Entries are keyed by the server
// name from the YAML config; existing entries of the same name are replaced.
func upsertOpenCodeMCP(cfgPath string, entry MCPServerConfig, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	value := map[string]interface{}{}
	switch entry.Type {
	case "remote":
		value["type"] = "remote"
		value["url"] = entry.URL
		value["enabled"] = true
	default:
		value["type"] = "local"
		value["command"] = entry.Command
		value["enabled"] = true
	}
	updated, err := sjson.Set(string(data), "mcp."+entry.Name, value)
	if err != nil {
		return fmt.Errorf("set mcp.%s: %w", entry.Name, err)
	}
	if dryRun {
		logInfo("[dry-run] set mcp." + entry.Name + " in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

// AgentConfigPath returns the harness's primary JSONC config path, preferring
// an existing file and defaulting to the canonical name. Shared by the memory
// writers (MCP registration) and the installer (harness config).
func AgentConfigPath(home, agent string) string {
	switch agent {
	case "opencode":
		dir := filepath.Join(home, ".config", "opencode")
		for _, name := range []string{"opencode.jsonc", "opencode.json"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return filepath.Join(dir, "opencode.jsonc")
	case "kilo":
		dir := filepath.Join(home, ".config", "kilo")
		for _, name := range []string{"kilo.jsonc", "opencode.json", "opencode.jsonc"} {
			p := filepath.Join(dir, name)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
		return filepath.Join(dir, "kilo.jsonc")
	default:
		return ""
	}
}
