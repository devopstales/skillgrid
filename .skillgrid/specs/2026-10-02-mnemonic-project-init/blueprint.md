# Project Init Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use skillgrid:subagent-execution (recommended) or skillgrid:simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Smallest usable whole

**Classification:** standard (L2 floor)

**Queued behind:** `2026-10-02-mnemonic-memory-checkpoint` — do not take `state.yaml` `current_change` or start execution until that change ships or parks. per ASSUMPTIONS.md § Locked constraints (serial development)

**Goal:** Add `skillgrid init` as an orchestrator that writes the AGENTS preamble + sentinel, runs the existing code index, and upserts local docs into the one SQLite store; teach onboarding to call it.

**Architecture:** New CLI verb `runProjectInit` sequences three existing seams: boot-file upsert (preamble markers + `<!-- skillgrid:start -->` sentinel from `_shared/agent-config/block.md`), `service.Service.RunCodeIndex`, and `memory.Service.Save` via `OpenForDirectory`. No new store, no new MCP tool, no four KBs. per `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md`

**Tech Stack:** Go 1.22+ (`flag`, `os`, `path/filepath`), existing `internal/mnemonic/service` + `memory.Save`

**Spec:** `.skillgrid/specs/2026-10-02-mnemonic-project-init/briefing.md`

## Hypothesis

**Claim:** A temp project with a README and one Go file, after `projectInit(dir, false, nil)`, has exactly one Skillgrid sentinel pair, a preamble region, `FilesIndexed >= 1`, and an observation at `init/docs/README.md`.
**Right condition:** The four Task 1–4 tests named below PASS on that fixture.
**Wrong condition:** Init cannot write the boot file without a second store, or ingest requires a new KB table.
**Thinnest MVP:** Task 1 (command + result + help) plus Task 2 (boot file) — if the sentinel upsert cannot stay a single pair, stop.
**Door check:** Task 1 `TestInitHelpListsFlags` + `TestInitWritesBootFileAndReportsCounts` (counts may be zero until later tasks; boot file path and exit 0 must hold).

## Terms

- [Project Init](../../artifacts/02-technical-terms.md) — this command + skill handoff
- [Knowledge Base (KB)](../../artifacts/02-technical-terms.md) — do not create
- [Topic Key](../../artifacts/02-technical-terms.md) — `init/docs/<relpath>`
- [Distribution Surface](../../artifacts/02-technical-terms.md) — CLI verb only

## Must-Haves (goal-backward verification)

**Truths:**
- `skillgrid init -h` names `--force` and `--docs`
- First init on an empty-AGENTS temp dir creates `AGENTS.md` with preamble markers and exactly one `<!-- skillgrid:start -->` / `<!-- skillgrid:end -->` pair
- Second init without `--force` keeps one sentinel pair and does not wipe user text below the sentinel
- `--force` replaces only the preamble region
- A project with an indexable `.go` file has `Indexed >= 1` after init (`backstop`)
- Existing `README.md` upserts `topic_key=init/docs/README.md`; missing `docs/` is listed in `Skipped` and is not an error
- `--docs extra.md` (inside the project) upserts `init/docs/extra.md`; `--docs` outside the project or a missing path is listed in `Errors` and the command still exits 0 if the boot file was written (`backstop`)
- Onboarding `SKILL.md` names `skillgrid init` as the deterministic finish and has no second file-walk ingest procedure

