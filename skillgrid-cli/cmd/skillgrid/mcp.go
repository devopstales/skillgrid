package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/codeindex"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/config"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/setup"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/store"
)

// runMCP starts the MCP stdio server.
func runMCP(version string, args []string) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var debug bool
	var noWatch bool
	fs.BoolVar(&debug, "debug", false, "log MCP framing errors to stderr")
	fs.BoolVar(&noWatch, "no-watch", codeindex.WatchDisabled(), "disable the auto-sync watcher (manual index)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid mcp [flags]")
		fmt.Fprintln(fs.Output(), "  Starts the Mnemonic MCP stdio server (mem_*, code_*, web_* tools).")
		fmt.Fprintln(fs.Output(), "  The auto-sync watcher + fingerprint gate keep the index fresh; SKILLGRID_NO_WATCH=1 disables the watcher.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	_ = version
	_ = debug
	if noWatch {
		_ = os.Setenv(codeindex.NoWatchEnv, "1")
	}

	// Wire the response path (auto-reopen + fingerprint gate + staleness
	// banner) for the CWD project, and start the comfort-layer watcher unless
	// disabled. The watcher is the comfort layer; the fingerprint gate is the
	// correctness backstop (works with the watcher off).
	svc, err := newMnemonicService(envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""))
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	startAutoSync(svc)

	if err := mcp.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "mcp server exited:", err)
	}
}

// startAutoSync wires the response path (auto-reopen + fingerprint gate +
// staleness banner) for the CWD project and starts the comfort-layer watcher
// (unless SKILLGRID_NO_WATCH=1). It resolves the project + index config,
// attaches the watcher's pending set to the banner, and starts the watcher
// with a debounce-clamped window. The re-index is structural-only + uses the
// warm embedder for the search leg (the gate itself never touches the embedder).
func startAutoSync(svc *service.Service) {
	cwd, err := os.Getwd()
	if err != nil {
		return
	}
	projectID, err := svc.ResolveProject(cwd)
	if err != nil {
		return
	}
	cfg := config.Load(cwd)
	idxCfg := codeindex.Config{
		Include: cfg.Include, Exclude: cfg.Exclude,
		ChunkLines: cfg.ChunkLines, ChunkOverlap: cfg.ChunkOverlap, MaxFileSize: cfg.MaxFileSize,
	}
	// Wire the response path (auto-reopen ~5s + fingerprint gate + banner).
	mcp.InitFreshness(svc, projectID, cwd, idxCfg, codeindex.DefaultReopenWindow)

	if codeindex.WatchDisabled() {
		// Watcher disabled: manual index (the fingerprint gate is the backstop).
		return
	}
	// Comfort-layer watcher: debounced incremental re-index on source-file
	// create/modify/delete. The re-index is structural-only (no embedder).
	var w *codeindex.Watcher
	w = codeindex.NewWatcher(cwd, codeindex.WatchConfig{
		Debounced: codeindex.DefaultDebounce,
		Include:   cfg.Include,
		Exclude:   cfg.Exclude,
		OnSync: func(ctx context.Context, files []string) error {
			// Mark the burst pending (feeds the banner until it syncs), then
			// run the structural re-index (the (size,mtime)+hash guard makes it
			// incremental), then clear the pending set.
			if w != nil {
				for _, f := range files {
					w.MarkPending(f)
				}
			}
			_, _ = svc.ReindexStructural(ctx, cwd, idxCfg)
			if w != nil {
				w.ClearPending()
			}
			return nil
		},
	})
	// Attach the watcher's pending set to the staleness banner.
	mcp.AttachWatcher(w)
	if err := w.Start(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "note: auto-sync watcher off: %v (fingerprint gate is the backstop)\n", err)
		return
	}
	fmt.Fprintln(os.Stderr, "auto-sync watcher: on (debounce", codeindex.DefaultDebounce, ")")
}

