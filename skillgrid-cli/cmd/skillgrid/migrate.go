package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/tiered"
)

// runMigrate handles `skillgrid migrate --tier`.
func runMigrate(version string, args []string) {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		tier    bool
		dir     string
		project string
		root    string
	)
	fs.BoolVar(&tier, "tier", false, "backfill L0/L1 sidecars from existing L2 markdown")
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory (default ~/.skillgrid/mnemonic)")
	fs.StringVar(&project, "project", "", "project id (required with --tier)")
	fs.StringVar(&root, "root", "", "content root to walk for L2 .md files (required with --tier)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid migrate --tier --project ID --root DIR [--dir DATA_DIR]")
		fmt.Fprintln(fs.Output(), "  Backfills .abstract / .overview sidecars without rewriting L2 bytes.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	_ = version
	if !tier {
		fmt.Fprintln(os.Stderr, "error: migrate currently supports only --tier")
		fs.Usage()
		os.Exit(2)
	}
	if project == "" || root == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --root are required with --tier")
		fs.Usage()
		os.Exit(2)
	}
	dataDir := dir
	if dataDir == "" {
		d, err := service.DefaultDataDir()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		dataDir = d
	}
	st, err := store.Open(dataDir, project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: open store:", err)
		os.Exit(1)
	}
	defer st.Close()

	ts := &tiered.Store{
		DB:         st.DB,
		Summarizer: tiered.HeuristicSummarizer{},
		Logf: func(format string, args ...any) {
			fmt.Fprintf(os.Stderr, "warn: "+format+"\n", args...)
		},
	}
	n, err := tiered.MigrateTier(context.Background(), ts, project, root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Printf("migrate --tier: processed %d file(s)\n", n)
}
