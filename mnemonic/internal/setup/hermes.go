package setup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Hermes constants (2026-10-24-hermes-integration, TICKET-02).
const (
	// HermesHomeEnv relocates all Hermes state (config.yaml, SOUL.md,
	// plugins/, skills/, kanban.db) away from the default ~/.hermes.
	HermesHomeEnv = "HERMES_HOME"

	// ExcludeEnvVar is set on the mnemonic MCP entry so the MCP server
	// suppresses the mem_* tools (handled natively by the Hermes
	// MemoryProvider instead).
	ExcludeEnvVar = "MNEMONIC_MCP_EXCLUDE"

	// KanbanTicketType is the ticketing.type value SetupHermes writes to
	// .skillgrid/config.yaml so the ticketing skill dispatches to the
	// Hermes Kanban backend.
	KanbanTicketType = "hermes-kanban"

	hermesBeginMarker = "<!-- BEGIN SKILLGRID MNEMONIC -->"
	hermesEndMarker   = "<!-- END SKILLGRID MNEMONIC -->"
)

// SetupHermes configures Mnemonic for Hermes Agent (2026-10-24-hermes-
// integration, TICKET-02). Seven operations, each dry-run aware:
//
//  1. Copy the MemoryProvider plugin (plugins/hermes/) to
//     $HERMES_HOME/plugins/mnemonic/.
//  2. Write the mcp_servers block to $HERMES_HOME/config.yaml (YAML).
//     Every server from config.d/mcp.yaml is registered; the mnemonic
//     entry additionally carries env MNEMONIC_MCP_EXCLUDE=mem so the MCP
//     server hides the mem_* tools the provider handles natively.
//  3. Write the SOUL.md marker-delimited protocol block (idempotent: a
//     second run replaces the block in place, never duplicates it).
//  4. Create the project's Kanban board via `hermes kanban boards create
//     <slug>` (non-fatal: a failure only warns, the install continues).
//  5. Copy .agents/skills/** to $HERMES_HOME/skills/. A failure here only
//     warns (the memory + MCP setup already completed).
//  6. Update .skillgrid/config.yaml: ticketing.type=hermes-kanban plus
//     ticketing.kanban.board/assignee (skip when ticketing is disabled).
func SetupHermes(home, repoRoot string, mcpEntries []MCPServerConfig, dryRun bool) error {
	if repoRoot == "" {
		repoRoot = FindRepoRoot("")
	}
	if repoRoot == "" {
		return fmt.Errorf("repo root not found (run from skillgrid checkout or sync repo)")
	}
	hermesHome := hermesHome(home)

	// 1. MemoryProvider plugin copy.
	pluginRel := "plugins/hermes"
	pluginSrc := filepath.Join(repoRoot, pluginRel)
	if staged := stagedPluginPath(pluginRel + "/__init__.py"); staged != "" {
		pluginSrc = filepath.Dir(staged)
	}
	if err := copyDir(pluginSrc, filepath.Join(hermesHome, "plugins", "mnemonic"), dryRun); err != nil {
		return err
	}

	// 2. MCP servers in $HERMES_HOME/config.yaml (YAML). The pre-write
	// backup covers the well-formed path; the parse-failure path takes its
	// own backup (see upsertHermesMCP).
	cfgPath := filepath.Join(hermesHome, "config.yaml")
	if err := backupConfigFile(home, "hermes", cfgPath, dryRun); err != nil {
		return err
	}
	if err := upsertHermesMCP(home, cfgPath, mcpEntries, dryRun); err != nil {
		return err
	}

	// 3. SOUL.md protocol block (fallback when the provider is not active).
	if err := writeHermesSoul(hermesHome, repoRoot, dryRun); err != nil {
		return err
	}

	// 4. Kanban board (non-fatal).
	slug, name := kanbanProject(repoRoot)
	if err := createKanbanBoard(hermesHome, slug, name, dryRun); err != nil {
		logInfof("warning: kanban board: %v", err)
	}

	// 5. Skills copy (non-fatal: memory + MCP are already in place).
	if err := copyDir(filepath.Join(repoRoot, ".agents", "skills"), filepath.Join(hermesHome, "skills"), dryRun); err != nil {
		logInfof("warning: skills copy: %v", err)
	}

	// 6. Ticketing config.
	if err := updateHermesTicketing(repoRoot, slug, dryRun); err != nil {
		return err
	}

	// 7. Cron templates as next-steps (informational, never auto-created).
	if dryRun {
		logInfo("[dry-run] install complete; next steps below")
	} else {
		logInfo("hermes install complete")
	}
	printCronTemplates(slug)
	return nil
}

