package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/aiskillgrid/aiskillgrid/home"
)

// NpmInstaller installs npm packages into the managed prefix. Injectable for tests.
type NpmInstaller func(p home.Paths, pkgs []string) error

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

// InstallNPMPackages installs packages global-style into the managed prefix so
// executables land in NpmBinDir instead of a project-local node_modules/.bin.
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
	args := []string{"install", "-g", "--prefix", p.NpmDir, "--cache", p.NpmCacheDir}
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

// ManagedBin returns the canonical path for a managed npm executable.
func ManagedBin(p home.Paths, name string) string {
	bin := filepath.Join(p.NpmBinDir, name)
	if runtime.GOOS == "windows" {
		return bin + ".cmd"
	}
	return bin
}

// NpmModulesDirs lists where `npm install -g --prefix` unpacks packages: Unix
// uses <prefix>/lib/node_modules, Windows uses <prefix>/node_modules.
func NpmModulesDirs(p home.Paths) []string {
	unix := filepath.Join(p.NpmDir, "lib", "node_modules")
	win := filepath.Join(p.NpmDir, "node_modules")
	if runtime.GOOS == "windows" {
		return []string{win, unix}
	}
	return []string{unix, win}
}

// npmBinDirs lists where npm may place executables for the managed prefix.
// Windows global installs write shims directly into the prefix root.
func npmBinDirs(p home.Paths) []string {
	if runtime.GOOS == "windows" {
		return []string{p.NpmBinDir, p.NpmDir}
	}
	return []string{p.NpmBinDir}
}

// ResolveManagedBin returns the path of the first installed executable matching
// any candidate name, or "" when none of them exist.
func ResolveManagedBin(p home.Paths, names ...string) string {
	for _, dir := range npmBinDirs(p) {
		for _, name := range names {
			candidates := []string{filepath.Join(dir, name)}
			if runtime.GOOS == "windows" {
				candidates = append(candidates, filepath.Join(dir, name+".cmd"))
			}
			for _, c := range candidates {
				if fileExists(c) {
					return c
				}
			}
		}
	}
	return ""
}

// ManagedBinOrDefault resolves an installed executable, falling back to the
// canonical path for the first name so callers can report or drop it.
func ManagedBinOrDefault(p home.Paths, names ...string) string {
	if got := ResolveManagedBin(p, names...); got != "" {
		return got
	}
	return ManagedBin(p, names[0])
}

// NpmPackageInstalled reports whether a package is available in the managed
// prefix, either as an executable shim or an unpacked package directory.
func NpmPackageInstalled(p home.Paths, pkg string, binNames ...string) bool {
	if ResolveManagedBin(p, binNames...) != "" {
		return true
	}
	for _, dir := range NpmModulesDirs(p) {
		if fileExists(filepath.Join(dir, pkg, "package.json")) {
			return true
		}
	}
	return false
}
