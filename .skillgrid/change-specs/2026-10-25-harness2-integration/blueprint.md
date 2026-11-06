# Harness² Host Integration — Implementation Blueprint (steps, files, verification)

> **Role:** this is the *how* — the implementation plan: concrete steps, files to touch, code, tests, and verification. The companion `briefing.md` is the *what*. The product *why* lives in `.skillgrid/knowledge/reference/00-prd.md`; the decisions are ADR-0050 + the in-force set in `.skillgrid/knowledge/ASSUMPTIONS.md` § LOCKED. This blueprint is pure-technical and cites them.
>
> Assume the engineer has zero context for the codebase. Document every file to touch, the code, the tests, and how to verify. DRY. YAGNI. TDD. Frequent commits. No placeholders.

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED
**Critic:** skipped (T2 fast-path — the change is a known-shape external-CLI host; no one-way-door; see Self-Review)
**Tier:** T2
**Build shape:** Tracer thread — the thinnest end-to-end path (Go parses one fixture `runs/` tree → CLI `list` prints it → HTTP `GET /harness/runs` returns it → SPA renders it) is built first, then thickens with the spawn path, judge/report, and the write routes.

**Goal:** skillgrid can install, launch, monitor, and surface judged results of Harness² runs through the existing CLI, REST, and embedded SPA — hosting the external Python CLI without adding a Go dependency.

**Architecture:** a new `mnemonic/internal/harness/` package owns three concerns: (1) a defensive reader over the external `runs/` file contract, (2) a fixed-argv, env-allow-listed, timeout-bounded `os/exec` wrapper around the `harness2-*` CLI, and (3) a capability preflight (`doctor`-style). A `mnemonic harness …` subcommand group and read-heavy REST routes are thin over that package. The installer gets Harness² as a first-class `ExternalTool` (rename of `SecTool`). No Go module dependency; no schema change; no CGo.

**Tech Stack:** Go 1.22+ stdlib (`os/exec`, `encoding/json`, `os`, `path/filepath`); existing `mark3labs/mcp-go`/`modernc.org/sqlite` untouched. Python 3.12/3.13 Harness² (external, `uv`-managed). SPA: Vite + React 19 + TS + Tailwind 4 (existing `skillgrid-ui/`).

**Spec:** `./briefing.md` (requirements & intent) · **ADR:** `.skillgrid/knowledge/decisions/04-adr-0050-harness2-host-integration.md` · **ADR manifest:** `./adr.md`

## Terms

- **Harness² / harness2** — the external Python benchmarking + harness-optimization engine (arXiv:2610.14224). A *machine-level* dependency; not a Go package. `.skillgrid/knowledge/glossary/02-technical-terms.md` (add entry: "Harness² — external agent-harness optimization engine, hosted as a subprocess").
- **runs tree** — the file contract Harness² writes under `HARNESS2_RUNS_DIR`: `config.json` (run identity), task summaries, `judged.json`/`base_judged.json`, adopted harnesses. Read-only to skillgrid.
- **host, not subject** — skillgrid runs and observes Harness²; it does not feed its own memory/skills into Harness². (Sharpened in this plan; see briefing § Design Decisions 1.)
- **ExternalTool** — the installer struct (renamed from `SecTool`) describing an `uv`/`go`-managed external CLI. `.skillgrid/knowledge/glossary/02-technical-terms.md` (add entry).

---

## Must-Haves (goal-backward verification)

**Truths** (observable behaviors that must hold):
- `mnemonic harness list` prints a table of runs parsed from `HARNESS2_RUNS_DIR`; with no `runs/` dir it prints "no runs" and exits 0.
- `mnemonic harness status <id>` prints one run's identity, task count, and judge verdict (if present); unknown id → non-zero exit with a message naming the id.
- `GET /harness/runs` returns `200` + a JSON array of runs (empty array when no runs); `GET /harness/runs/{id}` returns `200` for a known id and `404` for an unknown one.
- A **malformed** `judged.json` (truncated JSON) does not crash the server: that run is returned with `parse_status: "unparsed"` and the rest of the list is intact.
- `POST /harness/runs` without a `SKILLGRID_HTTP_TOKEN` returns `401`; with the token it spawns `harness2` and returns `202` with the run id.
- `mnemonic harness run --dry-run` prints the exact argv it *would* exec and does not spawn a process.
- The installer, when `harness2` is not on PATH and `uv` is present, runs `uv tool install harness2` (or the pinned equivalent) and reports it; when `uv` is missing it warns + skips (non-fatal).
- **backstop** — on a machine with `uv` + Node 20 + `harness2` installed, `mnemonic harness run` (offline `harness2-demo`) spawns the child, captures stdout/stderr into session/traffic logging, and a subsequent `GET /harness/runs` shows the finished run with a `judged.json` parsed. (Held-out: env-gated integration test `TestHarnessDemoEndToEnd`.)

