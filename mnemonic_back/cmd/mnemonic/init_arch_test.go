package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadArchitectureTemplateFromSkillTree(t *testing.T) {
	dir := t.TempDir()
	rel := ".agents/skills/lifecycle/onboarding/templates/architecture.md"
	// The real scaffold contains the full 17-section structure.
	body := "# ARCHITECTURE.md\n\n**Project:** {project}\n\n## 1. Architecture Overview\n\n<detect>\n\n## 17. Known Gaps & TODOs\n"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	tpl, _, err := loadArchitectureTemplate(dir)
	if err != nil {
		t.Fatalf("loadArchitectureTemplate: %v", err)
	}
	if !strings.Contains(tpl, "## 1. Architecture Overview") {
		t.Errorf("template should contain section headings, got: %q", tpl)
	}
	if !strings.Contains(tpl, "{project}") {
		t.Errorf("template should keep the {project} placeholder, got: %q", tpl)
	}
}

func TestLoadArchitectureTemplateNotFound(t *testing.T) {
	dir := t.TempDir()
	// No skill tree and no repo root with the template -> error, not a crash.
	oldPath := architecturePath
	architecturePath = func(d string) string { return filepath.Join(d, "no-such-dir/architecture.md") }
	oldRoot := archRepoRoot
	archRepoRoot = func(string) string { return "" }
	defer func() { architecturePath, archRepoRoot = oldPath, oldRoot }()

	if _, _, err := loadArchitectureTemplate(dir); err == nil {
		t.Fatal("expected an error when the template is missing")
	}
}

func TestRenderArchitectureFillsPlaceholders(t *testing.T) {
	tpl := "# ARCHITECTURE.md\n\n**Project:** {project}\n**Version:** {version}\n**Last Updated:** {last_updated}\n\n## 1. Overview\n<detect>\n"
	out := renderArchitecture(archCfg{Project: "acme", Version: "1.2.3"}, tpl)

	if !strings.Contains(out, "**Project:** acme") {
		t.Errorf("project not filled, got: %q", out)
	}
	if !strings.Contains(out, "**Version:** 1.2.3") {
		t.Errorf("version not filled, got: %q", out)
	}
	if strings.Contains(out, "{last_updated}") {
		t.Errorf("last_updated placeholder left in: %q", out)
	}
	// The <detect> marker for unfilled architecture content must survive.
	if !strings.Contains(out, "<detect>") {
		t.Errorf("expected <detect> markers to remain for unfilled sections: %q", out)
	}
}

func TestRenderArchitectureDetectFallback(t *testing.T) {
	tpl := "**Project:** {project}\n**Version:** {version}\n"
	out := renderArchitecture(archCfg{}, tpl)
	if !strings.Contains(out, "<detect>") {
		t.Errorf("empty cfg should render <detect>, got: %q", out)
	}
	if strings.Contains(out, "{project}") || strings.Contains(out, "{version}") {
		t.Errorf("unfilled placeholders must not remain: %q", out)
	}
}

func TestWriteArchitectureFileCreatesScaffold(t *testing.T) {
	dir := t.TempDir()
	// Put the real scaffold template where the resolver finds it.
	rel := ".agents/skills/lifecycle/onboarding/templates/architecture.md"
	body := "# ARCHITECTURE.md\n\n**Project:** {project}\n\n## 1. Architecture Overview\n<detect>\n"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, rel), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	// Provide a project name so the header is filled, not <detect>.
	if err := os.MkdirAll(filepath.Join(dir, ".skillgrid"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".skillgrid", "config.yaml"),
		[]byte("project: acme\nversion: 9.9.9\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	path, err := writeArchitectureFile(dir)
	if err != nil {
		t.Fatalf("writeArchitectureFile: %v", err)
	}
	want := filepath.Join(dir, ".skillgrid", "ARCHITECTURE.md")
	if path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	data, _ := os.ReadFile(want)
	if !strings.Contains(string(data), "**Project:** acme") {
		t.Errorf("scaffold should fill project, got: %q", data)
	}
	if !strings.Contains(string(data), "<detect>") {
		t.Errorf("scaffold should keep <detect> for unfilled sections, got: %q", data)
	}

	// Idempotent: a second call keeps the existing file and returns "".
	marker := "# ARCHITECTURE.md\n\n**FILLED BY ONBOARDING**\n"
	if err := os.WriteFile(want, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := writeArchitectureFile(dir); err != nil || got != "" {
		t.Errorf("second call should be a no-op (got %q, err %v)", got, err)
	}
	if data, _ := os.ReadFile(want); !strings.Contains(string(data), "FILLED BY ONBOARDING") {
		t.Errorf("existing ARCHITECTURE.md must not be clobbered, got: %q", data)
	}
}

func TestRenderArchitectureDateIsUTC(t *testing.T) {
	tpl := "updated on {last_updated}"
	out := renderArchitecture(archCfg{}, tpl)
	// 2006-01-02 format -> 10 chars of digits/dashes.
	got := strings.TrimPrefix(out, "updated on ")
	if len(got) != 10 || got[4] != '-' || got[7] != '-' {
		t.Errorf("expected a YYYY-MM-DD date, got %q", got)
	}
}
