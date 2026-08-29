package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/install"
)

// flagShorthands maps long flag names to their shorthand aliases.
var flagShorthands = map[string]string{
	"version":    "v",
	"skip-clone": "s",
	"sync-repo":  "n",
	"verbose":    "vv",
	"yes":        "y",
}

// boolFlags lists every boolean flag name (long + shorthand).
var boolFlags = map[string]bool{
	"version": true, "v": true,
	"skip-clone": true, "s": true,
	"dry-run": true,
	"verbose": true, "vv": true,
	"yes": true, "y": true,
	"skip-tools":  true,
	"skip-agents": true,
}

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
		fmt.Fprintln(w, `  skillgrid <command> [flags]`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Commands:`)
		fmt.Fprintln(w, `  install, in   Run full install`)
		fmt.Fprintln(w, `  sync-repo     Sync git repo contents without full install`)
		fmt.Fprintln(w, `  mcp           Run the Mnemonic MCP stdio server`)
		fmt.Fprintln(w, `  serve         Run the Mnemonic HTTP API (default :7438)`)
		fmt.Fprintln(w, `  index         Incremental code indexing`)
		fmt.Fprintln(w, `  setup         Install agent plugins (opencode|kilocode|cursor)`)
		fmt.Fprintln(w, `  help          Show this help`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Flags (install):`)
		printFlags(fs, w)
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
	fs.BoolVar(&vVersion, "v", false, "shorthand for --version")
	fs.BoolVar(&vSkip, "skip-clone", false, "skip the git clone step")
	fs.BoolVar(&vSkip, "s", false, "shorthand for --skip-clone")
	fs.StringVar(&vSync, "sync-repo", "", "sync a repo path into ~/.skillgrid/repos/skillgrid")
	fs.StringVar(&vSync, "n", "", "shorthand for --sync-repo")
	fs.StringVar(&vRepoURL, "repo-url", install.DefaultRepoURL, "git URL to clone")
	fs.StringVar(&vBranch, "branch", install.DefaultBranch, "branch to check out")
	fs.BoolVar(&vDry, "dry-run", false, "print planned changes without writing")
	fs.BoolVar(&vVerbose, "verbose", false, "print detailed changes (MCP entries etc.)")
	fs.BoolVar(&vVerbose, "vv", false, "shorthand for --verbose")
	fs.BoolVar(&vYes, "yes", false, "skip interactive prompts (default agent selection)")
	fs.BoolVar(&vYes, "y", false, "shorthand for --yes")
	fs.StringVar(&vAgents, "agents", "", "comma-separated agent keys (opencode,kilo,cursor)")
	fs.BoolVar(&vSkipTools, "skip-tools", false, "skip global npm tool install")
	fs.BoolVar(&vSkipAgents, "skip-agents", false, "skip the ~/.agents override step")

	args := os.Args[1:]

	// Subcommands own their flags; dispatch on the first bare-arg token
	// before the install flag set runs (subcommand flags like --dir/--port
	// are not part of the top-level flag set).
	rest := args
	for i, a := range args {
		if a == "-" || strings.HasPrefix(a, "--") {
			continue
		}
		if isBoolFlagToken(a) {
			continue
		}
		rest = args[i:]
		break
	}

	rest0 := ""
	if len(rest) > 0 {
		rest0 = rest[0]
	}

	switch rest0 {
	case "mcp":
		runMCP(version, rest[1:])
		return
	case "serve":
		runServe(version, rest[1:])
		return
	case "index":
		runIndex(version, rest[1:])
		return
	case "setup":
		runSetup(version, rest[1:])
		return
	case "sync-repo":
		syncPath := ""
		if len(rest) >= 2 {
			syncPath = rest[1]
		}
		if syncPath == "" {
			for _, a := range rest[1:] {
				if strings.HasPrefix(a, "--sync-repo=") {
					syncPath = strings.TrimPrefix(a, "--sync-repo=")
				}
			}
		}
		if syncPath == "" {
			fmt.Fprintln(os.Stderr, "error: sync-repo requires a PATH argument (see --help)")
			os.Exit(2)
		}
		if h, err := os.UserHomeDir(); err == nil && h != "" {
			cfg := install.Config{
				Version:   version,
				HomeDir:   h,
				RepoHome:  h + "/.skillgrid",
				RepoDir:   h + "/.skillgrid/repos/skillgrid",
				AgentsDir: h + "/.agents",
			}
			if err := cfg.SyncRepo(syncPath); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		} else {
			if err := syncRepoOnly(version, syncPath); err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
		return
	case "help", "-h", "--help":
		fallthrough
	case "":
		fs.Usage()
		return
	case "install", "in":
		// continue to install flag parsing below
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q (see --help)\n", rest0)
		os.Exit(2)
	}

	if err := fs.Parse(reorderArgs(args)); err != nil {
		os.Exit(2)
	}

	if vVersion {
		fmt.Println("skillgrid", version)
		return
	}

	pos := fs.Args()

	// --sync-repo PATH (flag form) — subcommand form handled above.
	syncPath := vSync
	if syncPath != "" {
		home := homeDir()
		cfg := install.Config{
			Version:   version,
			HomeDir:   home,
			RepoHome:  home + "/.skillgrid",
			RepoDir:   home + "/.skillgrid/repos/skillgrid",
			AgentsDir: home + "/.agents",
		}
		if err := cfg.SyncRepo(syncPath); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		return
	}

	// Only run install when explicitly invoked with the "install" subcommand.
	if len(pos) == 0 || (pos[0] != "install" && pos[0] != "in") {
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

	if err := install.Run(&cfg); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func homeDir() string {
	h, _ := os.UserHomeDir()
	return h
}

func syncRepoOnly(version, syncPath string) error {
	h, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	cfg := install.Config{
		Version:   version,
		HomeDir:   h,
		RepoHome:  h + "/.skillgrid",
		RepoDir:   h + "/.skillgrid/repos/skillgrid",
		AgentsDir: h + "/.agents",
	}
	return cfg.SyncRepo(syncPath)
}

// isBoolFlagToken reports whether a "-x/--x" token is a known boolean flag
// (does not consume the following token as its value).
func isBoolFlagToken(a string) bool {
	name := a
	if idx := strings.Index(name, "="); idx >= 0 {
		return false
	}
	name = strings.TrimLeft(name, "-")
	return boolFlags[name]
}

// reorderArgs moves flag arguments in front of positional arguments so that
// `skillgrid install --skip-clone` parses like `skillgrid --skip-clone install`.
func reorderArgs(args []string) []string {
	var flags, positional []string
	seenSep := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" && !seenSep {
			seenSep = true
			positional = append(positional, a)
			continue
		}
		if seenSep || !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			continue
		}
		if idx := strings.Index(a, "="); idx >= 0 {
			flags = append(flags, a)
			continue
		}
		name := strings.TrimLeft(a, "-")
		isBool := boolFlags[name]
		flags = append(flags, a)
		if !isBool && i+1 < len(args) {
			flags = append(flags, args[i+1])
			i++
		}
	}
	return append(flags, positional...)
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

