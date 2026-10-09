package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/devopstales/skillgrid/mnemonic/internal/context_harness"
)

// runCtx is the `ctx` subcommand group: search (two-leg RRF fusion), prime
// (session-start context, reusing the loop prime renderer), and stats (sandbox
// + index row counts). Subcommands are dispatched by name; an unknown or missing
// subcommand prints usage to stderr and exits 2.
//
// This is the CLI surface for the context harness (PR 2 / Unit 2a). Search is the
// tracer's fusion payoff: it fuses the sandbox tool-output leg and the indexed
// code-file leg and labels each result's source. It reuses context_harness.CtxSearch
// (which owns the RRF fusion); this file only wires flags, opens the store, and
// renders rows.
func runCtx(args []string) {
	// No args / bare "ctx": usage.
	if len(args) == 0 {
		ctxUsage()
		os.Exit(2)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "search":
		runCtxSearch(rest)
	case "prime":
		runCtxPrime(rest)
	case "stats":
		runCtxStats(rest)
	case "purge":
		runCtxPurge(rest)
	case "index":
		runCtxIndex(rest)
	case "help", "-h", "--help":
		ctxUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown ctx subcommand %q\n", sub)
		ctxUsage()
		os.Exit(2)
	}
}

func ctxUsage() {
	w := os.Stderr
	fmt.Fprintln(w, "usage: mnemonic ctx <subcommand> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Subcommands:")
	fmt.Fprintln(w, "  search <query>   Two-leg RRF fusion over sandbox tool outputs + indexed files (labels each source; -limit caps results)")
	fmt.Fprintln(w, "  prime            Session-start context (reuses the loop prime renderer; --dir, --session)")
	fmt.Fprintln(w, "  stats            Row counts for the sandbox (tool_outputs) and index (indexed_files)")
	fmt.Fprintln(w, "  purge            Clear the session sandbox (tool_outputs, and context_revisions when present); prints rows removed (-project, -session)")
	fmt.Fprintln(w, "  index <path>     Index a file (or a directory of files) into the context-harness FTS index; re-indexing is idempotent (-project)")
}

// ctxSearchFlags parses search flags: -limit caps the fused results.
type ctxSearchFlags struct {
	limit int
}

