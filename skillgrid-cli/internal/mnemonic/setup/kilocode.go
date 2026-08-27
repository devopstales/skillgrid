package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"
)

// SetupKiloCode registers MCP, writes the AGENTS.md protocol block, and installs the HTTP plugin.
func SetupKiloCode(home, repoRoot string, dryRun bool) error {
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	cfgPath := agentConfigPath(home, "kilo")
	if err := ensureConfigFile(cfgPath, dryRun); err != nil {
		return err
	}
	if err := upsertOpenCodeMCP(cfgPath, dryRun); err != nil {
		return err
	}

	protocol := ProtocolMarkdownFromRepo(repoRoot)
	if protocol == "" {
		return fmt.Errorf("memory protocol not found (sync repo or run from checkout)")
	}

	agentsPath := filepath.Join(home, ".config", "kilo", "AGENTS.md")
	var content []byte
	if data, err := os.ReadFile(agentsPath); err == nil {
		content = data
	}
	updated := upsertMarkerBlock(string(content), kiloBeginMarker, kiloEndMarker, protocol)
	if dryRun {
		logging.Info("[dry-run] write " + agentsPath)
	} else {
		if err := os.MkdirAll(filepath.Dir(agentsPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(agentsPath, []byte(updated), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", agentsPath, err)
		}
	}

	kiloDir := filepath.Join(home, ".config", "kilo")
	pluginDst := filepath.Join(kiloDir, "plugins", "mnemonic.ts")
	sharedDst := filepath.Join(kiloDir, "shared", "http-client.ts")

	if err := copyFromRepo(repoRoot, pluginRelPath, pluginDst, dryRun); err != nil {
		return err
	}
	if err := copyFromRepo(repoRoot, httpClientRelPath, sharedDst, dryRun); err != nil {
		return err
	}

	opencodeDir := filepath.Join(home, ".config", "opencode")
	bridges := []struct{ src, dst string }{
		{
			filepath.Join(opencodeDir, "plugins", "mnemonic.ts"),
			pluginDst,
		},
		{
			filepath.Join(opencodeDir, "shared", "http-client.ts"),
			sharedDst,
		},
		{
			filepath.Join(opencodeDir, "tui.json"),
			filepath.Join(kiloDir, "tui.json"),
		},
	}
	for _, b := range bridges {
		if err := copyFirstWriteWins(b.src, b.dst, dryRun); err != nil {
			return err
		}
	}

	pluginRef := tildePath(home, pluginDst)
	return appendPluginPath(cfgPath, pluginRef, dryRun)
}