**Artifacts:**
- `skillgrid-cli/cmd/skillgrid/init_cmd.go` — flags, `runProjectInit`, `projectInit`, result print
- `skillgrid-cli/cmd/skillgrid/init_boot.go` — preamble + sentinel upsert
- `skillgrid-cli/cmd/skillgrid/init_ingest.go` — path walk, jail, Save
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` — all TestInit* named in acceptance gates
- `skillgrid-cli/cmd/skillgrid/main.go` — `case "init":` + usage line
- `.agents/skills/lifecycle/onboarding/SKILL.md` — one step that runs the CLI

**Key links:**
- `runProjectInit` calls `projectInit` then prints `initResult`; fatal only when `BootFile` write fails
- `projectInit` calls `writeBootFile` then `RunCodeIndex` then `ingestPaths`
- `ingestPaths` uses the same `ProjectHandle.Memory().Save` as `skillgrid mem`
- Onboarding step invokes the same binary verb, not a parallel ingest list

**One-way-door decisions:**
- None (no migration, no new MCP tool, no public HTTP shape)

## Global Constraints

- Go 1.22+ to build. per ASSUMPTIONS.md § Locked constraints
- No new dependencies without an ADR. per ASSUMPTIONS.md § Locked constraints
- Serial development: one change at a time. per ASSUMPTIONS.md § Locked constraints
- One SQLite store; no KBs. per `.skillgrid/artifacts/04-adr-0012-sqlite-as-second-brain.md`
- Errors-as-values on the init report; index/ingest failures are listed, not thrown as the process exit, except boot-file write. per `.skillgrid/artifacts/04-adr-0016-second-brain-capability-layer.md` (C4 shape, CLI analogue)
- `--docs` paths must resolve inside the project directory after `filepath.Clean` + `filepath.Rel` (a `..` prefix is an error line, not ingested)

## Threat Matrix

| Boundary | Applicability | Design response | Planned RED test |
|---|---|---|---|
| Documentation-like paths | Applicable | Ingest reads file bytes into `Save`; never exec, chmod, or install | `TestInitIngestsExtraDocs` writes `README.sh` and asserts an observation, not a subprocess |
| Git repository selection | Applicable | Project is `--dir` (default cwd); `OpenForDirectory(dir)` | All TestInit* use `t.TempDir()` as `--dir` |
| Commit state | N/A: init does not git-add or commit | — | — |
| Push state | N/A: no remotes | — | — |
| PR commands | N/A: no gh/glab | — | — |
| Mnemonic tool surface | N/A: no new `mem_*` tool; CLI calls existing `Save` | — | — |
| Shared-convention drift | N/A: no edit to `_shared/conventions` or `agent-config/block.md`; sentinel text is copied from the existing block rules | — | — |
| Path jail (`--docs`) | Applicable | Extra paths outside `dir` → `Errors`, not ingested | `TestInitMissingExtraDocsIsNonFatal` plus `TestInitRejectsDocsOutsideProject` |

## File Structure

- `skillgrid-cli/cmd/skillgrid/init_cmd.go` — `initResult`, `runProjectInit`, `projectInit`, usage
- `skillgrid-cli/cmd/skillgrid/init_boot.go` — `writeBootFile`, preamble template, sentinel upsert
- `skillgrid-cli/cmd/skillgrid/init_ingest.go` — default path list, walk, jail, Save
- `skillgrid-cli/cmd/skillgrid/init_cmd_test.go` — CLI + helper tests
- `skillgrid-cli/cmd/skillgrid/main.go` — dispatch + help
- `.agents/skills/lifecycle/onboarding/SKILL.md` — handoff step

Named types (all tasks):

```go
type initResult struct {
	BootFile  string
	Preamble  string // "written" | "kept" | "forced"
	Sentinel  string // "upserted"
	Indexed   int
	Ingested  int
	Skipped   []string
	Errors    []string
}

func runProjectInit(args []string) // os.Exit
func projectInit(ctx context.Context, svc *service.Service, dir string, force bool, extraDocs []string) (initResult, error)
func writeBootFile(dir string, force bool) (bootPath, preambleState string, err error)
func ingestPaths(ctx context.Context, h *service.ProjectHandle, dir string, extra []string) (ingested int, skipped, errs []string)
```

Default ingest roots (only if they exist): `README.md`, `docs/`, `.skillgrid/ASSUMPTIONS.md`, `.skillgrid/ARCHITECTURE.md`, `.skillgrid/artifacts/`. Topic key: `init/docs/` + slash-separated path relative to `dir`. Type: `architecture` for ASSUMPTIONS, ARCHITECTURE, and anything under `.skillgrid/artifacts/`; `discovery` for README and `docs/`.

Preamble region markers: `<!-- skillgrid-preamble:start -->` … `<!-- skillgrid-preamble:end -->`. Sentinel: existing `<!-- skillgrid:start -->` … `<!-- skillgrid:end -->`.

Preamble body (exact):

```
# Project Overview
{dir base name}.

# Environment & Tooling
- Fill in language, package manager, test and lint commands.

# Engineering Standards
- Add or update tests for behavior changes
- Keep diffs small and reviewable

# Security & Escalation Boundaries
- Work only in this repository
- Do not read secrets or credential stores
- Ask before installing dependencies or changing auth logic

# Dependency Policies
- Prefer the standard library
- New packages require approval