func parseCtxSearchFlags(args []string, pos *[]string) ctxSearchFlags {
	// Go's flag package stops at the first non-flag token, so a flag that
	// follows a positional (e.g. "ctx search limitcap -limit 2") would be
	// misread as a positional. Re-parse after each positional to keep
	// positionals and flags interleavable.
	f := ctxSearchFlags{limit: 20}
	rest := args
	for {
		fs := flag.NewFlagSet("ctx search", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		fs.IntVar(&f.limit, "limit", f.limit, "max fused results to return")
		if err := fs.Parse(rest); err != nil {
			os.Exit(2)
		}
		left := fs.Args()
		if len(left) == 0 {
			*pos = append(*pos, fs.Args()...)
			break
		}
		*pos = append(*pos, left[0])
		rest = left[1:]
	}
	return f
}

func runCtxSearch(args []string) {
	var pos []string
	f := parseCtxSearchFlags(args, &pos)
	query := ""
	if len(pos) > 0 {
		query = pos[0]
	}
	if len(query) == 0 {
		fmt.Fprintln(os.Stderr, "error: ctx search requires a query")
		ctxUsage()
		os.Exit(2)
	}

	h, cleanup, err := openLoop(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	results, err := context_harness.CtxSearch(h.Store().DB, query, f.limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if len(results) == 0 {
		fmt.Fprintln(os.Stdout, "no results")
		return
	}
	for _, r := range results {
		// r.Source is "sandbox" or "indexed"; r.ToolOrKind is the sandbox tool
		// name or the indexed file kind; r.Rank is the 1-based fused position.
		fmt.Fprintf(os.Stdout, "%d. [%s] %s (%s)\n    %s\n", r.Rank, r.Source, r.Name, r.ToolOrKind, r.Snippet)
	}
}

func runCtxPrime(args []string) {
	// Reuse the loop prime command's flag parsing + renderer so `ctx prime`
	// and `prime` render identical session-start context.
	dir, session := loopFlags("ctx prime", args)
	fmt.Fprint(os.Stdout, primeText(dir, session))
}

func runCtxStats(args []string) {
	// stats takes no flags beyond the shared -dir (accepted for consistency).
	fs := flag.NewFlagSet("ctx stats", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.String("dir", ".", "repository directory")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	h, cleanup, err := openLoop(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	var sandbox, indexed int
	if err := h.Store().DB.QueryRow(`SELECT count(*) FROM tool_outputs`).Scan(&sandbox); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := h.Store().DB.QueryRow(`SELECT count(*) FROM indexed_files`).Scan(&indexed); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "sandbox (tool_outputs): %d\n", sandbox)
	fmt.Fprintf(os.Stdout, "index (indexed_files): %d\n", indexed)
}

// runCtxPurge clears the session-scoped context-harness sandbox and prints the
// number of rows removed. It opens the store via openLoop (directory-resolved,
// matching search/stats), then delegates to ctxPurge.
func runCtxPurge(args []string) {
	fs := flag.NewFlagSet("ctx purge", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	project := fs.String("project", "", "project id (default: resolved from -dir)")
	session := fs.String("session", "default", "session id to purge")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	h, cleanup, err := openLoop(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	// -project is optional: default to the store's resolved project so the
	// sandbox (keyed by the store's project) is targeted without extra flags.
	pid := *project
	if pid == "" {
		pid = h.ProjectID()
	}
	n, err := ctxPurge(h.Store().DB, pid, *session)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "purged %d row(s)\n", n)
}

// ctxPurge is the testable purge code path: it clears the session sandbox
// (tool_outputs, and context_revisions when present) for (project, session) and
// returns the deleted row count. Factored from runCtxPurge so the test can call
// it directly without shelling out.
func ctxPurge(db *sql.DB, project, session string) (int, error) {
	return context_harness.PurgeSession(db, project, session)
}

// runCtxIndex indexes a single file or a directory of files into the
// context-harness FTS index. When a directory is given it walks the tree and
// indexes every regular file. Re-indexing the same file is idempotent
// (INSERT ... ON CONFLICT DO UPDATE on the (project, name, start_line) unique
// index). The 7-day TTL purge (DefaultIndexedTTL) runs before indexing so
// expired chunks are removed before new ones are written.
func runCtxIndex(args []string) {
	fs := flag.NewFlagSet("ctx index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	project := fs.String("project", "", "project id (default: resolved from -dir)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "error: ctx index requires a file or directory path")
		ctxUsage()
		os.Exit(2)
	}

	h, cleanup, err := openLoop(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	pid := *project
	if pid == "" {
		pid = h.ProjectID()
	}

	// Purge expired indexed rows before writing new ones.
	deleted, err := context_harness.PurgeExpiredIndexed(h.Store().DB, pid, time.Now(), context_harness.DefaultIndexedTTL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error purging expired index:", err)
		os.Exit(1)
	}
	if deleted > 0 {
		fmt.Fprintf(os.Stderr, "purged %d expired indexed row(s)\n", deleted)
	}

	// Collect files: a single path is used directly; a directory is walked.
	var files []string
	info, err := os.Stat(pos[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if info.IsDir() {
		root := pos[0]
		filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return nil // skip unreadable entries
			}
			if fi.IsDir() {
				return nil
			}
			files = append(files, path)
			return nil
		})
	} else {
		files = []string{pos[0]}
	}

	totalChunks := 0
	for _, f := range files {
		n, err := context_harness.IndexFile(h.Store().DB, pid, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error indexing %s: %v\n", f, err)
			os.Exit(1)
		}
		totalChunks += n
	}
	fmt.Fprintf(os.Stdout, "indexed %d file(s), %d chunk(s)\n", len(files), totalChunks)
}