// printCronTemplates prints the memory-maintenance cron jobs to set up after
// a hermes install (TICKET-05). Informational only: the jobs are never
// created here — the user runs the commands when (and if) they want the
// scheduler to do maintenance. Command shapes verified against
// `hermes cron create --help` (v0.22.5): `hermes cron create <schedule>
// <prompt> [--name <name>]`; `--workdir` pins the project cwd so the agent
// resolves the right repo.
func printCronTemplates(slug string) {
	logInfo("Next steps — optional cron jobs (not created automatically):")
	logInfo("  # Nightly memory maintenance for project " + slug + ":")
	logInfo("  hermes cron create \"0 2 * * *\" \"Run the mnemonic memory protocol maintenance pass for project " + slug + ": review recent observations, flag stale needs_review entries, and record what you changed.\" --name mnemonic-maintenance --workdir <repo-path>")
	logInfo("  # Weekly prune + distill (heavier LLM pass):")
	logInfo("  hermes cron create \"0 3 * * 0\" \"Run the weekly mnemonic prune + distill pass for project " + slug + ": compact superseded observations and merge duplicate topic_keys.\" --name mnemonic-prune --workdir <repo-path>")
	logInfo("  Manage jobs afterwards: hermes cron list | hermes cron runs | hermes cron doctor")
}

// hermesHome resolves $HERMES_HOME (relocates all Hermes state) with the
// default ~/.hermes under home.
func hermesHome(home string) string {
	if v := os.Getenv(HermesHomeEnv); v != "" {
		return v
	}
	return filepath.Join(home, ".hermes")
}

// hmcEntry is one MCP server entry inside the mcp_servers block of
// $HERMES_HOME/config.yaml: remote servers carry a URL, local servers a
// command. A command entry may carry extra env for its process.
type hmcEntry struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// upsertHermesMCP reads $HERMES_HOME/config.yaml as generic YAML, replaces
// the mcp_servers block with the entries from config.d/mcp.yaml (the
// mnemonic entry gets MNEMONIC_MCP_EXCLUDE=mem), and writes it back. Every
// other key in the file is preserved. A missing file is created.
//
// On a YAML parse error the original file is backed up and left untouched
// (the caller already took the pre-write backup; the parse failure means
// the merge cannot be trusted).
func upsertHermesMCP(home string, cfgPath string, entries []MCPServerConfig, dryRun bool) error {
	// Parse into a map so every top-level key (model, provider, …) is
	// preserved — a struct would silently drop keys it does not declare.
	doc := map[string]interface{}{}
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := yaml.Unmarshal(data, &doc); err != nil {
			// Malformed YAML: back the original up and leave it untouched —
			// the merge cannot be trusted, so report and stop.
			if berr := backupConfigFile(home, "hermes", cfgPath, false); berr != nil {
				return fmt.Errorf("parse %s: %w (backup also failed: %v)", cfgPath, err, berr)
			}
			return fmt.Errorf("parse %s: %w (file left unchanged)", cfgPath, err)
		}
	}
	if len(entries) == 0 {
		return nil
	}
	servers := map[string]hmcEntry{}
	if existing, ok := doc["mcp_servers"].(map[string]interface{}); ok {
		// Preserve user-added servers (e.g. ones the installer does not
		// manage); managed entries below are replaced wholesale.
		for name, raw := range existing {
			e := hmcEntry{}
			if m, ok := raw.(map[string]interface{}); ok {
				if v, ok := m["name"].(string); ok {
					e.Name = v
				}
				if v, ok := m["url"].(string); ok {
					e.URL = v
				}
				if v, ok := m["command"].(string); ok {
					e.Command = v
				}
				if v, ok := m["args"].([]interface{}); ok {
					e.Args = make([]string, 0, len(v))
					for _, a := range v {
						if s, ok := a.(string); ok {
							e.Args = append(e.Args, s)
						}
					}
				}
				if v, ok := m["env"].(map[string]interface{}); ok {
					e.Env = make(map[string]string)
					for k, val := range v {
						if s, ok := val.(string); ok {
							e.Env[k] = s
						}
					}
				}
			}
			servers[name] = e
		}
	}
	for _, e := range entries {
		servers[e.Name] = hmcEntryFromMCPServer(e)
	}
	doc["mcp_servers"] = servers
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal hermes config: %w", err)
	}
	if dryRun {
		logInfo("[dry-run] set mcp_servers (" + fmt.Sprintf("%d", len(entries)) + " servers) in " + cfgPath)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", cfgPath, err)
	}
	logInfo("wrote mcp_servers to " + cfgPath)
	return nil
}

