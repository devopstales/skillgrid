package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic/internal/project"
	"github.com/devopstales/skillgrid/mnemonic/internal/service"
)

// runSession handles `skillgrid session <id>`. It is the CLI read path for
// the session events layer (TICKET-04): it prints one session's events in
// sequence order with the net commit range, read-only over
// memory.SessionChanges. The old handoff/resume/status relay subcommands were
// deleted (TICKET-07, old-surfaces-removed): invoking one fails closed with an
// unknown-command error. A missing id / an unknown id / no usable store also
// fail closed (non-zero exit + stderr).
func runSession(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printSessionUsage()
		os.Exit(2)
	}
	cmd := args[0]
	rest := args[1:]

	fs := flag.NewFlagSet("session", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dataDir     string
		projectFlag string
		showDiff    bool
		jsonOut     bool
	)
	fs.StringVar(&dataDir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&projectFlag, "project", "", "project id (defaults to CWD-resolved)")
	fs.BoolVar(&showDiff, "show-diff", false, "show: include git diff --stat for the session commit range")
	fs.BoolVar(&jsonOut, "json", false, "show: emit JSON instead of a human summary")
	if err := fs.Parse(reorderSessionArgs(rest)); err != nil {
		os.Exit(2)
	}

	h, cleanup, err := openSessionService(dataDir, projectFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()
	pos := fs.Args()

	switch cmd {
	case "help", "-h", "--help":
		printSessionUsage()
	case "handoff", "resume", "status":
		// Removed relay subcommands (TICKET-07): never silently swallowed
		// as a session id — fail closed as an unknown command.
		fmt.Fprintf(os.Stderr, "error: unknown session command %q (the session relay was removed; use `session <id>`)\n", cmd)
		printSessionUsage()
		os.Exit(2)
	default:
		// `skillgrid session <id>`: the session read path (TICKET-04) over
		// SessionChanges. The id is args[0], unless flags precede it
		// (`session --show-diff <id>`), in which case it is the first
		// positional after flag parsing. A leading dash that is not a known
		// show flag stays a fail-closed unknown command (never silently
		// swallowed as an id).
		id := cmd
		if strings.HasPrefix(cmd, "-") {
			if cmd != "--show-diff" && cmd != "-show-diff" && cmd != "--json" && cmd != "-json" {
				fmt.Fprintf(os.Stderr, "error: unknown session command %q\n", cmd)
				printSessionUsage()
				os.Exit(2)
			}
			if len(pos) == 0 {
				fmt.Fprintln(os.Stderr, "error: session show requires a session id")
				printSessionUsage()
				os.Exit(2)
			}
			id = pos[0]
		}
		runSessionShow(h, id, showDiff, jsonOut)
	}
}

func printSessionUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid session <id> [flags]

   <id> [--show-diff] [--json]
           print the session's events in sequence order with the net commit
           range; --show-diff adds git diff --stat for from..to (an empty
           range prints a note, never errors) (session_changes)

Flags:
  --project ID    project bucket (defaults to CWD-resolved)
  --dir DATA_DIR  mnemonic data directory
