package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBootFileLinksSelectedTrackerConvention(t *testing.T) {
	dir := t.TempDir()
	writeTicketingConfig(t, dir, "ticketing:\n  enabled: true\n  type: backlogmd\n")

	body := bootFileBody(t, dir)
	// Line text comes from block.md's tracker table, not from the CLI.
	const want = "Backlog.md — reference `.agents/skills/_shared/rules/ticketing/` (backlogmd, github, gitlab, jira CLI + formatting standards) for ticketing conventions."
	if !strings.Contains(body, want) {
		t.Fatalf("boot file missing tracker convention pointer:\n%s", body)
	}
	const none = "None — work local-only from tasks.md."
	if strings.Contains(body, none) {
		t.Fatalf("boot file kept the no-tracker line:\n%s", body)
	}
}

func TestWriteBootFileTrackerLineFollowsConfig(t *testing.T) {
	cases := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "github",
			yaml: "ticketing:\n  enabled: true\n  type: gh\n",
			want: "GitHub — reference `.agents/skills/_shared/rules/ticketing/github.md` for conventions.",
		},
		{
			name: "gitlab",
			yaml: "ticketing:\n  enabled: true\n  type: glab\n",
			want: "GitLab — reference `.agents/skills/_shared/rules/ticketing/gitlab.md` for conventions.",
		},
		{
			name: "jira",
			yaml: "ticketing:\n  enabled: true\n  type: jira\n",
			want: "Jira — reference `.agents/skills/_shared/rules/ticketing/jira.md` for conventions.",
		},
		{
			name: "disabled keeps local-only line",
			yaml: "ticketing:\n  enabled: false\n  type: backlogmd\n",
			want: "None — work local-only from tasks.md.",
		},
		{
			name: "unknown type",
			yaml: "ticketing:\n  enabled: true\n  type: linear\n",
			want: "None — work local-only from tasks.md.",
		},
		{
			name: "missing config",
			yaml: "",
			want: "None — work local-only from tasks.md.",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if tc.yaml != "" {
				writeTicketingConfig(t, dir, tc.yaml)
			}
			body := bootFileBody(t, dir)
			if !strings.Contains(body, tc.want) {
				t.Fatalf("boot file missing %q:\n%s", tc.want, body)
			}
		})
	}
}

func writeTicketingConfig(t *testing.T, dir, yaml string) {
	t.Helper()
	cfgDir := filepath.Join(dir, ".skillgrid")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

func bootFileBody(t *testing.T, dir string) string {
	t.Helper()
	boot, _, err := writeBootFile(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(boot)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
