package install

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tidwall/sjson"
)

// piAgentSettingsPath resolves the skill-pi agent settings.json (default
// ~/.skill-pi/agent/settings.json; SKILL_PI_AGENT_DIR honored, mirroring
// skill-pi's own lib/config.js resolution and piLSPConfPath).
func piAgentSettingsPath(home string) (string, error) {
	agentDir := filepath.Join(home, ".skill-pi", "agent")
	if v := os.Getenv("SKILL_PI_AGENT_DIR"); v != "" {
		agentDir = v
	}
	return filepath.Join(agentDir, "settings.json"), nil
}

// piHooksBlock returns the Claude Code command-hook block the pi-hooks
// extension loads from settings.json "hooks": the skillgrid hook scripts
// under <hooksDir> speak the harness-agnostic stdin/stdout JSON contract.
func piHooksBlock(hooksDir string) string {
	node := "node "
	return fmt.Sprintf(`{"SessionStart":[{"hooks":[{"type":"command","command":%q}]}],"Stop":[{"hooks":[{"type":"command","command":%q},{"type":"command","command":%q}]}],"PostToolUse":[{"hooks":[{"type":"command","command":%q}]}],"PreToolUse":[{"matcher":"bash","hooks":[{"type":"command","if":"Bash(git *)","command":%q}]}]}`,
		node+filepath.Join(hooksDir, "session-start.js"),
		node+filepath.Join(hooksDir, "gate-stop.js"),
		node+filepath.Join(hooksDir, "stop-tests.js"),
		node+filepath.Join(hooksDir, "tool-call-capture.js"),
		node+filepath.Join(hooksDir, "ticketing-guard.js"),
	)
}

// installPiHookBridge writes (merges) the "hooks" block into the skill-pi
// agent settings.json so the pi-hooks extension wires the skillgrid hook
// scripts. We own the top-level "hooks" key: a pre-existing value is replaced
// wholesale; every other key survives. The settings file is created when
// missing. The caller (Run) has already backed up settings.json via
// backupAgentConfigs, so the pre-write snapshot is in place.
func installPiHookBridge(c *Config) error {
	home, err := configHome(c)
	if err != nil {
		return err
	}
	settingsPath, err := piAgentSettingsPath(home)
	if err != nil {
		return err
	}
	hooksDir := filepath.Join(home, ".skillgrid", "hooks")
	block := piHooksBlock(hooksDir)

	var raw []byte
	if data, err := os.ReadFile(settingsPath); err == nil {
		raw = data
	} else {
		raw = []byte("{}")
	}
	updated, err := sjson.SetRawBytes(raw, "hooks", []byte(block))
	if err != nil {
		return fmt.Errorf("set hooks: %w", err)
	}
	if c.DryRun {
		Out("      [dry-run] set hooks in", settingsPath)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", settingsPath, err)
	}
	Out("      wrote hooks in", settingsPath)
	return nil
}
