// index.go — the index command group (spec 008-transport §4): the
// code-index's CLI surface. Each subcommand parses its flags, calls the
// SHARED CodeHandler (the same one the MCP server wraps — the K5/K6
// byte-identical seam), and prints the result (human-readable by default,
// the envelope JSON with --json — the same marshal the MCP tools use).
//
// The subcommand list is the spec's literal table: index, status, search,
// orient, explain, path, impact, affected, grep, read, watch. The project
// selection (spec §4) resolves at call time: explicit --project > the
// detector's chain; --json for machine output (matches the MCP contract).

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/extract"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/graph"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/codeindex/search"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// codeHandlerFor builds the shared CodeHandler over a store (the production
// wiring; the test seam testCodeHandlerFor overrides it). The embedder is the
// semantic leg's seam (a nil seam is the degraded: no_vector floor); the
// detector is the project chain (explicit > env > git > basename).
var codeHandlerFor = func(cfg config.Config, st store.Store) *handler.CodeHandler {
	emb, err := cfg.BuildEmbedder()
	if err != nil {
		// A build failure is a nil seam (the degraded floor), not a hard
		// error: the FTS leg still answers.
		emb = nil
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	return handler.NewCodeHandler(st, emb, cfg, "", &gitDetector{det: det})
}

// runIndex runs the `mnemonic index` command group (spec 008 §4). The
// subcommand dispatches onto the shared CodeHandler (the same one the MCP
// server wraps — the K5/K6 seam).
func runIndex(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		// No subcommand → the incremental index (the spec's default:
		// `mnemonic index [flags]` runs the index).
		return runIndexIndex(fl, cwd, args)
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "index", "":
		return runIndexIndex(fl, cwd, rest)
	case "status":
		return runIndexStatus(fl, cwd, rest)
	case "search":
		return runIndexSearch(fl, cwd, rest)
	case "orient":
		return runIndexOrient(fl, cwd, rest)
	case "explain":
		return runIndexExplain(fl, cwd, rest)
	case "path":
		return runIndexPath(fl, cwd, rest)
	case "impact":
		return runIndexImpact(fl, cwd, rest)
	case "affected":
		return runIndexAffected(fl, cwd, rest)
	case "grep":
		return runIndexGrep(fl, cwd, rest)
	case "read":
		return runIndexRead(fl, cwd, rest)
	case "watch":
		return runIndexWatch(fl, cwd, rest)
	default:
		fmt.Fprintf(errW(), "error: unknown index subcommand %q (see: mnemonic index --help)\n", sub)
		return exitUsage
	}
}