// hmcEntryFromMCPServer renders one MCP entry: remote -> {url}, local ->
// {command, args}. The mnemonic entry additionally pins env
// MNEMONIC_MCP_EXCLUDE=mem so `mnemonic mcp` suppresses the mem_* tools.
func hmcEntryFromMCPServer(e MCPServerConfig) hmcEntry {
	h := hmcEntry{Name: e.Name}
	if e.Type == "remote" {
		h.URL = e.URL
	} else {
		if len(e.Command) > 0 {
			h.Command = e.Command[0]
		}
		if len(e.Command) > 1 {
			h.Args = e.Command[1:]
		}
	}
	// "mnemonic" is the server name from config.d/mcp.yaml (the map key that
	// becomes MCPServerConfig.Name).
	if e.Name == "mnemonic" {
		h.Env = map[string]string{ExcludeEnvVar: "mem"}
	}
	return h
}

// writeHermesSoul writes the marker-delimited protocol block into
// $HERMES_HOME/SOUL.md. The block is replaced in place on re-run (idempotent);
// surrounding SOUL.md content is preserved.
func writeHermesSoul(hermesHome, repoRoot string, dryRun bool) error {
	protocol := ProtocolMarkdownFromRepo(repoRoot)
	if protocol == "" {
		return fmt.Errorf("memory protocol not found (sync repo or run from checkout)")
	}
	path := filepath.Join(hermesHome, "SOUL.md")
	var content string
	if data, err := os.ReadFile(path); err == nil {
		content = string(data)
	}
	updated := upsertMarkerBlock(content, hermesBeginMarker, hermesEndMarker, protocol)
	if dryRun {
		logInfo("[dry-run] write " + path)
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	logInfo("wrote SOUL.md protocol block to " + path)
	return nil
}

// kanbanProject derives the board slug (the .skillgrid config `project:`
// name, defaulting to the repo dir name) and a display name.
func kanbanProject(repoRoot string) (slug, name string) {
	name = filepath.Base(repoRoot)
	if data, err := os.ReadFile(filepath.Join(repoRoot, ".skillgrid", "config.yaml")); err == nil {
		var raw struct {
			Project string `yaml:"project"`
		}
		if yaml.Unmarshal(data, &raw) == nil && raw.Project != "" {
			slug = raw.Project
			name = raw.Project
		}
	}
	if slug == "" {
		slug = name
	}
	return slug, name
}

// KanbanExec is the `hermes` CLI invoked for Kanban board creation; tests
// (in this package or the skillgrid-cli installer) replace it with a stub so
// the suites stay hermetic on machines with a real hermes binary on PATH.
var KanbanExec = func(args ...string) (string, error) {
	return runKanbanCLI(args...)
}

// kanbanTimeout bounds a real CLI call: a hung hermes (e.g. an interrupted
// first-run dependency install) must not stall the whole installer.
var kanbanTimeout = 60 * time.Second

// runKanbanCLI runs `hermes <args...>` with the board's state rooted at
// $HERMES_HOME (set by the caller) and the process detached from the
// installer's stdin.
func runKanbanCLI(args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), kanbanTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "hermes", args...)
	cmd.Env = os.Environ()
	devNull, err := os.OpenFile(os.DevNull, os.O_RDONLY, 0)
	if err == nil {
		cmd.Stdin = devNull
		defer devNull.Close()
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), err
	}
	return strings.TrimSpace(string(out)), nil
}

// createKanbanBoard creates the project's Kanban board via
// `hermes kanban boards create <slug> --name <name>`. Non-fatal by
// contract: any failure (hermes missing, board already exists) is an
// error the caller turns into a warning — the install continues.
func createKanbanBoard(hermesHome, slug, name string, dryRun bool) error {
	args := []string{"kanban", "boards", "create", slug}
	if name != "" && name != slug {
		args = append(args, "--name", name)
	}
	if dryRun {
		logInfo("[dry-run] hermes " + strings.Join(args, " "))
		return nil
	}
	// Pin HERMES_HOME for the subprocess so the board lands under the
	// target home even when the installer's own env differs.
	prev := os.Getenv(HermesHomeEnv)
	_ = os.Setenv(HermesHomeEnv, hermesHome)
	out, err := KanbanExec(args...)
	if prev == "" {
		_ = os.Unsetenv(HermesHomeEnv)
	} else {
		_ = os.Setenv(HermesHomeEnv, prev)
	}
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	logInfo("kanban board " + slug + " ready")
	return nil
}

