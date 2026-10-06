package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/affected"
)

// readAffectedChanged returns the changed file set for a `search affected`
// invocation: explicit file arguments, a --stdin diff list, or (for --base)
// the merge-base diff. Exactly one source is allowed; none is a validation
// error (no invented changed set).
func readAffectedChanged(fs *flag.FlagSet, stdin bool) ([]string, error) {
	if stdin {
		// Read the diff list into memory BEFORE opening the service: the
		// store init path may replace os.Stdin, and an already-open *os.File
		// read into bytes is immune to that swap.
		list, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		return affected.ParseFileList(string(list))
	}
	files := fs.Args()
	if len(files) > 0 {
		return files, nil
	}
	return nil, fmt.Errorf("affected: provide changed files or --stdin")
}



// runSearchAffected implements `skillgrid search affected` (CLI parity for
// code_affected): --stdin / FILES... / --base REF, --depth, --filter,
// --json, --quiet.
func runSearchAffected(version string, args []string) {
	fs := flag.NewFlagSet("search affected", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut  bool
		quiet    bool
		stdin    bool
		baseRef  string
		depth    int
		filter   string
	)
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.BoolVar(&quiet, "quiet", false, "print only affected test file paths")
	fs.BoolVar(&stdin, "stdin", false, "read the changed file list from stdin (git diff --name-only)")
	fs.StringVar(&baseRef, "base", "", "derive the changed set from git merge-base <ref> HEAD")
	fs.IntVar(&depth, "depth", 0, "bound the traversal depth (default 5)")
	fs.StringVar(&filter, "filter", "", "restrict reported test files to a substring/segment match")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search affected [FILES...] [--stdin | --base REF] [--depth N] [--filter F] [--json] [--quiet]")
		fs.PrintDefaults()
	}
	// Strip the leading "search" + "affected" tokens: main.go dispatches the
	// whole rest (`search affected --stdin ...`), and the flag set must see
	// only the affected-specific flags.
	fsArgs := args
	if len(fsArgs) >= 2 && fsArgs[0] == "search" && fsArgs[1] == "affected" {
		fsArgs = fsArgs[2:]
	} else if len(fsArgs) >= 1 && fsArgs[0] == "affected" {
		fsArgs = fsArgs[1:]
	}
	if err := fs.Parse(fsArgs); err != nil {
		os.Exit(2)
	}
	if stdin && baseRef != "" {
		fmt.Fprintln(os.Stderr, "error: affected: --stdin and --base are mutually exclusive")
		os.Exit(2)
	}
	if baseRef != "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
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
		base, err := affected.AffectedBase(ctx, h.Store().DB, cwd, baseRef, affected.Options{
			Depth: depth, Filter: filter,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		printAffected(base.Result, base, jsonOut, quiet)
		return
	}
	{
		changed, err := readAffectedChanged(fs, stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
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
		res, err := affected.Affected(ctx, h.Store().DB, affected.Options{
			Changed: changed, Depth: depth, Filter: filter,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		printAffected(res, res, jsonOut, quiet)
		return
	}
}

// printAffected renders a code_affected result (human or JSON). In --base
// mode the base variant also prints affected areas + owners, shaped as a
// ready PR comment / CI gate.
func printAffected(res affected.Result, baseRes any, jsonOut, quiet bool) {
	if jsonOut {
		var out any = res
		if b, ok := baseRes.(affected.BaseResult); ok {
			out = b
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	if res.Message != "" {
		fmt.Fprintln(os.Stdout, res.Message)
	}
	if quiet {
		for _, p := range res.TestFiles {
			fmt.Fprintln(os.Stdout, p)
		}
		return
	}
	if len(res.TestFiles) == 0 && res.Message == "" {
		fmt.Fprintln(os.Stderr, "no affected test files")
		return
	}
	fmt.Printf("affected test files (%d):\n", len(res.TestFiles))
	for _, p := range res.TestFiles {
		fmt.Printf("  %s\n", p)
	}
	if len(res.Relationships) > 0 {
		fmt.Printf("relationships (%d hops, depth cap %d):\n", len(res.Relationships), res.Depth)
		for _, r := range res.Relationships {
			fmt.Printf("  %s -> %s [%s %s] depth %d (%s)\n", r.From, r.To, r.Kind, r.Confidence, r.Depth, r.DepPath)
		}
	}
	if b, ok := baseRes.(affected.BaseResult); ok {
		if b.MergeBase != "" {
			fmt.Printf("merge-base: %s (base %s)\n", b.MergeBase, b.BaseRef)
		}
		if len(b.Areas) > 0 {
			fmt.Println("affected areas:")
			for _, a := range b.Areas {
				owners := ""
				if len(a.Owners) > 0 {
					owners = " (tag: " + strings.Join(a.Owners, ", ") + ")"
				}
				fmt.Printf("  %s [%d hops, %d affected tests]%s\n", a.Name, a.Hops, len(a.TestFiles), owners)
				for _, tf := range a.TestFiles {
					fmt.Printf("    test: %s\n", tf)
				}
			}
		}
	}
}
