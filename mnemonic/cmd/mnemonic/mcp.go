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

	"github.com/devopstales/skillgrid/mnemonic/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic/internal/projectwiki"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/handler"
	"github.com/devopstales/skillgrid/mnemonic/internal/transport/mcp"
)

// runMCP runs `mnemonic <family> mcp`: the stdio server entrypoint. The
// family selects the single-purpose server — identity, profile-filtered
// tool set, and the family-gated registrations (the RegisterCodeTools /
// RegisterProjectWikiTools seams are no-ops on the non-matching family,
// so `mnemonic memory mcp` serves mem_* only, `code mcp` code_* only,
// `projectwiki mcp` projectwiki_* only).
func runMCP(fl config.Flags, args []string, family mcp.Family) int {
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
	srv := mcp.NewServer(h, p, family)
	defer srv.Close()
	// The code_* surface (B3 — the registration seam): register the code
	// tools (the profile selects the set; the seam is a no-op unless this
	// is the code server). The root is the git toplevel of cwd (the index
	// walk's root); the onSync seam is the Index driver (the TICKET-06
	// watch loop's OnSyncFunc signature).
	ch := codeHandlerFor(h.Config(), st)
	root := cwd
	if gitRoot, err := gitToplevel(cwd); err == nil && gitRoot != "" {
		root = gitRoot
	}
	ch.SetRoot(root)
	srv.RegisterCodeTools(ch)
	// The projectwiki_* surface (008-transport §3 — the registration seam):
	// register the ProjectWiki tools (the profile selects the set; the seam
	// is a no-op unless this is the projectwiki server). The vault root is
	// the sentinel dir under cwd (001-structure);
	// the repo root is the git toplevel of cwd (006 §2 — the anchor paths
	// are working-tree-relative, so the pin-check runs against it).
	kh := handler.NewProjectWikiHandler(st, h.Config(), &gitDetector{det: project.Detector{EnvVar: h.Config().Project.EnvVar}})
	kh.SetRoots(projectwiki.Root(cwd), root)
	srv.RegisterProjectWikiTools(kh)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// The autosync manager (TICKET-17): when sync.auto=true, it debounces
	// the save commit's onWrite hook (the store's committed-txn seam),
	// holds the SQLite-backed lease (one worker per data dir), and runs
	// push→pull in its own goroutine — network I/O never blocks a save.
	// The stdio ctx (SIGINT/SIGTERM) is the manager's lifetime. A startup
	// failure is NOT fatal to the server: autosync is a feature, the MCP
	// loop must still serve (a warning to stderr, the store keeps serving).
	var auto *memory.Autosync
	var autoTr memory.MutationTransport
	if h.Config().Sync.Auto {
		var err error
		auto, autoTr, err = startAutosync(ctx, h, st)
		if err != nil {
			fmt.Fprintf(outW(), "warning: autosync disabled: %s\n", err.Error())
		}
	}
	if err := srv.Start(ctx); err != nil && ctx.Err() == nil {
		// stop the autosync manager and close its transport BEFORE the
		// store closes (the deferred st.Close): the run loop reads the
		// store, and the transport's connection is the manager's own.
		if auto != nil {
			auto.Stop()
			_ = autoTr.Close()
		}
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
// closes the transport). It returns the started manager + transport so the
// caller can stop them on an error path that skips the ctx teardown.
//
// The lease leg is SQLite-only (the sync_state lease rows use the SQLite
// dialect's raw-SQL shape): a non-SQLite dialect refuses to start (a
// warning, not a fatal — the MCP server still serves) until the PG lease
// leg is ported (sync.go documents the PG upgrade path).
func startAutosync(ctx context.Context, h *handler.Handler, st store.Store) (*memory.Autosync, memory.MutationTransport, error) {
	cfg := h.Config()
	if cfg.Datasource.Dialect != store.SQLite {
		return nil, nil, fmt.Errorf("PG dialect not supported yet (SQLite only)")
	}
	tr, err := openTransport(cfg)
	if err != nil {
		return nil, nil, err
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
	return a, tr, nil
}
