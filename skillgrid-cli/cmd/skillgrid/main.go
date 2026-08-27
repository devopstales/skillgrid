package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/install"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "0.1.0-dev"

func main() {
	fs := flag.NewFlagSet("skillgrid", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		w := fs.Output()
		fmt.Fprintln(w, `skillgrid — install the AI-assisted development hub`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Usage:`)
		fmt.Fprintln(w, `  skillgrid [flags]                 install into this machine (default)`)
		fmt.Fprintln(w, `  skillgrid sync-repo PATH [flags]  sync a repo PATH into ~/.skillgrid and override ~/.agents`)
		fmt.Fprintln(w, `  skillgrid --version             print version`)
		fmt.Fprintln(w, `  skillgrid --help                show this help`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Flags:`)
		fs.PrintDefaults()
	}

	var (
		vVersion    bool
		vSkip       bool
		vSync       string
		vDry        bool
		vVerbose    bool
		vYes        bool
		vRepoURL    string
		vBranch     string
		vAgents     string
		vSkipTools  bool
		vSkipAgents bool
	)
	fs.BoolVar(&vVersion, "version", false, "print version and exit")
	fs.BoolVar(&vVersion, "V", false, "shorthand for --version")
	fs.BoolVar(&vSkip, "skip-clone", false, "skip the git clone step")
	fs.BoolVar(&vSkip, "s", false, "shorthand for --skip-clone")
	fs.StringVar(&vSync, "sync-repo", "", "sync this repo path into ~/.skillgrid and override ~/.agents")
	fs.StringVar(&vRepoURL, "repo-url", install.DefaultRepoURL, "git URL to clone")
	fs.StringVar(&vBranch, "branch", install.DefaultBranch, "branch to check out")
	fs.BoolVar(&vDry, "dry-run", false, "print planned changes without writing")
	fs.BoolVar(&vDry, "n", false, "shorthand for --dry-run")
	fs.BoolVar(&vVerbose, "verbose", false, "print detailed changes (MCP entries etc.)")
	fs.BoolVar(&vVerbose, "l", false, "shorthand for --verbose (long)")
	fs.BoolVar(&vYes, "yes", false, "skip interactive prompts (default agent selection)")
	fs.BoolVar(&vYes, "y", false, "shorthand for --yes")
	fs.StringVar(&vAgents, "agents", "", "comma-separated agent keys (opencode,kilo,cursor)")
	fs.BoolVar(&vSkipTools, "skip-tools", false, "skip global npm tool install")
	fs.BoolVar(&vSkipAgents, "skip-agents", false, "skip the ~/.agents override step")

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}

	if vVersion {
		fmt.Println("skillgrid", version)
		return
	}

	pos := fs.Args()
	if len(pos) > 0 && (pos[0] == "help" || pos[0] == "-h" || pos[0] == "--help") {
		fs.Usage()
		return
	}

	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		fmt.Fprintln(os.Stderr, "error: unable to resolve the home directory")
		os.Exit(1)
	}

	cfg := install.Config{
		Version:        version,
		DryRun:         vDry,
		Verbose:        vVerbose,
		Yes:            vYes,
		Agents:         parseAgents(vAgents),
		HomeDir:        home,
		RepoHome:       home + "/.skillgrid",
		RepoDir:        home + "/.skillgrid/repos/skillgrid",
		AgentsDir:      home + "/.agents",
		RepoURL:        vRepoURL,
		Branch:         vBranch,
		SkipClone:      vSkip,
		SkipTools:      vSkipTools,
		SkipAgentsCopy: vSkipAgents,
	}

	// --sync-repo PATH (flag form) or "sync-repo PATH" (subcommand form).
	syncPath := vSync
	if syncPath == "" && len(pos) >= 2 && pos[0] == "sync-repo" {
		syncPath = pos[1]
	}
	if len(pos) > 0 && pos[0] == "sync-repo" && syncPath == "" {
		fmt.Fprintln(os.Stderr, "error: sync-repo requires a PATH argument (see --help)")
		os.Exit(2)
	}
	if syncPath != "" {
		if err := cfg.SyncRepo(syncPath); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	if len(pos) > 0 && pos[0] != "install" {
		fmt.Fprintf(os.Stderr, "error: unknown command %q (see --help)\n", pos[0])
		os.Exit(2)
	}

	// Default action: install.
	if err := install.Run(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func parseAgents(s string) []string {
	out := []string{}
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		p := strings.ToLower(part)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
