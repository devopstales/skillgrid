package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected"
)

// runSearchRename implements `skillgrid search rename OLD NEW` (CLI parity
// for code_rename). dry_run is the default (no writes); --apply writes only
// the planned files (no commit/push).
func runSearchRename(version string, args []string) {
	fs := flag.NewFlagSet("search rename", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut bool
		apply   bool
		file    string
		uid     string
		kind    string
	)
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.BoolVar(&apply, "apply", false, "write the planned edits (only listed files; no commit/push)")
	fs.StringVar(&file, "file", "", "narrow disambiguation to a file path")
	fs.StringVar(&uid, "uid", "", "narrow disambiguation to a symbol UID")
	fs.StringVar(&kind, "kind", "", "narrow disambiguation to a symbol kind")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search rename OLD NEW [--file F] [--uid U] [--kind K] [--apply] [--json]")
		fs.PrintDefaults()
	}
	// Strip the leading "search" + "rename" tokens (main.go dispatches the
	// whole rest).
	fsArgs := args
	if len(fsArgs) >= 2 && fsArgs[0] == "search" && fsArgs[1] == "rename" {
		fsArgs = fsArgs[2:]
	} else if len(fsArgs) >= 1 && fsArgs[0] == "rename" {
		fsArgs = fsArgs[1:]
	}
	if err := fs.Parse(fsArgs); err != nil {
		os.Exit(2)
	}
	if fs.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "usage: skillgrid search rename OLD NEW [--file F] [--uid U] [--kind K] [--apply] [--json]")
		os.Exit(2)
	}
	old, new := fs.Arg(0), fs.Arg(1)
	svc, projectID, err := openGraphService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	_ = version
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()
	plan, err := affected.Rename(ctx, h.Store().DB, affected.RenameOptions{
		Old: old, New: new, File: file, UID: uid, Kind: kind,
		DryRun: !apply, Apply: apply,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(plan, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	printRename(plan)
}

// printRename renders a code_rename plan (human form).
func printRename(plan affected.RenamePlan) {
	if plan.NotFound {
		fmt.Fprintf(os.Stderr, "not found: no symbol named %q\n", plan.Old)
		return
	}
	if plan.Ambiguous {
		fmt.Printf("ambiguous target %q: %d candidates (narrow with --file/--uid/--kind)\n", plan.Old, len(plan.Candidates))
		for i, c := range plan.Candidates {
			fmt.Printf("  %d. %s (%s) [uid %s]\n", i+1, c.Path, c.Name, c.UID)
		}
		return
	}
	if plan.Target != nil {
		fmt.Printf("%s (%s) -> %s\n", plan.Old, plan.Target["path"], plan.New)
	}
	mode := "dry-run (no writes; --apply to write)"
	if plan.Applied {
		mode = "applied"
	}
	fmt.Printf("%s: %d files, %d edits (%d graph, %d text-search)\n", mode, plan.FilesAffected, plan.TotalEdits, len(plan.GraphEdits), len(plan.TextSearchEdits))
	if len(plan.GraphEdits) > 0 {
		fmt.Println("graph edits (high-confidence):")
		for _, e := range plan.GraphEdits {
			fmt.Printf("  [high] %s %s -> %s (%s)\n", e.Path, e.Old, e.New, e.Source)
		}
	}
	if len(plan.TextSearchEdits) > 0 {
		fmt.Println("text-search edits (review carefully):")
		for _, e := range plan.TextSearchEdits {
			fmt.Printf("  [low] %s %s -> %s\n", e.Path, e.Old, e.New)
		}
	}
}
