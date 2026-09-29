package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/wiki"
)

// wikiExit is the process-exit function used by the wiki commands. It is a
// variable (defaulting to os.Exit) so the test binary can intercept a non-zero
// exit if needed; dispatchWiki itself returns the code and is test-pure.
var wikiExit = os.Exit

// runWiki handles `skillgrid wiki <subcommand>` (R9.1–R9.3). With no
// subcommand it prints usage and exits 2. `wiki compile` renders the pillar-1
// bundle (ADRs + State) to .wiki/; `wiki lint` walks .wiki/wiki/**/*.md and
// runs the OKF conformance gate, exiting non-zero on any failure (R9.2).
func runWiki(version string, args []string) {
	_ = version
	wikiExit(dispatchWiki(args))
}

// dispatchWiki is the pure decision layer of the wiki command. It prints
// usage / errors to the given writers and returns the exit code the process
// should exit with (0 for success). It never calls os.Exit itself, which makes
// it directly unit-testable.
func dispatchWiki(args []string) int {
	if len(args) == 0 {
		printWikiUsage(os.Stderr)
		return 2
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "compile":
		return runWikiCompile(rest)
	case "lint":
		return runWikiLint(rest)
	case "help", "-h", "--help":
		printWikiUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown wiki subcommand %q (see `skillgrid wiki --help`)\n", cmd)
		return 2
	}
}

func printWikiUsage(w io.Writer) {
	fmt.Fprintln(w, `skillgrid wiki — compile and lint the llmwiki-okf wiki bundle

Usage:
  skillgrid wiki compile [flags]
  skillgrid wiki lint [flags]

Commands:
  compile   Render the pillar-1 bundle (ADRs + State) to .wiki/
  lint      Run the OKF conformance gate over .wiki/wiki/**/*.md

Compile flags:
  --project DIR   project root containing .skillgrid/ (default: CWD)
  --out DIR       output directory (default: <project>/.wiki)
  --no-llm        do not invoke an LLM for summaries (pillar 1 is already LLM-free)
  --json          print the compile result as JSON

Lint flags:
  --project DIR   project root containing .skillgrid/ (default: CWD)`)
}

