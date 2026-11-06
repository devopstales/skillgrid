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


## Ticket (tasks.md TICKET-02)

### TICKET-02 — Preamble and sentinel upsert

- **Scope:** `writeBootFile` inserts/keeps/forces the preamble region and upserts exactly one Skillgrid sentinel pair.
- **Acceptance:** `TestInitUpsertsPreambleAndSentinel` and `TestInitForceRewritesPreamble` PASS. Second run does not duplicate the sentinel; `--force` replaces preamble only.
- **SATISFIES:** happy path init upserts preamble and sentinel; second init merges without duplicating the sentinel; force rewrites the preamble only
- **Files:** `init_boot.go`, `init_cmd.go`, `init_cmd_test.go`
- **Size:** ~220 (S)
- **Blocks:** none
- **Blocked by:** TICKET-01
- **Reversibility:** reversible
- **Fails-when:** `go test ... -run TestInitUpsertsPreambleAndSentinel|TestInitForceRewritesPreamble` is non-zero


## Prior commit
T01 landed at b5239413 — projectInit skeleton exists; replace empty AGENTS write with writeBootFile.