**Artifacts** (files that must exist with real implementation, not stubs):
- `mnemonic/internal/harness/runs.go` — defensive `runs/` file-contract reader (structs + `fs` reads, no Python).
- `mnemonic/internal/harness/runner.go` — `os/exec` wrapper (fixed argv, env allow-list, timeout, stdout/err capture).
- `mnemonic/internal/harness/preflight.go` — capability detection (`harness2`/`uv`/`node`/`gcloud`) returning a `doctor`-style report.
- `mnemonic/internal/harness/config.go` — `HARNESS2_*` env mapping + `runs` root resolution (`~/.skillgrid/mnemonic/harness2/runs/`).
- `mnemonic/cmd/mnemonic/harness.go` (+ `harness_run.go`, `harness_list.go`, `harness_status.go`, `harness_judge.go`, `harness_report.go`, `harness_setup.go`, `harness_check.go`) — the `mnemonic harness …` subcommand group.
- `mnemonic/internal/http/harness.go` — the `/harness/*` REST handlers.
- `skillgrid-cli/internal/install/config.go` + `install.go` — `SecTool`→`ExternalTool` rename + `HarnessTools()` list + `installHarnessTools()`.
- `skillgrid-ui/src/pages/HarnessPage.tsx` (+ route + nav entry) — the SPA runs view.
- Tests: `mnemonic/internal/harness/*_test.go`, `mnemonic/internal/http/harness_test.go`, `skillgrid-cli/internal/install/install_test.go` (new `TestHarnessTools`).

**Key links** (critical connections that must work together):
- `cmd/mnemonic/harness_list.go` → `harness.ListRuns(root)` → the JSON served by `internal/http` `GET /harness/runs` → the SPA table. One parser, three surfaces.
- `internal/http` `POST /harness/runs` → `harness.Runner.Run(ctx, …)` → `os/exec` of `harness2` with `HARNESS2_RUNS_DIR` set → the child writes `runs/` → `ListRuns` reads it back. The run id returned by the POST is the same id `GET /harness/runs/{id}` resolves.
- `installHarnessTools()` uses the *same* `ExternalTool` struct as `installSecurityTools()` — the rename is a single shared type, so both lists compile against it.

**One-way-door decisions** (hard to reverse — flag for explicit user approval before implementing):
- **Public REST contract addition** — `POST /harness/runs` and `POST /harness/runs/{id}/judge` are new write routes with a response shape (`202 {run_id}`) that, once consumed by the SPA, is a semi-public contract. Additive (no existing route touched), so reversible by removing the routes, but the SPA couples to it. Flagged on **Task 5**.
- *No migration. No existing route, type, or config key changed in a breaking way.* The `SecTool`→`ExternalTool` rename is mechanical and internal (not exported across a module boundary that matters) — see Task 1.

---

## Global Constraints

- **Go 1.22+ floor** (ADR-0031). `go build ./...` and `go test ./...` must pass.
- **No new Go module dependency** (ADR-0008 rule; ADR-0050). `mnemonic/go.mod` gains zero new direct deps. Harness² is a machine-level Python runtime, not a `go.mod` entry.
- **cgo-free invariant** preserved — pure-Go `modernc.org/sqlite` untouched; no CGo bridge.
- **Conventional commits only**; no AI-attribution trailers (commit-msg hook).
- **Spec-zone before code-zone** (pre-commit zone guard): `briefing.md`, `blueprint.md`, `adr.md`, the ADR file, and any `.skillgrid/config.yaml` / glossary edit commit *before* Go/UI code.
- **Fixed-argv exec** — never build a shell string; always `exec.Command(bin, args...)`.
- **Env allow-list** for the child — only `HARNESS2_*` + `PATH` are set/inherited for the spawned child; no other secret passthrough.
- **HTTP write routes** gated by `SKILLGRID_HTTP_TOKEN` via the existing `requireWriteAuth` middleware.
- `gofmt -w ./...` clean; `go vet ./...` clean.

---

## File Structure

- `mnemonic/internal/harness/config.go` — resolve `HARNESS2_RUNS_DIR` (default `~/.skillgrid/mnemonic/harness2/runs/`), build the `HARNESS2_*` env map, locate the `harness2` binary.
- `mnemonic/internal/harness/runs.go` — `Run`/`RunSummary` structs + `ListRuns(root)`, `GetRun(root, id)`, defensive `judged.json` parse.
- `mnemonic/internal/harness/runner.go` — `Runner` with `Run(ctx, RunReq) (RunResult, error)`: fixed argv, env allow-list, timeout, combined stdout/stderr capture.
- `mnemonic/internal/harness/preflight.go` — `Preflight() Report` detecting `harness2`/`uv`/`node`/`gcloud` presence + versions.
- `mnemonic/internal/harness/*_test.go` — unit tests (fixture `runs/` trees, argv/env builders, malformed-JSON cases).
- `mnemonic/cmd/mnemonic/harness.go` — group dispatch for `mnemonic harness …`.
- `mnemonic/cmd/mnemonic/harness_{run,list,status,judge,report,setup,check}.go` — one file per subcommand (stdlib `flag`).
- `mnemonic/internal/http/harness.go` — `registerHarnessRoutes` + handlers (`handleHarnessRuns`, `handleHarnessRunGet`, `handleHarnessRunCreate`, `handleHarnessRunJudge`).
- `mnemonic/internal/http/server.go` — mount `registerHarnessRoutes` (modify).
- `skillgrid-cli/internal/install/config.go` — rename `SecTool`→`ExternalTool`, add `HarnessTools() []ExternalTool`.
- `skillgrid-cli/internal/install/install.go` — `installHarnessTools(c)` + call in `Run`; re-point `installSecurityTools` at `SecurityTools()` returning `[]ExternalTool`.
- `skillgrid-cli/internal/install/install_test.go` — `TestHarnessTools`.
- `skillgrid-ui/src/pages/HarnessPage.tsx` — SPA view; route + nav entry in the existing router.
- `.skillgrid/config.yaml` — optional `harness:` block (model vars, runs dir override) — spec zone.
- `.skillgrid/knowledge/glossary/02-technical-terms.md` — add "Harness²" + "ExternalTool" terms.

