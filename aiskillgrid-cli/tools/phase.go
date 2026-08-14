package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/aiskillgrid/aiskillgrid/home"
)

// PhaseResult contains the results of the install phase.
type PhaseResult struct {
	Servers  map[string]any
	Warnings []string
	Present  map[string]bool
}

// PhaseOptions provides injectable dependencies for testing.
type PhaseOptions struct {
	Downloader           Downloader
	ReleaseAssetResolver ReleaseAssetResolver
	NpmInstaller         NpmInstaller
	SkipNetwork          bool
}

// releaseBinaries are installed from GitHub releases into DepsBinDir.
var releaseBinaries = []struct {
	name   string
	repo   string
	binary string
}{
	{"engram", "Gentleman-Programming/engram", "engram"},
	{"skills", "qntx/skill", "skills"},
}

// managedNpmPackages are installed into the managed npm prefix. bins lists the
// executable names the package may publish, newest naming first.
var managedNpmPackages = []struct {
	pkg  string
	bins []string
}{
	{"gitnexus", []string{"gitnexus"}},
	{"backlog.md", []string{"backlog"}},
	{"@fission-ai/openspec", []string{"openspec"}},
	{"@upstash/context7-mcp", []string{"context7-mcp"}},
	{"@playwright/mcp", []string{"playwright-mcp", "mcp-server-playwright"}},
}

// RunInstallPhase orchestrates the tools installation process:
// 1. Ensure layout dirs
// 2. Try binary installs (engram, skills) - failures → warning, continue
// 3. Try EnsureManagedNPM; on failure → warning, skip npm packages
// 4. Else InstallNPMPackages with: gitnexus, backlog.md, @fission-ai/openspec, @upstash/context7-mcp, @playwright/mcp
// 5. Build present map from files that exist under DepsBinDir / managed npm prefix; always set http:deepwiki true
// 6. ResolveMCPServers
// 7. If playwright MCP present, append warning: browsers may need install later
func RunInstallPhase(p home.Paths, packRoot string, opts PhaseOptions) (PhaseResult, error) {
	result := PhaseResult{
		Warnings: []string{},
		Present:  make(map[string]bool),
	}

	// Step 1: Ensure layout dirs (caller may already have, but safe to repeat)
	if err := home.EnsureLayout(p); err != nil {
		return result, fmt.Errorf("ensure layout: %w", err)
	}

	// Step 2: Try binary installs (best-effort, warn on failure)
	if !opts.SkipNetwork {
		installBinaries(p, &result, opts)
	}

	// Step 3-4: Try NPM setup and packages
	if !opts.SkipNetwork {
		installNPMPackages(p, &result, opts)
	}

	// Step 5: Build presence map
	buildPresenceMap(p, &result)

	// Step 6: Resolve MCP servers
	serversPath := filepath.Join(packRoot, "packs", "mcp", "servers.json")
	servers, warnings, err := ResolveMCPServers(serversPath, p, result.Present)
	if err != nil {
		return result, fmt.Errorf("resolve MCP servers: %w", err)
	}
	result.Servers = servers
	result.Warnings = append(result.Warnings, warnings...)

	// Step 7: Playwright warning
	if result.Present["npm:@playwright/mcp"] {
		result.Warnings = append(result.Warnings, "playwright browsers may need install later")
	}

	return result, nil
}

func installBinaries(p home.Paths, result *PhaseResult, opts PhaseOptions) {
	resolver := opts.ReleaseAssetResolver
	if resolver == nil {
		resolver = GitHubReleaseAssetURL(opts.Downloader)
	}

	for _, bin := range releaseBinaries {
		// Resolving the release costs a GitHub API call, so check first.
		if BinaryInstalled(filepath.Join(p.DepsBinDir, bin.binary)) {
			continue
		}

		assetURL, err := resolver(bin.repo, runtime.GOOS, runtime.GOARCH)
		if err != nil || assetURL == "" {
			result.Warnings = append(result.Warnings, resolveFailureWarning(bin.name, err))
			continue
		}

		if err := EnsureReleaseBinary(p.DepsBinDir, bin.binary, assetURL, opts.Downloader); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to install %s: %v", bin.name, err))
		}
	}
}

func resolveFailureWarning(name string, err error) string {
	if err != nil {
		return fmt.Sprintf("failed to resolve %s release: %v", name, err)
	}
	return fmt.Sprintf("failed to resolve %s release: no matching asset", name)
}

func installNPMPackages(p home.Paths, result *PhaseResult, opts PhaseOptions) {
	install := opts.NpmInstaller
	if install == nil {
		install = InstallNPMPackages
	}

	if err := EnsureManagedNPM(p); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("npm setup failed: %v", err))
		return
	}

	packages := make([]string, 0, len(managedNpmPackages))
	for _, pkg := range managedNpmPackages {
		packages = append(packages, pkg.pkg)
	}

	if err := install(p, packages); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("npm install failed: %v", err))
	}
}

func buildPresenceMap(p home.Paths, result *PhaseResult) {
	for _, bin := range releaseBinaries {
		if fileExists(filepath.Join(p.DepsBinDir, bin.binary)) {
			result.Present["binary:"+bin.binary] = true
		}
	}

	for _, pkg := range managedNpmPackages {
		if NpmPackageInstalled(p, pkg.pkg, pkg.bins...) {
			result.Present["npm:"+pkg.pkg] = true
		}
	}

	// DeepWiki is a remote HTTP server, so it needs nothing installed locally.
	result.Present["http:deepwiki"] = true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
