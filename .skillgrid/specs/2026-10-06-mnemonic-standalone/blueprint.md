# Mnemonic Standalone Module Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use subagent-execution (recommended) or simple-execution to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Status:** PROPOSED

**Tier:** T2

**Build shape:** Smallest usable whole

**Goal:** Extract mnemonic from skillgrid-cli into an independently buildable Go module with its own binary, clean break (no passthrough), and updated external configs.

**Architecture:** Two-module Go workspace. `mnemonic/` gets all 37 packages under `internal/`, a `cmd/mnemonic` binary with the full subcommand tree, and ~22 top-level facade files. `skillgrid-cli` shrinks to install + sync-repo. A `go.work` links both modules.

**Tech Stack:** Go 1.26, go.work, existing deps

**Spec:** `./briefing.md`

## Terms

- **Facade** — top-level `.go` file in `mnemonic/` (package `mnemonic`) that re-exports symbols from `mnemonic/internal/` for external import.
- **Clean break** — `skillgrid` CLI has no mnemonic subcommands; no passthrough.

## Hypothesis

**Claim:** The 37 mnemonic packages can be moved to a new `mnemonic/` module with a go.work and facade layer, and both modules build + test independently.
**Right condition:** `cd mnemonic && go build ./cmd/mnemonic && go test ./...` AND `cd skillgrid-cli && go build ./cmd/skillgrid && go test ./...` both pass.
**Wrong condition:** Circular import, missing facade symbol, or embed path break that requires refactoring internal package structure.
**Thinnest MVP:** `mnemonic mcp` starts the MCP stdio server as a standalone binary.
**Door check:** Task 2 (create mnemonic/go.mod + move packages) — if `go build` fails due to internal circularity that the move doesn't resolve, stop and reassess.

## Must-Haves

See `briefing.md` § Must-Haves for the full list. Summary of one-way-door decisions:

1. **Clean break: ~20 subcommands removed from `skillgrid` CLI** — existing `skillgrid mcp` / `skillgrid serve` invocations break until `mnemonic` binary is installed.
2. **Module path `github.com/devopstales/skillgrid/mnemonic`** — permanent import path once published.

## Global Constraints

- Go 1.26 (both modules)
- No new external dependencies
- All mnemonic packages move as-is (no internal refactoring)
- `mnemonic` binary independently installable
- `skillgrid install` installs the `mnemonic` binary
- Both modules pass `go vet` + `go test` independently
- `go.work` is dev convenience; each module builds without it

## File Structure

See `briefing.md` § File Structure for the complete tree.

---

### Task 1: Delete stray nested git repo + inline logging

> ⚠ one-way: inlining `logging.go` into `mnemonic/setup` changes the import path for 4 files. Reversible but must be done before the move.

**Files:**
- Delete: `skillgrid-cli/internal/mnemonic/mcp/.git` (entire nested git repo)
- Delete: `skillgrid-cli/internal/mnemonic/mcp/.skillgrid` (if present)
- Create: `skillgrid-cli/internal/mnemonic/setup/logging.go` (inlined copy)
- Modify: `skillgrid-cli/internal/mnemonic/setup/setup.go` — remove import of `skillgrid-cli/internal/logging`, use local `logging` package
- Modify: `skillgrid-cli/internal/mnemonic/setup/kilocode.go` — same
- Modify: `skillgrid-cli/internal/mnemonic/setup/opencode.go` — same
- Modify: `skillgrid-cli/internal/mnemonic/setup/cursor.go` — same

**Interfaces:**
- Consumes: `logging.Info`, `logging.Infof`, `logging.Error`, `logging.Errorf` (4 functions)
- Produces: `mnemonic/setup` package with self-contained logging (no external import of `skillgrid-cli/internal/logging`)
- Seam: none (in-process)
- Deletion test: if `skillgrid-cli/internal/logging/` is deleted, `mnemonic/setup` still compiles
- Adapters: 1 (the inlined file)

**SATISFIES:** nested-git-deleted, logging-inlined

- [ ] **Step 1: Delete the stray nested git repo**

