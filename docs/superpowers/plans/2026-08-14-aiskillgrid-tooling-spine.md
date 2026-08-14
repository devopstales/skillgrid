# Tooling Spine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** On `aiskillgrid install`, provision managed binaries + managed npm packages and merge resolved `aiskillgrid-*` MCP servers into selected agents.

**Architecture:** Extend `home.Paths` for `dependencies/bin` and `npm/`. New `tools` package installs release binaries and npm packages (no brew/nix), resolves `packs/mcp/servers.json` placeholders, and injects the resolved map into agent MCP merges via `agents.Context`.

**Tech Stack:** Go 1.25, cobra, survey, encoding/json, net/http (GitHub releases), os/exec (npm), existing mcpmerge.

**Spec:** [docs/superpowers/specs/2026-08-14-aiskillgrid-tooling-spine-design.md](../specs/2026-08-14-aiskillgrid-tooling-spine-design.md)

## Global Constraints

- Binary where possible; managed npm where not. **No Homebrew / Nix** for tool installs.
- Never pollute user global npm: always `npm install --prefix ~/.aiskillgrid/npm` with cache under `~/.aiskillgrid/npm/cache`.
- MCP owned keys only: prefix `aiskillgrid-`; one-time `.bak`.
- Default unit tests: **no network** (inject downloader / skip real fetch).
- Out of slice: `skills add` (see `packs/skills/sources.yaml` / docs/05-skills.md), AGENT.md generation, portable Node, Playwright browser ensure. OpenSpec/Backlog deferred from default stack.

## File map

| File | Role |
|------|------|
| `aiskillgrid-cli/home/home.go` | Add `DepsBinDir`, `NpmDir`, `NpmBinDir`, `NpmCacheDir`; create in `EnsureLayout` |
| `aiskillgrid-cli/tools/npm.go` | Detect node/npm; install packages into managed prefix |
| `aiskillgrid-cli/tools/binary.go` | Download GitHub release assets into `dependencies/bin` (mockable) |
| `aiskillgrid-cli/tools/resolve.go` | Load pack, substitute placeholders, honor `requires`, strip metadata |
| `aiskillgrid-cli/tools/phase.go` | Orchestrate install tools phase; return resolved servers + warnings |
| `aiskillgrid-cli/tools/*_test.go` | Unit tests |
| `packs/mcp/servers.json` | Templated MCP entries |
| `aiskillgrid-cli/agents/agents.go` | `Context.ResolvedMCP`; merge uses injected map when set |
| `aiskillgrid-cli/install/install.go` | Call tools phase; set `ctx.ResolvedMCP` |
| `aiskillgrid-cli/cmd/root.go` | Status: list managed bins / npm presence |
| `docs/04-tools.md`, `docs/TODO.md` | Align language + checkboxes after code |

**Pinned packages / binaries**

| Item | Pin |
|------|-----|
| Engram | GitHub `Gentleman-Programming/engram` latest release asset for GOOS/GOARCH → `engram` |
| skills | GitHub `qntx/skill` latest release → `skills` |
| GitNexus | npm `gitnexus` |
| Context7 | npm `@upstash/context7-mcp` |
| Playwright MCP | npm `@playwright/mcp` |
| DeepWiki | **HTTP remote** (official): `https://mcp.deepwiki.com/mcp` — no npm (community scraper packages are broken) |

OpenSpec / Backlog.md: **deferred** from default stack (Superpowers owns plans/specs/tasks).

---

### Task 1: Home layout paths for bin + npm

**Files:**
- Modify: `aiskillgrid-cli/home/home.go`
- Modify: `aiskillgrid-cli/home/home_test.go`

**Interfaces:**
- Produces: `Paths.DepsBinDir`, `Paths.NpmDir`, `Paths.NpmBinDir`, `Paths.NpmCacheDir` populated by `Resolve`; created by `EnsureLayout`

- [ ] **Step 1: Write the failing test**

Extend `TestEnsureLayoutAndConfig` to assert new dirs:

