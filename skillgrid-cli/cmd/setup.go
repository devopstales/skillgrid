package main

import (
	"flag"
	"fmt"
	"io"

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
	return setup.RunSetup(agent, repoRoot, *dryRun)
}
