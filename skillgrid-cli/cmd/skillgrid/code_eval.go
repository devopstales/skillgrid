package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/eval"
)

// evalCorpus is one named corpus for the eval runner: its name (the pooled axis
// — "self" / a second-language name) and its root (the git repo / file tree it
// is derived from).
type evalCorpus struct {
	Name string
	Root string
}

// readGitHistory reads a repo's commit log (hash, subject, changed files) in
// reverse-chronological order. It shells out to git (the cmd layer may use
// os/exec; the eval package stays pure-Go and just consumes these rows).
func readGitHistory(root string) ([]eval.Commit, error) {
	cmd := exec.Command("git", "-C", root,
		"log", "--no-color", "--pretty=format:%H%x1f%s", "--name-only")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log in %s: %w", root, err)
	}
	// Format: <hash>\x1f<subject>\n<file>\n<file>\n ... per commit, newest first.
	// Split on newlines; a line with \x1f starts a new commit.
	lines := strings.Split(string(out), "\n")
	var commits []eval.Commit
	for _, line := range lines {
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "\x1f"); idx >= 0 {
			commits = append(commits, eval.Commit{
				Hash:    line[:idx],
				Subject: strings.TrimSpace(line[idx+1:]),
			})
			continue
		}
		if len(commits) > 0 {
			commits[len(commits)-1].Files = append(commits[len(commits)-1].Files, strings.TrimSpace(line))
		}
	}
	return commits, nil
}

// loadCorpusFiles returns the corpus's file set at HEAD (path → content),
// excluding benchmark scaffolding (01.18) and common non-code files.
func loadCorpusFiles(root string) map[string]string {
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if eval.ExcludedFromCorpus(rel) {
			return nil
		}
		// Only text-y code files (the eval grades code retrieval).
		if !isCodeFile(rel) {
			return nil
		}
		if b, rerr := os.ReadFile(p); rerr == nil {
			out[rel] = string(b)
		}
		return nil
	})
	return out
}

// isCodeFile reports whether a path is a code file the eval grades.
func isCodeFile(path string) bool {
	lower := strings.ToLower(path)
	for _, ext := range []string{".go", ".py", ".js", ".ts", ".java", ".rb", ".rs", ".c", ".cpp", ".h", ".hpp", ".cs", ".kt", ".swift"} {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// parseEvalCorpora parses the repeated --corpus flags. Each value is `name` or
// `name=path`. `self` (no path) resolves to the cwd's git root; any other name
// REQUIRES a path that exists — an unknown name or missing path is rejected
// clearly (no invented run).
func parseEvalCorpora(values []string) ([]evalCorpus, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("eval: at least one --corpus is required (e.g. --corpus self or --corpus second=/path/to/repo)")
	}
	cwd, _ := os.Getwd()
	var out []evalCorpus
	seen := map[string]bool{}
	for _, v := range values {
		name, path, hasPath := strings.Cut(v, "=")
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("eval: empty corpus name in %q", v)
		}
		if seen[name] {
			return nil, fmt.Errorf("eval: duplicate corpus name %q", name)
		}
		seen[name] = true
		switch {
		case name == "self" && !hasPath:
			root, err := gitRootOf(cwd)
			if err != nil {
				return nil, fmt.Errorf("eval: cannot resolve the self corpus (cwd is not in a git repo): %w", err)
			}
			out = append(out, evalCorpus{Name: name, Root: root})
		case hasPath:
			path = strings.TrimSpace(path)
			if fi, err := os.Stat(path); err != nil || !fi.IsDir() {
				return nil, fmt.Errorf("eval: corpus %q path %q does not exist or is not a directory", name, path)
			}
			out = append(out, evalCorpus{Name: name, Root: path})
		default:
			return nil, fmt.Errorf("eval: unknown corpus %q — use `self` or `name=path` (a path to a corpus root)", name)
		}
	}
	return out, nil
}

