package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/memory"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// cmdEmbedder builds the process embedder from project config (mirrors the
// service's resolveEmbedder).
func cmdEmbedder(cfgRoot string) embedder.Embedder {
	cfg := config.Load(cfgRoot)
	switch cfg.Embedder.Provider {
	case "external":
		return embedder.NewExternal(embedder.ExternalConfig{
			BaseURL:   cfg.Embedder.BaseURL,
			Model:     cfg.Embedder.Model,
			APIKey:    cfg.Embedder.APIKey,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  cmdAsymParams(cfg.Embedder.Indexing),
			Query:     cmdAsymParams(cfg.Embedder.Query),
		})
	case "off", "":
		return nil
	default: // "onnx" is the default
		return embedder.NewOnnx(embedder.OnnxConfig{
			Model:     cfg.Embedder.Model,
			Dimension: cfg.Embedder.Dimension,
			Indexing:  cmdAsymParams(cfg.Embedder.Indexing),
			Query:     cmdAsymParams(cfg.Embedder.Query),
		})
	}
}

func cmdAsymParams(p config.EmbedderParams) embedder.AsymParams {
	return embedder.AsymParams{
		Instructions: p.Instructions,
		InputType:    p.InputType,
		MaxTokens:    p.MaxTokens,
	}
}

// nonDegenerate reports whether v is not all-zeros and contains no NaN/Inf.
func nonDegenerate(v memory.Vector) bool {
	anyNonZero := false
	for _, f := range v.Data {
		if math.IsNaN(float64(f)) || math.IsInf(float64(f), 0) {
			return false
		}
		if f != 0 {
			anyNonZero = true
		}
	}
	return anyNonZero
}

func doctorEmbedRoundTrip(ctx context.Context, emb embedder.Embedder) (dim int, degenerate bool, err error) {
	const probe = "hello world test"
	iv, err := emb.Embed(ctx, probe)
	if err != nil {
		return 0, false, err
	}
	if len(iv.Data) != emb.Dimension() {
		return len(iv.Data), false, fmt.Errorf("indexing dimension %d != configured %d", len(iv.Data), emb.Dimension())
	}
	if !nonDegenerate(iv) {
		return emb.Dimension(), true, nil
	}
	qv, err := emb.EmbedQuery(ctx, probe)
	if err != nil {
		return 0, false, err
	}
	if len(qv.Data) != emb.Dimension() {
		return len(qv.Data), false, fmt.Errorf("query dimension %d != configured %d", len(qv.Data), emb.Dimension())
	}
	if !nonDegenerate(qv) {
		return emb.Dimension(), true, nil
	}
	return emb.Dimension(), false, nil
}

func doctorOnnxModelPath(model string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".skillgrid", "models", model+".onnx")
	}
	return ""
}

func doctorWALMode(st *store.Store) string {
	var mode string
	if err := st.DB.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		return "unknown: " + err.Error()
	}
	return mode
}

// runDoctor handles `skillgrid doctor` — a functional health check of the
// embedder round-trip and the local capability set.
func runDoctor(version string, args []string) {
	_ = version
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var strict bool
	fs.BoolVar(&strict, "strict", false, "exit non-zero on a redaction or freshness violation (CI-usable)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid doctor [--strict]")
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

	fmt.Println("skillgrid doctor")
	fmt.Printf("  project: %s\n", projectID)

	emb := cmdEmbedder(".")
	if emb == nil {
		fmt.Println("  embedder: off")
	} else {
		fmt.Printf("  embedder: %s (%s)\n", cfgProviderOf("."), emb.Model())
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		dim, degenerate, rtErr := doctorEmbedRoundTrip(ctx, emb)
		if rtErr != nil {
			fmt.Printf("  embed round-trip: error: %v\n", rtErr)
		} else {
			nd := "yes"
			if degenerate {
				nd = "no"
			}
			fmt.Println("  embed round-trip:")
			fmt.Printf("    indexing: dim=%d non-degenerate=%s\n", dim, nd)
			fmt.Printf("    query:    dim=%d non-degenerate=%s\n", dim, nd)
		}
	}

	fmt.Println("  capabilities:")
	fmt.Println("    gotreesitter: pure-Go, CGo-free")
	model := config.Load(".").Embedder.Model
	modelPath := doctorOnnxModelPath(model)
	onnxState := "absent"
	if modelPath != "" {
		if fi, statErr := os.Stat(modelPath); statErr == nil && fi.Mode().IsRegular() {
			onnxState = "present"
		}
	}
	fmt.Printf("    onnx model: %s (%s)\n", onnxState, model)

	walMode := "unavailable"
	st, stErr := store.Open(dataDirForDoctor(dataDir), projectID)
	if stErr == nil {
		walMode = doctorWALMode(st)
	}
	fmt.Printf("    wal: %s\n", walMode)
	fmt.Println("    cgo: free (modernc.org/sqlite + gotreesitter)")

	// doctor --strict (01.4): report the redaction + freshness state and exit
	// non-zero when a redaction or freshness violation exists (CI-usable).
	if strict && st != nil {
		defer st.Close()
		rep := runDoctorStrictChecks(st.DB, defaultStrictMaxAge)
		fmt.Println()
		printStrictReport(rep)
		if !rep.Clean() {
			os.Exit(1)
		}
		return
	}
	if st != nil {
		_ = st.Close()
	}
}

func dataDirForDoctor(dataDir string) string {
	if dataDir != "" {
		return dataDir
	}
	if d, err := service.DefaultDataDir(); err == nil {
		return d
	}
	return ""
}

func cfgProviderOf(cfgRoot string) string {
	p := config.Load(cfgRoot).Embedder.Provider
	if p == "" {
		return "onnx"
	}
	return p
}
