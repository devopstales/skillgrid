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


## Ticket (tasks.md TICKET-03)
### TICKET-03 — Forced code index

- **Scope:** After the boot file, `projectInit` calls `ResolveProject` + `RunCodeIndex` on the same `svc`; index errors go in `Errors`, exit still 0.
- **Acceptance:** `TestInitIndexesTheProject` (`Indexed >= 1` on a `.go` fixture) and `TestInitIndexFailureIsNonFatal` PASS.
- **SATISFIES:** happy path init indexes the project; init reuses the existing indexer; index failure after boot file still exits 0
- **Files:** `init_cmd.go`, `init_cmd_test.go`
- **Size:** ~80 (S)
- **Blocks:** none
- **Blocked by:** TICKET-01
- **Reversibility:** reversible
- **Fails-when:** `go test ... -run TestInitIndexesTheProject|TestInitIndexFailureIsNonFatal` is non-zero


## Prior: T02 at 9dd840f5 — call RunCodeIndex after writeBootFile on same svc
