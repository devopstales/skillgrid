package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
)

// runSearchEmbeddingStatus handles `skillgrid embedding-status`.
func runSearchEmbeddingStatus(version string, args []string) {
	_ = version
	fs := flag.NewFlagSet("embedding-status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var jsonOut bool
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid embedding-status [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	projectID, err := svc.ResolveProject(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	out, err := svc.CodeEmbeddingStatus(ctx, projectID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	provider, _ := out["provider"].(string)
	model, _ := out["model"].(string)
	dim, _ := out["dimension"].(int)
	embedded, _ := out["embedded_symbols"].(int)
	indexedModel, _ := out["indexed_model"].(string)
	active, _ := out["active"].(bool)
	fmt.Printf("provider: %s\n", provider)
	if active {
		fmt.Printf("model: %s (dim %d)\n", model, dim)
	} else {
		fmt.Printf("model: off\n")
	}
	fmt.Printf("embedded symbols: %d\n", embedded)
	if indexedModel != "" {
		fmt.Printf("indexed model: %s\n", indexedModel)
	}
}

// runSearch handles `skillgrid search QUERY [flags]`.
func runSearch(version string, args []string) {
	_ = version
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		jsonOut      bool
		ftsOnly      bool
		semanticOnly bool
		limit        int
	)
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.BoolVar(&ftsOnly, "fts", false, "select the FTS leg only")
	fs.BoolVar(&semanticOnly, "semantic", false, "select the semantic leg only")
	fs.IntVar(&limit, "limit", 20, "maximum hits")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid search QUERY [--json] [--fts] [--semantic] [--limit N]")
		fmt.Fprintln(fs.Output(), "       skillgrid search affected [--stdin | FILES...] [--base REF] [--depth N] [--filter F] [--json] [--quiet]")
		fmt.Fprintln(fs.Output(), "       skillgrid search rename OLD NEW [--file F] [--uid U] [--kind K] [--apply] [--json]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	// `search affected` / `search rename` are PR commands (010 step 02): the
	// first bare argument selects the mode, not a query. Run BEFORE flag
	// parsing so affected's own flags (--stdin/--base/...) don't leak into
	// the shared search flag set.
	if len(args) >= 1 && args[0] == "affected" {
		runSearchAffected(version, args)
		return
	}
	if len(args) >= 1 && args[0] == "rename" {
		runSearchRename(version, args)
		return
	}
	if len(args) >= 1 && args[0] == "pdg" {
		runSearchPdg(version, args)
		return
	}
	if len(args) >= 1 && args[0] == "taint" {
		runSearchTaint(version, args)
		return
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: search requires exactly one QUERY argument")
		os.Exit(2)
	}
	query := fs.Arg(0)

	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	projectID, err := svc.ResolveProject(".")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var out *service.CodeHybridResult
	if semanticOnly {
		out, err = svc.CodeSemanticSearch(ctx, projectID, query, limit)
	} else {
		out, err = svc.CodeHybridSearch(ctx, projectID, query, limit)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if jsonOut {
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}

	if len(out.Hits) == 0 {
		fmt.Fprintln(os.Stderr, "no matches")
		return
	}
	fmt.Printf("query: %s (legs: %v)\n", out.Query, out.Legs)
	for i, h := range out.Hits {
		sym := ""
		if h.Symbol != "" {
			sym = " [" + h.Symbol + "]"
		}
		prov := ""
		if h.Provenance.FTS && h.Provenance.Signal && h.Provenance.Semantic {
			prov = " [fts+signal+semantic]"
		} else if h.Provenance.FTS && h.Provenance.Semantic {
			prov = " [fts+semantic]"
		} else if h.Provenance.Signal && h.Provenance.Semantic {
			prov = " [signal+semantic]"
		} else if h.Provenance.FTS {
			prov = " [fts]"
		} else if h.Provenance.Signal {
			prov = " [signal]"
		} else if h.Provenance.Semantic {
			prov = " [semantic]"
		}
		fmt.Printf("%2d. %s:%d-%d%s %s (score %.4f)%s\n", i+1, h.Path, h.StartLine, h.EndLine, sym, h.Kind, h.Score, prov)
	}
	for _, w := range out.Warnings {
		fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	}
}