// runServe starts the Mnemonic HTTP API.
func runServe(version string, args []string) {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		addr string
		dir  string
		bind string
	)
	fs.StringVar(&addr, "port", envOr("SKILLGRID_MNEMONIC_PORT", "7438"), "listen port (default 7438)")
	fs.StringVar(&bind, "bind", "127.0.0.1", "bind address")
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "data directory (default ~/.skillgrid/mnemonic)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid serve [flags]")
		fmt.Fprintln(fs.Output(), "  Starts the Mnemonic HTTP API (default http://127.0.0.1:7438).")
		fmt.Fprintln(fs.Output(), "  Opens the data viewer at / and Swagger UI at /swagger-ui.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	svc, err := newMnemonicService(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	_ = version

	mux := mnemonichttp.NewServer(svc).Handler()
	listenAddr := net.JoinHostPort(bind, addr)
	if bind == "" {
		listenAddr = ":" + addr
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: listen:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "skillgrid serve listening on http://%s\n", listenAddr)

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- serveHTTP(ctx, ln, mux)
	}()
	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "shutting down")
	case err := <-srvErr:
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
}

func serveHTTP(ctx context.Context, ln net.Listener, handler http.Handler) (err error) {
	srv := &http.Server{Handler: handler}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.Serve(ln) }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		select {
		case e := <-errCh:
			return e
		case <-shutdownCtx.Done():
			return nil
		}
	case e := <-errCh:
		return e
	}
}

// runIndex runs incremental code indexing for a directory.
func runIndex(version string, args []string) {
	// `skillgrid index status` is the index-status subcommand (pending-sync
	// section + watcher state). It is dispatched before the index flag-set
	// parses so `status` is not treated as a directory.
	if len(args) >= 1 && args[0] == "status" {
		runIndexStatus(version, args[1:])
		return
	}
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir     string
		project string
		pdg     bool
		lsp     bool
	)
	fs.StringVar(&dir, "dir", ".", "directory to index")
	fs.StringVar(&project, "project", envOr("SKILLGRID_MNEMONIC_PROJECT", ""), "fixed project identity")
	fs.BoolVar(&pdg, "pdg", false, "opt-in per-function CFG + PDG pass (fills cfg_blocks/cfg_edges/pdg_edges)")
	fs.BoolVar(&lsp, "lsp", false, "opt-in LSP edge tier (adds LSP_RESOLVED member-call edges; best-effort, static index unchanged when no server)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid index [flags]")
		fmt.Fprintln(fs.Output(), "  Run incremental code indexing for a directory.")
		fmt.Fprintln(fs.Output(), "  Subcommands: status (index health + pending sync + watcher state)")
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
	_ = version
	_ = project

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	stats, err := svc.RunCodeIndexPDG(ctx, dir, pdg, lsp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "indexed: %d files, %d chunks (+%d skipped, -%d deleted)\n",
		stats.FilesIndexed, stats.ChunksAdded, stats.FilesSkipped, stats.FilesDeleted)
}

