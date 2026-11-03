package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/setup"
	"gopkg.in/yaml.v3"
)

const (
	preambleStart = "<!-- skillgrid-preamble:start -->"
	preambleEnd   = "<!-- skillgrid-preamble:end -->"
	sentinelStart = "<!-- skillgrid:start -->"
	sentinelEnd   = "<!-- skillgrid:end -->"

	claudePointer = "See AGENTS.md — the Skillgrid block there is the source of truth."

	// The ## Skillgrid block's structure is owned by block.md (the single
	// source of truth). The CLI never hardcodes it: loadSentinelTemplate reads
	// it from the installed skill tree at runtime, and fillBlockValues pulls the
	// tracker/memory line values from its "Placeholders" tables. defaultRulesBlock
	// is the only block text that depends on the project's ASSUMPTIONS.md, so it
	// stays here.
	defaultRulesBlock = "No locked constraints yet — see `.skillgrid/ASSUMPTIONS.md`."

	// blockRel is block.md's path relative to the skillgrid repo root.
	blockRel = ".agents/skills/_shared/agent-config/block.md"
)

// blockTemplateRe extracts the markdown-fenced template (the region between
// block.md's first ``` pair). The block's sentinels are inside that fence.
var blockTemplateRe = regexp.MustCompile("(?s)```[^\n]*\n(.*?)```")

func writeBootFile(dir string, force bool) (bootPath, preambleState string, err error) {
	bootPath, err = resolveBootPath(dir)
	if err != nil {
		return "", "", err
	}

	if fi, lstatErr := os.Lstat(bootPath); lstatErr == nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", "", fmt.Errorf("%s: refusing to write through symlink", bootPath)
		}
	} else if !os.IsNotExist(lstatErr) {
		return "", "", lstatErr
	}

	existing := ""
	if data, readErr := os.ReadFile(bootPath); readErr == nil {
		existing = string(data)
	} else if !os.IsNotExist(readErr) {
		return "", "", readErr
	}

	template, blockMDPath, err := loadSentinelTemplate(dir)
	if err != nil {
		return "", "", err
	}
	project := loadProjectName(dir)
	rules := loadRulesBlock(dir)
	blockMD := readBlockMD(blockMDPath)
	sentinel := renderSentinel(project, rules, loadTrackerLine(dir, blockMD), loadMemoryLine(dir, blockMD), template)

	cfg, _ := loadAgentsConfig(dir)
	prem, _, premErr := loadAgentsPreambleTemplate(dir)
	if premErr != nil {
		prem = defaultPreambleTemplate
	}
	body, preambleState := upsertPreamble(existing, force, renderPreamble(cfg, prem))
	body = upsertSentinel(body, sentinel)

	if err := os.WriteFile(bootPath, []byte(body), 0o644); err != nil {
		return "", "", err
	}

	if err := ensureClaudePointer(dir, bootPath); err != nil {
		return "", "", err
	}

	return bootPath, preambleState, nil
}

func resolveBootPath(dir string) (string, error) {
	agents := filepath.Join(dir, "AGENTS.md")
	claude := filepath.Join(dir, "CLAUDE.md")

	_, agentsErr := os.Stat(agents)
	agentsExists := agentsErr == nil
	if agentsErr != nil && !os.IsNotExist(agentsErr) {
		return "", agentsErr
	}

	_, claudeErr := os.Stat(claude)
	claudeExists := claudeErr == nil
	if claudeErr != nil && !os.IsNotExist(claudeErr) {
		return "", claudeErr
	}

	// AGENTS.md if it exists or if CLAUDE.md does not exist; else CLAUDE.md.
	if agentsExists || !claudeExists {
		return agents, nil
	}
	return claude, nil
}

func ensureClaudePointer(dir, bootPath string) error {
	if filepath.Base(bootPath) != "AGENTS.md" {
		return nil
	}
	claude := filepath.Join(dir, "CLAUDE.md")
	data, err := os.ReadFile(claude)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	body := string(data)
	if strings.Contains(body, claudePointer) {
		return nil
	}
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	body += claudePointer + "\n"
	return os.WriteFile(claude, []byte(body), 0o644)
}

// blockMDPath resolves the block.md path to read from, for a given project dir
// (the project's own tree first). It is a var so tests can point init at a
// synthetic or a specific block.md instead of the installed ~/.skillgrid mirror.
var blockMDPath = func(dir string) string {
	return filepath.Join(dir, ".agents", "skills", "_shared", "agent-config", "block.md")
}

