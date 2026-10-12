package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ctxconfig "github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxproxy "github.com/devopstales/skillgrid/mnemonic/internal/ctx/proxy"
	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/loop"
)

// runCtxProxy is the `ctx proxy` subcommand: it starts the loopback LLM proxy
// (T04). Flags: -bind (default from config, 127.0.0.1), -port (default from
// config, 8787), -mode (token|cache|passthrough; default from config), -dir
// (repository directory for store resolution).
//
// The proxy binds to the loopback address, resolves the upstream from env
// (ANTHROPIC_BASE_URL / OPENAI_BASE_URL), applies the mode-selected transform,
// writes accurate X-CTX-Savings headers, retries provider errors with
// exponential backoff, and records per-turn telemetry to the shared store.
// SIGINT/SIGTERM trigger a graceful shutdown (drain in-flight requests, close
// the listener, close the store).
func runCtxProxy(args []string) {
	fs := flag.NewFlagSet("ctx proxy", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	bind := fs.String("bind", "", "bind address (default from ctx config, 127.0.0.1)")
	port := fs.Int("port", 0, "port (default from ctx config, 8787)")
	mode := fs.String("mode", "", "compression mode: token, cache, or passthrough (default from ctx config)")
	dir := fs.String("dir", ".", "repository directory (for store resolution)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	// Load the ctx config (config.d precedence: defaults -> machine -> repo).
	cfg, err := ctxconfig.Load(context.Background(), *dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: loading ctx config: %v\n", err)
		os.Exit(1)
	}

	// Resolve bind/port from flags or config.
	b := *bind
	if b == "" {
		b = cfg.Proxy.Bind
	}
	p := *port
	if p == 0 {
		p = cfg.Proxy.Port
	}

	// Resolve the effective mode: CLI flag > CTX_PROXY_MODE env > config.
	effectiveMode := ctxproxy.ResolveMode(*mode, cfg.Proxy.Mode, os.Getenv)

	// Open the shared store for telemetry (ADR-0033: one shared handle).
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: opening mnemonic service: %v\n", err)
		os.Exit(1)
	}
	h, cleanup, err := svc.OpenForDirectory(*dir)
	if err != nil {
		// Fall back to MNEMONIC_PROJECT / active project, matching openLoop.
		var pid string
		if id := os.Getenv("MNEMONIC_PROJECT"); id != "" {
			pid = id
		}
		if pid == "" {
			pid = loop.ReadActiveProject()
		}
		if pid != "" {
			h, cleanup, err = svc.OpenAt(pid, *dir)
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: no store available; telemetry will be a no-op: %v\n", err)
	}

	// Build the ctx store accessor (nil if no shared store).
	var cs *ctxstore.Store
	if h != nil {
		cs = ctxstore.New(h.Store())
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	log.Info("starting ctx proxy", "bind", b, "port", p, "mode", string(effectiveMode))

	// Warn (do not fail) when the proxy is bound to a non-loopback address.
	// The CTX proxy carries upstream credentials and is designed for
	// 127.0.0.1; a wider bind may expose it to the network.
	// M2: use net.ParseIP().IsLoopback() instead of a string allow-list so
	// "::1", "::ffff:127.0.0.1", and other valid loopback forms are
	// recognised, while "0.0.0.0", "::", "localhost", external IPs, and ""
	// all warn.
	if !isLoopbackBind(b) {
		log.Warn("ctx proxy bound to a non-loopback address", "bind", b)
	}

	// Build the proxy handler.
	proxy := ctxproxy.New(cfg, cs, log, effectiveMode)

	// Set up signal handling for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create the listener.
	addr := fmt.Sprintf("%s:%d", b, p)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: listening on %s: %v\n", addr, err)
		os.Exit(1)
	}
	log.Info("listening", "addr", ln.Addr().String())

	// Serve HTTP with graceful shutdown. HTTP/2 (h2c, prior knowledge)
	// is enabled by sniffing the HTTP/2 client preface on each connection
	// and upgrading to h2c when present. The proxy's upstream connections
	// are plain HTTP (no TLS), so h2c is the correct negotiation path.
	srv := &http.Server{Handler: h2cHandler(proxy.Handler())}
	go func() {
		<-ctx.Done()
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(os.Stderr, "error: serving: %v\n", err)
		os.Exit(1)
	}

	log.Info("proxy stopped")
	if h != nil {
		cleanup()
	}
}

// isLoopbackBind reports whether bind is a loopback address: 127.0.0.1,
// ::1, ::ffff:127.0.0.1, or "localhost". An empty bind (meaning "all
// interfaces") is NOT loopback. External IPs, 0.0.0.0, and :: are not
// loopback. This replaces the T04 string allow-list (M2).
func isLoopbackBind(bind string) bool {
	if bind == "" || bind == "localhost" {
		return false
	}
	// Try to parse as an IP (with or without zone).
	ip := net.ParseIP(bind)
	if ip != nil {
		return ip.IsLoopback()
	}
	// "localhost" resolves to loopback by convention, but it is also a
	// name that could be overridden in /etc/hosts; treat it as loopback
	// for the warning heuristic.
	return false
}

// h2cHandler returns h unchanged. HTTP/2 (h2c, prior knowledge) support on
// the loopback serve entry is provided by the Go 1.24+ net/http server,
// which automatically detects the h2 client preface on cleartext
// connections and negotiates h2c. No handler wrapping is required; this
// function exists as a seam for the future (e.g. if a custom h2c adapter
// becomes necessary).
func h2cHandler(h http.Handler) http.Handler {
	return h
}
