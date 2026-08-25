package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveGitHubShorthandToRegistryName(t *testing.T) {
	cases := map[string]string{
		"vercel-labs/skills":         "skills",
		"vercel-labs/agent-browser":  "agent-browser",
		"@kilocode/cli":              "@kilocode/cli",
		"@playwright/cli@latest":     "@playwright/cli@latest",
		"husky":                      "husky",
		"git+https://example.com/a.git": "git+https://example.com/a.git",
	}
	for in, want := range cases {
		if got := resolveNPMPackage(in); got != want {
			t.Fatalf("resolveNPMPackage(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestSplitNPMPackagesAfterResolve(t *testing.T) {
	var resolved []string
	for _, p := range []string{
		"@kilocode/cli",
		"husky",
		"vercel-labs/skills",
		"@playwright/cli@latest",
		"vercel-labs/agent-browser",
	} {
		resolved = append(resolved, resolveNPMPackage(p))
	}
	reg, git := splitNPMPackages(resolved)
	if strings.Join(reg, ",") != "@kilocode/cli,husky,skills,@playwright/cli@latest,agent-browser" {
		t.Fatalf("registry pkgs: %v", reg)
	}
	if len(git) != 0 {
		t.Fatalf("github shorthand should not use git install, got %v", git)
	}
}

func TestNPMInstallEnvSkipsHuskyPrepare(t *testing.T) {
	env := npmInstallEnv("/tmp/prefix", false)
	found := false
	for _, e := range env {
		if e == "HUSKY=0" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected HUSKY=0 in npm env, got %v", env)
	}
}

func TestNPMInstallArgsAllowScriptsAsInstallerUser(t *testing.T) {
	args := npmInstallArgs("/tmp/prefix", "/tmp/prefix/cache", []string{"@kilocode/cli", "skills"}, false)
	got := strings.Join(args, " ")
	for _, want := range []string{
		"install -g",
		"--prefix /tmp/prefix",
		"--cache /tmp/prefix/cache",
		"--script-shell /bin/sh",
		"--dangerously-allow-all-scripts",
		"@kilocode/cli",
		"skills",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in npm args: %s", want, got)
		}
	}
	if strings.Contains(got, "--unsafe-perm") {
		t.Fatalf("npm 11 dropped --unsafe-perm: %s", got)
	}
	if strings.Contains(got, "--ignore-scripts") {
		t.Fatalf("registry install must run postinstall: %s", got)
	}
}

func TestResetNPMInstallTreeKeepsCache(t *testing.T) {
	prefix := t.TempDir()
	cacheFile := filepath.Join(prefix, "cache", "keep")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cacheFile, []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{
		filepath.Join(prefix, "lib", "node_modules", "stale"),
		filepath.Join(prefix, "node_modules", "stale"),
		filepath.Join(prefix, "bin"),
	} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := resetNPMInstallTree(prefix); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheFile); err != nil {
		t.Fatalf("cache was removed: %v", err)
	}
	for _, p := range []string{
		filepath.Join(prefix, "lib", "node_modules"),
		filepath.Join(prefix, "node_modules"),
		filepath.Join(prefix, "bin"),
	} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("expected %s removed, stat err=%v", p, err)
		}
	}
}
