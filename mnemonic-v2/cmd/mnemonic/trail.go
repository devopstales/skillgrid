// trail.go — the trail command group (spec 005-transport-cli §2 + §5):
// the CLI exposes the INV-5 audit trail (the retrieval trails the search
// / lookup / VectorSearch layer records — the "why did the agent recall
// this" audit record). `trail recent` lists the newest retrievals for
// the project; `trail show <id>` prints one trail row. The CLI only
// inspects the trails; the recording is the retrieval layer's job.

package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
)

// runTrail dispatches the trail subcommands.
func runTrail(fl config.Flags, cwd string, args []string) int {
	if len(args) == 0 {
		return errExit("trail: a subcommand is required (recent|show)")
	}
	switch args[0] {
	case "recent":
		return trailRecent(fl, cwd, args[1:])
	case "show":
		return trailShow(fl, cwd, args[1:])
	default:
		if len(args) > 0 && args[0][0] == '-' {
			return errExit("trail: unknown flag %q before a subcommand", args[0])
		}
		return errExit("trail: unknown subcommand %q", args[0])
	}
}

// trailRow is one retrieval_trails row (the inspection shape).
type trailRow struct {
	ID        int64
	Route     string
	Query     string
	Project   string
	Filters   string
	IDs       string
	Degraded  bool
	Timestamp string
}

// queryTrails runs a raw SELECT through the store's RawQuery seam (the
// trail inspection is a read; the retrieval layer owns the writes).
func queryTrails(st store.Store, query string, args ...any) ([][]any, error) {
	rq, ok := st.(interface {
		RawQuery(query string, args ...any) ([][]any, error)
	})
	if !ok {
		return nil, fmt.Errorf("trail: store %T does not expose the RawQuery seam", st)
	}
	return rq.RawQuery(query, args...)
}

// trailRecent runs `trail recent`: the newest 20 retrievals for the
// project (the spec's read surface; the project+timestamp index is the
// query shape).
func trailRecent(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("trail recent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	h, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	project := pf.project
	if project == "" {
		project, err = h.CurrentProject("", "")
		if err != nil {
			return finish(fs, *jsonOut, dispatchResult{err: err})
		}
	}
	rows, err := queryTrails(st, `
		SELECT id, route, query, ifnull(project,''), ifnull(filters,''), ifnull(ids,''),
		       degraded, timestamp
		FROM retrieval_trails WHERE project = ?
		ORDER BY timestamp DESC, id DESC LIMIT 20`, project)
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	out := make([]trailRow, 0, len(rows))
	for _, r := range rows {
		out = append(out, trailRow{
			ID:        int64At(r, 0),
			Route:     strAt(r, 1),
			Query:     strAt(r, 2),
			Project:   strAt(r, 3),
			Filters:   strAt(r, 4),
			IDs:       strAt(r, 5),
			Degraded:  int64At(r, 6) != 0,
			Timestamp: strAt(r, 7),
		})
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: map[string]any{"trails": out}})
	}
	if len(out) == 0 {
		fmt.Fprintln(outW(), "no retrieval trails recorded")
		return exitOK
	}
	fmt.Fprintln(outW(), "id\ttimestamp\troute\tquery\tdegraded")
	for _, tr := range out {
		d := "no"
		if tr.Degraded {
			d = "yes"
		}
		fmt.Fprintf(outW(), "%d\t%s\t%s\t%s\t%s\n", tr.ID, tr.Timestamp, tr.Route, tr.Query, d)
	}
	return exitOK
}

// trailShow runs `trail show <id>`: one trail row by id.
func trailShow(fl config.Flags, cwd string, args []string) int {
	fs := flag.NewFlagSet("trail show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var pf projectFlag
	pf.add(fs)
	jsonOut := jsonFlag(fs)
	if err := fs.Parse(reorderArgs(args)); err != nil {
		return exitUsage
	}
	if len(fs.Args()) != 1 {
		return errExit("trail show: a trail id is required")
	}
	id, err := strconv.ParseInt(fs.Args()[0], 10, 64)
	if err != nil {
		return errExit("trail show: id must be an integer, got %q", fs.Args()[0])
	}
	_, st, err := openRuntime(cwd, fl, pf.project)
	if err != nil {
		fmt.Fprintf(outW(), "error: %s\n", err.Error())
		return exitErr
	}
	defer st.Close()
	rows, err := queryTrails(st, `
		SELECT id, route, query, ifnull(project,''), ifnull(filters,''), ifnull(ids,''),
		       degraded, timestamp
		FROM retrieval_trails WHERE id = ?`, id)
	if err != nil {
		return finish(fs, *jsonOut, dispatchResult{err: err})
	}
	if len(rows) == 0 {
		return finish(fs, *jsonOut, dispatchResult{err: fmt.Errorf("trail: no rows found for id %d", id)})
	}
	r := rows[0]
	tr := trailRow{
		ID:        int64At(r, 0),
		Route:     strAt(r, 1),
		Query:     strAt(r, 2),
		Project:   strAt(r, 3),
		Filters:   strAt(r, 4),
		IDs:       strAt(r, 5),
		Degraded:  int64At(r, 6) != 0,
		Timestamp: strAt(r, 7),
	}
	if *jsonOut {
		return finish(fs, true, dispatchResult{result: tr})
	}
	fmt.Fprintf(outW(), "id: %d\n", tr.ID)
	fmt.Fprintf(outW(), "timestamp: %s\n", tr.Timestamp)
	fmt.Fprintf(outW(), "route: %s\n", tr.Route)
	fmt.Fprintf(outW(), "query: %s\n", tr.Query)
	fmt.Fprintf(outW(), "project: %s\n", tr.Project)
	fmt.Fprintf(outW(), "filters: %s\n", tr.Filters)
	fmt.Fprintf(outW(), "ids: %s\n", tr.IDs)
	fmt.Fprintf(outW(), "degraded: %v\n", tr.Degraded)
	return exitOK
}

// int64At / strAt read a RawQuery column (the []any row shape).
func int64At(r []any, i int) int64 {
	if i >= len(r) || r[i] == nil {
		return 0
	}
	switch v := r[i].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	default:
		return 0
	}
}

func strAt(r []any, i int) string {
	if i >= len(r) || r[i] == nil {
		return ""
	}
	switch v := r[i].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
