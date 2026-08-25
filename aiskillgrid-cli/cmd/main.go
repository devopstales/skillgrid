package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func printUsage() {
	fmt.Fprintln(os.Stdout, "AI Skill Grid Installer\n\nUsage:\n  aiskillgrid <command> [flags]\n\nCommands:\n  install, in   Run full install\n  sync-repo     Sync repo contents without full install\n  help          Show this help\n\nFlags (install):\n  -skip-clone        skip git clone step\n  -sync-repo path    sync a repo path into ~/.aiskillgrid/repos/aiskillgrid\n  -dry-run           print planned changes without writing\n  -verbose           print detailed changes (MCP entries etc.)\n  -yes               skip interactive prompts (default agent selection)")
}

func wantHelp(argv []string) bool {
	for _, a := range argv {
		if a == "-h" || a == "--help" || a == "-help" || a == "help" {
			return true
		}
	}
	return false
}

func splitCommand(args []string) (string, []string) {
	for i, a := range args {
		if len(a) > 0 && a[0] != '-' {
			return a, args[i+1:]
		}
	}
	return "", args
}

func parseInstallArgs(rest []string) (bool, string, bool, bool, bool, error) {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	skip := fs.Bool("skip-clone", false, "skip git clone step")
	sync := fs.String("sync-repo", "", "sync repo path into ~/.aiskillgrid/repos/aiskillgrid")
	dry := fs.Bool("dry-run", false, "print planned changes without writing")
	verbose := fs.Bool("verbose", false, "print detailed changes")
	yes := fs.Bool("yes", false, "skip interactive prompts (default agent selection)")
	if err := fs.Parse(rest); err != nil {
		return false, "", false, false, false, err
	}
	return *skip, *sync, *dry, *verbose, *yes, nil
}

func Run() int {
	args := os.Args[1:]
	if wantHelp(args) {
		printUsage()
		return 0
	}

	cmd, rest := splitCommand(args)
	if cmd == "" {
		printUsage()
		return 1
	}

	switch cmd {
	case "install", "in":
		skip, sync, dry, verbose, yes, err := parseInstallArgs(rest)
		if err != nil {
			printUsage()
			return 1
		}
		runInstall(skip, sync, dry, verbose, yes)
	case "sync-repo":
		fs := flag.NewFlagSet("sync-repo", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		sync := fs.String("sync-repo", "", "path to sync into ~/.aiskillgrid/repos/aiskillgrid")
		if err := fs.Parse(rest); err != nil {
			printUsage()
			return 1
		}
		path := *sync
		if path == "" && fs.NArg() > 0 {
			path = fs.Arg(0)
		}
		if path == "" {
			printUsage()
			return 1
		}
		runSyncRepo(path)
	case "help":
		printUsage()
	default:
		printUsage()
		return 1
	}
	return 0
}

func main() {
	os.Exit(Run())
}
