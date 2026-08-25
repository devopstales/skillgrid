package main

import (
	"skillgrid-cli/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func resolveNPMPackage(pkg string) string {
	if strings.HasPrefix(pkg, "github:") || strings.HasPrefix(pkg, "git+") || strings.Contains(pkg, ".git") {
		return pkg
	}
	if strings.HasPrefix(pkg, "@") {
		return pkg
	}
	slash := strings.Index(pkg, "/")
	if slash > 0 && !strings.Contains(pkg[:slash], ".") {
		return pkg[slash+1:]
	}
	return pkg
}

func isGitNPMPackage(pkg string) bool {
	if strings.HasPrefix(pkg, "github:") || strings.HasPrefix(pkg, "git+") || strings.Contains(pkg, ".git") {
		return true
	}
	if strings.HasPrefix(pkg, "@") {
		return false
	}
	slash := strings.Index(pkg, "/")
	return slash > 0 && !strings.Contains(pkg[:slash], ".")
}

func splitNPMPackages(pkgs []string) (registry, git []string) {
	for _, p := range pkgs {
		if isGitNPMPackage(p) {
			git = append(git, p)
		} else {
			registry = append(registry, p)
		}
	}
	return registry, git
}

func npmInstallArgs(prefix, cache string, pkgs []string, ignoreScripts bool) []string {
	args := []string{
		"install", "-g",
		"--prefix", prefix,
		"--cache", cache,
		"--script-shell", "/bin/sh",
	}
	if ignoreScripts {
		args = append(args, "--ignore-scripts")
	} else {
		args = append(args, "--dangerously-allow-all-scripts")
	}
	return append(args, pkgs...)
}

func npmInstallEnv(prefix string, ignoreScripts bool) []string {
	npmBin := filepath.Join(prefix, "bin")
	env := append(os.Environ(),
		"HUSKY=0",
		"PATH="+npmBin+":"+os.Getenv("PATH"),
	)
	if ignoreScripts {
		env = append(env, "npm_config_ignore_scripts=true")
	}
	return env
}

func resetNPMInstallTree(prefix string) error {
	for _, p := range []string{
		filepath.Join(prefix, "lib", "node_modules"),
		filepath.Join(prefix, "node_modules"),
		filepath.Join(prefix, "bin"),
	} {
		if err := os.RemoveAll(p); err != nil {
			return err
		}
	}
	return nil
}

func runNPMInstall(prefix, cache string, pkgs []string, ignoreScripts bool) error {
	if len(pkgs) == 0 {
		return nil
	}
	cmd := exec.Command("npm", npmInstallArgs(prefix, cache, pkgs, ignoreScripts)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = npmInstallEnv(prefix, ignoreScripts)
	return cmd.Run()
}

func installNPM(baseDir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	tools, err := config.LoadToolsYAML(filepath.Join(baseDir, "config.d", "tools.yaml"))
	if err != nil {
		return nil
	}
	var pkgs []string
	for _, p := range append(append([]string{}, tools.Agents...), tools.Tools...) {
		pkgs = append(pkgs, resolveNPMPackage(p))
	}
	prefix := mustExpandHomePath("~/.skillgrid/npm")
	cache := filepath.Join(prefix, "cache")
	if err := os.MkdirAll(prefix, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(cache, 0755); err != nil {
		return err
	}
	if err := resetNPMInstallTree(prefix); err != nil {
		return err
	}
	registry, git := splitNPMPackages(pkgs)
	if err := runNPMInstall(prefix, cache, registry, false); err != nil {
		return err
	}
	return runNPMInstall(prefix, cache, git, true)
}
