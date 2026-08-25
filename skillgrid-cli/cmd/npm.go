package main

import (
	"skillgrid-cli/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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
	prefix := filepath.Join(baseDir, "npm")
	cache := filepath.Join(prefix, "cache")
	args := append([]string{"install"}, pkgs...)
	args = append(args, "--prefix", prefix, "--cache", cache)
	return exec.Command("npm", args...).Run()
}

var _ = os.Stdout
