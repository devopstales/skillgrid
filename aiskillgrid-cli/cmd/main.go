package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	skipClone = flag.Bool("skip-clone", false, "skip git clone step")
	syncRepo  = flag.String("sync-repo", "", "sync extra paths into ~/.aiskillgrid/repos/aiskillgrid")
	dryRun    = flag.Bool("dry-run", false, "print planned changes without writing")
)

func Run() int {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "AI Skill Grid Installer\n\nUsage:\n  aiskillgrid-cli <command> [flags]\n\nCommands:\n  install, in   Run full install\n  sync-repo     Sync repo contents without full install\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return 1
	}

	switch args[0] {
	case "install", "in":
		runInstall(*skipClone, *syncRepo, *dryRun)
	case "sync-repo":
		runSyncRepo(*syncRepo)
	default:
		flag.Usage()
		return 1
	}
	return 0
}

func main() {
	os.Exit(Run())
}
