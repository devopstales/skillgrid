// sync.go — the sync command group (spec 005-transport-cli §2 + §4): the
// full-sync-in-memory surface. `sync status` reads the per-target
// lifecycle table (the sync_state read surface, spec 006-sync §7);
// `sync enroll` toggles a project's enrollment (the gate the save's
// ENQUEUE step reads). `push`/`pull`/`export`/`import` are wired to the
// sync transport (TICKET-16): push/pull run one cycle over the PG
// transport, export/import run the chunk codec.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
)

// testTransportFor is the transport-construction seam: production builds
// the PG transport from the resolved config's DSN; the tests swap in a
// fake (the CLI wiring is under test without a PG server). It is a
// function of the transport config so the test can install a fixed fake.
var testTransportFor = func(cfg *memory.TransportConfig) (memory.MutationTransport, error) {
	return memory.NewPGTransport(cfg.DSN, cfg.PushLimit, cfg.MaxPushBytes)
}

// openTransport builds the MutationTransport for the resolved config (the
// sync peer's PG DSN). The DSN check is a no-op when the test seam is
// installed (the fake does not need a real DSN); otherwise a blank DSN is
// an actionable error (the sync peer is the team's Postgres — it must be
// configured for push/pull).
func openTransport(cfg config.Config) (memory.MutationTransport, error) {
	tc := memory.TransportConfig{
		DSN:          cfg.Datasource.DSN,
		PushLimit:    cfg.Sync.PushLimit,
		MaxPushBytes: cfg.Sync.MaxPushBytes,
	}
	return testTransportFor(&tc)
}

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
		return syncPush(fl, cwd, args[1:])
	case "pull":
		return syncPull(fl, cwd, args[1:])
	case "export":
		return syncExport(fl, cwd, args[1:])
	case "import":
		return syncImport(fl, cwd, args[1:])
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

// syncPush runs `sync push`: one push cycle — the local pending outbox is
// read, pushed to the peer in batches at the cap, and acked locally. It
// reports the pushed count (and "sync-paused" when the peer's project is
// paused).
func syncPush(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync push", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var project string
	fs.StringVar(&project, "project", "", "the project to push")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	tr, err := openTransport(h.Config())
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	defer tr.Close()
	resolved := project
	if resolved == "" {
		r, rerr := h.CurrentProject("", "")
		if rerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: rerr})
		}
		resolved = r
	}
	// push the outbox in batches (the cap is enforced at the transport
	// boundary; a full batch loops until the outbox drains — the transport's
	// ack advances the local cursor, so the next read is past it).
	limit := h.Config().Sync.PushLimit
	if limit <= 0 {
		limit = memory.MaxBatch
	}
	pushed := 0
	for {
		batch, cerr := st.ListPendingMutations(context.Background(), 0, limit)
		if cerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: cerr})
		}
		if len(batch) == 0 {
			break
		}
		if _, perr := memory.Push(context.Background(), st, tr, resolved, batch); perr != nil {
			if errors.Is(perr, memory.ErrPaused) {
				if *jsonOut {
					return finish(fs, true, dispatchResult{result: map[string]any{"project": resolved, "status": "sync-paused"}})
				}
				fmt.Fprintln(outW(), "sync-paused")
				return exitOK
			}
			return finish(fs, *jsonOut, dispatchResult{err: perr})
		}
		pushed += len(batch)
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"project": resolved, "pushed": pushed}})
	}
	fmt.Fprintf(outW(), "pushed %d\n", pushed)
	return exitOK
}

// syncPull runs `sync pull`: one pull cycle — the peer's mutations since
// the local cursor are fetched and applied through the store (INV-4), the
// deferred queue is replayed. It reports the applied count.
func syncPull(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync pull", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var project string
	fs.StringVar(&project, "project", "", "the project to pull")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	tr, err := openTransport(h.Config())
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	defer tr.Close()
	resolved := project
	if resolved == "" {
		r, rerr := h.CurrentProject("", "")
		if rerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: rerr})
		}
		resolved = r
	}
	// the pull cursor is the local last_pulled_seq.
	since := 0
	if n, e := st.(interface {
		ProbeInt(string, ...any) (int, error)
	}).ProbeInt(`SELECT COALESCE(MAX(last_pulled_seq),0) FROM sync_state WHERE target_key='cloud'`); e == nil {
		since = n
	}
	applied, err := memory.PullAndApply(context.Background(), st, tr, resolved, int64(since))
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"project": resolved, "applied": applied}})
	}
	fmt.Fprintf(outW(), "applied %d\n", applied)
	return exitOK
}

// syncExport runs `sync export --dir <dir>`: the project's entities are
// written to a chunk archive (manifest.json + chunks/<id>.jsonl.gz).
func syncExport(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync export", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var project, dir string
	fs.StringVar(&project, "project", "", "the project to export")
	fs.StringVar(&dir, "dir", "", "the archive directory")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if dir == "" {
		return errExit("sync export: --dir is required")
	}
	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	resolved := project
	if resolved == "" {
		r, rerr := h.CurrentProject("", "")
		if rerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: rerr})
		}
		resolved = r
	}
	man, err := memory.Export(context.Background(), st, resolved, dir)
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"project": resolved, "chunks": len(man.Chunks), "dir": dir}})
	}
	fmt.Fprintf(outW(), "exported %d chunks to %s\n", len(man.Chunks), dir)
	return exitOK
}

// syncImport runs `sync import --dir <dir>`: the chunk archive is applied
// through the store (the chunk-id dedupe makes re-import a no-op).
func syncImport(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("sync import", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var project, dir string
	fs.StringVar(&project, "project", "", "the project to import")
	fs.StringVar(&dir, "dir", "", "the archive directory")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if dir == "" {
		return errExit("sync import: --dir is required")
	}
	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	resolved := project
	if resolved == "" {
		r, rerr := h.CurrentProject("", "")
		if rerr != nil {
			return finish(fs, *jsonOut, dispatchResult{err: rerr})
		}
		resolved = r
	}
	n, err := memory.Import(context.Background(), st, resolved, dir)
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"project": resolved, "applied": n, "dir": dir}})
	}
	fmt.Fprintf(outW(), "imported %d chunks\n", n)
	return exitOK
}
