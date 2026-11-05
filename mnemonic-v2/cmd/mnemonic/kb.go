// kb.go — the kb command group (008-transport §6, the 011 openwiki port):
// a thin wrapper over the shared KbHandler (the same wiring the MCP server
// uses — the CLI and MCP never diverge, the K6 byte-identical seam).
// Exit codes: 0 ok · 1 drift under --strict · 2 usage/operational.
//
// kb_eval is CLI-only (a harness command — not in the 12-tool D1 surface).
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/kb"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

func runKb(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("kb: a subcommand is required (query|read|search|generate|validate|lint|status|cite-source|index|reindex|bootstrap|rebuild-meta|eval)")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "query":
		return kbQuery(fl, cwd, rest)
	case "read":
		return kbRead(fl, cwd, rest)
	case "search":
		return kbSearch(fl, cwd, rest)
	case "generate":
		return kbGenerate(fl, cwd, rest)
	case "validate":
		return kbValidate(fl, cwd, rest)
	case "lint":
		return kbLint(fl, cwd, rest)
	case "status":
		return kbStatus(fl, cwd, rest)
	case "cite-source", "cite":
		return kbCiteSource(fl, cwd, rest)
	case "index":
		return kbIndex(fl, cwd, rest)
	case "reindex":
		return kbReindex(fl, cwd, rest)
	case "bootstrap":
		return kbBootstrap(fl, cwd, rest)
	case "rebuild-meta":
		return kbRebuildMeta(fl, cwd, rest)
	case "eval":
		return kbEval(fl, cwd, rest)
	default:
		return errExit("kb: unknown subcommand %q (query|read|search|generate|validate|lint|status|cite-source|index|reindex|bootstrap|rebuild-meta|eval)", sub)
	}
}

// kbOpError prints an operational error and returns the usage/operational
// exit code (011 §3.9: 2 — distinct from 1, which is --strict drift only).
func kbOpError(format string, args ...any) int {
	fmt.Fprintf(errW(), "error: "+format+"\n", args...)
	return exitUsage
}

// kbJSON prints the --json output: the envelope wire shape (snake_case,
// the degraded flag with its default reason) — the same marshal the MCP
// tools use, the K6 byte-identical contract (finish does the same for the
// mem commands). res must marshal to a JSON object (wrap slices in a map).
func kbJSON(res any) int {
	degraded, reason := envelope.ExtractDegraded(res)
	env, err := envelope.WithDegraded(res, degraded, reason)
	if err != nil {
		return kbOpError("envelope: %v", err)
	}
	fmt.Fprintln(outW(), mustMarshal(env))
	return exitOK
}

// kbRoot returns the knowledge vault root for this project (spec
// 001-structure: the vault lives under a single sentinel dir,
// .skillgrid/knowledge/).
func kbRoot(cwd string) string {
	return filepath.Join(cwd, ".skillgrid", "knowledge")
}

// kbHandler opens the store for the resolved project and builds the shared
// KbHandler over it (008-transport §6: the CLI is a thin wrapper over the
// same handlers — the K6 seam). The roots are set per call (006 §2: the
// anchor paths are working-tree-relative, so the repo root is the git
// toplevel of cwd, the vault root the sentinel dir under it). The resolved
// project is returned for the callers that assert it (the write tools'
// expected_project).
func kbHandler(cwd string, fl config.Flags, explicit string) (store.Store, *handler.KbHandler, string, error) {
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
	kh := handler.NewKbHandler(st, cfg, &gitDetector{det: det})
	kh.SetRoots(kbRoot(cwd), repo)
	return st, kh, p, nil
}

// kbProjectFlag registers the project selector on fs. Read pf.project only
// AFTER fs.Parse (the parse fills the struct in place — the mem commands'
// convention; a pre-parse read is "").
func kbProjectFlag(fs *flag.FlagSet) *projectFlag {
	var pf projectFlag
	pf.add(fs)
	return &pf
}

// kbParsePrelim parses the shared flags (project selector + --json) and
// enforces the --all guard (the cross-project store is pending).
func kbParsePrelim(name string, args []string, fs *flag.FlagSet) (*projectFlag, *bool, error) {
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return nil, nil, fmt.Errorf("kb %s: %w", name, err)
	}
	if pf.all {
		return nil, nil, fmt.Errorf("kb %s: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>", name)
	}
	return pf, jsonOut, nil
}

