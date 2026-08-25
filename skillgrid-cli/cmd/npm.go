package main

import (
	"skillgrid-cli/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func npmInstallArgs(prefix, cache string, pkgs []string) []string {
	args := []string{
		"install", "-g",
		"--prefix", prefix,
		"--cache", cache,
		"--unsafe-perm",
		"--script-shell", "/bin/sh",
	}
	return append(args, pkgs...)
}

func npmInstallEnv(prefix string) []string {
	npmBin := filepath.Join(prefix, "bin")
	return append(os.Environ(),
		"HUSKY=0",
		"PATH="+npmBin+":"+os.Getenv("PATH"),
	)
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

func installNPM(baseDir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	tools, err := config.LoadToolsYAML(filepath.Join(baseDir, "config.d", "tools.yaml"))
	if err != nil {
		return nil
	}
	pkgs := append([]string{}, tools.Agents...)
	pkgs = append(pkgs, tools.Tools...)
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
	cmd := exec.Command("npm", npmInstallArgs(prefix, cache, pkgs)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = npmInstallEnv(prefix)
	return cmd.Run()
}