`)
}

// openSessionService resolves the project store the SAME way the MCP server
// does (CWD-resolved when no --project, explicit --project otherwise) and
// opens it once. Any resolution / open failure is returned so the caller fails
// closed (no usable store -> non-zero exit + stderr).
func openSessionService(dataDir, projectName string) (*service.ProjectHandle, func(), error) {
	dd := dataDir
	if dd == "" {
		d, err := service.DefaultDataDir()
		if err != nil {
			return nil, nil, fmt.Errorf("resolve data dir: %w", err)
		}
		dd = d
	}
	svc := service.New(dd)
	var h *service.ProjectHandle
	var cleanup func()
	var err error
	if strings.TrimSpace(projectName) != "" {
		h, cleanup, err = svc.Open(project.NormalizeID(projectName))
	} else {
		h, cleanup, err = svc.OpenForCWD()
	}
	if err != nil {
		return nil, nil, err
	}
	return h, cleanup, nil
}

// runSessionShow is the CLI read path for the session events layer
// (TICKET-04): it prints one session's events in sequence order with the net
// commit range, read-only over memory.SessionChanges. An unknown session id
// fails closed (non-zero exit + stderr, no output). With --show-diff it
// appends `git diff --stat from..to` run in the session's recorded directory;
// an empty range (from or to empty) prints a note and never errors, and a git
// failure degrades to a note so the read path itself never fails on diff.
func runSessionShow(h *service.ProjectHandle, sessionID string, showDiff, jsonOut bool) {
	if strings.TrimSpace(sessionID) == "" {
		fmt.Fprintln(os.Stderr, "error: session show requires a session id")
		os.Exit(2)
	}
	events, from, to, err := h.Memory().SessionChanges(hCtx(), sessionID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	diffStat, diffNote := "", ""
	if showDiff {
		switch {
		case from == "" || to == "":
			diffNote = "no commit range recorded; skipping diff"
		default:
			stat, derr := gitDiffStat(sessionRepoDir(h, sessionID), from, to)
			if derr != nil {
				diffNote = fmt.Sprintf("git diff --stat failed: %v", derr)
			} else {
				diffStat = stat
			}
		}
	}

	if jsonOut {
		out := map[string]any{
			"session_id":  sessionID,
			"from_commit": from,
			"to_commit":   to,
			"events":      events,
		}
		if showDiff {
			out["diff_stat"] = diffStat
			out["diff_note"] = diffNote
		}
		printJSON(out)
		return
	}

	fmt.Printf("session %s\n", sessionID)
	if from == "" && to == "" {
		fmt.Println("range (no commits recorded)")
	} else {
		fmt.Printf("range %s..%s\n", from, to)
	}
	fmt.Println("events:")
	for _, e := range events {
		line := fmt.Sprintf("  #%d %s", e.Sequence, e.ActionType)
		var bits []string
		if e.ToolName != "" {
			bits = append(bits, "tool="+e.ToolName)
		}
		if e.Path != "" {
			bits = append(bits, "path="+e.Path)
		}
		if e.Command != "" {
			bits = append(bits, "command="+e.Command)
		}
		if e.Commit != "" {
			bits = append(bits, "commit="+e.Commit)
		}
		if len(bits) > 0 {
			line += " " + strings.Join(bits, " ")
		}
		line += " " + e.Timestamp
		fmt.Println(line)
	}
	if showDiff {
		if diffNote != "" {
			fmt.Printf("diff: %s\n", diffNote)
		} else {
			fmt.Printf("diff %s..%s:\n", from, to)
			if diffStat != "" {
				fmt.Println(diffStat)
			}
		}
	}
}

// sessionRepoDir returns the directory the session started in (the repo its
// commit range belongs to), falling back to the project root when the row
// carries none. Best-effort: it never errors, so --show-diff always has a
// directory to run git in.
func sessionRepoDir(h *service.ProjectHandle, sessionID string) string {
	var dir sql.NullString
	if err := h.Store().DB.QueryRow(
		`SELECT directory FROM sessions WHERE id = ?`, sessionID,
	).Scan(&dir); err == nil && strings.TrimSpace(dir.String) != "" {
		return dir.String
	}
	return h.Root()
}

// gitDiffStat runs `git -C dir diff --stat from to` and returns its output.
func gitDiffStat(dir, from, to string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "diff", "--stat", from, to)
	out, err := cmd.CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// reorderSessionArgs moves flags in front of positionals (same convention as
// mem / trail) so `session resume <id> --archive` parses like
// `session resume --archive <id>`.
func reorderSessionArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "-" || strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			if !strings.Contains(a, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
			continue
		}
		pos = append(pos, a)
	}
	return append(flags, pos...)
}
