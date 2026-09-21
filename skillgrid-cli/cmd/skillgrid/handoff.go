package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/handoff"
)

// runHandoff handles `skillgrid handoff <subcommand>` — the Handoff Hub CLI
// (change 015-handoff-hub). It exposes the unified checkpoint surface:
// change-snapshot recording/backfill, named checkpoints with drift
// verification, per-change archiving, and the aggregate status/rollup.
//
// Subcommands:
//
//	record      record HEAD as a change snapshot (idempotent; backfills if empty)
//	backfill    seed snapshots from git log (default 100 commits)
//	status      one-call "where are we": latest snapshot + checkpoints + refs
//	checkpoint  place a named checkpoint marker
//	verify      drift-verify a checkpoint (no name = verify all open)
//	archive     archive checkpoints for a spec dir (ship/finish)
//	rollup      write a session/team rollup markdown to .skillgrid/sdd/rollups/
func runHandoff(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printHandoffUsage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("handoff", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir string
		project string
		limit   int
		name    string
		evidence string
		specDir string
		asJSON  bool
	)
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (defaults to CWD-resolved)")
	fs.IntVar(&limit, "limit", 100, "backfill/record list limit")
	fs.StringVar(&evidence, "evidence", "", "checkpoint verification note")
	fs.StringVar(&specDir, "spec", "", "spec dir (for archive; default auto-detect)")
	fs.BoolVar(&asJSON, "json", false, "emit JSON instead of a human summary")

	if err := fs.Parse(rest); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	// The checkpoint/verify name is the first positional argument.
	if len(pos) > 0 {
		name = pos[0]
	}

	svc, proj := openMemService(dataDir, project)
	h, cleanup, err := svc.Open(proj)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open project: %v\n", err)
		os.Exit(1)
	}
	defer cleanup()

	hub := &handoff.Hub{DB: h.Store().DB, Project: h.ProjectID(), RepoDir: h.Root()}
	ctx := context.Background()

	switch cmd {
	case "record":
		runHandoffRecord(ctx, hub, asJSON)
	case "backfill":
		runHandoffBackfill(ctx, hub, limit, asJSON)
	case "status":
		runHandoffStatus(ctx, hub, asJSON)
	case "checkpoint":
		runHandoffCheckpoint(ctx, hub, name, evidence, specDir, asJSON)
	case "verify":
		runHandoffVerify(ctx, hub, name, asJSON)
	case "archive":
		runHandoffArchive(ctx, hub, specDir, asJSON)
	case "rollup":
		runHandoffRollup(ctx, hub, h.Root(), asJSON)
	case "help", "-h", "--help":
		printHandoffUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown handoff subcommand %q\n\n", cmd)
		printHandoffUsage()
		os.Exit(2)
	}
	_ = pos
}

func emit(asJSON bool, jsonVal any, human string) {
	if asJSON {
		b, _ := json.MarshalIndent(jsonVal, "", "  ")
		fmt.Println(string(b))
		return
	}
	fmt.Println(human)
}

func runHandoffRecord(ctx context.Context, hub *handoff.Hub, asJSON bool) {
	n, _ := hub.CountSnapshots(ctx)
	if n == 0 {
		// First run: backfill so the hub is not empty until the next commit.
		if _, err := hub.Backfill(ctx, 100); err != nil {
			fmt.Fprintf(os.Stderr, "warning: backfill: %v\n", err)
		}
	}
	s, err := hub.Record(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: record snapshot: %v\n", err)
		os.Exit(1)
	}
	emit(asJSON, s, fmt.Sprintf("SNAPSHOT %s %q (%s)", handoff.ShortID(s.Commit), s.Subject, s.Branch))
}

func runHandoffBackfill(ctx context.Context, hub *handoff.Hub, limit int, asJSON bool) {
	n, err := hub.Backfill(ctx, limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: backfill: %v\n", err)
		os.Exit(1)
	}
	emit(asJSON, map[string]int{"backfilled": n}, fmt.Sprintf("BACKFILL %d commit(s)", n))
}

func runHandoffStatus(ctx context.Context, hub *handoff.Hub, asJSON bool) {
	snapshots, _ := hub.ListSnapshots(ctx, 1)
	checkpoints, _ := hub.ListCheckpoints(ctx, 20)
	refs, _ := hub.ListHandoffRefs(ctx, 20)
	var latest *handoff.Snapshot
	if len(snapshots) > 0 {
		latest = &snapshots[0]
	}
	status := map[string]any{
		"latest_snapshot": latest,
		"checkpoints":     checkpoints,
		"handoff_refs":    refs,
	}
	if asJSON {
		emit(asJSON, status, "")
		return
	}
	if latest == nil {
		fmt.Println("No snapshots recorded yet.")
		return
	}
	fmt.Printf("LATEST %s %q (%s)\n", handoff.ShortID(latest.Commit), latest.Subject, latest.Branch)
	if len(checkpoints) > 0 {
		fmt.Printf("CHECKPOINTS %d (newest: %s [%s])\n", len(checkpoints), checkpoints[0].Name, checkpoints[0].Status)
	}
	if len(refs) > 0 {
		fmt.Printf("HANDOFF REFS %d\n", len(refs))
	}
}