// runSetup installs agent plugins (opencode|kilocode|cursor).
func runSetup(version string, args []string) {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		agent    string
		repoRoot string
		dryRun   bool
	)
	fs.StringVar(&agent, "agent", "", "agent to configure: opencode, kilocode, cursor")
	fs.StringVar(&repoRoot, "repo-root", "", "skillgrid repo root (auto-detected)")
	fs.BoolVar(&dryRun, "dry-run", false, "print planned changes without writing")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), `usage: skillgrid setup <opencode|kilocode|cursor> [flags]
  (equivalently: skillgrid setup --agent <agent> [flags])
  Install Mnemonic plugins for an AI agent.`)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	_ = version

	// Positional agent name: first non-flag argument.
	positional := fs.Args()
	if agent == "" && len(positional) > 0 {
		agent = positional[0]
	}

	if agent == "" {
		fmt.Fprintln(os.Stderr, "error: missing agent (opencode, kilocode, or cursor)")
		os.Exit(2)
	}
	mcpEntries, err := setup.LoadMCPConfig(repoRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := setup.RunSetup(agent, repoRoot, mcpEntries, dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// newMnemonicService builds a Mnemonic service from a data dir (env or default).
func newMnemonicService(dir string) (*service.Service, error) {
	if cliService != nil {
		return cliService, nil
	}
	if dir == "" {
		var err error
		dir, err = service.DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	s := service.New(dir)
	cliService = s
	return s, nil
}

// cliService pins the service built by the first newMnemonicService call so
// every later handler in the same CLI invocation shares that data dir (the
// env var may be set after the first construction, e.g. by a test harness).
var cliService *service.Service

// runIndexStatus implements `skillgrid index status`: index health (file/chunk
// counts + last-indexed) plus the `### Pending sync:` section (the set of
// source files edited but not yet indexed) and the watcher state (enabled /
// disabled = manual index). With no watcher running the pending set is empty
// (the fingerprint gate is the correctness backstop for stale trees).
func runIndexStatus(version string, args []string) {
	fs := flag.NewFlagSet("index status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir     string
		project string
		jsonOut bool
	)
	fs.StringVar(&dir, "dir", ".", "directory the index tracks")
	fs.StringVar(&project, "project", envOr("SKILLGRID_MNEMONIC_PROJECT", ""), "fixed project identity")
	fs.BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid index status [--dir D] [--project P] [--json]")
		fmt.Fprintln(fs.Output(), "  Show index health + pending sync + watcher state.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	_ = version
	_ = project
	_ = dir

	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Resolve the project + its store (best-effort: the CWD project).
	projectID, err := svc.ResolveProject(".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "note: could not resolve project: %v\n", err)
	}
	var status codeindex.Status
	if st, closeFn, err := openProjectStore(svc, projectID); err == nil {
		defer closeFn()
		status, _ = codeindex.GetStatus(st)
	}

	// Watcher state + pending set. With no watcher running (CLI status call)
	// the pending set is empty and enabled reflects SKILLGRID_NO_WATCH.
	ws := codeindex.GetWatchStatus(nil)
	if jsonOut {
		printIndexStatusJSON(os.Stdout, status, ws)
		return
	}
	printIndexStatus(os.Stdout, status, ws)
}

// printIndexStatus renders the human index-status report (with the
// `### Pending sync:` section + watcher state) to w.
func printIndexStatus(w io.Writer, status codeindex.Status, ws codeindex.WatchStatus) {
	stale := status.FileCount == 0 || status.LastIndexed == ""
	fmt.Fprintf(w, "files: %d, chunks: %d, last_indexed: %s, stale: %v\n",
		status.FileCount, status.ChunkCount, orDash(status.LastIndexed), stale)
	if codeindex.WatchDisabled() {
		fmt.Fprintln(w, "watcher: disabled (manual index; the fingerprint gate is the backstop)")
	} else {
		fmt.Fprintln(w, "watcher: enabled")
	}
	fmt.Fprintln(w, "### Pending sync:")
	if len(ws.Pending) == 0 {
		fmt.Fprintln(w, "  (none)")
	} else {
		for _, p := range ws.Pending {
			fmt.Fprintf(w, "  %s\n", p)
		}
	}
}

// printIndexStatusJSON renders the machine-readable index-status report to w.
func printIndexStatusJSON(w io.Writer, status codeindex.Status, ws codeindex.WatchStatus) {
	stale := status.FileCount == 0 || status.LastIndexed == ""
	out := map[string]any{
		"file_count":     status.FileCount,
		"chunk_count":    status.ChunkCount,
		"last_indexed":   status.LastIndexed,
		"stale":          stale,
		"watcher":        ws.Enabled,
		"watch_disabled": codeindex.WatchDisabled(),
		"pending_sync":   ws.Pending,
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Fprintln(w, string(b))
}

// openProjectStore opens a project store (best-effort) returning the store +
// a close func.
func openProjectStore(svc *service.Service, projectID string) (*store.Store, func(), error) {
	if projectID == "" {
		return nil, func() {}, fmt.Errorf("no project resolved")
	}
	h, cleanup, err := svc.Open(projectID)
	if err != nil {
		return nil, cleanup, err
	}
	return h.Store(), cleanup, nil
}

// orDash returns s or "-" when empty (status display).
func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}


