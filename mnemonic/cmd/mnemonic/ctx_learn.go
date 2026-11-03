package main

import (
	"flag"
	"fmt"
	"os"

	ctxstore "github.com/devopstales/skillgrid/mnemonic/internal/ctx/store"
	"github.com/devopstales/skillgrid/mnemonic/internal/ctx/traffic"
	"github.com/devopstales/skillgrid/mnemonic/internal/loop"
)

// runCtxLearn is the `ctx learn` subcommand group: the human-gate for traffic
// memories (T13, plan §18 M23). The learner (T12) only emits candidates; a
// human promotes one with `ctx learn apply <id>` (→ status "applied", so it
// enters the compression context) or demotes it with `ctx learn revoke <id>`
// (→ status "revoked", so it leaves). Only existing ids can be promoted — a
// non-existent id is a clear error (plan §18.7 "No automatic promotion").
//
// Store resolution mirrors runCtxProxy (ADR-0033: one shared handle via
// ctxstore.New(h.Store())). A nil/missing store is a fail-open warning with a
// non-zero exit (the id cannot be found without a store), and a not-found id is
// a clear human-gate error (the ONE exception to the proxy's never-500 rule —
// it is a user action, not a request path).
func runCtxLearn(args []string) {
	if len(args) == 0 {
		ctxLearnUsage()
		os.Exit(2)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "apply", "revoke":
		runCtxLearnAction(verb, rest)
	case "help", "-h", "--help":
		ctxLearnUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown learn verb %q (want apply or revoke)\n", verb)
		ctxLearnUsage()
		os.Exit(2)
	}
}

func ctxLearnUsage() {
	w := os.Stderr
	fmt.Fprintln(w, "usage: mnemonic ctx learn <apply|revoke> <memory-id> [-dir <repo>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  apply <id>   Promote an existing traffic-memory candidate to applied (active).")
	fmt.Fprintln(w, "  revoke <id>  Demote an applied traffic memory to revoked (reverted).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Only existing ids can be promoted; a non-existent id is a clear error.")
	fmt.Fprintln(w, "The -dir flag resolves the shared store (ADR-0033).")
}

// runCtxLearnAction resolves the shared store, runs the named gate action, and
// prints the outcome. On success it prints the id + the transition and exits 0;
// on not-found (or no store) it prints a clear error to stderr and exits 1.
func runCtxLearnAction(verb string, args []string) {
	fs := flag.NewFlagSet("ctx learn "+verb, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dir := fs.String("dir", ".", "repository directory (for store resolution)")
	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if len(pos) == 0 {
		fmt.Fprintf(os.Stderr, "error: ctx learn %s requires a memory id\n", verb)
		ctxLearnUsage()
		os.Exit(2)
	}
	id := pos[0]

	cs, cleanup := openCtxStoreForLearn(*dir)
	if cs == nil {
		// Fail-open (ADR-0016): without a store the id cannot be found. The
		// proxy's request path degrades to no context; the CLI surfaces the
		// missing store as a clear, non-zero exit.
		fmt.Fprintf(os.Stderr, "warning: no store available; traffic memory %q cannot be %s'd\n", id, verb)
		os.Exit(1)
	}
	defer cleanup()

	g := traffic.NewGate(cs)
	if verb == "apply" {
		if err := g.Apply(id); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "%s applied\n", id)
		return
	}
	// revoke
	if err := g.Revoke(id); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "%s revoked\n", id)
}

// openCtxStoreForLearn resolves the shared ctx store the way runCtxProxy does
// (newMnemonicService + OpenForDirectory / OpenAt fallback + ctxstore.New). It
// returns (nil, no-op) — and prints a warning — when no store can be opened, so
// the caller can fail open (ADR-0016). The returned cleanup closes the shared
// handle (safe to call even when cs is nil).
func openCtxStoreForLearn(dir string) (*ctxstore.Store, func()) {
	dataDir := envOr("SKILLGRID_MNEMONIC_DATA_DIR", "")
	svc, err := newMnemonicService(dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: opening mnemonic service: %v\n", err)
		return nil, func() {}
	}
	h, cleanup, err := svc.OpenForDirectory(dir)
	if err != nil {
		var pid string
		if id := os.Getenv("MNEMONIC_PROJECT"); id != "" {
			pid = id
		}
		if pid == "" {
			pid = loop.ReadActiveProject()
		}
		if pid != "" {
			h, cleanup, err = svc.OpenAt(pid, dir)
		}
	}
	if err != nil || h == nil {
		fmt.Fprintf(os.Stderr, "warning: no store available for ctx learn: %v\n", err)
		return nil, func() {}
	}
	return ctxstore.New(h.Store()), cleanup
}
