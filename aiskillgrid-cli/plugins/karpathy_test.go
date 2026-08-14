package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func TestInstallKarpathyForAgent_CopiesSkillAndRule(t *testing.T) {
	checkout := t.TempDir()
	skillSrc := filepath.Join(checkout, "skills", KarpathySkillName)
	if err := os.MkdirAll(skillSrc, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillSrc, "SKILL.md"), []byte("# karpathy\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ruleSrc := filepath.Join(checkout, ".cursor", "rules", KarpathyRuleFile)
	if err := os.MkdirAll(filepath.Dir(ruleSrc), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruleSrc, []byte("---\nalwaysApply: true\n---\n# rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	project := t.TempDir()
	opts := Options{
		Scope:      home.ScopeProject,
		ProjectDir: project,
		HomeRoot:   t.TempDir(),
		ConfigDir:  t.TempDir(),
	}

	for _, agent := range []string{"cursor", "kilo", "opencode", "copilot"} {
		paths, err := installKarpathyForAgent(agent, opts, skillSrc, ruleSrc)
		if err != nil {
			t.Fatalf("%s: %v", agent, err)
		}
		if len(paths) != 2 {
			t.Fatalf("%s: paths=%v", agent, paths)
		}
		for _, p := range paths {
			if _, err := os.Stat(p); err != nil {
				t.Fatalf("%s missing %s: %v", agent, p, err)
			}
		}
	}

	// Copilot should rename .mdc → .instructions.md
	copilotRule := filepath.Join(project, ".github", "instructions", "karpathy-guidelines.instructions.md")
	if _, err := os.Stat(copilotRule); err != nil {
		t.Fatal(err)
	}
}

func TestAgentSkillRuleDirs_Unknown(t *testing.T) {
	_, _, _, err := agentSkillRuleDirs("claude", Options{})
	if err == nil {
		t.Fatal("expected error")
	}
}
