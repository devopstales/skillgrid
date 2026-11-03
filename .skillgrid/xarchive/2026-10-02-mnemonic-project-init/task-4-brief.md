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


## Ticket (tasks.md TICKET-04)
### TICKET-04 — Default and extra ingest

- **Scope:** Walk default paths + jailed `--docs` into `memory.Save` with `init/docs/<relpath>`; skip missing defaults; list bad extras in `Errors`.
- **Acceptance:** `TestInitIngestsDefaultPaths`, `TestInitSkipsMissingDocs`, `TestInitIngestsExtraDocs`, `TestInitMissingExtraDocsIsNonFatal`, `TestInitRejectsDocsOutsideProject` PASS. Second init does not duplicate topic keys.
- **SATISFIES:** happy path init ingests default paths; missing docs directory is skipped; second init upserts the same topic keys; happy path init ingests extra --docs path; missing extra docs path is listed and non-fatal
- **Files:** `init_ingest.go`, `init_cmd.go`, `init_cmd_test.go`
- **Size:** ~200 (S)
- **Blocks:** none
- **Blocked by:** TICKET-01
- **Reversibility:** reversible
- **Fails-when:** `go test ... -run 'TestInitIngests|TestInitSkips|TestInitMissing|TestInitRejects'` is non-zero


## Prior: T03 at 03439ce7 — add ingestPaths after index; topic_key init/docs/<relpath>; jail --docs