func runHandoffCheckpoint(ctx context.Context, hub *handoff.Hub, name, evidence, specDir string, asJSON bool) {
	if name == "" {
		fmt.Fprintln(os.Stderr, "error: --name is required for checkpoint")
		os.Exit(2)
	}
	cp, err := hub.RecordCheckpoint(ctx, handoff.CheckpointInput{
		Name: name, Evidence: evidence, SpecDir: specDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: checkpoint: %v\n", err)
		os.Exit(1)
	}
	dirty := ""
	if cp.Dirty {
		dirty = " dirty"
	}
	emit(asJSON, cp, fmt.Sprintf("CHECKPOINT %s @ %s%s (spec: %s)", cp.Name, handoff.ShortID(cp.Commit), dirty, orDash(cp.SpecDir)))
}

func runHandoffVerify(ctx context.Context, hub *handoff.Hub, name string, asJSON bool) {
	if name == "" {
		// Verify all open checkpoints.
		open, _ := hub.ListCheckpoints(ctx, 200)
		var out []any
		for _, c := range open {
			if c.Status != "open" {
				continue
			}
			rep, err := hub.VerifyCheckpoint(ctx, c.Name)
			if err != nil {
				continue
			}
			out = append(out, rep)
		}
		emit(asJSON, out, fmt.Sprintf("VERIFIED %d open checkpoint(s)", len(out)))
		return
	}
	rep, err := hub.VerifyCheckpoint(ctx, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: verify: %v\n", err)
		os.Exit(1)
	}
	if asJSON {
		emit(asJSON, rep, "")
		return
	}
	fmt.Printf("VERIFY %s -> %s (branch %v, commit %v, dirty-now %v)\n",
		rep.Checkpoint.Name, rep.Recommendation, rep.BranchMatch, rep.CommitMatch, rep.DirtyNow)
	if len(rep.ChangedSince) > 0 {
		fmt.Printf("  changed since: %d file(s)\n", len(rep.ChangedSince))
	}
}

func runHandoffArchive(ctx context.Context, hub *handoff.Hub, specDir string, asJSON bool) {
	if specDir == "" {
		// Auto-detect the active spec dir.
		checkpoints, _ := hub.ListCheckpoints(ctx, 200)
		for _, c := range checkpoints {
			if c.SpecDir != "" {
				specDir = c.SpecDir
				break
			}
		}
	}
	if specDir == "" {
		fmt.Fprintln(os.Stderr, "error: no spec dir (pass --spec)")
		os.Exit(2)
	}
	n, err := hub.ArchiveCheckpoints(ctx, specDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: archive: %v\n", err)
		os.Exit(1)
	}
	emit(asJSON, map[string]any{"archived": n, "spec_dir": specDir}, fmt.Sprintf("ARCHIVED %d checkpoint(s) for %s", n, specDir))
}

func runHandoffRollup(ctx context.Context, hub *handoff.Hub, root string, asJSON bool) {
	snapshots, _ := hub.ListSnapshots(ctx, 200)
	checkpoints, _ := hub.ListCheckpoints(ctx, 200)
	refs, _ := hub.ListHandoffRefs(ctx, 200)
	p, err := handoff.WriteRollup(ctx, hub, root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: rollup: %v\n", err)
		os.Exit(1)
	}
	emit(asJSON, map[string]any{"path": p, "snapshots": len(snapshots), "checkpoints": len(checkpoints), "handoff_refs": len(refs)},
		fmt.Sprintf("ROLLUP %s", p))
}

func printHandoffUsage() {
	fmt.Print(`skillgrid handoff — Handoff Hub (change snapshots, checkpoints, handoff refs)

Usage:
  skillgrid handoff record [--json]
  skillgrid handoff backfill [--limit N] [--json]
  skillgrid handoff status [--json]
  skillgrid handoff checkpoint NAME [--evidence TEXT] [--spec DIR] [--json]
  skillgrid handoff verify [NAME] [--json]
  skillgrid handoff archive [--spec DIR] [--json]
  skillgrid handoff rollup [--json]

Flags:
  --dir D         mnemonic data directory (default $SKILLGRID_MNEMONIC_DATA_DIR)
  --project P     project id (default: CWD-resolved)
  --limit N       backfill/record list limit (default 100)
  --evidence TEXT checkpoint verification note
  --spec DIR      spec dir (archive target; default auto-detect)
  --json          emit JSON

`)
}

