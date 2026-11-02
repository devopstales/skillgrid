// Package main is the mnemonic-v2 CLI (spec 005-transport-cli): the
// human/ops interface over the SAME shared handler the MCP stdio server
// uses (the K6 byte-identical seam — the CLI and MCP never diverge in
// behavior; they are two adapters over one Handler).
//
// Dispatch is stdlib flag (the v1 pattern): each command builds a flag
// set, parses it, opens the store, and calls the handler. The store is
// opened per-invocation (the CLI is not a server); a usage error exits 2,
// a handler error is printed (the envelope's code + message) and exits 1.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/config"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/project"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/core/store"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/memory"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/envelope"
	"github.com/devopstales/skillgrid/mnemonic-v2/internal/transport/handler"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
const version = "0.1.0"

// exit codes (the v1 contract): 0 ok, 1 handler/doctor-strict error, 2
// usage error.
const (
	exitOK    = 0
	exitErr   = 1
	exitUsage = 2
)

// usageError marks a flag/arg error (exit 2, message on stderr, no
// envelope — the call never reached the handler).
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

// errUsage builds a usage error.
func errUsage(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// usageErrorf reports whether err is a usage error (the exit-code split).
func isUsageError(err error) bool {
	_, ok := err.(*usageError)
	return ok
}

// projectFlag is the shared --project / --all selector pair (spec
// 005-transport-cli §3). Resolve runs the detection chain at call time:
// explicit > env > git > basename; --all is the "every project" marker.
// An explicit selector combined with --all is the pinned usage error
// (project.ErrAllWithProject).
type projectFlag struct {
	project string
	all     bool
}

// add registers --project and --all on fs.
func (pf *projectFlag) add(fs *flag.FlagSet) {
	fs.StringVar(&pf.project, "project", "", "project selector (beats "+config.EnvProject+" and detection)")
	fs.BoolVar(&pf.all, "all", false, "read every project (do not combine with --project)")
}

// gitToplevel is the production git binding (shells out; a lookup error
// is a signal, not a failure — the detector falls through to basename).
// It is a var (not a func) so the test fixture can swap it for a no-op
// (the CLI tests never shell out to git).
var gitToplevel = func(dir string) (string, error) {
	out, err := gitOutput(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// openRuntime builds the resolved config, the detector, the store, and
// the shared handler for one invocation. It is the single wiring point
// the commands share: the store is opened for the resolved project (the
// dialect render), the service owns the store + the intelligence seams,
// and the handler is the seam the commands call (the same handler the
// MCP server wraps — the K6 seam).
//
// The cwd the detector resolves against is the invocation's cwd (the CLI
// is a human process; the human's directory is the project binding).
//
// explicit is the command's --project selector: it wins over the
// detector's chain (spec 005-transport-cli §3: explicit > env > git >
// basename). An empty explicit runs the chain.
func openRuntime(cwd string, fl config.Flags, explicit string) (*handler.Handler, store.Store, error) {
	cfg, err := config.Load(cwd, fl)
	if err != nil {
		return nil, nil, err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	p, err := det.Resolve(cwd, explicit, gitToplevel)
	if err != nil {
		return nil, nil, err
	}
	st, err := storeFor(cfg, p)
	if err != nil {
		return nil, nil, err
	}
	h, err := handlerFor(cfg, st)
	if err != nil {
		_ = st.Close()
		return nil, nil, err
	}
	return h, st, nil
}

// storeFor opens the store for project (the production wiring; the test
// seam testStoreFor overrides it). A non-SQLite dialect renders through
// the store's dialect switch.
var storeFor = func(cfg config.Config, project string) (store.Store, error) {
	return store.Open(context.Background(), cfg.DataDir, project, dialectFor(cfg), cfg.Datasource.DSN)
}

// handlerFor builds the shared handler over a store (the production
// wiring; the test seam testHandlerFor overrides it). The intelligence
// seams are built from the config (a nil seam is the INV-3 degraded
// floor); the detector is the project chain (explicit > env > git >
// basename) so a command that changes projects between calls re-resolves
// correctly.
var handlerFor = func(cfg config.Config, st store.Store) (*handler.Handler, error) {
	llm, err := cfg.BuildLLM()
	if err != nil {
		return nil, err
	}
	emb, err := cfg.BuildEmbedder()
	if err != nil {
		return nil, err
	}
	det := project.Detector{EnvVar: cfg.Project.EnvVar}
	svc := memory.NewService(st, cfg, memory.NewEventBus(), llm, emb)
	return handler.NewHandler(svc, cfg, &gitDetector{det: det}), nil
}

// gitDetector adapts the project.Detector onto the handler's Detector
// seam (the production binding: explicit > env > git toplevel >
// basename, at call time).
type gitDetector struct{ det project.Detector }

func (g *gitDetector) Resolve(cwd, explicit string) (string, error) {
	return g.det.Resolve(cwd, explicit, gitToplevel)
}

// dialectFor maps the config's dialect onto the store's dialect constant.
func dialectFor(cfg config.Config) store.Dialect { return cfg.Datasource.Dialect }

// run is the CLI's entry: parse the top-level command, dispatch, and map
// the error to its exit code (usage → 2, handler → 1).
func run(args []string) int {
	if len(args) == 0 {
		printUsage()
		return exitOK
	}
	// Global flags (--data-dir / --datasource / --dsn) may precede the
	// command; parse them off first so every command sees them.
	var fl config.Flags
	global := flag.NewFlagSet("mnemonic", flag.ContinueOnError)
	global.SetOutput(os.Stderr)
	global.StringVar(&fl.DataDir, "data-dir", "", "store data dir (beats "+config.EnvDataDir+")")
	global.StringVar(&fl.Dialect, "datasource", "", "store dialect: sqlite|postgres")
	global.StringVar(&fl.DSN, "dsn", "", "store DSN (postgres)")
	var vVersion bool
	global.BoolVar(&vVersion, "version", false, "print version and exit")
	global.BoolVar(&vVersion, "v", false, "shorthand for --version")
	rest, vVersion := stripGlobals(args, &fl)
	if vVersion {
		fmt.Fprintln(outW(), "mnemonic", version)
		return exitOK
	}
	if len(rest) == 0 {
		printUsage()
		return exitOK
	}
	cwd, _ := os.Getwd()
	cmd, args := rest[0], rest[1:]
	switch cmd {
	case "mcp":
		return runMCP(fl, args)
	case "serve":
		return runServe(fl, args)
	case "mem":
		return runMem(fl, cwd, args)
	case "sync":
		return runSync(fl, cwd, args)
	case "trail":
		return runTrail(fl, cwd, args)
	case "doctor":
		return runDoctor(fl, cwd, args)
	case "index":
		fmt.Fprintln(outW(), "mnemonic index delegates to the code-index change (not yet in this build).")
		return exitOK
	case "migrate":
		fmt.Fprintln(outW(), "mnemonic migrate delegates to the v1→v2 seed migration (TICKET-19, not yet in this build).")
		return exitOK
	case "setup":
		fmt.Fprintln(outW(), "mnemonic setup delegates to the agent-plugin change (not yet in this build).")
		return exitOK
	case "help", "--help", "-h":
		printUsage()
		return exitOK
	default:
		fmt.Fprintf(errW(), "error: unknown command %q (see --help)\n", cmd)
		return exitUsage
	}
}

// stripGlobals removes the global flags (and their values) from args,
// filling fl and reporting whether --version was set. The value-carrying
// globals come in both `--flag value` and `--flag=value` forms; a
// command positional (a non-flag next-arg) is never consumed as a value.
func stripGlobals(args []string, fl *config.Flags) ([]string, bool) {
	target := map[string]*string{
		"--data-dir":   &fl.DataDir,
		"--datasource": &fl.Dialect,
		"--dsn":        &fl.DSN,
	}
	out := make([]string, 0, len(args))
	vVersion := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		name, val, hasVal := a, "", false
		if eq := strings.IndexByte(a, '='); eq > 0 && strings.HasPrefix(a, "-") {
			name, val, hasVal = a[:eq], a[eq+1:], true
		}
		switch name {
		case "--version", "-v":
			vVersion = true
		case "--data-dir", "--datasource", "--dsn":
			if !hasVal && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				val = args[i]
			}
			*target[name] = val
		default:
			out = append(out, a)
		}
	}
	return out, vVersion
}

// jsonMarshal / jsonUnmarshal are the encoding/json re-exports (the
// commands' JSON needs are small; the envelope owns the wire shape).
var (
	jsonMarshal   = json.Marshal
	jsonUnmarshal = json.Unmarshal
)

// stdoutPipe / stderrPipe are the output seams the commands write to
// (the production wiring is os.Stdout/os.Stderr; the test fixture swaps
// them for buffers — os.Stdout is an *os.File and cannot be reassigned,
// so the one indirection is the test's capture seam).
var (
	stdoutPipe io.Writer = os.Stdout
	stderrPipe io.Writer = os.Stderr
)

// outW / errW are the writers the commands' Fprintln/Fprintf calls use
// (a one-line seam over stdoutPipe/stderrPipe).
func outW() io.Writer { return stdoutPipe }
func errW() io.Writer { return stderrPipe }

// main wires the stdlib flag dispatch onto run (the v1 entry shape).
func main() {
	os.Exit(run(os.Args[1:]))
}

// printUsage renders the command table (the v1 help shape, trimmed to
// the memory command surface).
func printUsage() {
	out := outW()
	fmt.Fprintln(out, `mnemonic — local-first memory (v2)

Usage:
  mnemonic <command> [flags]

Commands:
  mcp           Run the MCP stdio server (starts autosync when sync.auto=true)
  serve         (deprecated in v2 — the HTTP dashboard is dropped; see mcp)
  mem           Memory tools (save|search|get|lookup|context|review|timeline|pin|unpin|delete|session|stats|doctor)
  sync          Sync (status|push|pull|export|import|enroll)
  trail         Inspect retrieval trails (recent|show)
  doctor        Whole-system health (--strict for CI)
  index         (delegates to the code-index change)
  migrate       (delegates to the v1→v2 seed migration)
  setup <agent> (delegates to the agent-plugin change)
  help          Show this help

Global flags (before the command):
  --data-dir DIR    Store data dir
  --datasource D    Store dialect: sqlite|postgres
  --dsn DSN         Store DSN (postgres)
  --version         Print version and exit`)
}

// dispatchResult is the (result, err) a command returns: the result is
// printed (human or --json) and the err is mapped to an exit code.
// A nil result with a nil err is a no-op success (the command printed).
type dispatchResult struct {
	result any
	err    error
}

// finish maps a command's (result, err) onto the output + exit code: a
// usage error prints the message to stderr and exits 2; a handler error
// prints the envelope's (code, message) and exits 1; a result is printed
// (human-readable by default, the envelope JSON with --json).
func finish(fs *flag.FlagSet, jsonOut bool, r dispatchResult) int {
	if r.err != nil {
		if isUsageError(r.err) {
			fmt.Fprintf(outW(), "error: %s\n", r.err.Error())
			return exitUsage
		}
		code, msg := envelope.MapError(r.err)
		if jsonOut {
			fmt.Fprintln(outW(), mustMarshal(envelopeError(code, msg)))
		} else {
			fmt.Fprintf(outW(), "error: [%s] %s\n", code, msg)
		}
		return exitErr
	}
	if r.result == nil {
		return exitOK
	}
	if jsonOut {
		// The K6 seam: the --json output IS the envelope (the same
		// marshal the MCP tools use — the byte-identical contract).
		degraded, reason := envelope.ExtractDegraded(r.result)
		env, err := envelope.WithDegraded(r.result, degraded, reason)
		if err != nil {
			fmt.Fprintf(outW(), "error: [internal_error] %s\n", err.Error())
			return exitErr
		}
		fmt.Fprintln(outW(), mustMarshal(env))
		return exitOK
	}
	printHuman(r.result)
	return exitOK
}

// envelopeError is the error envelope (the MCP toolError shape: the same
// keys, so the --json error output matches the MCP surface — the K6
// contract covers errors too).
func envelopeError(code, message string) map[string]any {
	return map[string]any{"error": true, "code": code, "message": message}
}

// mustMarshal renders the envelope as JSON (a marshal failure is a
// wiring bug — the envelope's value types are all JSON-marshalable).
func mustMarshal(v any) string {
	b, err := jsonMarshal(v)
	if err != nil {
		return `{"error":true,"code":"internal_error","message":"envelope marshal failed"}`
	}
	return string(b)
}

// printHuman renders a result as human-readable text (the default
// output; the --json path renders the envelope instead).
func printHuman(v any) {
	switch r := v.(type) {
	case string:
		fmt.Fprintln(outW(), r)
	case map[string]any:
		for _, line := range humanLines(r) {
			fmt.Fprintln(outW(), line)
		}
	default:
		fmt.Fprintln(outW(), mustMarshal(v))
	}
}

// humanLines renders one human line per top-level key (a stable, readable
// summary; the full record is --json's job).
func humanLines(m map[string]any) []string {
	var out []string
	for k, v := range m {
		if k == "degraded_reason" {
			continue // folded into the degraded line
		}
		out = append(out, fmt.Sprintf("%s: %s", k, humanValue(v)))
	}
	return out
}

// humanValue renders one value (a scalar or a compact count).
func humanValue(v any) string {
	switch t := v.(type) {
	case []any:
		return fmt.Sprintf("%d items", len(t))
	case map[string]any:
		return fmt.Sprintf("%d keys", len(t))
	case bool:
		if t {
			return "yes"
		}
		return "no"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// gitOutput runs a git subcommand and returns its stdout (the detector's
// toplevel binding — a single shell-out, no streaming).
func gitOutput(dir string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", dir}, args...)
	out, err := runGit(cmdArgs)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// runGit runs a git command and returns its stdout (the detector's
// toplevel binding — a single shell-out, no streaming).
func runGit(args []string) ([]byte, error) {
	return exec.Command("git", args...).Output()
}
