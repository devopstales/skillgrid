package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"skillgrid-cli/internal/mnemonic/setup"
)

func runSetup(args []string) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dryRun := fs.Bool("dry-run", false, "print planned changes without writing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: skillgrid setup <opencode|kilocode|cursor> [--dry-run]")
	}
	agent := fs.Arg(0)
	repoRoot := setup.FindRepoRoot("")
	if err := setup.RunSetup(agent, repoRoot, *dryRun); err != nil {
		return err
	}
	if !*dryRun {
		fmt.Fprintf(os.Stderr, "skillgrid setup %s: ok (config in ~/.config/%s/)\n", agent, setupAgentConfigDir(agent))
	}
	return nil
}

func setupAgentConfigDir(agent string) string {
	switch agent {
	case "opencode":
		return "opencode"
	case "kilocode", "kilo":
		return "kilo"
	case "cursor":
		return "cursor (mcp.json + rules/)"
	default:
		return "?"
	}
}
