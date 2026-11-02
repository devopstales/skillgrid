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

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
)

// runDoctor runs `doctor` (and `mem doctor`, the same check).
func runDoctor(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := jsonFlag(fs)
	var strict bool
	fs.BoolVar(&strict, "strict", false, "exit 1 on any capability gap (CI)")
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
