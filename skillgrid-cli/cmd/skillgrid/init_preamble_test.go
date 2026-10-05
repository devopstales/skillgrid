package main

import (
	"os"
	"strings"
	"testing"
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
