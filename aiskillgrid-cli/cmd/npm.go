package main

import (
	"aiskillgrid-cli/internal/config"
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
	args := append([]string{"install"}, pkgs...)
	args = append(args, "--prefix", baseDir)
	return exec.Command("npm", args...).Run()
}

var _ = os.Stdout