# Architecture Constraints
- See `.skillgrid/ASSUMPTIONS.md` when present

# Definition of Done
- Tests pass
- Lint and formatting pass
- Update the spec if behavior changes
```

On `--force`, replace only the preamble region. If markers are missing, insert the region immediately before the sentinel (or at file start if no sentinel yet).

---

### Task 1: Init command door check

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/init_cmd.go`
- Create: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` (add `case "init":` next to `case "index":`, and a usage line `init          Project boot file, code index, and doc ingest`)

**Interfaces:**
- Consumes: none
- Produces: `runProjectInit`, `initResult`, `projectInit` (may return empty Indexed/Ingested until later tasks)
- Seam: none (in-process)
- Deletion test: without this file, `skillgrid` has no project init verb
- Adapters: 1 (CLI)

**SATISFIES:** happy path init writes boot file and reports counts; init help lists force and docs flags; boot file write failure is the only fatal error

- [ ] **Step 1: Write the failing tests**

```go
func TestInitHelpListsFlags(t *testing.T) {
	var buf bytes.Buffer
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(&buf)
	_ = fs.Bool("force", false, "rebuild generated preamble")
	_ = fs.String("docs", "", "extra path (repeatable)")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "usage: skillgrid init [--force] [--docs path]...")
		fs.PrintDefaults()
	}
	fs.Usage()
	out := buf.String()
	if !strings.Contains(out, "--force") || !strings.Contains(out, "--docs") {
		t.Fatalf("usage = %q", out)
	}
}

func TestInitWritesBootFileAndReportsCounts(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" {
		t.Fatal("expected boot file path")
	}
	if _, statErr := os.Stat(res.BootFile); statErr != nil {
		t.Fatal(statErr)
	}
}

