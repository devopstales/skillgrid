// projectwiki.go — the projectwiki command group (008-transport §6, the 011 openwiki port):
// a thin wrapper over the shared ProjectWikiHandler (the same wiring the MCP server
// uses — the CLI and MCP never diverge, the K6 byte-identical seam).
// Exit codes: 0 ok · 1 drift under --strict · 2 usage/operational.
//
// projectwiki_eval is CLI-only (a harness command — not in the 12-tool D1 surface).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/projectwiki"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/envelope"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/handler"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/mcp"
)

func runProjectWiki(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("projectwiki: a subcommand is required (query|read|search|generate|validate|lint|status|cite-source|index|reindex|bootstrap|rebuild-meta|eval|mcp)")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "query":
		return projectWikiQuery(fl, cwd, rest)
	case "read":
		return projectWikiRead(fl, cwd, rest)
	case "search":
		return projectWikiSearch(fl, cwd, rest)
	case "generate":
		return projectWikiGenerate(fl, cwd, rest)
	case "validate":
		return projectWikiValidate(fl, cwd, rest)
	case "lint":
		return projectWikiLint(fl, cwd, rest)
	case "status":
		return projectWikiStatus(fl, cwd, rest)
	case "cite-source", "cite":
		return projectWikiCiteSource(fl, cwd, rest)
	case "index":
		return projectWikiIndex(fl, cwd, rest)
	case "reindex":
		return projectWikiReindex(fl, cwd, rest)
	case "bootstrap":
		return projectWikiBootstrap(fl, cwd, rest)
	case "rebuild-meta":
		return projectWikiRebuildMeta(fl, cwd, rest)
	case "eval":
		return projectWikiEval(fl, cwd, rest)
	case "mcp":
		return runMCP(fl, rest, mcp.FamilyProjectWiki)
	default:
		return errExit("projectwiki: unknown subcommand %q (query|read|search|generate|validate|lint|status|cite-source|index|reindex|bootstrap|rebuild-meta|eval|mcp)", sub)
	}
}

// projectWikiOpError prints an operational error and returns the usage/operational
// exit code (011 §3.9: 2 — distinct from 1, which is --strict drift only).
func projectWikiOpError(format string, args ...any) int {
	fmt.Fprintf(errW(), "error: "+format+"\n", args...)
	return exitUsage
}

// projectWikiJSON prints the --json output: the envelope wire shape (snake_case,
// the degraded flag with its default reason) — the same marshal the MCP
// tools use, the K6 byte-identical contract (finish does the same for the
// memory commands). res must marshal to a JSON object (wrap slices in a map).
func projectWikiJSON(res any) int {
	degraded, reason := envelope.ExtractDegraded(res)
	env, err := envelope.WithDegraded(res, degraded, reason)
	if err != nil {
		return projectWikiOpError("envelope: %v", err)
	}
	fmt.Fprintln(outW(), mustMarshal(env))
	return exitOK
}

// projectWikiRoot returns the project wiki vault root for this project
// (.skillgrid/projectwiki/ — the delegate carries the legacy pre-rename
// fallback for old vaults).
func projectWikiRoot(cwd string) string {
	return projectwiki.Root(cwd)
}

// projectWikiHandler opens the store for the resolved project and builds the shared
// ProjectWikiHandler over it (008-transport §6: the CLI is a thin wrapper over the
// same handlers — the K6 seam). The roots are set per call (006 §2: the
// anchor paths are working-tree-relative, so the repo root is the git
// toplevel of cwd, the vault root the sentinel dir under it). The resolved
// project is returned for the callers that assert it (the write tools'
// expected_project).
func projectWikiHandler(cwd string, fl config.Flags, explicit string) (store.Store, *handler.ProjectWikiHandler, string, error) {
	cfg, err := config.Load(cwd, fl)
	if err != nil {
		return nil, nil, "", err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	p, err := det.Resolve(cwd, explicit, gitToplevel)
	if err != nil {
		return nil, nil, "", err
	}
	cfg.Project.Name = p
	st, err := storeFor(cfg, p)
	if err != nil {
		return nil, nil, "", err
	}
	repo := cwd
	if gitRoot, err := gitToplevel(cwd); err == nil && gitRoot != "" {
		repo = gitRoot
	}
	kh := handler.NewProjectWikiHandler(st, cfg, &gitDetector{det: det})
	kh.SetRoots(projectWikiRoot(cwd), repo)
	return st, kh, p, nil
}

// projectWikiProjectFlag registers the project selector on fs. Read pf.project only
// AFTER fs.Parse (the parse fills the struct in place — the memory commands'
// convention; a pre-parse read is "").
func projectWikiProjectFlag(fs *flag.FlagSet) *projectFlag {
	var pf projectFlag
	pf.add(fs)
	return &pf
}

// projectWikiParsePrelim parses the shared flags (project selector + --json) and
// enforces the --all guard (the cross-project store is pending).
func projectWikiParsePrelim(name string, args []string, fs *flag.FlagSet) (*projectFlag, *bool, error) {
	fs.SetOutput(os.Stderr)
	pf := projectWikiProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return nil, nil, fmt.Errorf("projectwiki %s: %w", name, err)
	}
	if pf.all {
		return nil, nil, fmt.Errorf("projectwiki %s: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>", name)
	}
	return pf, jsonOut, nil
}