---

## Threat Matrix

Touched boundaries: subprocess (process integration), external file contract (data ingest), new REST write routes (routing/auth).

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| **Subprocess: fixed-argv exec, no shell injection** | Applicable — Go spawns an external binary with user-influenced args (run id, model name). | `exec.Command(bin, args...)` with a validated argv slice; every user-supplied value is a discrete arg, never concatenated into a string; no `sh -c`. | `TestRunnerNoShellInjection`: a run-id containing `; touch /tmp/pwned` becomes a single argv element; assert the exec'd argv has N elements and no element contains a space-joined shell token; assert `/tmp/pwned` is not created. |
| **Subprocess: env allow-list (no secret leakage)** | Applicable — the child inherits the server env. | Build a fresh `os/exec.Cmd.Env` from `PATH` + only `HARNESS2_*` vars; never `os.Environ()` passthrough. | `TestRunnerEnvAllowList`: set a sentinel `SKILLGRID_HTTP_TOKEN=secret` in the test env; after `Runner` builds the child env, assert `SKILLGRID_HTTP_TOKEN` is **absent** and `HARNESS2_SOLVER_MODEL` is present. |
| **Subprocess: timeout + non-hang** | Applicable — a hung `harness2` must not block the server. | `context.WithTimeout` on `exec.CommandContext`; on timeout, kill the process group and return a classified error. | `TestRunnerTimeout`: spawn a fixture binary that sleeps 10s with a 50ms timeout; assert the call returns well before 10s with a `timeout`-classified error and no leaked process. |
| **File contract: malformed / hostile `runs/` JSON** | Applicable — skillgrid reads a tree an external process wrote. | `encoding/json` into typed structs with `parse_status` tracking; unknown fields ignored; a bad file marks that run `unparsed` and never returns an error that 500s the server. | `TestGetRunMalformedJudged`: write a truncated `judged.json`; `GetRun` returns the run with `parse_status:"unparsed"` and no error; `TestListRunsIgnoresUnknownFields`: extra keys are dropped, the run still lists. |
| **File contract: run-identity validation before resume** | Applicable — `POST /harness/runs` must not resume an incompatible run (Harness²'s own `config.json` guard). | `GetRun` reads `config.json`; the `Runner` sets `HARNESS2_RUNS_DIR` to a fresh per-run subdir unless `--resume <id>` is given and that id's `config.json` parses. | `TestRunIdentityMismatch`: two runs with different `config.json` `namespace`; launching a resume of the wrong id returns a classified `identity-mismatch` error, not a silent merge. |
| **REST write routes: auth** | Applicable — `POST /harness/runs`, `POST /harness/runs/{id}/judge` spawn a process. | Wrap with `requireWriteAuth` (`SKILLGRID_HTTP_TOKEN`), same as every other write route. | `TestHarnessWriteRequiresToken`: `POST /harness/runs` with no token → `401`; with token → `202`. |
| **Path traversal in run id** | Applicable — `GET /harness/runs/{id}` builds a path from the id. | Validate the id against `^[a-zA-Z0-9._-]+$` before `filepath.Join`; reject `..`. | `TestGetRunRejectsTraversal`: id `../secret` → `400` (not a path read outside `runs/`). |

---

## File Structure → Task map

Task 1 (config) → Task 2 (runs parser, tracer core) → Task 3 (CLI list/status, tracer surface) → Task 4 (runner + preflight) → Task 5 (CLI run/judge/report/setup/check + HTTP routes) → Task 6 (SPA) → Task 7 (installer) → Task 8 (env-gated E2E) → Task 9 (docs/config/glossary spec-zone).

---

### Task 1: `harness` config + runs root resolution

**Files:**
- Create: `mnemonic/internal/harness/config.go`
- Test: `mnemonic/internal/harness/config_test.go`

**Interfaces:**
- Consumes: stdlib only.
- Produces: `func RunsRoot(override string) (string, error)` (returns `~/.skillgrid/mnemonic/harness2/runs/` when `override==""`, else `override`); `func EnvMap(cfg HarnessConfig) []string` (the `HARNESS2_*` + `PATH` allow-list, exact `KEY=VALUE` slice); `type HarnessConfig struct { RunsDir, SolverModel, ImproverModel, JudgeModel, CodexBaseURL, CodexAPIKey, CodexModel string }`; `func LoadHarnessConfig() HarnessConfig` (reads `HARNESS2_*` env, falls back to config.d `harness:` block if present, else defaults).
- Seam: config resolution — behavior can be altered (e.g., a future per-project runs dir) by changing `RunsRoot` without touching the parser or runner.
- Deletion test: the runner and CLI both need a runs root + env; deleting this re-scatters `~/.skillgrid` path logic across both.
- Adapters: prod (env/config.d) + test (injected override) = 2.

**SATISFIES:** `harness-list-no-runs` (the `runs/` root is resolved before the list can say "no runs").

- [ ] **Step 1: Write the failing test**

```go
func TestRunsRootDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	root, err := RunsRoot("")
	if err != nil { t.Fatal(err) }
	want := filepath.Join(os.Getenv("HOME"), ".skillgrid/mnemonic/harness2/runs")
	if root != want { t.Fatalf("got %q want %q", root, want) }
}

func TestRunsRootOverride(t *testing.T) {
	root, err := RunsRoot("/tmp/x/runs")
	if err != nil { t.Fatal(err) }
	if root != "/tmp/x/runs" { t.Fatalf("got %q", root) }
}

func TestEnvMapAllowList(t *testing.T) {
	cfg := HarnessConfig{SolverModel: "m1", RunsDir: "/r"}
	got := EnvMap(cfg)
	// assert HARNESS2_SOLVER_MODEL=m1 present, HARNESS2_RUNS_DIR=/r present,
	// and no SKILLGRID_* key present.
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./mnemonic/internal/harness/ -run TestRunsRoot` → FAIL (undefined).
- [ ] **Step 3: Write minimal implementation** — `config.go` per the Interfaces above.
- [ ] **Step 4: Run test to verify it passes** — `go test ./mnemonic/internal/harness/` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(harness): add config + runs root resolution"` (spec-zone files already committed first).

### Task 2: `runs/` file-contract reader (tracer core)

**Files:**
- Create: `mnemonic/internal/harness/runs.go`
- Create: `mnemonic/internal/harness/testdata/` (fixture `runs/` trees: `good/`, `malformed/`, `traversal/`)
- Test: `mnemonic/internal/harness/runs_test.go`

**Interfaces:**
- Consumes: `RunsRoot` from Task 1; stdlib `os`, `path/filepath`, `encoding/json`, `regexp`.
- Produces:
  - `type RunSummary struct { ID string; Namespace string; Status string; ParseStatus string; TaskCount int; Score float64; BaseScore float64; Delta float64; HasJudged bool; CreatedAt string }`
  - `type Run struct { Summary RunSummary; Config map[string]any; Tasks []Task; Adopted []string }`
  - `type Task struct { ID string; Prompt string; Delivered string; Score float64; Verdict string }`
  - `func ListRuns(root string) ([]RunSummary, error)` — `root` missing → `([]RunSummary{}, nil)`.
  - `func GetRun(root string, id string) (*Run, error)` — unknown id → `nil, ErrNotFound`; traversal id → `nil, ErrBadID`; malformed `judged.json` → run with `ParseStatus:"unparsed"`, no error.
  - Sentinel errors: `var ErrNotFound = errors.New("run not found")`, `var ErrBadID = errors.New("invalid run id")`.
  - `const runIDPattern = "^[a-zA-Z0-9._-]+$"`.
- Seam: the file-contract boundary — a future Harness² version changing `judged.json` shape is absorbed here, not in the CLI/HTTP.
- Deletion test: CLI `list`/`status` and both HTTP GETs all call this; deleting it re-parses JSON in three places.
- Adapters: prod (real `runs/`) + test (fixture trees) = 2.

**SATISFIES:** `harness-list-no-runs`, `harness-list-shows-runs`, `harness-status-shows-run`, `harness-malformed-judged-does-not-crash`, `harness-reject-traversal-id`.

- [ ] **Step 1: Write the failing test** (fixture-driven)

```go
func TestListRunsMissingRoot(t *testing.T) {
	sums, err := ListRuns(filepath.Join(t.TempDir(), "absent"))
	if err != nil { t.Fatal(err) }
	if len(sums) != 0 { t.Fatalf("want empty, got %d", len(sums)) }
}

func TestGetRunMalformedJudged(t *testing.T) {
	root := t.TempDir()
	mkRun(t, root, "r1", `{"namespace":"ns"}`)          // valid config.json
	os.WriteFile(filepath.Join(root,"r1","judged.json"), []byte(`{"tasks": [`), 0o644) // truncated
	run, err := GetRun(root, "r1")
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	if run.Summary.ParseStatus != "unparsed" { t.Fatalf("got %q", run.Summary.ParseStatus) }
}

func TestGetRunRejectsTraversal(t *testing.T) {
	_, err := GetRun(t.TempDir(), "../secret")
	if !errors.Is(err, ErrBadID) { t.Fatalf("want ErrBadID, got %v", err) }
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./mnemonic/internal/harness/ -run TestGetRun` → FAIL (undefined).
- [ ] **Step 3: Write minimal implementation** — `runs.go`: walk `root` for run subdirs, read `config.json` (→ `Namespace`, `Status`), read `judged.json`/`base_judged.json` defensively (set `Score`, `BaseScore`, `Delta`, `TaskCount`, `ParseStatus`). Validate `id` against `runIDPattern` before `filepath.Join`.
- [ ] **Step 4: Run test to verify it passes** — `go test ./mnemonic/internal/harness/ -run TestGetRun` → PASS; then `go test ./mnemonic/internal/harness/` → all PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(harness): add defensive runs/ file-contract reader"`.

### Task 3: `mnemonic harness list|status` (tracer surface)

**Files:**
- Create: `mnemonic/cmd/mnemonic/harness.go` (group dispatch)
- Create: `mnemonic/cmd/mnemonic/harness_list.go`
- Create: `mnemonic/cmd/mnemonic/harness_status.go`
- Modify: `mnemonic/cmd/mnemonic/main.go` (register the `harness` group — match how `ctx` is registered)
- Test: `mnemonic/cmd/mnemonic/harness_list_test.go`

**Interfaces:**
- Consumes: `ListRuns`, `GetRun`, `RunsRoot`, `ErrNotFound`, `ErrBadID` from Task 2; stdlib `flag`, `fmt`, `os`.
- Produces: `func runHarnessList(args []string) int`, `func runHarnessStatus(args []string) int` (exit-code return, matching the existing `cmd/mnemonic` convention).
- Seam: CLI surface — the parsing/printing is here, the data comes from Task 2.
- Deletion test: none (leaf commands).
- Adapters: none (in-process).

**SATISFIES:** `harness-list-no-runs`, `harness-list-shows-runs`, `harness-status-shows-run`, `harness-status-unknown-id`.

- [ ] **Step 1: Write the failing test**

```go
func TestRunHarnessListNoRuns(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HARNESS2_RUNS_DIR", filepath.Join(dir, "absent"))
	// capture stdout, call runHarnessList(nil); assert exit 0 and "no runs"
}

func TestRunHarnessStatusUnknownID(t *testing.T) {
	t.Setenv("HARNESS2_RUNS_DIR", t.TempDir())
	code := runHarnessStatus([]string{"nope"})
	if code == 0 { t.Fatal("want non-zero exit for unknown id") }
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./mnemonic/cmd/mnemonic/ -run TestRunHarnessList` → FAIL (undefined).
- [ ] **Step 3: Write minimal implementation** — `harness.go` dispatches `list`/`status`/`run`/`judge`/`report`/`setup`/`check`; `harness_list.go` calls `ListRuns(RunsRoot(...))` and prints a table (`ID  NAMESPACE  STATUS  PARSE  TASKS  SCORE  Δ`); `harness_status.go` calls `GetRun`, prints identity + task table, maps `ErrNotFound`/`ErrBadID` to non-zero exit with a message naming the id.
- [ ] **Step 4: Run test to verify it passes** — `go test ./mnemonic/cmd/mnemonic/ -run TestRunHarness` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(harness): add mnemonic harness list/status commands"`.

### Task 4: `runner` (os/exec wrapper) + `preflight`

**Files:**
- Create: `mnemonic/internal/harness/runner.go`
- Create: `mnemonic/internal/harness/preflight.go`
- Test: `mnemonic/internal/harness/runner_test.go`, `mnemonic/internal/harness/preflight_test.go`

**Interfaces:**
- Consumes: `HarnessConfig`, `EnvMap`, `RunsRoot` from Task 1; stdlib `os/exec`, `context`, `time`, `os`.
- Produces:
  - `type RunReq struct { Command string; // "run"|"judge"|"report"|"setup"|"check"|"demo"`; `Args []string; // validated extra args`; `Timeout time.Duration; `ResumeID string` }`
  - `type RunResult struct { ExitCode int; Stdout string; Stderr string; TimedOut bool; IdentityMismatch bool; RunID string }`
  - `type Runner struct { Bin string; Env func(HarnessConfig) []string; Now func() time.Time }`
  - `func NewRunner(cfg HarnessConfig) *Runner`
  - `func (r *Runner) Run(ctx context.Context, req RunReq) (RunResult, error)` — `exec.CommandContext`; fixed argv; env = `PATH` + `HARNESS2_*` only; on timeout → `TimedOut:true`; on `config.json` mismatch (resume) → `IdentityMismatch:true`.
  - `type Report struct { Harness2, UV, Node, GCloud bool; Versions map[string]string; Notes []string }`
  - `func Preflight() Report` — `exec.LookPath` for `harness2`/`uv`/`node`/`gcloud`, `--version` where cheap.
- Seam: the process boundary — argv/env/timeout policy lives here and is unit-testable with a fixture binary.
- Deletion test: CLI `run/judge/report/setup/check` and the HTTP POST both spawn through this; deleting it re-implements `exec.CommandContext` + env-building in each.
- Adapters: prod (real `harness2`) + test (fixture shell binary) = 2.

**SATISFIES:** `harness-run-dry-run-prints-argv`, `harness-no-shell-injection`, `harness-env-allow-list`, `harness-timeout-classified`, `harness-preflight-reports`.

- [ ] **Step 1: Write the failing test** (fixture binary + the threat-matrix RED tests)

```go
func TestRunnerNoShellInjection(t *testing.T) {
	r := NewRunner(HarnessConfig{RunsDir: t.TempDir()})
	res, err := r.Run(context.Background(), RunReq{Command:"demo", Args: []string{"-n", `; touch /tmp/pwned`}})
	_ = res; _ = err
	// assert the exec'd argv had the run token as ONE element (capture via a
	// test binary that echoes argv length), and /tmp/pwned does not exist.
}

func TestRunnerEnvAllowList(t *testing.T) {
	t.Setenv("SKILLGRID_HTTP_TOKEN", "secret")
	r := NewRunner(HarnessConfig{SolverModel: "m1", RunsDir: t.TempDir()})
	// build child env via r.Env; assert SKILLGRID_HTTP_TOKEN absent, HARNESS2_SOLVER_MODEL present
}

func TestRunnerTimeout(t *testing.T) {
	r := NewRunner(HarnessConfig{RunsDir: t.TempDir()})
	start := time.Now()
	_, _ = r.Run(context.Background(), RunReq{Command:"demo", Args: []string{"--sleep", "10"}, Timeout: 50 * time.Millisecond})
	if time.Since(start) > 5*time.Second { t.Fatal("did not time out promptly") }
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./mnemonic/internal/harness/ -run TestRunner` → FAIL (undefined).
- [ ] **Step 3: Write minimal implementation** — `runner.go` per Interfaces; the fixture binary for tests is a tiny shell script under `testdata/` that echoes its argv length and sleeps when `--sleep`. `preflight.go` does `exec.LookPath` + `--version`.
- [ ] **Step 4: Run test to verify it passes** — `go test ./mnemonic/internal/harness/ -run 'TestRunner|TestPreflight'` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(harness): add exec runner (fixed argv, env allow-list, timeout) + preflight"`.

### Task 5: CLI `run/judge/report/setup/check` + REST routes

> ⚠ one-way: **new public REST write contract** — `POST /harness/runs` and `POST /harness/runs/{id}/judge` establish a semi-public response shape (`202 {run_id}`) the SPA couples to. Additive, but confirm the shape before the SPA (Task 6) builds on it. **STOP for user approval before Task 6.**

**Files:**
- Create: `mnemonic/cmd/mnemonic/harness_run.go`, `harness_judge.go`, `harness_report.go`, `harness_setup.go`, `harness_check.go`
- Create: `mnemonic/internal/http/harness.go`
- Modify: `mnemonic/internal/http/server.go` (call `s.registerHarnessRoutes()` alongside the other `register*Routes` calls)
- Test: `mnemonic/internal/http/harness_test.go`

**Interfaces:**
- Consumes: `Runner`, `RunReq`, `Preflight` (Task 4); `ListRuns`, `GetRun`, `RunsRoot` (Task 2); the server's `requireWriteAuth` + mux.
- Produces:
  - CLI: `runHarnessRun`, `runHarnessJudge`, `runHarnessReport`, `runHarnessSetup`, `runHarnessCheck` (each → `int`). `--dry-run` on `run` prints argv and returns 0 without exec.
  - HTTP handlers: `handleHarnessRuns` (GET list), `handleHarnessRunGet` (GET one), `handleHarnessRunCreate` (POST launch, `202` + `{"run_id": …}`), `handleHarnessRunJudge` (POST judge). Routes: `GET /harness/runs`, `GET /harness/runs/{id}`, `POST /harness/runs`, `POST /harness/runs/{id}/judge`. POSTs wrapped in `requireWriteAuth`.
  - **Launch semantics (decided): detached spawn.** `handleHarnessRunCreate` mints the run id as a timestamped slug (`<UTC-stamp>-<rand4>`), creates `runs/<run_id>/`, spawns `harness2` as a detached child with `HARNESS2_RUNS_DIR=…/runs/<run_id>/` (and `Setpgid` so the timeout kills the group), and returns `202 {"run_id": …}` immediately. The sweep continues in the background; the SPA polls `GET /harness/runs/{id}` for progress (the run's `Status`/`HasJudged` flip as Harness² writes the tree). No long-held HTTP request, no terminal block.
  - `func (s *Server) registerHarnessRoutes()`.
- Seam: the HTTP boundary — the spawn is delegated to `Runner`; handlers stay thin.
- Deletion test: none (surface).
- Adapters: none (in-process) for the mux; the spawn is Task 4.

**SATISFIES:** `harness-run-dry-run-prints-argv`, `harness-write-requires-token`, `harness-get-run-404`, `harness-launch-returns-202`.

- [ ] **Step 1: Write the failing test**

```go
func TestHarnessWriteRequiresToken(t *testing.T) {
	s := newTestServer(t, "secret") // helper that sets SKILLGRID_HTTP_TOKEN
	req := httptest.NewRequest("POST", "/harness/runs", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized { t.Fatalf("want 401, got %d", rec.Code) }
}

func TestHarnessGetRun404(t *testing.T) {
	s := newTestServer(t, "")
	t.Setenv("HARNESS2_RUNS_DIR", t.TempDir())
	req := httptest.NewRequest("GET", "/harness/runs/nope", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound { t.Fatalf("want 404, got %d", rec.Code) }
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./mnemonic/internal/http/ -run TestHarness` → FAIL (route not mounted / handler undefined).
- [ ] **Step 3: Write minimal implementation** — `harness.go` handlers + `registerHarnessRoutes`; mount in `server.go`. POST launch (**detached spawn**, decided): validate the request, mint `run_id` = `<UTC-stamp>-<rand4>`, `MkdirAll(runs/<run_id>)`, spawn `harness2` detached with `HARNESS2_RUNS_DIR=…/runs/<run_id>/` + `Setpgid`, write `202 {"run_id": …}` and return without waiting. Judge: validate id (`runIDPattern`), `Runner.Run` with `Command:"judge"`, `ResumeID:id`.
- [ ] **Step 4: Run test to verify it passes** — `go test ./mnemonic/internal/http/ -run TestHarness` → PASS; `go test ./mnemonic/...` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(harness): add run/judge/report/setup/check CLI + /harness/* REST routes"`.

### Task 6: SPA Harness view

**Files:**
- Create: `skillgrid-ui/src/pages/HarnessPage.tsx`
- Modify: `skillgrid-ui/src/` router (add `/harness` route) + nav (add entry) — match the existing route-registration file.
- Test: `skillgrid-ui/src/pages/HarnessPage.test.tsx` (render against a stubbed `GET /harness/runs` fixture)

**Interfaces:**
- Consumes: `GET /harness/runs`, `GET /harness/runs/{id}` (Task 5).
- Produces: `HarnessPage` — run list table (id, namespace, status, parse status, task count, score, Δ), click-through to a detail view (identity, task table with verdicts, adopted-harness list). Reuses existing table/SP components; Tailwind 4 classes.
- Seam: none (view).
- Deletion test: none (leaf page).
- Adapters: none.

**SATISFIES:** `harness-spa-list`, `harness-spa-detail`.

- [ ] **Step 1: Write the failing test** — `HarnessPage.test.tsx` renders with a stubbed fetch returning 2 runs; asserts 2 rows and the Δ column; asserts the `unparsed` badge appears for the malformed run.
- [ ] **Step 2: Run test to verify it fails** — `npx vitest run src/pages/HarnessPage.test.tsx` (or the project's `npm test`) → FAIL (component undefined).
- [ ] **Step 3: Write minimal implementation** — `HarnessPage.tsx` + route + nav.
- [ ] **Step 4: Run test to verify it passes** — the test → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(ui): add Harness runs view"`. (UI is a separate commit from Go per the zone guard; confirm `ui/dist` build is not required for the test to pass.)

### Task 7: Installer — `ExternalTool` rename + `harness2`

**Files:**
- Modify: `skillgrid-cli/internal/install/config.go` (rename `SecTool`→`ExternalTool`; `SecurityTools() []ExternalTool`; add `HarnessTools() []ExternalTool`)
- Modify: `skillgrid-cli/internal/install/install.go` (`installSecurityTools` re-pointed at `[]ExternalTool`; add `installHarnessTools(c)`; call it in `Run`)
- Modify: `skillgrid-cli/internal/install/install_test.go` (existing `TestSecurityTools` updated to the new type; add `TestHarnessTools`)

**Interfaces:**
- Consumes: the existing `installSecurityTools`/`runParallel`/`commandRunner`/`batchLookPath`/`toAny`/`Out` helpers.
- Produces: `type ExternalTool struct { Name, Manager string; InstallArgs []string; Bin string; Hint string }` (was `SecTool`); `func HarnessTools() []ExternalTool` returning `{Name:"harness2", Manager:"uv", InstallArgs:[]string{"tool","install","harness2"}, Bin:"harness2", Hint:"pipx install harness2  # requires Python 3.12+, Node 20+; gcloud for real runs"}`; `func installHarnessTools(c *Config) error`.
- Seam: the installer tool registry — the rename is one shared type both lists use.
- Deletion test: both `installSecurityTools` and `installHarnessTools` call into the shared struct + `runParallel`; the rename is atomic (one commit).
- Adapters: none.

**SATISFIES:** `install-harness2-nonfatal`.

- [ ] **Step 1: Write the failing test**

```go
func TestHarnessTools(t *testing.T) {
	tools := HarnessTools()
	if len(tools) == 0 { t.Fatal("want harness2 tool") }
	if tools[0].Name != "harness2" || tools[0].Manager != "uv" { t.Fatalf("got %+v", tools[0]) }
}

func TestInstallHarnessToolsNonFatal(t *testing.T) {
	// batchLookPath → err for "harness2" and "uv" (both missing)
	// assert installHarnessTools returns nil (warn+skip), not an error
}
```

- [ ] **Step 2: Run test to verify it fails** — `go test ./skillgrid-cli/internal/install/ -run 'TestHarness|TestSecurityTools'` → FAIL (rename not done / `TestHarnessTools` undefined).
- [ ] **Step 3: Write minimal implementation** — rename `SecTool`→`ExternalTool` everywhere in the package (3 functions + refs), add `HarnessTools()`, add `installHarnessTools` mirroring `installSecurityTools`, call it in `Run`.
- [ ] **Step 4: Run test to verify it passes** — `go test ./skillgrid-cli/internal/install/` → PASS.
- [ ] **Step 5: Commit** — `git commit -m "feat(install): host harness2 as an ExternalTool (rename SecTool)"`.

### Task 8: env-gated end-to-end integration test

**Files:**
- Create: `mnemonic/internal/harness/e2e_test.go`

**Interfaces:**
- Consumes: `Runner`, `RunsRoot`, `GetRun`, `Preflight`.
- Produces: `func TestHarnessDemoEndToEnd(t *testing.T)` — gated: `if !Preflight().Harness2 { t.Skip("harness2 not installed") }`. Runs the offline `harness2-demo` via `Runner`, waits for completion, asserts a run appears in `ListRuns` and `GetRun` parses its `judged.json` (`ParseStatus:"ok"`).
- Seam: none (test).
- Deletion test: none.
- Adapters: prod only (skips otherwise).

**SATISFIES:** `harness-e2e-demo` (the `backstop` truth).

- [ ] **Step 1: Write the test** (as above).
- [ ] **Step 2: Run on a machine without harness2** — `go test ./mnemonic/internal/harness/ -run TestHarnessDemoEndToEnd -v` → SKIP (proves the gate works).
- [ ] **Step 3: Run on a machine with harness2** (dev box with `uv tool install harness2`) → PASS.
- [ ] **Step 4: Commit** — `git commit -m "test(harness): add env-gated demo end-to-end"`.

### Task 9: spec-zone — config, glossary, doctor note

**Files:**
- Modify: `.skillgrid/config.yaml` (optional `harness:` block: `runs_dir`, `solver_model`, `improver_model`, `judge_model`)
- Modify: `.skillgrid/knowledge/glossary/02-technical-terms.md` (add "Harness²", "ExternalTool")
- Modify: `mnemonic/cmd/mnemonic/doctor.go` (add a `harness2` capability line to the doctor output via `Preflight()`) — *code-zone but additive, committed after Task 4's preflight exists*

**Interfaces:**
- Consumes: `Preflight` (Task 4).
- Produces: config keys read by `LoadHarnessConfig`; two glossary entries; one doctor line.

**SATISFIES:** `harness-preflight-reports` (doctor line).

- [ ] **Step 1:** Add the `harness:` block to `.skillgrid/config.yaml` + glossary terms (spec zone). Commit spec zone first.
- [ ] **Step 2:** Add the doctor `harness2` line in `doctor.go` using `Preflight()` (code zone).
- [ ] **Step 3: Run** — `go test ./...` and `go build ./...` → PASS.
- [ ] **Step 4: Commit** — `git commit -m "docs(harness): add config block, glossary terms, doctor capability line"`.

---

## Self-Review

**1. Spec coverage.** Briefing decisions mapped: (1) host-not-subject → scope + out-of-scope list; (2) Go-spawns-Python → Task 4 `Runner`; (3) `ExternalTool` install → Task 7; (4) `runs/` under data dir → Task 1 `RunsRoot`; (5) `mnemonic harness …` group → Tasks 3,5; (6) REST routes → Task 5; (7) SPA view → Task 6; (8) threat matrix → Tasks 2,4,5 + the matrix section. No gap.

**2. Must-haves coverage.** Every briefing requirement has a truth/artifact/key-link: install (Task 7 truth), CLI (Tasks 3,5), REST (Task 5), SPA (Task 6), file contract (Task 2), defense (Threat Matrix RED tests). The `backstop` truth has a held-out test (Task 8). The launch-semantics decision (detached spawn, `202` + minted `run_id`, SPA polls for progress) was routed to the user at the blueprint gate and is now pinned in Task 5's Interfaces + Step 3 — no residue.

**3. One-way-door completeness.** The only hard-to-reverse decision (new REST write contract) is listed in Must-Haves and tagged `> ⚠ one-way:` on Task 5 with a STOP before Task 6. The `SecTool`→`ExternalTool` rename is internal/mechanical and explicitly noted as not one-way. No migration.

**4. Placeholder scan.** No "TBD", "implement later", or "similar to Task N". Every step names exact files, exact signatures, exact test names, and exact run commands. The one deliberate branch in Task 5 (spawn detached vs. synchronous) is resolved inline to **spawn detached + `202`** — not left open.

**5. Type consistency.** `RunSummary`/`Run`/`Task` (Task 2) used by CLI (Task 3), HTTP (Task 5), and SPA (Task 6) — consistent. `HarnessConfig`/`EnvMap`/`RunsRoot` (Task 1) used by `Runner` (Task 4), `LoadHarnessConfig` (Task 9), and `installHarnessTools` (Task 7). `ErrNotFound`/`ErrBadID` (Task 2) referenced by CLI (Task 3) and HTTP (Task 5). `ExternalTool` (Task 7) used by both `SecurityTools` and `HarnessTools`. No rename drift.

## Owed-Decision Gate

Every value the build produces has a named source:
- runs root → Task 1 `RunsRoot` default `~/.skillgrid/mnemonic/harness2/runs/` (briefing decision 4).
- `HARNESS2_*` env values → `LoadHarnessConfig` reads env/config.d (named).
- run id returned by `POST /harness/runs` → a minted `<UTC-stamp>-<rand4>` slug the handler creates as the `runs/<run_id>/` subdir (named in Task 5, decided: detached spawn).
- judge verdict/score/Δ → parsed from `judged.json`/`base_judged.json` (Task 2, named).
- install command → `HarnessTools()` `InstallArgs` (Task 7, named).
- SPA columns → the `RunSummary` fields (Task 2, named).

**No owed decisions.** Proceeding to the Plan Review gate.

## Plan Review

Fresh-eyes structural pass: the tracer (Tasks 1→2→3) reaches a working, observable path before the spawn path (Task 4) and the write surface (Task 5). Dependencies are linear and each task ends with a green `go test`. The risk surface (file contract + subprocess) is concentrated in Tasks 2 and 4 with explicit RED tests per threat row. The SPA (Task 6) and installer (Task 7) are independent and could run in parallel after Task 5.

**Verdict: READY FOR EXECUTION.**

## Execution Handoff

The blueprint has 9 tasks (3+), so **slicing is required** to produce `tasks.md` with dependency edges + execution waves. Offered after slicing.

- **Task 4 (`Runner`)** and **Task 5 (REST POST)** are the two risk-first items; slicing orders them early.
- **Parallel-safe:** Task 6 (SPA) and Task 7 (installer) are independent of each other and of each other's waves once Task 5's contract is frozen.

Two execution options (chosen at handoff):
1. **Subagent-Driven (recommended)** — fresh subagent per ticket, review between tickets.
2. **Inline Execution** — `skillgrid:simple-execution`, batched with checkpoints.
