package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	ctxconfig "github.com/devopstales/skillgrid/mnemonic/internal/ctx/config"
	ctxwrap "github.com/devopstales/skillgrid/mnemonic/internal/ctx/wrap"
)

// runCtxWrap is the `ctx wrap <agent>` subcommand (T07, tier-2 launcher):
// it starts the loopback proxy, injects the agent's base-URL env to point
// at the proxy, backs up the pre-inject config state (first wrap only), and
// launches the agent as a subprocess with the injected env. The agent's
// code is never modified — only its config (env). Re-running is idempotent
// (no double-inject, no double-backup).
//
// Flags: -bind (default from ctx config, 127.0.0.1), -port (default from
// ctx config, 8787), -bin (agent binary, default: the agent name),
// -dir (repository directory for ctx config resolution). Remaining
// positional args are passed through to the agent.
func runCtxWrap(args []string) {
	fs := flag.NewFlagSet("ctx wrap", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	bind := fs.String("bind", "", "proxy bind address (default from ctx config, 127.0.0.1)")
	port := fs.Int("port", 0, "proxy port (default from ctx config, 8787)")
	bin := fs.String("bin", "", "agent binary to launch (default: the agent name)")
	dir := fs.String("dir", ".", "repository directory (for ctx config resolution)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if len(pos) == 0 {
		fmt.Fprintln(os.Stderr, "usage: mnemonic ctx wrap <agent> [agent args...] [-bind <addr>] [-port <n>] [-bin <path>] [-dir <dir>]")
		os.Exit(2)
	}
	agentName := pos[0]
	agentArgs := pos[1:]

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

	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	res, stopProxy, err := ctxwrap.Wrap(ctxwrap.Options{
		Cfg:       cfg,
		AgentName: agentName,
		Binary:    *bin,
		Args:      agentArgs,
		Env:       os.Environ(),
		Port:      p,
		Bind:      b,
		StateDir:  *dir,
		Log:       log,
	}, ctxwrap.DefaultStartProxy)
	if err != nil {
		// Fail-open (ADR-0016): clear error, non-zero exit, no panic.
		// Agent resolution, proxy start, and launch failures all land here.
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if res.BackupPath != "" {
		log.Info("backup written", "agent", agentName, "path", res.BackupPath)
	} else if !res.Injected {
		log.Info("already wrapped, skipping inject+backup", "agent", agentName)
	}
	log.Info("agent launched", "agent", agentName, "proxy", res.ProxyBaseURL, "port", res.Port)

	// Keep the proxy alive for the subprocess lifetime: SIGINT/SIGTERM on
	// the CLI stops the proxy (and the process, via the default signal
	// disposition after stopProxy), the agent exiting stops the proxy, and
	// we exit with the agent's exit code.
	ctx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	go func() {
		<-ctx.Done()
		stopProxy()
	}()

	if err := res.Command.Wait(); err != nil {
		stopProxy()
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "error: agent exited: %v\n", err)
		os.Exit(1)
	}
	stopProxy()
}
