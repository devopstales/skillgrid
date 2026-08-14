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
	gitnexus := fileExists(ManagedBin(p, "gitnexus"))
	backlog := fileExists(ManagedBin(p, "backlog"))
	openspec := fileExists(ManagedBin(p, "openspec"))
	if !openspec {
		openspec = fileExists(filepath.Join(p.NpmDir, "node_modules", "@fission-ai/openspec", "package.json"))
	}

	return []string{
		fmt.Sprintf("Managed npm: %s (node: %s)", p.NpmDir, nodeOK),
		fmt.Sprintf("Binaries: engram=%s skills=%s", yn(engram), yn(skills)),
		fmt.Sprintf("NPM bins: gitnexus=%s backlog=%s openspec=%s", yn(gitnexus), yn(backlog), yn(openspec)),
	}
}

func yn(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}
