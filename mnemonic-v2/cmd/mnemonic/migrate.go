// migrate.go — the v1→v2 seed migration command (TICKET-19, one-way door
// D4): a one-time export of the v1 store into the v2 store. `mnemonic
// migrate --v1-dir <dir> --project <p>` reads the v1 store at
// <v1-dir>/<p>.sqlite read-only (the v1 files come back byte-identical),
// and seeds the v2 store through the normal save pipeline (redaction,
// dedup, importance, and the sync journal all apply — INV-4, no special
// import path). Idempotent: a re-run is a per-row no-op (the replay
// ledger's v1migrate: operation ids replay each source row).
//
// The report (exported / skipped / errors) is the command's output; a
// per-row save failure is a report entry, not a command failure — a bad
// row must not wedge the rest of the seed. The command exits 0 with the
// report printed (the cutover is a one-way door: the operator reads the
// report, decides, and keeps or deletes the v1 files — the migration
// itself never touches them).

package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/migrate"
)

// runMigrate runs `migrate --v1-dir <dir> [--project <p>]`: the one-time
// v1→v2 seed. The v1 dir is the --v1-dir flag (brief: the CLI's v1
// selector); the project is the --project flag (beats the detector, like
// the other commands — the seed names the project explicitly). The v2
// store is opened through the normal openRuntime (the resolved config's
// data dir + dialect); the seed's data dir is the handler's resolved
// config's DataDir.
func runMigrate(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var (
		v1Dir   string
		project string
	)
	fs.StringVar(&v1Dir, "v1-dir", "", "the v1 store's data dir (required — the v1 files are read read-only from here)")
	fs.StringVar(&project, "project", "", "the project to migrate (beats detection)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if v1Dir == "" {
		fmt.Fprintln(outW(), "error: --v1-dir is required (the v1 store's data dir)")
		return exitUsage
	}

	h, st, err := openRuntime(cwd, fl, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()

	cfg := h.Config()
	// The project the store was opened for (openRuntime resolved it from
	// the --project flag / detector chain). Re-resolve with the same
	// explicit so the seed names the project explicitly (not re-resolved
	// at migrate time).
	resolved, err := h.CurrentProject(cwd, project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	_ = st // the store is the doctor's read surface; the seed re-opens its
	// own v2 store through store.Open (the seed's data dir is cfg.DataDir,
	// the same path the runtime opened).
	rep, err := migrate.Migrate(context.Background(), v1Dir, cfg.DataDir, resolved)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{
			"exported": rep.Exported,
			"skipped":  rep.Skipped,
			"errors":   orEmpty(rep.Errors),
		}})
	}
	fmt.Fprintf(outW(), "migrated %s from %s\n", resolved, v1Dir)
	fmt.Fprintf(outW(), "  exported: %d\n", rep.Exported)
	fmt.Fprintf(outW(), "  skipped:  %d (replayed by the ledger — a re-run)\n", rep.Skipped)
	for _, e := range rep.Errors {
		fmt.Fprintf(outW(), "  error: %s\n", e)
	}
	return exitOK
}

// orEmpty renders an empty slice as [] (not null) in the JSON report.
func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
