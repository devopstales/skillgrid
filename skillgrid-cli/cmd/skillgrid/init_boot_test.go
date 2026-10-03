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
	const want = "Backlog.md — reference `.agents/skills/planning/ticketing/references/backlogmd.md` for conventions."
	if !strings.Contains(body, want) {
		t.Fatalf("boot file missing tracker convention pointer:\n%s", body)
	}
	if strings.Contains(body, defaultTracker) {
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
			want: trackerGitHub,
		},
		{
			name: "gitlab",
			yaml: "ticketing:\n  enabled: true\n  type: glab\n",
			want: trackerGitLab,
		},
		{
			name: "jira",
			yaml: "ticketing:\n  enabled: true\n  type: jira\n",
			want: trackerJira,
		},
		{
			name: "disabled keeps local-only line",
			yaml: "ticketing:\n  enabled: false\n  type: backlogmd\n",
			want: defaultTracker,
		},
		{
			name: "unknown type",
			yaml: "ticketing:\n  enabled: true\n  type: linear\n",
			want: defaultTracker,
		},
		{
			name: "missing config",
			yaml: "",
			want: defaultTracker,
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
