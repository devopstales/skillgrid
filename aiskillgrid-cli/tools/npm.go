package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/aiskillgrid/aiskillgrid/home"
)

func LookPathNode() (nodePath, npmPath string, err error) {
	nodePath, err = exec.LookPath("node")
	if err != nil {
		return "", "", fmt.Errorf("node not found on PATH: %w", err)
	}
	npmPath, err = exec.LookPath("npm")
	if err != nil {
		return "", "", fmt.Errorf("npm not found on PATH: %w", err)
	}
	return nodePath, npmPath, nil
}

func EnsureManagedNPM(p home.Paths) error {
	if err := os.MkdirAll(p.NpmBinDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(p.NpmCacheDir, 0o755); err != nil {
		return err
	}
	_, _, err := LookPathNode()
	return err
}

func InstallNPMPackages(p home.Paths, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	if err := EnsureManagedNPM(p); err != nil {
		return err
	}
	_, npmPath, err := LookPathNode()
	if err != nil {
		return err
	}
	args := []string{"install", "--prefix", p.NpmDir, "--cache", p.NpmCacheDir}
	args = append(args, pkgs...)
	cmd := exec.Command(npmPath, args...)
	cmd.Env = append(os.Environ(),
		"npm_config_prefix="+p.NpmDir,
		"npm_config_cache="+p.NpmCacheDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install: %w\n%s", err, out)
	}
	return nil
}

func ManagedBin(p home.Paths, name string) string {
	bin := filepath.Join(p.NpmBinDir, name)
	if runtime.GOOS == "windows" {
		return bin + ".cmd"
	}
	return bin
}
