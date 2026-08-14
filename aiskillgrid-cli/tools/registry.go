package tools

import (
	"fmt"

	"github.com/aiskillgrid/aiskillgrid/home"
)

// Tool describes a future installable tool pack.
type Tool struct {
	Name string
	// InstallDeps will be filled when tools are selected later.
}

// Registry is empty in v1.
type Registry struct {
	Tools []Tool
}

func Default() Registry {
	return Registry{Tools: nil}
}

// InstallDeps now delegates to RunInstallPhase for actual installation.
// This is retained for backward compatibility with v1 callers.
func (r Registry) InstallDeps(p home.Paths, packRoot string) error {
	if len(r.Tools) > 0 {
		return fmt.Errorf("custom tool selection not yet supported")
	}
	// Run install phase with default options
	_, err := RunInstallPhase(p, packRoot, PhaseOptions{})
	return err
}
