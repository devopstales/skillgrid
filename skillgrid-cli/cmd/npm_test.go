package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNPMInstallEnvSkipsHuskyPrepare(t *testing.T) {
	env := npmInstallEnv("/tmp/prefix")
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
	args := npmInstallArgs("/tmp/prefix", "/tmp/prefix/cache", []string{"@kilocode/cli", "skills"})
	got := strings.Join(args, " ")
	for _, want := range []string{
		"install -g",
		"--prefix /tmp/prefix",
		"--cache /tmp/prefix/cache",
		"--unsafe-perm",
		"--script-shell /bin/sh",
		"@kilocode/cli",
		"skills",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in npm args: %s", want, got)
		}
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
