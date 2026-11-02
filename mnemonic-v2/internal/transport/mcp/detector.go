package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// detectorStub is a fixed-detector implementation the tests use (no git
// shell-out).
type detectorStub struct {
	resolve func(cwd, explicit string) (string, error)
}

func (d *detectorStub) Resolve(cwd, explicit string) (string, error) {
	if d.resolve != nil {
		return d.resolve(cwd, explicit)
	}
	return "", fmt.Errorf("detectorStub: no resolver set")
}

// gitDetector is the production detector: it wraps core/project.Detector
// and shells out to git for the toplevel (the Engram hostbind pattern).
// The MCP stdio process cwd is the detector's cwd — the spec §4 chain
// (explicit → env → git → basename) runs at call time.
type gitDetector struct {
	envVar string
}

func (g *gitDetector) Resolve(cwd, explicit string) (string, error) {
	d := project.Detector{EnvVar: g.envVar}
	gitToplevel := project.GitToplevel(func(dir string) (string, error) {
		out, err := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(out)), nil
	})
	return d.Resolve(cwd, explicit, gitToplevel)
}

// NewGitDetector builds the production detector (the cmd layer passes the
// config's EnvVar; "" falls back to the config's default).
func NewGitDetector(envVar string) handler.Detector {
	if envVar == "" {
		envVar = "MNEMONIC_PROJECT"
	}
	return &gitDetector{envVar: envVar}
}

// ctx is the fixed test context (the stdio server is process-scoped in the
// tests; per-call cancellation is a transport concern the queue passes
// through untouched).
var ctx = context.Background()
