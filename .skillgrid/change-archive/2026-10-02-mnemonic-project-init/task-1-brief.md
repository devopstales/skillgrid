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


## Ticket (tasks.md TICKET-01)

### TICKET-01 — Init command writes a boot file

- **Scope:** Dispatch `skillgrid init`, parse `--force` / `--docs`, write a boot file path on success, fatal only when the boot file cannot be written.
- **Acceptance:** `TestInitHelpListsFlags`, `TestInitWritesBootFileAndReportsCounts`, `TestInitBootFileWriteFailureIsFatal` PASS. `skillgrid init -h` names both flags. Non-directory target returns error.
- **SATISFIES:** happy path init writes boot file and reports counts; init help lists force and docs flags; boot file write failure is the only fatal error
- **Files:** `skillgrid-cli/cmd/skillgrid/init_cmd.go`, `init_cmd_test.go`, `main.go`
- **Size:** ~180 (S)
- **Blocks:** TICKET-02, TICKET-03, TICKET-04
- **Blocked by:** none
- **Reversibility:** reversible
- **Fails-when:** `go test ... -run TestInitHelpListsFlags|TestInitWritesBootFileAndReportsCounts|TestInitBootFileWriteFailureIsFatal` is non-zero