// gitRootOf returns the git root of dir (or an error when not in a repo).
func gitRootOf(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// runEval handles `skillgrid eval --corpus <name> [--corpus <name>] [--json]`.
// It derives a leak-free query set from each corpus's git history, builds ONE
// shared index per corpus, runs the ablation (baseline + the shipped ranker
// config), and prints the report with significance. Unknown corpora are
// rejected clearly (01.21).
func runEval(version string, args []string) {
	_ = version
	fs := flag.NewFlagSet("eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var corpora []string
	var jsonOut bool
	fs.Var((*stringSliceFlag)(&corpora), "corpus", "a corpus: `self` or `name=path` (repeatable)")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `usage: skillgrid eval --corpus <self|name=path> [--corpus ...] [--json]

Runs the retrieval-eval ablation over the given corpora. The query set is
derived from each corpus's git history (leak-free: merges/reverts/releases/
bumps/formatting/changelog/benchmark commits are dropped). One shared index per
corpus is built so deltas measure ranking only. The report carries the baseline
plus the shipped ranker config with a seeded paired bootstrap 95% CI and a
permutation p-value vs the baseline.

Examples:
  skillgrid eval --corpus self
  skillgrid eval --corpus self --corpus second=/path/to/other-repo`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	corps, err := parseEvalCorpora(corpora)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	fileSets := make(map[string]map[string]string)
	querySets := make(map[string][]*eval.QuerySet)
	for _, c := range corps {
		commits, herr := readGitHistory(c.Root)
		if herr != nil {
			fmt.Fprintf(os.Stderr, "error: corpus %q: %v\n", c.Name, herr)
			os.Exit(1)
		}
		files := loadCorpusFiles(c.Root)
		fileSets[c.Name] = files
		// Restrict each query to the gradeable (code-only) corpus: a commit
		// that touches both code and docs/config files grades only the code
		// files, and a docs/config-only commit is not a retrieval signal.
		// Restricting the expected files to the corpus keeps validate_queries
		// honest (it should only flag a genuinely stale CODE expectation, not
		// a docs commit's .md path that was never in the code corpus).
		present := make(map[string]bool, len(files))
		for p := range files {
			present[p] = true
		}
		var kept []*eval.QuerySet
		for _, q := range eval.DeriveQuerySet(commits) {
			var inCorpus []string
			for _, f := range q.ExpectedFiles {
				if present[f] {
					inCorpus = append(inCorpus, f)
				}
			}
			if len(inCorpus) == 0 {
				continue // docs/config-only commit — not a retrieval signal
			}
			kept = append(kept, &eval.QuerySet{Subject: q.Subject, ExpectedFiles: inCorpus})
		}
		querySets[c.Name] = kept
	}

	res, err := eval.Run(ctx, eval.RunConfig{
		Corpora: fileSets,
		Queries: querySets,
		Variants: []eval.Variant{
			{Name: "baseline", Rank: identityRank, Baseline: true},
			// The shipped variant applies the step-01 retrieval-quality factors
			// (a lexical projection of hybrid/rerank.go's ApplyRerankTable) so
			// the harness measures a REAL candidate vs the 005 baseline — not a
			// vacuous identity==baseline delta.
			{Name: "shipped", Rank: shippedRank},
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if jsonOut {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Fprintln(os.Stdout, string(b))
		return
	}
	printEvalReport(res, querySets)
}

// identityRank is the 005 baseline behavior for the CLI: a plain lexical/path
// ordering of the corpus (no rerank factors). This is the reference the
// shipped variant is measured against.
func identityRank(q *eval.QuerySet, idx *eval.CorpusIndex) []string {
	return idx.Paths()
}

// shippedRank is the lexical PROJECTION of the step-01 retrieval-quality
// factors onto the corpus's file paths. It is a faithful lexical analogue of
// hybrid/rerank.go's ApplyRerankTable (exact-symbol, path-match,
// documentation/generated/test penalties, source-over-prose), reordering the
// corpus paths for the query by the same named factors the production rerank
// table uses — but operating purely on paths, because the eval CorpusIndex only
// carries paths (it has no sqlite store, symbol table, or graph degree).
//
// This is NOT a call into hybrid.Search (which is retriever-driven and needs the
// store). The full retriever-driven ranker lives in hybrid/rank.go and is
// exercised separately by the mcp/service tests; the eval's "shipped" row is
// the lexical projection of its factors, not a claim that the full hybrid
// ranker is measured here.
//
// The key property: shippedRank DIFFERS from identityRank on realistic queries
// (it reorders by the named factors), so the harness measures a genuine,
// non-trivial delta. Deterministic (no randomness).
func shippedRank(q *eval.QuerySet, idx *eval.CorpusIndex) []string {
	paths := idx.Paths()
	qLower := strings.ToLower(q.Subject)
	qTerms := strings.Fields(qLower)
	isTestQuery := strings.Contains(qLower, "test")

	type scored struct {
		path  string
		delta float64
	}
	var out []scored
	for _, p := range paths {
		out = append(out, scored{path: p, delta: lexicalProjectionDelta(p, qLower, qTerms, isTestQuery)})
	}
	// Score descending, path ascending for ties (deterministic, matches
	// hybrid.RerankTable's bounded additive model).
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].delta != out[j].delta {
			return out[i].delta > out[j].delta
		}
		return out[i].path < out[j].path
	})
	res := make([]string, len(out))
	for i, s := range out {
		res[i] = s.path
	}
	return res
}

// lexicalProjectionDelta scores one corpus path for a query using the same
// named, bounded factors as hybrid.RerankTable, projected onto the path only
// (no symbol/degree, which the corpus index does not carry). The factor names
// and signs mirror hybrid/rerank.go:
//
//	exact-symbol + (the filename stem contains the full query, or a long query
//	term — a lexical proxy for the symbol the query names)
//	path-match +  (a query term >=3 chars appears in the path)
//	source-over-prose + (code, not prose/config)
//	documentation − (a docs/config path)
//	generated-vendor − (a vendor/generated path)
//	test-on-non-test − (a test path for a non-test query)
func lexicalProjectionDelta(path, qLower string, qTerms []string, isTestQuery bool) float64 {
	delta := 0.0
	lower := strings.ToLower(path)
	// The filename stem (no extension) as a lexical proxy for the symbol.
	leaf := lower
	if i := strings.LastIndex(leaf, "/"); i >= 0 {
		leaf = leaf[i+1:]
	}
	stem := leaf
	if i := strings.LastIndex(leaf, "."); i >= 0 {
		stem = leaf[:i]
	}

	// exact-symbol (proxy): the full query (>=3 chars) appears in the stem, or
	// any long query term (>=4 chars) does. Bounded at 0.30 like the table.
	exact := 0.0
	if len(qLower) >= 3 && strings.Contains(stem, qLower) {
		exact = 0.30
	} else {
		for _, t := range qTerms {
			if len(t) >= 4 && strings.Contains(stem, t) {
				if t != "test" {
					exact = 0.30
					break
				}
			}
		}
	}
	delta += exact

	// path-match: a query term >=3 chars appears in the full path (not the
	// stem, so directory locality counts). Bounded at 0.10.
	for _, t := range qTerms {
		if len(t) >= 3 && strings.Contains(lower, t) {
			delta += 0.10
			break
		}
	}

	// source-over-prose: code beats prose/config.
	if isProsePath(path) {
		delta += -0.10 // documentation
	} else {
		delta += 0.05 // source-over-prose
	}

	// generated/vendor.
	if isVendorOrGeneratedPath(path) {
		delta += -0.15
	}

	// test-on-non-test.
	if isTestFilePath(path) && !isTestQuery {
		delta += -0.10
	}

	return delta
}

// isProsePath / isVendorOrGeneratedPath / isTestFilePath are path-only
// projections of the hybrid package's private classifiers, so the cmd layer
// (which cannot call unexported hybrid helpers) applies the same signals.
// They mirror hybrid.isProsePath / isVendorOrGenerated and the isTestPath used
// in mcp/tools_code.go.
func isProsePath(path string) bool {
	lower := strings.ToLower(path)
	for _, m := range []string{"/docs/", "/doc/", "readme", "changelog", ".md", ".rst", ".txt", ".yaml", ".yml", ".json", ".toml"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func isVendorOrGeneratedPath(path string) bool {
	lower := strings.ToLower(path)
	if strings.Contains(lower, "vendor/") || strings.Contains(lower, "/vendor/") {
		return true
	}
	for _, m := range []string{"_generated.", ".pb.", ".gen.", "generated.go", "zz_"} {
		if strings.Contains(lower, m) {
			return true
		}
	}
	return false
}

func isTestFilePath(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "_test.go") || strings.HasSuffix(lower, ".test.js") || strings.Contains(lower, "test_")
}

// stringSliceFlag is a repeatable string flag for --corpus.
type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// printEvalReport renders the eval ablation report (human table).
func printEvalReport(res *eval.Result, querySets map[string][]*eval.QuerySet) {
	fmt.Printf("skillgrid eval — corpora: %s (queries: %d)\n",
		strings.Join(res.Corpora, ", "), totalQueries(querySets))
	fmt.Println()
	if res.Baseline != nil {
		fmt.Printf("baseline : recall@5=%.3f recall@10=%.3f MRR=%.3f nDCG@10=%.3f dup%%=%.3f tokens=%d\n",
			res.Baseline.Recall5, res.Baseline.Recall10, res.Baseline.MRR, res.Baseline.NDGC10, res.Baseline.DupPercent, res.Baseline.Tokens)
	}
	for _, row := range res.Rows {
		d := res.Decisions[row.Name]
		fmt.Printf("%-10s: recall@5=%.3f (Δ%.3f) MRR=%.3f (Δ%.3f) CI[%.3f,%.3f] p=%.4f → %s\n",
			row.Name, row.Recall5, row.DeltaRecall5, row.MRR, row.DeltaMRR,
			row.CILower, row.CIUpper, row.PValue, d.Decision)
	}
	// One-index-per-corpus invariant (01.19).
	for name, n := range res.IndexInstances {
		if n != 1 {
			fmt.Printf("  note: corpus %q was indexed %d times (expected 1)\n", name, n)
		}
	}
}

func totalQueries(qs map[string][]*eval.QuerySet) int {
	n := 0
	for _, v := range qs {
		n += len(v)
	}
	return n
}