// codeRuntime opens the runtime + builds the CodeHandler + sets the root (the
// git toplevel of cwd, the index walk's root). It is the single wiring point
// the index subcommands share (the same store the mem commands open — the
// K6 seam).
func codeRuntime(cwd string, fl config.Flags, explicit string) (*handler.CodeHandler, store.Store, error) {
	cfg, err := config.Load(cwd, fl)
	if err != nil {
		return nil, nil, err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	p, err := det.Resolve(cwd, explicit, gitToplevel)
	if err != nil {
		return nil, nil, err
	}
	cfg.Project.Name = p
	st, err := storeFor(cfg, p)
	if err != nil {
		return nil, nil, err
	}
	ch := codeHandlerFor(cfg, st)
	// The root is the git toplevel of cwd (the index walk's root; the
	// detector's binding). A lookup failure falls back to cwd (the CLI is a
	// human process; the human's directory is the binding).
	root := cwd
	if gitRoot, err := gitToplevel(cwd); err == nil && gitRoot != "" {
		root = gitRoot
	}
	ch.SetRoot(root)
	return ch, st, nil
}

// indexProjectFlag is the shared --project selector for the index reads.
func indexProjectFlag(fs *flag.FlagSet) *string {
	return fs.String("project", "", "project selector (beats "+config.EnvProject+" and detection)")
}

// runIndexIndex runs `mnemonic index [flags]` (the incremental index — the
// code_index write).
func runIndexIndex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	full := fs.Bool("full", false, "full re-index (default: incremental)")
	lsp := fs.Bool("lsp", false, "run the LSP pass (the language-server member-call tier; off by default)")
	pdg := fs.Bool("pdg", false, "run the PDG pass (v2.1)")
	communities := fs.Bool("communities", false, "run the communities pass (v2.1)")
	process := fs.Bool("process", false, "run the process pass (v2.1)")
	noProgress := fs.Bool("no-progress", false, "suppress the live progress line (default: on for an interactive index)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	// Live progress (the v1 index streamed per-file feedback; a silent
	// multi-minute re-index looks stuck). Off for --json (stdout is the
	// machine result; the progress would interleave) and for --no-progress.
	// It writes a single \r-overwritten line to stderr (stdout stays clean
	// for the result), ending in a newline on completion.
	if !*jsonOut && !*noProgress {
		ch.SetProgressFn(func(done, total int, path string) {
			if total <= 0 {
				return
			}
			pct := done * 100 / total
			fmt.Fprintf(errW(), "\rindexing %d/%d (%d%%)  %s", done, total, pct, truncatePath(path, 48))
		})
	}
	res, err := ch.Index(context.Background(), *proj, *full, *lsp, *pdg, *communities, *process)
	if !*jsonOut {
		if !*noProgress {
			fmt.Fprintln(errW()) // terminate the \r progress line
		}
		// --full on an unchanged tree (or any run that examined files but wrote
		// none) must RESPOND, not be a silent no-op that looks like a hang.
		// FilesIndexed==0 + FilesUnchanged>0 = "nothing to do". (A true first
		// index of an empty tree has FilesUnchanged==0 and is not reported.)
		// The degradation report below is independent of --no-progress: a
		// dead/unpulled embedder is a result to surface, not the live stream.
		if err == nil && res.Degraded != "" {
			// The run degraded (the semantic leg failed / an optional tier was
			// unavailable). Surface it readably, not just in the JSON — a
			// dead/unpulled embedder must be visible in human mode too.
			fmt.Fprintf(errW(), "index: degraded: %s\n", res.Degraded)
			for _, w := range res.Warnings {
				fmt.Fprintf(errW(), "  - %s\n", w)
			}
		}
		if err == nil && res.Degraded == "" && res.FilesIndexed == 0 && res.FilesUnchanged > 0 {
			what := "index"
			if *full {
				what = "full re-index"
			}
			fmt.Fprintf(errW(), "%s: nothing to do — %d file(s) unchanged\n", what, res.FilesUnchanged)
		}
	}
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// truncatePath shortens a long repo-relative path for the progress line,
// keeping the tail (the file name + last dirs) — the head is the noise.
func truncatePath(p string, max int) string {
	if len(p) <= max {
		return p
	}
	return "…" + p[len(p)-max+1:]
}

// runIndexStatus runs `mnemonic status` (the code_status read — the 4-state
// freshness, the per-language tier, the schema version, the fair coverage).
func runIndexStatus(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Status(context.Background(), *proj)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexSearch runs `mnemonic search <query>` (the code_search read — the
// hybrid search). The K5 seam: the same CodeHandler method the MCP tool calls;
// the --json output is the envelope marshal of the same result (byte-
// identical to the MCP response).
func runIndexSearch(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	limit := fs.Int("limit", 10, "maximum hits (default 10)")
	lang := fs.String("language", "", "filter by language")
	under := fs.String("under", "", "package prefix scope")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("search: a query is required")})
	}
	query := fs.Arg(0)
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Search(context.Background(), query, search.SearchQuery{
		Project: *proj, Limit: *limit, Language: *lang, Under: *under,
	})
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexOrient runs `mnemonic orient <symbol|package|file>` (the
// code_orient read — the navigation entry point).
func runIndexOrient(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("orient", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	depth := fs.Int("depth", 1, "the orientation depth (default 1)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("orient: a symbol, package, or file is required")})
	}
	target := fs.Arg(0)
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	unit, err := orientUnitForTarget(context.Background(), ch, target)
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	res, err := ch.Orient(context.Background(), unit, *depth)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexExplain runs `mnemonic explain <symbol>` (the code_explain read).
func runIndexExplain(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("explain", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("explain: a symbol is required")})
	}
	sym := fs.Arg(0)
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	syms, err := ch.Store().SymbolByName(context.Background(), sym)
	if err != nil || len(syms) == 0 {
		return finish(fs, *jsonOut, dispatchResult{err: fmt.Errorf("explain: symbol %q not found", sym)})
	}
	res, err := ch.Explain(context.Background(), syms[0].ID)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexPath runs `mnemonic path <from> [to]` (the code_path read — the
// BFS shortest path, or the graph-stop view when to is omitted).
func runIndexPath(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("path", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("path: a from symbol is required")})
	}
	from := fs.Arg(0)
	to := ""
	if fs.NArg() >= 2 {
		to = fs.Arg(1)
	}
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	fromSyms, err := ch.Store().SymbolByName(context.Background(), from)
	if err != nil || len(fromSyms) == 0 {
		return finish(fs, *jsonOut, dispatchResult{err: fmt.Errorf("path: from symbol %q not found", from)})
	}
	toID := int64(0)
	if to != "" {
		toSyms, err := ch.Store().SymbolByName(context.Background(), to)
		if err != nil || len(toSyms) == 0 {
			return finish(fs, *jsonOut, dispatchResult{err: fmt.Errorf("path: to symbol %q not found", to)})
		}
		toID = toSyms[0].ID
	}
	res, err := ch.Path(context.Background(), fromSyms[0].ID, toID)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexImpact runs `mnemonic impact <symbol>` (the code_impact read — the
// risk-tiered blast radius).
func runIndexImpact(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("impact", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	minConf := fs.String("min-confidence", "", "minimum edge confidence")
	maxDepth := fs.Int("max-depth", 5, "maximum traversal depth (default 5)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("impact: a symbol is required")})
	}
	sym := fs.Arg(0)
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	syms, err := ch.Store().SymbolByName(context.Background(), sym)
	if err != nil || len(syms) == 0 {
		return finish(fs, *jsonOut, dispatchResult{err: fmt.Errorf("impact: symbol %q not found", sym)})
	}
	res, err := ch.Impact(context.Background(), syms[0].ID, *minConf, *maxDepth)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// gitChangedFiles returns the repo-relative paths changed between <base> and
// HEAD (git merge-base <base> HEAD + diff --name-only <mb> HEAD) — the
// code_affected base input (spec 004 §6); the CLI owns the git-diff.
func gitChangedFiles(dir, base string) ([]string, error) {
	mb, err := gitOutput(dir, "merge-base", base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("affected: git merge-base %s HEAD failed: %w", base, err)
	}
	mb = strings.TrimSpace(string(mb))
	out, err := gitOutput(dir, "diff", "--name-only", mb)
	if err != nil {
		return nil, fmt.Errorf("affected: git diff --name-only %s..HEAD failed: %w", mb, err)
	}
	var files []string
	for _, line := range strings.Split(string(out), "\n") {
		if p := strings.TrimSpace(line); p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// runIndexAffected runs `mnemonic affected [base]` (the code_affected read —
// the tests to run after a change). The --base flag takes a git ref and the
// CLI resolves the changed set via git merge-base + diff; the --changed flag
// takes a comma-separated file list; --stdin reads it from stdin.
func runIndexAffected(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("affected", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	changed := fs.String("changed", "", "comma-separated changed file paths")
	stdin := fs.Bool("stdin", false, "read the changed file list from stdin")
	base := fs.String("base", "", "git ref to diff against (the CLI does the git-diff)")
	depth := fs.Int("depth", 5, "the traversal depth (default 5)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	var files []string
	if *base != "" {
		f, err := gitChangedFiles(cwd, *base)
		if err != nil {
			fmt.Fprintf(outW(), "error: %s\n", err.Error())
			return exitErr
		}
		files = f
	} else if *changed != "" {
		files = splitComma(*changed)
	} else if *stdin {
		files = readStdinLines()
	} else if fs.NArg() >= 1 {
		files = []string{fs.Arg(0)}
	}
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Affected(context.Background(), files, *depth)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexGrep runs `mnemonic grep <pattern>` (the code_grep read — the
// structural AST grep).
func runIndexGrep(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("grep", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	lang := fs.String("language", "", "filter by language")
	path := fs.String("path", "", "the file path scope")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("grep: a pattern is required")})
	}
	pattern := fs.Arg(0)
	_ = path // the path scope is the under-prefix (the search's T10).
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Grep(context.Background(), pattern, *lang)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexRead runs `mnemonic read <path>` (the code_read read — fetch a
// span, secret redaction).
func runIndexRead(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("read", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	startLine := fs.Int("start-line", 0, "start line (1-based)")
	endLine := fs.Int("end-line", 0, "end line (1-based)")
	mode := fs.String("mode", "full", "full (the live span) or content (the drill handle)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if fs.NArg() < 1 {
		return finish(fs, *jsonOut, dispatchResult{err: errUsage("read: a path is required")})
	}
	path := fs.Arg(0)
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Read(context.Background(), path, *startLine, *endLine, *mode)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// runIndexWatch runs `mnemonic watch` (the code_watch — start the live
// incremental watcher; the onSync seam is the Index driver, the
// OnSyncFunc func(paths []string) error signature from TICKET-06).
func runIndexWatch(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	proj := indexProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	ch, st, err := codeRuntime(cwd, fl, *proj)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	res, err := ch.Watch(context.Background(), *proj)
	return finish(fs, *jsonOut, dispatchResult{result: res, err: err})
}

// orientUnitForTarget resolves a CLI target (a symbol name, a package path,
// or a file path) onto the graph.OrientUnit (the orient's unit). A symbol
// name resolves to its id; a package/file path is used as-is (the orient's
// path-based units).
func orientUnitForTarget(ctx context.Context, ch *handler.CodeHandler, target string) (graph.OrientUnit, error) {
	// A file path (it exists on disk or in the index) → the file unit.
	if _, err := ch.Store().FileByPath(ctx, target); err == nil {
		return graph.OrientUnit{Kind: graph.UnitFile, Path: target}, nil
	}
	// A symbol name → the symbol unit (resolve to its id).
	syms, err := ch.Store().SymbolByName(ctx, target)
	if err == nil && len(syms) > 0 {
		return graph.OrientUnit{Kind: graph.UnitSymbol, ID: syms[0].ID}, nil
	}
	// Otherwise treat it as a package path (the package unit).
	return graph.OrientUnit{Kind: graph.UnitPackage, Path: target}, nil
}

// splitComma splits a comma-separated string into trimmed non-empty parts.
func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// readStdinLines reads the changed file list from stdin (one path per line).
func readStdinLines() []string {
	var out []string
	buf := make([]byte, 4096)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			for _, line := range strings.Split(string(buf[:n]), "\n") {
				if line = strings.TrimSpace(line); line != "" {
					out = append(out, line)
				}
			}
		}
		if err != nil {
			break
		}
	}
	return out
}

// unused imports guard (the CLI imports the packages it composes).
var _ = codeindex.Index
var _ = extract.Stamp
var _ = graph.Impact
var _ = search.Hybrid