```bash
rm -rf skillgrid-cli/internal/mnemonic/mcp/.git
rm -rf skillgrid-cli/internal/mnemonic/mcp/.skillgrid
```

- [ ] **Step 2: Inline logging.go into mnemonic/setup**

Create `skillgrid-cli/internal/mnemonic/setup/logging.go`:

```go
// Package setup — inlined from skillgrid-cli/internal/logging.
package setup

import (
	"fmt"
	"os"
)

func logInfo(msg string) {
	fmt.Fprintln(os.Stderr, msg)
}

func logInfof(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func logError(msg string) {
	fmt.Fprintln(os.Stderr, "error:", msg)
}

func logErrorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
}
```

- [ ] **Step 3: Update 4 setup files to use local logging**

In `setup.go`, `kilocode.go`, `opencode.go`, `cursor.go`:
- Remove `"github.com/devopstales/skillgrid/skillgrid-cli/internal/logging"` import
- Replace `logging.Info(` → `logInfo(`, `logging.Infof(` → `logInfof(`, `logging.Error(` → `logError(`, `logging.Errorf(` → `logErrorf(`

- [ ] **Step 4: Verify build + test**

```bash
cd skillgrid-cli && go build ./... && go test ./...
```

Expected: PASS (all existing tests still pass)

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: inline logging into mnemonic/setup, delete nested git repo"
```

---

### Task 2: Create mnemonic/go.mod + move packages

> ⚠ one-way: module path `github.com/devopstales/skillgrid/mnemonic` is permanent.

**Files:**
- Create: `mnemonic/go.mod`
- Move: `skillgrid-cli/internal/mnemonic/` → `mnemonic/internal/` (git mv)

**Interfaces:**
- Consumes: all 37 packages under `skillgrid-cli/internal/mnemonic/`
- Produces: `mnemonic/internal/` with same package structure; `mnemonic/go.mod` with module path `github.com/devopstales/skillgrid/mnemonic`
- Seam: go.work at repo root
- Deletion test: `skillgrid-cli/internal/mnemonic/` no longer exists
- Adapters: go.work (dev) + replace directive (prod)

**SATISFIES:** module-created, packages-moved

- [ ] **Step 1: Create mnemonic/go.mod**

```bash
mkdir -p mnemonic
cd mnemonic
cat > go.mod <<'EOF'
module github.com/devopstales/skillgrid/mnemonic

go 1.26

require (
	github.com/benedoc-inc/onnxer v0.7.0
	github.com/bluuewhale/loom v0.0.0-20260402142357-ae7e94ccfead
	github.com/charmbracelet/bubbletea v1.3.6
	github.com/charmbracelet/huh v1.0.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/google/uuid v1.6.0
	github.com/mark3labs/mcp-go v0.58.0
	github.com/odvcencio/gotreesitter v0.52.0
	github.com/tidwall/gjson v1.18.0
	github.com/tidwall/sjson v1.2.5
	gopkg.in/yaml.v3 v3.0.1
	modernc.org/sqlite v1.59.0
)
EOF
```

Note: the exact `require` block will be finalized by `go mod tidy` after the move.

- [ ] **Step 2: Git mv the packages**

```bash
git mv skillgrid-cli/internal/mnemonic mnemonic/internal
```

- [ ] **Step 3: Update all import paths in mnemonic/internal/**

All files under `mnemonic/internal/` that import `github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/...` must change to `github.com/devopstales/skillgrid/mnemonic/internal/...`.

```bash
cd mnemonic
find internal -name '*.go' -exec sed -i '' 's|github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic|github.com/devopstales/skillgrid/mnemonic/internal|g' {} +
```

- [ ] **Step 4: Create go.work at repo root**

```bash
cat > go.work <<'EOF'
go 1.26

use (
	./mnemonic
	./skillgrid-cli
)
EOF
```

- [ ] **Step 5: Run go mod tidy in mnemonic**

```bash
cd mnemonic && go mod tidy
```

This will populate `go.sum` and adjust the `require` block.

- [ ] **Step 6: Verify mnemonic builds (internal packages only, no cmd yet)**

```bash
cd mnemonic && go build ./internal/...
```

Expected: PASS (or only missing cmd/ errors, which is expected since we haven't moved cmd files yet)

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: move mnemonic packages to standalone module"
```