// loadSentinelTemplate returns the canonical ## Skillgrid template by reading
// block.md — the single source of truth. Precedence: the project's own tree
// (dir/.agents/skills/...), then the installed mirror / dev checkout
// (setup.FindRepoRoot covers cwd/upward + ~/.skillgrid/repos/skillgrid). If
// none yield a parseable template, init fails rather than writing a stale
// block. No template text lives in the CLI.
func loadSentinelTemplate(dir string) (template, path string, err error) {
	seen := map[string]bool{}
	candidates := []string{blockMDPath(dir)}
	if root := setup.FindRepoRoot(dir); root != "" {
		candidates = append(candidates, filepath.Join(root, blockRel))
	}
	for _, p := range candidates {
		if seen[p] {
			continue
		}
		seen[p] = true
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			continue
		}
		if t := extractBlockTemplate(string(data)); t != "" {
			return t, p, nil
		}
	}
	return "", "", fmt.Errorf("block.md not found in the skill tree (expected at %s)", blockRel)
}

// extractBlockTemplate pulls the markdown-fenced template out of block.md.
// The block's sentinels are inside the first ``` fence.
func extractBlockTemplate(body string) string {
	if m := blockTemplateRe.FindStringSubmatch(body); m != nil {
		return m[1]
	}
	return ""
}

func renderSentinel(project, rulesBlock, trackerLine, memoryLine, template string) string {
	s := template
	if strings.TrimSpace(project) == "" {
		// No known project name (bare dir, no config): drop the "Project: ..."
		// clause so the block doesn't render a literal dot. onboarding fills in
		// the real name later.
		s = strings.ReplaceAll(s, " Project: **{project}**.", "")
	} else {
		s = strings.ReplaceAll(s, "{project}", project)
	}
	s = strings.ReplaceAll(s, "{rules_block}", rulesBlock)
	s = strings.ReplaceAll(s, "{tracker_line}", trackerLine)
	s = strings.ReplaceAll(s, "{memory_line}", memoryLine)
	return s
}

// loadProjectName reads the project: field from .skillgrid/config.yaml so the
// create path names the project by its configured name, not the directory name.
// Falls back to the directory base when the config is absent or has no name; a
// degenerate base ("." for a bare dir) is normalized to "" so the block renders
// "Project: ." as a clean placeholder rather than a literal dot.
func loadProjectName(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".skillgrid", "config.yaml"))
	if err != nil {
		return cleanProjectName(filepath.Base(dir))
	}
	var cfg struct {
		Project string `yaml:"project"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil || strings.TrimSpace(cfg.Project) == "" {
		return cleanProjectName(filepath.Base(dir))
	}
	return cfg.Project
}

// cleanProjectName normalizes degenerate directory bases (".", empty, a
// trailing ".git") so a fresh init does not render a literal dot as the name.
func cleanProjectName(name string) string {
	n := strings.TrimSpace(name)
	if n == "" || n == "." || n == ".." || strings.HasSuffix(n, ".git") {
		return ""
	}
	return n
}

// readBlockMD returns the full body of block.md at blockMDPath (for parsing its
// "Placeholders" tables). blockMDPath is the file loadSentinelTemplate resolved.
func readBlockMD(blockMDPath string) string {
	if blockMDPath == "" {
		return ""
	}
	if data, err := os.ReadFile(blockMDPath); err == nil {
		return string(data)
	}
	return ""
}

// blockLine returns the text after `marker` on the first line that starts with
// it (used for the memory table's "- Enabled:" / "- Disabled:" rows). "" if absent.
func blockLine(body, marker string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, marker) {
			return strings.TrimSpace(strings.TrimPrefix(t, marker))
		}
	}
	return ""
}

// trackerCell returns the second cell of the tracker-table row whose first cell
// is `type` (e.g. "gh", "backlogmd", or "none"). The first cell is wrapped in
// backticks in block.md (e.g. `gh`), so backticks are stripped before matching.
// "" when no such row exists.
func trackerCell(body, typ string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !strings.HasPrefix(t, "|") {
			continue
		}
		cells := strings.Split(t, "|")
		// A row "| a | b |" splits to ["", " a ", " b ", ""].
		if len(cells) < 4 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(cells[1]), "`")
		if key == typ {
			return strings.TrimSpace(cells[2])
		}
	}
	return ""
}

// loadMemoryLine picks the {memory_line} value from block.md's memory table.
// Enabled when the config has mnemonic.enabled: true; inactive otherwise (never
// claim memory is on when the config is absent or says false).
func loadMemoryLine(dir, blockMD string) string {
	enabled := blockLine(blockMD, "- Enabled:")
	disabled := blockLine(blockMD, "- Disabled:")
	if enabled != "" && configMnemonicEnabled(dir) {
		return enabled
	}
	if disabled != "" {
		return disabled
	}
	return "Mnemonic not detected — persistent memory is inactive. Install the `skillgrid` CLI to enable it."
}