func projectWikiQuery(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki query", flag.ContinueOnError)
	var typ, matchMode string
	var limit int
	groundedOnly := fs.Bool("grounded-only", false, "restrict to grounded pages")
	fs.StringVar(&typ, "type", "", "filter by page type")
	fs.StringVar(&matchMode, "match-mode", "", "term matching: any|all")
	fs.IntVar(&limit, "limit", 10, "max hits")
	pf, jsonOut, err := projectWikiParsePrelim("query", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return errExit("projectwiki query: a query is required")
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Query(context.Background(), cwd, p, handler.QueryRequest{
		Query: query, Type: typ, Limit: limit, MatchMode: matchMode, GroundedOnly: *groundedOnly,
	})
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	for _, hit := range res.Results {
		fmt.Fprintf(outW(), "%s | %s | %s (score %.3f)\n", hit.RelPath, hit.Title, hit.Type, hit.Score)
	}
	fmt.Fprintf(outW(), "projectwiki query: %d hit(s)\n", res.Count)
	if res.Thin {
		fmt.Fprintf(outW(), "projectwiki query: thin result — fallback: %s\n", res.Fallback)
	}
	if res.Degraded != "" {
		fmt.Fprintf(outW(), "projectwiki query: degraded: %s\n", res.Degraded)
	}
	return exitOK
}

func projectWikiRead(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki read", flag.ContinueOnError)
	var relPath, section string
	backlinks := fs.Bool("backlinks", false, "list the pages that link here")
	fs.StringVar(&relPath, "path", "", "the page rel_path")
	fs.StringVar(&section, "section", "", "a single ## section name (empty = full body)")
	pf, jsonOut, err := projectWikiParsePrelim("read", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if relPath == "" {
		return errExit("projectwiki read: --path <rel_path> is required")
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Read(context.Background(), cwd, relPath, section, p, *backlinks)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "%s (%s%s)\n", res.Title, res.Type, pinCheckLabel(res))
	fmt.Fprintln(outW(), res.Body)
	if *backlinks && len(res.Backlinks) > 0 {
		fmt.Fprintf(outW(), "backlinks:\n")
		for _, bl := range res.Backlinks {
			fmt.Fprintf(outW(), "  %s\n", bl)
		}
	}
	return exitOK
}

// pinCheckLabel renders the pin-check state for grounded pages ("" for
// non-grounded).
func pinCheckLabel(res *projectwiki.ReadResult) string {
	if res.PinCheck == nil {
		return ""
	}
	return ", pin: " + res.PinCheck.Status
}

func projectWikiSearch(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki search", flag.ContinueOnError)
	var limit int
	fs.IntVar(&limit, "limit", 25, "max hits")
	pf, jsonOut, err := projectWikiParsePrelim("search", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return errExit("projectwiki search: a query is required")
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Search(context.Background(), cwd, query, p, limit)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	for _, hit := range res.Results {
		fmt.Fprintf(outW(), "%s | %s | %s\n", hit.RelPath, hit.Title, hit.Snippet)
	}
	fmt.Fprintf(outW(), "projectwiki search: %d hit(s)\n", res.Count)
	return exitOK
}

func projectWikiGenerate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki generate", flag.ContinueOnError)
	full := fs.Bool("full", false, "full re-derive (ignore the processed set)")
	pf, jsonOut, err := projectWikiParsePrelim("generate", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Generate(context.Background(), cwd, pf.project, p, *full)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "projectwiki generate: changed=%d noop=%v degraded=%v run=%s", res.Changed, res.Noop, res.Degraded, res.RunID)
	for _, rel := range res.RelPaths {
		fmt.Fprintf(outW(), "\n  wrote %s", rel)
	}
	fmt.Fprintln(outW())
	return exitOK
}

func projectWikiValidate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki validate", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "fail (exit 1) on any drifted page")
	pf, jsonOut, err := projectWikiParsePrelim("validate", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	rels := fs.Args()
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Validate(context.Background(), cwd, p, rels, *strict)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	for _, v := range res.Results {
		fmt.Fprintf(outW(), "projectwiki validate: %s %s (%s)\n", v.RelPath, map[bool]string{true: "ok", false: "drift"}[v.OK], v.Reason)
	}
	if res.Checked > 1 {
		fmt.Fprintf(outW(), "projectwiki validate: %d checked, %d drifted\n", res.Checked, res.Drifted)
	}
	if *strict && !res.OK {
		return exitErr
	}
	return exitOK
}

func projectWikiLint(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki lint", flag.ContinueOnError)
	fix := fs.Bool("fix", false, "auto-repair the safe findings (missing titles)")
	strict := fs.Bool("strict", false, "fail (exit 1) when issues remain")
	pf, jsonOut, err := projectWikiParsePrelim("lint", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Lint(context.Background(), cwd, p, *fix)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	if len(res.Issues) == 0 {
		fmt.Fprintln(outW(), "projectwiki lint: clean")
	} else {
		for _, iss := range res.Issues {
			fmt.Fprintf(outW(), "projectwiki lint: %s %s [%s] %s\n", iss.RelPath, iss.Code, iss.Severity, iss.Message)
		}
		fmt.Fprintf(outW(), "projectwiki lint: %d issue(s)", res.Count)
		if res.Fixed > 0 {
			fmt.Fprintf(outW(), " (fixed %d)", res.Fixed)
		}
		fmt.Fprintln(outW())
	}
	if *strict {
		// Spec 007 §2: only error-severity findings block under --strict.
		for _, iss := range res.Issues {
			if iss.Severity == projectwiki.SeverityError {
				return exitErr
			}
		}
	}
	return exitOK
}

func projectWikiStatus(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki status", flag.ContinueOnError)
	pf, jsonOut, err := projectWikiParsePrelim("status", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Status(context.Background(), cwd, p)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "projectwiki status: project=%s indexed=%d phase=%s baselines=%d events=%d stale=%d pending=%d orphans=%d\n",
		res.Project, res.Indexed, res.Phase, res.Baselines, res.Events, res.Stale, res.Pending, res.Orphans)
	return exitOK
}

func projectWikiCiteSource(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki cite-source", flag.ContinueOnError)
	var name, version, url, commit string
	fs.StringVar(&name, "name", "", "the external source name (required)")
	fs.StringVar(&version, "cite-version", "", "the cited version/tag (--version is the global version flag)")
	fs.StringVar(&url, "url", "", "the source URL")
	fs.StringVar(&commit, "commit", "", "the source commit (when known)")
	pf, jsonOut, err := projectWikiParsePrelim("cite-source", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	relPath := fs.Arg(0)
	if relPath == "" {
		return errExit("projectwiki cite-source: a page rel_path is required")
	}
	if name == "" {
		return errExit("projectwiki cite-source: --name <external name> is required")
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.CiteSource(context.Background(), cwd, pf.project, p, relPath, name, version, url, commit)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	if res.AlreadyCited {
		fmt.Fprintf(outW(), "projectwiki cite-source: %s already cited on %s\n", res.Source, relPath)
	} else {
		fmt.Fprintf(outW(), "projectwiki cite-source: %s recorded on %s (fingerprint %s)\n", res.Source, relPath, res.BuildFingerprint)
	}
	return exitOK
}

func projectWikiIndex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki index", flag.ContinueOnError)
	pf, jsonOut, err := projectWikiParsePrelim("index", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Index(context.Background(), cwd, p)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "projectwiki index: scanned=%d changed=%d deleted=%d\n", res.Scanned, res.Changed, res.Deleted)
	return exitOK
}

func projectWikiReindex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki reindex", flag.ContinueOnError)
	pf, jsonOut, err := projectWikiParsePrelim("reindex", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Reindex(context.Background(), cwd, p)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "projectwiki reindex: scanned=%d changed=%d\n", res.Scanned, res.Changed)
	return exitOK
}

func projectWikiBootstrap(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki bootstrap", flag.ContinueOnError)
	pf, _, err := projectWikiParsePrelim("bootstrap", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Bootstrap(context.Background(), cwd, p)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if len(res.Created) == 0 {
		fmt.Fprintf(outW(), "projectwiki bootstrap: vault present at %s (no-op)\n", res.Root)
	} else {
		fmt.Fprintf(outW(), "projectwiki bootstrap: created %s at %s\n", strings.Join(res.Created, ", "), res.Root)
	}
	return exitOK
}

func projectWikiRebuildMeta(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki rebuild-meta", flag.ContinueOnError)
	pf, jsonOut, err := projectWikiParsePrelim("rebuild-meta", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	st, kh, p, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.RebuildMeta(context.Background(), cwd, p)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(res)
	}
	fmt.Fprintf(outW(), "projectwiki rebuild-meta: pages=%d phase=%s run=%s\n", res.Pages, res.RunPhase, res.RunID)
	return exitOK
}

func projectWikiEval(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("projectwiki eval", flag.ContinueOnError)
	pf, jsonOut, err := projectWikiParsePrelim("eval", args, fs)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	// The store is opened per project (storeFor(cfg, p)); Eval reads the
	// opened store directly.
	st, _, _, err := projectWikiHandler(cwd, fl, pf.project)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	defer st.Close()
	rep, err := projectwiki.Eval(context.Background(), st, nil, 10, 1.0)
	if err != nil {
		return projectWikiOpError("%v", err)
	}
	if *jsonOut {
		return projectWikiJSON(rep)
	}
	fmt.Fprintf(outW(), "projectwiki eval: cases=%d recall_at_k=%.3f failed=%v\n", rep.Cases, rep.RecallAtK, rep.Failed)
	return exitOK
}