// updateHermesTicketing rewrites ticketing.type to hermes-kanban in
// .skillgrid/config.yaml and records the board slug + assignee profile
// (ticketing.kanban.board / .assignee). The text-level merge preserves
// every comment and key a YAML round-trip would flatten. A missing or
// disabled ticketing block is left untouched (nothing to dispatch to).
func updateHermesTicketing(repoRoot, slug string, dryRun bool) error {
	path := filepath.Join(repoRoot, ".skillgrid", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil // no skillgrid project here; nothing to update
	}
	var raw struct {
		Ticketing struct {
			Enabled bool `yaml:"enabled"`
			Type    string
			Kanban  struct {
				Board    string
				Assignee string
			}
		} `yaml:"ticketing"`
	}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if !raw.Ticketing.Enabled {
		return nil
	}
	merged := mergeTicketingType(string(data), KanbanTicketType, slug)
	if merged == string(data) {
		return nil // already installed
	}
	if dryRun {
		logInfo("[dry-run] set ticketing.type=" + KanbanTicketType + " in " + path)
		return nil
	}
	if err := os.WriteFile(path, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	logInfo("ticketing.type -> " + KanbanTicketType + " in " + path)
	return nil
}

// mergeTicketingType does a surgical text merge in the config: the
// ticketing block's `type:` line becomes hermes-kanban, and
// kanban.board/assignee are set inside the block (or the block is
// created when absent) — preserving every comment and key a YAML
// round-trip would flatten. Returns the merged text (identical to doc
// when already installed).
func mergeTicketingType(doc, ticketType, slug string) string {
	lines := strings.Split(doc, "\n")
	out := make([]string, 0, len(lines)+5)
	inTicketing := false
	typeSet := false
	kanbanSet := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimLeft(line, " \t")
		if inTicketing && line != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			inTicketing = false // dedent: the ticketing block ended
		}
		if !inTicketing {
			if trimmed == "ticketing:" {
				inTicketing = true
			}
			out = append(out, line)
			continue
		}
		if !typeSet && strings.HasPrefix(trimmed, "type:") {
			out = append(out, "  type: "+ticketType)
			typeSet = true
			continue
		}
		if !kanbanSet && trimmed == "kanban:" {
			out = append(out, line)
			// Keep the kanban sub-keys, forcing board + assignee.
			boardSet := false
			assigneeSet := false
			for i+1 < len(lines) {
				sub := lines[i+1]
				if sub == "" || !strings.HasPrefix(sub, "    ") {
					break
				}
				subT := strings.TrimLeft(sub, " ")
				i++
				if strings.HasPrefix(subT, "board:") {
					out = append(out, "    board: "+slug)
					boardSet = true
					continue
				}
				if strings.HasPrefix(subT, "assignee:") {
					out = append(out, "    assignee: default")
					assigneeSet = true
					continue
				}
				out = append(out, sub)
			}
			if !boardSet {
				out = append(out, "    board: "+slug)
			}
			if !assigneeSet {
				out = append(out, "    assignee: default")
			}
			kanbanSet = true
			continue
		}
		out = append(out, line)
	}
	if inTicketing {
		if !kanbanSet {
			out = append(out, "  kanban:")
			out = append(out, "    board: "+slug)
			out = append(out, "    assignee: default")
		}
		if !typeSet {
			out = append(out, "  type: "+ticketType)
		}
	}
	if !typeSet {
		// No ticketing block at all: append a minimal one.
		out = append(out, "", "ticketing:", "  enabled: true", "  type: "+ticketType,
			"  kanban:", "    board: "+slug, "    assignee: default")
	}
	return strings.Join(out, "\n")
}

// copyDir copies every file under src into dst, preserving the relative
// layout. Existing dst files are overwritten (re-install refreshes the
// content); __pycache__ dirs are skipped. Missing src is a no-op.
func copyDir(src, dst string, dryRun bool) error {
	info, err := os.Stat(src)
	if err != nil || !info.IsDir() {
		return nil
	}
	err = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if dryRun {
			logInfo("[dry-run] cp " + p + " " + target)
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o644)
		if st, err2 := os.Stat(p); err2 == nil {
			mode = st.Mode().Perm()
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(target, data, mode); err != nil {
			return fmt.Errorf("write %s: %w", target, err)
		}
		if err := os.Chmod(target, mode); err != nil {
			return err
		}
		logInfo("copied " + target)
		return nil
	})
	if err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	return nil
}