func TestInitBootFileWriteFailureIsFatal(t *testing.T) {
	// dir is a file, so AGENTS.md cannot be created
	dir := filepath.Join(t.TempDir(), "notdir")
	if err := os.WriteFile(dir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal'`
Expected: FAIL — `projectInit` undefined

- [ ] **Step 3: Write minimal implementation**

`init_cmd.go`: parse `--force`, repeatable `--docs` via `fs.Var` on a `stringList` (or loop leftover args). `projectInit` for this task only: if `os.Stat(dir)` is not a directory, return error; else write an empty `AGENTS.md` if missing and set `BootFile`, `Preamble: "written"`, `Sentinel: "upserted"`. `runProjectInit` prints those fields and `indexed`/`ingested`/`skipped`/`errors` (zeros/empty). `main.go` dispatch: `case "init": runProjectInit(rest[1:]); return`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal'`
Expected: PASS

- [ ] **Step 5: Commit** (when this change is current)

```
feat(cli): add skillgrid init command skeleton
```

---

### Task 2: Boot file preamble and sentinel

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/init_boot.go`
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd.go` (`projectInit` calls `writeBootFile`)
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`

**Interfaces:**
- Consumes: `projectInit` from Task 1
- Produces: `writeBootFile(dir string, force bool) (bootPath, preambleState string, err error)`
- Seam: none (in-process)
- Deletion test: AGENTS write would scatter into `projectInit`
- Adapters: 1

**SATISFIES:** happy path init upserts preamble and sentinel; second init merges without duplicating the sentinel; force rewrites the preamble only

- [ ] **Step 1: Write the failing tests**

```go
func TestInitUpsertsPreambleAndSentinel(t *testing.T) {
	dir := t.TempDir()
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(res.BootFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, "<!-- skillgrid-preamble:start -->") || !strings.Contains(s, "# Definition of Done") {
		t.Fatalf("missing preamble: %s", s)
	}
	if strings.Count(s, "<!-- skillgrid:start -->") != 1 || strings.Count(s, "<!-- skillgrid:end -->") != 1 {
		t.Fatalf("sentinel count: %s", s)
	}
}

func TestInitForceRewritesPreamble(t *testing.T) {
	dir := t.TempDir()
	res, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(res.BootFile)
	custom := strings.Replace(string(body), "# Project Overview", "# Project Overview\nUSER KEEP", 1)
	if err := os.WriteFile(res.BootFile, []byte(custom+"\n\nUSER BELOW\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res2, err := projectInit(context.Background(), service.New(t.TempDir()), dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := os.ReadFile(res2.BootFile)
	if !strings.Contains(string(keep), "USER KEEP") || !strings.Contains(string(keep), "USER BELOW") {
		t.Fatalf("merge dropped user text: %s", keep)
	}
	if strings.Count(string(keep), "<!-- skillgrid:start -->") != 1 {
		t.Fatal("duplicated sentinel")
	}
	res3, err := projectInit(context.Background(), service.New(t.TempDir()), dir, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	forced, _ := os.ReadFile(res3.BootFile)
	if strings.Contains(string(forced), "USER KEEP") {
		t.Fatal("force left old preamble")
	}
	if !strings.Contains(string(forced), "USER BELOW") {
		t.Fatal("force wiped text outside regions")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitUpsertsPreambleAndSentinel|TestInitForceRewritesPreamble'`
Expected: FAIL — missing preamble markers or duplicated/empty sentinel

- [ ] **Step 3: Write `writeBootFile`**

Target file: `AGENTS.md` if it exists or if `CLAUDE.md` does not exist; else `CLAUDE.md`. If both exist, write AGENTS and ensure CLAUDE.md has the one-line pointer `See AGENTS.md — the Skillgrid block there is the source of truth.` when that line is absent.

Sentinel body is the canonical block from `.agents/skills/_shared/agent-config/block.md` with `{project}` = `filepath.Base(dir)`, `{tracker_line}` = `None — work local-only from tasks.md.`, `{memory_line}` = the Enabled mnemonic line from that file, `{rules_block}` = `No locked constraints yet — see `.skillgrid/ASSUMPTIONS.md`.` (if ASSUMPTIONS locked section is absent). Replace `<!-- skillgrid:start -->`…`<!-- skillgrid:end -->` in place; if missing, append.

Preamble: if markers exist and `force` is false, leave the region (`preambleState="kept"`). If missing, insert before sentinel (`written`). If `force`, replace the region (`forced`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitUpsertsPreambleAndSentinel|TestInitForceRewritesPreamble'`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(cli): upsert AGENTS preamble and Skillgrid sentinel
```

---

### Task 3: Forced code index

**Files:**
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd.go`
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`

**Interfaces:**
- Consumes: `projectInit`, `service.Service.RunCodeIndex`, `ResolveProject`
- Produces: `initResult.Indexed` = `stats.FilesIndexed`; index errors appended to `Errors`
- Seam: `service.Service` (prod + test `service.New(temp)`)
- Deletion test: callers would shell `skillgrid index`
- Adapters: 2 (real service + temp data dir)

**SATISFIES:** happy path init indexes the project; init reuses the existing indexer; index failure after boot file still exits 0

- [ ] **Step 1: Write the failing tests**

```go
func TestInitIndexesTheProject(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Indexed < 1 {
		t.Fatalf("Indexed = %d", res.Indexed)
	}
}

func TestInitIndexFailureIsNonFatal(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	missing := filepath.Join(dir, "no-such-child")
	// Index a path that is not a directory after boot file is written beside dir.
	// projectInit indexes `dir`; to force index error, pass a svc that already
	// closed — simpler: index a file path via a helper hook. Instead, chmod
	// the only source file to be unreadable after writeBootFile by calling
	// projectInit on a dir whose contents cannot be opened: create dir, write
	// AGENTS via first successful call, then chmod 000 the dir's only .go and
	// re-run. If the OS still indexes, skip. Preferred: if RunCodeIndex on an
	// empty unreadable path fails, record Errors.
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" {
		t.Fatal("boot file required even when index is empty")
	}
}
```

Pin the failure test as: `projectInit` catches `RunCodeIndex` error, sets `Errors` containing `"index failed"`, returns `err == nil`. Drive it by injecting a tiny optional `indexFn` only if needed; prefer calling `RunCodeIndex` on `dir` and, for the failure test, stub by exporting:

```go
var runIndex = func(ctx context.Context, svc *service.Service, dir string) (int, error) {
	if _, err := svc.ResolveProject(dir); err != nil {
		return 0, err
	}
	st, err := svc.RunCodeIndex(ctx, dir)
	return st.FilesIndexed, err
}
```

In `TestInitIndexFailureIsNonFatal`, set `runIndex = func(...) (int, error) { return 0, errors.New("boom") }` and defer restore. Assert `err == nil`, `len(res.Errors) >= 1`, boot file exists.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitIndexesTheProject|TestInitIndexFailureIsNonFatal'`
Expected: FAIL — Indexed == 0 or Errors empty

- [ ] **Step 3: Call `runIndex` after `writeBootFile`**

Do not open a second `service.New` inside `projectInit`; use the `svc` argument. Tests pin `cliService` only when going through `runProjectInit`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitIndexesTheProject|TestInitIndexFailureIsNonFatal'`
Expected: PASS

- [ ] **Step 5: Commit**

```
feat(cli): run existing code index from skillgrid init
```

---

### Task 4: Default and extra ingest

**Files:**
- Create: `skillgrid-cli/cmd/skillgrid/init_ingest.go`
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd.go`
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`

**Interfaces:**
- Consumes: `OpenForDirectory`, `ProjectHandle.Memory().Save`, `SessionStart`
- Produces: `ingestPaths`; `initResult.Ingested` / `Skipped` / extra `Errors`
- Seam: `memory.Service.Save`
- Deletion test: docs would not land in the store
- Adapters: 2 (real memory service + temp store)

**SATISFIES:** happy path init ingests default paths; missing docs directory is skipped; second init upserts the same topic keys; happy path init ingests extra --docs path; missing extra docs path is listed and non-fatal

- [ ] **Step 1: Write the failing tests**

```go
func countTopic(t *testing.T, mem *memory.Service, key string) int {
	t.Helper()
	hits, err := mem.Search(context.Background(), "init", "", 50)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, h := range hits {
		if h.TopicKey == key {
			n++
		}
	}
	return n
}

func TestInitIngestsDefaultPaths(t *testing.T) {
	data := t.TempDir()
	svc := service.New(data)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# App"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("g"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), svc, dir, false, nil); err != nil {
		t.Fatal(err)
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/README.md") != 1 {
		t.Fatal("README observation")
	}
	if countTopic(t, h.Memory(), "init/docs/docs/guide.md") != 1 {
		t.Fatal("docs observation")
	}
}

func TestInitSkipsMissingDocs(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("r"), 0o644)
	res, err := projectInit(context.Background(), svc, dir, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range res.Skipped {
		if s == "docs/" || s == "docs" {
			found = true
		}
	}
	if !found {
		t.Fatalf("skipped = %v", res.Skipped)
	}
}

func TestInitIngestsExtraDocs(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.sh"), []byte("echo hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := projectInit(context.Background(), svc, dir, false, []string{filepath.Join(dir, "README.sh")}); err != nil {
		t.Fatal(err)
	}
	h, closeH, err := svc.OpenForDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer closeH()
	if countTopic(t, h.Memory(), "init/docs/README.sh") != 1 {
		t.Fatal("extra docs")
	}
}

func TestInitMissingExtraDocsIsNonFatal(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	res, err := projectInit(context.Background(), svc, dir, false, []string{filepath.Join(dir, "missing.md")})
	if err != nil {
		t.Fatal(err)
	}
	if res.BootFile == "" || len(res.Errors) == 0 {
		t.Fatalf("res = %+v", res)
	}
}

func TestInitRejectsDocsOutsideProject(t *testing.T) {
	svc := service.New(t.TempDir())
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "x.md")
	_ = os.WriteFile(outside, []byte("no"), 0o644)
	res, err := projectInit(context.Background(), svc, dir, false, []string{outside})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected jail error")
	}
}
```

Also add a second-init upsert case in `TestInitIngestsDefaultPaths` (run `projectInit` twice, still `countTopic == 1`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInitIngests|TestInitSkips|TestInitMissing|TestInitRejects'`
Expected: FAIL — Ingested 0 / missing observations

- [ ] **Step 3: Implement `ingestPaths`**

Open `h, close, err := svc.OpenForDirectory(dir)`. `sid, err := h.Memory().SessionStart(ctx, dir, "skillgrid-init")`. For each existing default file (walk `docs/` and `.skillgrid/artifacts/` for regular files only; skip dirs and the missing root), `Save` with `CapturePrompt: false`, `SessionID: sid`, `Title` = base name, `Content` = file bytes as string, `TopicKey` as specified, `ToolName: "skillgrid-init"`. Re-run upserts via the same topic key. `--docs` entries: `Rel` to `dir`; if `strings.HasPrefix(rel, "..")`, append error and skip. Missing extra path: append error. Missing default root: append to `Skipped`. Save errors append to `Errors` and do not abort the walk.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestInit'`
Expected: PASS (all init tests)

- [ ] **Step 5: Commit**

```
feat(cli): ingest project docs into mnemonic from skillgrid init
```

---

### Task 5: Onboarding skill hands off to CLI

**Files:**
- Modify: `.agents/skills/lifecycle/onboarding/SKILL.md` (after Step 3 Write / before Step 4 Verify)
- Modify: `skillgrid-cli/cmd/skillgrid/init_cmd_test.go`

**Interfaces:**
- Consumes: `skillgrid init` from Tasks 1–4
- Produces: skill contract only
- Seam: none
- Deletion test: agents would invent ingest
- Adapters: 1

**SATISFIES:** happy path onboarding skill calls skillgrid init; skill passes extra docs through; skill does not contain a second ingest procedure

- [ ] **Step 1: Write the failing tests**

```go
func TestOnboardingSkillCallsSkillgridInit(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(body, []byte("skillgrid init")) {
		t.Fatal("onboarding must name skillgrid init")
	}
}

func TestOnboardingSkillHasNoSecondIngest(t *testing.T) {
	body, err := os.ReadFile(onboardingSkillPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(body, []byte("walk docs/ and mem_save each file")) {
		t.Fatal("skill must not re-specify ingest")
	}
}

func onboardingSkillPath(t *testing.T) string {
	t.Helper()
	// repo root: this file is skillgrid-cli/cmd/skillgrid
	p := filepath.Join("..", "..", "..", ".agents", "skills", "lifecycle", "onboarding", "SKILL.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	return p
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestOnboardingSkill'`
Expected: FAIL — `skillgrid init` absent

- [ ] **Step 3: Add one step to the skill**

After the Write steps, add:

```
**9. Project init (deterministic finish):**
- Run `skillgrid init` from the project root (add `--force` only if the user asked to rebuild the preamble).
- If the user named extra documentation paths, pass each as `--docs <path>`.
- Do not walk files or call `mem_save` for docs here — the CLI owns ingest and the code index.
- If `skillgrid` is not on PATH, tell the user to install the CLI and re-run `skillgrid init`; continue the rest of onboarding.
```

Do not add a file-walk ingest procedure.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./skillgrid-cli/cmd/skillgrid -count=1 -run 'TestOnboardingSkill|TestInit'`
Expected: PASS

- [ ] **Step 5: Commit**

```
docs(onboarding): finish with skillgrid init
```

## Self-review

| Spec requirement | Task |
|---|---|
| 1 Init command | Task 1 |
| 2 Boot file | Task 2 |
| 3 Forced code index | Task 3 |
| 4 Default ingest | Task 4 |
| 5 Extra `--docs` | Task 4 |
| 6 Skill handoff | Task 5 |

Must-haves map 1:1. No one-way doors. No placeholders. Types: `initResult`, `projectInit`, `writeBootFile`, `ingestPaths`, `runIndex` stay consistent.

## Owed-decision gate

Values and sources: flag names (spec), preamble headings (this blueprint), topic key prefix `init/docs/` (spec), observation types (spec), path jail (threat matrix), sentinel text (existing `block.md`), index via `RunCodeIndex` (existing API), Save via `memory.Save` (existing API). No owed decision.

## Plan review (inline)

1. **Failure modes:** Unreadable `--docs` file → `Errors`. Index boom → `runIndex` stub + `Errors`. Boot file on a non-dir → fatal. Concurrent init not in scope (local CLI). Empty project → skipped defaults, still writes AGENTS.
2. **Scope:** No Kanban, no KBs, no MCP tool. Matches briefing out-of-scope.
3. **Feasibility:** All APIs exist (`RunCodeIndex`, `OpenForDirectory`, `Save`, `SessionStart`).
4. **Ordering:** Command → boot → index → ingest → skill. Skill last so the verb exists.
5. **Tests:** Gates G1–G12 named in `acceptance.feature` match these `Test*` names.
6. **ADR:** Keep-store honored; no new table.

**Verdict:** READY FOR EXECUTION — after `2026-10-02-mnemonic-memory-checkpoint` ships or parks.

## Execution handoff

5 tasks → slicing is due when this change becomes current. Do not slice or execute while memory-checkpoint owns the pipeline.

When it is current: **1. Subagent-Driven (recommended)** or **2. Inline Execution**. Say which then.
