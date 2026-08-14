package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/sync"
)

const (
	KarpathyRepoURL      = "https://github.com/multica-ai/andrej-karpathy-skills.git"
	KarpathyCheckoutName = "andrej-karpathy-skills"
	KarpathySkillName    = "karpathy-guidelines"
	KarpathyRuleFile     = "karpathy-guidelines.mdc"
)

// InstallKarpathyGuidelines clones/updates multica-ai/andrej-karpathy-skills and
// installs the skill + Cursor-style rule into every selected agent.
func InstallKarpathyGuidelines(opts Options) (Result, error) {
	out := Result{Written: map[string][]string{}}
	if opts.DepsDir == "" {
		return out, fmt.Errorf("DepsDir is required")
	}
	checkout := filepath.Join(opts.DepsDir, KarpathyCheckoutName)
	res, err := sync.Sync(checkout, KarpathyRepoURL)
	if err != nil {
		return out, fmt.Errorf("karpathy checkout: %w", err)
	}
	out.Checkout = res.Path
	out.Rev = res.Rev

	skillSrc := filepath.Join(checkout, "skills", KarpathySkillName)
	ruleSrc := filepath.Join(checkout, ".cursor", "rules", KarpathyRuleFile)
	if _, err := os.Stat(skillSrc); err != nil {
		return out, fmt.Errorf("karpathy skill missing at %s: %w", skillSrc, err)
	}
	if _, err := os.Stat(ruleSrc); err != nil {
		return out, fmt.Errorf("karpathy rule missing at %s: %w", ruleSrc, err)
	}

	for _, name := range opts.Agents {
		paths, err := installKarpathyForAgent(name, opts, skillSrc, ruleSrc)
		if err != nil {
			out.Warnings = append(out.Warnings, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if len(paths) > 0 {
			out.Written[name] = paths
		}
	}
	return out, nil
}

func installKarpathyForAgent(name string, opts Options, skillSrc, ruleSrc string) ([]string, error) {
	skillsDir, rulesDir, rulesExt, err := agentSkillRuleDirs(name, opts)
	if err != nil {
		return nil, err
	}
	var written []string

	skillDst := filepath.Join(skillsDir, KarpathySkillName)
	if err := os.RemoveAll(skillDst); err != nil {
		return nil, err
	}
	if err := copyDirSkipGit(skillSrc, skillDst); err != nil {
		return nil, fmt.Errorf("copy skill: %w", err)
	}
	written = append(written, skillDst)

	dstName := KarpathyRuleFile
	if rulesExt != "" {
		dstName = strings.TrimSuffix(KarpathyRuleFile, filepath.Ext(KarpathyRuleFile)) + rulesExt
	}
	ruleDst := filepath.Join(rulesDir, dstName)
	if err := copyFile(ruleSrc, ruleDst); err != nil {
		return written, fmt.Errorf("copy rule: %w", err)
	}
	written = append(written, ruleDst)
	return written, nil
}

func agentSkillRuleDirs(name string, opts Options) (skillsDir, rulesDir, rulesExt string, err error) {
	switch name {
	case "cursor":
		if opts.Scope == home.ScopeProject {
			return filepath.Join(opts.ProjectDir, ".cursor", "skills"),
				filepath.Join(opts.ProjectDir, ".cursor", "rules"),
				"", nil
		}
		return filepath.Join(opts.HomeRoot, ".cursor", "skills"),
			filepath.Join(opts.HomeRoot, ".cursor", "rules"),
			"", nil
	case "kilo":
		if opts.Scope == home.ScopeProject {
			return filepath.Join(opts.ProjectDir, ".kilo", "skills"),
				filepath.Join(opts.ProjectDir, ".kilo", "rules"),
				"", nil
		}
		return filepath.Join(opts.HomeRoot, ".kilo", "skills"),
			filepath.Join(opts.HomeRoot, ".kilo", "rules"),
			"", nil
	case "opencode":
		if opts.Scope == home.ScopeProject {
			return filepath.Join(opts.ProjectDir, ".opencode", "skills"),
				filepath.Join(opts.ProjectDir, ".opencode", "rules"),
				"", nil
		}
		return filepath.Join(opts.ConfigDir, "opencode", "skills"),
			filepath.Join(opts.ConfigDir, "opencode", "rules"),
			"", nil
	case "copilot":
		if opts.Scope == home.ScopeProject {
			return filepath.Join(opts.ProjectDir, ".github", "skills"),
				filepath.Join(opts.ProjectDir, ".github", "instructions"),
				".instructions.md", nil
		}
		return filepath.Join(opts.HomeRoot, ".copilot", "skills"),
			filepath.Join(opts.HomeRoot, ".copilot", "instructions"),
			".instructions.md", nil
	default:
		return "", "", "", fmt.Errorf("unknown agent %q", name)
	}
}
