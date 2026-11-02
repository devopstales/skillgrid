// sync.go — the sync command group (spec 005-transport-cli §2 + §4): the
// full-sync-in-memory surface. `sync status` reads the per-target
// lifecycle table (the sync_state read surface, spec 006-sync §7);
// `sync enroll` toggles a project's enrollment (the gate the save's
// ENQUEUE step reads). `push`/`pull`/`export`/`import` are delegation
// stubs that become real when the sync transport (TICKET-16) lands — the
// dispatch is complete, the delegation is a one-line change.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// runSync dispatches the sync subcommands.
func runSync(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("sync: a subcommand is required (status|push|pull|export|import|enroll)")
	}
	switch args[0] {
	case "status":
		return syncStatus(fl, cwd, args[1:])
	case "enroll":
		return syncEnroll(fl, cwd, args[1:])
	case "push":
		fmt.Fprintln(outW(), "mnemonic sync push delegates to the sync transport (TICKET-16, not yet in this build).")
		return exitOK
	case "pull":
		fmt.Fprintln(outW(), "mnemonic sync pull delegates to the sync transport (TICKET-16, not yet in this build).")
		return exitOK
	case "export":
		fmt.Fprintln(outW(), "mnemonic sync export delegates to the sync transport (TICKET-16, not yet in this build).")
		return exitOK
	case "import":
		fmt.Fprintln(outW(), "mnemonic sync import delegates to the sync transport (TICKET-16, not yet in this build).")
		return exitOK
	default:
		if strings.HasPrefix(args[0], "-") {
			return errExit("sync: unknown flag %q before a subcommand", args[0])
		}
		return errExit("sync: unknown subcommand %q", args[0])
	}
}

// syncStatus runs `sync status`: the per-target lifecycle table (spec
// 006-sync §7 — the sync_state read surface).
func syncStatus(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync status", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	status, err := memory.NewSyncService(st, h.Config()).Status(context.Background())
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"targets": status}})
	}
	// Human: the per-target lifecycle table (the header always renders,
	// even when no target has synced yet).
	fmt.Fprintln(outW(), "target\tlifecycle\tlast_enqueued\tlast_acked\tlast_pulled\tbackoff_until")
	for target, ts := range status {
		fmt.Fprintf(outW(), "%s\t%s\t%d\t%d\t%d\t%s\n",
			target, ts.Lifecycle, ts.LastEnqueued, ts.LastAcked, ts.LastPulled, ts.BackoffUntil)
	}
	return exitOK
}

// syncEnroll runs `sync enroll <project> [--off]`: toggles the project's
// enrollment (the gate the save's ENQUEUE step reads — only enrolled
// projects generate remote journal entries).
func syncEnroll(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync enroll", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var off bool
	fs.BoolVar(&off, "off", false, "un-enroll the project")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	project := ""
	if len(fs.Args()) > 0 {
		project = fs.Args()[0]
	}
	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	// The enrollment is keyed by the resolved project: a positional
	// argument selects it (the store is opened for it), an absent one
	// resolves through the detector's chain (the handler's call-time
	// resolution — explicit > env > git > basename).
	resolved := project
	if resolved == "" {
		r, rerr := h.CurrentProject("", "")
		if rerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: rerr})
		}
		resolved = r
	}
	herr := memory.NewSyncService(st, h.Config()).Enroll(context.Background(), resolved, !off)
	if herr != nil {
		return finish(fs, *jsonOut, dispatchResult{err: herr})
	}
	verb := "enrolled"
	if off {
		verb = "unenrolled"
	}
	res := map[string]any{"project": resolved, verb: true}
	return finish(fs, *jsonOut, dispatchResult{result: res})
}
