package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mnemonichttp "github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/http"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/mcp"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/service"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/setup"
)

// runMCP starts the MCP stdio server.
func runMCP(version string, args []string) {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var debug bool
	fs.BoolVar(&debug, "debug", false, "log MCP framing errors to stderr")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid mcp [flags]")
		fmt.Fprintln(fs.Output(), "  Starts the Mnemonic MCP stdio server (mem_*, code_*, web_* tools).")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	_ = version
	_ = debug

	if err := mcp.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "mcp server exited:", err)
	}
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
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir     string
		project string
	)
	fs.StringVar(&dir, "dir", ".", "directory to index")
	fs.StringVar(&project, "project", envOr("SKILLGRID_MNEMONIC_PROJECT", ""), "fixed project identity")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid index [flags]")
		fmt.Fprintln(fs.Output(), "  Run incremental code indexing for a directory.")
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

	stats, err := svc.RunCodeIndex(ctx, dir)
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
	if err := setup.RunSetup(agent, repoRoot, dryRun); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// newMnemonicService builds a Mnemonic service from a data dir (env or default).
func newMnemonicService(dir string) (*service.Service, error) {
	if dir == "" {
		var err error
		dir, err = service.DefaultDataDir()
		if err != nil {
			return nil, err
		}
	}
	return service.New(dir), nil
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
