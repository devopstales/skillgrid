// mcp.go — the mcp command (spec 005-transport-cli §2): the MCP stdio
// server entrypoint. It builds the shared handler (the same wiring the
// CLI uses) and the MCP server over it (the profile-filtered tool set),
// then blocks on the stdio loop until the client disconnects. The
// autosync manager starts when sync.auto=true (TICKET-17): it debounces
// the save commit's onWrite hook, holds the SQLite-backed lease (one
// worker per data dir), and runs push→pull off the write path.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/mcp"
)

// runMCP runs `mnemonic mcp`: the stdio server entrypoint.
func runMCP(fl config.Flags, args []string) int {
	fs := flag.NewFlagSet("mcp", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var profile string
	fs.StringVar(&profile, "profile", string(mcp.Agent), "tool profile: agent|all")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	cwd, _ := os.Getwd()
	h, st, err := openRuntime(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	p := mcp.Profile(profile)
	switch p {
	case mcp.Agent, mcp.All:
	default:
		fmt.Fprintf(outW(), "error: unknown profile %q (agent|all)\n", profile)
		return exitUsage
	}
	srv := mcp.NewServer(h, p)
	defer srv.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The autosync manager (TICKET-17): when sync.auto=true, it debounces
	// the save commit's onWrite hook (the store's committed-txn seam),
	// holds the SQLite-backed lease (one worker per data dir), and runs
	// push→pull in its own goroutine — network I/O never blocks a save.
	// The stdio ctx (SIGINT/SIGTERM) is the manager's lifetime.
	if h.Config().Sync.Auto {
		if err := startAutosync(ctx, h, st); err != nil {
			fmt.Fprintf(outW(), "error: autosync: %s\n", err.Error())
			return exitErr
		}
	}
	if err := srv.Start(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(outW(), "error: mcp server: %s\n", err.Error())
		return exitErr
	}
	return exitOK
}

// startAutosync wires the autosync manager (TICKET-17) over the runtime:
// it builds the transport from the resolved config (the test seam swaps in
// a fake), builds the SyncService (the journal apply seam), replaces the
// store's committed-transaction hook with the manager's NotifyDirty (the
// onWrite hook — the commit path's only cross-goroutine touch), and starts
// the run loop on the given ctx (Stop on cancel releases the lease and
// closes the transport).
func startAutosync(ctx context.Context, h *handler.Handler, st store.Store) error {
	cfg := h.Config()
	tr, err := openTransport(cfg)
	if err != nil {
		return err
	}
	svc := memory.NewSyncService(st, cfg)
	a := memory.NewAutosync(svc, tr, cfg)
	st.SetTxnHook(a.NotifyDirty)
	a.Start(ctx)
	// the manager stops with the process (the lease releases on exit, the
	// transport closes): a cancelled ctx (SIGINT/SIGTERM) tears down the
	// run loop before the stdio loop's return.
	go func() {
		<-ctx.Done()
		a.Stop()
		_ = tr.Close()
	}()
	return nil
}

// runServe runs `serve`: the deprecated no-op alias (the v2 HTTP
// dashboard is dropped — the pointer to mcp/CLI is the retained surface).
func runServe(fl config.Flags, args []string) int {
	// The flag set accepts the globals (the command is a no-op; the
	// parse is here so a mistyped flag is a loud error, not a silent
	// one).
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Parse(args)
	fmt.Fprintln(outW(), "mnemonic serve: v2 drops the HTTP dashboard. Use `mnemonic mcp` (the stdio server) or the CLI (`mnemonic mem ...`).")
	return exitOK
}