```go
for _, d := range []string{p.ToolsDir, p.DepsDir, p.DepsBinDir, p.NpmDir, p.NpmBinDir, p.NpmCacheDir, p.LogsDir, p.SessionsDir, p.MemoriesDir} {
	if st, err := os.Stat(d); err != nil || !st.IsDir() {
		t.Fatalf("missing dir %s: %v", d, err)
	}
}
if p.DepsBinDir != filepath.Join(root, "dependencies", "bin") {
	t.Fatalf("DepsBinDir=%s", p.DepsBinDir)
}
if p.NpmCacheDir != filepath.Join(root, "npm", "cache") {
	t.Fatalf("NpmCacheDir=%s", p.NpmCacheDir)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd aiskillgrid-cli && go test ./home/ -run TestEnsureLayoutAndConfig -v`  
Expected: FAIL (unknown fields / missing dirs)

- [ ] **Step 3: Minimal implementation**

In `Paths` add:

```go
DepsBinDir  string
NpmDir      string
NpmBinDir   string
NpmCacheDir string
```

In `Resolve`:

```go
DepsBinDir:  filepath.Join(root, "dependencies", "bin"),
NpmDir:      filepath.Join(root, "npm"),
NpmBinDir:   filepath.Join(root, "npm", "bin"),
NpmCacheDir: filepath.Join(root, "npm", "cache"),
```

