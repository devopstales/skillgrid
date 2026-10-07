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
		fmt.Fprintln(w, `  skillgrid <command> [flags]`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Commands:`)
		fmt.Fprintln(w, `  install, in   Run full install`)
		fmt.Fprintln(w, `  sync-repo     Sync git repo contents without full install`)
		fmt.Fprintln(w, `  help          Show this help`)
		fmt.Fprintln(w)
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
		vProvider   string
		vSkipProvider bool
		vBaseURL    string
		vAPIKey     string
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
	fs.StringVar(&vProvider, "provider", "local", "LLM/embedder provider: local (Ollama) or external (OpenAI-compatible)")
	fs.StringVar(&vProvider, "p", "local", "shorthand for --provider")
	fs.BoolVar(&vSkipProvider, "skip-provider", false, "skip the LLM/embedder provider setup step")
	fs.StringVar(&vBaseURL, "base-url", os.Getenv("SKILLGRID_LLM_BASE_URL"), "external provider base URL (or SKILLGRID_LLM_BASE_URL)")
	fs.StringVar(&vAPIKey, "api-key", os.Getenv("SKILLGRID_LLM_API_KEY"), "external provider API key (or SKILLGRID_LLM_API_KEY)")

	rest := os.Args[1:]
	rest0 := ""
	if len(rest) > 0 {
		rest0 = rest[0]
	}

	if rest0 == "help" || rest0 == "-h" || rest0 == "--help" || rest0 == "" {
		fs.Usage()
		return
	}

	if rest0 == "sync-repo" {
		syncPath := ""
		if len(rest) >= 2 {
			syncPath = rest[1]
		}
		if syncPath == "" {
			fmt.Fprintln(os.Stderr, "error: sync-repo requires a PATH argument")
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
		}
		return
	}

	if err := fs.Parse(reorderArgs(rest)); err != nil {
		os.Exit(2)
	}

	if vVersion {
		fmt.Println("skillgrid", version)
		return
	}

	pos := fs.Args()

	syncPath := vSync
	if syncPath != "" {
		home, _ := os.UserHomeDir()
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
		if rest0 != "install" && rest0 != "in" {
			return
		}
	}

	if len(pos) > 0 {
		switch pos[0] {
		case "install", "in":
		default:
			fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", pos[0])
			fs.Usage()
			os.Exit(2)
		}
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
		Provider:       vProvider,
		SkipProvider:   vSkipProvider,
		LLMBaseURL:     vBaseURL,
		LLMApiKey:      vAPIKey,
	}

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

func reorderArgs(args []string) []string {
	boolFlags := map[string]bool{
		"version": true, "v": true,
		"skip-clone": true, "s": true,
		"dry-run": true,
		"verbose": true, "vv": true,
		"yes": true, "y": true,
		"skip-tools":    true,
		"skip-agents":   true,
		"skip-provider": true,
	}
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
