// kb.go — the kb command group (the KB engine's CLI surface, 011
// deliverable 9 / KB-T6). Exit codes: 0 ok · 1 drift under --strict ·
// 2 usage/operational.
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
)

func runKb(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("kb: a subcommand is required (generate|index|reindex|validate|query|status|bootstrap|lint|cite-source|eval)")
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "generate":
		return kbGenerate(fl, cwd, rest)
	case "index":
		return kbIndex(fl, cwd, rest)
	case "reindex":
		return kbReindex(fl, cwd, rest)
	case "validate":
		return kbValidate(fl, cwd, rest)
	case "query":
		return kbQuery(fl, cwd, rest)
	case "status":
		return kbStatus(fl, cwd, rest)
	case "bootstrap":
		return kbBootstrap(fl, cwd, rest)
	case "lint":
		return kbLint(fl, cwd, rest)
	case "cite-source":
		return kbCiteSource(fl, cwd, rest)
	case "eval":
		return kbEval(fl, cwd, rest)
	default:
		return errExit("kb: unknown subcommand %q (generate|index|reindex|validate|query|status|bootstrap|lint|cite-source|eval)", sub)
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

// kbStore opens the store for the resolved project (explicit --project
// beats the detector chain).
func kbStore(cwd string, fl config.Flags, explicit string) (store.Store, string, error) {
	cfg, err := config.Load(cwd, fl)
	if err != nil {
		return nil, "", err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	p, err := det.Resolve(cwd, explicit, gitToplevel)
	if err != nil {
		return nil, "", err
	}
	cfg.Project.Name = p
	st, err := storeFor(cfg, p)
	if err != nil {
		return nil, "", err
	}
	return st, p, nil
}

// kbRoot returns the knowledge vault root for this project (spec
// 001-structure: the vault lives under a single sentinel dir,
// .skillgrid/knowledge/).
func kbRoot(cwd string) string {
	return filepath.Join(cwd, ".skillgrid", "knowledge")
}

// kbProjectFlag registers the project selector on fs. Read pf.project only
// AFTER fs.Parse (the parse fills the struct in place — the mem commands'
// convention; a pre-parse read is "").
func kbProjectFlag(fs *flag.FlagSet) *projectFlag {
	var pf projectFlag
	pf.add(fs)
	return &pf
}

func kbGenerate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb generate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	full := fs.Bool("full", false, "full re-derive (ignore the processed set)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	facts, err := kb.AllFacts(context.Background(), st, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	res, err := kb.Generate(context.Background(), st, kbRoot(cwd), p, *full, nil, facts)
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

func kbIndex(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.Index(context.Background(), st, kbRoot(cwd), p)
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
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.Reindex(context.Background(), st, kbRoot(cwd), p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb reindex: scanned=%d changed=%d\n", res.Scanned, res.Changed)
	return exitOK
}

func kbValidate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	strict := fs.Bool("strict", false, "fail (exit 1) on any drifted page")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	root := kbRoot(cwd)
	rels := fs.Args()
	if len(rels) == 0 {
		// Default scope: every grounded page in the vault (006 §5).
		rels, err = kb.GroundedPages(root)
		if err != nil {
			return kbOpError("%v", err)
		}
	}
	ctx := context.Background()
	results := make([]*kb.ValidateResult, 0, len(rels))
	drifted := 0
	for _, rel := range rels {
		res, err := kb.Validate(ctx, st, root, rel, p)
		if err != nil {
			return kbOpError("%v", err)
		}
		if !res.OK {
			drifted++
		}
		results = append(results, res)
	}
	if *jsonOut {
		return kbJSON(map[string]any{"results": results, "checked": len(results), "drifted": drifted})
	}
	for _, res := range results {
		fmt.Fprintf(outW(), "kb validate: %s %s (%s)\n", res.RelPath, map[bool]string{true: "ok", false: "drift"}[res.OK], res.Reason)
	}
	if len(results) > 1 {
		fmt.Fprintf(outW(), "kb validate: %d checked, %d drifted\n", len(results), drifted)
	}
	if *strict && drifted > 0 {
		return exitErr
	}
	return exitOK
}

func kbQuery(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb query", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	var typ string
	var limit int
	groundedOnly := fs.Bool("grounded-only", false, "restrict to grounded pages")
	fs.StringVar(&typ, "type", "", "filter by page type")
	fs.IntVar(&limit, "limit", 10, "max hits")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	query := strings.TrimSpace(strings.Join(fs.Args(), " "))
	if query == "" {
		return errExit("kb query: a query is required")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.Query(context.Background(), st, query, kb.QueryFilters{
		Project: p, Type: typ, Limit: limit, GroundedOnly: *groundedOnly,
	})
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(map[string]any{"results": res, "count": len(res)})
	}
	for _, hit := range res {
		fmt.Fprintf(outW(), "%s | %s | %s (score %.3f)\n", hit.RelPath, hit.Title, hit.Type, hit.Score)
	}
	fmt.Fprintf(outW(), "kb query: %d hit(s)\n", len(res))
	return exitOK
}

func kbStatus(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.Status(context.Background(), st, p)
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

func kbBootstrap(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb bootstrap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.Bootstrap(context.Background(), st, kbRoot(cwd), kb.Options{Project: p})
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

func kbLint(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb lint", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	fix := fs.Bool("fix", false, "auto-repair the safe findings (missing titles)")
	strict := fs.Bool("strict", false, "fail (exit 1) when issues remain")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	ctx := context.Background()
	// Sync the index first: lint is store-level (it never reads the vault),
	// so the findings reflect the current vault only after the index caught
	// up. The sync runs before any fix so a body drift cannot clobber the
	// backfilled titles afterward.
	if _, err := kb.Index(ctx, st, kbRoot(cwd), p); err != nil {
		return kbOpError("%v", err)
	}
	issues, err := kb.Lint(ctx, st, p)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *fix {
		// The safe fix set is the missing_title finding (Fix backfills the
		// kb_pages title from the rel_path; it never touches the file).
		var rels []string
		for _, iss := range issues {
			if iss.Code == "missing_title" {
				rels = append(rels, iss.RelPath)
			}
		}
		if len(rels) > 0 {
			if _, err := kb.Fix(ctx, st, p, rels); err != nil {
				return kbOpError("%v", err)
			}
			issues, err = kb.Lint(ctx, st, p)
			if err != nil {
				return kbOpError("%v", err)
			}
		}
	}
	if *jsonOut {
		return kbJSON(map[string]any{"issues": issues, "count": len(issues)})
	}
	if len(issues) == 0 {
		fmt.Fprintln(outW(), "kb lint: clean")
	} else {
		for _, iss := range issues {
			fmt.Fprintf(outW(), "kb lint: %s %s %s\n", iss.RelPath, iss.Code, iss.Message)
		}
		fmt.Fprintf(outW(), "kb lint: %d issue(s)\n", len(issues))
	}
	if *strict && len(issues) > 0 {
		return exitErr
	}
	return exitOK
}

func kbCiteSource(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb cite-source", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	ref := fs.Arg(0)
	if ref == "" {
		return errExit("kb cite-source: a page rel_path is required")
	}
	st, p, err := kbStore(cwd, fl, pf.project)
	if err != nil {
		return kbOpError("%v", err)
	}
	defer st.Close()
	res, err := kb.CiteSource(context.Background(), st, p, ref)
	if err != nil {
		return kbOpError("%v", err)
	}
	if *jsonOut {
		return kbJSON(res)
	}
	fmt.Fprintf(outW(), "kb cite-source: %s\n", res.Source)
	if res.Content != "" {
		fmt.Fprintln(outW(), res.Content)
	}
	return exitOK
}

func kbEval(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("kb eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	pf := kbProjectFlag(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if pf.all {
		return errExit("kb: --all fan-out is not yet wired (the cross-project store is pending); use --project <name>")
	}
	// The store is opened per project (storeFor(cfg, p)); Eval reads the
	// opened store directly.
	st, _, err := kbStore(cwd, fl, pf.project)
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
