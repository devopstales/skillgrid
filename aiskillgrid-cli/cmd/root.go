package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/aiskillgrid/aiskillgrid/home"
	"github.com/aiskillgrid/aiskillgrid/install"
	"github.com/aiskillgrid/aiskillgrid/sync"
	"github.com/aiskillgrid/aiskillgrid/tools"
	"github.com/spf13/cobra"
)

var (
	Version    = "0.1.0"
	Commit     = "dev"
	scopeFlag  string
	agentsFlag string
	yesFlag    bool
	skipSync   bool
	repoURL    string
)

func Execute() error {
	root := &cobra.Command{
		Use:   "aiskillgrid",
		Short: "Install Skillgrid skills and MCP wiring across AI agents",
	}
	root.AddCommand(versionCmd(), syncCmd(), installCmd(), statusCmd())
	return root.Execute()
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aiskillgrid %s (%s)\n", Version, Commit)
		},
	}
}

func syncCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "sync",
		Short: "Clone or pull the Skillgrid GitHub repo into the managed home tools/ directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := home.Root()
			if err != nil {
				return err
			}
			paths := home.Resolve(root)
			if err := home.EnsureLayout(paths); err != nil {
				return err
			}
			cfg, err := home.LoadConfig(paths.ConfigFile)
			if err != nil {
				return err
			}
			url := repoURL
			if url == "" {
				url = cfg.RepoURL
			}
			res, err := sync.Sync(paths.ToolsDir, url)
			if err != nil {
				return err
			}
			fmt.Printf("Synced %s @ %s\n", res.Path, res.Rev)
			_ = home.AppendLog(paths.LogsDir, fmt.Sprintf("sync ok rev=%s", res.Rev))
			return nil
		},
	}
	c.Flags().StringVar(&repoURL, "repo", "", "Git repository URL override")
	return c
}

func installCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "install",
		Short: "Interactively install skills and MCP configs into selected agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := install.Options{
				Yes:      yesFlag,
				SkipSync: skipSync,
				RepoURL:  repoURL,
			}
			if scopeFlag != "" {
				opts.Scope = home.Scope(scopeFlag)
			}
			if agentsFlag != "" {
				for _, a := range strings.Split(agentsFlag, ",") {
					a = strings.TrimSpace(a)
					if a != "" {
						opts.Agents = append(opts.Agents, a)
					}
				}
			}
			if yesFlag && opts.Scope == "" {
				opts.Scope = home.ScopeGlobal
			}
			return install.Run(opts)
		},
	}
	c.Flags().StringVar(&scopeFlag, "scope", "", "global or project")
	c.Flags().StringVar(&agentsFlag, "agents", "", "comma-separated v1 clients: kilo,opencode,cursor,copilot")
	c.Flags().BoolVar(&yesFlag, "yes", false, "non-interactive; requires --agents (and defaults scope to global)")
	c.Flags().BoolVar(&skipSync, "skip-sync", false, "do not git sync before install")
	c.Flags().StringVar(&repoURL, "repo", "", "Git repository URL override")
	return c
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show managed home, sync revision, and last install state",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := home.Root()
			if err != nil {
				return err
			}
			paths := home.Resolve(root)
			fmt.Printf("Home: %s\n", paths.Root)
			cfg, err := home.LoadConfig(paths.ConfigFile)
			if err != nil {
				fmt.Printf("Config: (error: %v)\n", err)
			} else {
				fmt.Printf("Repo URL: %s\n", cfg.RepoURL)
			}
			if rev, err := sync.RevParse(paths.ToolsDir); err != nil {
				fmt.Printf("Tools sync: not synced (%v)\n", err)
			} else {
				fmt.Printf("Tools rev: %s\n", rev)
			}
			st, err := home.LoadState(paths.StateFile)
			if err != nil {
				return err
			}
			if st.UpdatedAt.IsZero() {
				fmt.Println("Last install: none")
			} else {
				fmt.Printf("Last install: %s\n", st.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))
				fmt.Printf("  Scope: %s\n", st.Scope)
				if st.ProjectDir != "" {
					fmt.Printf("  Project: %s\n", st.ProjectDir)
				}
				fmt.Printf("  Agents: %s\n", strings.Join(st.Agents, ", "))
				if st.RepoRev != "" {
					fmt.Printf("  Install rev: %s\n", st.RepoRev)
				}
				for agent, pathsWritten := range st.WrittenPaths {
					fmt.Printf("  %s: %d paths\n", agent, len(pathsWritten))
				}
			}
			for _, line := range tools.StatusLines(paths) {
				fmt.Println(line)
			}
			return nil
		},
	}
}

func Main() {
	if err := Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