---

### Task 3: Move cmd files + create mnemonic binary

**Files:**
- Create: `mnemonic/cmd/mnemonic/` (directory)
- Move: ~45 `.go` files from `skillgrid-cli/cmd/skillgrid/` → `mnemonic/cmd/mnemonic/`
- Modify: `mnemonic/cmd/mnemonic/main.go` — change package imports from `skillgrid-cli/internal/mnemonic/...` to `mnemonic/internal/...`

**The files to move** (all mnemonic-related cmd files):
`mcp.go`, `mem.go`, `memhelpers.go`, `search.go`, `code_intel.go`, `loop_cmd.go`, `index_progress.go`, `init_arch.go`, `init_boot.go`, `init_preamble.go`, `init_ingest.go`, `doctor.go`, `doctor_strict.go`, `migrate.go`, `policy_cmd.go`, `session.go`, `skill.go`, `trail.go`, `logs.go`, `memory.go`, `code_eval.go`, `search_affected_cli.go`, `search_pdg_cli.go`, `search_rename_cli.go`, `search_taint_cli.go`, `handoff_gone.go`, `reorder.go` + all corresponding `*_test.go` files.

**Files that STAY in skillgrid-cli:**
`main.go` (trimmed), `init_cmd.go` (if it's the install entry — verify), `init_cmd_test.go`.

**Interfaces:**
- Consumes: `mnemonic/internal/...` packages (all 37)
- Produces: `mnemonic` binary with full subcommand tree
- Seam: `cmd/mnemonic/main.go` is the entry point
- Deletion test: if `mnemonic/cmd/` is deleted, the module still builds (internal packages only)
- Adapters: 1 (the binary)

**SATISFIES:** mnemonic-binary-builds, mnemonic-mcp-works, mnemonic-serve-works

- [ ] **Step 1: Create cmd/mnemonic directory**

```bash
mkdir -p mnemonic/cmd/mnemonic
```

- [ ] **Step 2: Git mv the cmd files**

```bash
cd skillgrid-cli/cmd/skillgrid
for f in mcp.go mem.go memhelpers.go search.go code_intel.go loop_cmd.go \
         index_progress.go init_arch.go init_boot.go init_preamble.go init_ingest.go \
         doctor.go doctor_strict.go migrate.go policy_cmd.go session.go skill.go \
         trail.go logs.go memory.go code_eval.go search_affected_cli.go \
         search_pdg_cli.go search_rename_cli.go search_taint_cli.go \
         handoff_gone.go reorder.go; do
  git mv "$f" ../../mnemonic/cmd/mnemonic/ 2>/dev/null || true
done
# Move test files
for f in mcp_test.go mem_test.go mem_context_test.go mem_distill_test.go \
         mem_expire_test.go mem_export_test.go mem_fs_test.go mem_graph_test.go \
         mem_provenance_test.go mem_relations_test.go mem_search_trajectory_test.go \
         mem_skills_test.go mem_snapshot_test.go memory_test.go migrate_test.go \
         policy_cmd_test.go prime_test.go reorder_test.go search_affected_test.go \
         search_pdg_cli_test.go search_rename_cli_test.go search_taint_cli_test.go \
         session_show_test.go session_test.go skill_test.go trail_test.go \
         code_community_test.go code_eval_test.go code_intel_test.go \
         doctor_strict_test.go handoff_gone_test.go index_progress_test.go \
         init_arch_test.go init_boot_test.go init_preamble_test.go logs_test.go; do
  git mv "$f" ../../mnemonic/cmd/mnemonic/ 2>/dev/null || true
done
```

- [ ] **Step 3: Update main.go in mnemonic/cmd/mnemonic/**

The moved `main.go` (from skillgrid-cli) needs:
- Package stays `main`
- All imports of `github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/...` → `github.com/devopstales/skillgrid/mnemonic/internal/...`
- The `switch rest0` block keeps all mnemonic subcommands
- Remove the `install` and `sync-repo` cases (those stay in skillgrid-cli)
- The `flagShorthands`, `boolFlags`, `reorderArgs`, `parseAgents`, `printFlags` helpers stay (they're needed for the mnemonic flag parsing)

```bash
cd mnemonic/cmd/mnemonic
sed -i '' 's|github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic|github.com/devopstales/skillgrid/mnemonic/internal|g' *.go
```

- [ ] **Step 4: Verify mnemonic binary builds**

```bash
cd mnemonic && go build -o /tmp/mnemonic-test ./cmd/mnemonic
/tmp/mnemonic-test --help
/tmp/mnemonic-test mcp --help
```

Expected: help text shows all mnemonic subcommands. `mcp --help` shows MCP server flags.

- [ ] **Step 5: Verify mnemonic mcp starts**

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | timeout 5 /tmp/mnemonic-test mcp 2>/dev/null | head -1
```

Expected: JSON response with `serverInfo`.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: create mnemonic binary with full subcommand tree"
```

---

### Task 4: Create facade files for skillgrid-cli

**Files:**
- Create: ~22 top-level `.go` files in `mnemonic/` (package `mnemonic`)

The facade files re-export the symbols that `skillgrid-cli/internal/install` needs. Based on the import analysis, `skillgrid-cli` only imports `mnemonic/setup` (from `install.go` and `agents.go`). The facade needs to expose whatever `setup` exports that `install` uses.

**Interfaces:**
- Consumes: `mnemonic/internal/setup` package
- Produces: `mnemonic` package (top-level) with re-exported symbols
- Seam: `mnemonic/` top-level package
- Deletion test: if facade files are deleted, `skillgrid-cli` fails to compile (it imports `mnemonic/setup` → now `mnemonic`)
- Adapters: 1 (the facade package)

**SATISFIES:** facade-exposed, cli-compiles-with-facade

- [ ] **Step 1: Identify exactly what skillgrid-cli needs from mnemonic**

```bash
cd skillgrid-cli
grep -rn "mnemonic/setup\|mnemonic\." internal/install/ cmd/skillgrid/ | grep -v "_test.go"
```

List every symbol used. This determines the facade surface.

- [ ] **Step 2: Create facade files**

For each package that `skillgrid-cli` imports from `mnemonic/internal/`, create a top-level file in `mnemonic/`:

Example `mnemonic/setup.go`:

```go
package mnemonic

import (
	"github.com/devopstales/skillgrid/mnemonic/internal/setup"
)

type SetupConfig = setup.Config
func SetupRun(c *setup.Config) error { return setup.Run(c) }
// ... re-export every symbol that skillgrid-cli uses
```

Repeat for each needed package. The count is expected to be ~22 files but the exact list is determined by Step 1.

- [ ] **Step 3: Update skillgrid-cli imports to use facade**

In `skillgrid-cli/internal/install/install.go`:
- Change `"github.com/devopstales/skillgrid/skillgrid-cli/internal/mnemonic/setup"` → `"github.com/devopstales/skillgrid/mnemonic"`
- Update all `setup.X` references to the facade names

In `skillgrid-cli/internal/install/agents.go`:
- Same import change if it references mnemonic types

- [ ] **Step 4: Add replace directive to skillgrid-cli/go.mod**

```bash
cd skillgrid-cli
go mod edit -replace github.com/devopstales/skillgrid/mnemonic=../mnemonic
go mod edit -require github.com/devopstales/skillgrid/mnemonic@v0.0.0
```

- [ ] **Step 5: Verify skillgrid-cli builds**

```bash
cd skillgrid-cli && go build ./...
```

Expected: PASS (install compiles against facade)

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "feat: create mnemonic facade for skillgrid-cli imports"
```

---

### Task 5: Trim skillgrid-cli (remove mnemonic subcommands + unused deps)

> ⚠ one-way: clean break — `skillgrid mcp|serve|index|...` are removed.

**Files:**
- Modify: `skillgrid-cli/cmd/skillgrid/main.go` — remove all mnemonic subcommand cases
- Delete: `skillgrid-cli/internal/logging/` (no longer needed; inlined into mnemonic/setup)
- Delete: `skillgrid-cli/skillgrid` (61 MB committed binary)
- Modify: `skillgrid-cli/go.mod` — remove mnemonic-only deps

**Interfaces:**
- Consumes: trimmed `main.go` with only `install`, `sync-repo`, `help`
- Produces: `skillgrid` binary that only does install + sync-repo
- Seam: `main.go` switch statement
- Deletion test: if a mnemonic subcommand is called, it prints "unknown command"
- Adapters: 1 (the trimmed binary)

**SATISFIES:** clean-break, cli-trimmed, binary-deleted

- [ ] **Step 1: Trim main.go**

In `skillgrid-cli/cmd/skillgrid/main.go`:
- Remove all `case "mcp":`, `case "serve":`, `case "init":`, `case "index":`, `case "prime":`, `case "compact":`, `case "pages":`, `case "diff-impact":`, `case "orient":`, `case "grep":`, `case "callers":` ... etc.
- Keep only: `case "help"`, `case ""`, `case "install"`, `case "in"`, `case "sync-repo"`
- Remove the `runMCP`, `runServe`, `runProjectInit`, `runIndex`, etc. function references
- Update the usage/help text to only show install + sync-repo
- Remove imports that are no longer needed

- [ ] **Step 2: Delete internal/logging**

```bash
git rm -r skillgrid-cli/internal/logging/
```

- [ ] **Step 3: Delete the 61 MB binary**

```bash
git rm skillgrid-cli/skillgrid
```

- [ ] **Step 4: Clean up go.mod**

```bash
cd skillgrid-cli
# Remove deps that only mnemonic used
go mod edit -droprequire github.com/benedoc-inc/onnxer
go mod edit -droprequire github.com/bluuewhale/loom
go mod edit -droprequire github.com/charmbracelet/bubbletea
go mod edit -droprequire github.com/charmbracelet/huh
go mod edit -droprequire github.com/fsnotify/fsnotify
go mod edit -droprequire github.com/google/uuid
go mod edit -droprequire github.com/mark3labs/mcp-go
go mod edit -droprequire github.com/odvcencio/gotreesitter
go mod edit -droprequire github.com/tidwall/gjson
go mod edit -droprequire github.com/tidwall/sjson
go mod edit -droprequire gopkg.in/yaml.v3
go mod edit -droprequire modernc.org/sqlite
go mod tidy
```

Note: `gopkg.in/yaml.v3` may still be needed by `install` (it parses tools.yaml). `go mod tidy` will keep it if so.

- [ ] **Step 5: Verify skillgrid-cli builds + tests**

```bash
cd skillgrid-cli && go build ./... && go test ./...
```

Expected: PASS. `skillgrid --help` shows only install + sync-repo.

- [ ] **Step 6: Verify `skillgrid install` still works**

```bash
cd skillgrid-cli && go run ./cmd/skillgrid install --dry-run --yes
```

Expected: dry-run output shows install plan (clone, configure agents, write MCP entries).

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "refactor: trim skillgrid-cli to install + sync-repo only (clean break)"
```

---

### Task 6: Update Taskfile.yml (build + test both modules)

**Files:**
- Modify: `Taskfile.yml`

**Interfaces:**
- Consumes: both module directories
- Produces: `task build` builds both binaries, `task test` tests both modules
- Seam: Taskfile variables
- Deletion test: N/A (build config)
- Adapters: 2 (two module build tasks)

**SATISFIES:** taskfile-dual-build, taskfile-dual-test

- [ ] **Step 1: Update Taskfile variables**

```yaml
vars:
  MNEMONIC_DIR: mnemonic
  MODULE_DIR: skillgrid-cli
  MNEMONIC_BIN: mnemonic
  BIN: skillgrid
  OUT: dist
  MNEMONIC_PKG: "./cmd/mnemonic"
  PKG: "./cmd/skillgrid"
  VERSION:
    sh: 'echo "${SKILLGRID_VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)}"'
```

- [ ] **Step 2: Update ui:build task**

```yaml
ui:build:
  desc: Build the skillgrid-ui SPA into mnemonic/internal/http/ui/dist (go:embed input).
  cmds:
    - cd skillgrid-ui && npm ci && npm run build
```

- [ ] **Step 3: Update build task to build both modules**

```yaml
build:
  desc: Build both the mnemonic and skillgrid binaries.
  cmds:
    - cd {{.MNEMONIC_DIR}} && mkdir -p ../{{.OUT}}
    - cd {{.MNEMONIC_DIR}} && go build -tags ui -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.MNEMONIC_BIN}} {{.MNEMONIC_PKG}}
    - cd {{.MODULE_DIR}} && mkdir -p ../{{.OUT}}
    - cd {{.MODULE_DIR}} && go build -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.BIN}} {{.PKG}}
    - rm -f ~/.local/bin/mnemonic ~/.local/bin/skillgrid
    - cp dist/mnemonic dist/skillgrid ~/.local/bin/
```

- [ ] **Step 4: Update all cross-build tasks**

`build-linux` and `build-darwin` must cross-build both modules:

```yaml
build-linux:
  desc: Build linux binaries (amd64 + 386) for both modules.
  cmds:
    - cd {{.MNEMONIC_DIR}} && GOOS=linux GOARCH=amd64 go build -tags ui -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.MNEMONIC_BIN}}-linux-amd64 {{.MNEMONIC_PKG}}
    - cd {{.MNEMONIC_DIR}} && GOOS=linux GOARCH=386 go build -tags ui -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.MNEMONIC_BIN}}-linux-386 {{.MNEMONIC_PKG}}
    - cd {{.MODULE_DIR}} && GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.BIN}}-linux-amd64 {{.PKG}}
    - cd {{.MODULE_DIR}} && GOOS=linux GOARCH=386 go build -ldflags "-s -w -X main.version={{.VERSION}}" -o ../{{.OUT}}/{{.BIN}}-linux-386 {{.PKG}}
```

Same pattern for `build-darwin`.

- [ ] **Step 5: Update test task**

```yaml
test:
  desc: Run go vet and go test ./... in both modules.
  cmds:
    - cd {{.MNEMONIC_DIR}} && go vet ./...
    - cd {{.MNEMONIC_DIR}} && go test ./...
    - cd {{.MODULE_DIR}} && go vet ./...
    - cd {{.MODULE_DIR}} && go test ./...
```

- [ ] **Step 6: Update seed-test task**

```yaml
seed-test:
  desc: Run the Mnemonic all-stores seed integration tests.
  cmds:
    - cd {{.MNEMONIC_DIR}} && go test -v -run TestSeed ./internal/integration/
```

- [ ] **Step 7: Update fmt + install tasks**

```yaml
fmt:
  desc: gofmt -w both modules.
  cmds:
    - cd {{.MNEMONIC_DIR}} && gofmt -l -w .
    - cd {{.MODULE_DIR}} && gofmt -l -w .

install:
  desc: Install linux-amd64 binaries into ~/.skillgrid/bin.
  cmds:
    - mkdir -p "$HOME/.skillgrid/bin"
    - cp {{.OUT}}/{{.MNEMONIC_BIN}}-linux-amd64 "$HOME/.skillgrid/bin/{{.MNEMONIC_BIN}}"
    - chmod +x "$HOME/.skillgrid/bin/{{.MNEMONIC_BIN}}"
    - cp {{.OUT}}/{{.BIN}}-linux-amd64 "$HOME/.skillgrid/bin/{{.BIN}}"
    - chmod +x "$HOME/.skillgrid/bin/{{.BIN}}"
    - echo "installed mnemonic + skillgrid to $HOME/.skillgrid/bin/"
```

- [ ] **Step 8: Commit**

```bash
git add Taskfile.yml
git commit -m "build: update Taskfile for dual-module build + test"
```

---

### Task 7: Update external configs + hooks + git-hooks + vite

**Files:**
- Modify: `config.d/mcp.yaml`
- Modify: `plugins/cursor/mcp.json`
- Modify: `hooks/cursor-session-start.sh`
- Modify: `hooks/opencode-session-start.sh`
- Modify: `git-hooks/post-commit`
- Modify: `git-hooks/post-checkout`
- Modify: `skillgrid-ui/vite.config.ts`
- Modify: `scripts/sync-mnemonic-rule.sh`

**Interfaces:**
- Consumes: `mnemonic` binary on PATH
- Produces: all configs reference `mnemonic` instead of `skillgrid`
- Seam: file contents (text replacements)
- Deletion test: if `mnemonic` binary is not on PATH, hooks fail-open (curl health check fails, no crash)
- Adapters: 8 files

**SATISFIES:** mcp-config-updated, cursor-mcp-updated, hooks-updated, githooks-updated, vite-path-updated, sync-script-fixed

- [ ] **Step 1: Update config.d/mcp.yaml**

```yaml
# Change:
  mnemonic:
    type: local
    command:
      - skillgrid
      - mcp
# To:
  mnemonic:
    type: local
    command:
      - mnemonic
      - mcp
```

- [ ] **Step 2: Update plugins/cursor/mcp.json**

```json
{
  "mcpServers": {
    "mnemonic": {
      "command": "mnemonic",
      "args": ["mcp"]
    }
  }
}
```

- [ ] **Step 3: Update hooks/cursor-session-start.sh**

Replace all `skillgrid serve` → `mnemonic serve`, `skillgrid prime` → `mnemonic prime`, `command -v skillgrid` → `command -v mnemonic`.

- [ ] **Step 4: Update hooks/opencode-session-start.sh**

Same replacements: `skillgrid serve` → `mnemonic serve`, `skillgrid prime` → `mnemonic prime`, `command -v skillgrid` → `command -v mnemonic`.

- [ ] **Step 5: Update git-hooks/post-commit**

```sh
command -v mnemonic >/dev/null 2>&1 || exit 0
mnemonic index --dir "$root" >/dev/null 2>&1 || true
mnemonic pages --dir "$root" >/dev/null 2>&1 || true
```

- [ ] **Step 6: Update git-hooks/post-checkout**

Same as post-commit.

- [ ] **Step 7: Update skillgrid-ui/vite.config.ts**

```ts
// Change:
outDir: '../skillgrid-cli/internal/mnemonic/http/ui/dist',
// To:
outDir: '../mnemonic/internal/http/ui/dist',
```

- [ ] **Step 8: Fix scripts/sync-mnemonic-rule.sh**

The script references `plugins/_shared/memory-protocol.md` which is stale. Update the path to wherever the memory protocol file actually lives (check `plugins/` directory structure). If the file was moved to `.agents/skills/_shared/`, update accordingly.

- [ ] **Step 9: Verify build with updated vite path**

```bash
cd skillgrid-ui && npm ci && npm run build
ls mnemonic/internal/http/ui/dist/
cd mnemonic && go build -tags ui ./cmd/mnemonic
```

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "chore: update all external configs to reference mnemonic binary"
```

---

### Task 8: Update skillgrid install to install mnemonic binary

**Files:**
- Modify: `skillgrid-cli/internal/install/install.go` (or wherever the install flow writes binaries)

**Interfaces:**
- Consumes: `mnemonic` binary from `dist/` or built from source
- Produces: `~/.skillgrid/bin/mnemonic` (or `~/.local/bin/mnemonic`) after `skillgrid install`
- Seam: install flow step that copies binaries
- Deletion test: if mnemonic binary is not installed, `skillgrid mcp` (on the agent side) fails
- Adapters: 1 (the install flow)

**SATISFIES:** install-copies-mnemonic

- [ ] **Step 1: Find where install copies the skillgrid binary**

```bash
grep -n "skillgrid" skillgrid-cli/internal/install/install.go | grep -i "bin\|copy\|cp\|install"
```

- [ ] **Step 2: Add mnemonic binary copy to install flow**

After the existing binary copy step, add:

```go
// Install mnemonic binary alongside skillgrid
if mnemonicBin := filepath.Join(distDir, "mnemonic-"+goos+"-"+goarch); fileExists(mnemonicBin) {
    copyFile(mnemonicBin, filepath.Join(binDir, "mnemonic"))
    os.Chmod(filepath.Join(binDir, "mnemonic"), 0755)
}
```

If the install flow doesn't copy binaries from `dist/` (it may clone from git instead), add a step to build or download the mnemonic binary.

- [ ] **Step 3: Verify install dry-run shows mnemonic**

```bash
cd skillgrid-cli && go run ./cmd/skillgrid install --dry-run --yes
```

Expected: output mentions copying/installing the mnemonic binary.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "feat: skillgrid install copies mnemonic binary"
```

---

### Task 9: Add CI Go jobs

**Files:**
- Modify: `.github/workflows/hub-sync-check.yml` (or create new workflow)

**Interfaces:**
- Consumes: both module directories
- Produces: CI runs `go build` + `go test` for both modules on every PR
- Seam: GitHub Actions workflow
- Deletion test: if CI job is removed, no automated build/test on PRs
- Adapters: 2 (two job steps, one per module)

**SATISFIES:** ci-dual-build

- [ ] **Step 1: Add Go CI job**

In `.github/workflows/hub-sync-check.yml`, add a new job:

```yaml
  go-build-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
      - name: Build + test mnemonic
        working-directory: mnemonic
        run: |
          go mod tidy
          go vet ./...
          go build ./...
          go test ./...
      - name: Build + test skillgrid-cli
        working-directory: skillgrid-cli
        run: |
          go mod tidy
          go vet ./...
          go build ./...
          go test ./...
```

- [ ] **Step 2: Verify workflow YAML is valid**

```bash
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/hub-sync-check.yml'))"
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/hub-sync-check.yml
git commit -m "ci: add Go build + test jobs for both modules"
```

---

### Task 10: Final verification + cleanup

**Files:**
- Verify: all must-haves from briefing.md

**Interfaces:**
- Consumes: all previous tasks
- Produces: verified dual-module build
- Seam: N/A
- Deletion test: N/A
- Adapters: N/A

**SATISFIES:** all-remaining-scenarios

- [ ] **Step 1: Full build from repo root (go.work)**

```bash
go build ./...
```

Expected: builds both modules.

- [ ] **Step 2: Full test from each module**

```bash
cd mnemonic && go vet ./... && go test ./...
cd ../skillgrid-cli && go vet ./... && go test ./...
```

Expected: all PASS.

- [ ] **Step 3: Verify mnemonic binary standalone**

```bash
mnemonic mcp --help
mnemonic serve --help
mnemonic index --help
mnemonic orient --help
mnemonic search --help
mnemonic mem --help
```

Expected: all show help text with correct subcommands.

- [ ] **Step 4: Verify skillgrid binary is trimmed**

```bash
skillgrid --help
skillgrid mcp  # should show "unknown command"
```

Expected: `--help` shows only install + sync-repo. `skillgrid mcp` prints error.

- [ ] **Step 5: Verify no remaining references to old paths**

```bash
grep -r "skillgrid-cli/internal/mnemonic" --include="*.go" . | grep -v "^mnemonic/"
grep -r "skillgrid-cli/internal/logging" --include="*.go" . | grep -v "^mnemonic/"
```

Expected: zero results (or only in mnemonic/ where the import was updated).

- [ ] **Step 6: Verify external configs**

```bash
grep "mnemonic" config.d/mcp.yaml
grep "mnemonic" plugins/cursor/mcp.json
grep "mnemonic" hooks/cursor-session-start.sh
grep "mnemonic" hooks/opencode-session-start.sh
grep "mnemonic" git-hooks/post-commit
grep "mnemonic" git-hooks/post-checkout
grep "mnemonic/internal/http/ui/dist" skillgrid-ui/vite.config.ts
```

Expected: all reference `mnemonic`, none reference `skillgrid` for mnemonic commands.

- [ ] **Step 7: Commit**

```bash
git add -A
git commit -m "chore: final cleanup + verification for mnemonic standalone module"
```