In `EnsureLayout`, include `p.DepsBinDir`, `p.NpmDir`, `p.NpmBinDir`, `p.NpmCacheDir` in the dirs slice (keep `p.DepsDir` as parent).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd aiskillgrid-cli && go test ./home/ -run TestEnsureLayoutAndConfig -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add aiskillgrid-cli/home/home.go aiskillgrid-cli/home/home_test.go
git commit -m "feat(home): add managed dependencies/bin and npm layout paths"
```

---

### Task 2: Managed npm install helper

**Files:**
- Create: `aiskillgrid-cli/tools/npm.go`
- Create: `aiskillgrid-cli/tools/npm_test.go`

**Interfaces:**
- Consumes: `home.Paths`
- Produces:
  - `func LookPathNode() (node, npm string, err error)`
  - `func EnsureManagedNPM(p home.Paths) error` — mkdir only if needed; error if node/npm missing
  - `func InstallNPMPackages(p home.Paths, pkgs []string) error` — runs `npm install --prefix <NpmDir> --cache <NpmCacheDir> <pkgs...>`
  - `func ManagedBin(p home.Paths, name string) string` — `filepath.Join(p.NpmBinDir, name)` (+ `.cmd`/`.exe` awareness on Windows later; v1: Unix name + optional `.exe` if `runtime.GOOS == "windows"`)

- [ ] **Step 1: Write the failing test**

```go
func TestInstallNPMPackagesInvokesNpmWithPrefix(t *testing.T) {
	// Use a fake npm script on PATH that records args to a file.
	dir := t.TempDir()
	record := filepath.Join(dir, "args.txt")
	fake := filepath.Join(dir, "npm")
	script := "#!/bin/sh\necho \"$@\" > " + record + "\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	// Also need a fake node for EnsureManagedNPM
	node := filepath.Join(dir, "node")
	_ = os.WriteFile(node, []byte("#!/bin/sh\n"), 0o755)

	p := home.Resolve(t.TempDir())
	_ = home.EnsureLayout(p)
	if err := InstallNPMPackages(p, []string{"gitnexus", "backlog.md"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(record)
	got := string(b)
	if !strings.Contains(got, "--prefix") || !strings.Contains(got, p.NpmDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "--cache") || !strings.Contains(got, p.NpmCacheDir) {
		t.Fatalf("args=%q", got)
	}
	if !strings.Contains(got, "gitnexus") || !strings.Contains(got, "backlog.md") {
		t.Fatalf("args=%q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd aiskillgrid-cli && go test ./tools/ -run TestInstallNPMPackages -v`  
Expected: FAIL (package/functions missing)

- [ ] **Step 3: Implement `npm.go`**

```go
func LookPathNode() (nodePath, npmPath string, err error) {
	nodePath, err = exec.LookPath("node")
	if err != nil {
		return "", "", fmt.Errorf("node not found on PATH: %w", err)
	}
	npmPath, err = exec.LookPath("npm")
	if err != nil {
		return "", "", fmt.Errorf("npm not found on PATH: %w", err)
	}
	return nodePath, npmPath, nil
}

func EnsureManagedNPM(p home.Paths) error {
	if err := os.MkdirAll(p.NpmBinDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(p.NpmCacheDir, 0o755); err != nil {
		return err
	}
	_, _, err := LookPathNode()
	return err
}

func InstallNPMPackages(p home.Paths, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	if err := EnsureManagedNPM(p); err != nil {
		return err
	}
	_, npmPath, err := LookPathNode()
	if err != nil {
		return err
	}
	args := []string{"install", "--prefix", p.NpmDir, "--cache", p.NpmCacheDir}
	args = append(args, pkgs...)
	cmd := exec.Command(npmPath, args...)
	cmd.Env = append(os.Environ(),
		"npm_config_prefix="+p.NpmDir,
		"npm_config_cache="+p.NpmCacheDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install: %w\n%s", err, out)
	}
	return nil
}

func ManagedBin(p home.Paths, name string) string {
	bin := filepath.Join(p.NpmBinDir, name)
	if runtime.GOOS == "windows" {
		return bin + ".cmd"
	}
	return bin
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd aiskillgrid-cli && go test ./tools/ -run TestInstallNPMPackages -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add aiskillgrid-cli/tools/npm.go aiskillgrid-cli/tools/npm_test.go
git commit -m "feat(tools): managed npm install into ~/.aiskillgrid/npm"
```

---

### Task 3: Release binary downloader (mockable, no network in tests)

**Files:**
- Create: `aiskillgrid-cli/tools/binary.go`
- Create: `aiskillgrid-cli/tools/binary_test.go`

**Interfaces:**
- Produces:
  - `type Downloader func(url string) ([]byte, error)`
  - `var HTTPGet Downloader` — default uses `http.Get`; tests replace it
  - `func EnsureFileExecutable(path string, data []byte) error`
  - `func EnsureReleaseBinary(destDir, destName string, assetURL string, get Downloader) error` — skip download if dest exists and is executable; else download + write
  - `func EngramAssetURL(goos, goarch string) (string, error)` — for tests, may return a constructed GitHub release URL pattern; production can resolve via API in `phase.go` with injected JSON fetcher

Keep GitHub API resolution in `phase.go` / `github.go` with injectable `fetchJSON` for tests. For Task 3, focus on write/skip/executable behavior with a fixed URL.

- [ ] **Step 1: Write the failing test**

```go
func TestEnsureReleaseBinaryWritesAndSkips(t *testing.T) {
	dir := t.TempDir()
	calls := 0
	get := func(url string) ([]byte, error) {
		calls++
		return []byte("#!/bin/sh\necho ok\n"), nil
	}
	dest := filepath.Join(dir, "engram")
	if err := EnsureReleaseBinary(dir, "engram", "https://example.invalid/engram", get); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}
	st, err := os.Stat(dest)
	if err != nil || st.Mode()&0o111 == 0 {
		t.Fatalf("not executable: %v %#o", err, st.Mode())
	}
	if err := EnsureReleaseBinary(dir, "engram", "https://example.invalid/engram", get); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected skip, calls=%d", calls)
	}
}
```

- [ ] **Step 2: Run test — expect FAIL**

Run: `cd aiskillgrid-cli && go test ./tools/ -run TestEnsureReleaseBinary -v`

- [ ] **Step 3: Implement**

```go
func EnsureReleaseBinary(destDir, destName, assetURL string, get Downloader) error {
	if get == nil {
		get = HTTPGet
	}
	dest := filepath.Join(destDir, destName)
	if st, err := os.Stat(dest); err == nil && st.Mode().IsRegular() {
		return nil
	}
	data, err := get(assetURL)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
```

Note: real Engram/skills assets are often `.tar.gz` / `.zip`. Add `ExtractBinaryFromArchive(data []byte, wantName string) ([]byte, error)` in the same file for tar.gz (Unix) and zip (Windows) — cover with a small fixture test using an in-memory tar created in the test.

- [ ] **Step 4: Pass tests**

Run: `cd aiskillgrid-cli && go test ./tools/ -run 'TestEnsureReleaseBinary|TestExtract' -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add aiskillgrid-cli/tools/binary.go aiskillgrid-cli/tools/binary_test.go
git commit -m "feat(tools): mockable GitHub release binary install into dependencies/bin"
```

---

### Task 4: MCP pack resolve (placeholders + requires)

**Files:**
- Create: `aiskillgrid-cli/tools/resolve.go`
- Create: `aiskillgrid-cli/tools/resolve_test.go`
- Modify: `packs/mcp/servers.json`

**Interfaces:**
- Consumes: pack JSON path; `home.Paths`; presence map
- Produces: `func ResolveMCPServers(packPath string, p home.Paths, present map[string]bool) (servers map[string]any, warnings []string, err error)`
  - `present` keys like `"binary:engram"`, `"npm:gitnexus"`, `"npm:@upstash/context7-mcp"`, `"http:deepwiki"`
  - Strip `requires` from entries before return
  - Substitute `{{AISKILLGRID_*}}` placeholders in all string fields (command, args, env values, url fields)

- [ ] **Step 1: Write failing tests**

```go
func TestResolveMCPServersSubstitutesAndFilters(t *testing.T) {
	dir := t.TempDir()
	pack := filepath.Join(dir, "servers.json")
	content := `{
	  "mcpServers": {
	    "aiskillgrid-engram": {
	      "command": "{{AISKILLGRID_ENGRAM}}",
	      "args": ["mcp"],
	      "requires": "binary:engram"
	    },
	    "aiskillgrid-gitnexus": {
	      "command": "{{AISKILLGRID_GITNEXUS}}",
	      "args": ["mcp"],
	      "requires": "npm:gitnexus"
	    },
	    "aiskillgrid-deepwiki": {
	      "url": "https://mcp.deepwiki.com/mcp",
	      "requires": "http:deepwiki"
	    }
	  }
	}`
	_ = os.WriteFile(pack, []byte(content), 0o644)
	p := home.Resolve(t.TempDir())
	_ = home.EnsureLayout(p)
	// Pretend engram + deepwiki present, gitnexus missing
	present := map[string]bool{"binary:engram": true, "http:deepwiki": true}
	servers, warns, err := ResolveMCPServers(pack, p, present)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["aiskillgrid-gitnexus"]; ok {
		t.Fatal("expected gitnexus skipped")
	}
	if len(warns) == 0 {
		t.Fatal("expected warning for missing gitnexus")
	}
	en := servers["aiskillgrid-engram"].(map[string]any)
	if en["command"] != filepath.Join(p.DepsBinDir, "engram") {
		t.Fatalf("command=%v", en["command"])
	}
	if _, ok := en["requires"]; ok {
		t.Fatal("requires must be stripped")
	}
}
```

- [ ] **Step 2: Run — expect FAIL**

- [ ] **Step 3: Implement resolve + write real `packs/mcp/servers.json`**

```json
{
  "mcpServers": {
    "aiskillgrid-engram": {
      "command": "{{AISKILLGRID_ENGRAM}}",
      "args": ["mcp"],
      "requires": "binary:engram"
    },
    "aiskillgrid-gitnexus": {
      "command": "{{AISKILLGRID_GITNEXUS}}",
      "args": ["mcp"],
      "requires": "npm:gitnexus"
    },
    "aiskillgrid-backlog": {
      "command": "{{AISKILLGRID_BACKLOG}}",
      "args": ["mcp"],
      "requires": "npm:backlog.md"
    },
    "aiskillgrid-context7": {
      "command": "{{AISKILLGRID_NPX}}",
      "args": ["@upstash/context7-mcp"],
      "env": {
        "npm_config_prefix": "{{AISKILLGRID_NPM}}",
        "npm_config_cache": "{{AISKILLGRID_NPM_CACHE}}"
      },
      "requires": "npm:@upstash/context7-mcp"
    },
    "aiskillgrid-playwright": {
      "command": "{{AISKILLGRID_NPX}}",
      "args": ["@playwright/mcp"],
      "env": {
        "npm_config_prefix": "{{AISKILLGRID_NPM}}",
        "npm_config_cache": "{{AISKILLGRID_NPM_CACHE}}"
      },
      "requires": "npm:@playwright/mcp"
    },
    "aiskillgrid-deepwiki": {
      "url": "https://mcp.deepwiki.com/mcp",
      "requires": "http:deepwiki"
    }
  }
}
```

Placeholder map in resolve:

```go
repl := map[string]string{
  "{{AISKILLGRID_NPM}}":        p.NpmDir,
  "{{AISKILLGRID_NPM_CACHE}}":  p.NpmCacheDir,
  "{{AISKILLGRID_BIN}}":        p.DepsBinDir,
  "{{AISKILLGRID_NPX}}":        filepath.Join(p.NpmBinDir, "npx"),
  "{{AISKILLGRID_ENGRAM}}":     filepath.Join(p.DepsBinDir, "engram"),
  "{{AISKILLGRID_GITNEXUS}}":   ManagedBin(p, "gitnexus"),
  "{{AISKILLGRID_BACKLOG}}":    ManagedBin(p, "backlog"), // backlog.md package bin name is typically `backlog`
}
```

Verify backlog bin name against package during implement; if bin is `backlog.md`, use that in `ManagedBin`.

- [ ] **Step 4: Pass tests**

- [ ] **Step 5: Commit**

```bash
git add aiskillgrid-cli/tools/resolve.go aiskillgrid-cli/tools/resolve_test.go packs/mcp/servers.json
git commit -m "feat(tools): resolve MCP pack placeholders and requires filters"
```

---

### Task 5: Tools install phase orchestration

**Files:**
- Create: `aiskillgrid-cli/tools/phase.go`
- Create: `aiskillgrid-cli/tools/github.go` (latest release asset URL picker)
- Create: `aiskillgrid-cli/tools/phase_test.go`
- Delete or gut stub: `aiskillgrid-cli/tools/registry.go` (replace `InstallDeps` no-op with call into `RunInstallPhase` or remove registry)

**Interfaces:**
- Produces:
  - `type PhaseResult struct { Servers map[string]any; Warnings []string; Present map[string]bool }`
  - `func RunInstallPhase(p home.Paths, packRoot string, opts PhaseOptions) (PhaseResult, error)`
  - `PhaseOptions` includes injectable `Downloader`, `ReleaseAssetResolver`, and `SkipNetwork bool` for tests

Phase steps:
1. Ensure layout dirs (caller may already have)
2. Try binary installs: engram, skills (failures → warning, continue)
3. Try `EnsureManagedNPM`; on failure → warning, skip npm packages
4. Else `InstallNPMPackages` with: `gitnexus`, `backlog.md`, `@fission-ai/openspec`, `@upstash/context7-mcp`, `@playwright/mcp`
5. Build `present` map from files that exist under DepsBinDir / NpmBinDir; always set `http:deepwiki` true
6. `ResolveMCPServers(filepath.Join(packRoot, "packs", "mcp", "servers.json"), …)`
7. If playwright MCP present, append warning: browsers may need install later

- [ ] **Step 1: Test phase with fakes (no network)**

```go
func TestRunInstallPhaseWithFakes(t *testing.T) {
	root := t.TempDir()
	p := home.Resolve(root)
	_ = home.EnsureLayout(p)
	packRoot := t.TempDir()
	_ = os.MkdirAll(filepath.Join(packRoot, "packs", "mcp"), 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "packs", "mcp", "servers.json"), []byte(`{
	  "mcpServers": {
	    "aiskillgrid-engram": {"command":"{{AISKILLGRID_ENGRAM}}","args":["mcp"],"requires":"binary:engram"},
	    "aiskillgrid-deepwiki": {"url":"https://mcp.deepwiki.com/mcp","requires":"http:deepwiki"}
	  }
	}`), 0o644)

	// Pre-seed engram binary so SkipNetwork path marks present
	_ = os.WriteFile(filepath.Join(p.DepsBinDir, "engram"), []byte("x"), 0o755)

	res, err := RunInstallPhase(p, packRoot, PhaseOptions{SkipNetwork: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Servers["aiskillgrid-engram"]; !ok {
		t.Fatalf("servers=%v", res.Servers)
	}
	if _, ok := res.Servers["aiskillgrid-deepwiki"]; !ok {
		t.Fatal("deepwiki missing")
	}
}
```

- [ ] **Step 2–4:** Implement phase + github asset resolution (best-effort: match `goos`/`goarch` in asset name; prefer `.tar.gz` then `.zip`). On API failure, warn and continue.

- [ ] **Step 5: Commit**

```bash
git add aiskillgrid-cli/tools/
git commit -m "feat(tools): install phase for binaries, managed npm, and MCP resolve"
```

---

### Task 6: Wire agents + install command to resolved MCP

**Files:**
- Modify: `aiskillgrid-cli/agents/agents.go`
- Modify: `aiskillgrid-cli/agents/agents_test.go` (pass ResolvedMCP or keep loading pack when nil for backward compat)
- Modify: `aiskillgrid-cli/install/install.go`
- Modify: `aiskillgrid-cli/tools/registry.go` — remove obsolete `InstallDeps` call path

**Interfaces:**
- Produces: `Context.ResolvedMCP map[string]any` — when non-nil, `mergeMCPFile` merges this map instead of loading pack from disk

- [ ] **Step 1: Failing test — merge uses injected servers**

In `agents_test.go`, set `ctx.ResolvedMCP = map[string]any{"aiskillgrid-engram": map[string]any{"command": "/tmp/engram", "args": []any{"mcp"}}}` and assert written mcp.json contains that key without needing pack file.

- [ ] **Step 2: Implement**

```go
type Context struct {
	// ...existing...
	ResolvedMCP map[string]any // optional; when set, used instead of pack file
}

func mergeMCPFile(path, mcpKey, packRoot string, resolved map[string]any) error {
	servers := resolved
	var err error
	if servers == nil {
		servers, err = mcpmerge.LoadPackServers(mcpPack(packRoot))
		if err != nil {
			return err
		}
	}
	// ... merge as today ...
}
```

Update `installSkillsAndMCP` signature to pass `ctx.ResolvedMCP`.

In `install.Run`, after sync / packRoot resolution:

```go
phase, err := tools.RunInstallPhase(paths, packRoot, tools.PhaseOptions{})
if err != nil {
	return err
}
for _, w := range phase.Warnings {
	fmt.Fprintf(os.Stderr, "warning: %s\n", w)
	_ = home.AppendLog(paths.LogsDir, "warn: "+w)
}
ctx.ResolvedMCP = phase.Servers
```

Remove `reg := tools.Default(); reg.InstallDeps(...)`.

- [ ] **Step 3: Run agent + install-related tests**

Run: `cd aiskillgrid-cli && go test ./agents/ ./tools/ ./home/ ./mcpmerge/ -count=1`  
Expected: PASS (use `required_permissions: ["all"]` if sandbox blocks mkdir under temp `.cursor`)

- [ ] **Step 4: Commit**

```bash
git add aiskillgrid-cli/agents aiskillgrid-cli/install aiskillgrid-cli/tools
git commit -m "feat(install): run tools phase and inject resolved MCP into agents"
```

---

### Task 7: Status output + docs

**Files:**
- Modify: `aiskillgrid-cli/cmd/root.go` (`statusCmd`)
- Modify: `docs/04-tools.md` — replace “detect + warn only” with managed install policy; note no brew; DeepWiki HTTP
- Modify: `docs/TODO.md` — check off completed tooling spine items
- Modify: `docs/superpowers/specs/2026-08-14-aiskillgrid-tooling-spine-design.md` — Status: Approved / Implemented

- [ ] **Step 1: Extend status**

Print lines like:

```
Managed npm: /path/npm (node: yes|no)
Binaries: engram=yes skills=yes
NPM bins: gitnexus=yes backlog=yes openspec=yes
```

Helper can live in `tools.StatusLines(p home.Paths) []string`.

- [ ] **Step 2: Update docs**

- [ ] **Step 3: Full test suite**

Run: `cd aiskillgrid-cli && go test ./... -count=1` with full permissions  
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add aiskillgrid-cli/cmd/root.go docs/
git commit -m "docs: align tools docs with managed binary/npm install spine"
```

---

## Spec coverage self-check

| Spec requirement | Task |
|------------------|------|
| Home `dependencies/bin` + `npm/` | 1 |
| System node/npm; warn if missing | 2, 5 |
| Binary Engram + skills | 3, 5 |
| npm GitNexus, Backlog, OpenSpec, Context7, Playwright | 2, 5 |
| DeepWiki HTTP | 4, 5 |
| No brew/nix | Global + phase (no brew calls) |
| Placeholder resolve + requires | 4 |
| Inject resolved MCP into agents | 6 |
| Status + doc/TODO updates | 7 |
| No network in default tests | 2–5 |
| Out of scope scaffolds / skills add | Not scheduled |

## Placeholder scan

No TBD/TODO left in task steps; DeepWiki pin is HTTP URL; Backlog is npm for v1.

---

## Execution handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-14-aiskillgrid-tooling-spine.md`.

**Two execution options:**

1. **Subagent-Driven (recommended)** — fresh subagent per task, review between tasks  
2. **Inline Execution** — execute tasks in this session with checkpoints  

Which approach?
