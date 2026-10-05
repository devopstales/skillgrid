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
