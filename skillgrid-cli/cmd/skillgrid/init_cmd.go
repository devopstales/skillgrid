package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

type initResult struct {
	BootFile string
	Preamble string // "written" | "kept" | "forced"
	Sentinel string // "upserted"
	Indexed  int
	Ingested int
	Skipped  []string
	Errors   []string
}

func newInitFlagSet(out *flag.FlagSet) (force *bool, docs *stringSliceFlag) {
	out.SetOutput(os.Stderr)
	force = out.Bool("force", false, "rebuild generated preamble")
	docs = new(stringSliceFlag)
	out.Var(docs, "docs", "extra path (repeatable)")
	out.Usage = func() {
		fmt.Fprintln(out.Output(), "usage: skillgrid init [--force] [--docs path]...")
		out.PrintDefaults()
	}
	return force, docs
}

func runProjectInit(args []string) {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			fs := flag.NewFlagSet("init", flag.ContinueOnError)
			newInitFlagSet(fs)
			fs.Usage()
			return
		}
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	force, docs := newInitFlagSet(fs)
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	dir := "."
	if rest := fs.Args(); len(rest) > 0 {
		dir = rest[0]
	}

	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	res, err := projectInit(context.Background(), svc, dir, *force, []string(*docs))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	printInitResult(os.Stdout, res)
}

func printInitResult(w *os.File, res initResult) {
	fmt.Fprintf(w, "boot_file: %s\n", res.BootFile)
	fmt.Fprintf(w, "preamble: %s\n", res.Preamble)
	fmt.Fprintf(w, "sentinel: %s\n", res.Sentinel)
	fmt.Fprintf(w, "indexed: %d\n", res.Indexed)
	fmt.Fprintf(w, "ingested: %d\n", res.Ingested)
	if len(res.Skipped) == 0 {
		fmt.Fprintln(w, "skipped:")
	} else {
		fmt.Fprintf(w, "skipped: %s\n", strings.Join(res.Skipped, ", "))
	}
	if len(res.Errors) == 0 {
		fmt.Fprintln(w, "errors:")
	} else {
		fmt.Fprintf(w, "errors: %s\n", strings.Join(res.Errors, "; "))
	}
}

// initRunIndex resolves the project then runs the existing code-index pipeline
// on the same svc. Named apart from CLI runIndex (mcp.go). Tests may stub it.
var initRunIndex = func(ctx context.Context, svc *service.Service, dir string) (int, error) {
	if _, err := svc.ResolveProject(dir); err != nil {
		return 0, err
	}
	st, err := svc.RunCodeIndex(ctx, dir)
	return st.FilesIndexed, err
}

func projectInit(ctx context.Context, svc *service.Service, dir string, force bool, extraDocs []string) (initResult, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return initResult{}, err
	}
	if !info.IsDir() {
		return initResult{}, fmt.Errorf("%s is not a directory", dir)
	}

	bootPath, preambleState, err := writeBootFile(dir, force)
	if err != nil {
		return initResult{}, err
	}

	res := initResult{
		BootFile: bootPath,
		Preamble: preambleState,
		Sentinel: "upserted",
	}

	indexed, indexErr := initRunIndex(ctx, svc, dir)
	if indexErr != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("index failed: %v", indexErr))
	} else {
		res.Indexed = indexed
	}

	h, closeH, openErr := svc.OpenForDirectory(dir)
	if openErr != nil {
		res.Errors = append(res.Errors, fmt.Sprintf("open project: %v", openErr))
		return res, nil
	}
	defer closeH()

	ingested, skipped, ingestErrs := ingestPaths(ctx, h, dir, extraDocs)
	res.Ingested = ingested
	res.Skipped = append(res.Skipped, skipped...)
	res.Errors = append(res.Errors, ingestErrs...)

	return res, nil
}
