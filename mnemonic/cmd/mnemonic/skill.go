package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/embedder"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/facts"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/hybrid"
	"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/skills"
)

// runSkill handles `skillgrid skill <write|list|search|execute>` — the CLI
// parity layer for the Agent Skill MCP tools (change 2026-09-04-hermes-memory,
// TICKET-06). It routes through the same service seams the MCP tools use
// (svc.Open + skills.Store) so outcomes match MCP: write returns the skill id
// and a code path, list/search return the registry rows, and execute runs the
// skill in the sandbox and reports stdout/stderr/exit code. Table is the
// default output; --json emits the machine-readable form. Invalid actions fail
// before any store is opened.
func runSkill(version string, args []string) {
	_ = version
	if len(args) == 0 {
		printSkillUsage()
		os.Exit(2)
	}
	cmd := args[0]
	fs := flag.NewFlagSet("skill", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		dir         string
		project     string
		root        string
		name        string
		language    string
		description string
		code        string
		overwrite   bool
		query       string
		limit       int
		includeDel  bool
		input       string
		asJSON      bool
		mode        string
		sessionID   string
	)
	fs.StringVar(&dir, "dir", envOr("SKILLGRID_MNEMONIC_DATA_DIR", ""), "mnemonic data directory")
	fs.StringVar(&project, "project", "", "project id (defaults to CWD-resolved)")
	fs.StringVar(&root, "root", "", "workspace root for the skill files (default: current directory)")
	fs.StringVar(&name, "name", "", "skill name (write|execute)")
	fs.StringVar(&language, "language", "", "language tag, e.g. sh, python, go (write)")
	fs.StringVar(&description, "description", "", "what the skill does (write)")
	fs.StringVar(&code, "code", "", "skill code written verbatim to the FS file (write)")
	fs.BoolVar(&overwrite, "overwrite", false, "replace the skill if the name is already taken (write)")
	fs.StringVar(&query, "query", "", "search terms (search)")
	fs.IntVar(&limit, "limit", 20, "max results (list|search)")
	fs.BoolVar(&includeDel, "include-deleted", false, "list|search: include soft-deleted skills")
	fs.StringVar(&input, "input", "", "input passed to the skill as a trailing argument (execute)")
	fs.StringVar(&sessionID, "session-id", envOr("SKILLGRID_SESSION_ID", ""), "session id for the skill_use trail event (execute)")
	fs.BoolVar(&asJSON, "json", false, "print JSON instead of a table")
	fs.StringVar(&mode, "mode", "", "search: FTS match mode (fts|hybrid; hybrid degrades to fts with no embedder)")
	if err := fs.Parse(reorderSkillArgs(args[1:])); err != nil {
		os.Exit(2)
	}
	pos := fs.Args()
	if name == "" {
		name = posArg(pos, 0)
	}
	if query == "" {
		query = strings.Join(pos, " ")
	}
	if code == "" && cmd == "write" && len(pos) > 1 {
		// A positional code after the name is a convenience for short skills.
		code = pos[1]
	}

	switch cmd {
	case "write":
		if name == "" || language == "" || code == "" {
			fmt.Fprintln(os.Stderr, "error: skill write requires --name, --language and --code")
			os.Exit(2)
		}
	case "list":
		// read-only; honors --limit/--include-deleted.
	case "search":
		if query == "" {
			fmt.Fprintln(os.Stderr, "error: skill search requires a --query or positional query")
			os.Exit(2)
		}
		if mode != "" && !searchModeValid(mode) {
			fmt.Fprintf(os.Stderr, "error: --mode must be fts or hybrid, got %q\n", mode)
			os.Exit(2)
		}
	case "execute":
		if name == "" {
			fmt.Fprintln(os.Stderr, "error: skill execute requires a --name or positional skill name")
			os.Exit(2)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown skill command %q\n", cmd)
		printSkillUsage()
		os.Exit(2)
	}

	svc, projID := openMemService(dir, project)
	h, cleanup, err := svc.Open(projID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer cleanup()

	storeRoot := root
	if storeRoot == "" {
		// The ContentPlane scratch tree (.skillgrid/files/skills) lives under
		// the workspace root the operator is working in, matching the MCP
		// handler's h.Root(). The CLI has no CWD root, so the default is the
		// current working directory (the workspace the operator is in).
		storeRoot, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
	st := skills.New(h.Store().DB, storeRoot, projID)
	ctx := hCtx()
	switch cmd {
	case "write":
		id, err := st.Write(ctx, name, language, description, code, overwrite)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		sk, _ := st.List(ctx)
		codePath := ""
		for _, s := range sk {
			if s.Name == name {
				codePath = s.CodePath
				break
			}
		}
		printSkillOut(asJSON, map[string]any{
			"skill_id": id, "name": name, "project": projID, "code_path": codePath, "event": "skill_write",
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "wrote skill %q (id %d) → %s\n", name, id, codePath)
		})
	case "list":
		out, err := st.SearchWith(ctx, "", limit, false)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		if includeDel {
			out, err = st.SearchWith(ctx, "", limit, true)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
		// A blank-query FTS match returns nothing; fall back to the full
		// registry listing for the no-filter `skill list` contract.
		if len(out) == 0 {
			out, err = st.List(ctx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
		}
		printSkillOut(asJSON, map[string]any{"skills": out, "count": len(out)}, func(w *tabwriter.Writer) {
			writeSkillTable(w, out)
		})
	case "search":
		if mode == "hybrid" {
			factSt := facts.New(h.Store().DB, projID)
			res, err := hybrid.SearchMemory(ctx, factSt, st, query, limit, hybrid.MemoryOptions{
				Embedder:       embedder.Default(),
				IncludeDeleted: includeDel,
			})
			if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				os.Exit(1)
			}
			hybridNote(asJSON, res.Legs)
			printSkillOut(asJSON, map[string]any{"skills": res.Skills, "count": len(res.Skills), "legs": res.Legs}, func(w *tabwriter.Writer) {
				writeHybridSkillTable(w, res.Skills)
			})
			break
		}
		out, err := st.SearchWith(ctx, query, limit, includeDel)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		printSkillOut(asJSON, map[string]any{"skills": out, "count": len(out)}, func(w *tabwriter.Writer) {
			writeSkillTable(w, out)
		})
	case "execute":
		res, err := st.ExecuteWith(ctx, name, input, sessionID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		status := "success"
		if res.ExitCode != 0 || res.TimedOut {
			status = "failure"
		}
		printSkillOut(asJSON, map[string]any{
			"stdout": res.Stdout, "stderr": res.Stderr, "exit_code": res.ExitCode,
			"timed_out": res.TimedOut, "usage_id": res.UsageID, "status": status,
		}, func(w *tabwriter.Writer) {
			fmt.Fprintf(w, "exit %d %s\n", res.ExitCode, status)
			if res.Stdout != "" {
				fmt.Fprintln(w, "stdout:")
				fmt.Fprintln(w, res.Stdout)
			}
			if res.Stderr != "" {
				fmt.Fprintln(w, "stderr:")
				fmt.Fprintln(w, res.Stderr)
			}
		})
	}
}

// writeSkillTable renders the skill list as a human-readable table (id, name,
// language, description). It is the default (non-JSON) output.
func writeSkillTable(w *tabwriter.Writer, out []skills.Skill) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no skills")
		return
	}
	fmt.Fprintln(w, "ID\tNAME\tLANGUAGE\tDESCRIPTION")
	for _, s := range out {
		desc := strings.ReplaceAll(s.Description, "\n", " ")
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.ID, s.Name, s.Language, desc)
	}
}

// printSkillOut emits either the JSON object (the machine-readable form the
// MCP tools return) or the table produced by tableFn (the default).
func printSkillOut(asJSON bool, obj map[string]any, tableFn func(w *tabwriter.Writer)) {
	if asJSON {
		printJSON(obj)
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	tableFn(w)
	w.Flush()
}

// posArg returns args[i] when it exists, else "".
func posArg(args []string, i int) string {
	if i < len(args) {
		return args[i]
	}
	return ""
}

// searchModeValid reports whether mode is an accepted CLI search mode: empty
// (default), fts, or hybrid (hybrid degrades to the lexical path with no
// embedder). Anything else is a usage error before any store is opened.
func searchModeValid(mode string) bool {
	switch mode {
	case "", "fts", "hybrid":
		return true
	default:
		return false
	}
}

// reorderSkillArgs moves flags before positionals so `search q --json` and
// `search --json q` both parse (the same convention as reorderMemArgs).
func reorderSkillArgs(args []string) []string {
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

// writeHybridSkillTable renders the hybrid RRF-fused skill list as a table
// (id, score, source, name, language, description).
func writeHybridSkillTable(w *tabwriter.Writer, out []hybrid.MemorySkill) {
	if len(out) == 0 {
		fmt.Fprintln(w, "no skills")
		return
	}
	fmt.Fprintln(w, "ID\tSCORE\tSOURCE\tNAME\tLANGUAGE\tDESCRIPTION")
	for _, s := range out {
		desc := strings.ReplaceAll(s.Description, "\n", " ")
		src := "fts"
		if s.Provenance.Semantic {
			src = "semantic"
		}
		fmt.Fprintf(w, "%d\t%.4f\t%s\t%s\t%s\t%s\n", s.ID, s.Score, src, s.Name, s.Language, desc)
	}
}

func printSkillUsage() {
	fmt.Fprint(os.Stderr, `usage: skillgrid skill <write|list|search|execute> [args]

  write --name NAME --language LANG --code CODE [--description D] [--overwrite] [--json]
                                     create (or replace) an Agent Skill; prints id + code path
  list [--limit N] [--include-deleted] [--json]
                                     list live skills, newest first
  search <query> [--limit N] [--include-deleted] [--mode fts|hybrid] [--json]
                                     lexical FTS search over the skill registry
  execute <name> [--input IN] [--json]
                                     run the skill in the sandbox; prints stdout/stderr/exit code

Flags:
  --project ID     project bucket (defaults to CWD-resolved)
  --dir DATA_DIR   mnemonic data directory
  --root DIR       workspace root for the skill files (default: current directory)
  --json           print the machine-readable JSON form instead of a table
`)
}
