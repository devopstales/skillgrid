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
	SkipNetwork          bool
}

// RunInstallPhase orchestrates the tools installation process:
// 1. Ensure layout dirs
// 2. Try binary installs (engram, skills) - failures → warning, continue
// 3. Try EnsureManagedNPM; on failure → warning, skip npm packages
// 4. Else InstallNPMPackages with: gitnexus, backlog.md, @fission-ai/openspec, @upstash/context7-mcp, @playwright/mcp
// 5. Build present map from files that exist under DepsBinDir / NpmBinDir; always set http:deepwiki true
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
		installNPMPackages(p, &result)
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
	binaries := []struct {
		name   string
		repo   string
		binary string
	}{
		{"engram", "Gentleman-Programming/engram", "engram"},
		{"skills", "qntx/skill", "skills"},
	}

	resolver := opts.ReleaseAssetResolver
	if resolver == nil {
		resolver = GitHubReleaseAssetURL(opts.Downloader)
	}

	for _, bin := range binaries {
		assetURL, err := resolver(bin.repo, runtime.GOOS, runtime.GOARCH)
		if err != nil || assetURL == "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to resolve %s release: %v", bin.name, err))
			continue
		}

		// Download and extract
		get := opts.Downloader
		if get == nil {
			get = HTTPGet
		}

		data, err := get(assetURL)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to download %s: %v", bin.name, err))
			continue
		}

		// Try to extract from archive
		binaryData, err := ExtractBinaryFromArchive(data, bin.binary)
		if err != nil {
			// Maybe it's a raw binary
			binaryData = data
		}

		dest := filepath.Join(p.DepsBinDir, bin.binary)
		if err := EnsureFileExecutable(dest, binaryData); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("failed to write %s: %v", bin.name, err))
			continue
		}
	}
}

func installNPMPackages(p home.Paths, result *PhaseResult) {
	if err := EnsureManagedNPM(p); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("npm setup failed: %v", err))
		return
	}

	packages := []string{
		"gitnexus",
		"backlog.md",
		"@fission-ai/openspec",
		"@upstash/context7-mcp",
		"@playwright/mcp",
	}

	if err := InstallNPMPackages(p, packages); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("npm install failed: %v", err))
	}
}

func buildPresenceMap(p home.Paths, result *PhaseResult) {
	// Check binaries in DepsBinDir
	checkBinaries := []string{"engram", "skills"}
	for _, name := range checkBinaries {
		path := filepath.Join(p.DepsBinDir, name)
		if fileExists(path) {
			result.Present["binary:"+name] = true
		}
	}

	// Check npm packages in NpmBinDir
	checkNPMBinaries := []string{
		"gitnexus",
		"backlog",
	}
	for _, name := range checkNPMBinaries {
		// Use package name for presence key
		pkgName := name
		if name == "backlog" {
			pkgName = "backlog.md"
		}
		path := ManagedBin(p, name)
		if fileExists(path) {
			result.Present["npm:"+pkgName] = true
		}
	}

	// Check npm packages that are run via npx (no dedicated binary)
	checkNPMPackages := []string{
		"@upstash/context7-mcp",
		"@playwright/mcp",
		"@fission-ai/openspec",
	}
	for _, pkgName := range checkNPMPackages {
		// Check if package.json exists in node_modules
		pkgDir := filepath.Join(p.NpmDir, "node_modules", pkgName)
		pkgJSON := filepath.Join(pkgDir, "package.json")
		if fileExists(pkgJSON) {
			result.Present["npm:"+pkgName] = true
		}
	}

	// Always mark http:deepwiki as present
	result.Present["http:deepwiki"] = true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