// printFlags prints all flags with double-dash long form, sorted alphabetically.
func printFlags(fs *flag.FlagSet, w io.Writer) {
	type entry struct {
		name      string
		usage     string
		defValue  string
		shorthand string
	}
	var entries []entry
	fs.VisitAll(func(f *flag.Flag) {
		// Skip shorthand-only entries (length 1 or known shorthands)
		if len(f.Name) == 1 || f.Name == "vv" {
			return
		}
		sh, _ := flagShorthands[f.Name]
		entries = append(entries, entry{
			name:      f.Name,
			usage:     f.Usage,
			defValue:  f.DefValue,
			shorthand: sh,
		})
	})

	sort.Slice(entries, func(i, j int) bool { return entries[i].name < entries[j].name })

	for _, e := range entries {
		prefix := fmt.Sprintf("  --%s", e.name)
		if e.shorthand != "" {
			prefix = fmt.Sprintf("  -%s, --%s", e.shorthand, e.name)
		}
		if e.defValue != "" && e.defValue != "false" && e.defValue != "0" {
			fmt.Fprintf(w, "%-24s %s\n", prefix, e.usage+" (default "+fmt.Sprintf("%q", e.defValue)+")")
		} else {
			fmt.Fprintf(w, "%-24s %s\n", prefix, e.usage)
		}
	}
}
