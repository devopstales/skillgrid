package tools

import (
	"fmt"
	"path/filepath"

	"github.com/aiskillgrid/aiskillgrid/home"
)

// StatusLines reports managed npm layout and which tooling spine binaries are present.
func StatusLines(p home.Paths) []string {
	nodeOK := "no"
	if _, _, err := LookPathNode(); err == nil {
		nodeOK = "yes"
	}

	engram := fileExists(filepath.Join(p.DepsBinDir, "engram"))
	skills := fileExists(filepath.Join(p.DepsBinDir, "skills"))

	return []string{
		fmt.Sprintf("Managed npm: %s (node: %s)", p.NpmDir, nodeOK),
		fmt.Sprintf("Binaries: engram=%s skills=%s", yn(engram), yn(skills)),
		fmt.Sprintf("NPM bins: gitnexus=%s context7=%s playwright=%s",
			yn(npmPresent(p, "gitnexus")),
			yn(npmPresent(p, "@upstash/context7-mcp")),
			yn(npmPresent(p, "@playwright/mcp")),
		),
	}
}

func npmPresent(p home.Paths, pkg string) bool {
	for _, entry := range managedNpmPackages {
		if entry.pkg == pkg {
			return NpmPackageInstalled(p, entry.pkg, entry.bins...)
		}
	}
	return false
}

func yn(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}
