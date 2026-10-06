package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

func TestLoadAgentsPreambleTemplate(t *testing.T) {
	prev := agentsPreamblePath
	t.Cleanup(func() { agentsPreamblePath = prev })
	agentsPreamblePath = func(dir string) string {
		return "../../../.agents/skills/lifecycle/onboarding/templates/agents-preamble.md"
	}
	tmpl, _, err := loadAgentsPreambleTemplate(t.TempDir())
	if err != nil {
		t.Fatalf("loadAgentsPreambleTemplate: %v", err)
	}
	for _, want := range []string{"## Environment & Tooling", "## Key Directories",
		"## Security & Escalation Boundaries", "## Dependency Policies",
		"## Architecture Constraints", "## Definition of Done",
		"## Issue Tracker", "## Memory", "## Code Indexing",
		"## Testing standards", "## SDD Standards"} {
		if !strings.Contains(tmpl, want) {
			t.Fatalf("template missing section %q\n%s", want, tmpl)
		}
	}
}

func TestRenderPreambleFillsConfigCommands(t *testing.T) {
	cfg := agentsConfig{
		Commands: commandsCfg{Build: "task build", Lint: "task lint", Format: "task fmt", Typecheck: "task check"},
		Testing:  testingCfg{Runner: "./run_test.sh", Setup: "task install"},
		Security: securityCfg{Trivy: trivyCfg{Command: "task security", Severities: "CRITICAL,HIGH", ScanTypes: "vuln,secret"}},
	}
	out := renderPreamble(cfg, "- **Test**: {test}\n- **Lint**: {lint}")
	if !strings.Contains(out, "- **Test**: ./run_test.sh") {
		t.Fatalf("test command not filled:\n%s", out)
	}
	if !strings.Contains(out, "- **Lint**: task lint") {
		t.Fatalf("lint command not filled:\n%s", out)
	}
}

func TestRenderPreambleDetectFallback(t *testing.T) {
	cfg := agentsConfig{} // all empty
	out := renderPreamble(cfg, "- **Build**: {build}")
	if !strings.Contains(out, "- **Build**: <detect>") {
		t.Fatalf("empty field did not fall back to <detect>:\n%s", out)
	}
}

func TestRenderPreambleDropsEmptyKeyDirectories(t *testing.T) {
	cfg := agentsConfig{} // no directories
	out := renderPreamble(cfg, "## Key Directories\n\n{directories_table}\n\n## Next\n")
	if strings.Contains(out, "## Key Directories") {
		t.Fatalf("empty Key Directories section should be dropped:\n%s", out)
	}
	if !strings.Contains(out, "## Next") {
		t.Fatalf("next section clobbered by drop:\n%s", out)
	}
}

func TestOnboardingSkillDocumentsAgentsPreamble(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "agents.directories") && !strings.Contains(s, "agents:") {
		t.Fatalf("onboarding must document the agents.* config fields:\n%s", s)
	}
	if !strings.Contains(s, "agents-preamble") {
		t.Fatalf("onboarding must name the agents-preamble template")
	}
}

func TestInitWritesRichPreambleFromConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := "project: Demo\ncommands:\n  build: \"task build\"\ntesting:\n  runner: \"./run_test.sh\"\n  setup: \"task install\"\nsecurity:\n  trivy:\n    command: \"task security\"\n    severities: \"CRITICAL\"\n    scan_types: \"vuln\"\n"
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res.BootFile)
	s := string(body)
	for _, want := range []string{"## Environment & Tooling", "## Definition of Done", "## Security & Escalation Boundaries"} {
		if !strings.Contains(s, want) {
			t.Fatalf("preamble missing %q:\n%s", want, s)
		}
	}
	if !strings.Contains(s, "- **Test**: ./run_test.sh") {
		t.Fatalf("test command not from config:\n%s", s)
	}
}

func TestInitPreambleIdempotentAndKeepsSentinel(t *testing.T) {
	dir := t.TempDir()
	onboarded := "Onboarding custom row: keep this."
	agents := "<!-- skillgrid:start -->\n## Skillgrid\n\n" + onboarded + "\n<!-- skillgrid:end -->\n"
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil); err != nil {
		t.Fatal(err)
	}
	res2, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res2.BootFile)
	s := string(body)
	if !strings.Contains(s, onboarded) {
		t.Fatalf("onboarding sentinel clobbered:\n%s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid-preamble:start -->") != 1 {
		t.Fatalf("sentinel/preamble duplicated:\n%s", s)
	}
}

func TestConfigTemplateHasAgentsBlock(t *testing.T) {
	body, err := os.ReadFile("../../../.agents/skills/lifecycle/onboarding/templates/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "agents:") {
		t.Fatalf("config template missing agents: block:\n%s", s)
	}
	for _, key := range []string{"directories:", "security_boundaries:", "dependency_policies:", "architecture_constraints:", "definition_of_done:", "engineering_standards:"} {
		if !strings.Contains(s, key) {
			t.Fatalf("config template missing agents key %q", key)
		}
	}
}
