// Package setup configures Mnemonic for supported AI agents (opencode, kilocode, cursor).
package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"
)

const (
	mcpServerName = "skillgrid-mnemonic"

	pluginRelPath     = "plugins/mnemonic/opencode/mnemonic.ts"
	httpClientRelPath = "plugins/mnemonic/shared/http-client.ts"
	cursorTemplateRel = "plugins/mnemonic/cursor/mnemonic.mdc"

	kiloBeginMarker = "<!-- BEGIN SKILLGRID MNEMONIC — managed by skillgrid setup kilocode -->"
	kiloEndMarker   = "<!-- END SKILLGRID MNEMONIC -->"
)

// RunSetup configures Mnemonic for the given agent (opencode, kilocode, cursor).
func RunSetup(agent, repoRoot string, dryRun bool) error {
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
		return SetupOpenCode(home, repoRoot, dryRun)
	case "kilocode", "kilo":
		return SetupKiloCode(home, repoRoot, dryRun)
	case "cursor":
		return SetupCursor(home, dryRun)
	default:
		return fmt.Errorf("unknown agent %q (use opencode, kilocode, or cursor)", agent)
	}
}

// FindRepoRoot walks upward from start (or cwd when empty) for pluginRelPath.
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
		if _, err := os.Stat(filepath.Join(dir, pluginRelPath)); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	if home, err := os.UserHomeDir(); err == nil {
		synced := filepath.Join(home, ".skillgrid", "repos", "skillgrid", pluginRelPath)
		if _, err := os.Stat(synced); err == nil {
			return filepath.Join(home, ".skillgrid", "repos", "skillgrid")
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