// resolveWikiProjectDir resolves --project (default CWD) to an absolute path.
func resolveWikiProjectDir(flag string) (string, error) {
	dir := flag
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

// runWikiCompile is the testable body of `wiki compile`. It returns the exit
// code (0 success, 1 failure, 2 usage error) and prints output to
// os.Stdout/os.Stderr.
func runWikiCompile(args []string) int {
	fs := flag.NewFlagSet("wiki compile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		project string
		out     string
		jsonOut bool
	)
	fs.StringVar(&project, "project", "", "project root containing .skillgrid/ (default: CWD)")
	fs.StringVar(&out, "out", "", "output directory (default: <project>/.wiki)")
	fs.BoolVar(&_blankNoLLM, "no-llm", false, "do not invoke an LLM for summaries")
	fs.BoolVar(&jsonOut, "json", false, "print the compile result as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	proj, err := resolveWikiProjectDir(project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	outDir := out
	if outDir == "" {
		outDir = filepath.Join(proj, ".wiki")
	}

	webRows, err := loadFreshWebRows(proj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: web_cache: %v (continuing without web rows)\n", err)
	}

	res, err := wiki.Compile(wiki.CompileInput{
		ProjectDir: proj,
		OutDir:     outDir,
		Now:        time.Now(),
		WebRows:    webRows,
		RawRoot:    filepath.Join(proj, "raw"),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if jsonOut {
		printWikiCompileJSON(res)
		return 0
	}
	fmt.Printf("wiki compile: %d written, %d unchanged (out: %s)\n", res.Written, res.Unchanged, outDir)
	return 0
}

// _blankNoLLM backs the --no-llm flag. Pillar 1 is already LLM-free, so the
// flag is accepted but has no effect; it is present for CLI parity and so a
// future LLM-driven pillar can honor it without a breaking change.
var _blankNoLLM bool

// loadFreshWebRows reads fresh web_cache rows for the project at proj (R5):
// source in {context7, exa, deepwiki, fetch}, expires_at IS NULL or in the
// future. A missing store or project yields no rows (not an error), so
// compiling a project without mnemonic data still works.
func loadFreshWebRows(proj string) ([]wiki.WebRow, error) {
	dd, err := service.DefaultDataDir()
	if err != nil {
		return nil, err
	}
	svc := service.New(dd)
	h, cleanup, err := svc.OpenForDirectory(proj)
	if err != nil {
		if isMissingStore(err) {
			return nil, nil
		}
		return nil, err
	}
	defer cleanup()

	rows, err := h.Store().DB.QueryContext(context.Background(), `
		SELECT source, url, title, query, library_id, version_tag,
		       content_hash, fetched_at, expires_at
		FROM web_cache
		WHERE project = ?
		  AND source IN ('context7', 'exa', 'deepwiki', 'fetch')
		  AND (expires_at IS NULL OR datetime(expires_at) > datetime('now'))
		ORDER BY id`, h.ProjectID())
	if err != nil {
		return nil, fmt.Errorf("query web_cache: %w", err)
	}
	defer rows.Close()

	var out []wiki.WebRow
	for rows.Next() {
		var r wiki.WebRow
		var url, title, query, libraryID, versionTag, contentHash, expiresAt sql.NullString
		if err := rows.Scan(&r.Source, &url, &title, &query, &libraryID, &versionTag,
			&contentHash, &r.FetchedAt, &expiresAt); err != nil {
			return nil, fmt.Errorf("scan web_cache row: %w", err)
		}
		r.URL = url.String
		r.Title = title.String
		r.Query = query.String
		r.LibraryID = libraryID.String
		r.VersionTag = versionTag.String
		r.ContentHash = contentHash.String
		r.ExpiresAt = expiresAt.String
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate web_cache rows: %w", err)
	}
	return out, nil
}

// isMissingStore reports whether err indicates the project's store file is
// absent (a project that has never been opened with mnemonic).
func isMissingStore(err error) bool {
	return err != nil && strings.Contains(err.Error(), "no such file")
}

// runWikiLint is the testable body of `wiki lint`. It returns the exit code
// (0 clean, 1 conformance failures, 2 usage error).
func runWikiLint(args []string) int {
	fs := flag.NewFlagSet("wiki lint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var project string
	fs.StringVar(&project, "project", "", "project root containing .skillgrid/ (default: CWD)")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	proj, err := resolveWikiProjectDir(project)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	wikiDir := filepath.Join(proj, ".wiki", "wiki")
	fails := wikiLintFiles(wikiDir)
	for _, f := range fails {
		fmt.Fprintf(os.Stderr, "FAIL %s\n", f.RelPath)
		for _, e := range f.Errors {
			fmt.Fprintf(os.Stderr, "  - %v\n", e)
		}
	}
	if len(fails) > 0 {
		fmt.Fprintf(os.Stderr, "wiki lint: %d page(s) failed conformance\n", len(fails))
		return 1
	}
	fmt.Printf("wiki lint: all pages conformant (%s)\n", wikiDir)
	return 0
}

// printWikiCompileJSON prints a stable JSON summary of a compile.
func printWikiCompileJSON(res wiki.CompileResult) {
	type pathList []string
	payload := map[string]interface{}{
		"written":   res.Written,
		"unchanged": res.Unchanged,
		"paths":     pathList(res.Paths),
	}
	enc, err := json.Marshal(payload)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return
	}
	fmt.Println(enc)
}

// wikiLintFailure is one page that failed the conformance gate.
type wikiLintFailure struct {
	RelPath string
	Errors  []error
}

// wikiLintFiles walks dir for *.md files and runs Conform on each, returning
// the failures in stable (path-sorted) order. A missing dir yields no
// failures (lint of an uncompiled bundle is clean, not an error).
func wikiLintFiles(dir string) []wikiLintFailure {
	var fails []wikiLintFailure
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // missing dir → clean
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		if errs := wiki.Conform(string(data)); len(errs) > 0 {
			rel, _ := filepath.Rel(dir, path)
			fails = append(fails, wikiLintFailure{RelPath: filepath.ToSlash(rel), Errors: errs})
		}
		return nil
	})
	sort.Slice(fails, func(i, j int) bool { return fails[i].RelPath < fails[j].RelPath })
	return fails
}
