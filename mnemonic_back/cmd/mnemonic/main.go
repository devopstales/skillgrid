package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "0.1.0-dev"

func main() {
	fs := flag.NewFlagSet("mnemonic", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		w := fs.Output()
		for _, line := range []string{
			" \033[38;5;141m███████╗██╗  ██╗██╗██╗     ██╗     ██████╗  ██████╗ ██╗██████╗  \033[0m",
			" \033[38;5;141m██╔════╝██║  ██║██║██║     ██║    ██╔════╝  ██╔══██╗██║██╔══██╗ \033[0m",
			" \033[38;5;141m███████╗███████║██║██║     ██║    ██║   ███╗██████╔╝██║██║  ██║ \033[0m",
			" \033[38;5;141m╚════██║██╔══██║██║██║     ██║    ██║    ██║██╔══██╗██║██║  ██║ \033[0m",
			" \033[38;5;141m███████║██║  ██║██║███████╗███████╗╚██████╔╝██║  ██║██║██████╔╝ \033[0m",
			" \033[38;5;141m╚══════╝╚═╝  ╚═╝╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝╚═════╝  \033[0m",
		} {
			fmt.Fprintln(w, line)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w, `mnemonic — local-first memory and code intelligence`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Usage:`)
		fmt.Fprintln(w, `  mnemonic <command> [flags]`)
		fmt.Fprintln(w)
		fmt.Fprintln(w, `Commands:`)
		fmt.Fprintln(w, `  mcp           Run the Mnemonic MCP stdio server`)
		fmt.Fprintln(w, `  serve         Run the Mnemonic HTTP API (default :7438)`)
		fmt.Fprintln(w, `  init          Project boot file, code index, and doc ingest`)
		fmt.Fprintln(w, `  index         Incremental code indexing`)
		fmt.Fprintln(w, `  prime         Session-start context (last task, diff, impact)`)
		fmt.Fprintln(w, `  compact       Session-close one-liner`)
		fmt.Fprintln(w, `  pages         Regenerate gitignored architecture and decisions pages`)
		fmt.Fprintln(w, `  diff-impact   Caller counts for files in the current diff`)
		fmt.Fprintln(w, `  orient        Tier-1 code orientation (signature, TOC, map, list, metadata)`)
		fmt.Fprintln(w, `  grep          Structural by-example grep (index-free, per-language AST)`)
		fmt.Fprintln(w, `  callers       Graph callers for a symbol (confidence-labeled)`)
		fmt.Fprintln(w, `  callees       Graph callees for a symbol (confidence-labeled)`)
		fmt.Fprintln(w, `  dependents    Graph dependents for a symbol (confidence-labeled)`)
		fmt.Fprintln(w, `  implementors  Graph implementors for a symbol`)
		fmt.Fprintln(w, `  hierarchy     Structural hierarchy (extends/implements)`)
		fmt.Fprintln(w, `  tests-for     Test relationships for a symbol`)
		fmt.Fprintln(w, `  path          Shortest edge path, or where the graph stops`)
		fmt.Fprintln(w, `  explain       Symbol node + degree + connections ranked by degree`)
		fmt.Fprintln(w, `  impact        Risk-tiered blast radius (WILL BREAK / LIKELY AFFECTED)`)
		fmt.Fprintln(w, `  explore       Composite: source + call-flow + blast radius in one call`)
		fmt.Fprintln(w, `  setup         Install agent plugins (opencode|kilocode|cursor)`)
		fmt.Fprintln(w, `  migrate       Backfill mnemonic tier sidecars (--tier)`)
		fmt.Fprintln(w, `  trail         Inspect retrieval trails (recent|show)`)
		fmt.Fprintln(w, `  search        Hybrid code search (FTS + signals + semantic, per-signal provenance)`)
		fmt.Fprintln(w, `  mem           Memory tools (layers|governance|share|search|context|timeline)`)
		fmt.Fprintln(w, `  memory        Fact Memory (fact add|search|forget|decay|list)`)
		fmt.Fprintln(w, `  skill         Agent Skill registry (write|list|search|execute)`)
		fmt.Fprintln(w, `  logs          Agent tool-call audit log (--today, --agent, --action, --file, -f to follow live)`)
		fmt.Fprintln(w, `  sessions      List agent sessions (harness, tool calls, last tool, cost)`)
		fmt.Fprintln(w, `  session       Show one session's events (session <id>)`)
		fmt.Fprintln(w, `  stats         Per-agent activity rollup (--since 24h)`)
		fmt.Fprintln(w, `  policy        Pre-tool policy (init|validate|test --action --path --command --tool)`)
		fmt.Fprintln(w, `  embedding-status  Report active embedder provider/model and embedded counts`)
		fmt.Fprintln(w, `  doctor        Functional health check (embed round-trip, capabilities; --strict for CI)`)
		fmt.Fprintln(w, `  eval          Retrieval-eval ablation (--corpus self | name=path; leak-free git queries)`)
		fmt.Fprintln(w, `  help          Show this help`)
		fmt.Fprintln(w)
	}

	var vVersion bool
	fs.BoolVar(&vVersion, "version", false, "print version and exit")
	fs.BoolVar(&vVersion, "v", false, "shorthand for --version")

	rest := os.Args[1:]
	rest0 := ""
	if len(rest) > 0 {
		rest0 = rest[0]
	}

	switch rest0 {
	case "mcp":
		runMCP(version, rest[1:])
		return
	case "serve":
		runServe(version, rest[1:])
		return
	case "init":
		runProjectInit(rest[1:])
		return
	case "index":
		runIndex(version, rest[1:])
		return
	case "prime":
		runPrime(rest[1:])
		return
	case "compact":
		runCompact(rest[1:])
		return
	case "pages":
		runPages(rest[1:])
		return
	case "diff-impact":
		runDiffImpact(rest[1:])
		return
	case "orient":
		runCodeIntel(version, rest)
		return
	case "grep":
		runCodeIntel(version, rest)
		return
	case "callers", "callees", "dependents", "implementors", "hierarchy",
		"tests-for", "path", "explain", "impact", "explore",
		"docs", "configs", "sql-schema", "sql-access":
		runCodeIntel(version, rest)
		return
	case "setup":
		runSetup(version, rest[1:])
		return
	case "migrate":
		runMigrate(version, rest[1:])
		return
	case "trail":
		runTrail(version, rest[1:])
		return
	case "search":
		runSearch(version, rest[1:])
		return
	case "mem":
		runMem(version, rest[1:])
		return
	case "memory":
		runMemory(version, rest[1:])
		return
	case "skill":
		runSkill(version, rest[1:])
		return
	case "session":
		runSession(version, rest[1:])
		return
	case "sessions":
		runSessions(rest[1:])
		return
	case "logs":
		runLogs(rest[1:])
		return
	case "stats":
		runStats(rest[1:])
		return
	case "policy":
		runPolicy(rest[1:])
		return
	case "embedding-status":
		runSearchEmbeddingStatus(version, rest[1:])
		return
	case "doctor":
		runDoctor(version, rest[1:])
		return
	case "eval":
		runEval(version, rest[1:])
		return
	case "help", "-h", "--help":
		fallthrough
	case "":
		fs.Usage()
		return
	default:
		if strings.HasPrefix(rest0, "-") {
			if err := fs.Parse(rest); err != nil {
				os.Exit(2)
			}
			if vVersion {
				fmt.Println("mnemonic", version)
				return
			}
			fs.Usage()
			return
		}
		fmt.Fprintf(os.Stderr, "error: unknown command %q (see --help)\n", rest0)
		os.Exit(2)
	}
}
