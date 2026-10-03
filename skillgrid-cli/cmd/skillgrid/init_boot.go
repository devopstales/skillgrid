package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	preambleStart = "<!-- skillgrid-preamble:start -->"
	preambleEnd   = "<!-- skillgrid-preamble:end -->"
	sentinelStart = "<!-- skillgrid:start -->"
	sentinelEnd   = "<!-- skillgrid:end -->"

	claudePointer = "See AGENTS.md — the Skillgrid block there is the source of truth."

	defaultRulesBlock = "No locked constraints yet — see `.skillgrid/ASSUMPTIONS.md`."
	defaultTracker    = "None — work local-only from tasks.md."
	enabledMemoryLine = "Persistent memory is active (Mnemonic). Use `mem_save` for decisions, `mem_search` for recall, `code_status` → `code_index` → `code_search` for code orientation. Full protocol: `skillgrid:mnemonic` skill."

	// Tracker lines match `{tracker_line}` in .agents/skills/_shared/agent-config/block.md.
	trackerBacklog = "Backlog.md — reference `.agents/skills/planning/ticketing/references/backlogmd.md` for conventions."
	trackerGitHub  = "GitHub — reference `.agents/skills/planning/ticketing/references/github.md` for conventions."
	trackerGitLab  = "GitLab — reference `.agents/skills/planning/ticketing/references/gitlab.md` for conventions."
	trackerJira    = "Jira — reference `.agents/skills/planning/ticketing/references/jira.md` for conventions."
)

// Canonical Skillgrid sentinel body (from .agents/skills/_shared/agent-config/block.md).
const sentinelTemplate = `<!-- skillgrid:start -->
## Skillgrid

This project is configured with Skillgrid. Project: **{project}**.

Config: ` + "`.skillgrid/config.yaml`" + ` (static) — read it before running any Skillgrid skill.
State: ` + "`.skillgrid/state.yaml`" + ` (dynamic) — where the project is right now (phase, current change, progress).

### Artifacts

| Artifact | Path |
|----------|------|
| Project knowledge (PRD, architecture, terms, ADRs, constraints, research) | ` + "`.skillgrid/artifacts/`" + ` |
| Project state (phase, current change, progress) | ` + "`.skillgrid/state.yaml`" + ` |
| Specs (briefing, blueprint, tasks) | ` + "`.skillgrid/specs/`" + ` |
| Execution ledger | ` + "`.skillgrid/sdd/`" + ` (gitignored) |

**Domain model:** read the vocabulary + ADRs under ` + "`.skillgrid/artifacts/`" + ` (index: ` + "`README.md`" + `) before designing or implementing.

**Rules & standards:** locked project constraints render under ` + "`### Rules`" + ` below. Shared standards are referenced, never inlined — each loads via the skills that apply it.

- **Coding conventions** — reference ` + "`.agents/skills/_shared/rules/code-standards.md`" + ` for detailed coding conventions.
- **Testing conventions** — reference ` + "`.agents/skills/_shared/references/strict-tdd.md`" + ` for the TDD cycle and testing conventions.
- **Commits & verification** — reference ` + "`.agents/skills/_shared/rules/`" + ` for the commit contract, verification ladder, and rigor tiers.
- **Memory, code index & web cache** — reference the ` + "`skillgrid:mnemonic`" + ` skill (shared rules: ` + "`.agents/skills/_shared/rules/mnemonic-memory.md`" + `).

### Rules

*Locked constraints only (source of truth: ` + "`### Locked constraints`" + ` in ` + "`.skillgrid/ASSUMPTIONS.md`" + ` — edit there, then mirror here).*

{rules_block}

### Issue Tracker

{tracker_line}

### Memory

{memory_line}

### Workflow

` + "`brainstorming` → `writing-blueprints` → `slicing` → `ticketing` → execution (`subagent-execution` or `simple-execution`) → `qa` → `requesting-code-review` → `receiving-code-review` → `ship` → `reflect`" + `

Run ` + "`skillgrid:onboarding`" + ` to update config after stack changes.
<!-- skillgrid:end -->`

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

	project := filepath.Base(dir)
	rules := loadRulesBlock(dir)
	sentinel := renderSentinel(project, rules, loadTrackerLine(dir))

	body, preambleState := upsertPreamble(existing, force, project)
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

func renderSentinel(project, rulesBlock, trackerLine string) string {
	s := sentinelTemplate
	s = strings.ReplaceAll(s, "{project}", project)
	s = strings.ReplaceAll(s, "{rules_block}", rulesBlock)
	s = strings.ReplaceAll(s, "{tracker_line}", trackerLine)
	s = strings.ReplaceAll(s, "{memory_line}", enabledMemoryLine)
	return s
}

type bootTicketingConfig struct {
	Ticketing struct {
		Enabled bool   `yaml:"enabled"`
		Type    string `yaml:"type"`
	} `yaml:"ticketing"`
}

func loadTrackerLine(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, ".skillgrid", "config.yaml"))
	if err != nil {
		return defaultTracker
	}
	var cfg bootTicketingConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil || !cfg.Ticketing.Enabled {
		return defaultTracker
	}
	return trackerLineFor(cfg.Ticketing.Type)
}

func trackerLineFor(kind string) string {
	switch strings.TrimSpace(kind) {
	case "backlogmd":
		return trackerBacklog
	case "gh":
		return trackerGitHub
	case "glab":
		return trackerGitLab
	case "jira":
		return trackerJira
	default:
		return defaultTracker
	}
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

func preambleBody(project string) string {
	return fmt.Sprintf(`# Project Overview
%s.

# Environment & Tooling
- Fill in language, package manager, test and lint commands.

# Engineering Standards
- Add or update tests for behavior changes
- Keep diffs small and reviewable

# Security & Escalation Boundaries
- Work only in this repository
- Do not read secrets or credential stores
- Ask before installing dependencies or changing auth logic

# Dependency Policies
- Prefer the standard library
- New packages require approval

# Architecture Constraints
- See `+"`"+`.skillgrid/ASSUMPTIONS.md`+"`"+` when present

# Definition of Done
- Tests pass
- Lint and formatting pass
- Update the spec if behavior changes`, project)
}

func wrapPreamble(project string) string {
	return preambleStart + "\n" + preambleBody(project) + "\n" + preambleEnd
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

func upsertPreamble(existing string, force bool, project string) (string, string) {
	start := strings.Index(existing, preambleStart)
	end := strings.Index(existing, preambleEnd)

	if start >= 0 && end > start {
		end += len(preambleEnd)
		if !force {
			return existing, "kept"
		}
		return existing[:start] + wrapPreamble(project) + existing[end:], "forced"
	}

	region := wrapPreamble(project)
	if si := strings.Index(existing, sentinelStart); si >= 0 {
		return ensureBlankLineSuffix(existing[:si]) + region + "\n\n" + existing[si:], "written"
	}
	if existing == "" {
		return region + "\n", "written"
	}
	return ensureBlankLineSuffix(existing) + region + "\n", "written"
}

func upsertSentinel(existing, sentinel string) string {
	start := strings.Index(existing, sentinelStart)
	end := strings.Index(existing, sentinelEnd)
	if start >= 0 && end > start {
		end += len(sentinelEnd)
		return existing[:start] + sentinel + existing[end:]
	}
	return ensureBlankLineSuffix(existing) + sentinel + "\n"
}
