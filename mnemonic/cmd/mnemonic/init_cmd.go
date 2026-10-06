package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/extract"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

type initResult struct {
	BootFile       string
	IndexingConfig string // path to generated indexing.yaml, "" if skipped
	Architecture   string // path to .skillgrid/ARCHITECTURE.md, "" if skipped
	Preamble       string // "written" | "kept" | "forced"
	Sentinel       string // "upserted"
	Indexed        int
	Ingested       int
	Skipped        []string
	Errors         []string
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
	if res.IndexingConfig != "" {
		fmt.Fprintf(w, "indexing_config: %s\n", res.IndexingConfig)
	}
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

// manifestLangs maps a manifest filename to the languages it implies. A
// project can have multiple manifests (e.g. go.mod + package.json for a
// polyglot repo); all matching languages are included.
var manifestLangs = map[string][]string{
	"go.mod":        {"go"},
	"package.json":  {"typescript", "tsx", "javascript"},
	"pyproject.toml": {"python"},
	"setup.py":      {"python"},
	"requirements.txt": {"python"},
	"Cargo.toml":    {"rust"},
	"pom.xml":       {"java"},
	"build.gradle":  {"java"},
	"build.gradle.kts": {"java", "kotlin"},
	"*.csproj":      {"c_sharp"},
	"composer.json": {"php"},
	"Gemfile":       {"ruby"},
	"Package.swift": {"swift"},
	"build.sbt":     {"scala"},
	"pubspec.yaml":  {"dart"},
}

// detectProjectLanguages returns the set of supported languages present in
// dir, based on manifest files at the project root.
func detectProjectLanguages(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	var langs []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if l, ok := manifestLangs[name]; ok {
			for _, lang := range l {
				if !seen[lang] {
					seen[lang] = true
					langs = append(langs, lang)
				}
			}
			continue
		}
		// Pattern match for *.csproj
		if strings.HasSuffix(name, ".csproj") && !seen["c_sharp"] {
			seen["c_sharp"] = true
			langs = append(langs, "c_sharp")
		}
	}
	// Always include markdown for docs
	if !seen["markdown"] {
		langs = append(langs, "markdown")
	}
	return langs
}

// writeIndexingConfig generates .skillgrid/config.d/indexing.yaml with
// language-appropriate include globs. Only writes when the file does not
// already exist, so operator-customized configs are never clobbered.
// Returns the written path or "" if it was skipped.
func writeIndexingConfig(dir string) string {
	path := filepath.Join(dir, ".skillgrid", "config.d", "indexing.yaml")
	if _, err := os.Stat(path); err == nil {
		return ""
	}

	langs := detectProjectLanguages(dir)
	if len(langs) == 0 {
		return ""
	}

	globs := extract.IncludeGlobs(langs)
	if len(globs) == 0 {
		return ""
	}

	// Markdown has no extToLang entry; add it explicitly.
	globs = append(globs, "**/*.md")
	// De-dupe and sort for stable output.
	seen := make(map[string]bool)
	var final []string
	for _, g := range globs {
		if !seen[g] {
			seen[g] = true
			final = append(final, g)
		}
	}

	var b strings.Builder
	b.WriteString("mnemonic:\n")
	b.WriteString("  include:\n")
	for _, g := range final {
		fmt.Fprintf(&b, "    - %q\n", g)
	}
	b.WriteString("  exclude:\n")
	for _, e := range defaultExcludes {
		fmt.Fprintf(&b, "    - %q\n", e)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ""
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return ""
	}
	return path
}

var defaultExcludes = []string{
	"**/node_modules/**",
	"**/.git/**",
	"**/dist/**",
	"**/.skillgrid/**",
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

	archPath, archErr := writeArchitectureFile(dir)
	if archErr != nil {
		return initResult{}, fmt.Errorf("write ARCHITECTURE.md: %w", archErr)
	}

	cfgPath := writeIndexingConfig(dir)

	res := initResult{
		BootFile: bootPath,
		Preamble: preambleState,
		Sentinel: "upserted",
	}
	if cfgPath != "" {
		res.IndexingConfig = cfgPath
	}
	if archPath != "" {
		res.Architecture = archPath
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
