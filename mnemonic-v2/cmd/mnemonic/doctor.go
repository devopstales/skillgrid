// doctor.go — the doctor command (spec 005-transport-cli §2 + the
// doctor-strict-exits-nonzero-on-gap scenario): the whole-system health
// check. It runs the handler's capability checks (config, FTS, the
// vector leg, the embedder seam, the LLM seam) and reports them — a
// capability gap (a seam absent, a check failing) renders `degraded`.
// `--strict` exits 1 on any gap (the CI switch); without it the report
// is human-readable and exits 0 (the gap is reported, not fatal).

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/migrate"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// runDoctor runs `doctor` (and `mem doctor`, the same check). The
// migration-state check (TICKET-19 doctor completeness) is advisory: a v1
// dir detected but not migrated renders a `migrate-pending` line (the
// operator reads it and runs `mnemonic migrate`); it never fails the
// doctor (a pending migration is a report line, not a capability gap).
func runDoctor(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var (
		strict bool
		v1Dir  string
	)
	fs.BoolVar(&strict, "strict", false, "exit 1 on any capability gap (CI)")
	fs.StringVar(&v1Dir, "v1-dir", "", "the v1 store's data dir (the migration-state check's selector — absent when the v1 cutover is not in play)")
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, "")
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	result, err := h.Doctor(context.Background())
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	// The migration-state check (TICKET-19 doctor completeness): when a
	// --v1-dir is given, the doctor reports the v1 dir's migration state
	// (migrate-pending / migrate-done / advisory-ok when absent). The
	// check is advisory — it never fails the doctor (a pending migration
	// is a report line, not a capability gap), so it is not part of the
	// strict gate.
	if v1Dir != "" {
		if line, ok := migrationStateLine(h, st, v1Dir); ok {
			if *jsonOut {
				// Fold into the result's checks (the JSON shape is the
				// contract — the advisory line is a check entry).
				appendDoctorCheck(result, line)
			} else {
				fmt.Fprintln(outW(), line)
			}
		}
	}
	if *jsonOut {
		out := finish(fs, true, dispatchResult{result: result})
		if strict && doctorDegraded(result) {
			return exitErr
		}
		return out
	}
	printDoctorHuman(result)
	if strict && doctorDegraded(result) {
		return exitErr
	}
	return exitOK
}

// migrationStateLine runs the v1→v2 seed migration state check for the
// doctor's resolved project and renders the advisory line ("", false when
// the v1 store is absent — the check stays advisory-ok, no line). It calls
// migrate.Report directly (the cutover's own package owns the layout
// knowledge); the service's MigrationState hook is the library's wiring
// point, the CLI's direct call is the doctor's wiring point.
func migrationStateLine(h *handler.Handler, st store.Store, v1Dir string) (string, bool) {
	resolved, err := h.CurrentProject("", "")
	if err != nil {
		return "", false // a project-resolution gap is the doctor's own check's concern
	}
	cfg := h.Config()
	sts, err := migrate.Report(context.Background(), v1Dir, cfg.DataDir, resolved, st)
	if err != nil || !sts.V1Present {
		return "", false // no v1 store (or a read gap): advisory-ok, no line
	}
	line := fmt.Sprintf("migrate\t%s\tv1=%d rows, seeded=%d ledger rows",
		sts.MigrationState, sts.V1Rows, sts.SeededRows)
	return line, true
}

// appendDoctorCheck appends one check entry to the doctor's result (the
// JSON shape is the contract: the advisory line is a check entry with
// ok=true — a pending migration is not a capability gap, so the strict
// gate never fires on it).
func appendDoctorCheck(result any, line string) {
	m, ok := result.(map[string]any)
	if !ok {
		return
	}
	checks, ok := m["checks"].([]any)
	if !ok {
		return
	}
	// Parse the line's fields (name\tstate\tdetail).
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return
	}
	checks = append(checks, map[string]any{
		"name":   "migrate",
		"ok":     true, // advisory
		"detail": parts[1] + " — " + parts[2],
	})
	m["checks"] = checks
}

// doctorDegraded reports whether any capability check failed (the gap
// the strict mode exits on). The result is the handler's Doctor return
// (map[string]any with a "checks" key holding []DoctorCheck structs).
func doctorDegraded(result any) bool {
	// The handler's Doctor returns map[string]any{"checks": []DoctorCheck{...}}.
	// The DoctorCheck type is in the handler package; the doctor command
	// does not import it (the seam is the handler's return type). Instead,
	// marshal to JSON and re-parse (the JSON shape is the contract).
	b, err := jsonMarshal(result)
	if err != nil {
		return false
	}
	var m map[string]any
	if err := jsonUnmarshal(b, &m); err != nil {
		return false
	}
	checks, ok := m["checks"].([]any)
	if !ok {
		return false
	}
	for _, c := range checks {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if okFlag, _ := cm["ok"].(bool); !okFlag {
			return true
		}
	}
	return false
}

// printDoctorHuman renders the check table (the human-readable report:
// each check's name, its ok flag, and its detail on a gap).
func printDoctorHuman(result any) {
	m, ok := result.(map[string]any)
	if !ok {
		fmt.Fprintln(outW(), mustMarshal(result))
		return
	}
	checks, _ := m["checks"].([]any)
	fmt.Fprintln(outW(), "check\tok\tdetail")
	for _, c := range checks {
		cm, _ := c.(map[string]any)
		name, _ := cm["name"].(string)
		okFlag, _ := cm["ok"].(bool)
		detail, _ := cm["detail"].(string)
		s := "yes"
		if !okFlag {
			s = "no"
		}
		fmt.Fprintf(outW(), "%s\t%s\t%s\n", name, s, detail)
	}
	if doctorDegraded(result) {
		fmt.Fprintln(outW(), "\ndegraded: a capability gap is present (run with --strict to fail on it)")
	} else {
		fmt.Fprintln(outW(), "\nhealthy: all capability checks passed")
	}
}
