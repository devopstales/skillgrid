package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tidwall/sjson"

)

// SetupCursor registers MCP servers from config.d/mcp.yaml, adds the
// session/tool-call/policy hooks to ~/.cursor/hooks.json, and copies the
// always-applied Mnemonic rule from rules/mnemonic.mdc.
func SetupCursor(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		repoRoot = FindRepoRoot("")
	}
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}

	mcpPath := filepath.Join(home, ".cursor", "mcp.json")
	if err := ensureConfigFile(mcpPath, dryRun); err != nil {
		return err
	}
	if err := backupConfigFile(home, "cursor", mcpPath, dryRun); err != nil {
		return err
	}
	for _, entry := range mcpEntries {
		if err := upsertCursorMCP(mcpPath, entry, dryRun); err != nil {
			return err
		}
	}

	if err := installCursorHookScripts(home, repoRoot, dryRun); err != nil {
		return err
	}
	if err := upsertCursorHooks(home, dryRun); err != nil {
		return err
	}

	ruleSrc := filepath.Join(repoRoot, cursorRuleRel)
	body, err := os.ReadFile(ruleSrc)
	if err != nil {
		return fmt.Errorf("read cursor rule %s: %w", cursorRuleRel, err)
	}

	rulePath := filepath.Join(home, ".cursor", "rules", "mnemonic.mdc")
	if dryRun {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(rulePath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(rulePath, body, 0o644)
}

// cursorHookScripts maps each Cursor hook event to the script under
// ~/.skillgrid/hooks/ (mirrored by `skillgrid install`) that handles it. The
// same map is the repo's hooks/hooks-cursor.json for the Cursor plugin; the
// user-level entries make the hooks run whether or not the plugin is enabled.
var cursorHookScripts = map[string]string{
	"sessionStart":         "cursor-session-start.sh",
	"sessionEnd":           "cursor-session-end.sh",
	"stop":                 "cursor-session-end.sh",
	"postToolUse":          "cursor-tool-capture.sh",
	"beforeShellExecution": "cursor-policy.sh",
	"beforeMCPExecution":   "cursor-policy.sh",
	"beforeReadFile":       "cursor-policy.sh",
}

// CursorHookCommand is the hooks.json command for one skillgrid script.
func CursorHookCommand(home, script string) string {
	return cursorHookCommand(home, script, "")
}

func cursorHookCommand(home, script, event string) string {
	cmd := "bash " + filepath.Join(home, ".skillgrid", "hooks", script)
	if script == "cursor-session-end.sh" && event != "" {
		cmd += " " + event
	}
	return cmd
}

func cursorHookEntry(home, script, event string) map[string]any {
	entry := map[string]any{"command": cursorHookCommand(home, script, event)}
	if event == "stop" {
		entry["loop_limit"] = 2
	}
	return entry
}

// upsertCursorHooks adds the skillgrid entries to ~/.cursor/hooks.json,
// keeping every other hook. An event that already runs the script (any path
// ending in .skillgrid/hooks/<script>) is left alone, so re-running setup is
// idempotent.
// installCursorHookScripts copies the hook scripts referenced by
// cursorHookScripts (plus the shared tool-call-capture.js they run) from the
// repo's hooks/ dir into ~/.skillgrid/hooks/ so a standalone
// `skillgrid setup cursor` works without a prior `skillgrid install` mirror.
// Missing sources are skipped: an older checkout simply installs fewer hooks.
func installCursorHookScripts(home, repoRoot string, dryRun bool) error {
	names := map[string]bool{"tool-call-capture.js": true}
	for _, s := range cursorHookScripts {
		names[s] = true
	}
	sorted := make([]string, 0, len(names))
	for n := range names {
		sorted = append(sorted, n)
	}
	sort.Strings(sorted)
	dstDir := filepath.Join(home, ".skillgrid", "hooks")
	for _, name := range sorted {
		src := filepath.Join(repoRoot, "hooks", name)
		data, err := os.ReadFile(src)
		if err != nil {
			continue
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

func upsertCursorHooks(home string, dryRun bool) error {
	path := filepath.Join(home, ".cursor", "hooks.json")
	if err := ensureConfigFile(path, dryRun); err != nil {
		return err
	}
	if err := backupConfigFile(home, "cursor", path, dryRun); err != nil {
		return err
	}
	doc := map[string]any{}
	if data, err := os.ReadFile(path); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		if err := json.Unmarshal(data, &doc); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
	}
	hooks, _ := doc["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	changed := false
	events := make([]string, 0, len(cursorHookScripts))
	for ev := range cursorHookScripts {
		events = append(events, ev)
	}
	sort.Strings(events)
	for _, ev := range events {
		script := cursorHookScripts[ev]
		entries, _ := hooks[ev].([]any)
		if idx := cursorSkillgridHookIndex(entries, script); idx >= 0 {
			if ev == "stop" {
				m, _ := entries[idx].(map[string]any)
				if m == nil {
					m = map[string]any{}
				}
				if _, ok := m["loop_limit"]; !ok {
					m["loop_limit"] = 2
					entries[idx] = m
					hooks[ev] = entries
					changed = true
				}
			}
			continue
		}
		hooks[ev] = append(entries, cursorHookEntry(home, script, ev))
		changed = true
	}
	if !changed {
		return nil
	}
	doc["hooks"] = hooks
	if _, ok := doc["version"]; !ok {
		doc["version"] = 1
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	if dryRun {
		logInfo("[dry-run] add skillgrid hooks to " + path)
		return nil
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	logInfo("added skillgrid hooks to " + path)
	return nil
}

func cursorSkillgridHookIndex(entries []any, script string) int {
	want := filepath.Join(".skillgrid", "hooks", script)
	for i, e := range entries {
		m, _ := e.(map[string]any)
		cmd, _ := m["command"].(string)
		if strings.Contains(cmd, want) {
			return i
		}
	}
	return -1
}

func upsertCursorMCP(mcpPath string, entry MCPServerConfig, dryRun bool) error {
	data, err := os.ReadFile(mcpPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", mcpPath, err)
	}
	path := "mcpServers." + entry.Name
	var mcpEntry map[string]interface{}
	if entry.Type == "remote" {
		mcpEntry = map[string]interface{}{
			"type": "streamable-http",
			"url":  entry.URL,
		}
	} else {
		var args []interface{}
		if len(entry.Command) > 1 {
			args = make([]interface{}, len(entry.Command)-1)
			for i, c := range entry.Command[1:] {
				args[i] = c
			}
		}
		mcpEntry = map[string]interface{}{
			"command": entry.Command[0],
			"args":    args,
		}
	}
	updated, err := sjson.Set(string(data), path, mcpEntry)
	if err != nil {
		return fmt.Errorf("upsert cursor mcp: %w", err)
	}
	if dryRun {
		return nil
	}
	return os.WriteFile(mcpPath, []byte(updated), 0o644)
}