func kbQuery(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb query", flag.ContinueOnError)
	var typ, matchMode string
	var limit int
	groundedOnly := fs.Bool("grounded-only", false, "restrict to grounded pages")
	fs.StringVar(&typ, "type", "", "filter by page type")
	fs.StringVar(&matchMode, "match-mode", "", "term matching: any|all")
	fs.IntVar(&limit, "limit", 10, "max hits")
	pf, jsonOut, err := kbParsePrelim("query", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return errExit("kb query: a query is required")
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Query(context.Background(), cwd, p, handler.QueryRequest{
		Query: query, Type: typ, Limit: limit, MatchMode: matchMode, GroundedOnly: *groundedOnly,
	})
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	for _, hit := range res.Results {
		fmt.Fprintf(outW(), "%s | %s | %s (score %.3f)\n", hit.RelPath, hit.Title, hit.Type, hit.Score)
	}
	fmt.Fprintf(outW(), "kb query: %d hit(s)\n", res.Count)
	return exitOK
}

func kbRead(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb read", flag.ContinueOnError)
	var relPath, section string
	backlinks := fs.Bool("backlinks", false, "list the pages that link here")
	fs.StringVar(&relPath, "path", "", "the page rel_path")
	fs.StringVar(&section, "section", "", "a single ## section name (empty = full body)")
	pf, jsonOut, err := kbParsePrelim("read", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	if relPath == "" {
		return errExit("kb read: --path <rel_path> is required")
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Read(context.Background(), cwd, relPath, section, p, *backlinks)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
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
func pinCheckLabel(res *kb.ReadResult) string {
	if res.PinCheck == nil {
		return ""
	}
	return ", pin: " + res.PinCheck.Status
}

func kbSearch(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb search", flag.ContinueOnError)
	var limit int
	fs.IntVar(&limit, "limit", 25, "max hits")
	pf, jsonOut, err := kbParsePrelim("search", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return errExit("kb search: a query is required")
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Search(context.Background(), cwd, query, p, limit)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	for _, hit := range res.Results {
		fmt.Fprintf(outW(), "%s | %s | %s\n", hit.RelPath, hit.Title, hit.Snippet)
	}
	fmt.Fprintf(outW(), "kb search: %d hit(s)\n", res.Count)
	return exitOK
}

func kbGenerate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb generate", flag.ContinueOnError)
	full := fs.Bool("full", false, "full re-derive (ignore the processed set)")
	pf, jsonOut, err := kbParsePrelim("generate", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Generate(context.Background(), cwd, pf.project, p, *full)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb generate: changed=%d noop=%v degraded=%v run=%s", res.Changed, res.Noop, res.Degraded, res.RunID)
	for _, rel := range res.RelPaths {
		fmt.Fprintf(outW(), "\n  wrote %s", rel)
	}
	fmt.Fprintln(outW())
	return exitOK
}

func kbValidate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb validate", flag.ContinueOnError)
	strict := fs.Bool("strict", false, "fail (exit 1) on any drifted page")
	pf, jsonOut, err := kbParsePrelim("validate", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	rels := fs.Args()
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Validate(context.Background(), cwd, p, rels, *strict)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	for _, v := range res.Results {
		fmt.Fprintf(outW(), "kb validate: %s %s (%s)\n", v.RelPath, map[bool]string{true: "ok", false: "drift"}[v.OK], v.Reason)
	}
	if res.Checked > 1 {
		fmt.Fprintf(outW(), "kb validate: %d checked, %d drifted\n", res.Checked, res.Drifted)
	}
	if *strict && !res.OK {
		return exitErr
	}
	return exitOK
}

func kbLint(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb lint", flag.ContinueOnError)
	fix := fs.Bool("fix", false, "auto-repair the safe findings (missing titles)")
	strict := fs.Bool("strict", false, "fail (exit 1) when issues remain")
	pf, jsonOut, err := kbParsePrelim("lint", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Lint(context.Background(), cwd, p, *fix)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	if len(res.Issues) == 0 {
		fmt.Fprintln(outW(), "kb lint: clean")
	} else {
		for _, iss := range res.Issues {
			fmt.Fprintf(outW(), "kb lint: %s %s %s\n", iss.RelPath, iss.Code, iss.Message)
		}
		fmt.Fprintf(outW(), "kb lint: %d issue(s)", res.Count)
		if res.Fixed > 0 {
			fmt.Fprintf(outW(), " (fixed %d)", res.Fixed)
		}
		fmt.Fprintln(outW())
	}
	if *strict && len(res.Issues) > 0 {
		return exitErr
	}
	return exitOK
}

func kbStatus(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb status", flag.ContinueOnError)
	pf, jsonOut, err := kbParsePrelim("status", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Status(context.Background(), cwd, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb status: project=%s indexed=%d phase=%s baselines=%d events=%d\n",
		res.Project, res.Indexed, res.Phase, res.Baselines, res.Events)
	return exitOK
}

func kbCiteSource(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb cite-source", flag.ContinueOnError)
	var name, version, url, commit string
	fs.StringVar(&name, "name", "", "the external source name (required)")
	fs.StringVar(&version, "cite-version", "", "the cited version/tag (--version is the global version flag)")
	fs.StringVar(&url, "url", "", "the source URL")
	fs.StringVar(&commit, "commit", "", "the source commit (when known)")
	pf, jsonOut, err := kbParsePrelim("cite-source", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	relPath := fs.Arg(0)
	if relPath == "" {
		return errExit("kb cite-source: a page rel_path is required")
	}
	if name == "" {
		return errExit("kb cite-source: --name <external name> is required")
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.CiteSource(context.Background(), cwd, pf.project, p, relPath, name, version, url, commit)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	if res.AlreadyCited {
		fmt.Fprintf(outW(), "kb cite-source: %s already cited on %s\n", res.Source, relPath)
	} else {
		fmt.Fprintf(outW(), "kb cite-source: %s recorded on %s (fingerprint %s)\n", res.Source, relPath, res.BuildFingerprint)
	}
	return exitOK
}

func kbIndex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb index", flag.ContinueOnError)
	pf, jsonOut, err := kbParsePrelim("index", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Index(context.Background(), cwd, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb index: scanned=%d changed=%d deleted=%d\n", res.Scanned, res.Changed, res.Deleted)
	return exitOK
}

func kbReindex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb reindex", flag.ContinueOnError)
	pf, jsonOut, err := kbParsePrelim("reindex", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Reindex(context.Background(), cwd, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb reindex: scanned=%d changed=%d\n", res.Scanned, res.Changed)
	return exitOK
}

func kbBootstrap(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb bootstrap", flag.ContinueOnError)
	pf, _, err := kbParsePrelim("bootstrap", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.Bootstrap(context.Background(), cwd, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if len(res.Created) == 0 {
		fmt.Fprintf(outW(), "kb bootstrap: vault present at %s (no-op)\n", res.Root)
	} else {
		fmt.Fprintf(outW(), "kb bootstrap: created %s at %s\n", strings.Join(res.Created, ", "), res.Root)
	}
	return exitOK
}

func kbRebuildMeta(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb rebuild-meta", flag.ContinueOnError)
	pf, jsonOut, err := kbParsePrelim("rebuild-meta", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	st, kh, p, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kh.RebuildMeta(context.Background(), cwd, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb rebuild-meta: pages=%d phase=%s run=%s\n", res.Pages, res.RunPhase, res.RunID)
	return exitOK
}

func kbEval(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb eval", flag.ContinueOnError)
	pf, jsonOut, err := kbParsePrelim("eval", args, fs)
	if err != nil {
		return kbOpError("%v", err)
	}
	// The store is opened per project (storeFor(cfg, p)); Eval reads the
	// opened store directly.
	st, _, _, err := kbHandler(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	rep, err := kb.Eval(context.Background(), st, nil, 10, 1.0)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(rep)
	}
	fmt.Fprintf(outW(), "kb eval: cases=%d recall_at_k=%.3f failed=%v\n", rep.Cases, rep.RecallAtK, rep.Failed)
	return exitOK
}
