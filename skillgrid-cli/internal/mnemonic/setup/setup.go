// Package setup configures Mnemonic for supported AI agents (opencode, kilocode, cursor).
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v3"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"
)

const (
	mcpServerName = "skillgrid-mnemonic"

	opencodePluginRel = "plugins/opencode/mnemonic.ts"
	kiloPluginRel     = "plugins/kilo/mnemonic.ts"
	cursorTemplateRel = "plugins/cursor/mnemonic.mdc"
	opencodeLogoRel   = "plugins/opencode/skillgrid-logo.tsx"
	kiloLogoRel       = "plugins/kilo/skillgrid-logo.tsx"

	kiloBeginMarker = "<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->"
	kiloEndMarker   = "<!-- END SKILLGRID MNEMONIC -->"
)

// RunSetup configures Mnemonic for the given agent (opencode, kilocode, cursor).
func RunSetup(agent, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if repoRoot == "" {
		repoRoot = FindRepoRoot("")
	}
	if dryRun {
		logging.Info("[dry-run] skillgrid setup " + agent)
	}
	switch agent {
	case "opencode":
		return SetupOpenCode(home, repoRoot, mcpEntries, dryRun)
	case "kilocode", "kilo":
		return SetupKiloCode(home, repoRoot, mcpEntries, dryRun)
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
			Name:     name,
			Type:     srv.Type,
			URL:      srv.URL,
			Command:  srv.Command,
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

func tildePath(home, absPath string) string {
	if home != "" && strings.HasPrefix(absPath, home) {
		return "~" + strings.TrimPrefix(absPath, home)
	}
	return absPath
}

func copyFromRepo(repoRoot, relPath, dst string, dryRun bool) error {
	src := filepath.Join(repoRoot, relPath)
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if dryRun {
		logging.Info("[dry-run] cp " + src + " " + dst)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dst, err)
	}
	logging.Info("copied " + dst)
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
		logging.Info("[dry-run] cp " + src + " " + dst)
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
	logging.Info("copied " + dst)
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
		logging.Info("[dry-run] cp " + path + " " + bak)
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
	logging.Info("backed up " + path + " → " + bak)
	return nil
}

// BackupAgentConfigs backs up the agent config files (kilo.jsonc/opencode.json,
// AGENTS.md, cursor mcp.json) before install-mcp mutates them, so a failed or
// unwanted install-mcp run can be reverted from ~/.skillgrid/backup/<agent>/.
func BackupAgentConfigs(agents []string, dryRun bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for _, agent := range agents {
		var paths []string
		switch agent {
		case "opencode":
			dir := filepath.Join(home, ".config", "opencode")
			paths = []string{filepath.Join(dir, "opencode.jsonc"), filepath.Join(dir, "opencode.json")}
		case "kilocode", "kilo":
			dir := filepath.Join(home, ".config", "kilo")
			paths = []string{
				filepath.Join(dir, "kilo.jsonc"),
				filepath.Join(dir, "opencode.json"),
				filepath.Join(dir, "opencode.jsonc"),
				filepath.Join(dir, "AGENTS.md"),
			}
		case "cursor":
			paths = []string{filepath.Join(home, ".cursor", "mcp.json")}
		default:
			continue
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err != nil {
				continue
			}
			if err := backupConfigFile(home, agent, p, dryRun); err != nil {
				return err
			}
		}
	}
	return nil
}

func agentConfigPath(home, agent string) string {
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

func ensureConfigFile(path string, dryRun bool) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	if dryRun {
		logging.Info("[dry-run] create " + path)
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
		logging.Info("[dry-run] set mcp." + entry.Name + " in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}

// appendPluginPath appends path to a "plugin" array in the given JSONC config,
// skipping the write when the value is already present. Used by the kilo
// installer to register the mnemonic plugin under the user's home.
func appendPluginPath(cfgPath, path string, dryRun bool) error {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("read config %s: %w", cfgPath, err)
	}
	arr := gjson.Get(string(data), "plugin").Array()
	for _, v := range arr {
		if v.Str == path {
			return nil
		}
	}
	arr = append(arr, gjson.Result{Str: path})
	updated, err := sjson.Set(string(data), "plugin", arr)
	if err != nil {
		return fmt.Errorf("set plugin: %w", err)
	}
	if dryRun {
		logging.Info("[dry-run] append " + path + " to plugin[] in " + cfgPath)
		return nil
	}
	return os.WriteFile(cfgPath, []byte(updated), 0o644)
}
