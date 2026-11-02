// mcp.go — the mcp command (spec 005-transport-cli §2): the MCP stdio
// server entrypoint. It builds the shared handler (the same wiring the
// CLI uses) and the MCP server over it (the profile-filtered tool set),
// then blocks on the stdio loop until the client disconnects. The
// autosync manager starts when sync.auto=true (TICKET-17 — the start/
// stop hook lands with the manager; the dispatch is complete).

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
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
	// ponytail: the autosync manager (TICKET-17) starts here when
	// h.Config().Sync.Auto is true — the start/stop hook is a one-line
	// change when the manager lands; the server is the entrypoint.
	// A cancelled ctx (SIGINT/SIGTERM) stops the stdio loop cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := srv.Start(ctx); err != nil && ctx.Err() == nil {
		fmt.Fprintf(outW(), "error: mcp server: %s\n", err.Error())
		return exitErr
	}
	return exitOK
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