func configMnemonicEnabled(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, ".skillgrid", "config.yaml"))
	if err != nil {
		return false
	}
	var cfg struct {
		Mnemonic struct {
			Enabled bool `yaml:"enabled"`
		} `yaml:"mnemonic"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return false
	}
	return cfg.Mnemonic.Enabled
}

type bootTicketingConfig struct {
	Ticketing struct {
		Enabled bool   `yaml:"enabled"`
		Type    string `yaml:"type"`
	} `yaml:"ticketing"`
}

// loadTrackerLine picks the {tracker_line} from block.md's tracker table,
// matching the config's ticketing.type. Falls back to the "none" row when
// ticketing is disabled, the config is absent, or the type is unknown.
func loadTrackerLine(dir, blockMD string) string {
	none := trackerCell(blockMD, "none")
	data, err := os.ReadFile(filepath.Join(dir, ".skillgrid", "config.yaml"))
	if err != nil {
		return fallbackTrackerLine(none)
	}
	var cfg bootTicketingConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil || !cfg.Ticketing.Enabled {
		return fallbackTrackerLine(none)
	}
	return fallbackTrackerLine(trackerCell(blockMD, strings.TrimSpace(cfg.Ticketing.Type)))
}

func fallbackTrackerLine(row string) string {
	if row != "" {
		return row
	}
	return "None — work local-only from tasks.md."
}

func loadRulesBlock(dir string) string {
	path := filepath.Join(dir, ".skillgrid", "ASSUMPTIONS.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultRulesBlock
	}
	bullets := extractLockedConstraints(string(data))
	if len(bullets) == 0 {
		return defaultRulesBlock
	}
	return strings.Join(bullets, "\n")
}

func extractLockedConstraints(body string) []string {
	const heading = "### Locked constraints"
	idx := strings.Index(body, heading)
	if idx < 0 {
		return nil
	}
	rest := body[idx+len(heading):]
	if nl := strings.Index(rest, "\n"); nl >= 0 {
		rest = rest[nl+1:]
	}
	var bullets []string
	for _, line := range strings.Split(rest, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "### ") {
			break
		}
		if strings.HasPrefix(trimmed, "- ") {
			bullets = append(bullets, trimmed)
		}
	}
	return bullets
}

// defaultPreambleTemplate is the fallback structure used only when the
// agents-preamble.md template cannot be resolved from the skill tree (e.g. the
// CLI runs against a fresh checkout without the skill tree). init prefers the
// resolved template; this keeps init from failing on a missing template file.
const defaultPreambleTemplate = "# Standards\n\n## Environment & Tooling\n\n- **Language**: {language}\n- **Install**: {install}\n- **Test**: {test}\n- **Lint**: {lint}\n- **Build (dev)**: {build}\n- **Security scan**: {security_scan}\n\n## Definition of Done\n\n{definition_of_done}\n"

func wrapPreamble(region string) string {
	return preambleStart + "\n" + strings.TrimSpace(region) + "\n" + preambleEnd
}

func ensureBlankLineSuffix(s string) string {
	if s == "" {
		return s
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if !strings.HasSuffix(s, "\n\n") {
		s += "\n"
	}
	return s
}

func upsertPreamble(existing string, force bool, region string) (string, string) {
	start := strings.Index(existing, preambleStart)
	end := strings.Index(existing, preambleEnd)

	if start >= 0 && end > start {
		end += len(preambleEnd)
		if !force {
			return existing, "kept"
		}
		return existing[:start] + wrapPreamble(region) + existing[end:], "forced"
	}

	r := wrapPreamble(region)
	if si := strings.Index(existing, sentinelStart); si >= 0 {
		return ensureBlankLineSuffix(existing[:si]) + r + "\n\n" + existing[si:], "written"
	}
	if existing == "" {
		return r + "\n", "written"
	}
	return ensureBlankLineSuffix(existing) + r + "\n", "written"
}

// upsertSentinel keeps an existing onboarding-rendered ## Skillgrid block as-is.
// The block is owned by the onboarding skill (rendered from block.md); init's job
// is the preamble, ingest, and index — not re-rendering the agent block. The
// canonical template is written only when no sentinel exists yet (greenfield).
func upsertSentinel(existing, sentinel string) string {
	if start := strings.Index(existing, sentinelStart); start >= 0 {
		return existing
	}
	return ensureBlankLineSuffix(existing) + sentinel + "\n"
}
